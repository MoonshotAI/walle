> 🌐 **Language:** English | [简体中文 (Chinese)](./mfjs-spec.zh.md)

# Moonshot Flavored JSON Schema Spec

Moonshot Flavored JSON Schema (MFJS) is a carefully curated JSON Schema subset designed by Moonshot AI to address the difficulties Large Language Models (LLM) face in understanding the full JSON Schema specification. MFJS is specifically optimized for LLM interaction scenarios.

It is primarily designed for defining **Tool Calling** `parameters` fields and **Response Format**, ensuring models can accurately understand data requirements and generate structured JSON output by removing complex features from the original specification while retaining core structure.

While MFJS supports basic JSON Schema structure and type system, the following features have been streamlined or restricted to ensure large models can accurately understand and generate content:

* **`$defs`, `$ref`, `$id` are supported with limitations:** `$defs` must be defined at the schema root level; `$ref` only supports referencing **internal** subschemas within this document (indexed through `$defs`, or using `"$ref": "#"` for self-reference)
* **No external resources:** Meta-Schemas and any HTTP-based external network resource references are not supported
* **No metadata and annotations:** Descriptive metadata like `title`, `$comment`, and `format` annotation fields are not supported
* **No advanced array definitions:** Tuple simulation using `prefixItems` or `unevaluatedItems` is not supported
* **No complex validation:** Complex validation keywords such as `minimum`, `exclusiveMinimum`, `maximum`, `exclusiveMaximum`, `minItems`, `maxItems` are not supported

MFJS operates as a curated JSON Schema subset within model interactions, maintaining recognition of this format while striving to produce outputs that conform to the specified schema definitions. This document provides a comprehensive specification of MFJS, including detailed definitions, field specifications, and practical examples for both Tool Calling and Response Format use cases.

## Prerequisites

### JSON Documents
According to the definition of [JSON](https://www.json.org/json-en.html), it is a lightweight, text-based, language-independent syntax used to define data interchange formats. It is derived from the ECMAScript programming language but is independent of programming languages.

### Schema and Instance
JSON Schema interprets documents based on a data model. The JSON value interpreted according to this data model is called an "Instance".

Typically, in Moonshot interactions, the input parameter is a valid JSON that defines the data format (the JSON Schema). Our language model then outputs a JSON that conforms to this schema definition, which is referred to as the instance.

### JSON Instance Data Types
An instance has one of six primitive types and has a range of possible values according to the type:

- `null`: A JSON "null" value, usually None in Python
- `boolean`: A JSON "true" or "false" value, usually a boolean type in Python
- `object`: An unordered set of properties mapping strings to instances, from JSON "object" values, possibly a Dict type in Python
- `array`: An ordered list of instances, from JSON "array" values, possibly a List in Python
- `number`: An arbitrary-precision, base-10 decimal number, from JSON "number" values, possibly mapped to a float in Python
- `string`: A string, from JSON "string" values, or a str in Python

Whitespace and formatting issues, including different lexical representations of numbers, are equal at the data model level and therefore outside the scope of JSON Schema.

In MFJS "type" definitions, our type system is not completely consistent with this, for example, the "integer" type strictly constrains it to output an integer, which is a valid "type" but may appear as a number in JSON's data types.

## A Top-Down Case Study

Let's consider a requirement in a Tool Use scenario: we want to provide a tool that informs the model we have a search engine. We might define the tool as follows to describe its function:

```json
{
  "type": "function",
  "function": {
    "name": "web_search",
    "description": "搜索引擎，可以按需查询这个世界",
    "parameters": {}
  }
}
```

This tool introduces itself as web_search and describes that it can perform searches. In the parameters field, we can define a JSON Schema to inform the model about the format and content of the data that should be returned.

When the model needs to perform a search, it calls `web_search` and returns parameters conforming to this Schema. These returned parameters constitute a valid "Instance" as defined earlier.

Next, we will use this requirement as an example to try to iteratively improve the value of parameters in the example so that the results better meet the caller's needs.

### Starting from Zero

If we do nothing, that is, the value of parameters is `{}`

This means you allow the model to pass any content, including empty content, as arguments when calling `web_search`.

Examples like the following are all legal under the above definition:

```json
null
true
"a sample query"
{ "qs": "a sample query"}
["first query", "second query"]
```

### Defining Types

Considering ease of parsing, and we would like the model to output some complex query conditions, we would like the output structure to be a valid JSONObject. Then we can define a type constraint:

```json
{
  "type": "object"
}
```

This way, the content output by the model can be parsed by python, and the root object is a Dict.

Examples like the following are legal under the above definition:

```json
{ "qs": "a sample query"}
{ "query": "another sample query"}
{ "query": 42 }
```

### Defining Parameter Lists

We can fix parameters. The query must use the field `qs`, and prompt optional parameters `lang` to filter language, and `limit` to search for top_n results.

```json
{
  "type": "object",
  "properties": {
    "qs": { "type": "string", "description": "place your query here" },
    "limit": { "type": "integer", "default": 10 },
    "lang": { "enum": [ "Chinese", "English" ] }
  },
  "required": [ "qs" ]
}
```

As in the example above, we use `required` to define `qs` as a required field. In `properties`, we define these three fields. In each key-value pair, the key is the field id to be output, and the value is a child Schema. The child Schema is a self-contained structure that can be used in isolation according to this Spec definition.

Examples like the following are legal under the above definition:

```json
{ "qs": "a sample query"}
{ "qs": "a sample query", "limit": 20 }
{ "qs": "a sample query", "lang": "Chinese" }
{ "qs": "a sample query", "limit": 20, "lang": "Chinese", "foo": "bar" }
```

### Building More Complex Expression Queries

Sometimes, we may need to perform multiple different query statements separately and merge the results manually. Then we can use the `anyOf` keyword to organize more complex `qs`.

In addition, we might consider closing other possible parameters. We can modify it further as follows:

```json
{
  "type": "object",
  "properties": {
    "qs": {
      "anyOf": [
        {
          "type": "string"
        },
        {
          "type": "array",
          "items": {
            "type": "string"
          }
        }
      ],
      "description": "place your query or queries here"
    },
    "limit": { "type": "integer", "default": 10 },
    "lang": { "enum": [ "Chinese", "English" ] }
  },
  "additionalProperties": false,
  "required": [ "qs" ]
}
```

So our output becomes one of two controllable types.

Examples like the following are legal under the above definition:

```json
{ "qs": "a sample query"}
{ "qs": ["first query", "second query"]}
```

Finally, if we populate this JSON Schema back, the request might look like this:

```json
{
  "type": "function",
  "function": {
    "name": "web_search",
    "description": "搜索引擎，可以按需查询这个世界",
    "parameters": {
      "type": "object",
      "properties": {
        "qs": {
          "anyOf": [
            {
              "type": "string"
            },
            {
              "type": "array",
              "items": {
                "type": "string"
              }
            }
          ],
          "description": "place your query or queries here"
        },
        "limit": { "type": "integer", "default": 10 },
        "lang": { "enum": [ "Chinese", "English" ] }
      },
      "additionalProperties": false,
      "required": [ "qs" ]
    }
  }
}
```

### Recursive Schema Example
`$defs`, `$ref`, and `$id` enable recursive structure definitions.

Example: Linked list node with self-reference and null termination:

```json
{
  "type": "object",
  "properties": {
    "linked_list": {
      "$ref": "#/$defs/linked_list_node"  // Reference within same schema
    }
  },
  "$defs": {          // Must be at root level
    "linked_list_node": {
      "type": "object",
      "properties": {
        "value": {
          "type": "number"
        },
        "next": {
          "anyOf": [
            {
              "$ref": "#/$defs/linked_list_node"  // Self-reference
            },
            {
              "type": "null"  // List termination
            }
          ]
        }
      },
      "additionalProperties": false,
      "required": ["next","value"]
    }
  },
  "additionalProperties": false,
  "required": ["linked_list"]
}
```

## Field Definitions

### Meta Data Fields

Meta Data fields provide annotation information about the Schema. They are provided to the language model based on their location within the schema to guide the model's output. Meta Data is usually not directly reflected in the output result, but in some cases it will be provided as input to the model.

#### description
**Type**: str

The `description` field provides a brief explanation of the intended output. The type must be a String. It can be valid anywhere in the Schema, but will only prompt the model in appropriate places.

Specifically, `"response_format": { "type": "json_object" }` when json mode is turned on, this will not take effect.

#### default
**Type**: Any

The default field provides a default value for the target output. It suggests to the model what default value might be used. No type checking is performed here, and the caller needs to ensure that the value type and content match what is expected in the location.

Specifically, `"response_format": { "type": "json_object" }` when json mode is turned on, this will not take effect.

### Applicator Fields

Applicator fields are used to specify how the Schema is applied to specific parts of JSON instances. Generally, it applies to any internal control of the Schema.

#### anyOf
**Type**: List[Schema]

The output instance matches at least one child Schema defined by this field.

JSON Schema includes some keywords for combining schemas, we can use anyOf to combine multiple type definitions to achieve a similar Union effect in python.

#### properties
**Type**: Dict[str, Schema]

properties is a set of KV pairs. In each key-value pair, the key is the field id to be output, and the value is a child Schema. The child Schema is a self-contained structure.

This typically applies only when `type` is `object`.

#### additionalProperties
**Type**: Union[bool, Schema]

The default value of additionalProperties is `{}`, that is, if it is not set, then in addition to the fields in properties, other fields are allowed to be generated by default and the type is arbitrary.

For example, if we define `qs` as string in properties, then the model may still be allowed to output fields like `city: beijing`.

We can also add options to the value of additionalProperties, such as using `{"additionalProperties":{"type":"string"}}` to limit the type of values of other unlisted properties to ensure that the generated additional fields are of the expected type.

At the current model behavior level, our model may not be inclined to generate additional fields. If compatibility is desired, consider setting `"additionalProperties": false` to prohibit the generation of other fields in the future.

#### items
**Type**: Schema

When the `type` is `array`, it defines the sub-schema of each element of the array.

### Validation Fields

Validation fields are used to define the content format of the target instance, which is mainly applied to the values of leaf nodes.

#### type
**Type**: str

We currently support the following types:

| Type | Corresponding Python Type | Description |
|-----|---------------------------|-------------|
| null | None | Null constant in JSON |
| boolean | bool | Boolean constants in JSON, including true and false |
| object | Dict | JSON object, composed of key-value pairs |
| array | List | JSON array, an ordered set of values |
| number | Union[int, float] | JSON number, including integers and floating-point numbers |
| integer | int | Integer numbers in JSON |
| string | str | JSON string, text enclosed in double quotes |

Similar to JSON instances, it just adds an `integer` type to explicitly declare that the output is an integer type number.

#### enum
**Type**: List[Union[float, int, str]]

Enumerate possible outputs. Note that its elements are not a Schema, but literals.

Specifically, we currently only support float, int, and str as enumeration values. Other primitive types, as well as object and array, are not supported. Mixed types are also prohibited (all elements in the enum list must be of the same type, e.g., you cannot mix str and int).

#### required
**Type**: List[str]

Specifies the required fields that must be present in the instance at this level. By default all parameters are optional.


## References

The definitions and design of this document reference the following JSON Schema official specifications. Since Moonshot Flavored JSON Schema (MFJS) is a curated subset optimized for LLM interaction scenarios, please pay special attention to the following when referring to the official documentation:

* **Specification Priority**: If there are any conflicts or discrepancies between the definitions in this document and the official references, **the detailed definitions in this document shall prevail**. Strict adherence to the constraints in this document can prevent system validation errors in scenarios such as Tool Calling.
* **Unsupported Features**: If certain concepts, keywords, or features in the official specifications are not mentioned in this document, it means that MFJS does not officially support them yet. While these unlisted features may not be directly blocked by the system when passed in and may be observed by the model, **we do not guarantee** that they will produce stable, expected results in the final structured output.

### Normative References

- [JSON-Schema.org Specification Release Page](https://json-schema.org/specification#specification-section) - Official JSON Schema specification release page
- [JSON Schema Core](https://json-schema.org/draft/2020-12/json-schema-core.html) - Defines the foundational core architecture of JSON Schema
- [JSON Schema Validation](https://json-schema.org/draft/2020-12/json-schema-validation.html) - Defines the validation keywords of JSON Schema

### Informational References

- [JSON Schema Reference](https://json-schema.org/understanding-json-schema/reference#json-schema-reference) - Official JSON Schema reference manual
- [JSON Schema 2019-09](https://www.learnjsonschema.com/2019-09/) - Includes detailed information about each keyword, plus many examples