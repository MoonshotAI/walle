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
| Structural errors | Every subschema must declare `type` explicitly. Beside `anyOf` or `$ref` it is legal—2020-12 evaluates siblings as a logical AND—so lite accepts it: `Canonical` pushes it into each `anyOf` branch, or folds it into the referenced schema. An empty intersection with the referenced `type` admits no instance at all: lite still accepts it but `Canonical` degrades that subschema to `{}`, and strict and above reject it. | Two exceptions:<br>Case 1: the entire schema is `{}` → ANY.<br>Case 2: `"additionalProperties": {}` → ANY.<br>Note: in all **other** cases, `{}` is **not** inferred as ANY—for example:<br><br><pre><code class="language-json">"properties": {&#10;  "key1": {},&#10;  "key2": {}&#10;}</code></pre> |
|  | Only the seven types `null` / `boolean` / `object` / `array` / `number` / `integer` / `string` are supported; the root schema must be a JSON object. |  |
|  | Only keywords allowed by MFJS. | Two cases: illegal keywords, and keywords that are legal in JSON Schema but not supported by MFJS. |
|  | Keyword placement must follow JSON Schema conventions. | For `type: object`, only keywords such as `type`, `properties`, `required`, `additionalProperties`, `anyOf`, and `$ref` apply (see MFJS for the full list). |
|  | Object: every name in `required` must be declared in `properties`. |  |
|  | Object: `properties` keys must be unique. |  |
|  | Object: nesting depth and property count are capped. | Defaults: **3000** `properties` keys across all objects, **30** levels of nesting, and **120000** bytes per schema. Callers can change these with `WithMaxTotalPropertiesKeysNum` / `WithMaxSchemaDepth` / `WithMaxSchemaSize`. |
|  | Object: `properties` keys must not be named `"$defs"`, `"$ref"`, `"anyOf"`, `"required"`, or `"additionalProperties"`. |  |
|  | The `type` keyword may sit beside `anyOf` or `$ref`, and lite accepts both. Ultra reports it so that `Canonical` pushes it into each `anyOf` branch, or folds it into the referenced schema. | See [validation-principles.md](./validation-principles.md) for the per-level behaviour. |
|  | `anyOf` must have between **1** and **500** items by default (`WithMaxAnyOfItems`). |  |
|  | `$defs` and `$id` may only appear at the **root**. |  |
|  | `$ref` must resolve within this schema or its `$defs`; **no** remote, cross-file, or URL refs. | Self-reference: `"$ref": "#"`. |
|  | `$ref` / `$defs` must admit a sound termination condition; **infinite** recursive loops are forbidden. | The test is whether a finite instance exists: a cycle through an optional property, or through an array without `minItems`, terminates; a cycle through a required property, or through an array with `minItems >= 1`, is rejected. |
|  | `$ref` may only appear where allowed. | For example: `properties`, `$defs`, `additionalProperties`, `anyOf`, `items`, or the root. |
|  | Array: enum size limits. | A single `enum` may hold up to **1000** values by default. A separate cap limits their combined string length to **75000** characters, but it is only checked once an `enum` exceeds **2500** values—above the 1000-value cap—so it never fires under the default config, only after a caller raises the item cap with `WithMaxEnumItems`. |
|  | Array: `items` may be omitted; if present it must not be empty. |  |
|  | Total string-size limits. | As above: **1000** values per `enum` by default, a **75000**-character total with a **2500**-value trigger. Length is measured after Go’s `encoding/json` marshaling; **whitespace is not** counted toward that total. |
|  | Constraint keywords are allowed beside both `anyOf` and `$ref` (plus `$defs` / `$id` at the **root**). | Siblings are legal under 2020-12's AND semantics, so lite accepts them. Ultra reports them so that `Canonical` distributes the ones beside `anyOf` into every branch and inlines the ones beside `$ref`, keeping the stricter value for each shared keyword either way. |
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
|  | Any `min*` / `max*` pair must satisfy `min <= max`. | `min > max` cannot be satisfied by any instance: lite accepts it but `Canonical` degrades that subschema to `{}`, while strict and above reject it. Separately, with `"type": "integer"` floating-point `minimum` / `maximum` are accepted, but the enforcer **truncates toward zero**. |
|  | `$defs` names must not contain `/`. |  |
