package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestProvider(t *testing.T) {
	p := Provider()
	if p == nil {
		t.Fatal("Provider should not be nil")
	}

	if p.Schema == nil {
		t.Fatal("Provider schema should not be nil")
	}

	if p.ResourcesMap == nil {
		t.Fatal("Provider ResourcesMap should not be nil")
	}

	if p.ConfigureContextFunc == nil {
		t.Fatal("Provider ConfigureContextFunc should not be nil")
	}
}

func TestProviderSchema(t *testing.T) {
	p := Provider()

	// Check required fields
	requiredFields := []string{"base_url", "admin_key"}
	for _, field := range requiredFields {
		schema, ok := p.Schema[field]
		if !ok {
			t.Errorf("Schema missing required field: %s", field)
		}
		if !schema.Required {
			t.Errorf("Field %s should be Required", field)
		}
	}

	// Check optional fields
	optionalFields := []string{"timeout"}
	for _, field := range optionalFields {
		schema, ok := p.Schema[field]
		if !ok {
			t.Errorf("Schema missing optional field: %s", field)
		}
		if !schema.Optional {
			t.Errorf("Field %s should be Optional", field)
		}
	}

	// Check sensitive field
	adminKeySchema := p.Schema["admin_key"]
	if !adminKeySchema.Sensitive {
		t.Error("admin_key should be Sensitive")
	}
}

func TestProviderConfigure(t *testing.T) {
	// Set environment variables for testing
	os.Setenv("APISIX_BASE_URL", "http://localhost:9180/apisix/admin")
	os.Setenv("APISIX_ADMIN_KEY", "test123")
	defer os.Unsetenv("APISIX_BASE_URL")
	defer os.Unsetenv("APISIX_ADMIN_KEY")

	p := Provider()
	diag := p.InternalValidate()
	if diag != nil {
		t.Fatalf("Provider validation failed: %v", diag)
	}
}

func TestProviderConfigureContext(t *testing.T) {
	p := Provider()

	// Create test resource data
	d := schema.TestResourceDataRaw(t, p.Schema, map[string]interface{}{
		"base_url":  "http://localhost:9180/apisix/admin",
		"admin_key": "test123",
		"timeout":   30,
	})

	// Call configure function
	client, diags := providerConfigure(context.Background(), d)
	if diags.HasError() {
		t.Fatalf("Provider configuration failed: %v", diags)
	}

	if client == nil {
		t.Fatal("Provider configuration returned nil client")
	}
}

func TestProviderConfigure_MissingBaseUrl(t *testing.T) {
	p := Provider()

	d := schema.TestResourceDataRaw(t, p.Schema, map[string]interface{}{
		"admin_key": "test123",
	})

	_, diags := providerConfigure(context.Background(), d)
	if !diags.HasError() {
		t.Fatal("Expected error for missing base_url")
	}
}

func TestProviderConfigure_MissingAdminKey(t *testing.T) {
	p := Provider()

	d := schema.TestResourceDataRaw(t, p.Schema, map[string]interface{}{
		"base_url": "http://localhost:9180/apisix/admin",
	})

	_, diags := providerConfigure(context.Background(), d)
	if !diags.HasError() {
		t.Fatal("Expected error for missing admin_key")
	}
}

func TestGetClient(t *testing.T) {
	// Test with nil meta
	client, err := GetClient(nil)
	if err != nil {
		t.Errorf("GetClient(nil) should not return error: %v", err)
	}
	if client != nil {
		t.Errorf("GetClient(nil) should return nil client")
	}
}
