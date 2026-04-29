package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/apisix"
)

func ResourceApisixPluginConfig() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an APISIX Plugin Config resource. Plugin configs allow you to create reusable plugin configurations that can be referenced by multiple routes.",

		CreateContext: resourceApisixPluginConfigCreate,
		ReadContext:   resourceApisixPluginConfigRead,
		UpdateContext: resourceApisixPluginConfigUpdate,
		DeleteContext: resourceApisixPluginConfigDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"config_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the plugin config. This is the unique identifier. Changing this forces a new resource to be created.",
			},
			"desc": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the plugin config.",
			},
			"plugins": {
				Type:        schema.TypeMap,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Plugin configurations as JSON-encoded strings. At least one plugin is required.",
			},
			"labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Labels as key-value pairs.",
			},
		},
	}
}

func resourceApisixPluginConfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	pluginConfig := expandPluginConfig(d)
	configID := d.Get("config_id").(string)

	err = client.Create(ctx, "plugin_configs", configID, pluginConfig)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create plugin config: %w", err))
	}

	d.SetId(configID)
	return resourceApisixPluginConfigRead(ctx, d, meta)
}

func resourceApisixPluginConfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data, err := client.Read(ctx, "plugin_configs", d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read plugin config: %w", err))
	}

	var resp apisix.APISIXResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	return flattenPluginConfig(d, resp.Value)
}

func resourceApisixPluginConfigUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	pluginConfig := expandPluginConfig(d)
	err = client.Update(ctx, "plugin_configs", d.Id(), pluginConfig)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update plugin config: %w", err))
	}

	return resourceApisixPluginConfigRead(ctx, d, meta)
}

func resourceApisixPluginConfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Delete(ctx, "plugin_configs", d.Id(), false)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete plugin config: %w", err))
	}

	d.SetId("")
	return nil
}

// expandPluginConfig converts Terraform resource data to APISIX plugin config map
func expandPluginConfig(d *schema.ResourceData) map[string]interface{} {
	config := make(map[string]interface{})

	// Config ID is required in the request body
	config["id"] = d.Get("config_id").(string)

	if v, ok := d.GetOk("desc"); ok {
		config["desc"] = v.(string)
	}
	if v, ok := d.GetOk("plugins"); ok {
		plugins := make(map[string]interface{})
		for k, v := range v.(map[string]interface{}) {
			var pluginConfig interface{}
			if err := json.Unmarshal([]byte(v.(string)), &pluginConfig); err == nil {
				plugins[k] = pluginConfig
			}
		}
		config["plugins"] = plugins
	}
	if v, ok := d.GetOk("labels"); ok {
		labels := make(map[string]string)
		for k, val := range v.(map[string]interface{}) {
			labels[k] = val.(string)
		}
		config["labels"] = labels
	}

	return config
}

// flattenPluginConfig sets Terraform state from APISIX plugin config response
func flattenPluginConfig(d *schema.ResourceData, value interface{}) diag.Diagnostics {
	data, ok := value.(map[string]interface{})
	if !ok {
		return diag.Errorf("failed to convert plugin config data")
	}

	d.Set("config_id", data["id"])
	d.Set("desc", data["desc"])

	if plugins, ok := data["plugins"].(map[string]interface{}); ok {
		pluginMap := make(map[string]string)
		for k, v := range plugins {
			if jsonBytes, err := json.Marshal(v); err == nil {
				pluginMap[k] = string(jsonBytes)
			}
		}
		d.Set("plugins", pluginMap)
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
