package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceApisixRoute(t *testing.T) {
	resource := ResourceApisixRoute()
	assert.NotNil(t, resource)
	assert.NotNil(t, resource.CreateContext)
	assert.NotNil(t, resource.ReadContext)
	assert.NotNil(t, resource.UpdateContext)
	assert.NotNil(t, resource.DeleteContext)
	assert.NotNil(t, resource.Importer)
}

func TestResourceApisixRouteSchema(t *testing.T) {
	resource := ResourceApisixRoute()
	schema := resource.Schema

	assert.NotNil(t, schema["name"])
	assert.NotNil(t, schema["uri"])
	assert.NotNil(t, schema["uris"])
	assert.NotNil(t, schema["host"])
	assert.NotNil(t, schema["hosts"])
	assert.NotNil(t, schema["plugins"])
	assert.NotNil(t, schema["upstream_id"])
	assert.NotNil(t, schema["upstream"])
	assert.NotNil(t, schema["labels"])
	assert.NotNil(t, schema["status"])
}

func TestExpandRoute(t *testing.T) {
	resource := ResourceApisixRoute()
	d := resource.TestResourceData()

	d.Set("name", "test-route")
	d.Set("uri", "/test/*")
	d.Set("status", 1)

	route := expandRoute(d)

	assert.NotNil(t, route)
	assert.Equal(t, "test-route", route["name"])
	assert.Equal(t, "/test/*", route["uri"])
}

func TestExpandRouteWithLabels(t *testing.T) {
	resource := ResourceApisixRoute()
	d := resource.TestResourceData()

	d.Set("name", "labeled-route")
	d.Set("uri", "/test")
	d.Set("labels", map[string]interface{}{
		"env":  "production",
		"team": "platform",
	})

	route := expandRoute(d)

	assert.NotNil(t, route)
	assert.NotNil(t, route["labels"])
}

func TestExpandRouteWithPlugins(t *testing.T) {
	resource := ResourceApisixRoute()
	d := resource.TestResourceData()

	d.Set("name", "plugin-route")
	d.Set("uri", "/api/*")
	d.Set("plugins", map[string]interface{}{
		"limit-count": `{"count":100,"time_window":60}`,
	})

	route := expandRoute(d)

	assert.NotNil(t, route)
	assert.NotNil(t, route["plugins"])
}

func TestExpandRouteWithTimeout(t *testing.T) {
	resource := ResourceApisixRoute()
	d := resource.TestResourceData()

	d.Set("name", "timeout-route")
	d.Set("uri", "/test")
	d.Set("timeout", []interface{}{
		map[string]interface{}{
			"connect": 5,
			"send":    10,
			"read":    30,
		},
	})

	route := expandRoute(d)

	assert.NotNil(t, route)
	assert.NotNil(t, route["timeout"])
}

func TestFlattenRoute(t *testing.T) {
	resource := ResourceApisixRoute()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"name":   "test-route",
		"uri":    "/test",
		"status": 1,
	}

	diags := flattenRoute(d, value)

	assert.Len(t, diags, 0)
	assert.Equal(t, "test-route", d.Get("name"))
	assert.Equal(t, "/test", d.Get("uri"))
}

func TestFlattenRouteWithLabels(t *testing.T) {
	resource := ResourceApisixRoute()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"name": "labeled-route",
		"uri":  "/test",
		"labels": map[string]interface{}{
			"env":  "production",
			"team": "platform",
		},
	}

	diags := flattenRoute(d, value)

	assert.Len(t, diags, 0)
	labels := d.Get("labels").(map[string]interface{})
	assert.Equal(t, "production", labels["env"])
}

func TestFlattenRouteWithNilValue(t *testing.T) {
	resource := ResourceApisixRoute()
	d := resource.TestResourceData()

	var value interface{} = nil

	diags := flattenRoute(d, value)

	assert.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "failed to convert route data")
}

func TestFlattenRouteInvalidType(t *testing.T) {
	resource := ResourceApisixRoute()
	d := resource.TestResourceData()

	value := "invalid-type"

	diags := flattenRoute(d, value)

	assert.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "failed to convert route data")
}
