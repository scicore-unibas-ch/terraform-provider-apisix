package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceApisixService(t *testing.T) {
	resource := ResourceApisixService()
	assert.NotNil(t, resource)
	assert.NotNil(t, resource.CreateContext)
	assert.NotNil(t, resource.ReadContext)
	assert.NotNil(t, resource.UpdateContext)
	assert.NotNil(t, resource.DeleteContext)
	assert.NotNil(t, resource.Importer)
}

func TestResourceApisixServiceSchema(t *testing.T) {
	resource := ResourceApisixService()
	schema := resource.Schema

	assert.NotNil(t, schema["name"])
	assert.NotNil(t, schema["desc"])
	assert.NotNil(t, schema["hosts"])
	assert.NotNil(t, schema["plugins"])
	assert.NotNil(t, schema["upstream_id"])
	assert.NotNil(t, schema["upstream"])
	assert.NotNil(t, schema["labels"])
	assert.NotNil(t, schema["enable_websocket"])
}

func TestExpandService(t *testing.T) {
	resource := ResourceApisixService()
	d := resource.TestResourceData()

	d.Set("name", "test-service")
	d.Set("desc", "Test service")

	service := expandService(d)

	assert.NotNil(t, service)
	assert.Equal(t, "test-service", service["name"])
}

func TestExpandServiceWithLabels(t *testing.T) {
	resource := ResourceApisixService()
	d := resource.TestResourceData()

	d.Set("name", "labeled-service")
	d.Set("labels", map[string]interface{}{
		"env":  "production",
		"team": "platform",
	})

	service := expandService(d)

	assert.NotNil(t, service)
	assert.NotNil(t, service["labels"])
}

func TestExpandServiceWithPlugins(t *testing.T) {
	resource := ResourceApisixService()
	d := resource.TestResourceData()

	d.Set("name", "plugin-service")
	d.Set("plugins", map[string]interface{}{
		"limit-count": `{"count":100,"time_window":60}`,
	})

	service := expandService(d)

	assert.NotNil(t, service)
	assert.NotNil(t, service["plugins"])
}

func TestExpandServiceWithScript(t *testing.T) {
	resource := ResourceApisixService()
	d := resource.TestResourceData()

	d.Set("name", "script-service")
	d.Set("script", "local _M = {}\nreturn _M")

	service := expandService(d)

	assert.NotNil(t, service)
	assert.NotNil(t, service["script"])
}

func TestFlattenService(t *testing.T) {
	resource := ResourceApisixService()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"name": "test-service",
		"desc": "Test service",
	}

	diags := flattenService(d, value)

	assert.Len(t, diags, 0)
	assert.Equal(t, "test-service", d.Get("name"))
}

func TestFlattenServiceWithLabels(t *testing.T) {
	resource := ResourceApisixService()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"name": "labeled-service",
		"labels": map[string]interface{}{
			"env":  "production",
			"team": "platform",
		},
	}

	diags := flattenService(d, value)

	assert.Len(t, diags, 0)
	labels := d.Get("labels").(map[string]interface{})
	assert.Equal(t, "production", labels["env"])
}

func TestFlattenServiceWithNilValue(t *testing.T) {
	resource := ResourceApisixService()
	d := resource.TestResourceData()

	var value interface{} = nil

	diags := flattenService(d, value)

	assert.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "failed to convert service data")
}

func TestFlattenServiceInvalidType(t *testing.T) {
	resource := ResourceApisixService()
	d := resource.TestResourceData()

	value := "invalid-type"

	diags := flattenService(d, value)

	assert.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "failed to convert service data")
}
