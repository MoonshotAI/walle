> 🌐 **语言 (Language):** [English](./mfjs-spec.md) | 简体中文

# Moonshot Flavored JSON Schema Spec

Moonshot Flavored JSON Schema (MFJS) 是 Moonshot AI 为解决大语言模型（LLM）在理解完整 JSON Schema 规范时的困难而推出的一个经过精心裁剪的 JSON Schema 子集，专为 LLM 的交互场景优化。

它主要应用于定义 **Tool Calling** 的 `parameters` 字段和 **Response Format**，通过移除原规范中的复杂特性并保留核心结构，确保模型能准确理解数据要求，并生成结构化的 JSON 输出。

虽然 MFJS 支持基本的 JSON Schema 结构和类型系统，但为确保大模型能准确理解和生成，对以下功能进行了精简或限制：

* **`$defs`, `$ref` 受限支持：** `$defs` 必须定义在 schema 的根级别 (root level)；`$ref` 仅支持引用**本文档内部**的 subschema（通过 `$defs` 索引，或使用 `"$ref": "#"` 自我引用）。
* **不支持外部资源：** 暂不支持 Meta-Schemas，也不支持任何基于 HTTP 的外部网络资源引用。
* **不支持元数据与注解：** 暂不支持 `title`、`$comment` 等描述性元数据，也不支持 `format` 格式注解字段。
* **不支持高级数组定义：** 暂不支持使用 `prefixItems` 或 `unevaluatedItems` 等方式模拟实现元组（Tuple）。
* **不支持复杂校验：** 暂不支持如 `exclusiveMinimum`，`exclusiveMaximum`，`minContains`, `maxContains` 等复杂校验定义。

在模型和接口的交互中，我们实现了一个基于 JSON Schema 的子集，我们会识别这个格式，并且努力给出符合规范的输出。

本文档试图明确无误地规定一个 JSON Schema 子集格式，它包括许多我们对这个 JSON Schema 子集的定义和示例。

## 预备知识

### JSON 文档
参考 [JSON](https://www.json.org/json-en.html) 的定义，JSON 是一种轻量级的、基于文本的、独立于语言的语法，用于定义数据交换格式。它源自 ECMAScript 编程语言，但独立于具体编程语言。

### Schema 和 Instance
JSON Schema 根据数据模型解释文档。根据此数据模型解释的 JSON 值称为 "实例（Instance）"。

通常，在 Moonshot 的交互中，我们输入的参数是一个定义数据格式的合法 JSON（即 JSON Schema），而语言模型通常会输出一个符合该定义格式的 JSON（即实例 Instance）。

### JSON 实例的数据类型
实例具有六种原始类型之一，并根据类型具有一系列可能的值：

- `null`: 一个 JSON "null" 值，在 python 通常为 None
- `boolean`: 一个 JSON 的 "true" 或 "false" 值，在 python 通常为 bool 类型的值
- `object`: 一个无序的属性集合，将字符串映射到实例，来自 JSON "object" 值，在 python 可能是一个 Dict 类型
- `array`: 一个实例的有序列表，来自 JSON "array" 值，在 python 可能是一个 List
- `number`: 一个任意精度，基于 10 的十进制数值，来自 JSON "number" 值，在 python 中可能映射为一个 float
- `string`: 一个字符串，来自 JSON "string" 值，或者在 python 中为一个 str

包括数字的不同词法表示形式在内的空白字符和格式化问题，在数据模型层面是相等的，因此不在 JSON Schema 的范围内。

在 MFJS 的 "type" 定义中，我们定义的类型系统和此处不完全一致，比如 "integer" 类型严格约束了它会输出一个整数，它是一个合法的 "type" 但在 JSON 的数据类型中可能表现为 number。

## 一个自顶向下的案例分析

假设在 Tool Use 场景中，我们需要提供一个工具，告知模型我们有一个搜索引擎。那么我们可能会定义这样一个工具来描述自己的功能：

```json
{
  "type": "function",
  "function": {
    "name": "web_search",
    "description": "Search the web for information",
    "parameters": {}
  }
}
```

这个工具自我介绍名为 web_search，自述可以进行搜索，而在参数的地方，我们可以定义一个 JSON Schema，这个用于告知模型应该返回的数据的格式和内容。

模型如果需要进行搜索了，它会按照需要调用 web_search，并且使用这个 Schema 指定的参数来告知用户，这儿有一个工具调用，而参数是如上，也就是前面定义中的一个合法的"实例（Instance )"。

接下来我们都以这个需求为例，尝试逐步迭代例子中 parameters 的值，使得它的结果更加符合调用者的需求。

### 从零开始

假如我们什么也不做，即 parameters 的值为 `{}`

这意味着您允许模型在调用 `web_search` 时的 `arguments` 为任何内容，包括空内容。

如下的例子在上面定义下全部都是合法的：

```json
null
true
"a sample query"
{ "qs": "a sample query"}
["first query", "second query"]
```

### 定义类型

考虑到易于解析，并且我们会希望模型能输出一些复杂的查询条件，我们会希望输出的结构是一个合法的 JSONObject。那么我们可以定义一个类型约束：

```json
{
  "type": "object"
}
```

这样模型输出的内容一定可以被 python 解析，并且根对象是一个 Dict。

如下的例子在上面的定义下都是合法的：

```json
{ "qs": "a sample query"}
{ "query": "another sample query"}
{ "query": 42 }
```

### 定义参数列表

我们可以固定参数，查询必须使用字段 `qs`，并且提示可选参数 `lang` 筛选语言，`limit` 搜索 top_n 个结果。

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

如上面的例子，我们使用 `required` 定义 `qs` 为必填字段，在 `properties` 中，我们定义了这三个字段，每一对 key-value 对中，key 是需要输出的字段 id，而 value 是一个子 Schema，子 Schema 是一个自包含结构，单独抽取出来也适用于本 Spec 的定义。

如下的例子在上面的定义下都是合法的：

```json
{ "qs": "a sample query"}
{ "qs": "a sample query", "limit": 20 }
{ "qs": "a sample query", "lang": "Chinese" }
{ "qs": "a sample query", "limit": 20, "lang": "Chinese", "foo": "bar" }
```

### 构建更复杂的表达查询

有的时候，我们可能需要进行多个不同的查询语句分别查询，人工合并结果。那么我们可以利用 `anyOf` 关键词组织更加复杂的 `qs`。

此外，我们可能考虑关闭其它可能的参数，我们可以进一步修改如下：

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

于是我们的输出即成为两种可控类型的其一。

如下的例子在上面的定义下都是合法的：

```json
{ "qs": "a sample query"}
{ "qs": ["first query", "second query"]}
```

最后，如果我们把这个 JSON Schema 回填回去后，请求可能类似这样：

```json
{
  "type": "function",
  "function": {
    "name": "web_search",
    "description": "Search the web for information",
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

### 递归模式定义
`$defs`、`$ref` 支持递归结构定义。例如链表节点模式：

示例：节点引用自身，或用 null 表示链表末尾

```json
{
  "type": "object",
  "properties": {
    "linked_list": {
      "$ref": "#/$defs/linked_list_node"  // Reference within the same schema
    }
  },
  "$defs": {          // Must be defined at the root level
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
              "type": "null"  // Linked list termination
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

## 字段定义

### Meta Data 字段

Meta Data 字段提供关于 Schema 的注解信息，它会按照所在位置提供给语言模型，帮助模型按照说明写入信息。Meta Data 通常不直接体现在输出结果上，但是部分情况下会作为输入提供给模型。

#### description
**类型**: str

description 字段提供请求所描述输出目的的简短描述，类型必须是一个 String。它可以在 Schema 任何位置有效，但是只在合适的地方会提示模型。

特别的，`"response_format": { "type": "json_object" }` 方式开启 json mode 时这儿不生效。

#### default
**类型**: Any

default 字段为目标输出提供一个默认值。它会提示模型默认可能会填的值。这儿不做类型检查，需要调用者自行保证其值类型内容和所在位置期望的内容相匹配。

特别的，`"response_format": { "type": "json_object" }` 方式开启 json mode 时这儿不生效。

### Applicator 字段

Applicator 字段用于指定如何将 Schema 应用于 JSON 实例的特定部分。一般的，它适用于任何 Schema 内部的控制。

#### anyOf
**类型**: List[Schema]

输出的实例至少与此字段定义的一个子 Schema 匹配。

JSON Schema 包括一些用于将模式组合在一起的关键词，我们可以用 anyOf 把多个类型定义组合起来，起到返回为类似 python 的 Union 效果。

#### properties
**类型**: Dict[str, Schema]

properties 是一组 KV 对。每一对 key-value 对中，key 是需要输出的字段 id，而 value 是一个子 Schema，子 Schema 是一个自包含结构。

通常这个只在 object 的类型中生效。

#### additionalProperties
**类型**: Union[bool, Schema]

additionalProperties 默认值为 `{}`，也就是说如果不设置它，那么模型除了 properties 中的字段外，其它字段默认也是允许产生的，并且类型是任意的。

比如如果我们在 properties 中定义了 `qs` 为 string, 那么模型仍然可能被允许输出类似 `city: beijing` 这样的字段。

我们也可以在 additionalProperties 的值中增加选项，比如使用 `{"additionalProperties":{"type":"string"}}` 来限定其他未列举的 properties 的值的类型，以保证生成的额外的字段都是期望的类型。

在目前的模型行为上，我们模型可能不倾向于生成额外的字段，如果期望保持兼容，可以考虑设置 `"additionalProperties": false` 来禁止未来可能的生成其他字段。

#### items
**类型**: Schema

当 `type` 为 `array` 的时候，它定义了每个 array 元素的子 schema。

### Validation 字段

Validation 字段用于定义目标实例的内容格式，它主要应用于叶子结点的值。

#### type
**类型**: str

目前我们枚举支持如下类型：

| 类型 | Python 对应类型 | 描述 |
|-----|-----------------|------|
| null | None | JSON中的空值常量 |
| boolean | bool | JSON中的布尔值常量，包括 true 和 false |
| object | Dict | JSON对象，由键值对组成 |
| array | List | JSON数组，有序的值集合 |
| number | Union[int, float] | JSON数值，包括整数和浮点数 |
| integer | int | JSON中的整数数值 |
| string | str | JSON字符串，由双引号括起来的文本 |

和 JSON 实例类似，只是增加 `integer` 类型，显式声明输出的是一个整数类型的数字。

#### enum
**类型**: List[Union[float, int, str]]

枚举可能的输出。注意，它的元素不是一个 Schema，而是字面量。

特别的，在这儿枚举值我们只支持基础类型中的 float, int 和 str，其它基础类型以及 object 和 array 暂不支持，并且不支持混合类型 (List 中的每个枚举值的类型必须相同，如不可以 str + int)。

#### required
**类型**: List[str]

定义了这一层实例必须包含此数组中列出的所有字段。默认情况下，所有参数都是可选的 (optional)。

## 参考文献

本文档的定义与设计参考了以下 JSON Schema 官方文献。由于 Moonshot Flavored JSON Schema (MFJS) 是专为大模型交互场景优化的精简子集，请在参考官方文献时特别注意以下事项：

* **规范优先级**：若本文档的定义与官方参考文献存在任何冲突或差异，请务必**以本文档的详细定义为准**。严格遵循本文档的约束可以避免在 Tool Calling 等场景中触发系统校验错误。
* **未支持特性说明**：若官方规范中的某些概念、关键字或特性在本文档中未被提及，则表示当前 MFJS 尚未提供官方支持。虽然将这些未列举的特性传入时可能不会被系统直接拦截，且可能被大模型观察到，但**我们不保证**它们能在最终的结构化输出中产生稳定、符合预期的结果。

### 规范性参考文献

- [JSON-Schema.org Specification Release Page](https://json-schema.org/specification#specification-section) - JSON Schema 官方规范发布主页
- [JSON Schema Core](https://json-schema.org/draft/2020-12/json-schema-core.html) - 定义了 JSON Schema 的核心基础架构
- [JSON Schema Validation](https://json-schema.org/draft/2020-12/json-schema-validation.html) - 定义了 JSON Schema 的校验关键字

### 信息性参考文献

- [JSON Schema Reference](https://json-schema.org/understanding-json-schema/reference#json-schema-reference) - JSON Schema 的官方参考手册
- [JSON Schema 2020-12](https://www.learnjsonschema.com/2020-12/) - 包含各个关键字的详细信息及丰富的应用示例
