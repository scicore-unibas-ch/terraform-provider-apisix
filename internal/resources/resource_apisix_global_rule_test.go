package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceApisixGlobalRule(t *testing.T) {
	resource := ResourceApisixGlobalRule()

	assert.NotNil(t, resource)
	assert.NotNil(t, resource.CreateContext)
	assert.NotNil(t, resource.ReadContext)
	assert.NotNil(t, resource.UpdateContext)
	assert.NotNil(t, resource.DeleteContext)
	assert.NotNil(t, resource.Importer)
}

func TestResourceApisixGlobalRuleSchema(t *testing.T) {
	resource := ResourceApisixGlobalRule()

	schema := resource.Schema

	// Test rule_id field
	assert.NotNil(t, schema["rule_id"])
	assert.True(t, schema["rule_id"].Required)
	assert.True(t, schema["rule_id"].ForceNew)

	// Test plugins field
	assert.NotNil(t, schema["plugins"])
	assert.True(t, schema["plugins"].Required)
}

func TestExpandGlobalRule(t *testing.T) {
	resource := ResourceApisixGlobalRule()
	d := resource.TestResourceData()

	d.Set("rule_id", "test-global-rule")
	d.Set("plugins", map[string]interface{}{
		"limit-count": `{"count":1000,"time_window":60}`,
		"cors":        `{"allow_origins":"*"}`,
	})

	rule := expandGlobalRule(d)

	assert.Equal(t, "test-global-rule", rule["id"])
	assert.NotNil(t, rule["plugins"])

	plugins := rule["plugins"].(map[string]interface{})
	assert.NotNil(t, plugins["limit-count"])
	assert.NotNil(t, plugins["cors"])
}

func TestExpandGlobalRuleMinimal(t *testing.T) {
	resource := ResourceApisixGlobalRule()
	d := resource.TestResourceData()

	d.Set("rule_id", "minimal-rule")
	d.Set("plugins", map[string]interface{}{
		"limit-count": `{"count":500,"time_window":60}`,
	})

	rule := expandGlobalRule(d)

	assert.Equal(t, "minimal-rule", rule["id"])
	assert.NotNil(t, rule["plugins"])
}

func TestExpandGlobalRuleMultiplePlugins(t *testing.T) {
	resource := ResourceApisixGlobalRule()
	d := resource.TestResourceData()

	d.Set("rule_id", "multi-plugin-rule")
	d.Set("plugins", map[string]interface{}{
		"limit-count":    `{"count":100,"time_window":60}`,
		"cors":           `{"allow_origins":"*"}`,
		"ip-restriction": `{"blacklist":["127.0.0.1"]}`,
	})

	rule := expandGlobalRule(d)

	assert.Equal(t, "multi-plugin-rule", rule["id"])
	plugins := rule["plugins"].(map[string]interface{})
	assert.Len(t, plugins, 3)
	assert.NotNil(t, plugins["limit-count"])
	assert.NotNil(t, plugins["cors"])
	assert.NotNil(t, plugins["ip-restriction"])
}

func TestFlattenGlobalRule(t *testing.T) {
	resource := ResourceApisixGlobalRule()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"id": "test-rule",
		"plugins": map[string]interface{}{
			"limit-count": map[string]interface{}{
				"count":       1000,
				"time_window": 60,
			},
		},
	}

	diags := flattenGlobalRule(d, value)

	assert.Len(t, diags, 0)
	assert.Equal(t, "test-rule", d.Get("rule_id"))
	assert.NotNil(t, d.Get("plugins"))
}

func TestFlattenGlobalRuleEmptyPlugins(t *testing.T) {
	resource := ResourceApisixGlobalRule()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"id":      "test-rule",
		"plugins": map[string]interface{}{},
	}

	diags := flattenGlobalRule(d, value)

	assert.Len(t, diags, 0)
	assert.Equal(t, "test-rule", d.Get("rule_id"))
	assert.NotNil(t, d.Get("plugins"))
}

func TestFlattenGlobalRuleWithNilValue(t *testing.T) {
	resource := ResourceApisixGlobalRule()
	d := resource.TestResourceData()

	var value interface{} = nil

	diags := flattenGlobalRule(d, value)

	assert.Len(t, diags, 1)
	assert.Equal(t, "failed to convert global rule data", diags[0].Summary)
}

func TestFlattenGlobalRuleInvalidType(t *testing.T) {
	resource := ResourceApisixGlobalRule()
	d := resource.TestResourceData()

	value := "invalid-type"

	diags := flattenGlobalRule(d, value)

	assert.Len(t, diags, 1)
	assert.Equal(t, "failed to convert global rule data", diags[0].Summary)
}
