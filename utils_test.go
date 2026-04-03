package walle

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRefPathParts(t *testing.T) {
	t.Run("Reference path parsing", func(t *testing.T) {
		must := require.New(t)
		utils := &validateUtils{}
		context := newValidationContext()

		// valid reference path
		parts, err := utils.GetRefPathParts("#/$defs/user", context, newSchemaPath("root"))
		must.NoError(err)
		must.Equal([]string{"$defs", "user"}, parts)

		// root reference
		parts, err = utils.GetRefPathParts("#", context, newSchemaPath("root"))
		must.NoError(err)
		must.Equal([]string{"root"}, parts)
		parts, err = utils.GetRefPathParts("#", context, newSchemaPath(""))
		must.NoError(err)
		must.Equal([]string{"root"}, parts)

		// invalid ref path
		_, err = utils.GetRefPathParts("", context, newSchemaPath("root"))
		must.Error(err)
		must.Contains(strings.ToLower(err.Error()), "only local references are supported")

		_, err = utils.GetRefPathParts("$defs/user", context, newSchemaPath("root"))
		must.Error(err)
		must.Contains(strings.ToLower(err.Error()), "only local references are supported")
	})
}

func TestIsTypeMatch(t *testing.T) {
	t.Run("Type matching validation", func(t *testing.T) {
		must := require.New(t)
		utils := &validateUtils{}
		context := newValidationContext()
		path := newSchemaPath("root")

		// Test string type
		match, err := utils.IsTypeMatch("test", "string", context, path)
		must.NoError(err)
		must.True(match)

		// Test number type
		match, err = utils.IsTypeMatch(42.0, "number", context, path)
		must.NoError(err)
		must.True(match)

		match, err = utils.IsTypeMatch(42.0, "integer", context, path)
		must.NoError(err)
		must.True(match)

		// Test boolean type
		match, err = utils.IsTypeMatch(true, "boolean", context, path)
		must.NoError(err)
		must.True(match)

		// Test array type
		match, err = utils.IsTypeMatch([]any{1, 2, 3}, "array", context, path)
		must.NoError(err)
		must.True(match)

		// Test object type
		match, err = utils.IsTypeMatch(map[string]any{"key": "value"}, "object", context, path)
		must.NoError(err)
		must.True(match)

		// Test type mismatch
		match, err = utils.IsTypeMatch("string", "number", context, path)
		must.Error(err)
		must.False(match)

		// Test null type
		match, err = utils.IsTypeMatch(nil, "null", context, path)
		must.NoError(err)
		must.True(match)

		// Test invalid type
		_, err = utils.IsTypeMatch("value", "invalid-type", context, path)
		must.Error(err)
		must.Contains(strings.ToLower(err.Error()), "invalid type")
	})
}

func TestIsValidNumber(t *testing.T) {
	t.Run("Number validation", func(t *testing.T) {
		must := require.New(t)
		utils := &validateUtils{}
		context := newValidationContext()
		path := newSchemaPath("root")

		// Test valid number
		err := utils.IsValidNumber(123.45, context, path)
		must.NoError(err)

		// Test maximum range
		err = utils.IsValidNumber(9007199254740991.0, context, path)
		must.NoError(err)

		// Test minimum range
		err = utils.IsValidNumber(-9007199254740991.0, context, path)
		must.NoError(err)

		// Test Inf and NaN
		err = utils.IsValidNumber(math.Inf(1), context, path)
		must.Error(err)
		must.Contains(strings.ToLower(err.Error()), "nan or infinity not allowed")

		err = utils.IsValidNumber(math.NaN(), context, path)
		must.Error(err)
		must.Contains(strings.ToLower(err.Error()), "nan or infinity not allowed")

		// Test scientific notation (which should be rejected)
		// scientificNotation := 1.23e5
		// err = utils.IsValidNumber(scientificNotation, context, path)
		// must.Error(err)
		// must.Contains(strings.ToLower(err.Error()), "scientific notation not allowed")

		// Test zero
		err = utils.IsValidNumber(0.0, context, path)
		must.NoError(err)

		// Test negative zero
		// err = utils.IsValidNumber(-0.0, context, path)
		// must.NoError(err)

		// Test decimal starting with 0
		decimalWithLeadingZero := 0.123
		err = utils.IsValidNumber(decimalWithLeadingZero, context, path)
		must.NoError(err)

		// Test decimal starting with -0
		negativeDecimalWithLeadingZero := -0.123
		err = utils.IsValidNumber(negativeDecimalWithLeadingZero, context, path)
		must.NoError(err)
	})
}

func TestResolveSubschema(t *testing.T) {
	t.Run("Schema resolution", func(t *testing.T) {
		must := require.New(t)
		utils := &validateUtils{}
		context := newValidationContext()

		// Create test schema
		rootSchema := SchemaDict{
			"type": "object",
			"properties": SchemaDict{
				"name": SchemaDict{
					"type": "string",
				},
				"nested": SchemaDict{
					"type": "object",
					"properties": SchemaDict{
						"value": SchemaDict{
							"type": "number",
						},
					},
				},
			},
			"$defs": SchemaDict{
				"userId": SchemaDict{
					"type": "integer",
				},
			},
		}

		context.SchemaRoot = rootSchema

		// Test simple path
		path := newSchemaPath("properties.name")
		subschema, err := utils.ResolveSubschema(rootSchema, path, context, path)
		must.NoError(err)
		must.Equal("string", subschema["type"])

		// Test nested path
		path = newSchemaPath("properties.nested.properties.value")
		subschema, err = utils.ResolveSubschema(rootSchema, path, context, path)
		must.NoError(err)
		must.Equal("number", subschema["type"])

		// Test $defs path
		path = newSchemaPath("$defs.userId")
		subschema, err = utils.ResolveSubschema(rootSchema, path, context, path)
		must.NoError(err)
		must.Equal("integer", subschema["type"])

		// Test non-existent path
		path = newSchemaPath("properties.missing")
		subschema, err = utils.ResolveSubschema(rootSchema, path, context, path)
		must.NoError(err)
		must.Nil(subschema)

		// Test non-object type path
		rootSchema = SchemaDict{
			"type": "string",
		}
		path = newSchemaPath("properties")
		subschema, err = utils.ResolveSubschema(rootSchema, path, context, path)
		must.NoError(err)
		must.Nil(subschema)
	})
}

func TestCalculateSchemaSize(t *testing.T) {
	t.Run("Schema size calculation", func(t *testing.T) {
		must := require.New(t)
		utils := &validateUtils{}

		// Test empty schema
		size := utils.CalculateSchemaSize(nil)
		must.Equal(4, size)

		size = utils.CalculateSchemaSize(SchemaDict{})
		must.Equal(2, size)

		// Test simple schema
		schema := SchemaDict{
			"type": "string",
		}
		size = utils.CalculateSchemaSize(schema)
		must.Equal(17, size)

		// Test array within SchemaDict
		schema = SchemaDict{
			"items": []any{
				"value1",
				"value2",
				SchemaDict{"type": "string"},
			},
		}
		size = utils.CalculateSchemaSize(schema)
		must.Equal(47, size)

		// Test schema that might cause json.Marshal to fail
		// Create a schema with circular reference (which would cause json.Marshal to fail)
		circularSchema := SchemaDict{}
		circularSchema["self"] = circularSchema
		// This should return 0 as circular references can't be marshaled
		size = utils.CalculateSchemaSize(circularSchema)
		must.Equal(0, size)

		// Test complex schema
		schema = SchemaDict{
			"type": "object",
			"properties": SchemaDict{
				"name": SchemaDict{
					"type": "string",
				},
				"age": SchemaDict{
					"type": "integer",
				},
			},
			"required": []any{"name"},
		}
		size = utils.CalculateSchemaSize(schema)
		must.Equal(102, size)

		complexSchema := SchemaDict{
			"type": "object",
			"properties": SchemaDict{
				"deep": SchemaDict{
					"type": "object",
					"properties": SchemaDict{
						"deeper": SchemaDict{
							"type": "object",
							"properties": SchemaDict{
								"deepest": SchemaDict{
									"type": "string",
								},
							},
						},
					},
				},
			},
		}
		size = utils.CalculateSchemaSize(complexSchema)
		must.Equal(size, 142)
	})
}

func TestNewSchemaPathFromParts(t *testing.T) {
	t.Run("Schema path creation from parts", func(t *testing.T) {
		must := require.New(t)

		// Test with empty parts (should return root)
		path := newSchemaPathFromParts([]string{})
		must.Equal([]string{"root"}, path.Parts)
		must.True(path.IsRoot())

		// Test with single part
		path = newSchemaPathFromParts([]string{"properties"})
		must.Equal([]string{"properties"}, path.Parts)
		must.Equal("properties", path.String())

		// Test with multiple parts
		path = newSchemaPathFromParts([]string{"properties", "name", "type"})
		must.Equal([]string{"properties", "name", "type"}, path.Parts)
		must.Equal("properties.name.type", path.String())

		// Test with nil
		path = newSchemaPathFromParts(nil)
		must.Equal([]string{"root"}, path.Parts)
		must.True(path.IsRoot())
	})
}

func TestTraverseAndCheckRefs(t *testing.T) {
	t.Run("Reference traversal and checking", func(t *testing.T) {
		must := require.New(t)

		// Setup test validator and context
		validator := newSchemaValidator(WithValidateLevel(ValidateLevelTest))
		context := newValidationContext()
		validator.context = context
		validator.utils = &validateUtils{}

		// Test with nil schema (should return nil)
		err := validator.TraverseAndCheckRefs(nil, true, nil, newSchemaPath(""))
		must.NoError(err)

		// Setup root schema with references
		rootSchema := SchemaDict{
			"$defs": SchemaDict{
				"simpleType": SchemaDict{
					"type": "string",
				},
				"complexType": SchemaDict{
					"$ref": "#/$defs/simpleType",
				},
				"recursiveType": SchemaDict{
					"type": "object",
					"properties": SchemaDict{
						"self": SchemaDict{
							"$ref": "#/$defs/recursiveType",
						},
					},
					"required": []any{"self"},
				},
				"recursiveType2": SchemaDict{
					"$ref": "#/$defs/circular1",
				},
				"circular1": SchemaDict{
					"$ref": "#/$defs/circular2",
				},
				"circular2": SchemaDict{
					"$ref": "#/$defs/circular1",
				},
			},
			"type": "object",
			"properties": SchemaDict{
				"name": SchemaDict{
					"$ref": "#/$defs/simpleType",
				},
				"complex": SchemaDict{
					"$ref": "#/$defs/complexType",
				},
				"recursive": SchemaDict{
					"$ref": "#/$defs/recursiveType",
				},
				"recursive2": SchemaDict{
					"$ref": "#/$defs/recursiveType2",
				},
			},
			"required": []any{"complex", "recursive", "recursive2"},
		}

		context.SchemaRoot = rootSchema

		// Test reference expansion with simple reference
		schema := SchemaDict{
			"$ref": "#/$defs/simpleType",
		}
		err = validator.TraverseAndCheckRefs(schema, false, nil, newSchemaPath("properties.name"))
		must.NoError(err)

		// Test infinite recursion
		schema = SchemaDict{
			"$ref": "#/$defs/recursiveType",
		}
		err = validator.TraverseAndCheckRefs(schema, true, nil, newSchemaPath("properties.recursive"))
		must.Error(err)
		must.Contains(strings.ToLower(err.Error()), "detected infinite recursion")
		schema = SchemaDict{
			"$ref": "#/$defs/recursiveType2",
		}
		err = validator.TraverseAndCheckRefs(schema, true, nil, newSchemaPath("properties.recursive2"))
		must.Error(err)
		must.Contains(strings.ToLower(err.Error()), "detected infinite recursion")

		if validator.config.IsUltra() || validator.config.IsTest() {
			// Test reference with conflicting keywords
			schema = SchemaDict{
				"$ref": "#/$defs/simpleType",
				"type": "number",
			}
			err = validator.TraverseAndCheckRefs(schema, true, nil, newSchemaPath("properties.conflicting"))
			must.Error(err)
			must.Contains(strings.ToLower(err.Error()), "conflicting keywords")
		}

		// Test traversal of complex nested structure
		// "name" is not required
		schema = rootSchema
		err = validator.TraverseAndCheckRefs(schema, false, nil, newSchemaPath(""))
		must.NoError(err)
	})
}

func TestExpandRef(t *testing.T) {
	t.Run("Reference expansion", func(t *testing.T) {
		must := require.New(t)

		// Setup test validator and context
		validator := newSchemaValidator(WithValidateLevel(ValidateLevelTest))
		context := newValidationContext()
		validator.context = context
		validator.utils = &validateUtils{}

		// Setup root schema with references
		rootSchema := SchemaDict{
			"$defs": SchemaDict{
				"simpleType": SchemaDict{
					"type": "string",
				},
				"nestedRef": SchemaDict{
					"$ref":        "#/$defs/simpleType",
					"description": "A nested reference",
				},
				"doubleNestedRef": SchemaDict{
					"$ref":  "#/$defs/nestedRef",
					"title": "Double nested reference",
				},
				"circularRef": SchemaDict{
					"$ref": "#/$defs/circularRef",
				},
				"invalidRef": SchemaDict{
					"$ref": 123,
				},
			},
		}

		context.SchemaRoot = rootSchema

		// Test expansion of nil schema
		expanded, err := validator.ExpandRef(nil, make(map[string]struct{}), newSchemaPath(""))
		must.NoError(err)
		must.Nil(expanded)

		// Test expansion of schema without $ref
		schema := SchemaDict{
			"type": "string",
		}
		expanded, err = validator.ExpandRef(schema, make(map[string]struct{}), newSchemaPath(""))
		must.NoError(err)
		must.Equal(schema, expanded)

		// Test expansion of simple reference
		schema = SchemaDict{
			"$ref": "#/$defs/simpleType",
		}
		expanded, err = validator.ExpandRef(schema, make(map[string]struct{}), newSchemaPath(""))
		must.NoError(err)
		must.Equal("string", expanded["type"])

		if validator.config.IsUltra() || validator.config.IsTest() {
			// Test expansion of nested reference
			schema = SchemaDict{
				"$ref": "#/$defs/nestedRef",
			}
			expanded, err = validator.ExpandRef(schema, make(map[string]struct{}), newSchemaPath(""))
			must.NoError(err)
			must.Equal("string", expanded["type"])
			must.Equal("A nested reference", expanded["description"])

			// Test expansion of double nested reference
			schema = SchemaDict{
				"$ref": "#/$defs/doubleNestedRef",
			}
			expanded, err = validator.ExpandRef(schema, make(map[string]struct{}), newSchemaPath(""))
			must.NoError(err)
			must.Equal("string", expanded["type"])
			must.Equal("A nested reference", expanded["description"])
			must.Equal("Double nested reference", expanded["title"])
		}

		// Test handling of circular references
		schema = SchemaDict{
			"$ref": "#/$defs/circularRef",
		}
		expanded, err = validator.ExpandRef(schema, make(map[string]struct{}), newSchemaPath(""))
		must.NoError(err)
		must.Contains(expanded, "$ref")

		// Test invalid reference type
		schema = SchemaDict{
			"$ref": "#/$defs/invalidRef",
		}
		_, err = validator.ExpandRef(schema, make(map[string]struct{}), newSchemaPath(""))
		must.Error(err)
		must.Contains(strings.ToLower(err.Error()), "ref must be a string")
	})
}

func TestCheckRefContext(t *testing.T) {
	t.Run("Reference context checking", func(t *testing.T) {
		must := require.New(t)

		// Setup test validator and context
		validator := newSchemaValidator(WithValidateLevel(ValidateLevelTest))
		context := newValidationContext()
		validator.context = context

		// Test with nil refSchema
		err := validator.CheckRefContext(SchemaDict{}, nil, newSchemaPath(""))
		must.NoError(err)

		// Test with no conflicts
		parent := SchemaDict{
			"$ref":        "#/$defs/type",
			"description": "A string type",
		}
		refSchema := SchemaDict{
			"type": "string",
		}
		err = validator.CheckRefContext(parent, refSchema, newSchemaPath(""))
		must.NoError(err)

		if validator.config.IsUltra() || validator.config.IsTest() {
			// Test with direct keyword conflict
			parent = SchemaDict{
				"$ref": "#/$defs/type",
				"type": "number", // Conflicts with refSchema
			}
			refSchema = SchemaDict{
				"type": "string",
			}
			err = validator.CheckRefContext(parent, refSchema, newSchemaPath(""))
			must.Error(err)
			must.Contains(strings.ToLower(err.Error()), "conflicting keywords")

			// Test type + anyOf conflict
			parent = SchemaDict{
				"$ref": "#/$defs/type",
				"type": "string",
			}
			refSchema = SchemaDict{
				"anyOf": []any{
					SchemaDict{"type": "number"},
					SchemaDict{"type": "boolean"},
				},
			}
			err = validator.CheckRefContext(parent, refSchema, newSchemaPath(""))
			must.Error(err)
			must.Contains(strings.ToLower(err.Error()), "invalid schema after $ref expansion")
		}

		// Test normal anyOf without conflicts
		parent = SchemaDict{
			"$ref": "#/$defs/type",
		}
		refSchema = SchemaDict{
			"anyOf": []any{
				SchemaDict{"type": "number"},
				SchemaDict{"type": "string"},
			},
		}
		err = validator.CheckRefContext(parent, refSchema, newSchemaPath(""))
		must.NoError(err)
	})
}

func TestMakeSubSchema(t *testing.T) {
	must := require.New(t)

	// Test valid JSON string
	validJSON := `{"type": "object", "properties": {"name": {"type": "string"}}}`
	schema1, err1 := ParseSchema(validJSON)
	must.NoError(err1)
	must.NotNil(schema1)
	must.Equal("object", schema1["type"])
	must.NotNil(schema1["properties"])

	// Test invalid JSON format
	invalidJSON := `{"type": "object", "properties": {`
	schema2, err2 := ParseSchema(invalidJSON)
	must.Error(err2)
	must.Nil(schema2)
	must.True(IsUnmarshalError(err2), "Error should be an UnmarshalError")

	// Test non-object JSON
	nonObjectJSON := `"just a string"`
	schema3, err3 := ParseSchema(nonObjectJSON)
	must.Error(err3)
	must.Nil(schema3)
	must.True(IsUnmarshalError(err2), "Error should be an UnmarshalError")

	// Test JSON array
	arrayJSON := `[1, 2, 3]`
	schema4, err4 := ParseSchema(arrayJSON)
	must.Error(err4)
	must.Nil(schema4)
	must.True(IsUnmarshalError(err2), "Error should be an UnmarshalError")

	// Test empty JSON object
	emptyJSON := `{}`
	schema5, err5 := ParseSchema(emptyJSON)
	must.NoError(err5)
	must.NotNil(schema5)
	must.Equal(0, len(schema5))

	// Test complex nested JSON
	complexJSON := `{
		"type": "object",
		"properties": {
			"person": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "integer", "minimum": 0}
				},
				"required": ["name"]
			}
		}
	}`
	schema6, err6 := ParseSchema(complexJSON)
	must.NoError(err6)
	must.NotNil(schema6)
	must.Equal("object", schema6["type"])
	properties, ok := schema6["properties"].(map[string]any)
	must.True(ok)
	person, ok := properties["person"].(map[string]any)
	must.True(ok)
	must.Equal("object", person["type"])
}

func TestSchemaPath(t *testing.T) {
	must := require.New(t)

	// Test empty path
	emptyPath := schemaPath{Parts: []string{}}
	must.Equal("", emptyPath.Last(), "Empty path should return empty string")

	// Test single element path
	singlePath := schemaPath{Parts: []string{"properties"}}
	must.Equal("properties", singlePath.Last(), "Single element path should return that element")

	// Test multi-element path
	multiPath := schemaPath{Parts: []string{"properties", "name", "type"}}
	must.Equal("type", multiPath.Last(), "Multi-element path should return last element")

	// Test with root path
	rootPath := rootSchemaPath
	last := rootPath.Last()
	must.Equal(rootPath.Parts[len(rootPath.Parts)-1], last, "Root path Last() should match last element")

	// Test with path containing empty strings
	mixedPath := schemaPath{Parts: []string{"properties", "", "type"}}
	must.Equal("type", mixedPath.Last(), "Path with empty strings should still return last element")

	// Test with path ending in anyOf
	anyOfPath := schemaPath{Parts: []string{"properties", "address", "anyOf"}}
	modifiedPath := anyOfPath.ModifyAnyOfPart(2)
	must.Equal([]string{"properties", "address", "anyOf{2}"}, modifiedPath.Parts, "Should append index to anyOf")

	// Test with empty path
	modifiedEmptyPath := emptyPath.ModifyAnyOfPart(0)
	must.Equal([]string{}, modifiedEmptyPath.Parts, "Empty path should remain empty")
}
