package walle

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBasicTypes(t *testing.T) {
	validCases := []string{
		// Empty schema
		`{}`,
		// Multiple types
		`{"type": ["string", "null", "integer", "boolean", "array", "object", "number"]}`,
		// String type
		`{"type": "string"}`,
		// String with constraints
		`{
			"type": "string",
			"minLength": 1,
			"maxLength": 100,
			"description": "A string",
			"default": "hello",
			"pattern": "^[a-z]+$"
		}`,
		// Number type
		`{
			"type": "number",
			"minimum": -10,
			"maximum": 10,
			"enum": [1, 2.5, -3.7],
			"description": "A number",
			"default": 1.5
		}`,
		// Integer type
		`{
			"type": "integer",
			"minimum": -10,
			"maximum": 10,
			"enum": [1, 2, 3],
			"description": "An integer",
			"default": 1
		}`,
		// Boolean type
		`{
			"type": "boolean",
			"description": "A boolean",
			"default": true
		}`,
		// Null type
		`{
			"type": "null",
			"description": "A null value",
			"default": null
		}`,
		// Array type
		`{
			"type": "array",
			"items": {"type": "string"},
			"minItems": 0,
			"maxItems": 10,
			"description": "An array",
			"default": []
		}`,
		// Object type
		`{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "integer"}
			},
			"required": ["name"],
			"additionalProperties": false,
			"description": "An object",
			"default": {"name": "John", "age": 30}
		}`,
		// Using "type" as property name
		`{
			"type": "object",
			"properties": {
				"type": {"type": "string"}
			}
		}`,
		// Nested objects
		`{
			"type": "object",
			"properties": {
				"user": {
					"type": "object",
					"properties": {
						"name": {"type": "string"},
						"address": {
							"type": "object",
							"properties": {
								"street": {"type": "string"},
								"city": {"type": "string"}
							}
						}
					}
				}
			}
		}`,
		// Nested arrays
		`{
			"type": "array",
			"items": {
				"type": "array",
				"items": {
					"type": "string",
					"default": "hello"
				}
			}
		}`,
		// Mixed structure
		`{
			"type": "object",
			"properties": {
				"tags": {
					"type": "array",
					"items": {"type": "string"}
				},
				"metadata": {
					"type": "object",
					"properties": {
						"created": {"type": "string", "default": "2021-01-01"},
						"modified": {"type": "string", "default": "2021-01-01"}
					}
				}
			}
		}`,
		// key name with dot
		`{
			"type": "object",
			"properties": {
				"a.b.c": {"type": "string"},
				"d.e.f": {
					"anyOf": [
						{"type": "string"},
						{"type": "object",
							"properties": {
								"g.f": {"type": "string"}
							}
						}
					]
				}
			}
		}`,
		// key name with slash
		`{
			"type": "object",
			"properties": {
				"a/b/c": {"type": "string"},
				"d/e/f": {
					"anyOf": [
						{"type": "string"},
						{"type": "object",
							"properties": {
								"g/f": {"type": "string"}
							}
						}
					]
				}
			}
		}`,
		`{
			"type": "object",
			"properties": {
				"a/b/c": {"type": "string"},
				"d/e/f": {
					"anyOf": [
						{"type": "string"},
						{"type": "object",
							"properties": {
								"g/f": {"type": "string"}
							}
						}
					]
				}
			},
			"$defs": {
				"User": {
					"type": "object",
					"properties": {
						"name": {"type": "string"}
					}
				}
			}
		}`,
		// property name with enum
		`{
			"type": "object",
			"properties": {
				"enum": {
					"type": "string"
				}
			}
		}`,
		`{
			"type": "object",
			"properties": {
				"items": {
					"type": "string"
				}
			}
		}`,
		`{
			"type": "object",
			"properties": {
				"defs": {
					"type": "string"
				}
			}
		}`,
		`{
			"type": "object",
			"properties": {
				"type": {
					"type": "string"
				}
			}
		}`,
		`{
			"type": "string",
			"description": "A string type",
			"title": "String Type"
		}`,
		`{
			"type": "string",
			"enum": ["", " ", "\n", "\t", "special\\chars"]
		}`,
		`{
			"type": "number",
			"enum": [0, 1, -1, 3.14, 2.71828]
		}`,
		`{
			"type": "boolean",
			"enum": [true, false]
		}`,
		`{
			"type": ["string", "null"],
			"enum": ["value", null]
		}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		// Empty type array
		{
			`{"type": []}`,
			"type array cannot be empty",
			false,
		},
		// Nil schema
		{
			"null",
			"schema must be a dict",
			false,
		},
		// Non-dict schema
		{
			`["type", "string"]`,
			"JSON schema parsing error",
			true,
		},
		// Invalid type
		{
			`{"type": "invalid"}`,
			"invalid type",
			false,
		},
		{
			`{
				"type": null
			}`,
			"type must be string or array of strings",
			false,
		},
		{
			`{
				"type": ["string", ["number"]]
			}`,
			"invalid type in type array",
			false,
		},
		{
			`{
				"type": ["string", "null"],
				"enum": ["value", null],
				"properties": {}
			}`,
			"invalid keywords",
			false,
		},
		{
			`{
				"type": {}
			}`,
			"type must be string or array of strings",
			false,
		},
		{
			`{"type": 123}`,
			"type must be string or array of strings",
			false,
		},
		{
			`{
				"type": ["string", "invalid-type"]
			}`,
			"invalid type",
			false,
		},
		{
			`{"type": [123, "string"]}`,
			"invalid type",
			false,
		},
		// String instead of dict
		{
			`""`,
			"JSON schema parsing error",
			true,
		},
		// Object with missing type in properties
		// {
		// 	`{
		// 		"type": "object",
		// 		"properties": {
		// 			"key1": {"type": "string"},
		// 			"key2": {"type": "string"},
		// 			"builtin": {}
		// 		},
		// 		"additionalProperties": false
		// 	}`,
		// 	"type need to be defined explicitly",
		// 	false,
		// },
		// Object with missing type in multiple properties
		// {
		// 	`{
		// 		"type": "object",
		// 		"properties": {
		// 			"key1": {},
		// 			"key2": {"type": "string"},
		// 			"builtin": {}
		// 		},
		// 		"additionalProperties": false
		// 	}`,
		// 	"type need to be defined explicitly",
		// 	false,
		// },
		// Object with all properties missing type
		// {
		// 	`{
		// 		"type": "object",
		// 		"properties": {
		// 			"key1": {},
		// 			"key2": {},
		// 			"builtin": {}
		// 		},
		// 		"additionalProperties": false
		// 	}`,
		// 	"type need to be defined explicitly",
		// 	false,
		// },
		// Properties not an object
		{
			`{
				"type": "object",
				"properties": "not an object"
			}`,
			"properties must be an object",
			false,
		},
		// Empty property name
		{
			`{
				"type": "object",
				"properties": {
					"": {"type": "string"}
				}
			}`,
			"property name cannot be empty",
			false,
		},
		// Nested non-string property key
		{
			`{
				"type": "object",
				"properties": {
					"user": {
						"type": "object",
						"properties": {
							true: {"type": "string"}
						}
					}
				}
			}`,
			"JSON schema parsing error",
			true,
		},
		// Non-string property key in anyOf
		{
			`{
				"anyOf": [
					{
						"type": "object",
						"properties": {
							3.14: {"type": "string"}
						}
					}
				]
			}`,
			"JSON schema parsing error",
			true,
		},
		// Non-string property key in $defs
		{
			`{
				"$defs": {
					"User": {
						"type": "object",
						"properties": {
							null: {"type": "string"}
						}
					}
				}
			}`,
			"JSON schema parsing error",
			true,
		},
		// Multiple non-string property keys
		{
			`{
				"type": "object",
				"properties": {
					"valid": {"type": "string"},
					123: {"type": "string"},
					false: {"type": "number"}
				}
			}`,
			"JSON schema parsing error",
			true,
		},
		{
			`{
				"type": "object",
				"properties": {
					"nested": {
						"type": "object",
						"required": ["nonexistent"]
					}
				},
				"required": ["nested"]
			}`,
			"'properties' keyword is missing",
			false,
		},
		{
			`{
				"type": "string",
				"enum": [1, 2, 3]
			}`,
			"does not match any type",
			false,
		},
		{
			`{
				"type": ["string", 123],
				"enum": [1, 2, 3]
			}`,
			"type array",
			false,
		},
		{
			`{
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
			"must be an object",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"name": {
						"$ref": "#/$defs/User",
						"minLength": 10
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
			"minLength",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"name": {
						"$ref": "#/$defs/User",
						"required": ["name"]
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
			"required",
			false,
		},
		// $defs name contain '/' character
		{
			`{
				"type": "object",
				"properties": {
					"a/b/c": {"type": "string"},
					"d/e/f": {
						"anyOf": [
							{"type": "string"},
							{"type": "object",
								"properties": {
									"g/f": {"type": "string"}
								}
							}
						]
					}
				},
				"$defs": {
					"c/d": {
						"type": "object",
						"properties": {
							"name": {"type": "string"}
						}
					}
				}
			}`,
			"contain '/' character",
			false,
		},
		{
			`{
                "type": "object",
                "$defs": {
                    "user": {
                        "type": "object",
                        "properties": {
                            "first/name": {
                                "type": "string"
                            }
                        }
                    }
                }
            }`,
			"contain '/' character",
			false,
		},
		{
			`{
                "type": "object",
                "$defs": {
                    "users": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "user/id": {
                                    "type": "string"
                                }
                            }
                        }
                    }
                }
            }`,
			"contain '/' character",
			false,
		},
		{
			`{
                "type": "object",
                "$defs": {
                    "address": {
                        "type": "object",
                        "properties": {
                            "location": {
                                "type": "object",
                                "properties": {
                                    "city/state": {
                                        "type": "string"
                                    }
                                }
                            }
                        }
                    }
                }
            }`,
			"contain '/' character",
			false,
		},
		{
			`{
                "type": "object",
                "$defs": {
                    "contact": {
                        "anyOf": [
                            {
                                "type": "object",
                                "properties": {
                                    "phone/number": {
                                        "type": "string"
                                    }
                                }
                            }
                        ]
                    }
                }
            }`,
			"contain '/' character",
			false,
		},
		{
			`{
                "type": "object",
                "$defs": {
                    "name": {
                        "type": "string"
                    },
                    "email/address": {
                        "type": "string"
                    }
                }
            }`,
			"contain '/' character",
			false,
		},
		{
			`{
				"type": ["array", "boolean"],
				"items": {
					"type": "string"
				},
				"enum": [true, [1, 2, 3]],
				"minItems": 2,
				"maxItems": 10
			}`,
			"is not allowed in combination",
			false,
		},
		{
			`{
				"type": ["array", "object"],
				"items": {
					"type": "number"
				},
				"properties": {
					"name": {"type": "string"}
				},
				"required": ["name"],
				"minItems": 3,
				"additionalProperties": false
			}`,
			"is not allowed in combination",
			false,
		},
		{
			`{
				"type": ["string", "integer"],
				"maxItems": 5,
				"minLength": 10,
				"maximum": 100
			}`,
			"is not allowed in combination",
			false,
		},
		{
			`{
				"properties": {
					"acronym": {
						"type": "string"
					},
					"code": {
						"type": "string"
					},
					"enddate": {
						"type": "string"
					},
					"funder": {
						"properties": {
							"$ref": {
								"type": "string"
							}
						},
						"type": "object"
					},
					"identifiers": {
						"properties": {
							"eurepo": {
								"anyOf": [
									{
										"type": "string"
									},
									{
										"type": "null"
									}
								]
							},
							"oaf": {
								"anyOf": [
									{
										"type": "string"
									},
									{
										"type": "null"
									}
								]
							},
							"purl": {
								"anyOf": [
									{
										"type": "string"
									},
									{
										"type": "null"
									}
								]
							}
						},
						"type": "object"
					},
					"internal_id": {
						"type": "string"
					},
					"program": {
						"type": "string"
					},
					"remote_modified": {
						"anyOf": [
							{
								"type": "string"
							},
							{
								"type": "null"
							}
						]
					},
					"startdate": {
						"type": "string"
					},
					"title": {
						"type": "string"
					},
					"url": {
						"type": "string"
					}
				},
				"type": "object"
			}`,
			"reserved for JSON Schema",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"$ref": {
						"type": "string"
					}
				}
			}`,
			"reserved for JSON Schema",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"anyOf": {
						"type": "string"
					}
				}
			}`,
			"reserved for JSON Schema",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"required": {
						"type": "string"
					}
				}
			}`,
			"reserved for JSON Schema",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"additionalProperties": {
						"type": "string"
					}
				}
			}`,
			"reserved for JSON Schema",
			false,
		},
		{
			`{
				"type": "string",
				"enum": [true, null, false]
			}`,
			"does not match any type",
			false,
		},
		{
			`{
				"type": "string",
				"enum": []
			}`,
			"enum array cannot be empty",
			false,
		},
		{
			`{
				"type": ["string", "object"],
				"enum": ["value1", {}]
			}`,
			"not allowed in combination with enum",
			false,
		},
		{
			`{
				"type": "string",
				"enum": {}
			}`,
			"enum must be an array",
			false,
		},
		{
			`{
				"type": "string",
				"enum": 133
			}`,
			"enum must be an array",
			false,
		},
		{
			`{
				"type": "string",
				"enum": "value1"
			}`,
			"enum must be an array",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"name": {
						"type": ["string", "object"],
						"$id": "https://example.com/name"
					}
				}
			}`,
			"allowed in combination with multiple types",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"$id": "https://example.com/name"
					}
				}
			}`,
			"must be at root level",
			false,
		},
		// Array with invalid items
		{
			`{
				"type": "array",
				"items": 123
			}`,
			"items must be an object",
			false,
		},
		{
			`{
				"type": "array",
				"items": []
			}`,
			"items must be an object",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestSingleTypeInArray(t *testing.T) {
	validCases := []string{
		`{
			"type": ["object"],
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "integer"}
			},
			"required": ["name"]
		}`,
		`{
			"type": ["array"],
			"items": {"type": "string"},
			"minItems": 0,
			"maxItems": 10
		}`,
		`{
			"type": ["string"],
			"minLength": 1,
			"maxLength": 100
		}`,
		`{
			"type": ["number"],
			"minimum": -10,
			"maximum": 10
		}`,
		`{
			"type": ["integer"],
			"minimum": -10,
			"maximum": 10
		}`,
		`{
			"type": ["boolean"]
		}`,
		`{
			"type": ["null"]
		}`,
		`{
			"type": ["object"],
			"properties": {
				"user": {
					"type": ["object"],
					"properties": {
						"name": {"type": ["string"]},
						"address": {
							"type": ["object"],
							"properties": {
								"street": {"type": ["string"]},
								"city": {"type": ["string"]}
							}
						}
					}
				}
			}
		}`,
		`{
			"type": ["array"],
			"items": {
				"type": ["array"],
				"items": {
					"type": ["string"],
					"minLength": 1,
					"enum": ["a", "b"]
				}
			}
		}`,
		`{
			"type": ["object"],
			"properties": {
				"tags": {
					"type": ["array"],
					"items": {"type": ["string"]}
				},
				"metadata": {
					"type": ["object"],
					"properties": {
						"created": {"type": ["string"]},
						"modified": {"type": ["string"]}
					}
				}
			}
		}`,
		`{
			"type": ["object"],
			"properties": {
				"data": {
					"anyOf": [
						{"type": ["string"]},
						{"type": ["integer"]},
						{
							"type": ["object"],
							"properties": {
								"value": {"type": ["string"]}
							}
						}
					]
				},
				"reference": {
					"$ref": "#/$defs/TypeAsList"
				}
			},
			"$defs": {
				"TypeAsList": {
					"type": ["object"],
					"properties": {
						"name": {"type": ["string"]}
					}
				}
			}
		}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		// Empty type array
		{
			`{"type": []}`,
			"type array cannot be empty",
			false,
		},
		// Array with invalid type
		{
			`{"type": ["invalid"]}`,
			"invalid type",
			false,
		},
		// Array with non-string item
		{
			`{"type": [123]}`,
			"invalid type in type array",
			false,
		},
		// Object type in array form with non-object keywords
		{
			`{
				"type": ["object"],
				"minLength": 5
			}`,
			"invalid keywords",
			false,
		},
		// Array type in array form with non-array keywords
		{
			`{
				"type": ["array"],
				"minimum": 0
			}`,
			"invalid keywords",
			false,
		},
		// String type in array form with non-string keywords
		{
			`{
				"type": ["string"],
				"minItems": 1
			}`,
			"invalid keywords",
			false,
		},
		// Object type without explicit property type definition
		// {
		// 	`{
		// 		"type": ["object"],
		// 		"properties": {
		// 			"name": {}
		// 		}
		// 	}`,
		// 	"type need to be defined explicitly",
		// 	false,
		// },
	}

	runTestCases(t, validCases, invalidCases)
}

func TestAdditionalProperties(t *testing.T) {
	validCases := []string{
		`{
			"type": "object",
			"additionalProperties": true
		}`,
		`{
			"type": "object",
			"additionalProperties": false
		}`,
		`{
			"type": "object",
			"additionalProperties": {
				"type": "string"
			}
		}`,
		`{
			"type": "object",
			"additionalProperties": {
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}
		}`,
		`{
			"type": "object",
			"properties": {
				"name": {"type": "string"}
			},
			"additionalProperties": false
		}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		// number type
		{
			`{
				"type": "object",
				"additionalProperties": 123
			}`,
			"additionalProperties must be a boolean or an object",
			false,
		},
		{
			// string type
			`{
				"type": "object",
				"additionalProperties": "invalid"
			}`,
			"additionalProperties must be a boolean or an object",
			false,
		},
		{
			// array type
			`{
				"type": "object",
				"additionalProperties": ["invalid"]
			}`,
			"additionalProperties must be a boolean or an object",
			false,
		},
		{
			// null value
			`{
				"type": "object",
				"additionalProperties": null
			}`,
			"additionalProperties must be a boolean or an object",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestRequired(t *testing.T) {
	validCases := []string{
		// empty array required
		`{
			"type": "object",
			"properties": {
				"name": {"type": "string"}
			},
			"required": []
		}`,
		// single required property
		`{
			"type": "object",
			"properties": {
				"name": {"type": "string"}
			},
			"required": ["name"]
		}`,
		// multiple required properties
		`{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "integer"},
				"email": {"type": "string"}
			},
			"required": ["name", "age", "email"]
		}`,
		// required in nested object
		`{
			"type": "object",
			"properties": {
				"user": {
					"type": "object",
					"properties": {
						"id": {"type": "string"},
						"info": {"type": "string"}
					},
					"required": ["id"]
				}
			}
		}`,
		`{
			"$defs": {
				"FlightInfo": {
					"properties": {
						"flight_number": {
							"description": "Flight number, such as 'HAT001'.",
							"title": "Flight Number",
							"type": "string"
						},
						"date": {
							"description": "The date for the flight in the format 'YYYY-MM-DD', such as '2024-05-01'.",
							"title": "Date",
							"type": "string"
						}
					},
					"required": [
						"flight_number",
						"date"
					],
					"title": "FlightInfo",
					"type": "object"
				},
				"Passenger": {
					"properties": {
						"first_name": {
							"description": "Passenger's first name",
							"title": "First Name",
							"type": "string"
						},
						"last_name": {
							"description": "Passenger's last name",
							"title": "Last Name",
							"type": "string"
						},
						"dob": {
							"description": "Date of birth in YYYY-MM-DD format",
							"title": "Dob",
							"type": "string"
						}
					},
					"required": [
						"first_name",
						"last_name",
						"dob"
					],
					"title": "Passenger",
					"type": "object"
				},
				"Payment": {
					"properties": {
						"payment_id": {
							"description": "Unique identifier for the payment",
							"title": "Payment Id",
							"type": "string"
						},
						"amount": {
							"description": "Payment amount in dollars",
							"title": "Amount",
							"type": "integer"
						}
					},
					"required": [
						"payment_id",
						"amount"
					],
					"title": "Payment",
					"type": "object"
				}
			},
			"properties": {
				"user_id": {
					"description": "The ID of the user to book the reservation such as 'sara_doe_496'.",
					"title": "User Id",
					"type": "string"
				},
				"origin": {
					"description": "The IATA code for the origin city such as 'SFO'.",
					"title": "Origin",
					"type": "string"
				},
				"destination": {
					"description": "The IATA code for the destination city such as 'JFK'.",
					"title": "Destination",
					"type": "string"
				},
				"flight_type": {
					"description": "The type of flight such as 'one_way' or 'round_trip'.",
					"enum": [
						"round_trip",
						"one_way"
					],
					"title": "Flight Type",
					"type": "string"
				},
				"cabin": {
					"description": "The cabin class such as 'basic_economy', 'economy', or 'business'.",
					"enum": [
						"business",
						"economy",
						"basic_economy"
					],
					"title": "Cabin",
					"type": "string"
				},
				"flights": {
					"description": "An array of objects containing details about each piece of flight.",
					"items": {
						"anyOf": [
							{
								"$ref": "#/$defs/FlightInfo"
							},
							{
								"type": "object"
							}
						]
					},
					"title": "Flights",
					"type": "array"
				},
				"passengers": {
					"description": "An array of objects containing details about each passenger.",
					"items": {
						"anyOf": [
							{
								"$ref": "#/$defs/Passenger"
							},
							{
								"type": "object"
							}
						]
					},
					"title": "Passengers",
					"type": "array"
				},
				"payment_methods": {
					"description": "An array of objects containing details about each payment method.",
					"items": {
						"anyOf": [
							{
								"$ref": "#/$defs/Payment"
							},
							{
								"type": "object"
							}
						]
					},
					"title": "Payment Methods",
					"type": "array"
				},
				"total_baggages": {
					"description": "The total number of baggage items to book the reservation.",
					"title": "Total Baggages",
					"type": "integer"
				},
				"nonfree_baggages": {
					"description": "The number of non-free baggage items to book the reservation.",
					"title": "Nonfree Baggages",
					"type": "integer"
				},
				"insurance": {
					"description": "Whether the reservation has insurance.",
					"enum": [
						"yes",
						"no"
					],
					"title": "Insurance",
					"type": "string"
				}
			},
			"required": [
				"user_id",
				"origin",
				"destination",
				"flight_type",
				"cabin",
				"flights",
				"passengers",
				"payment_methods",
				"total_baggages",
				"nonfree_baggages",
				"insurance"
			],
			"title": "parameters",
			"type": "object"
		}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		{
			// required is not an array
			`{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"required": "name"
			}`,
			"required must be an array",
			false,
		},
		{
			// required contains non-string
			`{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"required": ["name", 123]
			}`,
			"items in required array must be strings",
			false,
		},
		{
			// required contains empty string
			`{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"required": ["name", ""]
			}`,
			"property names in required array cannot be empty",
			false,
		},
		{
			// required references non-existent property
			`{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"required": ["age"]
			}`,
			"required property 'age' is not defined in properties",
			false,
		},
		{
			// non-object type uses required
			`{
				"type": "string",
				"required": ["value"]
			}`,
			"invalid keywords: required",
			false,
		},
		{
			// object without properties uses required
			`{
				"type": "object",
				"required": ["name"]
			}`,
			"'properties' keyword is missing",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestKeywordsValidation(t *testing.T) {
	validCases := []string{
		// boolean type allowed keywords
		`{
			"anyOf": [
				{"type": "boolean"},
				{"type": "boolean"}
			]
		}`,
		// null type allowed keywords
		`{
			"anyOf": [
				{"type": "null"},
				{"type": "null"}
			]
		}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		// object type with invalid keywords
		{
			`{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"minLength": 1,
				"maxLength": 100
			}`,
			"invalid keywords",
			false,
		},
		// array type with invalid keywords
		{
			`{
				"type": "array",
				"items": {"type": "string"},
				"properties": {
					"test": {"type": "string"}
				}
			}`,
			"invalid keywords",
			false,
		},
		// string type with invalid keywords
		{
			`{
				"type": "string",
				"minItems": 1
			}`,
			"invalid keywords",
			false,
		},
		// number type with invalid keywords
		{
			`{
				"type": "number",
				"minLength": 1,
				"maxLength": 100
			}`,
			"invalid keywords",
			false,
		},
		// boolean type with invalid keywords
		{
			`{
				"type": "boolean",
				"minimum": 0
			}`,
			"invalid keywords",
			false,
		},
		// null type with invalid keywords
		{
			`{
				"type": "null",
				"maxLength": 10
			}`,
			"invalid keywords",
			false,
		},
		// conflicting keywords in anyOf
		{
			`{
				"properties": {
					"name": {"type": "string"}
				},
				"additionalProperties": false,
				"anyOf": [
					{
						"type": "object",
						"properties": {
							"a": {"type": "string"}
						}
					},
					{
						"type": "object",
						"properties": {
							"b": {"type": "string"}
						}
					}
				]
			}`,
			"is not allowed at the same level as anyOf",
			false,
		},
		// conflicting array keywords in anyOf
		{
			`{
				"items": {"type": "string"},
				"minItems": 0,
				"maxItems": 10,
				"anyOf": [
					{
						"type": "array",
						"items": {"type": "string"}
					},
					{
						"type": "array",
						"items": {"type": "number"}
					}
				]
			}`,
			"is not allowed at the same level as anyOf",
			false,
		},
		// conflicting number keywords in anyOf
		{
			`{
				"minimum": 0,
				"maximum": 100,
				"anyOf": [
					{
						"type": "number",
						"minimum": 0
					},
					{
						"type": "number",
						"maximum": 10
					}
				]
			}`,
			"is not allowed at the same level as anyOf",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestReferences(t *testing.T) {
	validCases := []string{
		// Basic reference
		`{
			"type": "object",
			"properties": {
				"user": {
					"$ref": "#/$defs/User"
				}
			},
			"$defs": {
				"User": {
					"type": "object",
					"properties": {
						"name": {"type": "string"}
					}
				}
			}
		}`,
		// Circular reference (with termination condition)
		`{
			"type": "object",
			"properties": {
				"node": {
					"$ref": "#/$defs/Node"
				}
			},
			"$defs": {
				"Node": {
					"type": "object",
					"properties": {
						"value": {"type": "string"},
						"next": {
							"anyOf": [
								{"type": "null"},
								{"$ref": "#/$defs/Node"}
							]
						}
					}
				}
			}
		}`,
		// Multiple references
		`{
			"type": "object",
			"properties": {
				"user": {"$ref": "#/$defs/User"},
				"address": {"$ref": "#/$defs/Address"}
			},
			"$defs": {
				"User": {
					"type": "object",
					"properties": {
						"name": {"type": "string"}
					}
				},
				"Address": {
					"type": "object",
					"properties": {
						"street": {"type": "string"}
					}
				}
			}
		}`,
		// A->B->A but required is empty
		`{
		    "type": "object",
		    "properties": {
		        "x": { "$ref": "#/$defs/a" }
		    },
		    "$defs": {
		        "a": {
		            "type": "object",
		            "properties": {
		                "y": { "$ref": "#/$defs/b" }
		            }
		        },
		        "b": {
		            "type": "object",
		            "properties": {
		                "z": { "$ref": "#/$defs/a" }
		            }
		        }
		    }
		}`,
		`{
		    "type": "object",
		    "properties": {
		        "value": {
		            "anyOf": [
		                {"type": "string"},
		                {
		                    "type": "object",
		                    "properties": {
		                        "next": {"$ref": "#/$defs/Node"}
		                    },
		                    "required": ["next"]
		                }
		            ]
		        }
		    },
		    "$defs": {
		        "Node": {
		            "anyOf": [
		                {"type": "string"},
		                {
		                    "type": "object",
		                    "properties": {
		                        "next": {"$ref": "#/$defs/Node"}
		                    },
		                    "required": ["next"]
		                }
		            ]
		        }
		    }
		}`,
		`{
			"type": "object",
			"properties": {
				"node": {
					"$ref": "#/$defs/Node"
				}
			},
			"required": ["node"],
			"$defs": {
				"Node": {
					"type": "object",
					"properties": {
						"next": {
							"$ref": "#/$defs/Node"
						}
					}
				}
			}
		}`,
		`{
			"type": "object",
			"properties": {
				"node": {
					"$ref": "#/$defs/XXX"
				}
			},
			"$defs": {
				"XXX": {
					"type": "object",
					"required": ["next"],
					"properties": {
						"next": {
							"$ref": "#/$defs/XXX"
						}
					}
				}
			}
		}`,
		`{
			"type": "object",
			"properties": {
				"xxx": {
					"$ref": "#/$defs/Node"
				}
			},
			"$defs": {
				"Node": {
					"type": "object",
					"properties": {
						"next": {
							"$ref": "#/$defs/Node"
						}
					}
				}
			}
		}`,
		// array items is empty
		`{
			"type": "object",
			"properties": {
				"type_info": {
					"type": "string",
					"description": "The type of the UI component",
					"enum": ["div", "button", "header", "section", "field", "form"]
				},
				"label": {
					"type": "string",
					"description": "The label of the UI component, used for buttons or form fields"
				},
				"children": {
					"type": "array",
					"description": "Nested UI components",
					"items": {
						"$ref": "#"
					}
				},
				"attributes": {
					"type": "array",
					"description": "Arbitrary attributes for the UI component, suitable for any element",
					"items": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string",
								"description": "The name of the attribute, for example onClick or className"
							},
							"value": {
								"type": "string",
								"description": "The value of the attribute"
							}
						},
						"additionalProperties": false,
						"required": ["name", "value"]
					}
				}
			},
			"required": ["type_info", "label", "children", "attributes"],
			"additionalProperties": false
		}`,
		`{
		  "$defs": {
		    "Def_0": {
		      "description": "Schema structure data definition",
		      "items": {
		        "anyOf": [
		          {
		            "items": {
		              "anyOf": [
		                {
		                  "type": "null"
		                },
		                {
		                  "type": "object"
		                },
		                {
		                  "type": "string"
		                }
		              ]
		            },
		            "type": "array"
		          },
		          {
		            "properties": {
		              "lmGuD": {
		                "type": "object"
		              }
		            },
		            "required": [
		              "lmGuD"
		            ],
		            "type": "object"
		          }
		        ]
		      },
		      "type": "array"
		    }
		  },
		  "description": "Example structure schema property",
		  "properties": {
		    "JOvrn": {
		      "$ref": "#/$defs/Def_0"
		    },
		    "riiIj": {
		      "properties": {
		        "MEaFc": {
		          "items": {
		            "type": "boolean"
		          },
		          "type": "array"
		        }
		      },
		      "required": [
		        "MEaFc"
		      ],
		      "type": "object"
		    },
		    "vXhVI": {
		      "additionalProperties": false,
		      "properties": {
		        "mFlXV": {
		          "properties": {
		            "xWcek": {
		              "type": "null"
		            }
		          },
		          "required": [
		            "xWcek"
		          ],
		          "type": "object"
		        }
		      },
		      "required": [
		        "mFlXV"
		      ],
		      "type": "object"
		    }
		  },
		  "required": [
		    "riiIj"
		  ],
		  "type": "object"
		}`,
		`{
		  "$defs": {
		    "game": {
		      "description": "Loosely based off the example in the README",
		      "properties": {
		        "black_id": {
		          "$ref": "#/$defs/player/properties/id"
		        },
		        "date_played": {
		          "type": [
		            "string"
		          ]
		        },
		        "game_record": {
		          "type": [
		            "string"
		          ]
		        },
		        "game_server": {
		          "type": [
		            "string"
		          ]
		        },
		        "rated": {
		          "type": [
		            "boolean"
		          ]
		        },
		        "white_id": {
		          "$ref": "#/$defs/player/properties/id"
		        }
		      },
		      "title": "Game",
		      "type": [
		        "object"
		      ]
		    },
		    "player": {
		      "description": "I copied the database table here, didn't document any endpoints here.",
		      "properties": {
		        "id": {
		          "description": "this could be a uuid to make it db agnostic",
		          "type": [
		            "number"
		          ]
		        },
		        "name": {
		          "description": "should this be expanded to first_name, last_name?",
		          "type": [
		            "string"
		          ]
		        },
		        "server_id": {
		          "description": "not sure if this should be exposed, im just copying the table definition",
		          "type": [
		            "number"
		          ]
		        },
		        "token": {
		          "description": "im totally guessing that this is a uuid",
		          "type": [
		            "string"
		          ]
		        }
		      },
		      "title": "Player",
		      "type": [
		        "object"
		      ]
		    }
		  },
		  "properties": {
		    "game": {
		      "$ref": "#/$defs/game"
		    },
		    "player": {
		      "$ref": "#/$defs/player"
		    }
		  },
		  "type": ["object"]
		}`,
		`{
			"type": ["object"],
			"properties": {
				"root": {
				"$ref": "#/$defs/TreeNode"
				}
			},
			"$defs": {
				"TreeNode": {
				"type": ["object"],
				"properties": {
					"value": {
					"type": ["string"]
					},
					"children": {
					"type": ["array"],
					"items": {
						"anyOf": [
						{
							"type": ["null"]
						},
						{
							"$ref": "#/$defs/TreeNode"
						}
						]
					}
					}
				},
				"required": ["value"]
				}
			}
		}`,
		`{
			"type": ["object"],
			"properties": {
				"node": {
				"$ref": "#/$defs/Node"
				}
			},
			"$defs": {
				"Node": {
				"type": ["object"],
				"properties": {
					"value": {
					"type": ["string"]
					},
					"next": {
					"$ref": "#/$defs/Node"
					}
				},
				"required": ["value"]
				}
			}
		}`,
		`{
			"type": ["object"],
			"properties": {
				"data": {
				"$ref": "#/$defs/ArrayWithTermination"
				}
			},
			"$defs": {
				"ArrayWithTermination": {
				"type": ["array"],
				"items": {
					"anyOf": [
					{
						"type": ["string"]
					},
					{
						"type": ["object"],
						"properties": {
						"next": {
							"$ref": "#/$defs/ArrayWithTermination"
						}
						}
					}
					]
				}
				}
			}
		}`,
		// ref to itself, can terminate
		`{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"child": {"$ref": "#"}
			},
			"required": ["name"]
		}`,
		`{
			"type": "object",
			"properties": {
				"node": {
					"anyOf": [
						{"type": "null"},
						{
							"type": "object",
							"properties": {
								"value": {"type": "string"},
								"next": {"$ref": "#"}
							},
							"required": ["next"]
						}
					]
				}
			},
			"required": ["node"]
		}`,
		// child is not required
		`{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"child": {"$ref": "#"}
			},
			"required": ["name"]
		}`,
		`{
			"type": "object",
			"required": ["required_prop"],
			"properties": {
				"required_prop": {
					"type": "string"
				},
				"optional_prop": {
					"type": "object",
					"properties": {
						"self_ref": {
							"$ref": "#"
						}
					}
				}
			}
		}`,
		`{
			"type": "object",
			"required": []
		}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		{
			`{
				"type": "object",
				"properties": {
					"user": {
						"$ref": 123
					}
				}
			}`,
			"$ref must be a string",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"user": {
						"$ref": []
					}
				}
			}`,
			"$ref must be a string",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"user": {
						"$ref": {}
					}
				}
			}`,
			"$ref must be a string",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"user": {
						"$ref": true
					}
				}
			}`,
			"$ref must be a string",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"user": {
						"$ref": null
					}
				}
			}`,
			"$ref must be a string",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"user": {
						"$ref": "#",
						"type": "string"
					}
				}
			}`,
			"when using $ref, type should be defined in the referenced schema instead of the parent schema",
			false,
		},
		// Invalid reference path
		{
			`{
				"type": "object",
				"properties": {
					"user": {
						"$ref": "#/invalid/path"
					}
				}
			}`,
			"references must start with #/$defs/",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"user": {
						"$ref": "#/$defs/User",
						"minLength": 10
					}
				},
				"$defs": {
					"User": {
						"type": "object",
						"properties": {
							"name": {"type": "string"}
						}
					}
				}
			}`,
			"not allowed at the same level as $ref",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"user": {
						"$ref": "#/$defs/User",
						"minItems": 10
					}
				},
				"$defs": {
					"User": {
						"type": "string"
					}
				}
			}`,
			"not allowed at the same level as $ref",
			false,
		},
		// Reference to non-existent definition
		{
			`{
				"type": "object",
				"properties": {
					"user": {
						"$ref": "#/$defs/NonExistent"
					}
				}
			}`,
			"$defs not found for reference",
			false,
		},
		// Infinite circular reference
		{
			`{
				"type": "object",
				"properties": {
					"node": {
						"$ref": "#/$defs/YYY"
					}
				},
				"required": ["node"],
				"$defs": {
					"YYY": {
						"type": "object",
						"properties": {
							"next": {
								"$ref": "#/$defs/YYY"
							}
						},
						"required": ["next"]
					}
				}
			}`,
			"infinite recursion",
			false,
		},
		{
			`{
		        "type": "object",
		        "properties": {
		            "value": {
		                "anyOf": [
		                    {
		                        "type": "object",
		                        "properties": {
		                            "next": {"$ref": "#/$defs/Node"}
		                        },
		                        "required": ["next"]
		                    },
		                    {
		                        "type": "object",
		                        "properties": {
		                            "child": {"$ref": "#/$defs/Node"}
		                        },
		                        "required": ["child"]
		                    }
		                ]
		            }
		        },
				"required": ["value"],
		        "$defs": {
		            "Node": {
		                "anyOf": [
		                    {
		                        "type": "object",
		                        "properties": {
		                            "next": {"$ref": "#/$defs/Node"}
		                        },
		                        "required": ["next"]
		                    },
		                    {
		                        "type": "object",
		                        "properties": {
		                            "child": {"$ref": "#/$defs/Node"}
		                        },
		                        "required": ["child"]
		                    }
		                ]
		            }
		        }
		    }`,
			"infinite recursion",
			false,
		},
		// no termination condition
		{
			`{
		        "type": "object",
		        "properties": {
		            "x": { "$ref": "#/$defs/a" }
		        },
				"required": ["x"],
		        "$defs": {
		            "a": {
		                "type": "object",
		                "properties": {
		                    "y": { "$ref": "#/$defs/b" }
		                },
		                "required": ["y"]
		            },
		            "b": {
		                "type": "object",
		                "properties": {
		                    "z": { "$ref": "#/$defs/a" }
		                },
		                "required": ["z"]
		            }
		        }
		    }`,
			"infinite recursion",
			false,
		},
		{
			`{
                "type": "object",
                "properties": {
                    "level1": { "$ref": "#/$defs/level1" }
                },
                "required": ["level1"],
                "$defs": {
                    "level1": {
                        "type": "object",
                        "properties": {
                            "level2": { "$ref": "#/$defs/level2" }
                        },
                        "required": ["level2"]
                    },
                    "level2": {
                        "type": "object",
                        "properties": {
                            "level3": { "$ref": "#/$defs/level3" }
                        },
                        "required": ["level3"]
                    },
                    "level3": {
                        "type": "object",
                        "properties": {
                            "level4": { "$ref": "#/$defs/level4" }
                        },
                        "required": ["level4"]
                    },
                    "level4": {
                        "type": "object",
                        "properties": {
                            "level5": { "$ref": "#/$defs/level5" }
                        },
                        "required": ["level5"]
                    },
                    "level5": {
                        "type": "object",
                        "properties": {
                            "level6": { "$ref": "#/$defs/level6" }
                        },
                        "required": ["level6"]
                    },
                    "level6": {
                        "type": "object",
                        "properties": {
                            "level7": { "$ref": "#/$defs/level7" }
                        },
                        "required": ["level7"]
                    },
                    "level7": {
                        "type": "object",
                        "properties": {
                            "level8": { "$ref": "#/$defs/level8" }
                        },
                        "required": ["level8"]
                    },
                    "level8": {
                        "type": "object",
                        "properties": {
                            "level9": { "$ref": "#/$defs/level9" }
                        },
                        "required": ["level9"]
                    },
                    "level9": {
                        "type": "object",
                        "properties": {
                            "level10": { "$ref": "#/$defs/level10" }
                        },
                        "required": ["level10"]
                    },
                    "level10": {
                        "type": "object",
                        "properties": {
                            "level1": { "$ref": "#/$defs/level1" }
                        },
                        "required": ["level1"]
                    }
                }
            }`,
			"schema depth exceeds maximum limit",
			false,
		},
		{
			`{
		        "type": "object",
		        "properties": {
		            "root": { "$ref": "#/$defs/root" }
		        },
		        "required": ["root"],
		        "$defs": {
		            "root": {
		                "type": "object",
		                "properties": {
		                    "arrayProp": {
		                        "type": "array",
		                        "items": { "$ref": "#/$defs/arrayItem" }
		                    },
		                    "objectProp": { "$ref": "#/$defs/objectProp" }
		                },
		                "required": ["arrayProp", "objectProp"]
		            },
		            "arrayItem": {
		                "type": "object",
		                "properties": {
		                    "nestedArray": {
		                        "type": "array",
		                        "items": { "$ref": "#/$defs/nestedArrayItem" }
		                    }
		                },
		                "required": ["nestedArray"]
		            },
		            "nestedArrayItem": {
		                "type": "object",
		                "properties": {
		                    "root": { "$ref": "#/$defs/root" }
		                },
		                "required": ["root"]
		            },
		            "objectProp": {
		                "type": "object",
		                "properties": {
		                    "nestedObject": { "$ref": "#/$defs/nestedObject" }
		                },
		                "required": ["nestedObject"]
		            },
		            "nestedObject": {
		                "type": "object",
		                "properties": {
		                    "root": { "$ref": "#/$defs/root" }
		                },
		                "required": ["root"]
		            }
		        }
		    }`,
			"infinite recursion",
			false,
		},
		// anyOf value is List, not Dict
		{
			`{
		        "type": "object",
		        "properties": {
		            "value": {
		                "$ref": "#/$defs/Node/anyOf"
		            }
		        },
		        "$defs": {
		            "Node": {
		                "anyOf": [{"type": "string"}, {"type": "number"}]
		            }
		        }
		    }`,
			"invalid $ref path",
			false,
		},
		{
			`{
		        "type": "object",
		        "properties": {
		            "value": {
		                "$ref": "#/$defs/Node",
		                "anyOf": [{"type": "string"}, {"type": "number"}]
		            }
		        },
		        "$defs": {
		            "Node": {
		                "anyOf": [{"type": "string"}, {"type": "number"}]
		            }
		        }
		    }`,
			"not allowed at the same level",
			false,
		},
		{
			`{
				"type": ["object"],
				"properties": {
					"root": {
						"$ref": "#/$defs/Level1"
					}
				},
				"required": ["root"],
				"$defs": {
					"Level1": {
						"type": ["object"],
						"properties": {
							"level2": {
								"$ref": "#/$defs/Level2"
							}
						},
						"required": ["level2"]
					},
					"Level2": {
						"type": ["array"],
						"items": {
							"type": ["object"],
							"properties": {
								"level3": {
									"$ref": "#/$defs/Level3"
								}
							},
							"required": ["level3"]
						}
					},
					"Level3": {
						"type": ["object"],
						"properties": {
							"level1": {
								"$ref": "#/$defs/Level1"
							}
						},
						"required": ["level1"]
					}
				}
			}`,
			"infinite recursion",
			false,
		},
		{
			`{
				"type": ["object"],
				"properties": {
				  "node": {
					"$ref": "#/$defs/InfiniteNode"
				  }
				},
				"required": ["node"],
				"$defs": {
				  "InfiniteNode": {
					"type": ["object"],
					"properties": {
					  "next": {
						"$ref": "#/$defs/InfiniteNode"
					  }
					},
					"required": ["next"]
				  }
				}
			}`,
			"infinite recursion",
			false,
		},
		{
			`{
				"type": ["object"],
				"properties": {
					"data": {
					"$ref": "#/$defs/InfiniteArray"
					}
				},
				"required": ["data"],
				"$defs": {
					"InfiniteArray": {
					"type": ["array"],
					"items": {
						"type": ["object"],
						"properties": {
							"element": {
								"$ref": "#/$defs/InfiniteArray"
							}
						},
						"required": ["element"]
					}
					}
				}
			}`,
			"infinite recursion",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"node": {
						"anyOf": [
							{"$ref": "#"},
							{
								"type": "object",
								"properties": {
									"next": {"$ref": "#"}
								},
								"required": ["next"]
							}
						]
					}
				},
				"required": ["node"]
			}`,
			"infinite recursion",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"children": {
						"type": "array",
						"items": {"$ref": "#"}
					}
				},
				"required": ["children"]
			}`,
			"infinite recursion",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"node": {"$ref": "#"}
				},
				"required": ["node"]
			}`,
			"infinite recursion",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"data": {
					"type": "object",
					"properties": {
						"value": {"type": "integer"},
						"next": {"$ref": "#"}
					},
					"required": ["next"]
					}
				},
				"required": ["data"]
			}`,
			"infinite recursion",
			false,
		},
		{
			`{
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
					"elements": {"$ref": "#"}
					},
					"required": ["elements"]
				},
				"minItems": 1
			}`,
			"infinite recursion",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"level1": {
						"type": "object",
						"properties": {
							"level2": {
								"type": "object",
								"properties": {
									"back": {"$ref": "#"}
								},
								"required": ["back"]
							}
						},
						"required": ["level2"]
					}
				},
				"required": ["level1"]
			}`,
			"infinite recursion",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestAnyOf(t *testing.T) {
	validCases := []string{
		// Simple anyOf
		`{
			"anyOf": [
				{"type": "string"},
				{"type": "number"}
			]
		}`,
		// anyOf with nested schemas
		`{
			"anyOf": [
				{
					"type": "object",
					"properties": {
						"name": {"type": "string", "default": "John"}
					}
				},
				{
					"type": "object",
					"properties": {
						"age": {"type": "integer"}
					}
				}
			]
		}`,
		// anyOf with array items
		`{
			"type": "array",
			"items": {
				"anyOf": [
					{"type": "string"},
					{"type": "number"}
				]
			}
		}`,
		// anyOf with refs
		`{
			"title": "anyOf with refs",
			"$id": "anyOf with refs",
			"anyOf": [
				{"$ref": "#/$defs/StringType"},
				{"$ref": "#/$defs/NumberType"}
			],
			"$defs": {
				"StringType": {"type": "string"},
				"NumberType": {"type": "number"}
			}
		}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		{
			`{"anyOf": "not an array"}`,
			"anyOf must be an array",
			false,
		},
		{
			`{"anyOf": 123}`,
			"anyOf must be an array",
			false,
		},
		{
			`{
				"anyOf": [
					{"type": "string"},
					[]
				]
			}`,
			"schema in anyOf must be an object",
			false,
		},
		{
			`{"anyOf": [{"type": "string"}, "not an object"]}`,
			"schema in anyOf must be an object",
			false,
		},
		// type definition
		{
			`{
				"type": "string",
				"anyOf": [
					{"type": "string"},
					{"type": "number"}
				]
			}`,
			"type should be defined in anyOf",
			false,
		},
		// empty anyOf
		{
			`{"anyOf": []}`,
			"anyOf must have",
			false,
		},
		// anyOf & ref
		{
			`{
				"type": "object",
				"properties": {
					"name": {
						"anyOf": [
							{"type": "string"},
							{"type": "number"}
						],
						"$ref": "#/$defs/group"
					}
				},
				"$defs": {
					"group": {
						"type": "object",
						"properties": {
							"name": {"type": "object", "default": {"firstName": "John", "lastName": "Doe"}}
						}
					}
				}
			}`,
			"not allowed at the same level as",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"name": {
						"anyOf": [
							{"type": "string"},
							{"type": "object",
								"properties": {
									"name": {"type": "string"}
								}
							}
						],
						"required": ["name"]
					}
				}
			}`,
			"required",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"name": {
						"anyOf": [
							{"type": "string"},
							{"type": "object",
								"properties": {
									"name": {"type": "string"}
								}
							}
						],
						"minLength": 10
					}
				}
			}`,
			"minLength",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"name": {
						"anyOf": [
							{"$ref": "#/$defs/User"},
							{"type": "object",
								"properties": {
									"name": {"type": "string"}
								}
							}
						],
						"minLength": 10
					}
				},
				"$defs": {
					"User": {
						"type": "string"
					}
				}
			}`,
			"minLength",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestDefs(t *testing.T) {
	validCases := []string{
		`{
			"type": "object",
			"$defs": {
				"User": {
					"type": "object",
					"properties": {
						"name": {"type": "string"}
					}
				}
			}
		}`,
		`{
			"$defs": {
				"onePriv": {
					"type": "string",
					"enum": [
						"all",
						"insert",
						"delete",
						"select",
						"update",
						"truncate",
						"references",
						"trigger",
						"usage",
						"execute",
						"create"
					]
				},
				"privileges": {
					"type": "array",
					"items": {
						"type": "object",
						"additionalProperties": {
							"type": "array",
							"items": {
								"$ref": "#/$defs/onePriv"
							}
						}
					}
				},
				"db": {
					"type": "object",
					"additionalProperties": false
				},
				"schema": {
					"type": "object",
					"additionalProperties": false,
					"properties": {
						"owner": {
							"type": "string"
						},
						"description": {
							"type": "string"
						},
						"privileges": {
							"$ref": "#/$defs/privileges"
						}
					}
				}
			},
			"$ref": "#/$defs/db",
			"title": "JSON schema for Pyrseas yaml files"
		}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		{
			`{
				"type": "object",
				"$defs": {
					"User": []
				}
			}`,
			"$defs schema must be object",
			false,
		},
		// Missing $defs for reference
		{
			`{
				"type": "object",
				"properties": {
					"user": {"$ref": "#/$defs/User"}
				}
			}`,
			"$defs not found for reference",
			false,
		},
		// Empty definition name
		{
			`{
				"type": "object",
				"$defs": {
					"": {"type": "string"}
				}
			}`,
			"$defs property name cannot be empty",
			false,
		},
		// Non-string definition name
		{
			`{
				"$defs": {
					123: {"type": "string"}
				}
			}`,
			"JSON schema parsing error",
			true,
		},
		// Invalid reference format
		{
			`{
				"type": "object",
				"properties": {
					"user": {"$ref": "invalid_ref"}
				}
			}`,
			"references must start with #/$defs/",
			false,
		},
		// $defs is not an object
		{
			`{
				"type": "object",
				"$defs": "not an object"
			}`,
			"$defs must be an object",
			false,
		},
		// Definition schema is not an object
		{
			`{
				"type": "object",
				"$defs": {
					"User": "not an object"
				}
			}`,
			"$defs schema must be object",
			false,
		},
		{
			`{
				"$defs": {
					"nullDef": null
				},
				"type": "object"
			}`,
			"$defs schema must be object",
			false,
		},
		{
			`{
				"$defs": {
					"nullDef": []
				},
				"type": "object"
			}`,
			"$defs schema must be object",
			false,
		},
		{
			`{
				"$defs": {
					"positive/integer": {
						"type": "integer"
					}
				},
				"type": "object"
			}`,
			"cannot contain '/' character",
			false,
		},
		{
			`{
				"$defs": {
					"test": {
						"type": "object",
						"properties": {
							"a/b": {"type": "string"}
						}
					}
				},
				"type": "object"
			}`,
			"cannot contain '/' character",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestNumberFormat(t *testing.T) {
	validCases := []string{
		// Integers
		`{"type": "number", "enum": [1, 2, 3]}`,
		`{"type": "number", "enum": [-1, 0, 1]}`,
		// Floating point numbers
		`{"type": "number", "enum": [1.23, 4.56]}`,
		`{"type": "number", "enum": [-1.23, 0.0, 1.23]}`,
		// Mixed integers and floating point
		`{"type": "number", "enum": [1, 2.5, 3, 4.5]}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		// enum contains boolean
		{
			`{"type": "number", "enum": [1, 2, true]}`,
			"not a valid number",
			false,
		},
		// enum contains leading zero integer
		{
			`{"type": "number", "enum": [01.23]}`,
			"JSON schema parsing error",
			true,
		},
		// enum contains non-numeric string
		{
			`{"type": "number", "enum": [1, "01", 3]}`,
			"not a valid number",
			false,
		},
		// enum contains scientific notation string
		{
			`{"type": "number", "enum": [1, "1e5", 3]}`,
			"not a valid number",
			false,
		},
		// enum contains special float value
		{
			`{"type": "number", "enum": [1, 1.7976931348623157e+309, 3]}`,
			"JSON schema parsing error",
			true,
		},
		// enum contains non-numeric string (leading zero decimal)
		{
			`{"type": "number", "enum": [1, "01.23", 3]}`,
			"not a valid number",
			false,
		},
		// enum contains non-numeric string (negative with leading zero)
		{
			`{"type": "number", "enum": [1, "-01.23", 3]}`,
			"not a valid number",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestRefInProperties(t *testing.T) {
	validCases := []string{
		`{
			"type": "object",
			"properties": {
				"parent": {
					"$ref": "#/$defs/ParentObject"
				},
				"address": {
					"$ref": "#/$defs/Address"
				},
				"tags": {
					"type": "array",
					"items": {
						"$ref": "#/$defs/Tag"
					}
				}
			},
			"$defs": {
				"ParentObject": {
					"type": "object",
					"properties": {
						"id": {"type": "string"}
					}
				},
				"Address": {
					"type": "object",
					"properties": {
						"street": {"type": "string"},
						"city": {"type": "string"}
					}
				},
				"Tag": {
					"type": "string"
				}
			}
		}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		{
			`{
				"type": "object",
				"properties": {
					"parent": {
						"$ref": "#/$defs/NonExistent"
					}
				}
			}`,
			"$defs not found for reference",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestTypeLocation(t *testing.T) {
	validCases := []string{
		// anyOf all items have type definition, no need for external type
		`{
			"anyOf": [
				{"type": "string"},
				{"type": "number"}
			]
		}`,
		`{
			"type": "array",
			"items": {"type": "string"}
		}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		// anyOf internal type inconsistent with external type
		{
			`{
				"type": "string",
				"anyOf": [
					{"type": "string"},
					{"type": "number"}
				]
			}`,
			"type should be defined in anyOf",
			false,
		},
		// anyOf has items without type definition
		// {
		// 	`{
		// 		"anyOf": [
		// 			{"type": "string"},
		// 			{"minLength": 1, "maxLength": 100}
		// 		]
		// 	}`,
		// 	"type need to be defined explicitly",
		// 	false,
		// },
		// Completely missing type definition
		// {
		// 	`{
		// 		"properties": {
		// 			"name": {"type": "string"}
		// 		}
		// 	}`,
		// 	"type need to be defined explicitly",
		// 	false,
		// },
		// Invalid: missing type: "object"
		// {
		// 	`{
		// 		"properties": {
		// 			"name": {"type": "string"}
		// 		},
		// 		"required": ["name"]
		// 	}`,
		// 	"type need to be defined explicitly",
		// 	false,
		// },
		// Invalid: wrong type
		{
			`{
				"type": "array",
				"required": ["name"]
			}`,
			"invalid keywords",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestNestedDefsDepth(t *testing.T) {
	validCases := []string{
		`{
			"$defs": {
				"level1": {
					"type": "object",
					"properties": {
						"regular_prop": { "type": "string" }
					},
					"additionalProperties": {
						"type": "object",
						"properties": {
							"nested_prop": { "type": "number" }
						}
					}
				}
			},
			"type": "object"
        }`,
		`{
			"type": "object",
			"properties": {
				"root": {
					"$ref": "#/$defs/group/items"
				}
			},
			"$defs": {
				"group": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"data": {"type": "string"}
						},
						"additionalProperties": {
							"$ref": "#/$defs/group/items"
						}
					}
				}
			}
		}`,
		`{
			"type": "object",
			"properties": {
				"type_info": {
					"type": "string",
					"description": "The type of the UI component",
					"enum": ["div", "button", "header", "section", "field", "form"]
				},
				"label": {
					"type": "string",
					"description": "The label of the UI component, used for buttons or form fields"
				},
				"children": {
					"type": "array",
					"description": "Nested UI components",
					"items": {
						"$ref": "#"
					}
				},
				"attributes": {
					"type": "array",
					"description": "Arbitrary attributes for the UI component, suitable for any element",
					"items": {
						"type": "object",
						"properties": {
							"name": {
								"type": "string",
								"description": "The name of the attribute, for example onClick or className"
							},
							"value": {
								"type": "string",
								"description": "The value of the attribute"
							}
						},
						"additionalProperties": false,
						"required": ["name", "value"]
					}
				}
			},
			"required": ["type_info", "label", "children", "attributes"],
			"additionalProperties": false
		}`,
		`{
			"$defs": {
				"mySchema": {
					"type": "object",
					"properties": {
						"name": {
							"type": "string"
						},
						"child": {
							"$ref": "#/$defs/mySchema"
						}
					},
					"additionalProperties": false
				}
			},
			"type": "object",
			"properties": {
				"root": {
					"$ref": "#/$defs/mySchema"
				}
			}
		}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		{
			`{
				"$defs": {
					"type": "object",
					"properties": {
						"regular_prop": { "type": "string" }
					},
					"additionalProperties": {
						"type": "object",
						"properties": {
							"nested_prop": { "type": "number" }
						}
					}
				},
				"type": "object"
			}`,
			"unsupported keywords: regular_prop",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"root": {
						"$ref": "#/$defs/group/items/properties/properties/properties/properties/additionalProperties/properties"
					}
				},
				"$defs": {
					"group": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"data": {
									"type": "object",
									"properties": {
										"num": {
											"type": "object",
											"properties": {
												"value": {
													"type": "object",
													"properties": {
														"num": {
															"type": "object",
															"additionalProperties": {
																"type": "object",
																"properties": {
																	"value": {
																		"type": "object",
																		"properties": {
																			"value": {"type": "number"}
																		}
																	}
																}
															}
														}
													}
												}
											}
										}
									}
								}
							},
							"additionalProperties": {
								"$ref": "#/$defs/group/items/properties/properties/properties/properties/additionalProperties/properties"
							}
						}
					}
				}
			}`,
			"schema depth exceeds maximum limit",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"root": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"data": {
									"type": "object",
									"properties": {
										"num": {
											"type": "object",
											"properties": {
												"value": {
													"type": "object",
													"properties": {
														"num": {
															"type": "object",
															"additionalProperties": {
																"type": "object",
																"properties": {
																	"value": {
																		"type": "object",
																		"properties": {
																			"value": {"type": "number"}
																		}
																	}
																}
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}`,
			"schema depth exceeds maximum limit",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"root": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"data": {
									"type": "object",
									"properties": {
										"num": {
											"type": "object",
											"properties": {
												"value": {
													"type": "object",
													"properties": {
														"num": {
															"type": "object",
															"additionalProperties": {
																"type": "object",
																"properties": {
																	"value": {
																		"type": "object",
																		"properties": {
																			"value": {"type": "number"}
																		}
																	}
																}
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}`,
			"schema depth exceeds maximum limit",
			false,
		},
		{
			`{
				"type": "object",
				"properties": {
					"root": {
						"type": "array",
						"items": {
							"type": "object",
							"properties": {
								"data": {
									"type": "object",
									"properties": {
										"num": {
											"anyOf": [
												{
													"type": "object",
													"properties": {
														"value": {
															"type": "object",
															"properties": {
																"num": {
																	"type": "object",
																	"properties": {
																		"value": {
																			"type": "object",
																			"properties": {
																				"value": {"type": "number"}
																			}
																		}
																	}
																}
															}
														}
													}
												},
												{
													"type": "string"
												}
											]
										}
									}
								}
							}
						}
					}
				}
			}`,
			"schema depth exceeds maximum limit",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestRangeConstraints(t *testing.T) {
	validCases := []string{
		// String length constraints
		`{
			"type": "string",
			"minLength": 0,
			"maxLength": 10
		}`,
		`{
			"type": "string",
			"minLength": 5,
			"maxLength": 5
		}`,
		`{
			"type": "string",
			"minLength": 999999999999
		}`,
		`{
			"type": "string",
			"maxLength": 5000000
		}`,
		// Numeric range constraints
		`{
			"type": "number",
			"minimum": -10,
			"maximum": 10
		}`,
		`{
			"type": "number",
			"minimum": -10.5111111000000000,
			"maximum": 20.333333333
		}`,
		`{
			"type": "integer",
			"minimum": -0,
			"maximum": 0
		}`,
		`{
			"type": "integer",
			"maximum": 10000000000000000
		}`,
		`{
			"type": "integer",
			"minimum": -1000000
		}`,
		// Array length constraints
		`{
			"type": "array",
			"items": {"type": "string"},
			"minItems": 0,
			"maxItems": 10
		}`,
		`{
			"type": "array",
			"items": {"type": "string"},
			"minItems": 3,
			"maxItems": 3
		}`,
		`{
			"type": "array",
			"items": {"type": "integer"},
			"maxItems": 3
		}`,
		`{
			"type": "array",
			"items": {"type": "integer"},
			"maxItems": 3
		}`,
		`{
			"type": "string",
			"minLength": 5
		}`,
		`{
			"type": "string",
			"maxLength": 10
		}`,
		`{
			"type": "number",
			"minimum": 0
		}`,
		`{
			"type": "number",
			"maximum": 100
		}`,
		`{
			"type": "array",
			"items": {"type": "string"},
			"minItems": 1
		}`,
		`{
			"type": "array",
			"items": {"type": "string"},
			"maxItems": 10
		}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		// String length constraints
		{
			`{
				"type": "string",
				"minLength": 10,
				"maxLength": 5
			}`,
			"cannot be greater than maxLength",
			false,
		},
		{
			`{
				"type": "string",
				"minLength": -1,
				"maxLength": 5
			}`,
			"non-negative",
			false,
		},
		{
			`{
				"type": "string",
				"minLength": 0,
				"maxLength": -5
			}`,
			"non-negative",
			false,
		},
		{
			`{
				"type": "string",
				"minLength": "5",
				"maxLength": 10
			}`,
			"must be an integer",
			false,
		},
		{
			`{
				"type": "string",
				"minLength": 5,
				"maxLength": "10"
			}`,
			"must be an integer",
			false,
		},
		// Numeric range constraints
		{
			`{
				"type": "number",
				"minimum": 10,
				"maximum": 5
			}`,
			"cannot be greater than",
			false,
		},
		{
			`{
				"type": "number",
				"minimum": "0",
				"maximum": 10
			}`,
			"must be a number",
			false,
		},
		{
			`{
				"type": "number",
				"minimum": 0,
				"maximum": "10"
			}`,
			"must be a number",
			false,
		},
		// Integer type specific constraints
		{
			`{
				"type": "integer",
				"minimum": 1.5,
				"maximum": 10
			}`,
			"not a valid integer",
			false,
		},
		{
			`{
				"type": "integer",
				"minimum": 1,
				"maximum": 10.5
			}`,
			"not a valid integer",
			false,
		},
		// Array length constraints
		{
			`{
				"type": "array",
				"items": {"type": "string"},
				"minItems": 10,
				"maxItems": 5
			}`,
			"greater than",
			false,
		},
		{
			`{
				"type": "array",
				"items": {"type": "string"},
				"minItems": -1,
				"maxItems": 5
			}`,
			"must be non-negative",
			false,
		},
		{
			`{
				"type": "array",
				"items": {"type": "string"},
				"minItems": 0,
				"maxItems": -5
			}`,
			"must be non-negative",
			false,
		},
		{
			`{
				"type": "array",
				"items": {"type": "string"},
				"minItems": "5",
				"maxItems": 10
			}`,
			"must be an integer",
			false,
		},
		{
			`{
				"type": "array",
				"items": {"type": "string"},
				"minItems": 5,
				"maxItems": "10"
			}`,
			"must be an integer",
			false,
		},
		// String length - single constraint invalid cases
		{
			`{
				"type": "string",
				"minLength": -1
			}`,
			"minLength must be non-negative",
			false,
		},
		{
			`{
				"type": "string",
				"maxLength": -5
			}`,
			"maxLength must be non-negative",
			false,
		},
		{
			`{
				"type": "string",
				"minLength": "5"
			}`,
			"minLength must be an integer",
			false,
		},
		{
			`{
				"type": "string",
				"maxLength": "10"
			}`,
			"maxLength must be an integer",
			false,
		},
		// Numeric range - single constraint invalid cases
		{
			`{
				"type": "integer",
				"minimum": 1.5
			}`,
			"not a valid integer",
			false,
		},
		{
			`{
				"type": "integer",
				"maximum": 10.5
			}`,
			"not a valid integer",
			false,
		},
		{
			`{
				"type": "integer",
				"minimum": "0"
			}`,
			"minimum must be an integer",
			false,
		},
		{
			`{
				"type": "number",
				"minimum": "0"
			}`,
			"minimum must be a number",
			false,
		},
		{
			`{
				"type": "integer",
				"maximum": "10"
			}`,
			"maximum must be an integer",
			false,
		},
		{
			`{
				"type": "number",
				"maximum": "10"
			}`,
			"maximum must be a number",
			false,
		},
		// Array length - single constraint invalid cases
		{
			`{
				"type": "array",
				"items": {"type": "string"},
				"minItems": -1
			}`,
			"minItems must be non-negative",
			false,
		},
		{
			`{
				"type": "array",
				"items": {"type": "string"},
				"maxItems": -5
			}`,
			"maxItems must be non-negative",
			false,
		},
		{
			`{
				"type": "array",
				"items": {"type": "string"},
				"minItems": "5"
			}`,
			"minItems must be an integer",
			false,
		},
		{
			`{
				"type": "array",
				"items": {"type": "string"},
				"maxItems": "10"
			}`,
			"maxItems must be an integer",
			false,
		},
		{
			`{
				"type": "array",
				"items": 123,
				"minLength": 1,
				"maxLength": 10
			}`,
			"invalid keywords:",
			false,
		},
		{
			`{
				"type": "array",
				"items": 123,
				"minimum": 1,
				"maximum": 10
			}`,
			"invalid keywords:",
			false,
		},
		{
			`{
				"type": "string",
				"minItems": 1,
				"maxItems": 10
			}`,
			"invalid keywords:",
			false,
		},
		// {
		// 	`{
		// 		"minimum": 0,
		// 		"maximum": 100
		// 	}`,
		// 	"type need to be defined explicitly",
		// 	false,
		// },
	}

	runTestCases(t, validCases, invalidCases)
}

func TestID(t *testing.T) {
	validCases := []string{
		`{
			"$id": "http://example.com/schema.json",
			"type": "object"
		}`,
		`{
			"$id": "my-schema",
			"type": "string"
		}`,
		`{
			"$id": "#user",
			"type": "object"
		}`,
		`{
			"$id": "",
			"type": "object"
		}`,
		`{
			"$id": "https://example.com/schemas/user#/definitions/name",
			"type": "object"
		}`,
		`{
			"$id": "https://example.com/schemas/user#/$defs/name",
			"type": "object"
		}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		{
			`{
				"$id": 123,
				"type": "object"
			}`,
			"$id must be a string",
			false,
		},
		{
			`{
				"$id": ["http://example.com"],
				"type": "object"
			}`,
			"$id must be a string",
			false,
		},
		{
			`{
				"$id": {"url": "http://example.com"},
				"type": "object",
				"default": {"name": "John"}
			}`,
			"$id must be a string",
			false,
		},
		{
			`{
				"$id": true,
				"type": "object"
			}`,
			"$id must be a string",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestDescription(t *testing.T) {
	validCases := []string{
		`{
				"type": "string",
				"description": "This is a valid description"
			}`,
		// Empty description
		`{
				"type": "string",
				"description": ""
			}`,
		// With special chars
		`{
				"type": "string",
				"description": "Special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?"
			}`,
		// Multi-line description
		`{
				"type": "string",
				"description": "Line 1\nLine 2\nLine 3"
			}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		{
			// Non-string description
			`{
					"type": "string",
					"description": 123
				}`,
			"description must be a string",
			false,
		},
		{
			// Object type description
			`{
					"type": "string",
					"description": {"text": "Invalid description"}
				}`,
			"description must be a string",
			false,
		},
		{
			// Array type description
			`{
					"type": "string",
					"description": ["Invalid", "description"]
				}`,
			"description must be a string",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestMaxTotalProperties(t *testing.T) {
	validator := newSchemaValidator()

	// Create a schema with too many properties
	properties1 := make(SchemaDict)
	properties2 := make(SchemaDict)
	for i := 1; i <= 10000; i++ {
		properties1[fmt.Sprintf("%d", i)] = SchemaDict{"type": "string"}
		properties2[fmt.Sprintf("k%d", i)] = SchemaDict{"type": "string"}
	}
	properties2["k10001"] = SchemaDict{"type": "string"}

	// schema1 := SchemaDict{
	// 	"type":       "object",
	// 	"properties": properties1,
	// }
	schema2 := SchemaDict{
		"type":       "object",
		"properties": properties2,
	}

	// TODO: MaxSchemaSize maybe too small
	// if err1 := validator.Validate(schema1); err1 != nil {
	// 	t.Errorf("Valid schema failed: %v", err1)
	// }

	if err2 := validator.Validate(schema2); err2 == nil {
		if err2 == nil {
			t.Errorf("schema with too many properties should have failed")
		} else {
			expectedErr1 := "total number of properties keys across all objects exceeds maximum"
			expectedErr2 := "schema exceeds maximum allowed size"
			errMsg := strings.ToLower(err2.Error())
			if !strings.Contains(errMsg, strings.ToLower(expectedErr1)) && !strings.Contains(errMsg, strings.ToLower(expectedErr2)) {
				t.Errorf("Expected error containing '%s' or '%s', got '%s'", expectedErr1, expectedErr2, err2.Error())
			}
		}
	}
}

func TestEnumStringLength(t *testing.T) {
	createEnumJSON := func(prefix string, count int) string {
		var values []string
		for i := 0; i < count; i++ {
			values = append(values, fmt.Sprintf(`"%s%d"`, prefix, i))
		}
		return fmt.Sprintf(`{"type": "string", "enum": [%s]}`, strings.Join(values, ", "))
	}

	createNumericEnumJSON := func(count int) string {
		var values []string
		for i := 0; i < count; i++ {
			values = append(values, fmt.Sprintf("%d", i))
		}
		return fmt.Sprintf(`{"type": "number", "enum": [%s]}`, strings.Join(values, ", "))
	}

	createLargeNumericEnumJSON := func(value float64, count int) string {
		var values []string
		for i := 0; i < count; i++ {
			values = append(values, fmt.Sprintf("%f", value))
		}
		return fmt.Sprintf(`{"type": "number", "enum": [%s]}`, strings.Join(values, ", "))
	}

	createLargeIntegerEnumJSON := func(value int64, count int) string {
		var values []string
		for i := 0; i < count; i++ {
			values = append(values, fmt.Sprintf("%d", value))
		}
		return fmt.Sprintf(`{"type": "integer", "enum": [%s]}`, strings.Join(values, ", "))
	}

	validCases := []string{
		// Less than 250 enum values
		createEnumJSON("long_value_", 249),
		// Exactly 250 enum values
		createEnumJSON("long_value_", 250),
		// More than 250 but short string
		createEnumJSON("s_", 300),
		// Numeric enum values
		createNumericEnumJSON(250),
		// Integer enum values
		createNumericEnumJSON(300),
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		// More than 250 values and long string
		{
			createEnumJSON("very_loooooooooooong_enum_value_", 251),
			"total string length of enum values",
			false,
		},
		// Numeric type but value is too large
		{
			createLargeNumericEnumJSON(123456789010.123456789, 499),
			"total string length of enum values",
			false,
		},
		// Integer type but value is too large
		{
			createLargeIntegerEnumJSON((1<<53)-1, 499),
			"total string length of enum values",
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestConcurrentValidation(t *testing.T) {
	// Test concurrent validation
	concurrentNum := 32
	t.Run("Concurrent validation", func(t *testing.T) {
		schema := `{"type":"object","required":["id","name","details","tags","metadata"],"properties":{"id":{"type":"string","minLength":5,"maxLength":50},"name":{"type":"string","minLength":3,"maxLength":100},"age":{"type":"integer","minimum":0,"maximum":150},"email":{"type":"string"},"details":{"type":"object","required":["description","status"],"properties":{"description":{"type":"string"},"status":{"type":"string","enum":["active","inactive","pending"]},"createdAt":{"type":"string"},"score":{"type":"number","minimum":0,"maximum":10}}},"tags":{"type":"array","minItems":1,"maxItems":10,"items":{"type":"string","minLength":2}},"metadata":{"type":"object","additionalProperties":{"type":"string"}},"settings":{"type":"object","properties":{"notifications":{"type":"boolean"},"theme":{"type":"string","enum":["light","dark","system"]},"fontSize":{"type":"integer","minimum":8,"maximum":24}}}},"additionalProperties":false}`

		// Channel to collect execution times
		timings := make(chan time.Duration, concurrentNum)
		var wg sync.WaitGroup
		wg.Add(concurrentNum)

		for i := 0; i < concurrentNum; i++ {
			go func() {
				defer wg.Done()

				startTime := time.Now()

				validator := newSchemaValidator(WithValidateLevel(ValidateLevelTest))
				if err := validator.Validate(schema); err != nil {
					t.Errorf("Concurrent validation failed: %v", err)
				}

				executionTime := time.Since(startTime)
				timings <- executionTime
			}()
		}

		go func() {
			wg.Wait()
			close(timings)
		}()

		var times []time.Duration
		for duration := range timings {
			times = append(times, duration)
		}

		// Calculate statistics
		var totalTime time.Duration
		minTime := times[0]
		maxTime := times[0]

		for _, duration := range times {
			totalTime += duration
			if duration < minTime {
				minTime = duration
			}
			if duration > maxTime {
				maxTime = duration
			}
		}

		avgTime := totalTime / time.Duration(len(times))

		// Print statistics
		t.Logf("Validation Performance Statistics:")
		t.Logf("  Total goroutines: %d", concurrentNum)
		t.Logf("  Average time: %v", avgTime)
		t.Logf("  Minimum time: %v", minTime)
		t.Logf("  Maximum time: %v", maxTime)
	})

}

func TestLargeSchemaHandling(t *testing.T) {
	// Test large schema handling
	t.Run("Large schema handling", func(t *testing.T) {
		validator := newSchemaValidator()
		// Create a large schema with many properties
		properties := make([]string, 10000)
		for i := 0; i < 10000; i++ {
			properties[i] = fmt.Sprintf(`"prop%d": {"type": "string"}`, i)
		}

		largeSchema := fmt.Sprintf(`{
			"type": "object",
			"properties": {
				%s
			}
		}`, strings.Join(properties, ",\n"))

		err := validator.Validate(largeSchema)
		if err == nil {
			t.Error("Expected error for large schema")
		} else {
			errLower := strings.ToLower(err.Error())
			if !strings.Contains(errLower, "schema exceeds maximum allowed size") &&
				!strings.Contains(errLower, "exceeds maximum") {
				t.Errorf("Expected error about schema size, got: %v", err)
			}
		}
	})
}

func TestEnforcerCases(t *testing.T) {
	validCases := []string{
		`{"type": "object"}`,
		`{}`,
		`{"type": "object", "properties": {"\\uFACD": {"type": "string"}, "\ufacd": {"type": "string"}, "\\nxxx\\n\\t\\r\\/\\\\\\b\\f": {"type": "string"}}, "required": ["\\uFACD", "\ufacd", "\\nxxx\\n\\t\\r\\/\\\\\\b\\f"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"\\uFACD": {"type": "string"}, "\ufacd": {"type": "string"}, "\\nxxx\\n\\t\\r\\/\\\\\\b\\f": {"type": "string"}}, "required": ["\\uFACD", "\ufacd", "\\nxxx\\n\\t\\r\\/\\\\\\b\\f"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"first name": {"type": "string"}, "last_name": {"type": "string"}, "year_of_birth": {"type": "integer"}, "year_of_nba": {"type": "integer"}, "num_seasons_in_nba": {"type": "integer"}, "extra key": {"type": "string"}}, "required": ["first name", "last_name", "year_of_birth", "year_of_nba", "num_seasons_in_nba"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"first name": {"type": "string"}, "last_name": {"type": "string"}, "year_of_birth": {"type": "integer"}, "year_of_nba": {"type": "integer"}, "num_seasons_in_nba": {"type": "integer"}, "extra key": {"type": "string"}}, "required": ["first name", "last_name", "year_of_birth", "year_of_nba", "num_seasons_in_nba"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"AAA": {"type": "string"}, "BBB": {"type": "string"}}, "required": ["AAA", "BBB"]}`,
		`{"type": "object", "properties": {"AAA": {"type": "string"}, "BBB": {"type": "string"}}, "required": ["AAA", "BBB"]}`,
		`{"type": "object", "properties": {"AAA": {"type": "string"}, "BBB": {"type": "string"}, "x": {"type": "string"}}, "required": ["AAA", "BBB", "x"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"AAA": {"type": "string"}, "BBB": {"type": "string"}, "x": {"type": "string"}}, "required": ["AAA", "BBB", "x"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"AAA": {"type": "string"}, "BBB": {"type": "string"}, "x": {"type": "string"}}, "required": ["AAA", "BBB"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"AAA": {"type": "string"}, "BBB": {"type": "string"}, "x": {"type": "string"}}, "required": ["AAA", "BBB"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"first name": {"type": "string"}, "last_name": {"type": "string"}, "year_of_birth": {"type": "integer"}, "num_seasons_in_nba": {"type": "integer"}}, "additionalProperties": false}`,
		`{"type": "object", "properties": {"first name": {"type": "string"}, "last_name": {"type": "string"}, "year_of_birth": {"type": "integer"}, "num_seasons_in_nba": {"type": "integer"}}, "required": ["first name", "last_name", "num_seasons_in_nba"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"a": {"type": "string"}, "abcd": {"type": "string"}, "abc": {"type": "string"}, "abcde": {"type": "string"}, "abc\ud83d\ude0a": {"type": "string"}, "\ud83d\ude0aabc": {"type": "string"}}, "required": ["a", "abcd", "abc", "abcde", "abc\ud83d\ude0a"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"first/name/A": {"type": "string"}, "XXX/YYY/ZZZ/GGG/HHHH": {"type": "string"}}, "additionalProperties": false}`,
		`{"type": "object", "properties": {"first/name/A": {"type": "string"}, "XXX/YYY/ZZZ/GGG/HHHH": {"type": "string"}}, "required": ["first/name/A", "XXX/YYY/ZZZ/GGG/HHHH"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"first name": {"type": "string"}, "last_name": {"type": "string"}, "year_of_birth": {"type": "integer"}, "num_seasons_in_nba": {"type": "integer"}}, "required": ["first name", "last_name", "year_of_birth"], "additionalProperties": false}`,
		`{"type": "string"}`,
		`{"type": "object", "properties": {"location": {"type": "string"}, "unit": {"type": ["string", "null"]}, "extra": {"type": "string"}}, "required": ["location", "unit"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"location": {"type": "string"}, "unit": {"type": ["string", "null", "number", "boolean", "object"]}, "extra": {"type": "string"}}, "required": ["location", "unit"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"xxx": {"type": ["boolean", "null"], "enum": [true]}}, "required": ["xxx"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"value": {"type": ["string", "number", "null"]}, "items": {"type": "array", "items": {"type": ["string", "boolean", "number", "object", "null"]}}}, "required": ["value", "items"]}`,
		`{"type": "object", "properties": {" ": {"type": "string"}}, "required": [" "], "additionalProperties": false}`,
		`{"type": "object", "properties": {" H": {"type": "string"}}, "required": [" H"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"\ud83e\uddd1\ud83c\udffe\u200d\ud83e\udd1d\u200d\ud83e\uddd1\ud83c\udffe": {"type": "string"}}, "required": ["\ud83e\uddd1\ud83c\udffe\u200d\ud83e\udd1d\u200d\ud83e\uddd1\ud83c\udffe"], "additionalProperties": false}`,
		`{"type": "object", "properties": {" 12\ud83d\ude0ad\ud83e\uddd1\ud83c\udffe\u200d\ud83e\udd1d\u200d\ud83e\uddd1\ud83c\udffeabc\ud83d\ude0a": {"type": "string"}}, "required": [" 12\ud83d\ude0ad\ud83e\uddd1\ud83c\udffe\u200d\ud83e\udd1d\u200d\ud83e\uddd1\ud83c\udffeabc\ud83d\ude0a"], "additionalProperties": false}`,
		`{"type": "object", "properties": {" \u3111\u3127\u3121\u02ca \u310b\u3127\u3121\u7287 \u730b \u9a89 \u87f2 \u9ea4 \u6bf3 \u6dfc \u63b1 \u7131\u579a \u714a \u70dc \u70dc \u7150 \u7113 \u70d3 \u713a \u709c \u70d3 abc123@": {"type": "string"}}, "required": [" \u3111\u3127\u3121\u02ca \u310b\u3127\u3121\u7287 \u730b \u9a89 \u87f2 \u9ea4 \u6bf3 \u6dfc \u63b1 \u7131\u579a \u714a \u70dc \u70dc \u7150 \u7113 \u70d3 \u713a \u709c \u70d3 abc123@"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"3.55": {"type": "number"}}, "required": ["3.55"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"111": {"type": "integer"}}, "required": ["111"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"poseType": {"type": "integer", "enum": [-2, -1, 0, 1, 2, -22, -2222, -3]}}, "required": ["poseType"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"name": {"type": "string"}, "price": {"type": "integer"}, "location": {"type": "string", "enum": ["beijing", "shanghai", "shenzheng"]}}, "required": ["name", "location"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"name": {"type": "string"}, "price": {"type": "integer"}, "attr": {"type": "string", "enum": ["3.1", " a123\ud83e\uddd1\ud83c\udffe\u200d\ud83e\udd1d\u200d\ud83e\uddd1\ud83c\udffeb", "200", "1", ""]}}, "required": ["name", "attr"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"name": {"type": "string"}, "price": {"type": "integer"}, "age": {"type": "number", "enum": [100, 200, 666666666, 3.333333, 3.4444444444444446]}}, "required": ["name", "age"], "additionalProperties": false}`,
		`{"type": "string", "enum": ["red", "green", "blue"]}`,
		`{"type": "number", "enum": [2, 45, 100, 3.33]}`,
		`{"type": "object", "properties": {"name": {"type": "string", "enum": ["Eggs", ""]}, "price": {"type": "integer"}, "attr": {"type": "string", "enum": ["3.1", "", " a123\ud83e\uddd1\ud83c\udffe\u200d\ud83e\udd1d\u200d\ud83e\uddd1\ud83c\udffeb", "200"]}}, "required": ["name", "attr"], "additionalProperties": false}`,
		`{"type": "string", "enum": [" true", "\\ntrue", "  false", "\\rnull", "\\ttrue", "  null", "  true", "  false", "\\r", "\\n", "\\t", " ", "  ", " a", " they", " \\n\\r\\t ", "\\", "\\t\\n\\r"]}`,
		`{"type": "boolean", "enum": [true, false]}`,
		`{"type": "null", "enum": [null]}`,
		`{"type": ["string", "number"]}`,
		`{"type": "boolean"}`,
		`{"type": "null"}`,
		`{"type": "object", "additionalProperties": true}`,
		`{"type": "object", "additionalProperties": false}`,
		`{"type": "object", "properties": {"key1": {"type": "string"}}, "additionalProperties": false}`,
		`{"type": "object", "additionalProperties": {}}`,
		`{"type": "object", "properties": {"x": {"type": "string"}}, "additionalProperties": {"type": "integer"}, "required": ["x"]}`,
		`{"type": "object", "properties": {"x": {"type": "string"}}, "additionalProperties": {"type": "integer"}}`,
		`{"type": "object", "properties": {"xxx": {"type": "string"}}, "additionalProperties": {"type": "integer"}}`,
		`{"type": "object", "additionalProperties": {"type": "string", "enum": ["hello", "hi"]}}`,
		`{"type": "object", "properties": {"builtin": {"type": "number"}}, "additionalProperties": {"type": "string"}}`,
		`{"type": "object", "properties": {"builtin": {"type": "number"}}, "additionalProperties": {"type": "string"}}`,
		`{"type": "object", "properties": {"builtin": {"type": "number"}}, "additionalProperties": {"type": "string"}}`,
		`{"type": "object", "properties": {"builtin": {"type": "number"}}, "additionalProperties": {"type": "string"}}`,
		`{"type": "object", "properties": {"builtin": {"type": "number"}}, "required": ["builtin"], "additionalProperties": {"type": "string"}}`,
		`{"type": "object", "additionalProperties": {"type": "object", "additionalProperties": {"type": "string"}}}`,
		`{"type": "object", "additionalProperties": {"type": "object", "additionalProperties": {"type": "string", "enum": ["hello", "hi"]}}}`,
		`{"type": "object", "properties": {"builtin": {"type": "string"}}, "required": ["builtin"], "additionalProperties": {"type": "object", "additionalProperties": {"type": "string"}}}`,
		`{"type": "object", "additionalProperties": {"type": "string"}}`,
		`{"type": "object", "properties": {"age": {"type": "number"}}, "additionalProperties": {"type": "object", "properties": {"data": {"type": "string"}}, "additionalProperties": true}}`,
		`{"type": "object", "properties": {"age": {"type": "number"}}, "additionalProperties": {"type": "object", "additionalProperties": true}}`,
		`{"type": "object", "additionalProperties": {"type": "object", "additionalProperties": true}}`,
		`{"type": "object", "additionalProperties": {"type": "object", "additionalProperties": false}}`,
		`{"type": "object", "properties": {"data": {"type": "integer"}}, "additionalProperties": {"type": "object", "properties": {"data": {"type": "string"}}, "additionalProperties": true}}`,
		`{"type": "object", "properties": {"builtin": {"type": "number"}}, "additionalProperties": {"type": "string"}}`,
		`{"type": "object", "properties": {"age": {"type": "number"}}, "additionalProperties": {"type": "object", "properties": {"data": {"type": "string"}}}}`,
		`{"type": "object", "properties": {"color": {"type": "array", "items": {"type": "string"}}}, "required": ["color"], "additionalProperties": false}`,
		`{"type": "array", "items": {"anyOf": [{"type": "string"}, {"type": "integer"}]}}`,
		`{"type": "array", "items": {"anyOf": [{"type": "object", "properties": {"name": {"type": "string"}, "price": {"type": "number"}}, "required": ["name", "price"], "additionalProperties": false}, {"type": "object", "properties": {"model": {"type": "string"}, "weight": {"type": "number"}}, "required": ["model", "weight"], "additionalProperties": false}]}}`,
		`{"type": "object", "properties": {"color": {"type": "array", "minItems": 2, "items": {"type": "string"}}}, "required": ["color"], "additionalProperties": false}`,
		`{"type": "array", "items": {"type": "string"}}`,
		`{"type": "object", "properties": {"color": {"anyOf": [{"type": "string", "enum": ["red"]}, {"type": "string", "enum": ["green"]}, {"type": "string", "enum": ["blue"]}]}}, "required": ["color"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"shape": {"anyOf": [{"type": "object", "properties": {"circle": {"type": "object", "properties": {"radius": {"type": "number"}}, "required": ["radius"], "additionalProperties": false}}, "required": ["circle"], "additionalProperties": false}, {"type": "object", "properties": {"rectangle": {"type": "object", "properties": {"width": {"type": "number"}, "height": {"type": "number"}}, "required": ["width", "height"], "additionalProperties": false}}, "required": ["rectangle"], "additionalProperties": false}]}}, "required": ["shape"], "additionalProperties": false}`,
		`{"anyOf": [{"type": "array", "items": {"type": "string"}}, {"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "integer"}}, "required": ["name", "age"], "additionalProperties": false}]}`,
		`{"type": "object", "properties": {"data": {"anyOf": [{"type": "array", "items": {"type": "string"}}, {"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "integer"}}, "required": ["name", "age"], "additionalProperties": false}]}}, "required": ["data"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"data": {"anyOf": [{"type": "array", "items": {"type": "string"}}, {"type": "object", "properties": {"name": {"type": "string"}, "age": {"type": "integer"}, "info": {"anyOf": [{"type": "string"}, {"type": "integer"}]}}, "required": ["name", "age"], "additionalProperties": false}]}}, "required": ["data"], "additionalProperties": false}`,
		`{"anyOf": [{"type": "object", "properties": {"foo": {"type": "string"}}, "required": ["foo"], "additionalProperties": false}, {"type": "object", "properties": {"bar": {"type": "number"}}, "required": ["bar"], "additionalProperties": false}]}`,
		`{"type": "object", "properties": {"qs": {"anyOf": [{"type": "string"}, {"type": "array", "items": {"type": "string"}}], "description": "place your query or queries here"}, "limit": {"type": "integer"}, "lang": {"type": "string", "enum": ["Chinese", "English"]}}, "additionalProperties": false, "required": ["qs"]}`,
		`{"description": "A product from Acme's catalog", "type": "object", "properties": {"productId": {"description": "The unique identifier for a product", "type": "integer"}, "productName": {"description": "Name of the product", "type": "string"}, "price": {"description": "The price of the product", "type": "number"}, "tags": {"description": "Tags for the product", "type": "array", "items": {"type": "string"}, "minItems": 1}, "dimensions": {"type": "object", "properties": {"length": {"type": "object", "properties": {"stride": {"type": "number"}, "raw_length": {"type": "number"}}, "required": ["stride", "raw_length"], "additionalProperties": false}, "width": {"type": "number"}, "height": {"type": "number"}}, "required": ["length", "width", "height"], "additionalProperties": false}}, "required": ["productId", "productName", "price"], "additionalProperties": false}`,
		`{"type": "object", "properties": {"member": {"type": "object", "properties": {"phone": {"type": "string"}, "addr": {"type": "object", "properties": {"country": {"type": "string"}, "street": {"type": "string"}}, "additionalProperties": false}}, "required": ["phone", "addr"], "additionalProperties": false}, "name": {"type": "string"}}, "required": ["member", "name"], "additionalProperties": false}`,
		`{"type": "string", "description": "A product from Acme's catalog"}`,
		`{"$defs": {"build": {"description": "", "properties": {"bundle": {"type": "boolean"}, "esBuildPath": {"type": "string"}, "esbuildVersion": {"type": "string"}, "externals": {"type": "array"}, "injects": {"type": "array"}, "jsxFactory": {"type": "string"}, "jsxFragment": {"type": "string"}, "minify": {"type": "boolean"}, "outDir": {"type": "string"}, "target": {"type": "string"}}, "type": "object"}, "devServer": {"description": "", "properties": {"autoStart": {"type": "boolean"}, "liveReload": {"type": "boolean"}, "mountDirectories": {"type": "object"}, "port": {"type": "integer"}, "useSSL": {"type": "boolean"}, "watchConfig": {"properties": {"directories": {"type": "array"}, "extensions": {"type": "array"}}, "type": "object"}}, "type": "object"}, "fable": {"properties": {"autoStart": {"type": "boolean"}, "extension": {"type": "string"}, "outDir": {"type": "string"}, "project": {"type": "string"}}, "type": "object"}}, "properties": {"build": {"$ref": "#/$defs/build"}, "devServer": {"$ref": "#/$defs/devServer"}, "fable": {"$ref": "#/$defs/fable"}, "index": {"type": "string"}, "packages": {"type": "object"}}, "required": ["index"], "title": "Perla", "type": "object"}`,
		`{"anyOf": [{"type": "array", "items": {"anyOf": [{"type": "array", "items": {"anyOf": [{"type": "object"}, {"type": "string"}]}}]}}, {"type": "array", "items": {"anyOf": [{"type": "array", "items": {"type": "number", "enum": [59, 62, 65]}}]}}]}`,
		`{"anyOf": [{"type": "array", "items": {"type": "boolean"}}, {"type": "array", "items": {"type": "array", "items": {"type": "integer", "enum": [1, 2, 3]}}}]}`,
		`{"anyOf": [{"type": "array", "items": {"anyOf": [{"type": "array", "items": {"anyOf": [{"type": "object"}, {"type": "string"}]}}]}}]}`,
		`{"type": "array", "items": {"type": "array", "items": {"type": "integer", "enum": [1, 2, 3]}}}`,
		`{"type": "array", "items": {"anyOf": [{"type": "number", "enum": [-73]}, {"type": "number", "enum": [-2.2250738585072014]}, {"type": "integer", "maximum": 33333}]}, "minItems": 0, "maxItems": 4}`,
		`{"type": "object", "properties": {"steps": {"type": "array", "items": {"$ref": "#/$defs/step"}}, "final_answer": {"type": "string"}}, "$defs": {"step": {"type": "object", "properties": {"explanation": {"type": "string"}, "output": {"type": "string"}}, "required": ["explanation", "output"], "additionalProperties": false}}, "required": ["steps", "final_answer"], "additionalProperties": false}`,
		`{"type": "object", "$defs": {"ParentObject": {"type": "object", "properties": {"child": {"type": "string"}}}, "Addr": {"type": "object", "required": ["country", "city", "street"], "properties": {"country": {"type": "string"}, "city": {"type": "string"}, "street": {"type": "string"}}}}, "properties": {"parent": {"$ref": "#/$defs/ParentObject"}, "address": {"$ref": "#/$defs/Addr"}}}`,
		`{"type": "object", "$defs": {"contact": {"type": "object", "properties": {"email": {"type": "string"}, "phone": {"type": "string"}}, "required": ["email"]}, "address": {"type": "object", "properties": {"street": {"type": "string"}, "city": {"type": "string"}, "postal": {"$ref": "#/$defs/postal"}}, "required": ["street", "city"]}, "postal": {"type": "object", "properties": {"code": {"type": "string"}, "country": {"type": "string"}}, "required": ["code", "country"]}, "person": {"type": "object", "properties": {"name": {"type": "string"}, "contact": {"$ref": "#/$defs/contact"}, "address": {"$ref": "#/$defs/address"}}, "required": ["name", "contact", "address"]}}, "properties": {"employee": {"$ref": "#/$defs/person"}}, "required": ["employee"]}`,
		`{"type": "object", "properties": {"type_info": {"type": "string", "description": "The type of the UI component", "enum": ["div", "button", "header", "section", "field", "form"]}, "label": {"type": "string", "description": "The label of the UI component, used for buttons or form fields"}, "children": {"type": "array", "description": "Nested UI components", "items": {"$ref": "#"}}, "attributes": {"type": "array", "description": "Arbitrary attributes for the UI component, suitable for any element", "items": {"type": "object", "properties": {"name": {"type": "string", "description": "The name of the attribute, for example onClick or className"}, "value": {"type": "string", "description": "The value of the attribute"}}, "additionalProperties": false, "required": ["name", "value"]}}}, "required": ["type_info", "label", "children", "attributes"], "additionalProperties": false}`,
		`{"$defs": {"mySchema": {"type": "object", "properties": {"name": {"type": "string"}, "child": {"$ref": "#/$defs/mySchema"}}, "additionalProperties": false}}, "type": "object", "properties": {"root": {"$ref": "#/$defs/mySchema"}}}`,
		`{"type": "object", "properties": {"linked_list": {"$ref": "#/$defs/linked_list_node"}}, "$defs": {"linked_list_node": {"type": "object", "properties": {"value": {"type": "number"}, "next": {"anyOf": [{"$ref": "#/$defs/linked_list_node"}, {"type": "null"}]}}, "additionalProperties": false, "required": ["value", "next"]}}, "additionalProperties": false, "required": ["linked_list"]}`,
		`{"type": "object", "properties": {"name": {"$ref": "#/$defs/mySchema"}}, "$defs": {"mySchema": {"description": "mySchema", "$ref": "#/$defs/mySchema2"}, "mySchema2": {"title": "mySchema2", "$ref": "#/$defs/mySchema3"}, "mySchema3": {"type": "number"}}}`,
		`{"anyOf": [{"$ref": "#/$defs/a"}, {"$ref": "#/$defs/b"}], "$defs": {"a": {"$ref": "#/$defs/c"}, "b": {"$ref": "#/$defs/c"}, "c": {"type": "string"}}}`,
		`{"type": "object", "properties": {"name": {"anyOf": [{"$ref": "#/$defs/a"}, {"$ref": "#/$defs/b"}]}, "address": {"anyOf": [{"$ref": "#/$defs/a"}, {"$ref": "#/$defs/b"}]}}, "$defs": {"a": {"$ref": "#/$defs/c"}, "b": {"$ref": "#/$defs/c"}, "c": {"type": "string"}}}`,
	}

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		{
			`{"type": "function", "function": {"name": "memo", "description": "remember anything, in key-value manner", "parameters": {"type": "object", "additionalProperties": {"type": "string"}}}}`,
			`unsupported keywords: function`,
			false,
		},
		{
			`{"type": "object", "additionalProperties": true, "properties": {"key1": {"type": "boolean"}, "key2": {"type": "array"}}, "patternProperties": {"Key[0-9]*": {"type": "string"}}}`,
			`patternProperties`,
			false,
		},
		{
			`{"type": "object", "additionalProperties": {"type": "number"}, "properties": {"key1": {"type": "boolean"}, "key2": {"type": "array"}}, "patternProperties": {"Key3[0-9]*": {"type": "string"}}}`,
			`patternProperties`,
			false,
		},
		{
			`{"type": "array", "items": {"type": "object", "anyOf": [{"type": "string"}, {"type": "integer"}]}}`,
			`type should be defined in anyOf items`,
			false,
		},
		{
			`{"type": "object", "additionalProperties": true, "properties": {"key1": {"type": "boolean"}, "key2": {"type": "array"}}, "patternProperties": {"key3[0-9]*": {"type": "string"}}}`,
			`patternProperties`,
			false,
		},
		{
			`{"type": "object", "properties": {"builtin": {"type": "number"}}, "patternProperties": {"^S_": {"type": "string"}, "^I_": {"type": "integer"}}, "additionalProperties": {"type": "string"}}`,
			`patternProperties`,
			false,
		},
		{
			`{"type": "array", "items": {"type": "number", "anyOf": [{"enum": [-73]}, {"enum": [-2.2250738585072014e-308]}, {"maximum": 33333}]}, "minItems": 0, "maxItems": 4}`,
			`when using anyOf, type should be defined in anyOf`,
			false,
		},
		{
			`{"enum": [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126, 127, 128, 129, 130, 131, 132, 133, 134, 135, 136, 137, 138, 139, 140, 141, 142, 143, 144, 145, 146, 147, 148, 149, 150, 151, 152, 153, 154, 155, 156, 157, 158, 159, 160, 161, 162, 163, 164, 165, 166, 167, 168, 169, 170, 171, 172, 173, 174, 175, 176, 177, 178, 179, 180, 181, 182, 183, 184, 185, 186, 187, 188, 189, 190, 191, 192, 193, 194, 195, 196, 197, 198, 199, 200, 201, 202, 203, 204, 205, 206, 207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225, 226, 227, 228, 229, 230, 231, 232, 233, 234, 235, 236, 237, 238, 239, 240, 241, 242, 243, 244, 245, 246, 247, 248, 249, 250, 251, 252, 253, 254, 255, 256, 257, 258, 259, 260, 261, 262, 263, 264, 265, 266, 267, 268, 269, 270, 271, 272, 273, 274, 275, 276, 277, 278, 279, 280, 281, 282, 283, 284, 285, 286, 287, 288, 289, 290, 291, 292, 293, 294, 295, 296, 297, 298, 299, 300, 301, 302, 303, 304, 305, 306, 307, 308, 309, 310, 311, 312, 313, 314, 315, 316, 317, 318, 319, 320, 321, 322, 323, 324, 325, 326, 327, 328, 329, 330, 331, 332, 333, 334, 335, 336, 337, 338, 339, 340, 341, 342, 343, 344, 345, 346, 347, 348, 349, 350, 351, 352, 353, 354, 355, 356, 357, 358, 359, 360, 361, 362, 363, 364, 365, 366, 367, 368, 369, 370, 371, 372, 373, 374, 375, 376, 377, 378, 379, 380, 381, 382, 383, 384, 385, 386, 387, 388, 389, 390, 391, 392, 393, 394, 395, 396, 397, 398, 399, 400, 401, 402, 403, 404, 405, 406, 407, 408, 409, 410, 411, 412, 413, 414, 415, 416, 417, 418, 419, 420, 421, 422, 423, 424, 425, 426, 427, 428, 429, 430, 431, 432, 433, 434, 435, 436, 437, 438, 439, 440, 441, 442, 443, 444, 445, 446, 447, 448, 449, 450, 451, 452, 453, 454, 455, 456, 457, 458, 459, 460, 461, 462, 463, 464, 465, 466, 467, 468, 469, 470, 471, 472, 473, 474, 475, 476, 477, 478, 479, 480, 481, 482, 483, 484, 485, 486, 487, 488, 489, 490, 491, 492, 493, 494, 495, 496, 497, 498, 499, 500], "type": "number"}`,
			`enum array cannot have more than 500 items`,
			false,
		},
		{
			`{"enum": ["fontawesome/brands/42-group", "fontawesome/brands/500px", "fontawesome/brands/accessible-icon", "fontawesome/brands/accusoft", "fontawesome/brands/adn", "fontawesome/brands/adversal", "fontawesome/brands/affiliatetheme", "fontawesome/brands/airbnb", "fontawesome/brands/algolia", "fontawesome/brands/alipay", "fontawesome/brands/amazon-pay", "fontawesome/brands/amazon", "fontawesome/brands/amilia", "fontawesome/brands/android", "fontawesome/brands/angellist", "fontawesome/brands/angrycreative", "fontawesome/brands/angular", "fontawesome/brands/app-store-ios", "fontawesome/brands/app-store", "fontawesome/brands/apper", "fontawesome/brands/apple-pay", "fontawesome/brands/apple", "fontawesome/brands/artstation", "fontawesome/brands/asymmetrik", "fontawesome/brands/atlassian", "fontawesome/brands/audible", "fontawesome/brands/autoprefixer", "fontawesome/brands/avianex", "fontawesome/brands/aviato", "fontawesome/brands/aws", "fontawesome/brands/bandcamp", "fontawesome/brands/battle-net", "fontawesome/brands/behance", "fontawesome/brands/bilibili", "fontawesome/brands/bimobject", "fontawesome/brands/bitbucket", "fontawesome/brands/bitcoin", "fontawesome/brands/bity", "fontawesome/brands/black-tie", "fontawesome/brands/blackberry", "fontawesome/brands/blogger-b", "fontawesome/brands/blogger", "fontawesome/brands/bluetooth-b", "fontawesome/brands/bluetooth", "fontawesome/brands/bootstrap", "fontawesome/brands/bots", "fontawesome/brands/btc", "fontawesome/brands/buffer", "fontawesome/brands/buromobelexperte", "fontawesome/brands/buy-n-large", "fontawesome/brands/buysellads", "fontawesome/brands/canadian-maple-leaf", "fontawesome/brands/cc-amazon-pay", "fontawesome/brands/cc-amex", "fontawesome/brands/cc-apple-pay", "fontawesome/brands/cc-diners-club", "fontawesome/brands/cc-discover", "fontawesome/brands/cc-jcb", "fontawesome/brands/cc-mastercard", "fontawesome/brands/cc-paypal", "fontawesome/brands/cc-stripe", "fontawesome/brands/cc-visa", "fontawesome/brands/centercode", "fontawesome/brands/centos", "fontawesome/brands/chrome", "fontawesome/brands/chromecast", "fontawesome/brands/cloudflare", "fontawesome/brands/cloudscale", "fontawesome/brands/cloudsmith", "fontawesome/brands/cloudversify", "fontawesome/brands/cmplid", "fontawesome/brands/codepen", "fontawesome/brands/codiepie", "fontawesome/brands/confluence", "fontawesome/brands/connectdevelop", "fontawesome/brands/contao", "fontawesome/brands/cotton-bureau", "fontawesome/brands/cpanel", "fontawesome/brands/creative-commons-by", "fontawesome/brands/creative-commons-nc-eu", "fontawesome/brands/creative-commons-nc-jp", "fontawesome/brands/creative-commons-nc", "fontawesome/brands/creative-commons-nd", "fontawesome/brands/creative-commons-pd-alt", "fontawesome/brands/creative-commons-pd", "fontawesome/brands/creative-commons-remix", "fontawesome/brands/creative-commons-sa", "fontawesome/brands/creative-commons-sampling-plus", "fontawesome/brands/creative-commons-sampling", "fontawesome/brands/creative-commons-share", "fontawesome/brands/creative-commons-zero", "fontawesome/brands/creative-commons", "fontawesome/brands/critical-role", "fontawesome/brands/css3-alt", "fontawesome/brands/css3", "fontawesome/brands/cuttlefish", "fontawesome/brands/d-and-d-beyond", "fontawesome/brands/d-and-d", "fontawesome/brands/dailymotion", "fontawesome/brands/dashcube", "fontawesome/brands/debian", "fontawesome/brands/deezer", "fontawesome/brands/delicious", "fontawesome/brands/deploydog", "fontawesome/brands/deskpro", "fontawesome/brands/dev", "fontawesome/brands/deviantart", "fontawesome/brands/dhl", "fontawesome/brands/diaspora", "fontawesome/brands/digg", "fontawesome/brands/digital-ocean", "fontawesome/brands/discord", "fontawesome/brands/discourse", "fontawesome/brands/dochub", "fontawesome/brands/docker", "fontawesome/brands/draft2digital", "fontawesome/brands/dribbble", "fontawesome/brands/dropbox", "fontawesome/brands/drupal", "fontawesome/brands/dyalog", "fontawesome/brands/earlybirds", "fontawesome/brands/ebay", "fontawesome/brands/edge-legacy", "fontawesome/brands/edge", "fontawesome/brands/elementor", "fontawesome/brands/ello", "fontawesome/brands/ember", "fontawesome/brands/empire", "fontawesome/brands/envira", "fontawesome/brands/erlang", "fontawesome/brands/ethereum", "fontawesome/brands/etsy", "fontawesome/brands/evernote", "fontawesome/brands/expeditedssl", "fontawesome/brands/facebook-f", "fontawesome/brands/facebook-messenger", "fontawesome/brands/facebook", "fontawesome/brands/fantasy-flight-games", "fontawesome/brands/fedex", "fontawesome/brands/fedora", "fontawesome/brands/figma", "fontawesome/brands/firefox-browser", "fontawesome/brands/firefox", "fontawesome/brands/first-order-alt", "fontawesome/brands/first-order", "fontawesome/brands/firstdraft", "fontawesome/brands/flickr", "fontawesome/brands/flipboard", "fontawesome/brands/fly", "fontawesome/brands/font-awesome", "fontawesome/brands/fonticons-fi", "fontawesome/brands/fonticons", "fontawesome/brands/fort-awesome-alt", "fontawesome/brands/fort-awesome", "fontawesome/brands/forumbee", "fontawesome/brands/foursquare", "fontawesome/brands/free-code-camp", "fontawesome/brands/freebsd", "fontawesome/brands/fulcrum", "fontawesome/brands/galactic-republic", "fontawesome/brands/galactic-senate", "fontawesome/brands/get-pocket", "fontawesome/brands/gg-circle", "fontawesome/brands/gg", "fontawesome/brands/git-alt", "fontawesome/brands/git", "fontawesome/brands/github-alt", "fontawesome/brands/github", "fontawesome/brands/gitkraken", "fontawesome/brands/gitlab", "fontawesome/brands/gitter", "fontawesome/brands/glide-g", "fontawesome/brands/glide", "fontawesome/brands/gofore", "fontawesome/brands/golang", "fontawesome/brands/goodreads-g", "fontawesome/brands/goodreads", "fontawesome/brands/google-drive", "fontawesome/brands/google-pay", "fontawesome/brands/google-play", "fontawesome/brands/google-plus-g", "fontawesome/brands/google-plus", "fontawesome/brands/google-wallet", "fontawesome/brands/google", "fontawesome/brands/gratipay", "fontawesome/brands/grav", "fontawesome/brands/gripfire", "fontawesome/brands/grunt", "fontawesome/brands/guilded", "fontawesome/brands/gulp", "fontawesome/brands/hacker-news", "fontawesome/brands/hackerrank", "fontawesome/brands/hashnode", "fontawesome/brands/hips", "fontawesome/brands/hire-a-helper", "fontawesome/brands/hive", "fontawesome/brands/hooli", "fontawesome/brands/hornbill", "fontawesome/brands/hotjar", "fontawesome/brands/houzz", "fontawesome/brands/html5", "fontawesome/brands/hubspot", "fontawesome/brands/ideal", "fontawesome/brands/imdb", "fontawesome/brands/instagram", "fontawesome/brands/instalod", "fontawesome/brands/intercom", "fontawesome/brands/internet-explorer", "fontawesome/brands/invision", "fontawesome/brands/ioxhost", "fontawesome/brands/itch-io", "fontawesome/brands/itunes-note", "fontawesome/brands/itunes", "fontawesome/brands/java", "fontawesome/brands/jedi-order", "fontawesome/brands/jenkins", "fontawesome/brands/jira", "fontawesome/brands/joget", "fontawesome/brands/joomla", "fontawesome/brands/js", "fontawesome/brands/jsfiddle", "fontawesome/brands/kaggle", "fontawesome/brands/keybase", "fontawesome/brands/keycdn", "fontawesome/brands/kickstarter-k", "fontawesome/brands/kickstarter", "fontawesome/brands/korvue", "fontawesome/brands/laravel", "fontawesome/brands/lastfm", "fontawesome/brands/leanpub", "fontawesome/brands/less", "fontawesome/brands/line", "fontawesome/brands/linkedin-in", "fontawesome/brands/linkedin", "fontawesome/brands/linode", "fontawesome/brands/linux", "fontawesome/brands/lyft", "fontawesome/brands/magento", "fontawesome/brands/mailchimp", "fontawesome/brands/mandalorian", "fontawesome/brands/markdown", "fontawesome/brands/mastodon", "fontawesome/brands/maxcdn", "fontawesome/brands/mdb", "fontawesome/brands/medapps", "fontawesome/brands/medium", "fontawesome/brands/medrt", "fontawesome/brands/meetup", "fontawesome/brands/megaport", "fontawesome/brands/mendeley", "fontawesome/brands/meta", "fontawesome/brands/microblog", "fontawesome/brands/microsoft", "fontawesome/brands/mix", "fontawesome/brands/mixcloud", "fontawesome/brands/mixer", "fontawesome/brands/mizuni", "fontawesome/brands/modx", "fontawesome/brands/monero", "fontawesome/brands/napster", "fontawesome/brands/neos", "fontawesome/brands/nfc-directional", "fontawesome/brands/nfc-symbol", "fontawesome/brands/nimblr", "fontawesome/brands/node-js", "fontawesome/brands/node", "fontawesome/brands/npm", "fontawesome/brands/ns8", "fontawesome/brands/nutritionix", "fontawesome/brands/octopus-deploy", "fontawesome/brands/odnoklassniki", "fontawesome/brands/odysee", "fontawesome/brands/old-republic", "fontawesome/brands/opencart", "fontawesome/brands/openid", "fontawesome/brands/opera", "fontawesome/brands/optin-monster", "fontawesome/brands/orcid", "fontawesome/brands/osi", "fontawesome/brands/padlet", "fontawesome/brands/page4", "fontawesome/brands/pagelines", "fontawesome/brands/palfed", "fontawesome/brands/patreon", "fontawesome/brands/paypal", "fontawesome/brands/perbyte", "fontawesome/brands/periscope", "fontawesome/brands/phabricator", "fontawesome/brands/phoenix-framework", "fontawesome/brands/phoenix-squadron", "fontawesome/brands/php", "fontawesome/brands/pied-piper-alt", "fontawesome/brands/pied-piper-hat", "fontawesome/brands/pied-piper-pp", "fontawesome/brands/pied-piper", "fontawesome/brands/pinterest-p", "fontawesome/brands/pinterest", "fontawesome/brands/pix", "fontawesome/brands/playstation", "fontawesome/brands/product-hunt", "fontawesome/brands/pushed", "fontawesome/brands/python", "fontawesome/brands/qq", "fontawesome/brands/quinscape", "fontawesome/brands/quora", "fontawesome/brands/r-project", "fontawesome/brands/raspberry-pi", "fontawesome/brands/ravelry", "fontawesome/brands/react", "fontawesome/brands/reacteurope", "fontawesome/brands/readme", "fontawesome/brands/rebel", "fontawesome/brands/red-river", "fontawesome/brands/reddit-alien", "fontawesome/brands/reddit", "fontawesome/brands/redhat", "fontawesome/brands/renren", "fontawesome/brands/replyd", "fontawesome/brands/researchgate", "fontawesome/brands/resolving", "fontawesome/brands/rev", "fontawesome/brands/rocketchat", "fontawesome/brands/rockrms", "fontawesome/brands/rust", "fontawesome/brands/safari", "fontawesome/brands/salesforce", "fontawesome/brands/sass", "fontawesome/brands/schlix", "fontawesome/brands/screenpal", "fontawesome/brands/scribd", "fontawesome/brands/searchengin", "fontawesome/brands/sellcast", "fontawesome/brands/sellsy", "fontawesome/brands/servicestack", "fontawesome/brands/shirtsinbulk", "fontawesome/brands/shopify", "fontawesome/brands/shopware", "fontawesome/brands/simplybuilt", "fontawesome/brands/sistrix", "fontawesome/brands/sith", "fontawesome/brands/sitrox", "fontawesome/brands/sketch", "fontawesome/brands/skyatlas", "fontawesome/brands/skype", "fontawesome/brands/slack", "fontawesome/brands/slideshare", "fontawesome/brands/snapchat", "fontawesome/brands/soundcloud", "fontawesome/brands/sourcetree", "fontawesome/brands/space-awesome", "fontawesome/brands/speakap", "fontawesome/brands/speaker-deck", "fontawesome/brands/spotify", "fontawesome/brands/square-behance", "fontawesome/brands/square-dribbble", "fontawesome/brands/square-facebook", "fontawesome/brands/square-font-awesome-stroke", "fontawesome/brands/square-font-awesome", "fontawesome/brands/square-git", "fontawesome/brands/square-github", "fontawesome/brands/square-gitlab", "fontawesome/brands/square-google-plus", "fontawesome/brands/square-hacker-news", "fontawesome/brands/square-instagram", "fontawesome/brands/square-js", "fontawesome/brands/square-lastfm", "fontawesome/brands/square-odnoklassniki", "fontawesome/brands/square-pied-piper", "fontawesome/brands/square-pinterest", "fontawesome/brands/square-reddit", "fontawesome/brands/square-snapchat", "fontawesome/brands/square-steam", "fontawesome/brands/square-threads", "fontawesome/brands/square-tumblr", "fontawesome/brands/square-twitter", "fontawesome/brands/square-viadeo", "fontawesome/brands/square-vimeo", "fontawesome/brands/square-whatsapp", "fontawesome/brands/square-x-twitter", "fontawesome/brands/square-xing", "fontawesome/brands/square-youtube", "fontawesome/brands/squarespace", "fontawesome/brands/stack-exchange", "fontawesome/brands/stack-overflow", "fontawesome/brands/stackpath", "fontawesome/brands/staylinked", "fontawesome/brands/steam-symbol", "fontawesome/brands/steam", "fontawesome/brands/sticker-mule", "fontawesome/brands/strava", "fontawesome/brands/stripe-s", "fontawesome/brands/stripe", "fontawesome/brands/stubber", "fontawesome/brands/studiovinari", "fontawesome/brands/stumbleupon-circle", "fontawesome/brands/stumbleupon", "fontawesome/brands/superpowers", "fontawesome/brands/supple", "fontawesome/brands/suse", "fontawesome/brands/swift", "fontawesome/brands/symfony", "fontawesome/brands/teamspeak", "fontawesome/brands/telegram", "fontawesome/brands/tencent-weibo", "fontawesome/brands/the-red-yeti", "fontawesome/brands/themeco", "fontawesome/brands/themeisle", "fontawesome/brands/think-peaks", "fontawesome/brands/threads", "fontawesome/brands/tiktok", "fontawesome/brands/trade-federation", "fontawesome/brands/trello", "fontawesome/brands/tumblr", "fontawesome/brands/twitch", "fontawesome/brands/twitter", "fontawesome/brands/typo3", "fontawesome/brands/uber", "fontawesome/brands/ubuntu", "fontawesome/brands/uikit", "fontawesome/brands/umbraco", "fontawesome/brands/uncharted", "fontawesome/brands/uniregistry", "fontawesome/brands/unity", "fontawesome/brands/unsplash", "fontawesome/brands/untappd", "fontawesome/brands/ups", "fontawesome/brands/usb", "fontawesome/brands/usps", "fontawesome/brands/ussunnah", "fontawesome/brands/vaadin", "fontawesome/brands/viacoin", "fontawesome/brands/viadeo", "fontawesome/brands/viber", "fontawesome/brands/vimeo-v", "fontawesome/brands/vimeo", "fontawesome/brands/vine", "fontawesome/brands/vk", "fontawesome/brands/vnv", "fontawesome/brands/vuejs", "fontawesome/brands/watchman-monitoring", "fontawesome/brands/waze", "fontawesome/brands/weebly", "fontawesome/brands/weibo", "fontawesome/brands/weixin", "fontawesome/brands/whatsapp", "fontawesome/brands/whmcs", "fontawesome/brands/wikipedia-w", "fontawesome/brands/windows", "fontawesome/brands/wirsindhandwerk", "fontawesome/brands/wix", "fontawesome/brands/wizards-of-the-coast", "fontawesome/brands/wodu", "fontawesome/brands/wolf-pack-battalion", "fontawesome/brands/wordpress-simple", "fontawesome/brands/wordpress", "fontawesome/brands/wpbeginner", "fontawesome/brands/wpexplorer", "fontawesome/brands/wpforms", "fontawesome/brands/wpressr", "fontawesome/brands/x-twitter", "fontawesome/brands/xbox", "fontawesome/brands/xing", "fontawesome/brands/y-combinator", "fontawesome/brands/yahoo", "fontawesome/brands/yammer", "fontawesome/brands/yandex-international", "fontawesome/brands/yandex", "fontawesome/brands/yarn", "fontawesome/brands/yelp", "fontawesome/brands/yoast", "fontawesome/brands/youtube", "fontawesome/brands/zhihu", "fontawesome/regular/address-book", "fontawesome/regular/address-card", "fontawesome/regular/bell-slash", "fontawesome/regular/bell", "fontawesome/regular/bookmark", "fontawesome/regular/building", "fontawesome/regular/calendar-check", "fontawesome/regular/calendar-days", "fontawesome/regular/calendar-minus", "fontawesome/regular/calendar-plus", "fontawesome/regular/calendar-xmark", "fontawesome/regular/calendar", "fontawesome/regular/chart-bar", "fontawesome/regular/chess-bishop", "fontawesome/regular/chess-king", "fontawesome/regular/chess-knight", "fontawesome/regular/chess-pawn", "fontawesome/regular/chess-queen", "fontawesome/regular/chess-rook", "fontawesome/regular/circle-check", "fontawesome/regular/circle-dot", "fontawesome/regular/circle-down", "fontawesome/regular/circle-left", "fontawesome/regular/circle-pause", "fontawesome/regular/circle-play", "fontawesome/regular/circle-question", "fontawesome/regular/circle-right", "fontawesome/regular/circle-stop", "fontawesome/regular/circle-up", "fontawesome/regular/circle-user", "fontawesome/regular/circle-xmark", "fontawesome/regular/circle", "fontawesome/regular/clipboard", "fontawesome/regular/clock", "fontawesome/regular/clone", "fontawesome/regular/closed-captioning", "fontawesome/regular/comment-dots", "fontawesome/regular/comment", "fontawesome/regular/comments", "fontawesome/regular/compass", "fontawesome/regular/copy", "fontawesome/regular/copyright", "fontawesome/regular/credit-card", "fontawesome/regular/envelope-open", "fontawesome/regular/envelope", "fontawesome/regular/eye-slash", "fontawesome/regular/eye", "fontawesome/regular/face-angry", "fontawesome/regular/face-dizzy", "fontawesome/regular/face-flushed", "fontawesome/regular/face-frown-open", "fontawesome/regular/face-frown", "fontawesome/regular/face-grimace", "fontawesome/regular/face-grin-beam-sweat", "fontawesome/regular/face-grin-beam", "fontawesome/regular/face-grin-hearts", "fontawesome/regular/face-grin-squint-tears", "fontawesome/regular/face-grin-squint", "fontawesome/regular/face-grin-stars", "fontawesome/regular/face-grin-tears", "fontawesome/regular/face-grin-tongue-squint", "fontawesome/regular/face-grin-tongue-wink", "fontawesome/regular/face-grin-tongue", "fontawesome/regular/face-grin-wide", "fontawesome/regular/face-grin-wink", "fontawesome/regular/face-grin", "fontawesome/regular/face-kiss-beam", "fontawesome/regular/face-kiss-wink-heart", "fontawesome/regular/face-kiss", "fontawesome/regular/face-laugh-beam", "fontawesome/regular/face-laugh-squint", "fontawesome/regular/face-laugh-wink", "fontawesome/regular/face-laugh", "fontawesome/regular/face-meh-blank", "fontawesome/regular/face-meh", "fontawesome/regular/face-rolling-eyes", "fontawesome/regular/face-sad-cry", "fontawesome/regular/face-sad-tear", "fontawesome/regular/face-smile-beam", "fontawesome/regular/face-smile-wink", "fontawesome/regular/face-smile", "fontawesome/regular/face-surprise", "fontawesome/regular/face-tired", "fontawesome/regular/file-audio", "fontawesome/regular/file-code", "fontawesome/regular/file-excel", "fontawesome/regular/file-image", "fontawesome/regular/file-lines", "fontawesome/regular/file-pdf", "fontawesome/regular/file-powerpoint", "fontawesome/regular/file-video", "fontawesome/regular/file-word", "fontawesome/regular/file-zipper", "fontawesome/regular/file", "fontawesome/regular/flag", "fontawesome/regular/floppy-disk", "fontawesome/regular/folder-closed", "fontawesome/regular/folder-open", "fontawesome/regular/folder", "fontawesome/regular/font-awesome", "fontawesome/regular/futbol", "fontawesome/regular/gem", "fontawesome/regular/hand-back-fist", "fontawesome/regular/hand-lizard", "fontawesome/regular/hand-peace", "fontawesome/regular/hand-point-down", "fontawesome/regular/hand-point-left", "fontawesome/regular/hand-point-right", "fontawesome/regular/hand-point-up", "fontawesome/regular/hand-pointer", "fontawesome/regular/hand-scissors", "fontawesome/regular/hand-spock", "fontawesome/regular/hand", "fontawesome/regular/handshake", "fontawesome/regular/hard-drive", "fontawesome/regular/heart", "fontawesome/regular/hospital", "fontawesome/regular/hourglass-half", "fontawesome/regular/hourglass", "fontawesome/regular/id-badge", "fontawesome/regular/id-card", "fontawesome/regular/image", "fontawesome/regular/images", "fontawesome/regular/keyboard", "fontawesome/regular/lemon", "fontawesome/regular/life-ring", "fontawesome/regular/lightbulb", "fontawesome/regular/map"], "type": "string"}`,
			`schema exceeds maximum allowed size`,
			false,
		},
	}

	runTestCases(t, validCases, invalidCases)
}

func TestValidateAPI(t *testing.T) {
	t.Run("Validate API", func(t *testing.T) {
		must := require.New(t)
		validator := newSchemaValidator()
		err := validator.Validate(make(map[string]struct{}))
		must.Error(err)
		must.Contains(err.Error(), "input schema must be a string or map")
	})
}

func TestSchemaValidatorWithCustomConfig(t *testing.T) {
	t.Run("Custom configuration through options", func(t *testing.T) {
		must := require.New(t)
		// Create a validator with custom options
		validator := newSchemaValidator(
			WithMaxEnumItems(250),
			WithMaxSchemaDepth(10),
			WithMaxSchemaSize(30000),
		)

		// Check if the config values were applied correctly
		must.Equal(250, validator.config.MaxEnumItems)
		must.Equal(10, validator.config.MaxSchemaDepth)
		must.Equal(30000, validator.config.MaxSchemaSize)

		// Default values for others
		must.Equal(7500, validator.config.MaxEnumStringLength)
		must.Equal(250, validator.config.MaxEnumStringCheckThreshold)
		must.Equal(100, validator.config.MaxAnyOfItems)
		must.Equal(1000, validator.config.MaxTotalPropertiesKeysNum)
	})

	t.Run("MaxEnumStringLength and MaxEnumStringCheckThreshold limit", func(t *testing.T) {
		must := require.New(t)
		validator := newSchemaValidator(
			WithMaxEnumStringCheckThreshold(3),
			WithMaxEnumStringLength(50),
		)

		// Create a schema with multiple enum values and total length exceeding the limit
		longEnumSchema := `{
			"type": "string",
			"enum": [
				"string_value_1",
				"string_value_2", 
				"string_value_3",
				"string_value_4",
				"string_value_5"
			]
    	}`

		// Should fail because total length exceeds the limit
		err := validator.Validate(longEnumSchema)
		must.Error(err)
		must.Contains(err.Error(), "exceeds maximum limit of 50 characters when enum has more than 3 values")

		// Now increase the total length limit but keep the threshold unchanged
		validator = newSchemaValidator(
			WithMaxEnumStringCheckThreshold(3),
			WithMaxEnumStringLength(100),
		)
		err = validator.Validate(longEnumSchema)
		must.NoError(err)

		// Another test: keep low limit but raise threshold to avoid triggering check
		validator = newSchemaValidator(
			WithMaxEnumStringCheckThreshold(10),
			WithMaxEnumStringLength(50),
		)
		err = validator.Validate(longEnumSchema)
		must.NoError(err)
	})

	t.Run("MaxEnumItems limit", func(t *testing.T) {
		must := require.New(t)
		// Create a validator with a low enum item limit
		validator := newSchemaValidator(
			WithMaxEnumItems(3),
		)

		// Schema with more enum items than allowed
		schemaWithLargeEnum := `{
			"type": "string",
			"enum": ["option1", "option2", "option3", "option4", "option5"]
		}`

		// This validation should fail due to too many enum items
		err := validator.Validate(schemaWithLargeEnum)
		must.Error(err)
		must.Contains(err.Error(), "enum array cannot have more than 3 items")

		// Now increase the enum limit and try again
		validator = newSchemaValidator(
			WithMaxEnumItems(5),
		)

		// This should now succeed
		err = validator.Validate(schemaWithLargeEnum)
		must.NoError(err)
	})

	t.Run("MaxAnyOfItems limit", func(t *testing.T) {
		must := require.New(t)
		// Create a validator with a low anyOf item limit
		validator := newSchemaValidator(
			WithMaxAnyOfItems(2),
		)

		// Schema with more anyOf items than allowed
		schemaWithManyAnyOf := `{
			"anyOf": [
				{"type": "string"},
				{"type": "number"},
				{"type": "boolean"}
			]
		}`

		// This validation should fail due to too many anyOf items
		err := validator.Validate(schemaWithManyAnyOf)
		must.Error(err)
		must.Contains(err.Error(), "anyOf must have 1-2 items")

		// Now increase the limit and try again
		validator = newSchemaValidator(
			WithMaxAnyOfItems(3),
		)
		err = validator.Validate(schemaWithManyAnyOf)
		must.NoError(err)
	})

	t.Run("MaxSchemaDepth limit", func(t *testing.T) {
		must := require.New(t)
		// Create a validator with custom depth limit
		validator := newSchemaValidator(
			WithMaxSchemaDepth(3),
		)

		// Create a deeply nested schema that should exceed the depth limit
		deepSchema := `{
			"type": "object",
			"properties": {
				"level1": {
					"type": "object",
					"properties": {
						"level2": {
							"type": "object",
							"properties": {
								"level3": {
									"type": "object",
									"properties": {
										"level4": {
											"type": "string"
										}
									}
								}
							}
						}
					}
				}
			}
		}`

		// This validation should fail due to depth exceeding the limit
		err := validator.Validate(deepSchema)
		must.Error(err)
		must.Contains(err.Error(), "schema depth exceeds maximum limit of 3")

		// Now increase the depth limit and try again
		validator = newSchemaValidator(
			WithMaxSchemaDepth(4),
		)
		err = validator.Validate(deepSchema)
		must.NoError(err)
	})

	t.Run("MaxSchemaSize limit", func(t *testing.T) {
		must := require.New(t)
		// Create a validator with a very low schema size limit
		validator := newSchemaValidator(
			WithMaxSchemaSize(100), // Very small limit
		)

		// Create a schema that exceeds this small size limit
		largeSchema := `{
			"type": "object",
			"properties": {
				"prop1": {"type": "string", "description": "This is property 1 with a somewhat lengthy description"},
				"prop2": {"type": "number", "description": "This is property 2 with another lengthy description"},
				"prop3": {"type": "boolean", "description": "And here we have property 3 with yet another description"}
			},
			"required": ["prop1", "prop2"]
		}`

		// This validation should fail due to size exceeding the limit
		err := validator.Validate(largeSchema)
		must.Error(err)
		must.Contains(err.Error(), "schema exceeds maximum allowed size")

		// Now increase the size limit and try again
		validator = newSchemaValidator(
			WithMaxSchemaSize(1000),
		)
		err = validator.Validate(largeSchema)
		must.NoError(err)
	})

	t.Run("MaxTotalPropertiesKeysNum limit", func(t *testing.T) {
		must := require.New(t)
		// Create a validator with a low property keys limit
		validator := newSchemaValidator(
			WithMaxTotalPropertiesKeysNum(5),
		)

		// Create a schema with more properties than the limit
		schemaWithManyProperties := `{
			"type": "object",
			"properties": {
				"prop1": {"type": "string"},
				"prop2": {"type": "number"},
				"prop3": {"type": "boolean"},
				"prop4": {"type": "array", "items": {"type": "string"}},
				"prop5": {"type": "object", "properties": {"subprop": {"type": "string"}}},
				"prop6": {"type": "integer"}
			}
		}`

		// This validation should fail due to too many properties
		err := validator.Validate(schemaWithManyProperties)
		must.Error(err)
		must.Contains(err.Error(), "total number of properties keys(6) across all objects exceeds maximum limit of 5")

		// Now increase the limit and try again
		validator = newSchemaValidator(
			WithMaxTotalPropertiesKeysNum(7), // 6 + 1
		)
		err = validator.Validate(schemaWithManyProperties)
		must.NoError(err)
	})
}

func runTestCases(t *testing.T, validCases []string, invalidCases []struct {
	schema         string
	expectedErr    string
	isUnmarshalErr bool
}) {
	must := require.New(t)
	validator := newSchemaValidator(
		WithValidateLevel(ValidateLevelTest),
		WithMaxSchemaDepth(5),
	)

	for _, schema := range validCases {
		err := validator.Validate(schema)
		must.NoError(err, "Valid schema failed: %v\nSchema: %v", err, schema)
	}

	for _, tc := range invalidCases {
		err := validator.Validate(tc.schema)
		must.Error(err, "invalid schema should have failed: %s\nSchema: %v", tc.expectedErr, tc.schema)

		if err != nil {
			if tc.isUnmarshalErr {
				must.True(IsUnmarshalError(err),
					"Expected UnmarshalError, got: %T\nSchema: %v", err, tc.schema)
			} else {
				must.True(IsSchemaError(err),
					"Expected SchemaError, got: %T\nSchema: %v", err, tc.schema)
			}

			errLower := strings.ToLower(err.Error())
			expectedLower := strings.ToLower(tc.expectedErr)
			must.Contains(errLower, expectedLower,
				"Expected error containing '%s', got '%s', %v", tc.expectedErr, err.Error(), tc.schema)
		}
	}
}

func TestUltraValidate(t *testing.T) {
	must := require.New(t)
	validator := newSchemaValidator(WithValidateLevel(ValidateLevelUltra))

	invalidCases := []struct {
		schema         string
		expectedErr    string
		isUnmarshalErr bool
	}{
		// required contains duplicate items
		{
			`{
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				},
				"required": ["name", "name"]
			}`,
			"duplicate items in required array",
			false,
		},
		// duplicate types in type array
		{
			`{
				"type": ["string", "string"]
			}`,
			"duplicate types in type array",
			false,
		},
	}

	for _, tc := range invalidCases {
		err := validator.Validate(tc.schema)
		must.Error(err, "invalid schema should have failed: %s\nSchema: %v", tc.expectedErr, tc.schema)

		if err != nil {
			if tc.isUnmarshalErr {
				must.True(IsUnmarshalError(err),
					"Expected UnmarshalError, got: %T\nSchema: %v", err, tc.schema)
			} else {
				must.True(IsSchemaError(err),
					"Expected SchemaError, got: %T\nSchema: %v", err, tc.schema)
			}

			errLower := strings.ToLower(err.Error())
			expectedLower := strings.ToLower(tc.expectedErr)
			must.Contains(errLower, expectedLower,
				"Expected error containing '%s', got '%s', %v", tc.expectedErr, err.Error(), tc.schema)
		}
	}
}

func TestLiteAllowsMultipleTypesWithItems(t *testing.T) {
	must := require.New(t)
	schema := `{"additionalProperties": false, "properties": {"files": {"description": "List of files to read; request related files together when allowed", "items": {"additionalProperties": false, "properties": {"line_ranges": {"description": "Optional line ranges to read. Each range is a [start, end] tuple with 1-based inclusive line numbers. Use multiple ranges for non-contiguous sections.", "items": {"items": {"type": "integer"}, "maxItems": 2, "minItems": 2, "type": "array"}, "type": ["array", "null"]}, "path": {"description": "Path to the file to read, relative to the workspace", "type": "string"}}, "required": ["path", "line_ranges"], "type": "object"}, "minItems": 1, "type": "array"}}, "required": ["files"], "type": "object"}`

	lite := newSchemaValidator(WithValidateLevel(ValidateLevelLite))
	must.NoError(lite.Validate(schema), "lite should accept type[] + items")

	for _, level := range []ValidateLevel{ValidateLevelStrict, ValidateLevelUltra, ValidateLevelDefault} {
		v := newSchemaValidator(WithValidateLevel(level))
		err := v.Validate(schema)
		must.Error(err, "level %s should reject", level)
		must.Contains(strings.ToLower(err.Error()), "multiple types")
	}
}
