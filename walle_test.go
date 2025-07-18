package walle_test

import (
	"testing"

	"github.com/moonshotai/walle"
	"github.com/stretchr/testify/require"
)

func TestParseSchema(t *testing.T) {
	must := require.New(t)

	// valid schema
	validSchemaStr := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name"]
	}`
	schema, err := walle.ParseSchema(validSchemaStr)
	must.NoError(err, "Creating valid schema should not return error")
	must.NotNil(schema, "Schema should not be nil")
	err = schema.Validate(
		walle.WithMaxSchemaDepth(10),
		walle.WithMaxSchemaSize(30000),
	)
	must.NoError(err)
	props, ok := schema["properties"].(map[string]any)
	must.True(ok, "Schema should have properties field")
	must.Contains(props, "name", "Properties should contain 'name'")
	must.Contains(props, "age", "Properties should contain 'age'")

	// Invalid schema
	invalidSchemaStr := `{
		"type": "invalid-type"
	}`
	schema, err = walle.ParseSchema(invalidSchemaStr)
	must.NoError(err)
	err = schema.Validate()
	must.Contains(err.Error(), "invalid type", "Invalid schema should return error")
	must.True(walle.IsSchemaError(err), "Invalid schema should return error")

	// Test JSON syntax error
	invalidJSONStr := `{
		"type": "object",
		"properties": {
			123: {"type": "string"}
		}
	}`
	_, err = walle.ParseSchema(invalidJSONStr)
	must.True(walle.IsUnmarshalError(err), "Invalid JSON should return error")
}
