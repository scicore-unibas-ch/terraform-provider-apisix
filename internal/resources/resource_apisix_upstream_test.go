package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceApisixUpstream(t *testing.T) {
	resource := ResourceApisixUpstream()

	assert.NotNil(t, resource)
	assert.NotNil(t, resource.CreateContext)
	assert.NotNil(t, resource.ReadContext)
	assert.NotNil(t, resource.UpdateContext)
	assert.NotNil(t, resource.DeleteContext)
	assert.NotNil(t, resource.Importer)
}

func TestResourceApisixUpstreamSchema(t *testing.T) {
	resource := ResourceApisixUpstream()

	schema := resource.Schema

	// Test key fields
	assert.NotNil(t, schema["name"])
	assert.NotNil(t, schema["type"])
	assert.NotNil(t, schema["nodes"])
	assert.NotNil(t, schema["desc"])
	assert.NotNil(t, schema["labels"])
}

func TestExpandUpstreamBasic(t *testing.T) {
	resource := ResourceApisixUpstream()
	d := resource.TestResourceData()

	d.Set("name", "test-upstream")
	d.Set("type", "roundrobin")
	d.Set("desc", "Test upstream")

	upstream := expandUpstream(d)

	assert.NotNil(t, upstream)
	assert.Equal(t, "roundrobin", upstream.Type)
}

func TestExpandUpstreamWithLabels(t *testing.T) {
	resource := ResourceApisixUpstream()
	d := resource.TestResourceData()

	d.Set("name", "labeled-upstream")
	d.Set("type", "roundrobin")
	d.Set("labels", map[string]interface{}{
		"env":  "production",
		"team": "platform",
	})

	upstream := expandUpstream(d)

	assert.NotNil(t, upstream)
	assert.NotNil(t, upstream.Labels)
	assert.Equal(t, "production", upstream.Labels["env"])
	assert.Equal(t, "platform", upstream.Labels["team"])
}

func TestExpandUpstreamWithNodes(t *testing.T) {
	resource := ResourceApisixUpstream()
	d := resource.TestResourceData()

	d.Set("name", "nodes-upstream")
	d.Set("type", "roundrobin")
	d.Set("nodes", []interface{}{
		map[string]interface{}{
			"host":   "127.0.0.1",
			"port":   8080,
			"weight": 100,
		},
	})

	upstream := expandUpstream(d)

	assert.NotNil(t, upstream)
	assert.NotNil(t, upstream.Nodes)
	assert.NotEmpty(t, upstream.Nodes)
}

func TestExpandUpstreamWithTimeout(t *testing.T) {
	resource := ResourceApisixUpstream()
	d := resource.TestResourceData()

	d.Set("name", "timeout-upstream")
	d.Set("type", "roundrobin")
	d.Set("timeout", []interface{}{
		map[string]interface{}{
			"connect": 5,
			"send":    10,
			"read":    30,
		},
	})

	upstream := expandUpstream(d)

	assert.NotNil(t, upstream)
	assert.NotNil(t, upstream.Timeout)
}

func TestExpandUpstreamWithHealthCheck(t *testing.T) {
	resource := ResourceApisixUpstream()
	d := resource.TestResourceData()

	d.Set("name", "healthcheck-upstream")
	d.Set("type", "roundrobin")
	d.Set("health_check", []interface{}{
		map[string]interface{}{
			"active": []interface{}{
				map[string]interface{}{
					"http_path": []interface{}{"/health"},
					"timeout":   5,
				},
			},
		},
	})

	upstream := expandUpstream(d)

	assert.NotNil(t, upstream)
	assert.NotNil(t, upstream.HealthCheck)
}

func TestFlattenUpstreamBasic(t *testing.T) {
	resource := ResourceApisixUpstream()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"name": "test-upstream",
		"type": "roundrobin",
		"desc": "Test upstream",
	}

	diags := flattenUpstream(d, value)

	assert.Len(t, diags, 0)
	assert.Equal(t, "test-upstream", d.Get("name"))
	assert.Equal(t, "roundrobin", d.Get("type"))
}

func TestFlattenUpstreamWithLabels(t *testing.T) {
	resource := ResourceApisixUpstream()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"name": "labeled-upstream",
		"type": "roundrobin",
		"labels": map[string]interface{}{
			"env":  "production",
			"team": "platform",
		},
	}

	diags := flattenUpstream(d, value)

	assert.Len(t, diags, 0)
	assert.Equal(t, "labeled-upstream", d.Get("name"))
	labels := d.Get("labels").(map[string]interface{})
	assert.Equal(t, "production", labels["env"])
	assert.Equal(t, "platform", labels["team"])
}

func TestFlattenUpstreamWithNilValue(t *testing.T) {
	resource := ResourceApisixUpstream()
	d := resource.TestResourceData()

	var value interface{} = nil

	diags := flattenUpstream(d, value)

	assert.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "failed to convert upstream data")
}

func TestFlattenUpstreamInvalidType(t *testing.T) {
	resource := ResourceApisixUpstream()
	d := resource.TestResourceData()

	value := "invalid-type"

	diags := flattenUpstream(d, value)

	assert.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "failed to convert upstream data")
}

func TestFlattenUpstreamWithNodes(t *testing.T) {
	resource := ResourceApisixUpstream()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"name": "nodes-upstream",
		"type": "roundrobin",
		"nodes": []interface{}{
			map[string]interface{}{
				"host":   "127.0.0.1",
				"port":   8080,
				"weight": 100,
			},
		},
	}

	diags := flattenUpstream(d, value)

	assert.Len(t, diags, 0)
	assert.Equal(t, "nodes-upstream", d.Get("name"))
	assert.NotNil(t, d.Get("nodes"))
}
