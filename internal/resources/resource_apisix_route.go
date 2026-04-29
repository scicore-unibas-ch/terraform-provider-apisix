package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/apisix"
)

func ResourceApisixRoute() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an APISIX Route resource.",

		CreateContext: resourceApisixRouteCreate,
		ReadContext:   resourceApisixRouteRead,
		UpdateContext: resourceApisixRouteUpdate,
		DeleteContext: resourceApisixRouteDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the route.",
			},
			"desc": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the route.",
			},
			"uri": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"uris"},
				Description:   "Request URI prefix. Conflicts with `uris`.",
			},
			"uris": {
				Type:          schema.TypeList,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"uri"},
				Description:   "Request URI prefixes as a list. Conflicts with `uri`.",
			},
			"host": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"hosts"},
				Description:   "Request host. Conflicts with `hosts`.",
			},
			"hosts": {
				Type:          schema.TypeList,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"host"},
				Description:   "Request hosts as a list. Conflicts with `host`.",
			},
			"remote_addr": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"remote_addrs"},
				Description:   "Client IP. Conflicts with `remote_addrs`.",
			},
			"remote_addrs": {
				Type:          schema.TypeList,
				Optional:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				ConflictsWith: []string{"remote_addr"},
				Description:   "Client IPs as a list. Conflicts with `remote_addr`.",
			},
			"methods": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "HTTP methods. Valid values: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, TRACE, CONNECT, PURGE.",
			},
			"priority": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Priority of the route. Higher value means higher priority. Defaults to 0.",
			},
			"vars": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "JSON-encoded filter expressions for advanced routing.",
			},
			"filter_func": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Lua function for custom filtering.",
			},
			"plugins": {
				Type:        schema.TypeMap,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Plugin configurations as JSON-encoded strings.",
			},
			"script": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"plugins"},
				Description:   "Lua script for custom logic. Conflicts with `plugins`.",
			},
			"upstream_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"upstream"},
				Description:   "ID of the upstream resource. Conflicts with `upstream`.",
			},
			"upstream": {
				Type:          schema.TypeList,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"upstream_id"},
				Description:   "Inline upstream configuration. Conflicts with `upstream_id`.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Optional:    true,
							Default:     "roundrobin",
							Description: "Load balancing type.",
						},
						"nodes": {
							Type:        schema.TypeList,
							Required:    true,
							Description: "Upstream nodes.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"host": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Node host.",
									},
									"port": {
										Type:        schema.TypeInt,
										Required:    true,
										Description: "Node port.",
									},
									"weight": {
										Type:        schema.TypeInt,
										Optional:    true,
										Default:     1,
										Description: "Node weight.",
									},
								},
							},
						},
					},
				},
			},
			"service_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "ID of the service resource.",
			},
			"plugin_config_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "ID of the plugin configuration.",
			},
			"labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Labels as key-value pairs.",
			},
			"timeout": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Timeout configuration.",
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
			"enable_websocket": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable websocket support. Defaults to false.",
			},
			"status": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1,
				Description: "Route status. 1=enabled, 0=disabled. Defaults to 1.",
			},
		},
	}
}

func resourceApisixRouteCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	route := expandRoute(d)
	id := d.Get("name").(string)
	if id == "" {
		id = fmt.Sprintf("route-%d", time.Now().UnixNano())
	}

	err = client.Create(ctx, "routes", id, route)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create route: %w", err))
	}

	d.SetId(id)
	return resourceApisixRouteRead(ctx, d, meta)
}

func resourceApisixRouteRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data, err := client.Read(ctx, "routes", d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read route: %w", err))
	}

	var resp apisix.APISIXResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	return flattenRoute(d, resp.Value)
}

func resourceApisixRouteUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	route := expandRoute(d)
	err = client.Update(ctx, "routes", d.Id(), route)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update route: %w", err))
	}

	return resourceApisixRouteRead(ctx, d, meta)
}

func resourceApisixRouteDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Delete(ctx, "routes", d.Id(), false)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete route: %w", err))
	}

	d.SetId("")
	return nil
}

// expandRoute converts Terraform resource data to APISIX route map
func expandRoute(d *schema.ResourceData) map[string]interface{} {
	route := make(map[string]interface{})

	if v, ok := d.GetOk("name"); ok {
		route["name"] = v.(string)
	}
	if v, ok := d.GetOk("desc"); ok {
		route["desc"] = v.(string)
	}
	if v, ok := d.GetOk("uri"); ok {
		route["uri"] = v.(string)
	}
	if v, ok := d.GetOk("uris"); ok {
		route["uris"] = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("host"); ok {
		route["host"] = v.(string)
	}
	if v, ok := d.GetOk("hosts"); ok {
		route["hosts"] = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("remote_addr"); ok {
		route["remote_addr"] = v.(string)
	}
	if v, ok := d.GetOk("remote_addrs"); ok {
		route["remote_addrs"] = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("methods"); ok {
		route["methods"] = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("priority"); ok {
		route["priority"] = v.(int)
	}
	if v, ok := d.GetOk("vars"); ok {
		var vars interface{}
		if err := json.Unmarshal([]byte(v.(string)), &vars); err == nil {
			route["vars"] = vars
		}
	}
	if v, ok := d.GetOk("filter_func"); ok {
		route["filter_func"] = v.(string)
	}
	if v, ok := d.GetOk("plugins"); ok {
		plugins := make(map[string]interface{})
		for k, v := range v.(map[string]interface{}) {
			var pluginConfig interface{}
			if err := json.Unmarshal([]byte(v.(string)), &pluginConfig); err == nil {
				plugins[k] = pluginConfig
			}
		}
		route["plugins"] = plugins
	}
	if v, ok := d.GetOk("script"); ok {
		route["script"] = v.(string)
	}
	if v, ok := d.GetOk("upstream_id"); ok {
		route["upstream_id"] = v.(string)
	}
	if v, ok := d.GetOk("upstream"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		route["upstream"] = expandInlineUpstream(v.([]interface{}))
	}
	if v, ok := d.GetOk("service_id"); ok {
		route["service_id"] = v.(string)
	}
	if v, ok := d.GetOk("plugin_config_id"); ok {
		route["plugin_config_id"] = v.(string)
	}
	if v, ok := d.GetOk("labels"); ok {
		labels := make(map[string]string)
		for k, val := range v.(map[string]interface{}) {
			labels[k] = val.(string)
		}
		route["labels"] = labels
	}
	if v, ok := d.GetOk("timeout"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		route["timeout"] = expandRouteTimeout(v.([]interface{}))
	}
	if v, ok := d.GetOk("enable_websocket"); ok {
		route["enable_websocket"] = v.(bool)
	}
	if v, ok := d.GetOk("status"); ok {
		route["status"] = v.(int)
	}

	return route
}




func expandRouteTimeout(timeout []interface{}) map[string]interface{} {
	if len(timeout) == 0 || timeout[0] == nil {
		return nil
	}
	t := timeout[0].(map[string]interface{})
	result := make(map[string]interface{})
	if v, ok := t["connect"].(int); ok {
		result["connect"] = v
	}
	if v, ok := t["send"].(int); ok {
		result["send"] = v
	}
	if v, ok := t["read"].(int); ok {
		result["read"] = v
	}
	return result
}

// flattenRoute sets Terraform state from APISIX route response
func flattenRoute(d *schema.ResourceData, value interface{}) diag.Diagnostics {
	data, ok := value.(map[string]interface{})
	if !ok {
		return diag.Errorf("failed to convert route data")
	}

	d.Set("name", data["name"])
	d.Set("desc", data["desc"])
	d.Set("uri", data["uri"])
	d.Set("uris", data["uris"])
	d.Set("host", data["host"])
	d.Set("hosts", data["hosts"])
	d.Set("remote_addr", data["remote_addr"])
	d.Set("remote_addrs", data["remote_addrs"])
	d.Set("methods", data["methods"])
	d.Set("priority", data["priority"])
	d.Set("filter_func", data["filter_func"])
	d.Set("upstream_id", data["upstream_id"])
	d.Set("service_id", data["service_id"])
	d.Set("plugin_config_id", data["plugin_config_id"])
	d.Set("enable_websocket", data["enable_websocket"])
	d.Set("status", data["status"])

	if vars, ok := data["vars"]; ok {
		if jsonBytes, err := json.Marshal(vars); err == nil {
			d.Set("vars", string(jsonBytes))
		}
	}

	if plugins, ok := data["plugins"].(map[string]interface{}); ok {
		pluginMap := make(map[string]string)
		for k, v := range plugins {
			if jsonBytes, err := json.Marshal(v); err == nil {
				pluginMap[k] = string(jsonBytes)
			}
		}
		d.Set("plugins", pluginMap)
	}

	if script, ok := data["script"]; ok {
		d.Set("script", script)
	}

	if upstream, ok := data["upstream"]; ok {
		d.Set("upstream", []interface{}{upstream})
	}

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

	if timeout, ok := data["timeout"]; ok {
		d.Set("timeout", []interface{}{timeout})
	}

	return nil
}
