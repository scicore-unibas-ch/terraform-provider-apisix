package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/apisix"
	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/resources"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"base_url": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("APISIX_BASE_URL", nil),
				Description: "The base URL of the APISIX Admin API (e.g., http://localhost:9180/apisix/admin). Can be set via APISIX_BASE_URL environment variable.",
			},
			"admin_key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("APISIX_ADMIN_KEY", nil),
				Description: "The API key for authenticating with the APISIX Admin API. Can be set via APISIX_ADMIN_KEY environment variable.",
			},
			"timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     30,
				Description: "HTTP client timeout in seconds. Defaults to 30.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"apisix_upstream": resources.ResourceApisixUpstream(),
			"apisix_route":    resources.ResourceApisixRoute(),
			"apisix_service":  resources.ResourceApisixService(),
			"apisix_consumer": resources.ResourceApisixConsumer(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	baseURL := d.Get("base_url").(string)
	adminKey := d.Get("admin_key").(string)
	timeout := d.Get("timeout").(int)

	// Validate base_url
	if baseURL == "" {
		return nil, diag.Errorf("base_url is required")
	}

	// Validate admin_key
	if adminKey == "" {
		return nil, diag.Errorf("admin_key is required")
	}

	// Create APISIX client
	client := apisix.NewClient(baseURL, adminKey, timeout)

	return client, nil
}

// GetClient retrieves the APISIX client from the provider meta
func GetClient(meta interface{}) (*apisix.Client, error) {
	if meta == nil {
		return nil, nil
	}
	client, ok := meta.(*apisix.Client)
	if !ok {
		return nil, nil
	}
	return client, nil
}
