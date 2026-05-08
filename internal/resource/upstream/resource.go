// Package upstream implements the apisix_upstream resource.
//
// Upstream is the largest schema in the provider. Schema notes:
//   - id is the URL key (Required + RequiresReplace).
//   - type / scheme / pass_host / hash_on are Optional+Computed+Default with
//     validators bound to the APISIX-supported value sets.
//   - nodes is a ListNestedAttribute. host and port are Required; weight and
//     priority have explicit Defaults so server-side echoing stays in sync
//     with the plan.
//   - timeout / keepalive_pool / tls are SingleNestedAttribute (modern
//     Framework idiom replacing the SDK v2 MaxItems:1 list hack). For
//     keepalive_pool and tls.verify, inner fields have explicit Defaults so a
//     partial block plans cleanly. timeout fields are pure Optional and rely
//     on APISIX's sparse-storage behavior — if drift appears in practice,
//     swap to Optional+Computed and add a read-after-create.
//   - health_check is a JSON string with the same JSON-equivalence suppress
//     modifier used for plugins. Native modeling can be revisited later.
package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/client"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/planmodifier/jsonstring"
)

const apiKind = "upstreams"

var (
	validTypes     = []string{"roundrobin", "chash", "ewma", "least_conn"}
	validSchemes   = []string{"grpc", "grpcs", "http", "https", "tcp", "tls", "udp", "kafka"}
	validHashOn    = []string{"vars", "header", "cookie", "consumer", "vars_combinations"}
	validPassHosts = []string{"pass", "node", "rewrite"}
)

var (
	_ resource.Resource                = (*Resource)(nil)
	_ resource.ResourceWithConfigure   = (*Resource)(nil)
	_ resource.ResourceWithImportState = (*Resource)(nil)
)

// Resource is the apisix_upstream resource.
type Resource struct {
	client *client.Client
}

// NewResource is the constructor registered with the provider.
func NewResource() resource.Resource { return &Resource{} }

type model struct {
	ID            types.String `tfsdk:"id"`
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

var nodeAttrTypes = map[string]attr.Type{
	"host":     types.StringType,
	"port":     types.Int64Type,
	"weight":   types.Int64Type,
	"priority": types.Int64Type,
	"metadata": types.MapType{ElemType: types.StringType},
}

var timeoutAttrTypes = map[string]attr.Type{
	"connect": types.Int64Type,
	"send":    types.Int64Type,
	"read":    types.Int64Type,
}

var keepaliveAttrTypes = map[string]attr.Type{
	"size":         types.Int64Type,
	"idle_timeout": types.Int64Type,
	"requests":     types.Int64Type,
}

var tlsAttrTypes = map[string]attr.Type{
	"client_cert":    types.StringType,
	"client_key":     types.StringType,
	"client_cert_id": types.StringType,
	"verify":         types.BoolType,
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_upstream"
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
		Description: "Manages an APISIX Upstream: a backend definition (one or more nodes, or a service-discovery reference) used by routes and services.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "Unique upstream identifier. Used as the APISIX object key. Changing this forces replacement.",
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
			"type": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("roundrobin"),
				Description: "Load-balancing algorithm. Defaults to roundrobin.",
				Validators: []validator.String{
					stringvalidator.OneOf(validTypes...),
				},
			},
			"nodes": schema.ListNestedAttribute{
				Optional:    true,
				Description: "Backend nodes. Required when not using service discovery.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"host": schema.StringAttribute{
							Required:    true,
							Description: "Node hostname or IP.",
						},
						"port": schema.Int64Attribute{
							Required: true,
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
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
				Validators: []validator.String{
					stringvalidator.OneOf(validSchemes...),
				},
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
				Description: "Service discovery type (e.g. consul, nacos, eureka).",
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
				Validators: []validator.String{
					stringvalidator.OneOf(validHashOn...),
				},
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
		},
	}
}

// apiBody is the wire payload. Pointers separate "unset" from "zero".
type apiBody struct {
	ID            string                     `json:"id"`
	Name          *string                    `json:"name,omitempty"`
	Desc          *string                    `json:"desc,omitempty"`
	Type          *string                    `json:"type,omitempty"`
	Nodes         []apiNode                  `json:"nodes,omitempty"`
	HealthCheck   json.RawMessage            `json:"checks,omitempty"`
	Timeout       *apiTimeout                `json:"timeout,omitempty"`
	Retries       *int64                     `json:"retries,omitempty"`
	RetryTimeout  *int64                     `json:"retry_timeout,omitempty"`
	Scheme        *string                    `json:"scheme,omitempty"`
	Labels        map[string]string          `json:"labels,omitempty"`
	ServiceName   *string                    `json:"service_name,omitempty"`
	DiscoveryType *string                    `json:"discovery_type,omitempty"`
	DiscoveryArgs map[string]string          `json:"discovery_args,omitempty"`
	HashOn        *string                    `json:"hash_on,omitempty"`
	Key           *string                    `json:"key,omitempty"`
	PassHost      *string                    `json:"pass_host,omitempty"`
	UpstreamHost  *string                    `json:"upstream_host,omitempty"`
	KeepalivePool *apiKeepalive              `json:"keepalive_pool,omitempty"`
	TLS           *apiTLS                    `json:"tls,omitempty"`
}

type apiNode struct {
	Host     string            `json:"host"`
	Port     int64             `json:"port"`
	Weight   int64             `json:"weight"`
	Priority int64             `json:"priority"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type apiTimeout struct {
	Connect *int64 `json:"connect,omitempty"`
	Send    *int64 `json:"send,omitempty"`
	Read    *int64 `json:"read,omitempty"`
}

type apiKeepalive struct {
	Size        int64 `json:"size"`
	IdleTimeout int64 `json:"idle_timeout"`
	Requests    int64 `json:"requests"`
}

type apiTLS struct {
	ClientCert   *string `json:"client_cert,omitempty"`
	ClientKey    *string `json:"client_key,omitempty"`
	ClientCertID *string `json:"client_cert_id,omitempty"`
	Verify       bool    `json:"verify"`
}

func (r *Resource) buildBody(ctx context.Context, m *model) (*apiBody, diag.Diagnostics) {
	var diags diag.Diagnostics
	body := &apiBody{ID: m.ID.ValueString()}

	stringPtr := func(s types.String) *string {
		if s.IsNull() || s.IsUnknown() {
			return nil
		}
		v := s.ValueString()
		return &v
	}
	int64Ptr := func(i types.Int64) *int64 {
		if i.IsNull() || i.IsUnknown() {
			return nil
		}
		v := i.ValueInt64()
		return &v
	}

	body.Name = stringPtr(m.Name)
	body.Desc = stringPtr(m.Desc)
	body.Type = stringPtr(m.Type)
	body.Scheme = stringPtr(m.Scheme)
	body.HashOn = stringPtr(m.HashOn)
	body.Key = stringPtr(m.Key)
	body.PassHost = stringPtr(m.PassHost)
	body.UpstreamHost = stringPtr(m.UpstreamHost)
	body.ServiceName = stringPtr(m.ServiceName)
	body.DiscoveryType = stringPtr(m.DiscoveryType)
	body.Retries = int64Ptr(m.Retries)
	body.RetryTimeout = int64Ptr(m.RetryTimeout)

	if !m.Labels.IsNull() && !m.Labels.IsUnknown() {
		labels := map[string]string{}
		diags.Append(m.Labels.ElementsAs(ctx, &labels, false)...)
		if diags.HasError() {
			return nil, diags
		}
		body.Labels = labels
	}
	if !m.DiscoveryArgs.IsNull() && !m.DiscoveryArgs.IsUnknown() {
		args := map[string]string{}
		diags.Append(m.DiscoveryArgs.ElementsAs(ctx, &args, false)...)
		if diags.HasError() {
			return nil, diags
		}
		body.DiscoveryArgs = args
	}

	if !m.HealthCheck.IsNull() && !m.HealthCheck.IsUnknown() {
		raw := m.HealthCheck.ValueString()
		var probe any
		if err := json.Unmarshal([]byte(raw), &probe); err != nil {
			diags.AddAttributeError(
				path.Root("health_check"),
				"Invalid health_check JSON",
				err.Error(),
			)
			return nil, diags
		}
		body.HealthCheck = json.RawMessage(raw)
	}

	if !m.Nodes.IsNull() && !m.Nodes.IsUnknown() {
		nodes, d := buildNodes(ctx, m.Nodes)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.Nodes = nodes
	}

	if !m.Timeout.IsNull() && !m.Timeout.IsUnknown() {
		t, d := buildTimeout(ctx, m.Timeout)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.Timeout = t
	}

	if !m.KeepalivePool.IsNull() && !m.KeepalivePool.IsUnknown() {
		kp, d := buildKeepalive(ctx, m.KeepalivePool)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.KeepalivePool = kp
	}

	if !m.TLS.IsNull() && !m.TLS.IsUnknown() {
		t, d := buildTLS(ctx, m.TLS)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		body.TLS = t
	}

	return body, diags
}

func buildNodes(ctx context.Context, list types.List) ([]apiNode, diag.Diagnostics) {
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

	out := make([]apiNode, 0, len(nodes))
	for _, n := range nodes {
		node := apiNode{
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

func buildTimeout(ctx context.Context, obj types.Object) (*apiTimeout, diag.Diagnostics) {
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
	out := &apiTimeout{}
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

func buildKeepalive(ctx context.Context, obj types.Object) (*apiKeepalive, diag.Diagnostics) {
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
	return &apiKeepalive{
		Size:        kp.Size.ValueInt64(),
		IdleTimeout: kp.IdleTimeout.ValueInt64(),
		Requests:    kp.Requests.ValueInt64(),
	}, diags
}

func buildTLS(ctx context.Context, obj types.Object) (*apiTLS, diag.Diagnostics) {
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
	out := &apiTLS{Verify: t.Verify.ValueBool()}
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
		resp.Diagnostics.AddError("Failed to create upstream", err.Error())
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
		resp.Diagnostics.AddError("Failed to read upstream", err.Error())
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
		resp.Diagnostics.AddError("Failed to update upstream", err.Error())
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
		resp.Diagnostics.AddError("Failed to delete upstream", err.Error())
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func decodeInto(ctx context.Context, raw json.RawMessage, m *model) diag.Diagnostics {
	var diags diag.Diagnostics
	var body struct {
		ID            string            `json:"id"`
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
		return diags
	}

	m.ID = types.StringValue(body.ID)
	m.Name = nullableString(body.Name)
	m.Desc = nullableString(body.Desc)
	m.Key = nullableString(body.Key)
	m.UpstreamHost = nullableString(body.UpstreamHost)
	m.ServiceName = nullableString(body.ServiceName)
	m.DiscoveryType = nullableString(body.DiscoveryType)

	// Optional+Computed+Default fields: APISIX echoes them; if missing, use the default.
	m.Type = stringOrDefault(body.Type, "roundrobin")
	m.Scheme = stringOrDefault(body.Scheme, "http")
	m.HashOn = stringOrDefault(body.HashOn, "vars")
	m.PassHost = stringOrDefault(body.PassHost, "pass")

	m.Retries = optInt64(body.Retries)
	m.RetryTimeout = optInt64(body.RetryTimeout)

	if body.Labels == nil {
		m.Labels = types.MapNull(types.StringType)
	} else {
		v, d := types.MapValueFrom(ctx, types.StringType, body.Labels)
		diags.Append(d...)
		m.Labels = v
	}
	if body.DiscoveryArgs == nil {
		m.DiscoveryArgs = types.MapNull(types.StringType)
	} else {
		v, d := types.MapValueFrom(ctx, types.StringType, body.DiscoveryArgs)
		diags.Append(d...)
		m.DiscoveryArgs = v
	}

	if len(body.HealthCheck) == 0 || string(body.HealthCheck) == "null" {
		m.HealthCheck = types.StringNull()
	} else {
		// Re-marshal canonically (Go sorts map keys); the plan modifier handles
		// the plan/state JSON-equivalence comparison.
		var v any
		if err := json.Unmarshal(body.HealthCheck, &v); err == nil {
			canon, err2 := json.Marshal(v)
			if err2 == nil {
				m.HealthCheck = types.StringValue(string(canon))
			} else {
				m.HealthCheck = types.StringValue(string(body.HealthCheck))
			}
		} else {
			m.HealthCheck = types.StringValue(string(body.HealthCheck))
		}
	}

	if len(body.Nodes) == 0 || string(body.Nodes) == "null" {
		m.Nodes = types.ListNull(types.ObjectType{AttrTypes: nodeAttrTypes})
	} else {
		v, d := decodeNodes(ctx, body.Nodes)
		diags.Append(d...)
		m.Nodes = v
	}

	if len(body.Timeout) == 0 || string(body.Timeout) == "null" {
		m.Timeout = types.ObjectNull(timeoutAttrTypes)
	} else {
		v, d := decodeTimeout(body.Timeout)
		diags.Append(d...)
		m.Timeout = v
	}

	if len(body.KeepalivePool) == 0 || string(body.KeepalivePool) == "null" {
		m.KeepalivePool = types.ObjectNull(keepaliveAttrTypes)
	} else {
		v, d := decodeKeepalive(body.KeepalivePool)
		diags.Append(d...)
		m.KeepalivePool = v
	}

	if len(body.TLS) == 0 || string(body.TLS) == "null" {
		m.TLS = types.ObjectNull(tlsAttrTypes)
	} else {
		v, d := decodeTLS(body.TLS)
		diags.Append(d...)
		m.TLS = v
	}

	return diags
}

func decodeNodes(ctx context.Context, raw json.RawMessage) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	objType := types.ObjectType{AttrTypes: nodeAttrTypes}

	// APISIX returns nodes as either an array (modern) or a map "host:port" -> weight (legacy).
	// We only emit/handle the array form; the legacy form is treated as null.
	var arr []struct {
		Host     string            `json:"host"`
		Port     int64             `json:"port"`
		Weight   int64             `json:"weight"`
		Priority int64             `json:"priority"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &arr); err != nil {
		// Legacy map form or unknown shape — keep state empty.
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
		obj, d := types.ObjectValue(nodeAttrTypes, map[string]attr.Value{
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
		return types.ObjectNull(timeoutAttrTypes), diags
	}
	obj, d := types.ObjectValue(timeoutAttrTypes, map[string]attr.Value{
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
		return types.ObjectNull(keepaliveAttrTypes), diags
	}
	obj, d := types.ObjectValue(keepaliveAttrTypes, map[string]attr.Value{
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
		return types.ObjectNull(tlsAttrTypes), diags
	}
	verify := false
	if t.Verify != nil {
		verify = *t.Verify
	}
	obj, d := types.ObjectValue(tlsAttrTypes, map[string]attr.Value{
		"client_cert":    optString(t.ClientCert),
		"client_key":     optString(t.ClientKey),
		"client_cert_id": optString(t.ClientCertID),
		"verify":         types.BoolValue(verify),
	})
	diags.Append(d...)
	return obj, diags
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
