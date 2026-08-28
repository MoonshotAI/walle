> 🌐 **Language:** English | [简体中文 (Chinese)](./mfjs-walle-vs-draft-2020-12.zh.md)

# walle and JSON Schema Draft 2020-12

This document summarizes how the **walle** validator handles each keyword in **JSON Schema Draft 2020-12**. **Keyword** rows appear in the same top-to-bottom order as the vocabulary tables on [learnjsonschema.com/2020-12](https://www.learnjsonschema.com/2020-12/). For background semantics see [**Moonshot Flavored JSON Schema (MFJS)**](./mfjs-spec.md); **normative validation rules** are **[walle.md](./walle.md)** together with the code in this repository.

## Columns

| Column | Meaning |
| --- | --- |
| **Keyword** | Draft 2020-12 keyword (ordered as on learnjsonschema). |
| **walle** | Behaviour per [walle.md](./walle.md) and the validator implementation. |

**Legend:** ✅ Supported / allowed · ❌ Not supported or not provided

---

## 1. Core

| Keyword | walle |
| --- | --- |
| `$id` | ✅ **Root only**; value must be a string. |
| `$schema` | ❌ |
| `$ref` | ✅ **In-document references only**; remote / URL / cross-file references are disallowed; infinite recursion must be avoided (the test is whether a finite instance exists); siblings of `$ref` apply as a logical AND, per 2020-12. |
| `$comment` | ❌ |
| `$defs` | ✅ **Root only**; **definition names must not contain `/`**. |
| `$anchor` | ❌ |
| `$dynamicAnchor` | ❌ |
| `$dynamicRef` | ❌ |
| `$vocabulary` | ❌ |

---

## 2. Applicator

| Keyword | walle |
| --- | --- |
| `allOf` | ❌ |
| `anyOf` | ✅ Branch count **may be capped**; constraints such as `type` may sit beside it—they apply as a logical AND, and `Canonical` distributes them into every branch. |
| `oneOf` | ❌ |
| `if` | ❌ |
| `then` | ❌ |
| `else` | ❌ |
| `not` | ❌ |
| `properties` | ✅ When `type` is `object`: **keys must not** be `$defs`, `$ref`, `anyOf`, `required`, or `additionalProperties`; **no duplicate keys**. A name in `required` that `properties` does not declare is accepted by lite and pruned by `Canonical`. |
| `additionalProperties` | ✅ Value must be a **boolean** or an **object**; if omitted, **defaults to true**. |
| `patternProperties` | ❌ |
| `dependentSchemas` | ❌ |
| `propertyNames` | ❌ |
| `contains` | ❌ |
| `items` | ✅ Meaningful only for **`type: array`**: **`items` may be omitted**; if present, it must be a **single** non-empty subschema, and **every array element** is validated against it. The Draft 2020-12 pairing of **`prefixItems` for leading positions** and **`items` for the rest** is **not** supported. |
| `prefixItems` | ❌ |

---

## 3. Validation

| Keyword | walle |
| --- | --- |
| `type` | ✅ Seven literals: `null`, `boolean`, `object`, `array`, `number`, `integer`, `string`; **the root schema must be an object**; `type` must be a string or a permitted array (additional rules when combined with `enum`). |
| `enum` | ✅ **Homogeneous** literals only (`float` / `int` / `str`); **total enum usage** across the schema **may be limited**; large enums or strings may hit aggregate length limits. |
| `const` | ❌ |
| `maxLength` | ✅ |
| `minLength` | ✅ |
| `pattern` | ❌ Not yet supported (planned). |
| `exclusiveMaximum` | ❌ |
| `exclusiveMinimum` | ❌ |
| `maximum` | ✅ Allowed for **`type: number`** / **`integer`**; must satisfy **maximum ≥ minimum** when both are set. |
| `minimum` | ✅ |
| `multipleOf` | ❌ |
| `dependentRequired` | ❌ |
| `maxProperties` | ❌ |
| `minProperties` | ❌ |
| `required` | ✅ |
| `maxItems` | ✅ |
| `minItems` | ✅ |
| `maxContains` | ❌ |
| `minContains` | ❌ |
| `uniqueItems` | ❌ |

---

## 4. Meta Data

| Keyword | walle |
| --- | --- |
| `title` | ✅ |
| `description` | ✅ Value must be a string. |
| `default` | ✅ The **Moonshot server** constrained decoding module does **not** yet support `default`; **walle** still accepts schemas that include it (validation passes). |
| `deprecated` | ❌ |
| `examples` | ❌ |
| `readOnly` | ❌ |
| `writeOnly` | ❌ |

---

## 5. Format Annotation

| Keyword | walle |
| --- | --- |
| `format` | ❌ |

---

## 6. Content

| Keyword | walle |
| --- | --- |
| `contentEncoding` | ❌ |
| `contentMediaType` | ❌ |
| `contentSchema` | ❌ |

---

## 7. Unevaluated

| Keyword | walle |
| --- | --- |
| `unevaluatedItems` | ❌ |
| `unevaluatedProperties` | ❌ |

---

## 8. Format Assertion Official

| Keyword | walle |
| --- | --- |
| `format` | ❌ |

---

## 9. Structural rules

| Topic | walle |
| --- | --- |
| Empty object subschema `{}` | **ANY** is expressed only when the **entire root** is `{}` or when **`additionalProperties`** is `{}`. A `{}` **inside** `properties` is **not** treated as ANY. |
| `type` alongside `anyOf` | **Allowed**—2020-12 applies it as a logical AND. lite accepts it and `Canonical` pushes it into every branch, dropping the branches it contradicts; if that leaves no branch, the subschema degrades to `{}`. |
| `type` alongside `$ref` | **Allowed**—2020-12 applies it as a logical AND. lite accepts it and ultra folds it into the target. An empty intersection with the target's `type` admits no instance: lite still accepts it but `Canonical` degrades that subschema to `{}`, and strict and above reject it. |
| Keywords allowed on `object` | **`type`**, **`properties`**, **`required`**, **`additionalProperties`**, **`anyOf`**, **`$ref`**, plus annotations such as **`description`** / **`title`** where rules allow. |
| Siblings of `anyOf` | Constraint keywords are allowed, and the **root** may also include **`$defs`** / **`$id`**. `Canonical` distributes the constraints into every branch and leaves **`description`** / **`title`** where they are. |
| Siblings of `$ref` | Constraint keywords are **allowed**. lite accepts them; ultra inlines the definition and keeps the stricter value for each shared keyword—see [validation-principles.md](./validation-principles.md). |
| Nesting and size | For example, **total `properties` keys across objects** and **nesting depth** **may be limited**—see **[walle.md](./walle.md)**. |
| Numeric and enum literals | Integers **decimal only**; floating-point **no scientific notation**; further bounds as in **walle.md**. |

---

## References

- [learnjsonschema.com — JSON Schema 2020-12](https://www.learnjsonschema.com/2020-12/)
- This repository: [mfjs-spec.md](./mfjs-spec.md), [walle.md](./walle.md)
- Chinese edition of this mapping: [mfjs-walle-vs-draft-2020-12.zh.md](./mfjs-walle-vs-draft-2020-12.zh.md)
