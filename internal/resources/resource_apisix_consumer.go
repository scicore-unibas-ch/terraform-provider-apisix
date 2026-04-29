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

func ResourceApisixConsumer() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an APISIX Consumer resource.",

		CreateContext: resourceApisixConsumerCreate,
		ReadContext:   resourceApisixConsumerRead,
		UpdateContext: resourceApisixConsumerUpdate,
		DeleteContext: resourceApisixConsumerDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Username of the consumer. This is the unique identifier.",
			},
			"group_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Group ID of the consumer for consumer grouping.",
			},
			"desc": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the consumer.",
			},
			"plugins": {
				Type:        schema.TypeMap,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Plugin configurations as JSON-encoded strings. Common plugins: key-auth, jwt-auth, hmac-auth, basic-auth.",
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

func resourceApisixConsumerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	consumer := expandConsumer(d)
	username := d.Get("username").(string)

	err = client.Create(ctx, "consumers", username, consumer)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create consumer: %w", err))
	}

	d.SetId(username)
	return resourceApisixConsumerRead(ctx, d, meta)
}

func resourceApisixConsumerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data, err := client.Read(ctx, "consumers", d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read consumer: %w", err))
	}

	var resp apisix.APISIXResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	return flattenConsumer(d, resp.Value)
}

func resourceApisixConsumerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	consumer := expandConsumer(d)
	err = client.Update(ctx, "consumers", d.Id(), consumer)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update consumer: %w", err))
	}

	return resourceApisixConsumerRead(ctx, d, meta)
}

func resourceApisixConsumerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Delete(ctx, "consumers", d.Id(), false)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete consumer: %w", err))
	}

	d.SetId("")
	return nil
}

// expandConsumer converts Terraform resource data to APISIX consumer map
func expandConsumer(d *schema.ResourceData) map[string]interface{} {
	consumer := make(map[string]interface{})

	// Username is required in the request body
	consumer["username"] = d.Get("username").(string)

	if v, ok := d.GetOk("group_id"); ok {
		consumer["group_id"] = v.(string)
	}
	if v, ok := d.GetOk("desc"); ok {
		consumer["desc"] = v.(string)
	}
	if v, ok := d.GetOk("plugins"); ok {
		plugins := make(map[string]interface{})
		for k, v := range v.(map[string]interface{}) {
			var pluginConfig interface{}
			if err := json.Unmarshal([]byte(v.(string)), &pluginConfig); err == nil {
				plugins[k] = pluginConfig
			}
		}
		consumer["plugins"] = plugins
	}
	if v, ok := d.GetOk("labels"); ok {
		labels := make(map[string]string)
		for k, val := range v.(map[string]interface{}) {
			labels[k] = val.(string)
		}
		consumer["labels"] = labels
	}

	return consumer
}

// flattenConsumer sets Terraform state from APISIX consumer response
func flattenConsumer(d *schema.ResourceData, value interface{}) diag.Diagnostics {
	data, ok := value.(map[string]interface{})
	if !ok {
		return diag.Errorf("failed to convert consumer data")
	}

	d.Set("username", data["username"])
	d.Set("group_id", data["group_id"])
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
