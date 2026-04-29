package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceApisixSSL(t *testing.T) {
	resource := ResourceApisixSSL()
	assert.NotNil(t, resource)
	assert.NotNil(t, resource.CreateContext)
	assert.NotNil(t, resource.ReadContext)
	assert.NotNil(t, resource.UpdateContext)
	assert.NotNil(t, resource.DeleteContext)
	assert.NotNil(t, resource.Importer)
}

func TestResourceApisixSSLSchema(t *testing.T) {
	resource := ResourceApisixSSL()
	schema := resource.Schema

	assert.NotNil(t, schema["sni"])
	assert.NotNil(t, schema["snis"])
	assert.NotNil(t, schema["cert"])
	assert.NotNil(t, schema["key"])
	assert.NotNil(t, schema["certs"])
	assert.NotNil(t, schema["keys"])
	assert.NotNil(t, schema["ssl_protocols"])
	assert.NotNil(t, schema["client"])
	assert.NotNil(t, schema["labels"])
}

func TestExpandSSL(t *testing.T) {
	resource := ResourceApisixSSL()
	d := resource.TestResourceData()

	d.Set("sni", "example.com")
	d.Set("cert", "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----")
	d.Set("key", "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----")

	ssl := expandSSL(d)

	assert.NotNil(t, ssl)
	assert.Equal(t, "example.com", ssl["sni"])
	assert.NotNil(t, ssl["cert"])
	assert.NotNil(t, ssl["key"])
}

func TestExpandSSLWithMultipleSNIs(t *testing.T) {
	resource := ResourceApisixSSL()
	d := resource.TestResourceData()

	d.Set("snis", []interface{}{"api.example.com", "www.example.com"})
	d.Set("cert", "test-cert")
	d.Set("key", "test-key")

	ssl := expandSSL(d)

	assert.NotNil(t, ssl)
	assert.NotNil(t, ssl["snis"])
}

func TestExpandSSLWithLabels(t *testing.T) {
	resource := ResourceApisixSSL()
	d := resource.TestResourceData()

	d.Set("sni", "labeled.example.com")
	d.Set("cert", "test-cert")
	d.Set("key", "test-key")
	d.Set("labels", map[string]interface{}{
		"env":  "production",
		"team": "platform",
	})

	ssl := expandSSL(d)

	assert.NotNil(t, ssl)
	assert.NotNil(t, ssl["labels"])
}

func TestExpandSSLWithClientBlock(t *testing.T) {
	resource := ResourceApisixSSL()
	d := resource.TestResourceData()

	d.Set("sni", "mtls.example.com")
	d.Set("cert", "test-cert")
	d.Set("key", "test-key")
	d.Set("client", []interface{}{
		map[string]interface{}{
			"ca_cert": "test-ca-cert",
			"depth":   2,
		},
	})

	ssl := expandSSL(d)

	assert.NotNil(t, ssl)
	assert.NotNil(t, ssl["client"])
}

func TestExpandSSLWithProtocols(t *testing.T) {
	resource := ResourceApisixSSL()
	d := resource.TestResourceData()

	d.Set("sni", "secure.example.com")
	d.Set("cert", "test-cert")
	d.Set("key", "test-key")
	d.Set("ssl_protocols", []interface{}{"TLSv1.2", "TLSv1.3"})

	ssl := expandSSL(d)

	assert.NotNil(t, ssl)
	assert.NotNil(t, ssl["ssl_protocols"])
}

func TestFlattenSSL(t *testing.T) {
	resource := ResourceApisixSSL()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"sni": "example.com",
	}

	diags := flattenSSL(d, value)

	assert.Len(t, diags, 0)
	assert.Equal(t, "example.com", d.Get("sni"))
}

func TestFlattenSSLWithLabels(t *testing.T) {
	resource := ResourceApisixSSL()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"sni": "labeled.example.com",
		"labels": map[string]interface{}{
			"env":  "production",
			"team": "platform",
		},
	}

	diags := flattenSSL(d, value)

	assert.Len(t, diags, 0)
	labels := d.Get("labels").(map[string]interface{})
	assert.Equal(t, "production", labels["env"])
}

func TestFlattenSSLWithNilValue(t *testing.T) {
	resource := ResourceApisixSSL()
	d := resource.TestResourceData()

	var value interface{} = nil

	diags := flattenSSL(d, value)

	assert.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "failed to convert SSL data")
}

func TestFlattenSSLInvalidType(t *testing.T) {
	resource := ResourceApisixSSL()
	d := resource.TestResourceData()

	value := "invalid-type"

	diags := flattenSSL(d, value)

	assert.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "failed to convert SSL data")
}
