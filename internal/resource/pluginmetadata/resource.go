// Package pluginmetadata implements the apisix_plugin_metadata resource.
//
// APISIX exposes per-plugin metadata at /apisix/admin/plugin_metadata/{name}.
// The URL key is the plugin name (e.g. "syslog", "http-logger") and the body
// is an arbitrary JSON object whose shape is plugin-specific. Because there
// is no schema we can express statically, the metadata is carried as a
// JSON-encoded string and validated at plan time.
//
// Pattern is the same as consumergroup (see that package for the rationale):
//
//  1. id (= plugin name) is Required + RequiresReplace.
//  2. Update uses HTTP PUT (full replace).
//  3. A JSON-equivalence plan modifier suppresses drift from server-side
//     normalization (key reordering, default injection).
//  4. State after Create equals plan verbatim; Read reconciles drift.
package pluginmetadata

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
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/planmodifier/jsonstring"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/timeoutshelper"
)

const apiKind = "plugin_metadata"

var (
	_ resource.Resource                = (*Resource)(nil)
	_ resource.ResourceWithConfigure   = (*Resource)(nil)
	_ resource.ResourceWithImportState = (*Resource)(nil)
)

// Resource is the apisix_plugin_metadata resource.
type Resource struct {
	client *client.Client
}

// NewResource is the constructor registered with the provider.
func NewResource() resource.Resource { return &Resource{} }

type model struct {
	ID       types.String   `tfsdk:"id"`
	Metadata types.String   `tfsdk:"metadata"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin_metadata"
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = client.FromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages APISIX Plugin Metadata. The metadata body shape is plugin-specific; consult the APISIX plugin documentation.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Name of the plugin (e.g. \"syslog\", \"http-logger\"). Used as the APISIX object key. Changing this forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"metadata": schema.StringAttribute{
				Required:    true,
				Description: "JSON-encoded plugin metadata body. JSON-equivalent values are suppressed so server-side normalization does not produce drift.",
				PlanModifiers: []planmodifier.String{
					jsonstring.SuppressEquivalent(),
				},
			},
		},
	}
}

func (r *Resource) buildBody(_ context.Context, m *model) (json.RawMessage, diag.Diagnostics) {
	var diags diag.Diagnostics
	raw := m.Metadata.ValueString()

	var probe any
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		diags.AddAttributeError(
			path.Root("metadata"),
			"Invalid metadata JSON",
			err.Error(),
		)
		return nil, diags
	}
	if _, ok := probe.(map[string]any); !ok {
		diags.AddAttributeError(
			path.Root("metadata"),
			"Invalid metadata JSON",
			"metadata must be a JSON object (e.g. jsonencode({...})), not a scalar or array",
		)
		return nil, diags
	}
	return json.RawMessage(raw), diags
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

	if _, err := r.client.Put(ctx, apiKind, plan.ID.ValueString(), body); err != nil {
		resp.Diagnostics.AddError("Failed to create plugin_metadata", err.Error())
		return
	}

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
		resp.Diagnostics.AddError("Failed to read plugin_metadata", err.Error())
		return
	}

	resp.Diagnostics.Append(decodeInto(apiResp.Value, &state)...)
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

	if _, err := r.client.Put(ctx, apiKind, plan.ID.ValueString(), body); err != nil {
		resp.Diagnostics.AddError("Failed to update plugin_metadata", err.Error())
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
		resp.Diagnostics.AddError("Failed to delete plugin_metadata", err.Error())
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// decodeInto parses an APISIX response into the model. The response value is
// the metadata body, but APISIX decorates it with the plugin name as `id` and
// may add `create_time` / `update_time`. We strip those so the canonicalized
// form compared by jsonstring.SuppressEquivalent matches user input.
func decodeInto(raw json.RawMessage, m *model) diag.Diagnostics {
	var diags diag.Diagnostics

	// Unmarshal into any (not RawMessage) so json.Marshal canonicalizes keys
	// recursively, not just at the top level — needed so ImportStateVerify
	// produces a textually stable round-trip.
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		diags.AddError("Failed to decode plugin_metadata response", err.Error())
		return diags
	}

	for _, sys := range []string{"id", "create_time", "update_time"} {
		delete(body, sys)
	}

	canonical, err := json.Marshal(body)
	if err != nil {
		diags.AddError("Failed to canonicalize plugin_metadata response", err.Error())
		return diags
	}
	m.Metadata = types.StringValue(string(canonical))
	return diags
}
