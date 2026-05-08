// Package globalrule implements the apisix_global_rule resource.
//
// Global rules are plugin configurations that apply to every request flowing
// through APISIX, regardless of route. The schema is intentionally minimal:
// just the rule identifier and a map of plugin configurations as JSON strings.
package globalrule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/client"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/planmodifier/jsonmap"
)

const apiKind = "global_rules"

var (
	_ resource.Resource                = (*Resource)(nil)
	_ resource.ResourceWithConfigure   = (*Resource)(nil)
	_ resource.ResourceWithImportState = (*Resource)(nil)
)

// Resource is the apisix_global_rule resource.
type Resource struct {
	client *client.Client
}

// NewResource is the constructor registered with the provider.
func NewResource() resource.Resource { return &Resource{} }

type model struct {
	ID      types.String `tfsdk:"id"`
	Plugins types.Map    `tfsdk:"plugins"`
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_global_rule"
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data",
			fmt.Sprintf("expected *client.Client, got %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an APISIX Global Rule. Plugins defined here apply to every request flowing through APISIX.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Unique identifier of the global rule. Used as the APISIX object key. Changing this forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"plugins": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Map of plugin name to JSON-encoded configuration. APISIX requires at least one plugin.",
				PlanModifiers: []planmodifier.Map{
					jsonmap.SuppressEquivalent(),
				},
			},
		},
	}
}

type apiBody struct {
	ID      string                     `json:"id"`
	Plugins map[string]json.RawMessage `json:"plugins,omitempty"`
}

func (r *Resource) buildBody(ctx context.Context, m *model) (*apiBody, diag.Diagnostics) {
	var diags diag.Diagnostics
	body := &apiBody{ID: m.ID.ValueString()}

	if !m.Plugins.IsNull() && !m.Plugins.IsUnknown() {
		plugins := map[string]string{}
		diags.Append(m.Plugins.ElementsAs(ctx, &plugins, false)...)
		if diags.HasError() {
			return nil, diags
		}
		body.Plugins = make(map[string]json.RawMessage, len(plugins))
		for k, v := range plugins {
			var probe any
			if err := json.Unmarshal([]byte(v), &probe); err != nil {
				diags.AddAttributeError(
					path.Root("plugins").AtMapKey(k),
					"Invalid plugin JSON",
					fmt.Sprintf("plugin %q: %v", k, err),
				)
				return nil, diags
			}
			body.Plugins[k] = json.RawMessage(v)
		}
	}

	return body, diags
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := r.buildBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Put(ctx, apiKind, body.ID, body); err != nil {
		resp.Diagnostics.AddError("Failed to create global_rule", err.Error())
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

	apiResp, err := r.client.Get(ctx, apiKind, state.ID.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to read global_rule", err.Error())
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

	body, diags := r.buildBody(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := r.client.Put(ctx, apiKind, body.ID, body); err != nil {
		resp.Diagnostics.AddError("Failed to update global_rule", err.Error())
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

	err := r.client.Delete(ctx, apiKind, state.ID.ValueString(), false)
	if err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Failed to delete global_rule", err.Error())
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// decodeInto parses an APISIX response body into the model. Plugin values are
// re-marshaled canonically (Go sorts JSON map keys) so JSON-equivalence
// comparisons in the plan modifier stay stable across reads.
func decodeInto(ctx context.Context, raw json.RawMessage, m *model) diag.Diagnostics {
	var diags diag.Diagnostics
	var body struct {
		ID      string                     `json:"id"`
		Plugins map[string]json.RawMessage `json:"plugins"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		diags.AddError("Failed to decode global_rule response", err.Error())
		return diags
	}

	m.ID = types.StringValue(body.ID)

	pluginStrs := make(map[string]string, len(body.Plugins))
	for k, v := range body.Plugins {
		var obj any
		if err := json.Unmarshal(v, &obj); err != nil {
			pluginStrs[k] = string(v)
			continue
		}
		canonical, err := json.Marshal(obj)
		if err != nil {
			pluginStrs[k] = string(v)
			continue
		}
		pluginStrs[k] = string(canonical)
	}
	pVal, d := types.MapValueFrom(ctx, types.StringType, pluginStrs)
	diags.Append(d...)
	m.Plugins = pVal

	return diags
}
