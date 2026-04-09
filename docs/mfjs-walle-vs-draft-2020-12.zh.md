> 🌐 **语言 (Language):** [English](./mfjs-walle-vs-draft-2020-12.md) | 简体中文

# walle 与 JSON Schema Draft 2020-12 对标表

本文档说明 **walle** 校验器对各 **Draft 2020-12** 关键字的处理。**Keyword** 行顺序与 [learnjsonschema.com/2020-12](https://www.learnjsonschema.com/2020-12/) 各 vocabulary 主表一致。整体语义背景见 [Moonshot Flavored JSON Schema（MFJS）](./mfjs-spec.zh.md)；**具体校验规则以 [walle.zh.md](./walle.zh.md) 与本仓库实现为准。**

## 列说明

| 列 | 含义 |
| --- | --- |
| **Keyword** | Draft 2020-12 关键字（顺序见 learnjsonschema） |
| **walle** | [walle.zh.md](./walle.zh.md) 与本仓库校验实现 |

**图例：** ✅ 支持 / 允许 · ❌ 不支持或未承诺

---

## 1. Core

| Keyword | walle |
| --- | --- |
| `$id` | ✅ 仅 **root**；类型须为 string |
| `$schema` | ❌ |
| `$ref` | ✅ 只支持本地引用，禁止 remote / URL / 跨文件；须避免无限递归 |
| `$comment` | ❌ |
| `$defs` | ✅ 仅 **root**；**key 名不能含 `/`** |
| `$anchor` | ❌ |
| `$dynamicAnchor` | ❌ |
| `$dynamicRef` | ❌ |
| `$vocabulary` | ❌ |

---

## 2. Applicator

| Keyword | walle |
| --- | --- |
| `allOf` | ❌ |
| `anyOf` | ✅ 分支数量可能有限制；与 `type` / `$ref` **不得同级**（`type` 须在分支内） |
| `oneOf` | ❌ |
| `if` | ❌ |
| `then` | ❌ |
| `else` | ❌ |
| `not` | ❌ |
| `properties` | ✅ `type: object` 时；**key 不可** 为 `$defs`、`$ref`、`anyOf`、`required`、`additionalProperties`；**不可重复**；`required` 中每项须在 `properties` 中声明 |
| `additionalProperties` | ✅ 值为 **boolean 或 object**；未指定时 **默认 true** |
| `patternProperties` | ❌ |
| `dependentSchemas` | ❌ |
| `propertyNames` | ❌ |
| `contains` | ❌ |
| `items` | ✅ 只对 **`type: array`** 有意义：**可以不写** `items`；若写了，值必须是**一个**非空的 subschema，表示**每个数组元素**都用这一条规则校验。不支持 2020-12 里「`prefixItems` 管前几项、`items` 管剩余项」的组合用法 |
| `prefixItems` | ❌ |

---

## 3. Validation

| Keyword | walle |
| --- | --- |
| `type` | ✅ 七种：`null`、`boolean`、`object`、`array`、`number`、`integer`、`string`；**root 须为 object**；`type` 值须为 string 或合法数组（与 `enum` 组合时另有 walle 限制） |
| `enum` | ✅ 仅 **同类型** float / int / str；**全 schema 的 enum 数量可能有限制**；多 enum / 长字符串时有字符串总长等限制 |
| `const` | ❌ |
| `maxLength` | ✅ |
| `minLength` | ✅ |
| `pattern` | ❌ 即将支持 |
| `exclusiveMaximum` | ❌ |
| `exclusiveMinimum` | ❌ |
| `maximum` | ✅ **`type: number` / `integer`** 时允许；与 `minimum` 须 **maximum ≥ minimum** |
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
| `description` | ✅ 类型须为 string |
| `default` | ✅  moonshot server constrained decoding module 还没有支持，但是walle允许校验通过 |
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

## 9. 结构性规则

| 主题 | walle |
| --- | --- |
| 空 object subschema `{}` | 仅 **整份 root 为 `{}`** 或 **`additionalProperties` 值为 `{}`** 表示 **ANY**；**`properties` 内 `{}` 不自动视为 ANY** |
| `type` 与 `anyOf` / `$ref` 同级 | **禁止**；`type` 须在 `anyOf` / `$ref` 目标内部 |
| `object` 上允许的 keyword | **仅** `type`、`properties`、`required`、`additionalProperties`、`anyOf`、`$ref`（及注解规则中的 `description` / `title` 等） |
| `anyOf` / `$ref` 同级其它 keyword | 除 `description` / `title` 外，**root** 可额外有 `$defs` / `$id` |
| 嵌套与规模 | 如 **全 schema 中 object properties 数量可能有限制（累计）**、**嵌套层数可能有限制**（以 [walle.zh.md](./walle.zh.md) 为准） |
| 数值与枚举字面量 | 整数 **十进制**；浮点 **无科学计数法**；等 |

---

## 参考文献

- [learnjsonschema.com — JSON Schema 2020-12](https://www.learnjsonschema.com/2020-12/)
- 本仓库：[mfjs-spec.zh.md](./mfjs-spec.zh.md)、[walle.zh.md](./walle.zh.md)
