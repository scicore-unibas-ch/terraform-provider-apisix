// Package inlineupstream defines the schema, wire format, and codec for the
// inline upstream block embedded inside apisix_route and apisix_service.
//
// The inline upstream supports the same fields as the standalone apisix_upstream
// resource minus the id (which doesn't exist for inline objects). Sharing this
// definition keeps route, service, and the standalone resource consistent —
// users get the full APISIX upstream surface whether they reference an upstream
// by id or define it inline.
//
// Note: the standalone apisix_upstream resource currently has its own
// independent implementation. If you change attributes here, mirror the change
// in internal/resource/upstream/resource.go.
package inlineupstream

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/planmodifier/jsonstring"
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

// SchemaAttrs returns the inline upstream attribute set. Callers compose this
// into a SingleNestedAttribute of the appropriate optionality.
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
			Description: "Backend nodes.",
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
			Validators:  []validator.String{stringvalidator.OneOf(validPassHosts...)},
		},
		"upstream_host": schema.StringAttribute{
			Optional:    true,
			Description: "Custom Host header. Required when pass_host = rewrite.",
		},
		"keepalive_pool": schema.SingleNestedAttribute{
			Optional:    true,
			Description: "Keepalive pool configuration.",
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
				"client_cert":    schema.StringAttribute{Optional: true, Sensitive: true, Description: "Client certificate (PEM)."},
				"client_key":     schema.StringAttribute{Optional: true, Sensitive: true, Description: "Client private key (PEM)."},
				"client_cert_id": schema.StringAttribute{Optional: true, Description: "Reference to an apisix_ssl object."},
				"verify": schema.BoolAttribute{
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "Verify the server certificate. Defaults to false.",
				},
			},
		},
	}
}

// Body is the wire payload for an inline upstream.
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

// BuildBody converts the Terraform inline upstream object to a wire payload.
// pathBase is used to prefix attribute paths in error diagnostics so the user
// sees errors like "upstream.health_check" rather than just "health_check".
func BuildBody(ctx context.Context, obj types.Object, pathBase path.Path) (*Body, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}

	var u struct {
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
	diags.Append(obj.As(ctx, &u, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	body := &Body{
		Name:          stringPtr(u.Name),
		Desc:          stringPtr(u.Desc),
		Type:          stringPtr(u.Type),
		Scheme:        stringPtr(u.Scheme),
		HashOn:        stringPtr(u.HashOn),
		Key:           stringPtr(u.Key),
		PassHost:      stringPtr(u.PassHost),
		UpstreamHost:  stringPtr(u.UpstreamHost),
		ServiceName:   stringPtr(u.ServiceName),
		DiscoveryType: stringPtr(u.DiscoveryType),
		Retries:       int64Ptr(u.Retries),
		RetryTimeout:  int64Ptr(u.RetryTimeout),
	}

	if !u.Labels.IsNull() && !u.Labels.IsUnknown() {
		labels := map[string]string{}
		diags.Append(u.Labels.ElementsAs(ctx, &labels, false)...)
		if diags.HasError() {
			return nil, diags
		}
		body.Labels = labels
	}
	if !u.DiscoveryArgs.IsNull() && !u.DiscoveryArgs.IsUnknown() {
		args := map[string]string{}
		diags.Append(u.DiscoveryArgs.ElementsAs(ctx, &args, false)...)
		if diags.HasError() {
			return nil, diags
		}
		body.DiscoveryArgs = args
	}
	if !u.HealthCheck.IsNull() && !u.HealthCheck.IsUnknown() {
		raw := u.HealthCheck.ValueString()
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
	if !u.Nodes.IsNull() && !u.Nodes.IsUnknown() {
		nodes, d := buildNodes(ctx, u.Nodes)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.Nodes = nodes
	}
	if !u.Timeout.IsNull() && !u.Timeout.IsUnknown() {
		t, d := buildTimeout(ctx, u.Timeout)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.Timeout = t
	}
	if !u.KeepalivePool.IsNull() && !u.KeepalivePool.IsUnknown() {
		kp, d := buildKeepalive(ctx, u.KeepalivePool)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.KeepalivePool = kp
	}
	if !u.TLS.IsNull() && !u.TLS.IsUnknown() {
		t, d := buildTLS(ctx, u.TLS)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.TLS = t
	}

	return body, diags
}

// DecodeInto decodes a raw APISIX upstream JSON object into a types.Object
// matching AttrTypes.
func DecodeInto(ctx context.Context, raw json.RawMessage) (types.Object, diag.Diagnostics) {
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
		diags.AddError("Failed to decode upstream", err.Error())
		return types.ObjectNull(AttrTypes), diags
	}

	attrs := map[string]attr.Value{
		"name":           nullableString(body.Name),
		"desc":           nullableString(body.Desc),
		"type":           stringOrDefault(body.Type, "roundrobin"),
		"scheme":         stringOrDefault(body.Scheme, "http"),
		"hash_on":        stringOrDefault(body.HashOn, "vars"),
		"pass_host":      stringOrDefault(body.PassHost, "pass"),
		"key":            nullableString(body.Key),
		"upstream_host":  nullableString(body.UpstreamHost),
		"service_name":   nullableString(body.ServiceName),
		"discovery_type": nullableString(body.DiscoveryType),
		"retries":        optInt64(body.Retries),
		"retry_timeout":  optInt64(body.RetryTimeout),
	}

	if body.Labels == nil {
		attrs["labels"] = types.MapNull(types.StringType)
	} else {
		v, d := types.MapValueFrom(ctx, types.StringType, body.Labels)
		diags.Append(d...)
		attrs["labels"] = v
	}
	if body.DiscoveryArgs == nil {
		attrs["discovery_args"] = types.MapNull(types.StringType)
	} else {
		v, d := types.MapValueFrom(ctx, types.StringType, body.DiscoveryArgs)
		diags.Append(d...)
		attrs["discovery_args"] = v
	}

	if len(body.HealthCheck) == 0 || string(body.HealthCheck) == "null" {
		attrs["health_check"] = types.StringNull()
	} else {
		var v any
		if err := json.Unmarshal(body.HealthCheck, &v); err == nil {
			canon, err2 := json.Marshal(v)
			if err2 == nil {
				attrs["health_check"] = types.StringValue(string(canon))
			} else {
				attrs["health_check"] = types.StringValue(string(body.HealthCheck))
			}
		} else {
			attrs["health_check"] = types.StringValue(string(body.HealthCheck))
		}
	}

	if len(body.Nodes) == 0 || string(body.Nodes) == "null" {
		attrs["nodes"] = types.ListNull(types.ObjectType{AttrTypes: NodeAttrTypes})
	} else {
		v, d := decodeNodes(ctx, body.Nodes)
		diags.Append(d...)
		attrs["nodes"] = v
	}
	if len(body.Timeout) == 0 || string(body.Timeout) == "null" {
		attrs["timeout"] = types.ObjectNull(TimeoutAttrTypes)
	} else {
		v, d := decodeTimeout(body.Timeout)
		diags.Append(d...)
		attrs["timeout"] = v
	}
	if len(body.KeepalivePool) == 0 || string(body.KeepalivePool) == "null" {
		attrs["keepalive_pool"] = types.ObjectNull(KeepaliveAttrTypes)
	} else {
		v, d := decodeKeepalive(body.KeepalivePool)
		diags.Append(d...)
		attrs["keepalive_pool"] = v
	}
	if len(body.TLS) == 0 || string(body.TLS) == "null" {
		attrs["tls"] = types.ObjectNull(TLSAttrTypes)
	} else {
		v, d := decodeTLS(body.TLS)
		diags.Append(d...)
		attrs["tls"] = v
	}

	obj, d := types.ObjectValue(AttrTypes, attrs)
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

func buildTimeout(ctx context.Context, obj types.Object) (*Timeout, diag.Diagnostics) {
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
	out := &Timeout{}
	if !t.Connect.IsNull() && !t.Connect.IsUnknown() {
		v := t.Connect.ValueInt64()
		out.Connect = &v
	}
	if !t.Send.IsNull() && !t.Send.IsUnknown() {
		v := t.Send.ValueInt64()
		out.Send = &v
	}
	if !t.Read.IsNull() && !t.Read.IsUnknown() {
		v := t.Read.ValueInt64()
		out.Read = &v
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
	out := &TLS{Verify: t.Verify.ValueBool()}
	if !t.ClientCert.IsNull() && !t.ClientCert.IsUnknown() {
		v := t.ClientCert.ValueString()
		out.ClientCert = &v
	}
	if !t.ClientKey.IsNull() && !t.ClientKey.IsUnknown() {
		v := t.ClientKey.ValueString()
		out.ClientKey = &v
	}
	if !t.ClientCertID.IsNull() && !t.ClientCertID.IsUnknown() {
		v := t.ClientCertID.ValueString()
		out.ClientCertID = &v
	}
	return out, diags
}

func decodeNodes(ctx context.Context, raw json.RawMessage) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	objType := types.ObjectType{AttrTypes: NodeAttrTypes}

	var arr []struct {
		Host     string            `json:"host"`
		Port     int64             `json:"port"`
		Weight   int64             `json:"weight"`
		Priority int64             `json:"priority"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &arr); err != nil {
		// Legacy host:port→weight map form is not modeled.
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

func decodeTimeout(raw json.RawMessage) (types.Object, diag.Diagnostics) {
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
		"connect": optInt64(t.Connect),
		"send":    optInt64(t.Send),
		"read":    optInt64(t.Read),
	})
	diags.Append(d...)
	return obj, diags
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
		"size":         int64OrDefault(k.Size, 320),
		"idle_timeout": int64OrDefault(k.IdleTimeout, 60),
		"requests":     int64OrDefault(k.Requests, 1000),
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
		"client_cert":    optString(t.ClientCert),
		"client_key":     optString(t.ClientKey),
		"client_cert_id": optString(t.ClientCertID),
		"verify":         types.BoolValue(verify),
	})
	diags.Append(d...)
	return obj, diags
}

func stringPtr(s types.String) *string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	v := s.ValueString()
	return &v
}

func int64Ptr(i types.Int64) *int64 {
	if i.IsNull() || i.IsUnknown() {
		return nil
	}
	v := i.ValueInt64()
	return &v
}

func nullableString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func stringOrDefault(s, def string) types.String {
	if s == "" {
		return types.StringValue(def)
	}
	return types.StringValue(s)
}

func optInt64(p *int64) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*p)
}

func int64OrDefault(p *int64, def int64) types.Int64 {
	if p == nil {
		return types.Int64Value(def)
	}
	return types.Int64Value(*p)
}

func optString(p *string) types.String {
	if p == nil || *p == "" {
		return types.StringNull()
	}
	return types.StringValue(*p)
}
