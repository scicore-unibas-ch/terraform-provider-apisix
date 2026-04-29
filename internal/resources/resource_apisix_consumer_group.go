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

func ResourceApisixConsumerGroup() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an APISIX Consumer Group resource.",

		CreateContext: resourceApisixConsumerGroupCreate,
		ReadContext:   resourceApisixConsumerGroupRead,
		UpdateContext: resourceApisixConsumerGroupUpdate,
		DeleteContext: resourceApisixConsumerGroupDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"group_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the consumer group. This is the unique identifier. Changing this forces a new resource to be created.",
			},
			"desc": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the consumer group.",
			},
			"plugins": {
				Type:        schema.TypeMap,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Plugin configurations as JSON-encoded strings. Plugins applied to all consumers in this group.",
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

func resourceApisixConsumerGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	group := expandConsumerGroup(d)
	groupID := d.Get("group_id").(string)

	err = client.Create(ctx, "consumer_groups", groupID, group)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create consumer group: %w", err))
	}

	d.SetId(groupID)
	return resourceApisixConsumerGroupRead(ctx, d, meta)
}

func resourceApisixConsumerGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data, err := client.Read(ctx, "consumer_groups", d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read consumer group: %w", err))
	}

	var resp apisix.APISIXResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	return flattenConsumerGroup(d, resp.Value)
}

func resourceApisixConsumerGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	group := expandConsumerGroup(d)
	err = client.Update(ctx, "consumer_groups", d.Id(), group)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update consumer group: %w", err))
	}

	return resourceApisixConsumerGroupRead(ctx, d, meta)
}

func resourceApisixConsumerGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Delete(ctx, "consumer_groups", d.Id(), false)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete consumer group: %w", err))
	}

	d.SetId("")
	return nil
}

// expandConsumerGroup converts Terraform resource data to APISIX consumer group map
func expandConsumerGroup(d *schema.ResourceData) map[string]interface{} {
	group := make(map[string]interface{})

	// Group ID is required in the request body
	group["id"] = d.Get("group_id").(string)

	if v, ok := d.GetOk("desc"); ok {
		group["desc"] = v.(string)
	}
	if v, ok := d.GetOk("plugins"); ok {
		plugins := make(map[string]interface{})
		for k, v := range v.(map[string]interface{}) {
			var pluginConfig interface{}
			if err := json.Unmarshal([]byte(v.(string)), &pluginConfig); err == nil {
				plugins[k] = pluginConfig
			}
		}
		group["plugins"] = plugins
	}
	if v, ok := d.GetOk("labels"); ok {
		labels := make(map[string]string)
		for k, val := range v.(map[string]interface{}) {
			labels[k] = val.(string)
		}
		group["labels"] = labels
	}

	return group
}

// flattenConsumerGroup sets Terraform state from APISIX consumer group response
func flattenConsumerGroup(d *schema.ResourceData, value interface{}) diag.Diagnostics {
	data, ok := value.(map[string]interface{})
	if !ok {
		return diag.Errorf("failed to convert consumer group data")
	}

	d.Set("group_id", data["id"])
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
