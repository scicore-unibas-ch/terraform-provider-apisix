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

func ResourceApisixService() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an APISIX Service resource.",

		CreateContext: resourceApisixServiceCreate,
		ReadContext:   resourceApisixServiceRead,
		UpdateContext: resourceApisixServiceUpdate,
		DeleteContext: resourceApisixServiceDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the service.",
			},
			"desc": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the service.",
			},
			"hosts": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of hosts to match.",
			},
			"plugins": {
				Type:        schema.TypeMap,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Plugin configurations as JSON-encoded strings.",
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
			"labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Labels as key-value pairs.",
			},
			"enable_websocket": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable websocket support. Defaults to false.",
			},
		},
	}
}

func resourceApisixServiceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	service := expandService(d)
	id := d.Get("name").(string)
	if id == "" {
		id = fmt.Sprintf("service-%d", strings.ReplaceAll(fmt.Sprintf("%d", time.Now().UnixNano()), "-", ""))
	}

	err = client.Create(ctx, "services", id, service)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create service: %w", err))
	}

	d.SetId(id)
	return resourceApisixServiceRead(ctx, d, meta)
}

func resourceApisixServiceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data, err := client.Read(ctx, "services", d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read service: %w", err))
	}

	var resp apisix.APISIXResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	return flattenService(d, resp.Value)
}

func resourceApisixServiceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	service := expandService(d)
	err = client.Update(ctx, "services", d.Id(), service)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update service: %w", err))
	}

	return resourceApisixServiceRead(ctx, d, meta)
}

func resourceApisixServiceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Delete(ctx, "services", d.Id(), false)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete service: %w", err))
	}

	d.SetId("")
	return nil
}

// expandService converts Terraform resource data to APISIX service map
func expandService(d *schema.ResourceData) map[string]interface{} {
	service := make(map[string]interface{})

	if v, ok := d.GetOk("name"); ok {
		service["name"] = v.(string)
	}
	if v, ok := d.GetOk("desc"); ok {
		service["desc"] = v.(string)
	}
	if v, ok := d.GetOk("hosts"); ok {
		service["hosts"] = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("plugins"); ok {
		plugins := make(map[string]interface{})
		for k, v := range v.(map[string]interface{}) {
			var pluginConfig interface{}
			if err := json.Unmarshal([]byte(v.(string)), &pluginConfig); err == nil {
				plugins[k] = pluginConfig
			}
		}
		service["plugins"] = plugins
	}
	if v, ok := d.GetOk("upstream_id"); ok {
		service["upstream_id"] = v.(string)
	}
	if v, ok := d.GetOk("upstream"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		service["upstream"] = expandInlineUpstream(v.([]interface{}))
	}
	if v, ok := d.GetOk("labels"); ok {
		labels := make(map[string]string)
		for k, val := range v.(map[string]interface{}) {
			labels[k] = val.(string)
		}
		service["labels"] = labels
	}
	if v, ok := d.GetOk("enable_websocket"); ok {
		service["enable_websocket"] = v.(bool)
	}

	return service
}

// flattenService sets Terraform state from APISIX service response
func flattenService(d *schema.ResourceData, value interface{}) diag.Diagnostics {
	data, ok := value.(map[string]interface{})
	if !ok {
		return diag.Errorf("failed to convert service data")
	}

	d.Set("name", data["name"])
	d.Set("desc", data["desc"])
	d.Set("hosts", data["hosts"])
	d.Set("upstream_id", data["upstream_id"])
	d.Set("enable_websocket", data["enable_websocket"])

	if plugins, ok := data["plugins"].(map[string]interface{}); ok {
		pluginMap := make(map[string]string)
		for k, v := range plugins {
			if jsonBytes, err := json.Marshal(v); err == nil {
				pluginMap[k] = string(jsonBytes)
			}
		}
		d.Set("plugins", pluginMap)
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

	return nil
}

// Helper functions (reuse from Route resource)
