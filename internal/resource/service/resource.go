// Package service implements the apisix_service resource.
//
// A service is a reusable bundle of routing config (hosts, plugins, upstream)
// that one or more routes can reference. Schema highlights:
//   - upstream_id and the inline upstream block are mutually exclusive.
//   - script and plugins are mutually exclusive (APISIX rule).
//   - The inline upstream uses SingleNestedAttribute, which is the Plugin
//     Framework idiom replacing the SDK v2 MaxItems:1 list hack.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/client"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/inlineupstream"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/planmodifier/jsonmap"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/timeoutshelper"
)

const apiKind = "services"

var (
	_ resource.Resource                     = (*Resource)(nil)
	_ resource.ResourceWithConfigure        = (*Resource)(nil)
	_ resource.ResourceWithImportState      = (*Resource)(nil)
	_ resource.ResourceWithConfigValidators = (*Resource)(nil)
)

// Resource is the apisix_service resource.
type Resource struct {
	client *client.Client
}

// NewResource is the constructor registered with the provider.
func NewResource() resource.Resource { return &Resource{} }

type model struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Desc            types.String `tfsdk:"desc"`
	Hosts           types.Set    `tfsdk:"hosts"`
	Plugins         types.Map    `tfsdk:"plugins"`
	Script          types.String `tfsdk:"script"`
	UpstreamID      types.String `tfsdk:"upstream_id"`
	Upstream        types.Object `tfsdk:"upstream"`
	Labels          types.Map      `tfsdk:"labels"`
	EnableWebsocket types.Bool     `tfsdk:"enable_websocket"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
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

func (r *Resource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		// APISIX rejects setting both script and plugins.
		resourcevalidator.Conflicting(
			path.MatchRoot("script"),
			path.MatchRoot("plugins"),
		),
		// upstream_id (reference) vs upstream (inline) are exclusive.
		resourcevalidator.Conflicting(
			path.MatchRoot("upstream_id"),
			path.MatchRoot("upstream"),
		),
	}
}

func (r *Resource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an APISIX Service: a reusable bundle of host/plugin/upstream config that routes can reference via service_id.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Unique service identifier. Used as the APISIX object key. Changing this forces replacement.",
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
			"hosts": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Set of hostnames this service matches.",
			},
			"plugins": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Map of plugin name to JSON-encoded configuration. Mutually exclusive with `script`.",
				PlanModifiers: []planmodifier.Map{
					jsonmap.SuppressEquivalent(),
				},
			},
			"script": schema.StringAttribute{
				Optional:    true,
				Description: "Lua script for custom logic. Mutually exclusive with `plugins`.",
			},
			"upstream_id": schema.StringAttribute{
				Optional:    true,
				Description: "ID of an existing apisix_upstream. Mutually exclusive with `upstream`.",
			},
			"upstream": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Inline upstream definition. Supports the full APISIX upstream surface (scheme, timeouts, retries, hashing, keepalive, mTLS, service discovery, health checks). Mutually exclusive with `upstream_id`.",
				Attributes:  inlineupstream.SchemaAttrs(),
			},
			"labels": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Labels as key-value pairs.",
			},
			"enable_websocket": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Enable WebSocket upgrade for routes through this service. Defaults to false.",
			},
		},
	}
}

// apiBody is the wire payload for APISIX. Pointers separate "unset" from "empty".
type apiBody struct {
	ID              string                     `json:"id"`
	Name            *string                    `json:"name,omitempty"`
	Desc            *string                    `json:"desc,omitempty"`
	Hosts           []string                   `json:"hosts,omitempty"`
	Plugins         map[string]json.RawMessage `json:"plugins,omitempty"`
	Script          *string                    `json:"script,omitempty"`
	UpstreamID      *string                    `json:"upstream_id,omitempty"`
	Upstream        *inlineupstream.Body       `json:"upstream,omitempty"`
	Labels          map[string]string          `json:"labels,omitempty"`
	EnableWebsocket *bool                      `json:"enable_websocket,omitempty"`
}

func (r *Resource) buildBody(ctx context.Context, m *model) (*apiBody, diag.Diagnostics) {
	var diags diag.Diagnostics
	body := &apiBody{ID: m.ID.ValueString()}

	if !m.Name.IsNull() && !m.Name.IsUnknown() {
		v := m.Name.ValueString()
		body.Name = &v
	}
	if !m.Desc.IsNull() && !m.Desc.IsUnknown() {
		v := m.Desc.ValueString()
		body.Desc = &v
	}
	if !m.Script.IsNull() && !m.Script.IsUnknown() {
		v := m.Script.ValueString()
		body.Script = &v
	}
	if !m.UpstreamID.IsNull() && !m.UpstreamID.IsUnknown() {
		v := m.UpstreamID.ValueString()
		body.UpstreamID = &v
	}
	// enable_websocket is Optional+Computed+Default(false), so it always has a
	// known value after planning. Always emit so removing it from config
	// reverts the server to false.
	if !m.EnableWebsocket.IsNull() && !m.EnableWebsocket.IsUnknown() {
		v := m.EnableWebsocket.ValueBool()
		body.EnableWebsocket = &v
	}

	if !m.Hosts.IsNull() && !m.Hosts.IsUnknown() {
		hosts := []string{}
		diags.Append(m.Hosts.ElementsAs(ctx, &hosts, false)...)
		if diags.HasError() {
			return nil, diags
		}
		body.Hosts = hosts
	}

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

	if !m.Labels.IsNull() && !m.Labels.IsUnknown() {
		labels := map[string]string{}
		diags.Append(m.Labels.ElementsAs(ctx, &labels, false)...)
		if diags.HasError() {
			return nil, diags
		}
		body.Labels = labels
	}

	if !m.Upstream.IsNull() && !m.Upstream.IsUnknown() {
		up, d := inlineupstream.BuildBody(ctx, m.Upstream, path.Root("upstream"))
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.Upstream = up
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
		resp.Diagnostics.AddError("Failed to create service", err.Error())
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
		resp.Diagnostics.AddError("Failed to read service", err.Error())
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
		resp.Diagnostics.AddError("Failed to update service", err.Error())
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
		resp.Diagnostics.AddError("Failed to delete service", err.Error())
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func decodeInto(ctx context.Context, raw json.RawMessage, m *model) diag.Diagnostics {
	var diags diag.Diagnostics
	var body struct {
		ID              string                     `json:"id"`
		Name            string                     `json:"name"`
		Desc            string                     `json:"desc"`
		Hosts           []string                   `json:"hosts"`
		Plugins         map[string]json.RawMessage `json:"plugins"`
		Script          string                     `json:"script"`
		UpstreamID      string                     `json:"upstream_id"`
		Upstream        json.RawMessage            `json:"upstream"`
		Labels          map[string]string          `json:"labels"`
		EnableWebsocket *bool                      `json:"enable_websocket"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		diags.AddError("Failed to decode service response", err.Error())
		return diags
	}

	m.ID = types.StringValue(body.ID)
	m.Name = nullableString(body.Name)
	m.Desc = nullableString(body.Desc)
	m.Script = nullableString(body.Script)
	m.UpstreamID = nullableString(body.UpstreamID)

	if body.EnableWebsocket != nil {
		m.EnableWebsocket = types.BoolValue(*body.EnableWebsocket)
	} else {
		m.EnableWebsocket = types.BoolValue(false)
	}

	if body.Hosts == nil {
		m.Hosts = types.SetNull(types.StringType)
	} else {
		v, d := types.SetValueFrom(ctx, types.StringType, body.Hosts)
		diags.Append(d...)
		m.Hosts = v
	}

	if body.Labels == nil {
		m.Labels = types.MapNull(types.StringType)
	} else {
		v, d := types.MapValueFrom(ctx, types.StringType, body.Labels)
		diags.Append(d...)
		m.Labels = v
	}

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
	if len(pluginStrs) == 0 {
		m.Plugins = types.MapNull(types.StringType)
	} else {
		v, d := types.MapValueFrom(ctx, types.StringType, pluginStrs)
		diags.Append(d...)
		m.Plugins = v
	}

	if len(body.Upstream) == 0 || string(body.Upstream) == "null" {
		m.Upstream = types.ObjectNull(inlineupstream.AttrTypes)
	} else {
		v, d := inlineupstream.DecodeInto(ctx, body.Upstream)
		diags.Append(d...)
		m.Upstream = v
	}

	return diags
}

func nullableString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
