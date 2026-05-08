// Package provider is the Plugin Framework provider entrypoint.
package provider

import (
	"context"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/scicore-unibas-ch/terraform-provider-apisix/internal/client"
	consumerres "github.com/scicore-unibas-ch/terraform-provider-apisix/internal/resource/consumer"
	consumergroupres "github.com/scicore-unibas-ch/terraform-provider-apisix/internal/resource/consumergroup"
	globalruleres "github.com/scicore-unibas-ch/terraform-provider-apisix/internal/resource/globalrule"
	pluginconfigres "github.com/scicore-unibas-ch/terraform-provider-apisix/internal/resource/pluginconfig"
	routeres "github.com/scicore-unibas-ch/terraform-provider-apisix/internal/resource/route"
	serviceres "github.com/scicore-unibas-ch/terraform-provider-apisix/internal/resource/service"
	upstreamres "github.com/scicore-unibas-ch/terraform-provider-apisix/internal/resource/upstream"
)

var (
	_ provider.Provider = (*apisixProvider)(nil)
)

type apisixProvider struct {
	version string
}

type providerModel struct {
	BaseURL  types.String `tfsdk:"base_url"`
	AdminKey types.String `tfsdk:"admin_key"`
	Timeout  types.Int64  `tfsdk:"timeout"`
	Insecure types.Bool   `tfsdk:"insecure"`
}

// New returns a Plugin Framework provider factory.
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &apisixProvider{version: version} }
}

func (p *apisixProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "apisix"
	resp.Version = p.version
}

func (p *apisixProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provider for the Apache APISIX Admin API.",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the APISIX Admin API (e.g. http://localhost:9180/apisix/admin). Falls back to APISIX_BASE_URL.",
			},
			"admin_key": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Admin API key. Falls back to APISIX_ADMIN_KEY.",
			},
			"timeout": schema.Int64Attribute{
				Optional:    true,
				Description: "HTTP client timeout in seconds. Default: 30.",
			},
			"insecure": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS certificate verification. Default: false.",
			},
		},
	}
}

func (p *apisixProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := cfg.BaseURL.ValueString()
	if baseURL == "" {
		baseURL = os.Getenv("APISIX_BASE_URL")
	}
	adminKey := cfg.AdminKey.ValueString()
	if adminKey == "" {
		adminKey = os.Getenv("APISIX_ADMIN_KEY")
	}

	if baseURL == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_url"),
			"Missing base_url",
			"Set the base_url attribute or the APISIX_BASE_URL environment variable.",
		)
	}
	if adminKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("admin_key"),
			"Missing admin_key",
			"Set the admin_key attribute or the APISIX_ADMIN_KEY environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	timeoutSec := int64(30)
	if !cfg.Timeout.IsNull() && !cfg.Timeout.IsUnknown() {
		timeoutSec = cfg.Timeout.ValueInt64()
	}

	c := client.New(client.Config{
		BaseURL:  baseURL,
		AdminKey: adminKey,
		Timeout:  time.Duration(timeoutSec) * time.Second,
		Insecure: cfg.Insecure.ValueBool(),
	})
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *apisixProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		consumerres.NewResource,
		consumergroupres.NewResource,
		globalruleres.NewResource,
		pluginconfigres.NewResource,
		routeres.NewResource,
		serviceres.NewResource,
		upstreamres.NewResource,
	}
}

func (p *apisixProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
