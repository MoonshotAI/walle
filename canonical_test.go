package walle

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestCanonicalWithAutoFix(t *testing.T) {
	tests := []struct {
		name             string
		invalidSchema    string
		simplifiedSchema string
	}{
		{
			name: "invalid_properties_type_0",
			invalidSchema: `{
				"type": "object",
				"properties": "invalid_string"
			}`,
			simplifiedSchema: `{
				"type": "object"
			}`,
		},
		{
			name: "invalid_properties_type_1",
			invalidSchema: `{
				"type": "object",
				"properties": "invalid_string",
				"required": "invalid_string"
			}`,
			simplifiedSchema: `{
				"type": "object"
			}`,
		},
		{
			name: "invalid_properties_type_2",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "object",
						"properties": {
							"last": {
								"type": "string",
								"properties": "invalid_string"
							}
						}
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "object",
						"properties": {
							"last": {
								"type": "string"
							}
						}
					}
				}
			}`,
		},
		{
			name: "invalid_required_type",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"required": "invalid_string"
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}`,
		},
		{
			name: "invalid_eum_type",
			invalidSchema: `{
				"type": "string",
				"enum": "invalid_string"
			}`,
			simplifiedSchema: `{
				"type": "string"
			}`,
		},
		{
			name: "invalid_eum_type_1",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"enum": "invalid_string"
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "string"
					}
				}
			}`,
		},
		{
			name: "invalid_eum_type_2",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": ["null"],
						"enum": [false]
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": ["null"]
					}
				}
			}`,
		},
		{
			name: "invalid_defs_type",
			invalidSchema: `{
				"type": "null",
				"$defs": 123
			}`,
			simplifiedSchema: `{
				"type": "null"
			}`,
		},
		{
			name: "invalid_additional_properties_type",
			invalidSchema: `{
				"type": "object",
				"additionalProperties": "invalid"
			}`,
			simplifiedSchema: `{
				"type": "object"
			}`,
		},
		{
			name: "multiple_invalid_fields_0",
			invalidSchema: `{
				"type": "object",
				"properties": "invalid",
				"required": "invalid",
				"enum": "invalid",
				"$defs": 123,
				"additionalProperties": "invalid"
			}`,
			simplifiedSchema: `{
				"type": "object"
			}`,
		},
		{
			name: "multiple_invalid_fields_1",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"x": {
						"type": "object",
						"properties": "invalid",
						"required": "invalid",
						"additionalProperties": "invalid"
					},
					"y": {
						"type": "string",
						"enum": "invalid"
					},
					"z": {
						"type": "array",
						"items": "invalid"
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"x": {
						"type": "object"
					},
					"y": {
						"type": "string"
					},
					"z": {
						"type": "array"
					}
				}
			}`,
		},
		{
			// "$ref":"#" points at a root typed as object while the sibling asks for a
			// string, so nothing can satisfy both. Dropping only the sibling type used
			// to leave "$ref":"#" behind, which retypes the field to object and
			// inverts what the schema asked for. Degrading to {} keeps the field
			// unconstrained instead of quietly asserting the opposite type.
			name: "unsatisfiable_type_with_ref",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"user": {
						"$ref": "#",
						"type": "string"
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"user": {}
				}
			}`,
		},
		// only work in ultra mode
		// {
		// 	name: "duplicate_type",
		// 	invalidSchema: `{
		// 		"type": ["string", "string"]
		// 	}`,
		// 	simplifiedSchema: `{
		// 		"type": ["string"]
		// 	}`,
		// },
		// {
		// 	name: "duplicate_items_in_required_array",
		// 	invalidSchema: `{
		// 		"type": "object",
		// 		"properties": {
		// 			"name": {"type": "string"}
		// 		},
		// 		"required": ["name", "name"]
		// 	}`,
		// 	simplifiedSchema: `{
		// 		"type": "object",
		// 		"properties": {
		// 			"name": {"type": "string"}
		// 		},
		// 		"required": ["name"]
		// 	}`,
		// },
		{
			name:             "invalid_type_0",
			invalidSchema:    `{"type": 123}`,
			simplifiedSchema: `{}`,
		},
		{
			name: "invalid_type_1",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"user": {
						"type": "xxx"
					},
					"addr": {
						"type": "object",
						"properties": {
							"city": {
								"type": "invalid_type"
							}
						}
					},
					"age": {
						"type": "integer"
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"user": {
					},
					"addr": {
						"type": "object",
						"properties": {
							"city": {}
						}
					},
					"age": {
						"type": "integer"
					}
				}
			}`,
		},
		{
			name:             "invalid_type_2",
			invalidSchema:    `{"type": "invalid"}`,
			simplifiedSchema: `{}`,
		},
		{
			name: "invalid_type_3",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"user": {
						"type": "xxx"
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"user": {}
				}
			}`,
		},
		{
			name: "invalid_type_4",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"x": {"type": null}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"x": {}
				}
			}`,
		},
		{
			name: "invalid_type_5",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {"type": "xxx"},
					"age": {"type": "integer"}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {},
					"age": {"type": "integer"}
				}
			}`,
		},
		{
			name: "invalid_properties_key_0",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"user": {
						"type": "xxx"
					},
					"required": {
						"type": "object",
						"properties": {
							"city": {
								"type": "invalid_type"
							}
						}
					},
					"age": {
						"type": "integer"
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
				}
			}`,
		},
		{
			name: "invalid_properties_key_1",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"user": {
						"type": "xxx"
					},
					"addr": {
						"type": "object",
						"properties": {
							"city": {
								"type": "string"
							},
							"street": {
								"type": "object",
								"properties": {
									"door": {
										"type": "string"
									},
									"required": {
										"type": "string"
									}
								}
							}
						}
					},
					"age": {
						"type": "integer"
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"user": {},
					"addr": {
						"type": "object",
						"properties": {
							"city": {
								"type": "string"
							},
							"street": {
								"type": "object",
								"properties": {
								}
							}
						}
					},
					"age": {
						"type": "integer"
					}
				}
			}`,
		},
		{
			name: "invalid_property_schema_must_be_an_object",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"$ref": "#/$defs/User"
					},
					"minLength": 10
				},
				"$defs": {
					"User": {
						"anyOf": [
							{"type": "string"},
							{"type": "object",
								"properties": {
									"name": {"type": "string"}
								}
							}
						]
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
				},
				"$defs": {
					"User": {
						"anyOf": [
							{"type": "string"},
							{"type": "object",
								"properties": {
									"name": {"type": "string"}
								}
							}
						]
					}
				}
			}`,
		},
		{
			name: "items_in_required_array_must_be_strings",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"required": ["name", 123]
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}`,
		},
		{
			// Only the unusable entry goes; "name" is declared and genuinely required,
			// so releasing it along with the empty string would loosen the schema.
			name: "property_names_in_required_array_cannot_be_empty",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"required": ["name", ""]
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"required": ["name"]
			}`,
		},
		{
			name: "type_list_with_enum_0",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": ["string",  "boolean"],
						"enum": [false, true]
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {}
				}
			}`,
		},
		{
			name: "type_list_with_enum_1",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": ["object",  "null"],
						"enum": [null]
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {}
				}
			}`,
		},
		{
			name: "complex_case_0",
			invalidSchema: `{
				"properties": {
					"start_url": {
						"anyOf": [
							{
								"type": "string"
							},
							{
								"type": "null"
							}
						],
						"description": "The URL to navigate to after the browser\nlaunches. If not provided, the browser will open with a blank\npage. (default: :obj:None)",
						"type": [
							"null"
						]
					}
				},
				"type": "object",
				"additionalProperties": false,
				"required": [
					"start_url"
				]
			}`,
			// The outer "type":["null"] applies on top of whichever branch matches, so
			// the string branch can never be reached and is dropped. Keeping it would
			// let the property accept strings that the outer type ruled out.
			simplifiedSchema: `{
				"properties": {
					"start_url": {
						"anyOf": [
							{
								"type": "null"
							}
						],
						"description": "The URL to navigate to after the browser\nlaunches. If not provided, the browser will open with a blank\npage. (default: :obj:None)"
					}
				},
				"type": "object",
				"additionalProperties": false,
				"required": [
					"start_url"
				]
			}`,
		},
		{
			name: "complex_case_1",
			invalidSchema: `{
				"properties": {
					"textElements": {
					"type": "array",
					"items": {
						"anyOf": [
						{
							"description": "Regular text element with optional styling.",
							"properties": {
							"text": {
								"type": "string",
								"description": "Text content. Provide plain text without markdown syntax; use style object for formatting."
							}
							},
							"type": "object",
							"required": [
							"text"
							],
							"additionalProperties": false
						},
						{
							"description": "Mathematical equation element with optional styling.",
							"properties": {
							"style": {
								"$ref": "#\/properties\/textElements\/items\/anyOf\/0\/properties\/style"
							},
							"equation": {
								"type": "string",
								"description": "Mathematical equation content. The formula or expression to display. Format: LaTeX."
							}
							},
							"type": "object",
							"required": [
							"equation"
							],
							"additionalProperties": false
						}
						]
					},
					"description": "Array of text content objects. A block can contain multiple text segments with different styles. Example: [{text:\"Hello\",style:{bold:true}},{text:\" World\",style:{italic:true}}]"
					}
				},
				"type": "object",
				"required": [
					"textElements"
				],
				"additionalProperties": false
			}`,
			simplifiedSchema: `{
				"properties": {
					"textElements": {
					"type": "array",
					"items": {
						"anyOf": [
						{
							"description": "Regular text element with optional styling.",
							"properties": {
							"text": {
								"type": "string",
								"description": "Text content. Provide plain text without markdown syntax; use style object for formatting."
							}
							},
							"type": "object",
							"required": [
							"text"
							],
							"additionalProperties": false
						},
						{
							"description": "Mathematical equation element with optional styling.",
							"properties": {
							"style": {
							},
							"equation": {
								"type": "string",
								"description": "Mathematical equation content. The formula or expression to display. Format: LaTeX."
							}
							},
							"type": "object",
							"required": [
							"equation"
							],
							"additionalProperties": false
						}
						]
					},
					"description": "Array of text content objects. A block can contain multiple text segments with different styles. Example: [{text:\"Hello\",style:{bold:true}},{text:\" World\",style:{italic:true}}]"
					}
				},
				"type": "object",
				"required": [
					"textElements"
				],
				"additionalProperties": false
			}`,
		},
		{
			name: "invalid_ref_type_0",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {"$ref": 123}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {}
				}
			}`,
		},
		{
			name: "defs_properties_name_0",
			invalidSchema: `{
				"type": "object",
				"$defs": {
					"": {"type": "string"}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"$defs": {
				}
			}`,
		},
		{
			name: "defs_properties_name_1",
			invalidSchema: `{
				"$defs": {
					"positive/integer": {
						"type": "integer"
					},
					"x": {
						"type": "object"
					}
				},
				"type": "object"
			}`,
			simplifiedSchema: `{
				"type": "object",
				"$defs": {
					"x": {"type": "object"}
				}
			}`,
		},
		{
			name: "invalid_ref_type_1",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {"$ref": "#/invalid/path"}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {}
				}
			}`,
		},
		{
			name: "invalid_ref_type_2",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"$ref": "#/$defs/"
					}
				},
				"$defs": {
					"User": {
						"anyOf": [
							{"type": "string"},
							{"type": "object",
								"properties": {
									"name": {"type": "string"}
								}
							}
						]
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {}
				},
				"$defs": {
					"User": {
						"anyOf": [
							{"type": "string"},
							{"type": "object",
								"properties": {
									"name": {"type": "string"}
								}
							}
						]
					}
				}
			}`,
		},
		{
			name: "invalid_ref_type_3",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"parent": {
						"$ref": "#/$defs/NonExistent"
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"parent": {}
				}
			}`,
		},
		{
			name: "invalid_description_type",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "object",
						"description": 123
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "object"
					}
				}
			}`,
		},
		{
			name: "invalid_anyOf_type_0",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": "object",
						"anyOf": 123
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {
					}
				}
			}`,
		},
		{
			name:             "invalid_anyOf_type_1",
			invalidSchema:    `{"anyOf": "not an array"}`,
			simplifiedSchema: `{}`,
		},
		{
			name: "invalid_defs_schema_type_0",
			invalidSchema: `{
				"type": "object",
				"$defs": {
					"User": "not an object"
				}
			}`,
			simplifiedSchema: `{
				"type": "object"
			}`,
		},
		{
			name: "invalid_defs_schema_type_1",
			invalidSchema: `{
				"type": "object",
				"$defs": "not an object"
			}`,
			simplifiedSchema: `{
				"type": "object"
			}`,
		},
		{
			name: "invalid_id_type",
			invalidSchema: `{
				"$id": 123,
				"type": "object"
			}`,
			simplifiedSchema: `{"type": "object"}`,
		},
		// only work in ultra mode
		// {
		// 	name: "negative_items_val_0",
		// 	invalidSchema: `{
		// 		"type": "array",
		// 		"items": {"type": "string"},
		// 		"minItems": -1,
		// 		"maxItems": 5
		// 	}`,
		// 	simplifiedSchema: `{
		// 		"type": "array",
		// 		"items": {"type": "string"},
		// 		"minItems": 0,
		// 		"maxItems": 5
		// 	}`,
		// },
		// {
		// 	name: "negative_items_val_1",
		// 	invalidSchema: `{
		// 		"type": "array",
		// 		"items": {"type": "string"},
		// 		"minItems": -1,
		// 		"maxItems": -2
		// 	}`,
		// 	simplifiedSchema: `{
		// 		"type": "array",
		// 		"items": {"type": "string"},
		// 		"minItems": 0,
		// 		"maxItems": 9223372036854775807
		// 	}`,
		// },
		// {
		// 	name: "negative_items_val_2",
		// 	invalidSchema: `{
		// 		"type": "array",
		// 		"items": {"type": "string"},
		// 		"minItems": 10,
		// 		"maxItems": 5
		// 	}`,
		// 	simplifiedSchema: `{
		// 		"type": "array",
		// 		"items": {"type": "string"}
		// 	}`,
		// },
		// {
		// 	name: "negative_length_val_0",
		// 	invalidSchema: `{
		// 		"type": "string",
		// 		"minLength": -1,
		// 		"maxLength": 5
		// 	}`,
		// 	simplifiedSchema: `{
		// 		"type": "string",
		// 		"minLength": 0,
		// 		"maxLength": 5
		// 	}`,
		// },
		// {
		// 	name: "negative_length_val_1",
		// 	invalidSchema: `{
		// 		"type": "string",
		// 		"minLength": -1,
		// 		"maxLength": -2
		// 	}`,
		// 	simplifiedSchema: `{
		// 		"type": "string",
		// 		"minLength": 0,
		// 		"maxLength": 9223372036854775807
		// 	}`,
		// },
		// {
		// 	name: "negative_length_val_2",
		// 	invalidSchema: `{
		// 		"type": "string",
		// 		"minLength": 10,
		// 		"maxLength": 5
		// 	}`,
		// 	simplifiedSchema: `{
		// 		"type": "string"
		// 	}`,
		// },
		// {
		// 	name: "miminum_greater_than_maximum",
		// 	invalidSchema: `{
		// 		"type": "number",
		// 		"minimum": 10,
		// 		"maximum": 5
		// 	}`,
		// 	simplifiedSchema: `{
		// 		"type": "number"
		// 	}`,
		// },
		{
			// The outer minLength applies on top of whichever branch matches, so it
			// is pushed into both; the branch asking for 10 keeps the stricter 20.
			// The outer description carries no constraint and is simply dropped.
			name: "conflicting_keywords_expand_anyOf_0",
			invalidSchema: `{
				"description": "xxx",
				"minLength": 20,
				"anyOf": [
					{
						"description": "yyy",
						"type": "string",
						"minLength": 20
					},
					{
						"description": "zzz",
						"type": "string",
						"minLength": 10
					}
				]
			}`,
			simplifiedSchema: `{
				"anyOf": [
					{
						"description": "yyy",
						"type": "string",
						"minLength": 20
					},
					{
						"description": "zzz",
						"type": "string",
						"minLength": 20
					}
				]
			}`,
		},
		{
			name: "conflicting_keywords_expand_anyOf_1",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"description": "xxx",
						"minLength": 20,
						"anyOf": [
							{
								"description": "yyy",
								"type": "string",
								"minLength": 20
							},
							{
								"description": "zzz",
								"type": "string",
								"minLength": 10
							}
						]
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"anyOf": [
							{
								"description": "yyy",
								"type": "string",
								"minLength": 20
							},
							{
								"description": "zzz",
								"type": "string",
								"minLength": 20
							}
						]
					}
				}
			}`,
		},
		{
			name: "invalid_type_array_0",
			invalidSchema: `{
				"type": ["string", ["number"]]
			}`,
			simplifiedSchema: `{}`,
		},
		{
			name: "invalid_type_array_1",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": ["string", ["number"]]
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {}
				}
			}`,
		},
		{
			name:             "invalid_type_array_2",
			invalidSchema:    `{"type": []}`,
			simplifiedSchema: `{}`,
		},
		{
			name: "invalid_type_array_3",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"type": []
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {}
				}
			}`,
		},
		{
			name: "invalid_minLength_string_value",
			invalidSchema: `{
				"type": "string",
				"minLength": "1"
			}`,
			simplifiedSchema: `{
				"type": "string"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := ParseSchema(tt.invalidSchema)
			if err != nil {
				t.Fatalf("Failed to parse schema: %v", err)
			}

			result, _ := schema.Canonical()
			fixedSchema, err := ParseSchema(result)
			if err != nil {
				t.Errorf("Fixed schema is not valid JSON: %v", err)
				return
			}

			expectedSchema, err := ParseSchema(tt.simplifiedSchema)
			if err != nil {
				t.Errorf("Failed to parse simplified schema: %v", err)
				return
			}

			if !reflect.DeepEqual(expectedSchema, fixedSchema) {
				t.Errorf("Expected simplified schema: %s, but got: %s", expectedSchema, fixedSchema)
			}

			validator := newSchemaValidator(WithValidateLevel(ValidateLevelStrict))
			if err := validator.Validate(fixedSchema); err != nil {
				t.Errorf("Fixed schema should pass validation but failed: %v", err)
			}
		})
	}
}

func TestCanonicalCommonKeywordConflictSimplify(t *testing.T) {
	tests := []struct {
		name             string
		invalidSchema    string
		simplifiedSchema string
	}{
		{
			name: "ref_expansion_removes_outer_description_only",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"variantOptions": {
						"$ref": "#/$defs/VariantOptions",
						"description": "property-level description"
					}
				},
				"$defs": {
					"VariantOptions": {
						"type": "array",
						"description": "defs-level description"
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"variantOptions": {
						"$ref": "#/$defs/VariantOptions"
					}
				},
				"$defs": {
					"VariantOptions": {
						"type": "array",
						"description": "defs-level description"
					}
				}
			}`,
		},
		{
			name: "anyOf_with_parent_removes_outer_description_only",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"description": "outer description",
						"anyOf": [
							{ "type": "string", "description": "branch description" }
						]
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"anyOf": [
							{ "type": "string", "description": "branch description" }
						]
					}
				}
			}`,
		},
		{
			name: "anyOf_after_ref_expansion_removes_outer_description_only",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"$ref": "#/$defs/Foo",
						"description": "outer description"
					}
				},
				"$defs": {
					"Foo": {
						"anyOf": [
							{ "type": "string", "description": "branch description" }
						]
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"$ref": "#/$defs/Foo"
					}
				},
				"$defs": {
					"Foo": {
						"anyOf": [
							{ "type": "string", "description": "branch description" }
						]
					}
				}
			}`,
		},
		{
			name: "chained_ref_removes_outer_defs_description_only",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"foo": { "$ref": "#/$defs/DoubleNested" }
				},
				"$defs": {
					"Simple": { "type": "string" },
					"Nested": {
						"$ref": "#/$defs/Simple",
						"description": "level 1 description"
					},
					"DoubleNested": {
						"$ref": "#/$defs/Nested",
						"description": "level 2 description"
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"foo": { "$ref": "#/$defs/DoubleNested" }
				},
				"$defs": {
					"Simple": { "type": "string" },
					"Nested": {
						"$ref": "#/$defs/Simple",
						"description": "level 1 description"
					},
					"DoubleNested": {
						"$ref": "#/$defs/Nested"
					}
				}
			}`,
		},
		{
			name: "title_conflict_removes_outer_title_only",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"foo": {
						"$ref": "#/$defs/Foo",
						"title": "outer title"
					}
				},
				"$defs": {
					"Foo": { "type": "string", "title": "inner title" }
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"foo": { "$ref": "#/$defs/Foo" }
				},
				"$defs": {
					"Foo": { "type": "string", "title": "inner title" }
				}
			}`,
		},
		{
			name: "description_and_title_conflict_removes_both_outer_keys",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"foo": {
						"$ref": "#/$defs/Foo",
						"description": "outer description",
						"title": "outer title"
					}
				},
				"$defs": {
					"Foo": {
						"type": "string",
						"description": "inner description",
						"title": "inner title"
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"foo": { "$ref": "#/$defs/Foo" }
				},
				"$defs": {
					"Foo": {
						"type": "string",
						"description": "inner description",
						"title": "inner title"
					}
				}
			}`,
		},
		{
			// A constraint and an annotation clashing at once are handled one at a
			// time: the constraint is pushed into the branch with the stricter value
			// winning, and the annotation's outer copy is dropped.
			name: "mixed_structural_conflict_distributes_and_drops_the_annotation",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"description": "xxx",
						"minLength": 20,
						"anyOf": [
							{ "type": "string", "description": "yyy", "minLength": 10 }
						]
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"name": {
						"anyOf": [
							{ "type": "string", "description": "yyy", "minLength": 20 }
						]
					}
				}
			}`,
		},
		{
			name: "multiple_property_conflicts_are_fixed_across_rounds",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"a": {
						"$ref": "#/$defs/A",
						"description": "outer a"
					},
					"b": {
						"$ref": "#/$defs/B",
						"description": "outer b"
					}
				},
				"$defs": {
					"A": { "type": "string", "description": "inner a" },
					"B": { "type": "number", "description": "inner b" }
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"a": { "$ref": "#/$defs/A" },
					"b": { "$ref": "#/$defs/B" }
				},
				"$defs": {
					"A": { "type": "string", "description": "inner a" },
					"B": { "type": "number", "description": "inner b" }
				}
			}`,
		},
		{
			name: "triple_chained_ref_requires_multiple_simplify_rounds",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"foo": { "$ref": "#/$defs/TripleNested" }
				},
				"$defs": {
					"Simple": { "type": "string" },
					"Nested": {
						"$ref": "#/$defs/Simple",
						"description": "level 1 description"
					},
					"DoubleNested": {
						"$ref": "#/$defs/Nested",
						"description": "level 2 description"
					},
					"TripleNested": {
						"$ref": "#/$defs/DoubleNested",
						"description": "level 3 description"
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"foo": { "$ref": "#/$defs/TripleNested" }
				},
				"$defs": {
					"Simple": { "type": "string" },
					"Nested": {
						"$ref": "#/$defs/Simple",
						"description": "level 1 description"
					},
					"DoubleNested": {
						"$ref": "#/$defs/Nested"
					},
					"TripleNested": {
						"$ref": "#/$defs/DoubleNested"
					}
				}
			}`,
		},
		{
			name: "combined_ref_anyof_chained_common_keyword_conflicts",
			invalidSchema: `{
				"type": "object",
				"properties": {
					"variantOptions": {
						"$ref": "#/$defs/VariantOptions",
						"description": "prop variant outer desc",
						"title": "prop variant outer title"
					},
					"contact": {
						"description": "contact outer desc",
						"title": "contact outer title",
						"anyOf": [
							{
								"type": "string",
								"description": "contact branch desc",
								"title": "contact branch title"
							},
							{
								"type": "object",
								"description": "contact obj branch desc"
							}
						]
					},
					"profile": {
						"$ref": "#/$defs/Profile",
						"description": "profile outer desc"
					},
					"chained": {
						"$ref": "#/$defs/TripleNested"
					}
				},
				"$defs": {
					"VariantOptions": {
						"type": "array",
						"description": "defs variant inner desc",
						"title": "defs variant inner title"
					},
					"Profile": {
						"anyOf": [
							{
								"type": "string",
								"description": "profile branch desc",
								"minLength": 1
							},
							{ "type": "number" }
						]
					},
					"Simple": { "type": "string" },
					"Nested": {
						"$ref": "#/$defs/Simple",
						"description": "chain level 1 desc",
						"title": "chain level 1 title"
					},
					"DoubleNested": {
						"$ref": "#/$defs/Nested",
						"description": "chain level 2 desc",
						"title": "chain level 2 title"
					},
					"TripleNested": {
						"$ref": "#/$defs/DoubleNested",
						"description": "chain level 3 desc",
						"title": "chain level 3 title"
					}
				}
			}`,
			simplifiedSchema: `{
				"type": "object",
				"properties": {
					"variantOptions": {
						"$ref": "#/$defs/VariantOptions"
					},
					"contact": {
						"anyOf": [
							{
								"type": "string",
								"description": "contact branch desc",
								"title": "contact branch title"
							},
							{
								"type": "object",
								"description": "contact obj branch desc"
							}
						]
					},
					"profile": {
						"$ref": "#/$defs/Profile"
					},
					"chained": {
						"$ref": "#/$defs/TripleNested"
					}
				},
				"$defs": {
					"VariantOptions": {
						"type": "array",
						"description": "defs variant inner desc",
						"title": "defs variant inner title"
					},
					"Profile": {
						"anyOf": [
							{
								"type": "string",
								"description": "profile branch desc",
								"minLength": 1
							},
							{ "type": "number" }
						]
					},
					"Simple": { "type": "string" },
					"Nested": {
						"$ref": "#/$defs/Simple",
						"description": "chain level 1 desc",
						"title": "chain level 1 title"
					},
					"DoubleNested": {
						"$ref": "#/$defs/Nested"
					},
					"TripleNested": {
						"$ref": "#/$defs/DoubleNested"
					}
				}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := ParseSchema(tt.invalidSchema)
			if err != nil {
				t.Fatalf("Failed to parse schema: %v", err)
			}

			result, _ := schema.Canonical()
			fixedSchema, err := ParseSchema(result)
			if err != nil {
				t.Errorf("Fixed schema is not valid JSON: %v", err)
				return
			}

			expectedSchema, err := ParseSchema(tt.simplifiedSchema)
			if err != nil {
				t.Errorf("Failed to parse simplified schema: %v", err)
				return
			}

			if !reflect.DeepEqual(expectedSchema, fixedSchema) {
				t.Errorf("Expected simplified schema: %s, but got: %s", expectedSchema, fixedSchema)
			}

			validator := newSchemaValidator(WithValidateLevel(ValidateLevelStrict))
			if err := validator.Validate(fixedSchema); err != nil {
				t.Errorf("Fixed schema should pass validation but failed: %v", err)
			}
		})
	}

	t.Run("exceeding_max_attempts_returns_empty_schema", func(t *testing.T) {
		schema, err := ParseSchema(`{
			"type": "object",
			"properties": {
				"foo": { "$ref": "#/$defs/TripleNested" }
			},
			"$defs": {
				"Simple": { "type": "string" },
				"Nested": {
					"$ref": "#/$defs/Simple",
					"description": "level 1 description"
				},
				"DoubleNested": {
					"$ref": "#/$defs/Nested",
					"description": "level 2 description"
				},
				"TripleNested": {
					"$ref": "#/$defs/DoubleNested",
					"description": "level 3 description"
				}
			}
		}`)
		if err != nil {
			t.Fatalf("Failed to parse schema: %v", err)
		}

		validator := newSchemaValidator(WithValidateLevel(ValidateLevelUltra))
		result, rawErr := validator.CanonicalWithMaxAttempts(schema, 1)
		if rawErr == nil {
			t.Fatal("expected validation error")
		}
		if result != "{}" {
			t.Errorf("Expected {}, but got: %s", result)
		}
	})
}

func TestCanonicalRemovesSchemaKeywordOnly(t *testing.T) {
	schema, err := ParseSchema(`{
		"type": "object",
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"properties": {
			"command": {
				"type": "string",
				"$schema": "https://json-schema.org/draft/2020-12/schema"
			}
		},
		"required": ["command"]
	}`)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	result, warnErr := schema.Canonical()
	if warnErr == nil {
		t.Fatal("expected warning after simplifying $schema")
	}
	if result == "{}" {
		t.Fatalf("expected schema to be preserved, got empty object")
	}

	fixed, err := ParseSchema(result)
	if err != nil {
		t.Fatalf("Fixed schema is not valid JSON: %v", err)
	}
	if _, ok := fixed["$schema"]; ok {
		t.Error("root $schema should be removed")
	}
	if fixed["type"] != "object" {
		t.Errorf("type should remain, got %v", fixed["type"])
	}
	props, ok := fixed["properties"].(map[string]any)
	if !ok || props["command"] == nil {
		t.Error("properties.command should remain")
	}
	command, ok := props["command"].(map[string]any)
	if !ok {
		t.Fatalf("properties.command should be a schema, got %T", props["command"])
	}
	if _, ok := command["$schema"]; ok {
		t.Error("nested $schema should be removed")
	}
	if command["type"] != "string" {
		t.Errorf("nested type should remain, got %v", command["type"])
	}
	required, ok := fixed["required"].([]any)
	if !ok || len(required) != 1 || required[0] != "command" {
		t.Errorf("required should remain, got %v", fixed["required"])
	}
}

// Unsupported keywords are dropped individually rather than collapsing the whole
// schema: the enforcer skips keys it does not recognise, so dropping the key alone
// yields the same constrained decoding while keeping every sibling constraint.
func TestCanonicalRemovesUnsupportedKeywordsInsteadOfWiping(t *testing.T) {
	schema, err := ParseSchema(`{
		"type": "object",
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"zzz": true,
		"properties": {
			"command": { "type": "string" }
		},
		"required": ["command"]
	}`)
	if err != nil {
		t.Fatalf("Failed to parse schema: %v", err)
	}

	result, warnErr := schema.Canonical()
	if warnErr == nil {
		t.Fatal("expected warning for unsupported keywords")
	}
	if !strings.Contains(warnErr.Error(), "unsupported keywords: $schema, zzz") {
		t.Fatalf("expected warning to include both unsupported keywords, got %v", warnErr)
	}
	if result == "{}" {
		t.Fatal("expected schema to be preserved, got empty object")
	}

	fixed, err := ParseSchema(result)
	if err != nil {
		t.Fatalf("Fixed schema is not valid JSON: %v", err)
	}
	for _, k := range []string{"$schema", "zzz"} {
		if _, ok := fixed[k]; ok {
			t.Errorf("unsupported keyword %q should be removed", k)
		}
	}
	if fixed["type"] != "object" {
		t.Errorf("type should remain, got %v", fixed["type"])
	}
	props, ok := fixed["properties"].(map[string]any)
	if !ok || props["command"] == nil {
		t.Error("properties.command should remain")
	}
	required, ok := fixed["required"].([]any)
	if !ok || len(required) != 1 || required[0] != "command" {
		t.Errorf("required should remain, got %v", fixed["required"])
	}
}

// Canonical must degrade the offending subschema, never the whole document.
// Each case below used to collapse to "{}" because its error carried
// SimplifyDefault, which is a no-op: the schema never changed, the same error
// resurfaced every round, and the retry loop fell through to the empty fallback.
func TestCanonicalDegradesLocallyNotWholeSchema(t *testing.T) {
	cases := []struct {
		name string
		in   string
		// keep: JSON fragments that must survive; gone: fragments that must not
		keep []string
		gone []string
	}{
		{
			name: "unsupported keyword on a property",
			in:   `{"type":"object","properties":{"a":{"type":"string","format":"uuid"}}}`,
			keep: []string{`"type":"string"`},
			gone: []string{"format"},
		},
		{
			name: "unsupported keyword at root",
			in:   `{"type":"object","properties":{"a":{"type":"string"}},"required":["a"],"allOf":[{"title":"x"}]}`,
			keep: []string{`"required":["a"]`, `"type":"string"`},
			gone: []string{"allOf"},
		},
		{
			// The sibling constraint is folded into the definition instead of being
			// deleted, so nothing the schema asked for is lost.
			name: "sibling constraint next to $ref is merged in",
			in:   `{"type":"object","properties":{"t":{"maxLength":9,"$ref":"#/$defs/S"}},"$defs":{"S":{"type":"string","minLength":1}}}`,
			keep: []string{`"maxLength":9`, `"minLength":1`, `"type":"string"`},
			gone: []string{`"$ref"`},
		},
		{
			name: "$defs outside root",
			in:   `{"type":"object","properties":{"a":{"type":"object","$defs":{"X":{"type":"string"}},"properties":{"b":{"type":"string"}}}}}`,
			keep: []string{`"b":{"type":"string"}`},
			gone: []string{"$defs"},
		},
		{
			// The empty string is a property name like any other, and the enforcer
			// generates it, so nothing about this node needs rewriting.
			name: "empty property name",
			in:   `{"type":"object","properties":{"":{"type":"string"},"a":{"type":"number"}}}`,
			keep: []string{`"":{"type":"string"}`, `"a":{"type":"number"}`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schema, err := ParseSchema(tc.in)
			if err != nil {
				t.Fatalf("failed to parse schema: %v", err)
			}

			result, _ := schema.Canonical()
			if result == "{}" {
				t.Fatalf("schema collapsed to {}; expected only the offending part to be dropped")
			}
			for _, frag := range tc.keep {
				if !strings.Contains(result, frag) {
					t.Errorf("expected %s to survive, got %s", frag, result)
				}
			}
			for _, frag := range tc.gone {
				if strings.Contains(result, frag) {
					t.Errorf("expected %s to be dropped, got %s", frag, result)
				}
			}
		})
	}
}

// A keyword next to $ref is an independent assertion, so the effective
// constraint is the conjunction of both sides. Canonical folds the definition
// into the use site and keeps whichever value admits fewer instances, rather
// than deleting the sibling and silently accepting whatever the definition says.
func TestCanonicalMergesRefSiblingsKeepingStricterValue(t *testing.T) {
	cases := []struct {
		name string
		in   string
		keep []string
		gone []string
	}{
		{
			name: "lower bound keeps the larger value",
			in:   `{"type":"object","properties":{"t":{"minLength":5,"$ref":"#/$defs/S"}},"$defs":{"S":{"type":"string","minLength":1}}}`,
			keep: []string{`"minLength":5`},
			gone: []string{`"minLength":1`},
		},
		{
			name: "lower bound keeps the larger value when the definition is stricter",
			in:   `{"type":"object","properties":{"t":{"minLength":1,"$ref":"#/$defs/S"}},"$defs":{"S":{"type":"string","minLength":5}}}`,
			keep: []string{`"minLength":5`},
			gone: []string{`"minLength":1`},
		},
		{
			name: "upper bound keeps the smaller value",
			in:   `{"type":"object","properties":{"t":{"maxLength":9,"$ref":"#/$defs/S"}},"$defs":{"S":{"type":"string","maxLength":3}}}`,
			keep: []string{`"maxLength":3`},
			gone: []string{`"maxLength":9`},
		},
		{
			name: "a constraint the definition lacks is preserved",
			in:   `{"type":"object","properties":{"t":{"maxLength":9,"$ref":"#/$defs/S"}},"$defs":{"S":{"type":"string","minLength":1}}}`,
			keep: []string{`"maxLength":9`, `"minLength":1`, `"type":"string"`},
		},
		{
			name: "type narrows to the intersection",
			in:   `{"type":"object","properties":{"t":{"type":"integer","minimum":3,"$ref":"#/$defs/S"}},"$defs":{"S":{"type":"number"}}}`,
			keep: []string{`"type":"integer"`, `"minimum":3`},
			gone: []string{`"type":"number"`},
		},
		{
			name: "enum narrows to the intersection",
			in:   `{"type":"object","properties":{"t":{"enum":["a","b"],"minLength":1,"$ref":"#/$defs/S"}},"$defs":{"S":{"type":"string","enum":["b","c"]}}}`,
			keep: []string{`"enum":["b"]`},
		},
		{
			name: "required takes the union",
			in:   `{"type":"object","properties":{"t":{"required":["x"],"minLength":1,"$ref":"#/$defs/S"}},"$defs":{"S":{"type":"object","properties":{"x":{"type":"string"},"y":{"type":"string"}},"required":["y"]}}}`,
			keep: []string{`"x"`, `"y"`},
		},
		{
			// Two uses of one definition get their own merged copy, so neither
			// bound leaks into the other.
			name: "each use site is specialised independently",
			in:   `{"type":"object","properties":{"a":{"minLength":5,"$ref":"#/$defs/S"},"b":{"minLength":9,"$ref":"#/$defs/S"}},"$defs":{"S":{"type":"string","minLength":1}}}`,
			keep: []string{`"a":{"minLength":5`, `"b":{"minLength":9`},
		},
		{
			// Inlining a recursive definition would not terminate, so the sibling
			// is dropped as before and the reference stays.
			name: "recursive definition falls back to dropping the sibling",
			in:   `{"type":"object","properties":{"t":{"minLength":5,"$ref":"#/$defs/N"}},"$defs":{"N":{"type":"object","properties":{"next":{"$ref":"#/$defs/N"}}}}}`,
			keep: []string{`"$ref":"#/$defs/N"`},
			gone: []string{`"minLength"`},
		},
		{
			// Nothing would be lost by dropping the duplicate, so the reference is
			// left in place and stays shared.
			name: "identical values keep the reference",
			in:   `{"type":"object","properties":{"t":{"minLength":1,"$ref":"#/$defs/S"}},"$defs":{"S":{"type":"string","minLength":1}}}`,
			keep: []string{`"$ref":"#/$defs/S"`},
		},
		{
			name: "annotation-only siblings keep the reference",
			in:   `{"type":"object","properties":{"t":{"description":"outer","$ref":"#/$defs/S"}},"$defs":{"S":{"type":"string","description":"inner"}}}`,
			keep: []string{`"$ref":"#/$defs/S"`},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schema, err := ParseSchema(tc.in)
			if err != nil {
				t.Fatalf("failed to parse schema: %v", err)
			}

			result, _ := schema.Canonical()
			if result == "{}" {
				t.Fatalf("schema collapsed to {}")
			}
			for _, frag := range tc.keep {
				if !strings.Contains(result, frag) {
					t.Errorf("expected %s in result, got %s", frag, result)
				}
			}
			for _, frag := range tc.gone {
				if strings.Contains(result, frag) {
					t.Errorf("expected %s to be gone, got %s", frag, result)
				}
			}
		})
	}
}

// An unsatisfiable overlap must not be simplified by deleting one of the two
// sides: whichever side survives, the node ends up accepting exactly what the
// original rejected. Only the contradicting keyword goes, so the node keeps
// whatever the two sides still agree on and is left unconstrained in that one
// dimension.
//
// Reporting order matters here: the rules that delete $ref siblings to
// canonicalise a node used to run first, and once the contradicting sibling was
// gone nothing was left to notice the contradiction.
func TestCanonicalDoesNotSimplifyAwayUnsatisfiableRefSiblings(t *testing.T) {
	cases := []struct {
		name string
		in   string
		// wantProperty is the complete canonical form of the contradicting
		// property, so an inverted result cannot slip past
		wantProperty string
	}{
		{
			// Nothing survives an empty type intersection, so the property is left
			// fully unconstrained rather than typed as number.
			name:         "contradicting type",
			in:           `{"type":"object","properties":{"t":{"type":"string","$ref":"#/$defs/S"}},"$defs":{"S":{"type":"number"}}}`,
			wantProperty: `"t":{}`,
		},
		{
			// Constraints either side carries cannot be salvaged once the types
			// disagree: a string length and a numeric minimum in one typeless node
			// describe nothing at all.
			name:         "contradicting type discards both sides' constraints",
			in:           `{"type":"object","properties":{"t":{"type":"string","minLength":3,"$ref":"#/$defs/S"}},"$defs":{"S":{"type":"number","minimum":5}}}`,
			wantProperty: `"t":{}`,
		},
		{
			// Both sides agree the value is a string and only disagree on which
			// strings, so the type is kept and only the enum is dropped. Keeping
			// either enum would accept a value the other side ruled out.
			name:         "contradicting enum keeps the agreed type",
			in:           `{"type":"object","properties":{"t":{"type":"string","enum":["x"],"$ref":"#/$defs/S"}},"$defs":{"S":{"type":"string","enum":["y"]}}}`,
			wantProperty: `"t":{"type":"string"}`,
		},
		{
			// Dropping the enum must not drop everything else the two sides agreed
			// on, and what survives still has to be the stricter of the two bounds.
			name:         "contradicting enum keeps the stricter bound",
			in:           `{"type":"object","properties":{"t":{"type":"string","enum":["x"],"minLength":2,"$ref":"#/$defs/S"}},"$defs":{"S":{"type":"string","enum":["y"],"minLength":9}}}`,
			wantProperty: `"t":{"minLength":9,"type":"string"}`,
		},
		{
			// A recursive definition cannot be inlined, so there is no merged form
			// to keep and the node is emptied instead of adopting the object type.
			name:         "contradicting type against a recursive definition",
			in:           `{"type":"object","properties":{"t":{"type":"string","$ref":"#/$defs/S"}},"$defs":{"S":{"type":"object","properties":{"next":{"$ref":"#/$defs/S"}}}}}`,
			wantProperty: `"t":{}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schema, err := ParseSchema(tc.in)
			if err != nil {
				t.Fatalf("failed to parse schema: %v", err)
			}

			result, warnErr := schema.Canonical()
			if warnErr == nil {
				t.Fatal("expected a warning about the empty intersection")
			}
			if !strings.Contains(warnErr.Error(), "intersection is empty") {
				t.Fatalf("expected an empty-intersection warning, got %v", warnErr)
			}
			if !strings.Contains(result, tc.wantProperty) {
				t.Fatalf("expected the property to become %s, got %s", tc.wantProperty, result)
			}
		})
	}
}

// Both sides describing the same object must end up with both sets of
// properties, and a property both sides describe keeps both sides' constraints.
// Letting the use site win outright would quietly drop whatever the definition
// asserted about that property.
func TestCanonicalMergesPropertiesFromBothSides(t *testing.T) {
	const schema = `{
		"type":"object",
		"properties":{
			"node":{
				"properties":{"shared":{"type":"string"},"onlyHere":{"type":"boolean"}},
				"$ref":"#/$defs/S"
			}
		},
		"$defs":{
			"S":{
				"type":"object",
				"properties":{"shared":{"type":"string","minLength":10},"onlyThere":{"type":"integer"}}
			}
		}
	}`

	parsed, err := ParseSchema(schema)
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}
	result, _ := parsed.Canonical()

	var out map[string]any
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("canonical output is not valid JSON: %v", err)
	}

	properties := out["properties"].(map[string]any)["node"].(map[string]any)["properties"]
	merged, ok := properties.(map[string]any)
	if !ok {
		t.Fatalf("expected merged properties, got %s", result)
	}

	for _, name := range []string{"shared", "onlyHere", "onlyThere"} {
		if _, exists := merged[name]; !exists {
			t.Errorf("expected property %q to survive the merge, got %s", name, result)
		}
	}

	shared, ok := merged["shared"].(map[string]any)
	if !ok {
		t.Fatalf("expected shared to be a schema, got %s", result)
	}
	if shared["minLength"] != float64(10) {
		t.Errorf("expected the definition's minLength to survive on shared, got %s", result)
	}
	if shared["type"] != "string" {
		t.Errorf("expected the use site's type to survive on shared, got %s", result)
	}
}

// type is intersected rather than replaced, and integer wins an integer/number
// overlap because every integer is a number but not the reverse.
func TestCanonicalIntersectsTypesWhenMerging(t *testing.T) {
	cases := []struct {
		name       string
		siblings   string
		target     string
		expectType string
	}{
		{
			name:       "integer against number keeps integer",
			siblings:   `"type":"number","minimum":5`,
			target:     `"type":"integer"`,
			expectType: `"type":"integer"`,
		},
		{
			name:       "number against integer keeps integer",
			siblings:   `"type":"integer","minimum":5`,
			target:     `"type":"number"`,
			expectType: `"type":"integer"`,
		},
		{
			name:       "overlapping unions keep the shared members",
			siblings:   `"type":["string","integer","boolean"],"description":"d"`,
			target:     `"type":["string","boolean"]`,
			expectType: `"type":["boolean","string"]`,
		},
		{
			name:       "union narrowed to a single type collapses to a string",
			siblings:   `"type":["string","integer"],"minLength":2`,
			target:     `"type":["string"]`,
			expectType: `"type":"string"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			schema := fmt.Sprintf(
				`{"type":"object","properties":{"v":{%s,"$ref":"#/$defs/S"}},"$defs":{"S":{%s}}}`,
				tc.siblings, tc.target,
			)
			parsed, err := ParseSchema(schema)
			if err != nil {
				t.Fatalf("failed to parse schema: %v", err)
			}
			result, _ := parsed.Canonical()
			if !strings.Contains(result, tc.expectType) {
				t.Errorf("expected %s in the merged node, got %s", tc.expectType, result)
			}
		})
	}
}

// Inlining $ref siblings copies the referenced definition at every use site, so
// a chain of definitions referenced twice per level would double the output per
// level -- ~900KB here for a 5KB input. The copy budget stops the inlining
// before the size limit can, the siblings that could not be folded in are
// dropped in one pass, and Canonical returns the loosened schema with a warning
// that names what was lost -- not a {} from the size check.
func TestCanonicalStopsInliningBeyondCopyBudget(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`{"type":"object","properties":{"root":{"$ref":"#/$defs/D1","description":"use site"}},"$defs":{`)
	const levels = 14
	for i := 1; i < levels; i++ {
		if i > 1 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `"D%d":{"type":"object","properties":{"p":{"$ref":"#/$defs/D%d","additionalProperties":false},"q":{"$ref":"#/$defs/D%d","additionalProperties":false}}}`, i, i+1, i+1)
	}
	fmt.Fprintf(&sb, `,"D%d":{"type":"object","properties":{"x":{"type":"string"}}}}}`, levels)

	schema, err := ParseSchema(sb.String())
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	out, canonicalErr := schema.Canonical()
	if canonicalErr == nil {
		t.Fatal("expected a warning listing the dropped constraints")
	}
	if !strings.Contains(canonicalErr.Error(), "additionalProperties") {
		t.Fatalf("warning should name the dropped keyword, got: %v", canonicalErr)
	}
	if out == "{}" {
		t.Fatal("expected a loosened schema, got an empty one")
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("canonical output is not valid JSON: %v", err)
	}
	// Constraints folded in before the budget ran out survive inlined; the ones
	// that did not fit are gone from their $ref nodes.
	if !strings.Contains(out, `"$ref"`) {
		t.Fatalf("references should survive the drop: %s", out)
	}
	if !strings.Contains(out, `"additionalProperties":false`) {
		t.Fatalf("constraints inlined before the budget ran out should survive: %s", out)
	}
}

// Schemas well under the copy budget must still inline fully.
func TestCanonicalInliningBelowCopyBudgetUnchanged(t *testing.T) {
	raw := `{
		"type": "object",
		"properties": {
			"short": {"maxLength": 10, "$ref": "#/$defs/S"},
			"long": {"maxLength": 99, "$ref": "#/$defs/S"}
		},
		"$defs": {"S": {"type": "string", "minLength": 1}}
	}`

	schema, err := ParseSchema(raw)
	if err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}
	out, err := schema.Canonical()
	if err != nil {
		t.Fatalf("canonical failed: %v", err)
	}
	for _, want := range []string{`"maxLength":10`, `"maxLength":99`, `"minLength":1`} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %s in canonical output, got %s", want, out)
		}
	}
}
