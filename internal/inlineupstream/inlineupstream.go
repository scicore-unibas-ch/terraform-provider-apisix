// Package inlineupstream defines the schema, wire format, and codec for the
// APISIX upstream object. It is used two ways:
//
//   - embedded inline inside apisix_route and apisix_service as a
//     SingleNestedAttribute (via SchemaAttrs / BuildBody / DecodeInto), and
//   - as the body of the standalone apisix_upstream resource, whose model
//     embeds Fields and adds the id/timeouts attributes.
//
// Sharing one definition keeps the three surfaces identical: users get the
// full APISIX upstream feature set whether they reference an upstream by id
// or define it inline.
package inlineupstream

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/planmodifier/jsonstring"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/tfconv"
)

var (
	validTypes     = []string{"roundrobin", "chash", "ewma", "least_conn"}
	validSchemes   = []string{"grpc", "grpcs", "http", "https", "tcp", "tls", "udp", "kafka"}
	validHashOn    = []string{"vars", "header", "cookie", "consumer", "vars_combinations"}
	validPassHosts = []string{"pass", "node", "rewrite"}
)

// Attribute type maps. Exported so callers can construct ObjectNull / ListNull
// values matching the inline schema.
var (
	NodeAttrTypes = map[string]attr.Type{
		"host":     types.StringType,
		"port":     types.Int64Type,
		"weight":   types.Int64Type,
		"priority": types.Int64Type,
		"metadata": types.MapType{ElemType: types.StringType},
	}

	TimeoutAttrTypes = map[string]attr.Type{
		"connect": types.Int64Type,
		"send":    types.Int64Type,
		"read":    types.Int64Type,
	}

	KeepaliveAttrTypes = map[string]attr.Type{
		"size":         types.Int64Type,
		"idle_timeout": types.Int64Type,
		"requests":     types.Int64Type,
	}

	TLSAttrTypes = map[string]attr.Type{
		"client_cert":    types.StringType,
		"client_key":     types.StringType,
		"client_cert_id": types.StringType,
		"verify":         types.BoolType,
	}

	AttrTypes = map[string]attr.Type{
		"name":           types.StringType,
		"desc":           types.StringType,
		"type":           types.StringType,
		"nodes":          types.ListType{ElemType: types.ObjectType{AttrTypes: NodeAttrTypes}},
		"health_check":   types.StringType,
		"timeout":        types.ObjectType{AttrTypes: TimeoutAttrTypes},
		"retries":        types.Int64Type,
		"retry_timeout":  types.Int64Type,
		"scheme":         types.StringType,
		"labels":         types.MapType{ElemType: types.StringType},
		"service_name":   types.StringType,
		"discovery_type": types.StringType,
		"discovery_args": types.MapType{ElemType: types.StringType},
		"hash_on":        types.StringType,
		"key":            types.StringType,
		"pass_host":      types.StringType,
		"upstream_host":  types.StringType,
		"keepalive_pool": types.ObjectType{AttrTypes: KeepaliveAttrTypes},
		"tls":            types.ObjectType{AttrTypes: TLSAttrTypes},
	}
)

// Fields is the Terraform-side model of an upstream. Route and service carry
// it as a types.Object; the standalone apisix_upstream resource embeds it in
// its model struct (the framework promotes the tagged fields).
type Fields struct {
	Name          types.String `tfsdk:"name"`
	Desc          types.String `tfsdk:"desc"`
	Type          types.String `tfsdk:"type"`
	Nodes         types.List   `tfsdk:"nodes"`
	HealthCheck   types.String `tfsdk:"health_check"`
	Timeout       types.Object `tfsdk:"timeout"`
	Retries       types.Int64  `tfsdk:"retries"`
	RetryTimeout  types.Int64  `tfsdk:"retry_timeout"`
	Scheme        types.String `tfsdk:"scheme"`
	Labels        types.Map    `tfsdk:"labels"`
	ServiceName   types.String `tfsdk:"service_name"`
	DiscoveryType types.String `tfsdk:"discovery_type"`
	DiscoveryArgs types.Map    `tfsdk:"discovery_args"`
	HashOn        types.String `tfsdk:"hash_on"`
	Key           types.String `tfsdk:"key"`
	PassHost      types.String `tfsdk:"pass_host"`
	UpstreamHost  types.String `tfsdk:"upstream_host"`
	KeepalivePool types.Object `tfsdk:"keepalive_pool"`
	TLS           types.Object `tfsdk:"tls"`
}

// SchemaAttrs returns the upstream attribute set. Callers compose this into a
// SingleNestedAttribute (route/service) or merge it into a resource schema
// (apisix_upstream).
func SchemaAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Optional:    true,
			Description: "Human-readable name.",
		},
		"desc": schema.StringAttribute{
			Optional:    true,
			Description: "Description.",
		},
		"type": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString("roundrobin"),
			Description: "Load-balancing algorithm. Defaults to roundrobin.",
			Validators:  []validator.String{stringvalidator.OneOf(validTypes...)},
		},
		"nodes": schema.ListNestedAttribute{
			Optional:    true,
			Description: "Backend nodes. Required when not using service discovery.",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"host": schema.StringAttribute{Required: true, Description: "Node hostname or IP."},
					"port": schema.Int64Attribute{
						Required:    true,
						Validators:  []validator.Int64{int64validator.Between(1, 65535)},
						Description: "Node port (1-65535).",
					},
					"weight": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(1),
						Description: "Load-balancing weight. Defaults to 1.",
					},
					"priority": schema.Int64Attribute{
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(0),
						Description: "Node priority. Lower priority is tried first. Defaults to 0.",
					},
					"metadata": schema.MapAttribute{
						Optional:    true,
						ElementType: types.StringType,
						Description: "Per-node metadata key-value pairs.",
					},
				},
			},
		},
		"health_check": schema.StringAttribute{
			Optional:    true,
			Description: "JSON-encoded active/passive health-check configuration. JSON-equivalent values are suppressed.",
			PlanModifiers: []planmodifier.String{
				jsonstring.SuppressEquivalent(),
			},
		},
		"timeout": schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Connect/send/read timeout overrides (seconds).",
			Attributes: map[string]schema.Attribute{
				"connect": schema.Int64Attribute{Optional: true, Description: "Connect timeout in seconds."},
				"send":    schema.Int64Attribute{Optional: true, Description: "Send timeout in seconds."},
				"read":    schema.Int64Attribute{Optional: true, Description: "Read timeout in seconds."},
			},
		},
		"retries": schema.Int64Attribute{
			Optional:    true,
			Description: "Number of retry attempts.",
		},
		"retry_timeout": schema.Int64Attribute{
			Optional:    true,
			Description: "Total retry timeout in seconds.",
		},
		"scheme": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString("http"),
			Description: "Upstream protocol. Defaults to http.",
			Validators:  []validator.String{stringvalidator.OneOf(validSchemes...)},
		},
		"labels": schema.MapAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Labels as key-value pairs.",
		},
		"service_name": schema.StringAttribute{
			Optional:    true,
			Description: "Service name for service discovery (alternative to nodes).",
		},
		"discovery_type": schema.StringAttribute{
			Optional:    true,
			Description: "Service-discovery type (e.g. consul, nacos, eureka).",
		},
		"discovery_args": schema.MapAttribute{
			Optional:    true,
			ElementType: types.StringType,
			Description: "Service-discovery arguments (e.g. namespace_id, group_name).",
		},
		"hash_on": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString("vars"),
			Description: "Source of the hashing key for chash. Defaults to vars.",
			Validators:  []validator.String{stringvalidator.OneOf(validHashOn...)},
		},
		"key": schema.StringAttribute{
			Optional:    true,
			Description: "Hashing key for chash (e.g. remote_addr, uri, arg_name).",
		},
		"pass_host": schema.StringAttribute{
			Optional:    true,
			Computed:    true,
			Default:     stringdefault.StaticString("pass"),
			Description: "How to set the upstream Host header. Defaults to pass.",
			Validators: []validator.String{
				stringvalidator.OneOf(validPassHosts...),
				rewriteRequiresUpstreamHost{},
			},
		},
		"upstream_host": schema.StringAttribute{
			Optional:    true,
			Description: "Custom Host header. Required when pass_host = rewrite.",
		},
		"keepalive_pool": schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Keepalive pool configuration. Inner fields default to APISIX's standard pool sizing if a block is provided without them.",
			Attributes: map[string]schema.Attribute{
				"size": schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(320),
					Description: "Pool size. Defaults to 320.",
				},
				"idle_timeout": schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(60),
					Description: "Idle timeout in seconds. Defaults to 60.",
				},
				"requests": schema.Int64Attribute{
					Optional:    true,
					Computed:    true,
					Default:     int64default.StaticInt64(1000),
					Description: "Max requests per connection. Defaults to 1000.",
				},
			},
		},
		"tls": schema.SingleNestedAttribute{
			Optional:    true,
			Description: "TLS configuration for mTLS to the upstream.",
			Attributes: map[string]schema.Attribute{
				"client_cert": schema.StringAttribute{
					Optional:    true,
					Sensitive:   true,
					Description: "Client certificate (PEM). Mutually exclusive with client_cert_id.",
				},
				"client_key": schema.StringAttribute{
					Optional:    true,
					Sensitive:   true,
					Description: "Client private key (PEM). Pair with client_cert.",
				},
				"client_cert_id": schema.StringAttribute{
					Optional:    true,
					Description: "Reference to an apisix_ssl object providing the client certificate.",
				},
				"verify": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Verify the server certificate. Currently only effective for kafka upstreams. Defaults to false.",
				},
			},
		},
	}
}

// rewriteRequiresUpstreamHost enforces APISIX's conditional requirement that
// pass_host = "rewrite" comes with an upstream_host. It resolves
// upstream_host relative to its own path, so it works unchanged on the
// standalone resource (root level) and inside the inline upstream block.
type rewriteRequiresUpstreamHost struct{}

func (rewriteRequiresUpstreamHost) Description(context.Context) string {
	return `upstream_host must be set when pass_host is "rewrite"`
}

func (v rewriteRequiresUpstreamHost) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (rewriteRequiresUpstreamHost) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() != "rewrite" {
		return
	}

	hostPath := req.Path.ParentPath().AtName("upstream_host")
	var host types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, hostPath, &host)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Unknown means the value comes from another resource and only resolves
	// at apply time; give it the benefit of the doubt.
	if host.IsNull() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Missing upstream_host",
			`upstream_host must be set when pass_host is "rewrite".`,
		)
	}
}

// ConfigValidators returns the cross-attribute rules APISIX enforces on an
// upstream object, so invalid combinations fail at plan time instead of
// apply. attr maps an attribute name to its path expression: pass
// path.MatchRoot for the standalone apisix_upstream resource, or a prefixed
// expression (upstream.<name>) for the inline block in route/service.
func ConfigValidators(attr func(name string) path.Expression) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		// A static node list and service discovery are mutually exclusive.
		resourcevalidator.Conflicting(attr("nodes"), attr("discovery_type")),
		resourcevalidator.Conflicting(attr("nodes"), attr("service_name")),
		resourcevalidator.RequiredTogether(attr("service_name"), attr("discovery_type")),
		// Inline client cert/key pair vs. reference to an apisix_ssl object.
		resourcevalidator.Conflicting(attr("tls").AtName("client_cert"), attr("tls").AtName("client_cert_id")),
		resourcevalidator.RequiredTogether(attr("tls").AtName("client_cert"), attr("tls").AtName("client_key")),
	}
}

// Body is the wire payload for an upstream. Pointers separate "unset" from
// "zero".
type Body struct {
	Name          *string           `json:"name,omitempty"`
	Desc          *string           `json:"desc,omitempty"`
	Type          *string           `json:"type,omitempty"`
	Nodes         []Node            `json:"nodes,omitempty"`
	HealthCheck   json.RawMessage   `json:"checks,omitempty"`
	Timeout       *Timeout          `json:"timeout,omitempty"`
	Retries       *int64            `json:"retries,omitempty"`
	RetryTimeout  *int64            `json:"retry_timeout,omitempty"`
	Scheme        *string           `json:"scheme,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	ServiceName   *string           `json:"service_name,omitempty"`
	DiscoveryType *string           `json:"discovery_type,omitempty"`
	DiscoveryArgs map[string]string `json:"discovery_args,omitempty"`
	HashOn        *string           `json:"hash_on,omitempty"`
	Key           *string           `json:"key,omitempty"`
	PassHost      *string           `json:"pass_host,omitempty"`
	UpstreamHost  *string           `json:"upstream_host,omitempty"`
	KeepalivePool *Keepalive        `json:"keepalive_pool,omitempty"`
	TLS           *TLS              `json:"tls,omitempty"`
}

type Node struct {
	Host     string            `json:"host"`
	Port     int64             `json:"port"`
	Weight   int64             `json:"weight"`
	Priority int64             `json:"priority"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Timeout uses pointers so fields the user did not set are omitted rather
// than sent as zero (which APISIX would interpret as "no timeout").
type Timeout struct {
	Connect *int64 `json:"connect,omitempty"`
	Send    *int64 `json:"send,omitempty"`
	Read    *int64 `json:"read,omitempty"`
}

type Keepalive struct {
	Size        int64 `json:"size"`
	IdleTimeout int64 `json:"idle_timeout"`
	Requests    int64 `json:"requests"`
}

type TLS struct {
	ClientCert   *string `json:"client_cert,omitempty"`
	ClientKey    *string `json:"client_key,omitempty"`
	ClientCertID *string `json:"client_cert_id,omitempty"`
	Verify       bool    `json:"verify"`
}

// Body converts the Terraform fields to a wire payload. pathBase prefixes
// attribute paths in error diagnostics (e.g. "upstream.health_check" for the
// inline block, "health_check" for the standalone resource).
func (f *Fields) Body(ctx context.Context, pathBase path.Path) (*Body, diag.Diagnostics) {
	var diags diag.Diagnostics

	body := &Body{
		Name:          tfconv.StringPtr(f.Name),
		Desc:          tfconv.StringPtr(f.Desc),
		Type:          tfconv.StringPtr(f.Type),
		Scheme:        tfconv.StringPtr(f.Scheme),
		HashOn:        tfconv.StringPtr(f.HashOn),
		Key:           tfconv.StringPtr(f.Key),
		PassHost:      tfconv.StringPtr(f.PassHost),
		UpstreamHost:  tfconv.StringPtr(f.UpstreamHost),
		ServiceName:   tfconv.StringPtr(f.ServiceName),
		DiscoveryType: tfconv.StringPtr(f.DiscoveryType),
		Retries:       tfconv.Int64Ptr(f.Retries),
		RetryTimeout:  tfconv.Int64Ptr(f.RetryTimeout),
	}

	if !f.Labels.IsNull() && !f.Labels.IsUnknown() {
		labels := map[string]string{}
		diags.Append(f.Labels.ElementsAs(ctx, &labels, false)...)
		if diags.HasError() {
			return nil, diags
		}
		body.Labels = labels
	}
	if !f.DiscoveryArgs.IsNull() && !f.DiscoveryArgs.IsUnknown() {
		args := map[string]string{}
		diags.Append(f.DiscoveryArgs.ElementsAs(ctx, &args, false)...)
		if diags.HasError() {
			return nil, diags
		}
		body.DiscoveryArgs = args
	}
	if !f.HealthCheck.IsNull() && !f.HealthCheck.IsUnknown() {
		raw := f.HealthCheck.ValueString()
		var probe any
		if err := json.Unmarshal([]byte(raw), &probe); err != nil {
			diags.AddAttributeError(
				pathBase.AtName("health_check"),
				"Invalid health_check JSON",
				err.Error(),
			)
			return nil, diags
		}
		body.HealthCheck = json.RawMessage(raw)
	}
	if !f.Nodes.IsNull() && !f.Nodes.IsUnknown() {
		nodes, d := buildNodes(ctx, f.Nodes)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.Nodes = nodes
	}
	if !f.Timeout.IsNull() && !f.Timeout.IsUnknown() {
		t, d := BuildTimeout(ctx, f.Timeout)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.Timeout = t
	}
	if !f.KeepalivePool.IsNull() && !f.KeepalivePool.IsUnknown() {
		kp, d := buildKeepalive(ctx, f.KeepalivePool)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.KeepalivePool = kp
	}
	if !f.TLS.IsNull() && !f.TLS.IsUnknown() {
		t, d := buildTLS(ctx, f.TLS)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.TLS = t
	}

	return body, diags
}

// BuildBody converts the inline upstream object (as carried by route/service)
// to a wire payload.
func BuildBody(ctx context.Context, obj types.Object, pathBase path.Path) (*Body, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}
	var f Fields
	diags.Append(obj.As(ctx, &f, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	body, d := f.Body(ctx, pathBase)
	diags.Append(d...)
	return body, diags
}

// DecodeFields decodes a raw APISIX upstream JSON object into Fields.
// Optional+Computed attributes that APISIX omitted are backfilled with the
// schema defaults so refreshed state stays plan-stable.
func DecodeFields(ctx context.Context, raw json.RawMessage) (Fields, diag.Diagnostics) {
	var diags diag.Diagnostics
	var body struct {
		Name          string            `json:"name"`
		Desc          string            `json:"desc"`
		Type          string            `json:"type"`
		Nodes         json.RawMessage   `json:"nodes"`
		HealthCheck   json.RawMessage   `json:"checks"`
		Timeout       json.RawMessage   `json:"timeout"`
		Retries       *int64            `json:"retries"`
		RetryTimeout  *int64            `json:"retry_timeout"`
		Scheme        string            `json:"scheme"`
		Labels        map[string]string `json:"labels"`
		ServiceName   string            `json:"service_name"`
		DiscoveryType string            `json:"discovery_type"`
		DiscoveryArgs map[string]string `json:"discovery_args"`
		HashOn        string            `json:"hash_on"`
		Key           string            `json:"key"`
		PassHost      string            `json:"pass_host"`
		UpstreamHost  string            `json:"upstream_host"`
		KeepalivePool json.RawMessage   `json:"keepalive_pool"`
		TLS           json.RawMessage   `json:"tls"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		diags.AddError("Failed to decode upstream response", err.Error())
		return Fields{}, diags
	}

	f := Fields{
		Name:          tfconv.NullableString(body.Name),
		Desc:          tfconv.NullableString(body.Desc),
		Type:          tfconv.StringOrDefault(body.Type, "roundrobin"),
		Scheme:        tfconv.StringOrDefault(body.Scheme, "http"),
		HashOn:        tfconv.StringOrDefault(body.HashOn, "vars"),
		PassHost:      tfconv.StringOrDefault(body.PassHost, "pass"),
		Key:           tfconv.NullableString(body.Key),
		UpstreamHost:  tfconv.NullableString(body.UpstreamHost),
		ServiceName:   tfconv.NullableString(body.ServiceName),
		DiscoveryType: tfconv.NullableString(body.DiscoveryType),
		Retries:       tfconv.OptInt64(body.Retries),
		RetryTimeout:  tfconv.OptInt64(body.RetryTimeout),
		HealthCheck:   tfconv.CanonicalJSON(body.HealthCheck),
	}

	if body.Labels == nil {
		f.Labels = types.MapNull(types.StringType)
	} else {
		v, d := types.MapValueFrom(ctx, types.StringType, body.Labels)
		diags.Append(d...)
		f.Labels = v
	}
	if body.DiscoveryArgs == nil {
		f.DiscoveryArgs = types.MapNull(types.StringType)
	} else {
		v, d := types.MapValueFrom(ctx, types.StringType, body.DiscoveryArgs)
		diags.Append(d...)
		f.DiscoveryArgs = v
	}

	if len(body.Nodes) == 0 || string(body.Nodes) == "null" {
		f.Nodes = types.ListNull(types.ObjectType{AttrTypes: NodeAttrTypes})
	} else {
		v, d := decodeNodes(ctx, body.Nodes)
		diags.Append(d...)
		f.Nodes = v
	}
	if len(body.Timeout) == 0 || string(body.Timeout) == "null" {
		f.Timeout = types.ObjectNull(TimeoutAttrTypes)
	} else {
		v, d := DecodeTimeout(body.Timeout)
		diags.Append(d...)
		f.Timeout = v
	}
	if len(body.KeepalivePool) == 0 || string(body.KeepalivePool) == "null" {
		f.KeepalivePool = types.ObjectNull(KeepaliveAttrTypes)
	} else {
		v, d := decodeKeepalive(body.KeepalivePool)
		diags.Append(d...)
		f.KeepalivePool = v
	}
	if len(body.TLS) == 0 || string(body.TLS) == "null" {
		f.TLS = types.ObjectNull(TLSAttrTypes)
	} else {
		v, d := decodeTLS(body.TLS)
		diags.Append(d...)
		f.TLS = v
	}

	return f, diags
}

// DecodeInto decodes a raw APISIX upstream JSON object into a types.Object
// matching AttrTypes (the inline form carried by route/service).
func DecodeInto(ctx context.Context, raw json.RawMessage) (types.Object, diag.Diagnostics) {
	f, diags := DecodeFields(ctx, raw)
	if diags.HasError() {
		return types.ObjectNull(AttrTypes), diags
	}
	obj, d := types.ObjectValueFrom(ctx, AttrTypes, f)
	diags.Append(d...)
	return obj, diags
}

// BuildTimeout converts a connect/send/read timeout object to wire form,
// omitting fields the user did not set. Also used for the route-level timeout
// attribute, which has the same shape.
func BuildTimeout(ctx context.Context, obj types.Object) (*Timeout, diag.Diagnostics) {
	var diags diag.Diagnostics
	var t struct {
		Connect types.Int64 `tfsdk:"connect"`
		Send    types.Int64 `tfsdk:"send"`
		Read    types.Int64 `tfsdk:"read"`
	}
	diags.Append(obj.As(ctx, &t, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	return &Timeout{
		Connect: tfconv.Int64Ptr(t.Connect),
		Send:    tfconv.Int64Ptr(t.Send),
		Read:    tfconv.Int64Ptr(t.Read),
	}, diags
}

// DecodeTimeout decodes the APISIX timeout object. Fields not present in the
// response are stored as null so they don't appear in plans.
func DecodeTimeout(raw json.RawMessage) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	var t struct {
		Connect *int64 `json:"connect"`
		Send    *int64 `json:"send"`
		Read    *int64 `json:"read"`
	}
	if err := json.Unmarshal(raw, &t); err != nil {
		diags.AddError("Failed to decode timeout", err.Error())
		return types.ObjectNull(TimeoutAttrTypes), diags
	}
	obj, d := types.ObjectValue(TimeoutAttrTypes, map[string]attr.Value{
		"connect": tfconv.OptInt64(t.Connect),
		"send":    tfconv.OptInt64(t.Send),
		"read":    tfconv.OptInt64(t.Read),
	})
	diags.Append(d...)
	return obj, diags
}

// --- internal helpers below ---

func buildNodes(ctx context.Context, list types.List) ([]Node, diag.Diagnostics) {
	var diags diag.Diagnostics
	var nodes []struct {
		Host     types.String `tfsdk:"host"`
		Port     types.Int64  `tfsdk:"port"`
		Weight   types.Int64  `tfsdk:"weight"`
		Priority types.Int64  `tfsdk:"priority"`
		Metadata types.Map    `tfsdk:"metadata"`
	}
	diags.Append(list.ElementsAs(ctx, &nodes, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		node := Node{
			Host:     n.Host.ValueString(),
			Port:     n.Port.ValueInt64(),
			Weight:   n.Weight.ValueInt64(),
			Priority: n.Priority.ValueInt64(),
		}
		if !n.Metadata.IsNull() && !n.Metadata.IsUnknown() {
			md := map[string]string{}
			diags.Append(n.Metadata.ElementsAs(ctx, &md, false)...)
			if diags.HasError() {
				return nil, diags
			}
			node.Metadata = md
		}
		out = append(out, node)
	}
	return out, diags
}

func buildKeepalive(ctx context.Context, obj types.Object) (*Keepalive, diag.Diagnostics) {
	var diags diag.Diagnostics
	var kp struct {
		Size        types.Int64 `tfsdk:"size"`
		IdleTimeout types.Int64 `tfsdk:"idle_timeout"`
		Requests    types.Int64 `tfsdk:"requests"`
	}
	diags.Append(obj.As(ctx, &kp, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	// Inner fields are Optional+Computed+Default, so they're always known
	// after planning.
	return &Keepalive{
		Size:        kp.Size.ValueInt64(),
		IdleTimeout: kp.IdleTimeout.ValueInt64(),
		Requests:    kp.Requests.ValueInt64(),
	}, diags
}

func buildTLS(ctx context.Context, obj types.Object) (*TLS, diag.Diagnostics) {
	var diags diag.Diagnostics
	var t struct {
		ClientCert   types.String `tfsdk:"client_cert"`
		ClientKey    types.String `tfsdk:"client_key"`
		ClientCertID types.String `tfsdk:"client_cert_id"`
		Verify       types.Bool   `tfsdk:"verify"`
	}
	diags.Append(obj.As(ctx, &t, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	return &TLS{
		Verify:       t.Verify.ValueBool(),
		ClientCert:   tfconv.StringPtr(t.ClientCert),
		ClientKey:    tfconv.StringPtr(t.ClientKey),
		ClientCertID: tfconv.StringPtr(t.ClientCertID),
	}, diags
}

func decodeNodes(ctx context.Context, raw json.RawMessage) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	objType := types.ObjectType{AttrTypes: NodeAttrTypes}

	// APISIX returns nodes as either an array (modern) or a map
	// "host:port" -> weight (legacy). We only emit/handle the array form; the
	// legacy form is treated as null.
	var arr []struct {
		Host     string            `json:"host"`
		Port     int64             `json:"port"`
		Weight   int64             `json:"weight"`
		Priority int64             `json:"priority"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &arr); err != nil {
		return types.ListNull(objType), diags
	}

	values := make([]attr.Value, 0, len(arr))
	for _, n := range arr {
		var md attr.Value
		if n.Metadata == nil {
			md = types.MapNull(types.StringType)
		} else {
			v, d := types.MapValueFrom(ctx, types.StringType, n.Metadata)
			diags.Append(d...)
			md = v
		}
		obj, d := types.ObjectValue(NodeAttrTypes, map[string]attr.Value{
			"host":     types.StringValue(n.Host),
			"port":     types.Int64Value(n.Port),
			"weight":   types.Int64Value(n.Weight),
			"priority": types.Int64Value(n.Priority),
			"metadata": md,
		})
		diags.Append(d...)
		values = append(values, obj)
	}
	list, d := types.ListValue(objType, values)
	diags.Append(d...)
	return list, diags
}

func decodeKeepalive(raw json.RawMessage) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	var k struct {
		Size        *int64 `json:"size"`
		IdleTimeout *int64 `json:"idle_timeout"`
		Requests    *int64 `json:"requests"`
	}
	if err := json.Unmarshal(raw, &k); err != nil {
		diags.AddError("Failed to decode keepalive_pool", err.Error())
		return types.ObjectNull(KeepaliveAttrTypes), diags
	}
	obj, d := types.ObjectValue(KeepaliveAttrTypes, map[string]attr.Value{
		"size":         tfconv.Int64OrDefault(k.Size, 320),
		"idle_timeout": tfconv.Int64OrDefault(k.IdleTimeout, 60),
		"requests":     tfconv.Int64OrDefault(k.Requests, 1000),
	})
	diags.Append(d...)
	return obj, diags
}

func decodeTLS(raw json.RawMessage) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	var t struct {
		ClientCert   *string `json:"client_cert"`
		ClientKey    *string `json:"client_key"`
		ClientCertID *string `json:"client_cert_id"`
		Verify       *bool   `json:"verify"`
	}
	if err := json.Unmarshal(raw, &t); err != nil {
		diags.AddError("Failed to decode tls", err.Error())
		return types.ObjectNull(TLSAttrTypes), diags
	}
	verify := false
	if t.Verify != nil {
		verify = *t.Verify
	}
	obj, d := types.ObjectValue(TLSAttrTypes, map[string]attr.Value{
		"client_cert":    tfconv.OptString(t.ClientCert),
		"client_key":     tfconv.OptString(t.ClientKey),
		"client_cert_id": tfconv.OptString(t.ClientCertID),
		"verify":         types.BoolValue(verify),
	})
	diags.Append(d...)
	return obj, diags
}
