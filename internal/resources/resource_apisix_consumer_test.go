package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceApisixConsumer(t *testing.T) {
	resource := ResourceApisixConsumer()
	assert.NotNil(t, resource)
	assert.NotNil(t, resource.CreateContext)
	assert.NotNil(t, resource.ReadContext)
	assert.NotNil(t, resource.UpdateContext)
	assert.NotNil(t, resource.DeleteContext)
	assert.NotNil(t, resource.Importer)
}

func TestResourceApisixConsumerSchema(t *testing.T) {
	resource := ResourceApisixConsumer()
	schema := resource.Schema

	assert.NotNil(t, schema["username"])
	assert.True(t, schema["username"].Required)
	assert.True(t, schema["username"].ForceNew)
	assert.NotNil(t, schema["desc"])
	assert.NotNil(t, schema["plugins"])
	assert.NotNil(t, schema["labels"])
	assert.NotNil(t, schema["group_id"])
}

func TestExpandConsumer(t *testing.T) {
	resource := ResourceApisixConsumer()
	d := resource.TestResourceData()

	d.Set("username", "test-user")
	d.Set("desc", "Test consumer")

	consumer := expandConsumer(d)

	assert.NotNil(t, consumer)
	assert.Equal(t, "test-user", consumer["username"])
}

func TestExpandConsumerWithLabels(t *testing.T) {
	resource := ResourceApisixConsumer()
	d := resource.TestResourceData()

	d.Set("username", "labeled-user")
	d.Set("labels", map[string]interface{}{
		"env":  "production",
		"team": "platform",
	})

	consumer := expandConsumer(d)

	assert.NotNil(t, consumer)
	assert.NotNil(t, consumer["labels"])
}

func TestExpandConsumerWithPlugins(t *testing.T) {
	resource := ResourceApisixConsumer()
	d := resource.TestResourceData()

	d.Set("username", "plugin-user")
	d.Set("plugins", map[string]interface{}{
		"key-auth": `{"key":"test-key"}`,
	})

	consumer := expandConsumer(d)

	assert.NotNil(t, consumer)
	assert.NotNil(t, consumer["plugins"])
}

func TestExpandConsumerWithGroupId(t *testing.T) {
	resource := ResourceApisixConsumer()
	d := resource.TestResourceData()

	d.Set("username", "grouped-user")
	d.Set("group_id", "test-group")

	consumer := expandConsumer(d)

	assert.NotNil(t, consumer)
	assert.Equal(t, "test-group", consumer["group_id"])
}

func TestFlattenConsumer(t *testing.T) {
	resource := ResourceApisixConsumer()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"username": "test-user",
		"desc":     "Test consumer",
	}

	diags := flattenConsumer(d, value)

	assert.Len(t, diags, 0)
	assert.Equal(t, "test-user", d.Get("username"))
}

func TestFlattenConsumerWithLabels(t *testing.T) {
	resource := ResourceApisixConsumer()
	d := resource.TestResourceData()

	value := map[string]interface{}{
		"username": "labeled-user",
		"labels": map[string]interface{}{
			"env":  "production",
			"team": "platform",
		},
	}

	diags := flattenConsumer(d, value)

	assert.Len(t, diags, 0)
	labels := d.Get("labels").(map[string]interface{})
	assert.Equal(t, "production", labels["env"])
}

func TestFlattenConsumerWithNilValue(t *testing.T) {
	resource := ResourceApisixConsumer()
	d := resource.TestResourceData()

	var value interface{} = nil

	diags := flattenConsumer(d, value)

	assert.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "failed to convert consumer data")
}

func TestFlattenConsumerInvalidType(t *testing.T) {
	resource := ResourceApisixConsumer()
	d := resource.TestResourceData()

	value := "invalid-type"

	diags := flattenConsumer(d, value)

	assert.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "failed to convert consumer data")
}
