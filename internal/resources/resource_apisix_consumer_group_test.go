package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceApisixConsumerGroup(t *testing.T) {
	resource := ResourceApisixConsumerGroup()
	assert.NotNil(t, resource)
	assert.NotNil(t, resource.CreateContext)
	assert.NotNil(t, resource.ReadContext)
	assert.NotNil(t, resource.UpdateContext)
	assert.NotNil(t, resource.DeleteContext)
	assert.NotNil(t, resource.Importer)
}

func TestResourceApisixConsumerGroupSchema(t *testing.T) {
	resource := ResourceApisixConsumerGroup()
	schema := resource.Schema

	assert.NotNil(t, schema["group_id"])
	assert.True(t, schema["group_id"].Required)
	assert.True(t, schema["group_id"].ForceNew)
	assert.NotNil(t, schema["name"])
	assert.NotNil(t, schema["desc"])
	assert.NotNil(t, schema["plugins"])
	assert.NotNil(t, schema["labels"])
}

func TestExpandConsumerGroup(t *testing.T) {
	resource := ResourceApisixConsumerGroup()
	d := resource.TestResourceData()

	d.Set("group_id", "test-group")
	d.Set("name", "Test Group")
	d.Set("desc", "Test consumer group")

	group := expandConsumerGroup(d)

	assert.NotNil(t, group)
	assert.Equal(t, "test-group", group["id"])
	assert.Equal(t, "Test Group", group["name"])
}

func TestExpandConsumerGroupWithLabels(t *testing.T) {
	resource := ResourceApisixConsumerGroup()
	d := resource.TestResourceData()

	d.Set("group_id", "labeled-group")
	d.Set("labels", map[string]interface{}{
		"env":  "production",
		"team": "platform",
	})

	group := expandConsumerGroup(d)

	assert.NotNil(t, group)
	assert.NotNil(t, group["labels"])
}

func TestExpandConsumerGroupWithPlugins(t *testing.T) {
	resource := ResourceApisixConsumerGroup()
	d := resource.TestResourceData()

	d.Set("group_id", "plugin-group")
	d.Set("plugins", map[string]interface{}{
		"limit-count": `{"count":100,"time_window":60}`,
	})

	group := expandConsumerGroup(d)

	assert.NotNil(t, group)
	assert.NotNil(t, group["plugins"])
}

func TestFlattenConsumerGroup(t *testing.T) {
	resource := ResourceApisixConsumerGroup()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"id":   "test-group",
		"name": "Test Group",
		"desc": "Test consumer group",
	}

	diags := flattenConsumerGroup(d, value)

	assert.Len(t, diags, 0)
	assert.Equal(t, "test-group", d.Get("group_id"))
	assert.Equal(t, "Test Group", d.Get("name"))
}

func TestFlattenConsumerGroupWithLabels(t *testing.T) {
	resource := ResourceApisixConsumerGroup()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"id":   "labeled-group",
		"name": "Labeled Group",
		"labels": map[string]interface{}{
			"env":  "production",
			"team": "platform",
		},
	}

	diags := flattenConsumerGroup(d, value)

	assert.Len(t, diags, 0)
	labels := d.Get("labels").(map[string]interface{})
	assert.Equal(t, "production", labels["env"])
}

func TestFlattenConsumerGroupWithNilValue(t *testing.T) {
	resource := ResourceApisixConsumerGroup()
	d := resource.TestResourceData()

	var value interface{} = nil

	diags := flattenConsumerGroup(d, value)

	assert.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "failed to convert consumer group data")
}

func TestFlattenConsumerGroupInvalidType(t *testing.T) {
	resource := ResourceApisixConsumerGroup()
	d := resource.TestResourceData()

	value := "invalid-type"

	diags := flattenConsumerGroup(d, value)

	assert.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "failed to convert consumer group data")
}
