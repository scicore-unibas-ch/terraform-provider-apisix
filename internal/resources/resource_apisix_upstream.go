package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceApisixUpstream() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an APISIX Upstream resource.",

		CreateContext: resourceApisixUpstreamCreate,
		ReadContext:   resourceApisixUpstreamRead,
		UpdateContext: resourceApisixUpstreamUpdate,
		DeleteContext: resourceApisixUpstreamDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the upstream.",
			},
			"desc": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the upstream.",
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "roundrobin",
				Description: "Load balancing algorithm. Valid values: roundrobin, chash, ewma, least_conn.",
				ValidateFunc: validation.StringInSlice([]string{
					"roundrobin",
					"chash",
					"ewma",
					"least_conn",
				}, false),
			},
			"nodes": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of upstream nodes. Required when not using service discovery.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"host": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Hostname or IP of the node.",
						},
						"port": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Port of the node. Optional for some node types.",
						},
						"weight": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     1,
							Description: "Weight of the node for load balancing. Defaults to 1.",
						},
						"priority": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     0,
							Description: "Priority of the node. Nodes with lower priority are tried first. Defaults to 0.",
						},
						"metadata": {
							Type:        schema.TypeMap,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "Metadata for the node.",
						},
					},
				},
			},
			"health_check": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "JSON-encoded health check configuration.",
			},
			"timeout": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Timeout configuration for the upstream.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"connect": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Connect timeout in seconds.",
						},
						"send": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Send timeout in seconds.",
						},
						"read": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Read timeout in seconds.",
						},
					},
				},
			},
			"retries": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Number of retries for the upstream.",
			},
			"retry_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Timeout for retries in seconds.",
			},
			"scheme": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "http",
				Description: "Scheme to use when communicating with the upstream. Valid values: grpc, grpcs, http, https, tcp, tls, udp, kafka.",
				ValidateFunc: validation.StringInSlice([]string{
					"grpc",
					"grpcs",
					"http",
					"https",
					"tcp",
					"tls",
					"udp",
					"kafka",
				}, false),
			},
			"labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Labels for the upstream as key-value pairs.",
			},
			"service_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Service name for service discovery. Required when using service discovery.",
			},
			"discovery_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Type of service discovery. Required when using service discovery.",
			},
			"discovery_args": {
				Type:        schema.TypeMap,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Arguments for service discovery (namespace_id, group_name).",
			},
			"hash_on": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "vars",
				Description: "Hash on parameter for chash load balancing. Valid values: vars, header, cookie, consumer, vars_combinations.",
				ValidateFunc: validation.StringInSlice([]string{
					"vars",
					"header",
					"cookie",
					"consumer",
					"vars_combinations",
				}, false),
			},
			"key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The key for chash load balancing (e.g., remote_addr, uri, arg_name).",
			},
			"pass_host": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "pass",
				Description: "Mode of host passing. Valid values: pass, node, rewrite.",
				ValidateFunc: validation.StringInSlice([]string{
					"pass",
					"node",
					"rewrite",
				}, false),
			},
			"upstream_host": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Custom host for the upstream request. Required when pass_host is 'rewrite'.",
			},
			"keepalive_pool": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Keepalive pool configuration for the upstream.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"size": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     320,
							Description: "Size of the keepalive pool. Defaults to 320.",
						},
						"idle_timeout": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     60,
							Description: "Idle timeout for keepalive connections in seconds. Defaults to 60.",
						},
						"requests": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     1000,
							Description: "Maximum number of requests per connection. Defaults to 1000.",
						},
					},
				},
			},
			"tls": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "TLS client certificate configuration for mTLS.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"client_cert": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "Client certificate content for mTLS.",
						},
						"client_key": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "Client private key content for mTLS.",
						},
						"client_cert_id": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Reference to SSL object for client certificate.",
						},
						"verify": {
							Type:        schema.TypeBool,
							Optional:    true,
							Default:     false,
							Description: "Enable server certificate verification. Currently only for kafka upstream.",
						},
					},
				},
			},
		},
	}
}

func resourceApisixUpstreamCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: Implement create
	return diag.Diagnostics{}
}

func resourceApisixUpstreamRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: Implement read
	return diag.Diagnostics{}
}

func resourceApisixUpstreamUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: Implement update
	return diag.Diagnostics{}
}

func resourceApisixUpstreamDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO: Implement delete
	return diag.Diagnostics{}
}
