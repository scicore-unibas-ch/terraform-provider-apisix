// Package consumergroup implements the apisix_consumer_group resource.
//
// This is the reference implementation for all APISIX resources in the
// Plugin Framework rewrite. It demonstrates the standard pattern:
//
//  1. Resource ID is a dedicated `id` attribute (Required + ForceNew). It is
//     the URL key on the APISIX side. Renames force replacement, eliminating
//     the SDK v2 ambiguity around `name` doubling as URL ID and body field.
//  2. Update uses HTTP PUT (full replace), not PATCH. Removing a field from
//     config removes it server-side.
//  3. Plugin maps use a JSON-equivalence plan modifier to suppress drift from
//     APISIX's server-side normalization (key reordering, default injection).
//  4. State after Create equals plan verbatim (Plugin Framework strict-mode);
//     Read fully refreshes from the API and the plan modifier handles drift.
package consumergroup

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/client"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/planmodifier/jsonmap"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/pluginsmap"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/tfconv"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/timeoutshelper"
)

const apiKind = "consumer_groups"

var (
	_ resource.Resource                = (*Resource)(nil)
	_ resource.ResourceWithConfigure   = (*Resource)(nil)
	_ resource.ResourceWithImportState = (*Resource)(nil)
)

// Resource is the apisix_consumer_group resource.
type Resource struct {
	client *client.Client
}

// NewResource is the constructor registered with the provider.
func NewResource() resource.Resource { return &Resource{} }

type model struct {
	ID       types.String   `tfsdk:"id"`
	Name     types.String   `tfsdk:"name"`
	Desc     types.String   `tfsdk:"desc"`
	Plugins  types.Map      `tfsdk:"plugins"`
	Labels   types.Map      `tfsdk:"labels"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_consumer_group"
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = client.FromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an APISIX Consumer Group. Plugins defined here apply to every consumer in the group.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Unique identifier of the consumer group. Used as the APISIX object key. Changing this forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Description: "Human-readable name.",
			},
			"desc": schema.StringAttribute{
				Optional:    true,
				Description: "Description.",
			},
			"plugins": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Map of plugin name to JSON-encoded configuration. Required, but may be empty: APISIX requires the field to be present, not to hold any plugins.",
				PlanModifiers: []planmodifier.Map{
					jsonmap.SuppressEquivalent(),
				},
			},
			"labels": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Labels as key-value pairs.",
			},
		},
	}
}

// apiBody is the JSON payload sent to APISIX. Pointers let us distinguish
// "user did not set" (nil → omitted) from "user set empty string" (non-nil).
type apiBody struct {
	ID      string                     `json:"id"`
	Name    *string                    `json:"name,omitempty"`
	Desc    *string                    `json:"desc,omitempty"`
	Plugins map[string]json.RawMessage `json:"plugins"` // no omitempty: APISIX requires the key, and an empty map is valid
	Labels  map[string]string          `json:"labels,omitempty"`
}

func (r *Resource) buildBody(ctx context.Context, m *model) (*apiBody, diag.Diagnostics) {
	var diags diag.Diagnostics
	body := &apiBody{
		ID:   m.ID.ValueString(),
		Name: tfconv.StringPtr(m.Name),
		Desc: tfconv.StringPtr(m.Desc),
	}

	plugins, d := pluginsmap.Build(ctx, m.Plugins, path.Root("plugins"))
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}
	body.Plugins = plugins

	if !m.Labels.IsNull() && !m.Labels.IsUnknown() {
		labels := map[string]string{}
		diags.Append(m.Labels.ElementsAs(ctx, &labels, false)...)
		if diags.HasError() {
			return nil, diags
		}
		body.Labels = labels
	}

	return body, diags
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := timeoutshelper.Apply(ctx, plan.Timeouts, "create", timeoutshelper.Default, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := r.buildBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Put(ctx, apiKind, body.ID, body); err != nil {
		resp.Diagnostics.AddError("Failed to create consumer_group", err.Error())
		return
	}

	// State must equal plan verbatim for non-Computed attributes (Framework
	// strict mode). The next refresh will reconcile any server-side defaults.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := timeoutshelper.Apply(ctx, state.Timeouts, "read", timeoutshelper.Default, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.Get(ctx, apiKind, state.ID.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read consumer_group", err.Error())
		return
	}

	resp.Diagnostics.Append(decodeInto(ctx, apiResp.Value, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := timeoutshelper.Apply(ctx, plan.Timeouts, "update", timeoutshelper.Default, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := r.buildBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Put(ctx, apiKind, body.ID, body); err != nil {
		resp.Diagnostics.AddError("Failed to update consumer_group", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := timeoutshelper.Apply(ctx, state.Timeouts, "delete", timeoutshelper.Default, &resp.Diagnostics)
	defer cancel()
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, apiKind, state.ID.ValueString(), false)
	if err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Failed to delete consumer_group", err.Error())
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// decodeInto parses an APISIX response body into a model. Plugin values are
// re-marshaled canonically (Go sorts JSON map keys) so that downstream JSON
// equality comparisons stay stable.
func decodeInto(ctx context.Context, raw json.RawMessage, m *model) diag.Diagnostics {
	var diags diag.Diagnostics
	var body struct {
		ID      string                     `json:"id"`
		Name    string                     `json:"name"`
		Desc    string                     `json:"desc"`
		Plugins map[string]json.RawMessage `json:"plugins"`
		Labels  map[string]string          `json:"labels"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		diags.AddError("Failed to decode consumer_group response", err.Error())
		return diags
	}

	m.ID = types.StringValue(body.ID)
	m.Name = tfconv.NullableString(body.Name)
	m.Desc = tfconv.NullableString(body.Desc)

	pVal, d := pluginsmap.Decode(ctx, body.Plugins, false)
	diags.Append(d...)
	m.Plugins = pVal

	if body.Labels == nil {
		m.Labels = types.MapNull(types.StringType)
	} else {
		lVal, d := types.MapValueFrom(ctx, types.StringType, body.Labels)
		diags.Append(d...)
		m.Labels = lVal
	}

	return diags
}
