package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
)

func TestResourceApisixPluginConfig(t *testing.T) {
	resource := ResourceApisixPluginConfig()

	assert.NotNil(t, resource)
	assert.NotNil(t, resource.CreateContext)
	assert.NotNil(t, resource.ReadContext)
	assert.NotNil(t, resource.UpdateContext)
	assert.NotNil(t, resource.DeleteContext)
	assert.NotNil(t, resource.Importer)
}

func TestResourceApisixPluginConfigSchema(t *testing.T) {
	resource := ResourceApisixPluginConfig()

	schema := resource.Schema

	// Test config_id field
	assert.NotNil(t, schema["config_id"])
	assert.Equal(t, schema["config_id"].Type, schema.TypeString)
	assert.True(t, schema["config_id"].Required)
	assert.True(t, schema["config_id"].ForceNew)

	// Test desc field
	assert.NotNil(t, schema["desc"])
	assert.Equal(t, schema["desc"].Type, schema.TypeString)
	assert.True(t, schema["desc"].Optional)

	// Test plugins field
	assert.NotNil(t, schema["plugins"])
	assert.Equal(t, schema["plugins"].Type, schema.TypeMap)
	assert.True(t, schema["plugins"].Required)

	// Test labels field
	assert.NotNil(t, schema["labels"])
	assert.Equal(t, schema["labels"].Type, schema.TypeMap)
	assert.True(t, schema["labels"].Optional)
	assert.True(t, schema["labels"].Computed)
}

func TestExpandPluginConfig(t *testing.T) {
	resource := ResourceApisixPluginConfig()
	d := resource.TestResourceData()

	d.Set("config_id", "test-config")
	d.Set("desc", "Test plugin config")
	d.Set("plugins", map[string]interface{}{
		"limit-count": `{"count":100,"time_window":60}`,
		"cors":        `{"allow_origins":"*"}`,
	})
	d.Set("labels", map[string]interface{}{
		"env":  "test",
		"team": "platform",
	})

	config := expandPluginConfig(d)

	assert.Equal(t, "test-config", config["id"])
	assert.Equal(t, "Test plugin config", config["desc"])
	assert.NotNil(t, config["plugins"])
	assert.NotNil(t, config["labels"])

	plugins := config["plugins"].(map[string]interface{})
	assert.NotNil(t, plugins["limit-count"])
	assert.NotNil(t, plugins["cors"])
}

func TestExpandPluginConfigMinimal(t *testing.T) {
	resource := ResourceApisixPluginConfig()
	d := resource.TestResourceData()

	d.Set("config_id", "minimal-config")
	d.Set("plugins", map[string]interface{}{
		"limit-count": `{"count":1000,"time_window":60}`,
	})

	config := expandPluginConfig(d)

	assert.Equal(t, "minimal-config", config["id"])
	assert.NotNil(t, config["plugins"])
	assert.Nil(t, config["desc"])
	assert.Nil(t, config["labels"])
}

func TestExpandPluginConfigWithLabels(t *testing.T) {
	resource := ResourceApisixPluginConfig()
	d := resource.TestResourceData()

	d.Set("config_id", "labeled-config")
	d.Set("plugins", map[string]interface{}{
		"cors": `{"allow_origins":"*"}`,
	})
	d.Set("labels", map[string]interface{}{
		"env":        "production",
		"managed-by": "terraform",
	})

	config := expandPluginConfig(d)

	assert.Equal(t, "labeled-config", config["id"])
	assert.NotNil(t, config["labels"])
	labels := config["labels"].(map[string]string)
	assert.Equal(t, "production", labels["env"])
	assert.Equal(t, "terraform", labels["managed-by"])
}

func TestFlattenPluginConfig(t *testing.T) {
	resource := ResourceApisixPluginConfig()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"id":   "test-config",
		"desc": "Test configuration",
		"plugins": map[string]interface{}{
			"limit-count": map[string]interface{}{
				"count":       100,
				"time_window": 60,
			},
		},
		"labels": map[string]interface{}{
			"env":  "test",
			"team": "platform",
		},
	}

	diags := flattenPluginConfig(d, value)

	assert.Len(t, diags, 0)
	assert.Equal(t, "test-config", d.Get("config_id"))
	assert.Equal(t, "Test configuration", d.Get("desc"))
	assert.NotNil(t, d.Get("plugins"))
	assert.NotNil(t, d.Get("labels"))
}

func TestFlattenPluginConfigEmptyLabels(t *testing.T) {
	resource := ResourceApisixPluginConfig()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"id":      "test-config",
		"plugins": map[string]interface{}{},
	}

	diags := flattenPluginConfig(d, value)

	assert.Len(t, diags, 0)
	assert.Equal(t, "test-config", d.Get("config_id"))
	assert.NotNil(t, d.Get("labels"))
	labels := d.Get("labels").(map[string]interface{})
	assert.Empty(t, labels)
}

func TestFlattenPluginConfigWithNilValue(t *testing.T) {
	resource := ResourceApisixPluginConfig()
	d := resource.TestResourceData()

	var value interface{} = nil

	diags := flattenPluginConfig(d, value)

	assert.Len(t, diags, 1)
	assert.Equal(t, "failed to convert plugin config data", diags[0].Summary)
}

func TestFlattenPluginConfigInvalidType(t *testing.T) {
	resource := ResourceApisixPluginConfig()
	d := resource.TestResourceData()

	value := "invalid-type"

	diags := flattenPluginConfig(d, value)

	assert.Len(t, diags, 1)
	assert.Equal(t, "failed to convert plugin config data", diags[0].Summary)
}
