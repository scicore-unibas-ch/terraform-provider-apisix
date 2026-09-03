// Package route implements the apisix_route resource.
//
// Routes are the largest schema in APISIX after upstreams. Schema notes:
//   - id is the URL key (Required + RequiresReplace).
//   - The singular/plural pairs (uri/uris, host/hosts, remote_addr/remote_addrs)
//     are mutually exclusive and enforced by ConfigValidators.
//   - script and plugins are mutually exclusive (APISIX rule).
//   - upstream_id and inline upstream are mutually exclusive.
//   - vars is a free-form JSON string with JSON-equivalence suppression so
//     server-side normalization does not produce drift.
//   - The inline upstream and timeout blocks are SingleNestedAttribute
//     (Plugin Framework idiom replacing the SDK v2 MaxItems:1 list hack).
//   - Sets are used for unordered collections (methods, hosts, uris,
//     remote_addrs) so reordering does not produce diffs.
package route

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/client"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/inlineupstream"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/planmodifier/jsonmap"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/planmodifier/jsonstring"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/pluginsmap"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/tfconv"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/timeoutshelper"
)

const apiKind = "routes"

var validHTTPMethods = []string{
	"GET", "POST", "PUT", "DELETE", "PATCH",
	"HEAD", "OPTIONS", "TRACE", "CONNECT", "PURGE",
}

var (
	_ resource.Resource                     = (*Resource)(nil)
	_ resource.ResourceWithConfigure        = (*Resource)(nil)
	_ resource.ResourceWithImportState      = (*Resource)(nil)
	_ resource.ResourceWithConfigValidators = (*Resource)(nil)
)

// Resource is the apisix_route resource.
type Resource struct {
	client *client.Client
}

// NewResource is the constructor registered with the provider.
func NewResource() resource.Resource { return &Resource{} }

type model struct {
	ID              types.String   `tfsdk:"id"`
	Name            types.String   `tfsdk:"name"`
	Desc            types.String   `tfsdk:"desc"`
	URI             types.String   `tfsdk:"uri"`
	URIs            types.Set      `tfsdk:"uris"`
	Host            types.String   `tfsdk:"host"`
	Hosts           types.Set      `tfsdk:"hosts"`
	RemoteAddr      types.String   `tfsdk:"remote_addr"`
	RemoteAddrs     types.Set      `tfsdk:"remote_addrs"`
	Methods         types.Set      `tfsdk:"methods"`
	Priority        types.Int64    `tfsdk:"priority"`
	Vars            types.String   `tfsdk:"vars"`
	FilterFunc      types.String   `tfsdk:"filter_func"`
	Plugins         types.Map      `tfsdk:"plugins"`
	Script          types.String   `tfsdk:"script"`
	UpstreamID      types.String   `tfsdk:"upstream_id"`
	Upstream        types.Object   `tfsdk:"upstream"`
	ServiceID       types.String   `tfsdk:"service_id"`
	PluginConfigID  types.String   `tfsdk:"plugin_config_id"`
	Labels          types.Map      `tfsdk:"labels"`
	Timeout         types.Object   `tfsdk:"timeout"`
	EnableWebsocket types.Bool     `tfsdk:"enable_websocket"`
	Status          types.Int64    `tfsdk:"status"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route"
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = client.FromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (r *Resource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	validators := []resource.ConfigValidator{
		resourcevalidator.Conflicting(path.MatchRoot("uri"), path.MatchRoot("uris")),
		resourcevalidator.Conflicting(path.MatchRoot("host"), path.MatchRoot("hosts")),
		resourcevalidator.Conflicting(path.MatchRoot("remote_addr"), path.MatchRoot("remote_addrs")),
		resourcevalidator.Conflicting(path.MatchRoot("script"), path.MatchRoot("plugins")),
		resourcevalidator.Conflicting(path.MatchRoot("upstream_id"), path.MatchRoot("upstream")),
	}
	// The inline upstream carries the same cross-attribute rules as the
	// standalone apisix_upstream resource.
	return append(validators, inlineupstream.ConfigValidators(func(name string) path.Expression {
		return path.MatchRoot("upstream").AtName(name)
	})...)
}

func (r *Resource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an APISIX Route. The id attribute is the route's APISIX object key.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}),
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Unique route identifier. Used as the APISIX object key. Changing this forces replacement.",
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

			"uri": schema.StringAttribute{
				Optional:    true,
				Description: "Single request URI to match. Mutually exclusive with `uris`.",
			},
			"uris": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Set of request URIs to match. Mutually exclusive with `uri`.",
			},

			"host": schema.StringAttribute{
				Optional:    true,
				Description: "Single hostname to match. Mutually exclusive with `hosts`.",
			},
			"hosts": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Set of hostnames to match. Mutually exclusive with `host`.",
			},

			"remote_addr": schema.StringAttribute{
				Optional:    true,
				Description: "Single client IP/CIDR to match. Mutually exclusive with `remote_addrs`.",
			},
			"remote_addrs": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Set of client IPs/CIDRs to match. Mutually exclusive with `remote_addr`.",
			},

			"methods": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "HTTP methods to match. Valid values: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, TRACE, CONNECT, PURGE.",
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf(validHTTPMethods...)),
				},
			},
			"priority": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "Route priority. Higher values are matched first. Defaults to 0.",
			},
			"vars": schema.StringAttribute{
				Optional:    true,
				Description: "JSON-encoded list of variable conditions for advanced routing. JSON-equivalent values are suppressed.",
				PlanModifiers: []planmodifier.String{
					jsonstring.SuppressEquivalent(),
				},
			},
			"filter_func": schema.StringAttribute{
				Optional:    true,
				Description: "Lua function source for custom filtering.",
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
				Description: "Reference to an apisix_upstream by id. Mutually exclusive with `upstream`.",
			},
			"upstream": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Inline upstream definition. Supports the full APISIX upstream surface (scheme, timeouts, retries, hashing, keepalive, mTLS, service discovery, health checks). Mutually exclusive with `upstream_id`.",
				Attributes:  inlineupstream.SchemaAttrs(),
			},

			"service_id": schema.StringAttribute{
				Optional:    true,
				Description: "Reference to an apisix_service by id.",
			},
			"plugin_config_id": schema.StringAttribute{
				Optional:    true,
				Description: "Reference to an apisix_plugin_config by id.",
			},
			"labels": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Labels as key-value pairs.",
			},

			"timeout": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Per-route upstream timeout overrides (seconds).",
				Attributes: map[string]schema.Attribute{
					"connect": schema.Int64Attribute{
						Optional:    true,
						Description: "Connect timeout in seconds.",
					},
					"send": schema.Int64Attribute{
						Optional:    true,
						Description: "Send timeout in seconds.",
					},
					"read": schema.Int64Attribute{
						Optional:    true,
						Description: "Read timeout in seconds.",
					},
				},
			},

			"enable_websocket": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Enable WebSocket upgrade. Defaults to false.",
			},
			"status": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Description: "Route status: 1 = enabled, 0 = disabled. Defaults to 1.",
				Validators: []validator.Int64{
					int64validator.OneOf(0, 1),
				},
			},
		},
	}
}

// apiBody is the wire payload. Pointers separate "unset" from "zero".
type apiBody struct {
	ID              string                     `json:"id"`
	Name            *string                    `json:"name,omitempty"`
	Desc            *string                    `json:"desc,omitempty"`
	URI             *string                    `json:"uri,omitempty"`
	URIs            []string                   `json:"uris,omitempty"`
	Host            *string                    `json:"host,omitempty"`
	Hosts           []string                   `json:"hosts,omitempty"`
	RemoteAddr      *string                    `json:"remote_addr,omitempty"`
	RemoteAddrs     []string                   `json:"remote_addrs,omitempty"`
	Methods         []string                   `json:"methods,omitempty"`
	Priority        *int64                     `json:"priority,omitempty"`
	Vars            json.RawMessage            `json:"vars,omitempty"`
	FilterFunc      *string                    `json:"filter_func,omitempty"`
	Plugins         map[string]json.RawMessage `json:"plugins,omitempty"`
	Script          *string                    `json:"script,omitempty"`
	UpstreamID      *string                    `json:"upstream_id,omitempty"`
	Upstream        *inlineupstream.Body       `json:"upstream,omitempty"`
	ServiceID       *string                    `json:"service_id,omitempty"`
	PluginConfigID  *string                    `json:"plugin_config_id,omitempty"`
	Labels          map[string]string          `json:"labels,omitempty"`
	Timeout         *inlineupstream.Timeout    `json:"timeout,omitempty"`
	EnableWebsocket *bool                      `json:"enable_websocket,omitempty"`
	Status          *int64                     `json:"status,omitempty"`
}

func (r *Resource) buildBody(ctx context.Context, m *model) (*apiBody, diag.Diagnostics) {
	var diags diag.Diagnostics
	body := &apiBody{
		ID:              m.ID.ValueString(),
		Name:            tfconv.StringPtr(m.Name),
		Desc:            tfconv.StringPtr(m.Desc),
		URI:             tfconv.StringPtr(m.URI),
		Host:            tfconv.StringPtr(m.Host),
		RemoteAddr:      tfconv.StringPtr(m.RemoteAddr),
		FilterFunc:      tfconv.StringPtr(m.FilterFunc),
		Script:          tfconv.StringPtr(m.Script),
		UpstreamID:      tfconv.StringPtr(m.UpstreamID),
		ServiceID:       tfconv.StringPtr(m.ServiceID),
		PluginConfigID:  tfconv.StringPtr(m.PluginConfigID),
		Priority:        tfconv.Int64Ptr(m.Priority),
		Status:          tfconv.Int64Ptr(m.Status),
		EnableWebsocket: tfconv.BoolPtr(m.EnableWebsocket),
	}

	for _, c := range []struct {
		set types.Set
		dst *[]string
	}{
		{m.URIs, &body.URIs},
		{m.Hosts, &body.Hosts},
		{m.RemoteAddrs, &body.RemoteAddrs},
		{m.Methods, &body.Methods},
	} {
		if c.set.IsNull() || c.set.IsUnknown() {
			continue
		}
		out := []string{}
		diags.Append(c.set.ElementsAs(ctx, &out, false)...)
		if diags.HasError() {
			return nil, diags
		}
		*c.dst = out
	}

	if !m.Vars.IsNull() && !m.Vars.IsUnknown() {
		raw := m.Vars.ValueString()
		var probe any
		if err := json.Unmarshal([]byte(raw), &probe); err != nil {
			diags.AddAttributeError(
				path.Root("vars"),
				"Invalid vars JSON",
				err.Error(),
			)
			return nil, diags
		}
		body.Vars = json.RawMessage(raw)
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

	if !m.Upstream.IsNull() && !m.Upstream.IsUnknown() {
		up, d := inlineupstream.BuildBody(ctx, m.Upstream, path.Root("upstream"))
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.Upstream = up
	}

	if !m.Timeout.IsNull() && !m.Timeout.IsUnknown() {
		to, d := inlineupstream.BuildTimeout(ctx, m.Timeout)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.Timeout = to
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
		resp.Diagnostics.AddError("Failed to create route", err.Error())
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
		resp.Diagnostics.AddError("Failed to read route", err.Error())
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
		resp.Diagnostics.AddError("Failed to update route", err.Error())
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
		resp.Diagnostics.AddError("Failed to delete route", err.Error())
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
		URI             string                     `json:"uri"`
		URIs            []string                   `json:"uris"`
		Host            string                     `json:"host"`
		Hosts           []string                   `json:"hosts"`
		RemoteAddr      string                     `json:"remote_addr"`
		RemoteAddrs     []string                   `json:"remote_addrs"`
		Methods         []string                   `json:"methods"`
		Priority        *int64                     `json:"priority"`
		Vars            json.RawMessage            `json:"vars"`
		FilterFunc      string                     `json:"filter_func"`
		Plugins         map[string]json.RawMessage `json:"plugins"`
		Script          string                     `json:"script"`
		UpstreamID      string                     `json:"upstream_id"`
		Upstream        json.RawMessage            `json:"upstream"`
		ServiceID       string                     `json:"service_id"`
		PluginConfigID  string                     `json:"plugin_config_id"`
		Labels          map[string]string          `json:"labels"`
		Timeout         json.RawMessage            `json:"timeout"`
		EnableWebsocket *bool                      `json:"enable_websocket"`
		Status          *int64                     `json:"status"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		diags.AddError("Failed to decode route response", err.Error())
		return diags
	}

	m.ID = types.StringValue(body.ID)
	m.Name = tfconv.NullableStringPreserving(body.Name, m.Name)
	m.Desc = tfconv.NullableStringPreserving(body.Desc, m.Desc)
	m.URI = tfconv.NullableStringPreserving(body.URI, m.URI)
	m.Host = tfconv.NullableStringPreserving(body.Host, m.Host)
	m.RemoteAddr = tfconv.NullableStringPreserving(body.RemoteAddr, m.RemoteAddr)
	m.FilterFunc = tfconv.NullableStringPreserving(body.FilterFunc, m.FilterFunc)
	m.Script = tfconv.NullableStringPreserving(body.Script, m.Script)
	m.UpstreamID = tfconv.NullableStringPreserving(body.UpstreamID, m.UpstreamID)
	m.ServiceID = tfconv.NullableStringPreserving(body.ServiceID, m.ServiceID)
	m.PluginConfigID = tfconv.NullableStringPreserving(body.PluginConfigID, m.PluginConfigID)

	// Optional+Computed attributes: backfill the schema defaults when APISIX
	// omits the field so refreshed state stays plan-stable.
	m.Priority = tfconv.Int64OrDefault(body.Priority, 0)
	m.Status = tfconv.Int64OrDefault(body.Status, 1)
	if body.EnableWebsocket != nil {
		m.EnableWebsocket = types.BoolValue(*body.EnableWebsocket)
	} else {
		m.EnableWebsocket = types.BoolValue(false)
	}

	m.URIs = nullableStringSet(ctx, body.URIs, &diags)
	m.Hosts = nullableStringSet(ctx, body.Hosts, &diags)
	m.RemoteAddrs = nullableStringSet(ctx, body.RemoteAddrs, &diags)
	m.Methods = nullableStringSet(ctx, body.Methods, &diags)

	// Canonical re-marshal keeps the on-disk form stable; the plan modifier
	// handles equivalence vs. the user's literal HCL.
	m.Vars = tfconv.CanonicalJSON(body.Vars)

	if body.Labels == nil {
		m.Labels = types.MapNull(types.StringType)
	} else {
		v, d := types.MapValueFrom(ctx, types.StringType, body.Labels)
		diags.Append(d...)
		m.Labels = v
	}

	pVal, d := pluginsmap.Decode(ctx, body.Plugins, true)
	diags.Append(d...)
	m.Plugins = pVal

	if len(body.Upstream) == 0 || string(body.Upstream) == "null" {
		m.Upstream = types.ObjectNull(inlineupstream.AttrTypes)
	} else {
		v, d := inlineupstream.DecodeInto(ctx, body.Upstream)
		diags.Append(d...)
		m.Upstream = v
	}

	if len(body.Timeout) == 0 || string(body.Timeout) == "null" {
		m.Timeout = types.ObjectNull(inlineupstream.TimeoutAttrTypes)
	} else {
		v, d := inlineupstream.DecodeTimeout(body.Timeout)
		diags.Append(d...)
		m.Timeout = v
	}

	return diags
}

func nullableStringSet(ctx context.Context, vals []string, diags *diag.Diagnostics) types.Set {
	if vals == nil {
		return types.SetNull(types.StringType)
	}
	v, d := types.SetValueFrom(ctx, types.StringType, vals)
	diags.Append(d...)
	return v
}
