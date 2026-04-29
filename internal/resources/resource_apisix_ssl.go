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

func ResourceApisixSSL() *schema.Resource {
	return &schema.Resource{
		Description: "Manages an APISIX SSL/TLS certificate resource.",

		CreateContext: resourceApisixSSLCreate,
		ReadContext:   resourceApisixSSLRead,
		UpdateContext: resourceApisixSSLUpdate,
		DeleteContext: resourceApisixSSLDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"sni": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Server Name Indication (SNI) - the domain name for the SSL certificate.",
			},
			"snis": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of SNI names. Conflicts with `sni`.",
			},
			"cert": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "SSL certificate content (PEM format). Required unless using `certs`/`keys` for SNI.",
			},
			"key": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "SSL private key content (PEM format). Required unless using `certs`/`keys` for SNI.",
			},
			"certs": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of SSL certificates for SNI. Use with `keys`.",
			},
			"keys": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of SSL private keys for SNI. Use with `certs`.",
			},
			"ssl_protocols": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "SSL/TLS protocol versions to enable. Valid values: TLSv1, TLSv1.1, TLSv1.2, TLSv1.3.",
			},
			"client": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Client certificate verification configuration.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ca_cert": {
							Type:        schema.TypeString,
							Optional:    true,
							Sensitive:   true,
							Description: "CA certificate for client certificate verification.",
						},
						"depth": {
							Type:        schema.TypeInt,
							Optional:    true,
							Default:     1,
							Description: "Maximum depth of CA certificates in the client certificate chain. Defaults to 1.",
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
		},
	}
}

func resourceApisixSSLCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	ssl := expandSSL(d)
	id := d.Get("sni").(string)
	if id == "" && len(d.Get("snis").([]interface{})) > 0 {
		id = d.Get("snis").([]interface{})[0].(string)
	}
	if id == "" {
		id = fmt.Sprintf("ssl-%d", len(d.Get("cert").(string)))
	}

	err = client.Create(ctx, "ssl", id, ssl)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create SSL certificate: %w", err))
	}

	d.SetId(id)
	return resourceApisixSSLRead(ctx, d, meta)
}

func resourceApisixSSLRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	data, err := client.Read(ctx, "ssl", d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("failed to read SSL certificate: %w", err))
	}

	var resp apisix.APISIXResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	return flattenSSL(d, resp.Value)
}

func resourceApisixSSLUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	ssl := expandSSL(d)
	err = client.Update(ctx, "ssl", d.Id(), ssl)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update SSL certificate: %w", err))
	}

	return resourceApisixSSLRead(ctx, d, meta)
}

func resourceApisixSSLDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := getClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	err = client.Delete(ctx, "ssl", d.Id(), false)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete SSL certificate: %w", err))
	}

	d.SetId("")
	return nil
}

// expandSSL converts Terraform resource data to APISIX SSL map
func expandSSL(d *schema.ResourceData) map[string]interface{} {
	ssl := make(map[string]interface{})

	if v, ok := d.GetOk("sni"); ok {
		ssl["sni"] = v.(string)
	}
	if v, ok := d.GetOk("snis"); ok {
		ssl["snis"] = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("cert"); ok {
		ssl["cert"] = v.(string)
	}
	if v, ok := d.GetOk("key"); ok {
		ssl["key"] = v.(string)
	}
	if v, ok := d.GetOk("certs"); ok {
		ssl["certs"] = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("keys"); ok {
		ssl["keys"] = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("ssl_protocols"); ok {
		ssl["ssl_protocols"] = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("client"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		client := v.([]interface{})[0].(map[string]interface{})
		clientConfig := make(map[string]interface{})
		if caCert, ok := client["ca_cert"]; ok && caCert != "" {
			clientConfig["ca_cert"] = caCert.(string)
		}
		if depth, ok := client["depth"]; ok {
			clientConfig["depth"] = depth.(int)
		}
		ssl["client"] = clientConfig
	}
	if v, ok := d.GetOk("labels"); ok {
		labels := make(map[string]string)
		for k, val := range v.(map[string]interface{}) {
			labels[k] = val.(string)
		}
		ssl["labels"] = labels
	}

	return ssl
}

// flattenSSL sets Terraform state from APISIX SSL response
func flattenSSL(d *schema.ResourceData, value interface{}) diag.Diagnostics {
	data, ok := value.(map[string]interface{})
	if !ok {
		return diag.Errorf("failed to convert SSL data")
	}

	d.Set("sni", data["sni"])
	d.Set("snis", data["snis"])
	// Don't set cert and key as they are sensitive and returned masked by API
	d.Set("certs", data["certs"])
	d.Set("keys", data["keys"])
	d.Set("ssl_protocols", data["ssl_protocols"])

	if client, ok := data["client"].(map[string]interface{}); ok {
		clientList := make([]interface{}, 1)
		clientMap := make(map[string]interface{})
		if caCert, ok := client["ca_cert"]; ok {
			clientMap["ca_cert"] = caCert.(string)
		}
		if depth, ok := client["depth"]; ok {
			clientMap["depth"] = depth.(int)
		}
		clientList[0] = clientMap
		d.Set("client", clientList)
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
