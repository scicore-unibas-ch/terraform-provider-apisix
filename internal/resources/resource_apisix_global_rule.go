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

func ResourceApisixGlobalRule() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an APISIX Global Rule resource. Global rules are plugins that apply to ALL requests across all routes, useful for global rate limiting, logging, and security policies.",

		CreateContext: resourceApisixGlobalRuleCreate,
		ReadContext:   resourceApisixGlobalRuleRead,
		UpdateContext: resourceApisixGlobalRuleUpdate,
		DeleteContext: resourceApisixGlobalRuleDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"rule_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the global rule. This is the unique identifier. Changing this forces a new resource to be created.",
			},
			"plugins": {
				Type:        schema.TypeMap,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Plugin configurations as JSON-encoded strings. At least one plugin is required. These plugins apply to ALL routes.",
			},
		},
	}
}

func resourceApisixGlobalRuleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	globalRule := expandGlobalRule(d)
	ruleID := d.Get("rule_id").(string)

	err = client.Create(ctx, "global_rules", ruleID, globalRule)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create global rule: %w", err))
	}

	d.SetId(ruleID)
	return resourceApisixGlobalRuleRead(ctx, d, meta)
}

func resourceApisixGlobalRuleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data, err := client.Read(ctx, "global_rules", d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read global rule: %w", err))
	}

	var resp apisix.APISIXResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	return flattenGlobalRule(d, resp.Value)
}

func resourceApisixGlobalRuleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	globalRule := expandGlobalRule(d)
	err = client.Update(ctx, "global_rules", d.Id(), globalRule)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update global rule: %w", err))
	}

	return resourceApisixGlobalRuleRead(ctx, d, meta)
}

func resourceApisixGlobalRuleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Delete(ctx, "global_rules", d.Id(), false)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete global rule: %w", err))
	}

	d.SetId("")
	return nil
}

// expandGlobalRule converts Terraform resource data to APISIX global rule map
func expandGlobalRule(d *schema.ResourceData) map[string]interface{} {
	rule := make(map[string]interface{})

	// Rule ID is required in the request body
	rule["id"] = d.Get("rule_id").(string)

	if v, ok := d.GetOk("plugins"); ok {
		plugins := make(map[string]interface{})
		for k, v := range v.(map[string]interface{}) {
			var pluginConfig interface{}
			if err := json.Unmarshal([]byte(v.(string)), &pluginConfig); err == nil {
				plugins[k] = pluginConfig
			}
		}
		rule["plugins"] = plugins
	}

	return rule
}

// flattenGlobalRule sets Terraform state from APISIX global rule response
func flattenGlobalRule(d *schema.ResourceData, value interface{}) diag.Diagnostics {
	data, ok := value.(map[string]interface{})
	if !ok {
		return diag.Errorf("failed to convert global rule data")
	}

	d.Set("rule_id", data["id"])

	if plugins, ok := data["plugins"].(map[string]interface{}); ok {
		pluginMap := make(map[string]string)
		for k, v := range plugins {
			if jsonBytes, err := json.Marshal(v); err == nil {
				pluginMap[k] = string(jsonBytes)
			}
		}
		d.Set("plugins", pluginMap)
	}

	return nil
}
