> 🌐 **Language:** English | [简体中文 (Chinese)](./walle.zh.md)

# JSON Schema Validator (walle)

This document describes how **walle** validates JSON Schemas and classifies errors, in line with the [**Moonshot Flavored JSON Schema Spec** (MFJS)](./mfjs-spec.md).

## Terms

| Term | Meaning |
| --- | --- |
| **walle** | The MFJS schema validator (this repository). |
| **MFJS** | Moonshot Flavored JSON Schema Spec. |
| **root** | The root of the schema, corresponding to JSON Pointer `/`. |
| **ANY** | Any of `null`, `boolean`, `object`, `array`, `number`, `integer`, or `string`. |

## Error categories

| Category | Rules | Details |
| --- | --- | --- |
| Structural errors | Every subschema must declare `type` explicitly. If `anyOf` or `$ref` is present, `type` must appear **inside** that `anyOf` / `$ref`, not beside it at the same level. | Two exceptions:<br>Case 1: the entire schema is `{}` → ANY.<br>Case 2: `"additionalProperties": {}` → ANY.<br>Note: in all **other** cases, `{}` is **not** inferred as ANY—for example:<br><br><pre><code class="language-json">"properties": {&#10;  "key1": {},&#10;  "key2": {}&#10;}</code></pre> |
|  | Only the seven types `null` / `boolean` / `object` / `array` / `number` / `integer` / `string` are supported; the root schema must be a JSON object. |  |
|  | Only keywords allowed by MFJS. | Two cases: illegal keywords, and keywords that are legal in JSON Schema but not supported by MFJS. |
|  | Keyword placement must follow JSON Schema conventions. | For `type: object`, only keywords such as `type`, `properties`, `required`, `additionalProperties`, `anyOf`, and `$ref` apply (see MFJS for the full list). |
|  | Object: every name in `required` must be declared in `properties`. |  |
|  | Object: `properties` keys must be unique. |  |
|  | Object: nesting depth and property count are capped. | A schema may have up to **100** object properties in total, with up to **5** levels of nesting (OpenAI-style limit, for performance). |
|  | Object: `properties` keys must not be named `"$defs"`, `"$ref"`, `"anyOf"`, `"required"`, or `"additionalProperties"`. |  |
|  | The `type` keyword must not sit beside `anyOf` / `$ref`; `type` belongs **inside** them. |  |
|  | `anyOf` must have between **1** and **10** items. |  |
|  | `$defs` and `$id` may only appear at the **root**. |  |
|  | `$ref` must resolve within this schema or its `$defs`; **no** remote, cross-file, or URL refs. | Self-reference: `"$ref": "#"`. |
|  | `$ref` / `$defs` must admit a sound termination condition; **infinite** recursive loops are forbidden. |  |
|  | `$ref` may only appear where allowed. | For example: `properties`, `$defs`, `additionalProperties`, `anyOf`, `items`, or the root. |
|  | Array: enum size limits. | A schema may have up to **500** enum values across all enum properties. For a single `number` / `integer` enum whose values are treated as strings, when there are more than **250** values the total string length (after number/integer → string) must not exceed **7,500** characters. |
|  | Array: `items` may be omitted; if present it must not be empty. |  |
|  | Total string-size limits. | Same enum caps as above for string enums. Length is measured after Go’s `encoding/json` marshaling; **whitespace is not** counted toward that total. |
|  | Beside `anyOf` / `$ref`, only `description` and `title` are allowed at the same level; at the **root**, `$defs` and `$id` may also appear. | Strict rule to avoid type / keyword mismatches after expansion. |
|  | `default` is only allowed for `boolean`, `number`, `string`, `integer`, and `null`, and must match the declared type. |  |
| Data type errors | `type` must agree with `enum` values (e.g. not `integer` with `3.67`). |  |
|  | The value of `type` must be a string (or an MFJS-allowed `type` array). |  |
|  | Every entry in `required` must be a string. |  |
|  | `enum` must be an array. |  |
|  | If `type` is an array, extra rules apply when combined with `enum`. | **Case 1 — `type` without `enum`:** any legal combination of types is allowed, but the same subschema may only use `description` / `title`; at the root, `$id` / `$defs` are also allowed.<br>**Case 2 — `type` with `enum`:** (1) `type` has length **1** or **2**; (2) if length is **2**, only `number`, `integer`, `string`, or `boolean` paired with `null`; (3) enum values must match those types. |
|  | All `enum` elements must share the same type. | For `number`, `[2, 3.33]` is valid. |
|  | Values in `enum` and min/max for `integer` / `number` must lie in the allowed numeric range. | Integers: decimal only, no other bases. Floats: **no** scientific notation. `double`: roughly `(-1.8e308, 1.8e308)`; `int`: approximately `[-2**53 + 1, 2**53 - 1]`. |
|  | `properties` must be an object. |  |
|  | `description`, `title`, and `$id` must be strings. |  |
|  | `anyOf` must be an array; each item must be a valid subschema. |  |
|  | The value of `additionalPropertis` may only be `boolean` or `object`. | If omitted, the default is `true`. |
|  | Any `min*` / `max*` pair must satisfy `min <= max`. | With `"type": "integer"`, floating-point `minimum` / `maximum` are accepted at walle **Normal** validation level, but the enforcer **truncates toward zero**. |
|  | `$defs` names must not contain `/`. |  |
