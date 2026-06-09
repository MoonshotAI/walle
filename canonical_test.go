package walle

import (
	"reflect"
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
			name: "invalid_type_with_ref",
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
					"user": {
						"$ref": "#"
					}
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
				}
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
			simplifiedSchema: `{
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
			simplifiedSchema: `{}`,
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
					"name": {}
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
			name: "mixed_structural_conflict_still_clears_parent_sub_schema",
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
					"name": {}
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
