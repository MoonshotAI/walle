> 🌐 **Language:** English | [简体中文 (Chinese)](./validation-principles.zh.md)

# walle Validation Rules and Canonicalization

This document answers three questions: **why was my schema rejected**, **how do I fix it**, and **what will it be rewritten into once it passes**.

For the full rule table see [walle.md](./walle.md), for the spec itself see [mfjs-spec.md](./mfjs-spec.md), and for a keyword-by-keyword comparison with JSON Schema 2020-12 see [mfjs-walle-vs-draft-2020-12.md](./mfjs-walle-vs-draft-2020-12.md).

## 1. The four levels

| Level | Behaviour |
| --- | --- |
| `loose` | No validation at all; every schema passes |
| `lite` | Rejects misspelled schemas, unresolvable references, and non-terminating recursion |
| `strict` | On top of `lite`, additionally rejects schemas with mistyped or self-contradictory numeric bounds |
| `ultra` | The strictest; `Canonical` uses it to find everything that needs rewriting |

`lite` cares about exactly one thing: **is this schema well-formed and satisfiable**. Constructs that are legal but unusable downstream are all accepted and left for `Canonical` to rewrite.

**Rejected by the interface?** See section 2. **Accepted but the generated output is not what you expected?** See sections 3-5.

## 2. Constructs that get rejected

Everything below is rejected by `lite` (the interface default).

### 2.1 A keyword's value is written incorrectly

| Case | Error |
| --- | --- |
| `type` is not a string or an array of strings | `type must be string or array of strings` |
| `type` array is empty | `type array cannot be empty` |
| `type` array contains a non-string entry | `invalid type in type array` |
| `type` is not one of the 7 types | `invalid type` |
| `properties` is not an object | `properties must be an object` |
| A property's schema is not an object | `property schema for 'x' must be an object` |
| `required` is not an array | `required must be an array` |
| `required` contains a non-string entry | `items in required array must be strings` |
| `enum` is not an array | `enum must be an array` |
| `enum` is an empty array | `enum array cannot be empty` |
| `items` is not an object | `items must be an object` |
| `anyOf` is not an array | `anyOf must be an array` |
| `anyOf` is an empty array | `anyOf must have 1-500 items` |
| `$ref` is not a string | `$ref must be a string` |
| `pattern` is not a string (`null` excluded) | `pattern must be a string` |
| `description` is not a string | `description must be a string` |
| `additionalProperties` is not a boolean or an object | `additionalProperties must be a boolean or an object` |
| `$defs` is not an object | `$defs must be an object` |
| `$id` is not a string (`null` excluded) | `$id must be a string` |
| A numeric bound keyword is `null` | `minItems must be an integer` / `minimum must be a number` etc. |

Size limits also apply at `lite`: 120000 bytes per schema, 30 levels of nesting, 3000 property keys across all objects, 1000 items per `enum`, and 500 `anyOf` branches.

When a numeric bound keyword (`minLength` `maxLength` `minimum` `maximum` `minItems` `maxItems`) has a non-integer or negative value (`null` excluded), `lite` accepts it and the rewrite corrects it; `strict` and above reject it.

A mistyped `title`, and a `null` `$id` or `pattern`, are also accepted by `lite` and dropped by `Canonical`; `strict` and above reject them. Other non-string `$id` / `pattern` values are still rejected from `lite`. A non-string `description` (including `null`) is still rejected from `lite`.

### 2.2 References that cannot be resolved

Only references pointing at `$defs` inside this document are supported.

| Case | Error |
| --- | --- |
| External, cross-file, or URL reference, e.g. `"$ref":"https://example.com/s.json"` | `references must start with #/$defs/` |
| Points at a path outside `#/$defs/` | `references must start with #/$defs/` |
| Empty definition name, i.e. `"$ref":"#/$defs/"` | `definition name cannot be empty` |
| The referenced definition does not exist | `invalid $ref path: ...` |
| Uses `$ref` but the schema has no `$defs` at all | `$defs not found for reference: ...` |
| A `$defs` key contains `/` | `$defs property name 'a/b' cannot contain '/' character` |

### 2.3 Nothing can ever satisfy it

Only non-terminating recursion falls into this category. Contradictory `type` / `enum` / numeric bounds do not — they are equally unsatisfiable, but `lite` accepts them and the rewrite drops the contradicting constraint; see section 5.

| Case | Error |
| --- | --- |
| Required properties form a reference cycle | `detected infinite recursion without termination condition` |
| Mutual references where every hop is required | Same as above |
| Array self-reference with `minItems >= 1` | Same as above |

### 2.4 The reference graph is too complex

Diamond reference chains (every definition referenced by several properties, each level pointing at the next definition) make reference traversal grow exponentially with the chain length. Once the traversal exceeds the built-in step budget (500000 steps) the schema is rejected:

| Case | Error |
| --- | --- |
| Diamond / long-cycle reference chains exceeding the traversal step budget | `reference graph is too complex to validate within the step budget` |

Honest schemas stay orders of magnitude below the budget; only deliberately crafted ones hit this.

## 3. How keywords beside `$ref` are merged

| Keyword | Merge rule |
| --- | --- |
| `minLength` `minItems` `minimum` | Keep the larger value |
| `maxLength` `maxItems` `maximum` | Keep the smaller value |
| `type` | Intersect; the intersection of `integer` and `number` is `integer`; an empty intersection is covered in section 5 |
| `enum` | Intersect; an empty intersection is covered in section 5 |
| `required` | Union |
| `properties` | Union; a property described on both sides is merged one level deeper by the rules above |
| `description` `title` `pattern` and all other keywords | The use site's value wins |

The use site asks for `minLength: 5` while the definition says `1`; the merged result keeps `5`, and the definition's `maxLength: 50` comes along:

```json
{
  "type": "object",
  "properties": { "name": { "minLength": 5, "$ref": "#/$defs/Name" } },
  "$defs": { "Name": { "type": "string", "minLength": 1, "maxLength": 50 } }
}
```

Rewritten as:

```json
{
  "type": "object",
  "properties": {
    "name": { "type": "string", "minLength": 5, "maxLength": 50 }
  }
}
```

The remaining cases (the table omits the outer `"type":"object"` and `"properties"` wrapper):

| Use site + definition | Rewrite result | Notes |
| --- | --- | --- |
| `{"minLength":1,"$ref":"#/$defs/S"}`<br>`S = {"type":"string","minLength":1}` | `{"$ref":"#/$defs/S"}` with `S` unchanged | Both sides agree, so the duplicate is dropped and the definition stays shared |
| `{"maxLength":20,"$ref":"#/$defs/S"}`<br>`S = {"type":"string","minLength":1}` | `{"type":"string","minLength":1,"maxLength":20}` | The keyword is absent from the definition, so it is folded in directly |
| `{"description":"user name","$ref":"#/$defs/S"}` | Kept as written; passes at every level | Annotations do not affect constraints |
| `short: {"maxLength":10,"$ref":"#/$defs/S"}`<br>`long: {"maxLength":99,"$ref":"#/$defs/S"}`<br>`S = {"type":"string","minLength":1}` | `short: {"type":"string","minLength":1,"maxLength":10}`<br>`long: {"type":"string","minLength":1,"maxLength":99}` | Each use site specialises independently; they do not affect one another |

**Exception: when the referenced definition is recursive, sibling constraints are dropped.**

```json
{
  "type": "object",
  "properties": { "tree": { "minLength": 3, "$ref": "#/$defs/N" } },
  "$defs": {
    "N": { "type": "object", "properties": { "child": { "$ref": "#/$defs/N" } } }
  }
}
```

After the rewrite `minLength: 3` is gone and `tree` is constrained by `N` alone. If you need that constraint to take effect, do not write it at the reference site of a recursive definition; move it into the definition itself.

## 4. How keywords beside `anyOf` are distributed

A constraint written beside `anyOf` applies together with whichever branch matches. The rewrite **pushes the parent constraint into every branch**:

```json
{
  "type": "object",
  "properties": {
    "v": { "type": "string", "anyOf": [{ "minLength": 1 }, { "maxLength": 9 }] }
  }
}
```

Rewritten as:

```json
{
  "type": "object",
  "properties": {
    "v": {
      "anyOf": [
        { "type": "string", "minLength": 1 },
        { "type": "string", "maxLength": 9 }
      ]
    }
  }
}
```

Distribution rules:

- **Same-name keywords merge by the stricter-wins rules of section 3**: a parent `minLength: 20` meeting a branch's `minLength: 10` leaves the branch with 20.
- **A branch that contradicts the parent is dropped**, because no instance can ever reach it. A parent `"type":"string"` over `[{"minLength":1},{"type":"integer"}]` makes the `integer` branch disappear.
- **When every branch is dropped the field is emptied to `{}`** — such a schema had no satisfying instance to begin with.
- **`description` / `title` are not distributed**; they stay at the parent level because they are not constraints.

Contradictions count when a branch uses `$ref` too: a parent `"type":"string"` over a branch referencing a definition of `{"type":"number"}` gets that branch dropped as well.

## 5. Other constructs that get rewritten

`lite` accepts all of these; `Canonical` rewrites them. The table omits the outer wrapper.

| Case | Input | Rewrite result |
| --- | --- | --- |
| Uses an unsupported keyword | `{"type":"string","format":"uuid"}` | `{"type":"string"}` — only that keyword is removed |
| `title` has the wrong type | `{"type":"string","title":null}` | `{"type":"string"}` — only that keyword is removed |
| `$id` / `pattern` is `null` | `{"type":"string","pattern":null}` | `{"type":"string"}` — only that keyword is removed |
| `$defs` / `$id` not at the root | `$defs` inside a subschema | The keyword is removed, everything else kept |
| Duplicates in a `type` array or in `required` | `{"type":["string","string"]}` | `{"type":["string"]}` |
| Negative bound value | `{"type":"string","minLength":-1}` | `{"type":"string","minLength":0}` |
| `enum` holds values the `type` rules out | `{"type":"string","enum":["a",1,"b"]}` | `{"type":"string","enum":["a","b"]}` — only the ill-typed values go |
| Every `enum` value is ruled out by `type` | `{"type":["null"],"enum":[false]}` | `{"type":["null"]}` — the whole `enum` disappears |
| Lower bound above upper bound | `{"type":"string","minLength":10,"maxLength":2}` | `{}` — the field loses all constraints |
| Multiple `type`s plus other structural keywords | `{"type":["string","integer"],"minLength":1}` | `{}` — the field loses all constraints |
| `enum` has no overlap with the definition's `enum` | `{"type":"string","enum":["x"],"$ref":"#/$defs/S"}`<br>`S = {"type":"string","enum":["y"],"minLength":9}` | `{"type":"string","minLength":9}` — only the `enum` is dropped |
| `type` has no overlap with the definition's `type` | `{"type":"string","minLength":3,"$ref":"#/$defs/S"}`<br>`S = {"type":"number","minimum":5}` | `{}` — the field loses all constraints |

`required` is listed separately; these constructs are all legal under the spec, and the handling rules are:

| Case | Input | Rewrite result |
| --- | --- | --- |
| A required property is not declared in `properties` | `properties: {"a":{...}}`<br>`required: ["a","b"]` | `required: ["a"]` — only `b` is removed; `a` stays required |
| `required` without `properties` | `{"type":"object","required":["a"]}` | `{"type":"object"}` — the whole `required` disappears |
| `required` with a non-`object` `type` | `{"type":"string","required":["a"]}` | `{"type":"string"}` — `required` only applies to objects anyway |

The empty string is a perfectly normal property name: the spec puts no constraint on the keys of `properties`, and the downstream constraint engine can generate `{"": ...}`, so `{"":{"type":"string"}}` is kept as written and `required: [""]` works as usual. It follows the same rule as any other name: it is pruned only when it is not declared in `properties`.

The last four rows of the first table (lower bound above upper bound, multiple `type`s, `enum` / `type` with no overlap on either side) are rejected outright at `strict` and above, because no instance can satisfy them. Every other row is accepted at all levels.

The rewrite's trade-off is **drop only the one contradicting keyword and keep whatever the two sides still agree on**. In the row where the `enum`s do not overlap, both sides agree the value is a string — they just disagree on which strings — so `type` stays, `minLength` keeps the stricter 9, and only the `enum` disappears.

A contradiction on `type` is the exception: the whole field is emptied. Every other keyword only means something relative to a type, and combining one side's string length with the other side's numeric lower bound on a typeless field describes something that does not exist.

A full clash between `enum` and a sibling `type` is a compromise: one of the two has to go, and `type` stays. Keeping the `enum` would admit values of a type the schema explicitly ruled out; keeping the `type` only loosens, and is the tightest result still expressible.

## 6. Can recursive references be used

Self-reference itself is supported. The criterion is **whether a finite JSON document can be constructed**, not whether self-reference exists.

| Construct | Verdict | Reason |
| --- | --- | --- |
| `{"properties":{"next":{"$ref":"#"}}}` | Passes | `next` is optional; `{}` satisfies it |
| `{"properties":{"children":{"type":"array","items":{"$ref":"#"}}},"required":["children"]}` | Passes | `{"children":[]}` satisfies it; an empty array asks nothing of `items` |
| `{"properties":{"next":{"$ref":"#"}},"required":["next"]}` | Rejected | Every level demands a next one |
| The array version of the previous row, plus `"minItems":1` | Rejected | The array cannot be empty, so the recursion cannot bottom out |

## 7. Unsupported keywords

`allOf`, `oneOf`, `not`, `if` / `then` / `else`, `const`, `format`, `$schema`, `$comment`, `$anchor`, `$dynamicRef`, and any other unknown keyword: `lite` accepts them, `Canonical` deletes them.

## 8. Behaviour changes versus the old version

| Construct | Old behaviour | Now |
| --- | --- | --- |
| Constraints such as `type` or `minLength` beside `$ref` | Rejected | Accepted; merged stricter-wins on rewrite |
| `type` beside `$ref` with compatible types | Rejected | Accepted |
| `type` beside `anyOf` | Rejected | Accepted; distributed into every branch on rewrite |
| Same keyword both at the parent and inside an `anyOf` branch | Rejected | Accepted; distributed and merged stricter-wins on rewrite |
| A required property not declared in `properties` | Rejected | Accepted; only that entry is removed on rewrite |
| `required` without `properties`, or with a non-object `type` | Rejected | Accepted; `required` is removed on rewrite |
| `enum` holding values the `type` rules out | Rejected | Accepted; those values are removed on rewrite |
| Empty-string property name | The property was deleted on rewrite | Kept as written; listing it in `required` works as usual |
| Required properties forming a reference cycle | Accepted, but no valid output could be generated | Rejected |
| Required array self-reference without `minItems` | Rejected (a misjudgement) | Accepted |
| Lower bound above upper bound | Accepted; both bounds were deleted on rewrite | `lite` accepts but degrades the field to `{}`; `strict` and above reject |

There is one more pervasive change in rewrite results: previously, a single spot needing a rewrite could make `Canonical` return an empty `{}` and lose the whole set of constraints; now only the field at fault degrades, and everything else is preserved intact.
