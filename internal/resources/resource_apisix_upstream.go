package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/apisix"
)

// getClient retrieves the APISIX client from provider meta

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
				Computed:    true,
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
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	upstream := expandUpstream(d)
	id := d.Get("name").(string)
	if id == "" {
		id = fmt.Sprintf("upstream-%d", time.Now().UnixNano())
	}

	err = client.Create(ctx, "upstreams", id, upstream)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create upstream: %w", err))
	}

	d.SetId(id)
	return resourceApisixUpstreamRead(ctx, d, meta)
}

func resourceApisixUpstreamRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data, err := client.Read(ctx, "upstreams", d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read upstream: %w", err))
	}

	var resp apisix.APISIXResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	return flattenUpstream(d, resp.Value)
}

func resourceApisixUpstreamUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	upstream := expandUpstream(d)
	err = client.Update(ctx, "upstreams", d.Id(), upstream)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update upstream: %w", err))
	}

	return resourceApisixUpstreamRead(ctx, d, meta)
}

func resourceApisixUpstreamDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Delete(ctx, "upstreams", d.Id(), false)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete upstream: %w", err))
	}

	d.SetId("")
	return nil
}

// expandUpstream converts Terraform resource data to APISIX Upstream struct
func expandUpstream(d *schema.ResourceData) *apisix.Upstream {
	upstream := &apisix.Upstream{
		Type:   d.Get("type").(string),
		Scheme: d.Get("scheme").(string),
	}

	if v, ok := d.GetOk("name"); ok {
		upstream.Name = v.(string)
	}
	if v, ok := d.GetOk("desc"); ok {
		upstream.Desc = v.(string)
	}
	if v, ok := d.GetOk("nodes"); ok {
		upstream.Nodes = expandNodes(v.([]interface{}))
	}
	if v, ok := d.GetOk("health_check"); ok {
		var checks map[string]interface{}
		if err := json.Unmarshal([]byte(v.(string)), &checks); err == nil {
			upstream.HealthCheck = checks
		}
	}
	if v, ok := d.GetOk("timeout"); ok {
		upstream.Timeout = expandTimeout(v.([]interface{}))
	}
	if v, ok := d.GetOk("retries"); ok {
		upstream.Retries = v.(int)
	}
	if v, ok := d.GetOk("retry_timeout"); ok {
		upstream.RetryTimeout = v.(int)
	}
	if v, ok := d.GetOk("labels"); ok {
		labels := make(map[string]string)
		for k, v := range v.(map[string]interface{}) {
			labels[k] = v.(string)
		}
		upstream.Labels = labels
	}
	if v, ok := d.GetOk("service_name"); ok {
		upstream.ServiceName = v.(string)
	}
	if v, ok := d.GetOk("discovery_type"); ok {
		upstream.DiscoveryType = v.(string)
	}
	if v, ok := d.GetOk("discovery_args"); ok {
		args := make(map[string]string)
		for k, v := range v.(map[string]interface{}) {
			args[k] = v.(string)
		}
		upstream.DiscoveryArgs = args
	}
	if v, ok := d.GetOk("hash_on"); ok {
		upstream.HashOn = v.(string)
	}
	if v, ok := d.GetOk("key"); ok {
		upstream.Key = v.(string)
	}
	if v, ok := d.GetOk("pass_host"); ok {
		upstream.PassHost = v.(string)
	}
	if v, ok := d.GetOk("upstream_host"); ok {
		upstream.UpstreamHost = v.(string)
	}
	if v, ok := d.GetOk("keepalive_pool"); ok {
		upstream.KeepalivePool = expandKeepalivePool(v.([]interface{}))
	}
	if v, ok := d.GetOk("tls"); ok {
		upstream.TLS = expandTLS(v.([]interface{}))
	}

	return upstream
}

// expandNodes converts Terraform nodes list to APISIX UpstreamNode slice
func expandNodes(nodes []interface{}) []apisix.UpstreamNode {
	result := make([]apisix.UpstreamNode, 0, len(nodes))
	for _, node := range nodes {
		n := node.(map[string]interface{})
		upstreamNode := apisix.UpstreamNode{
			Host:   n["host"].(string),
			Weight: n["weight"].(int),
		}
		if v, ok := n["port"].(int); ok && v > 0 {
			upstreamNode.Port = v
		}
		if v, ok := n["priority"].(int); ok {
			upstreamNode.Priority = v
		}
		if v, ok := n["metadata"].(map[string]interface{}); ok && len(v) > 0 {
			metadata := make(map[string]string)
			for k, val := range v {
				metadata[k] = val.(string)
			}
			upstreamNode.Metadata = metadata
		}
		result = append(result, upstreamNode)
	}
	return result
}

// expandTimeout converts Terraform timeout block to APISIX UpstreamTimeout
func expandTimeout(timeouts []interface{}) *apisix.UpstreamTimeout {
	if len(timeouts) == 0 || timeouts[0] == nil {
		return nil
	}
	t := timeouts[0].(map[string]interface{})
	return &apisix.UpstreamTimeout{
		Connect: t["connect"].(int),
		Send:    t["send"].(int),
		Read:    t["read"].(int),
	}
}

// expandKeepalivePool converts Terraform keepalive_pool block to map
func expandKeepalivePool(pool []interface{}) map[string]interface{} {
	if len(pool) == 0 || pool[0] == nil {
		return nil
	}
	p := pool[0].(map[string]interface{})
	return map[string]interface{}{
		"size":         p["size"].(int),
		"idle_timeout": p["idle_timeout"].(int),
		"requests":     p["requests"].(int),
	}
}

// expandTLS converts Terraform tls block to map
func expandTLS(tls []interface{}) map[string]interface{} {
	if len(tls) == 0 || tls[0] == nil {
		return nil
	}
	t := tls[0].(map[string]interface{})
	result := make(map[string]interface{})
	if v, ok := t["client_cert"].(string); ok && v != "" {
		result["client_cert"] = v
	}
	if v, ok := t["client_key"].(string); ok && v != "" {
		result["client_key"] = v
	}
	if v, ok := t["client_cert_id"].(string); ok && v != "" {
		result["client_cert_id"] = v
	}
	if v, ok := t["verify"].(bool); ok {
		result["verify"] = v
	}
	return result
}

// flattenUpstream sets Terraform state from APISIX upstream response
func flattenUpstream(d *schema.ResourceData, value interface{}) diag.Diagnostics {
	data, ok := value.(map[string]interface{})
	if !ok {
		return diag.Errorf("failed to convert upstream data: got %T, expected map[string]interface{}", value)
	}

	// Set simple fields
	d.Set("name", data["name"])
	d.Set("desc", data["desc"])
	d.Set("type", data["type"])
	d.Set("scheme", data["scheme"])
	d.Set("hash_on", data["hash_on"])
	d.Set("key", data["key"])
	d.Set("pass_host", data["pass_host"])
	d.Set("upstream_host", data["upstream_host"])
	d.Set("service_name", data["service_name"])
	d.Set("discovery_type", data["discovery_type"])
	d.Set("retries", data["retries"])
	d.Set("retry_timeout", data["retry_timeout"])

	// Set labels - always set to handle Computed: true
	labels := make(map[string]string)
	if labelsRaw, ok := data["labels"]; ok && labelsRaw != nil {
		if labelsMap, ok := labelsRaw.(map[string]interface{}); ok {
			for k, v := range labelsMap {
				if str, ok := v.(string); ok {
					labels[k] = str
				}
			}
		}
	}
	d.Set("labels", labels)

	if nodes, ok := data["nodes"].([]interface{}); ok {
		d.Set("nodes", flattenNodes(nodes))
	}

	if timeout, ok := data["timeout"].(map[string]interface{}); ok {
		d.Set("timeout", []interface{}{timeout})
	}

	if healthCheck, ok := data["checks"]; ok {
		if jsonBytes, err := json.Marshal(healthCheck); err == nil {
			d.Set("health_check", string(jsonBytes))
		}
	}

	if keepalivePool, ok := data["keepalive_pool"]; ok {
		d.Set("keepalive_pool", []interface{}{keepalivePool})
	}

	if tls, ok := data["tls"]; ok {
		d.Set("tls", []interface{}{tls})
	}

	return nil
}

// flattenNodes converts APISIX nodes to Terraform format
func flattenNodes(nodes []interface{}) []interface{} {
	result := make([]interface{}, 0, len(nodes))
	for _, node := range nodes {
		if n, ok := node.(map[string]interface{}); ok {
			result = append(result, map[string]interface{}{
				"host":     n["host"],
				"port":     n["port"],
				"weight":   n["weight"],
				"priority": n["priority"],
				"metadata": n["metadata"],
			})
		}
	}
	return result
}
