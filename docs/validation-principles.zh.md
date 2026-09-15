> 🌐 **语言 (Language):** [English](./validation-principles.md) | 简体中文

# walle 校验规则与规范化说明

这份文档回答三个问题：**我的 schema 为什么被拒了**、**怎么改**、**通过之后它会被改写成什么样**。

规则总表见 [walle.zh.md](./walle.zh.md)，规范定义见 [mfjs-spec.zh.md](./mfjs-spec.zh.md)，与 JSON Schema 2020-12 的逐条对照见 [mfjs-walle-vs-draft-2020-12.zh.md](./mfjs-walle-vs-draft-2020-12.zh.md)。

## 一、四个级别

| 级别 | 行为 |
| --- | --- |
| `loose` | 完全不校验，任何 schema 都通过 |
| `lite` | 拒绝写错的、引用解析不了的、以及无终止递归 |
| `strict` | 在 `lite` 之上，额外拒绝数值边界写错或自相矛盾的 schema |
| `ultra` | 最严，`Canonical` 用它找出所有需要改写的地方 |

`lite` 只管一件事：**这份 schema 是不是合法且有解**。合法但下游用不了的写法它一律放行，交给 `Canonical` 改写。

**被接口拒了**看第二节；**通过了但生成结果不符合预期**看第三到五节。

## 二、会被拒绝的写法

以下都是 `lite`（接口默认）会拒绝的

### 2.1 关键字的值写错了

| 情形 | 报错 |
| --- | --- |
| `type` 不是字符串或字符串数组 | `type must be string or array of strings` |
| `type` 数组为空 | `type array cannot be empty` |
| `type` 数组里有非字符串元素 | `invalid type in type array` |
| `type` 不是那 7 种之一 | `invalid type` |
| `properties` 不是对象 | `properties must be an object` |
| 某个属性的 schema 不是对象 | `property schema for 'x' must be an object` |
| `required` 不是数组 | `required must be an array` |
| `required` 里有非字符串元素 | `items in required array must be strings` |
| `enum` 不是数组 | `enum must be an array` |
| `enum` 是空数组 | `enum array cannot be empty` |
| `items` 不是对象 | `items must be an object` |
| `anyOf` 不是数组 | `anyOf must be an array` |
| `anyOf` 是空数组 | `anyOf must have 1-500 items` |
| `$ref` 不是字符串 | `$ref must be a string` |
| `pattern` 不是字符串（`null` 除外） | `pattern must be a string` |
| `description` 不是字符串 | `description must be a string` |
| `additionalProperties` 不是布尔值或对象 | `additionalProperties must be a boolean or an object` |
| `$defs` 不是对象 | `$defs must be an object` |
| `$id` 不是字符串（`null` 除外） | `$id must be a string` |
| 数值边界关键字的值是 `null` | `minItems must be an integer` / `minimum must be a number` 等 |

规模上限同样在 `lite` 生效：整份 schema 120000 字节、嵌套 30 层、所有对象累计 3000 个属性键、单个 `enum` 1000 项、`anyOf` 500 个分支。

数值边界关键字（`minLength` `maxLength` `minimum` `maximum` `minItems` `maxItems`）的值不是整数或为负数时（`null` 除外），`lite` 放行并在改写时纠正，`strict` 及以上才拒绝。

`title` 类型不对，以及 `$id` / `pattern` 为 `null` 时，`lite` 同样放行，`Canonical` 删掉该关键字，`strict` 及以上才拒绝。`pattern` / `$id` 的其它非字符串值仍从 `lite` 拒绝。`description` 不是字符串（含 `null`）仍从 `lite` 拒绝。

### 2.2 引用无法解析

只支持指向本文档内部 `$defs` 的引用。

| 情形 | 报错 |
| --- | --- |
| 外部、跨文件、URL 引用，如 `"$ref":"https://example.com/s.json"` | `references must start with #/$defs/` |
| 指向 `#/$defs/` 以外的路径 | `references must start with #/$defs/` |
| 定义名为空，即 `"$ref":"#/$defs/"` | `definition name cannot be empty` |
| 引用的定义不存在 | `invalid $ref path: ...` |
| 用了 `$ref` 但整份 schema 没有 `$defs` | `$defs not found for reference: ...` |
| `$defs` 的键名里有 `/` | `$defs property name 'a/b' cannot contain '/' character` |

### 2.3 没有任何数据能满足

只有无终止递归属于这一类。矛盾的 `type` / `enum` / 数值边界不在此列——它们同样无解，但 `lite` 放行、改写时丢掉矛盾的约束，详见第五节。

| 情形 | 报错 |
| --- | --- |
| 必填属性构成引用环 | `detected infinite recursion without termination condition` |
| 互相引用且每一跳都必填 | 同上 |
| 数组自引用且 `minItems >= 1` | 同上 |

### 2.4 引用图过于复杂

菱形引用链（每个定义被多个属性引用、再层层指向下一个定义）会让引用遍历的成本随链长指数增长。遍历步数超过内置预算（500000 步）时直接拒绝：

| 情形 | 报错 |
| --- | --- |
| 菱形/长环引用链，引用遍历超出步数预算 | `reference graph is too complex to validate within the step budget` |

正常 schema 的遍历量比这低几个数量级，只有刻意构造的 schema 才会碰到这条。

## 三、$ref 同级关键字会被怎么合并

| 关键字 | 合并方式 |
| --- | --- |
| `minLength` `minItems` `minimum` | 取较大值 |
| `maxLength` `maxItems` `maximum` | 取较小值 |
| `type` | 取交集，`integer` 与 `number` 的交集是 `integer`；交集为空见第五节 |
| `enum` | 取交集；交集为空见第五节 |
| `required` | 取并集 |
| `properties` | 取并集；两侧都描述的属性按上面这些规则再合并一层 |
| `description` `title` `pattern` 等其余关键字 | 以使用处的值为准 |

使用处要求 `minLength: 5`、定义里是 `1`，合并结果取 `5`，定义里的 `maxLength: 50` 一并带过来：

```json
{
  "type": "object",
  "properties": { "name": { "minLength": 5, "$ref": "#/$defs/Name" } },
  "$defs": { "Name": { "type": "string", "minLength": 1, "maxLength": 50 } }
}
```

改写为：

```json
{
  "type": "object",
  "properties": {
    "name": { "type": "string", "minLength": 5, "maxLength": 50 }
  }
}
```

其余几种情形（下表省略了外层的 `"type":"object"` 和 `"properties"` 包裹）：

| 使用处 + 定义 | 改写结果 | 说明 |
| --- | --- | --- |
| `{"minLength":1,"$ref":"#/$defs/S"}`<br>`S = {"type":"string","minLength":1}` | `{"$ref":"#/$defs/S"}`，`S` 不变 | 两侧值相同，删掉重复的那个、继续共享定义 |
| `{"maxLength":20,"$ref":"#/$defs/S"}`<br>`S = {"type":"string","minLength":1}` | `{"type":"string","minLength":1,"maxLength":20}` | 使用处的关键字定义里没有，直接并入 |
| `{"description":"用户名","$ref":"#/$defs/S"}` | 原样保留，所有级别都通过 | 注解不影响约束 |
| `short: {"maxLength":10,"$ref":"#/$defs/S"}`<br>`long: {"maxLength":99,"$ref":"#/$defs/S"}`<br>`S = {"type":"string","minLength":1}` | `short: {"type":"string","minLength":1,"maxLength":10}`<br>`long: {"type":"string","minLength":1,"maxLength":99}` | 多个使用处各自独立特化，互不影响 |

**例外：被引用的定义是递归的，同级约束会被丢弃。** 

```json
{
  "type": "object",
  "properties": { "tree": { "minLength": 3, "$ref": "#/$defs/N" } },
  "$defs": {
    "N": { "type": "object", "properties": { "child": { "$ref": "#/$defs/N" } } }
  }
}
```

改写后 `minLength: 3` 消失，`tree` 只受 `N` 约束。需要这个约束生效的话，不要写在递归定义的引用处，改为写进定义本身。

## 四、anyOf 同级关键字会被怎么分发

写在 `anyOf` 旁边的约束，和命中的那个分支同时生效。改写时把父层的约束**逐个分发进每个分支**：

```json
{
  "type": "object",
  "properties": {
    "v": { "type": "string", "anyOf": [{ "minLength": 1 }, { "maxLength": 9 }] }
  }
}
```

改写为：

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

分发规则：

- **同名关键字按第三节的取严规则合并**，父层 `minLength: 20` 遇上分支的 `minLength: 10`，分支留下 20。
- **和父层矛盾的分支直接删掉**，因为没有数据能走到它。父层 `"type":"string"` 配上 `[{"minLength":1},{"type":"integer"}]`，`integer` 那个分支消失。
- **所有分支都被删掉时该字段清空为 `{}`**，这种 schema 本来就没有数据能满足。
- **`description` / `title` 不分发**，留在父层，它们不是约束。

分支里写 `$ref` 时，矛盾也算在内：父层 `"type":"string"` 而分支引用的定义是 `{"type":"number"}`，该分支同样被删。

## 五、其他会被改写的写法

这些 `lite` 都放行，`Canonical` 会改写。下表省略了外层包裹。

| 情形 | 输入 | 改写结果 |
| --- | --- | --- |
| 用了不支持的关键字 | `{"type":"string","format":"uuid"}` | `{"type":"string"}`，只删该关键字 |
| `title` 类型不对 | `{"type":"string","title":null}` | `{"type":"string"}`，只删该关键字 |
| `$id` / `pattern` 为 `null` | `{"type":"string","pattern":null}` | `{"type":"string"}`，只删该关键字 |
| `$defs` / `$id` 没写在根层 | 子 schema 里带 `$defs` | 删掉该关键字，其余保留 |
| `type` 数组或 `required` 里有重复项 | `{"type":["string","string"]}` | `{"type":["string"]}` |
| 边界值为负 | `{"type":"string","minLength":-1}` | `{"type":"string","minLength":0}` |
| `enum` 里有 `type` 不允许的值 | `{"type":"string","enum":["a",1,"b"]}` | `{"type":"string","enum":["a","b"]}`，只去掉不合类型的值 |
| `enum` 里全部值都不合 `type` | `{"type":["null"],"enum":[false]}` | `{"type":["null"]}`，`enum` 整个消失 |
| 下界大于上界 | `{"type":"string","minLength":10,"maxLength":2}` | `{}`，该字段失去全部约束 |
| 多个 `type` 且带其他结构关键字 | `{"type":["string","integer"],"minLength":1}` | `{}`，该字段失去全部约束 |
| `enum` 与定义的 `enum` 无交集 | `{"type":"string","enum":["x"],"$ref":"#/$defs/S"}`<br>`S = {"type":"string","enum":["y"],"minLength":9}` | `{"type":"string","minLength":9}`，只丢掉 `enum` |
| `type` 与定义的 `type` 无交集 | `{"type":"string","minLength":3,"$ref":"#/$defs/S"}`<br>`S = {"type":"number","minimum":5}` | `{}`，该字段失去全部约束 |

`required` 单独列出来，这几种写法在规范里都合法，处理规则：

| 情形 | 输入 | 改写结果 |
| --- | --- | --- |
| 必填的属性没在 `properties` 里声明 | `properties: {"a":{...}}`<br>`required: ["a","b"]` | `required: ["a"]`，只去掉 `b`，`a` 仍然必填 |
| 有 `required` 但没有 `properties` | `{"type":"object","required":["a"]}` | `{"type":"object"}`，`required` 整个消失 |
| 有 `required` 但 `type` 不是 `object` | `{"type":"string","required":["a"]}` | `{"type":"string"}`，`required` 本来就只对对象生效 |

空字符串是个正常的属性名，规范没有限制 `properties` 的键，下游的约束引擎也能生成 `{"": ...}`，所以 `{"":{"type":"string"}}` 原样保留、`required: [""]` 也照常生效。它和别的名字走同一条规则：只有没在 `properties` 里声明时才会被剔除。

第一张表的最后四行（下界大于上界、多个 `type`、两处 `enum` / `type` 无交集）`strict` 及以上级别会直接拒绝，因为它们都没有任何数据能满足。其余各行所有级别都放行。

改写的取舍是**只丢掉矛盾的那一个关键字，两侧还能达成一致的部分留着**。`enum` 与定义的 `enum` 无交集那行，两侧都同意值是字符串，只是不同意是哪些字符串，所以 `type` 保留、`minLength` 按取严规则留下更大的 9，只有 `enum` 消失。

矛盾出在 `type` 上是例外，整个字段清空：其余关键字都要依附于某个类型才有意义，把一侧的字符串长度和另一侧的数值下限凑在一个无类型的字段里，描述的是一个不存在的东西。

`enum` 和同级 `type` 全冲突时是个折中：两者留一个，留下的是 `type`。留 `enum` 会放进 schema 明确排除掉的类型的值，留 `type` 只是放宽，是能表达出来的最紧的结果。

## 六、递归引用能不能用

自引用本身是支持的。判据是**能否构造出有限的 JSON**，而不是有没有自引用。

| 写法 | 判定 | 原因 |
| --- | --- | --- |
| `{"properties":{"next":{"$ref":"#"}}}` | 通过 | `next` 可选，`{}` 就满足 |
| `{"properties":{"children":{"type":"array","items":{"$ref":"#"}}},"required":["children"]}` | 通过 | `{"children":[]}` 满足，空数组对 `items` 无要求 |
| `{"properties":{"next":{"$ref":"#"}},"required":["next"]}` | 拒绝 | 每一层都还需要下一层 |
| 上一行的数组版本再加 `"minItems":1` | 拒绝 | 数组不能为空，递归无法收尾 |

## 七、不支持的关键字

`allOf`、`oneOf`、`not`、`if` / `then` / `else`、`const`、`format`、`$schema`、`$comment`、`$anchor`、`$dynamicRef`，以及任何其他未知关键字：`lite` 放行，`Canonical` 删除。

## 八、与旧版本的行为差异

| 写法 | 旧行为 | 现在 |
| --- | --- | --- |
| `$ref` 同级带 `type`、`minLength` 等约束 | 拒绝 | 放行，改写时合并取严 |
| `type` 与 `$ref` 同级且类型兼容 | 拒绝 | 放行 |
| `type` 与 `anyOf` 同级 | 拒绝 | 放行，改写时分发进各分支 |
| 同一关键字既在父层又在 `anyOf` 分支里 | 拒绝 | 放行，改写时分发并取严 |
| 必填的属性没在 `properties` 里声明 | 拒绝 | 放行，改写时只去掉这一项 |
| 有 `required` 但没有 `properties`、或 `type` 不是对象 | 拒绝 | 放行，改写时删掉 `required` |
| `enum` 里有 `type` 不允许的值 | 拒绝 | 放行，改写时去掉这些值 |
| 属性名是空字符串 | 改写时删掉该属性 | 原样保留，`required` 里列它也照常生效 |
| 必填属性构成引用环 | 放行，但无法生成有效结果 | 拒绝 |
| 必填数组自引用、未写 `minItems` | 拒绝（误判） | 放行 |
| 下界大于上界 | 放行，改写时删掉两个边界 | `lite` 放行但该字段退化为 `{}`，`strict` 及以上拒绝 |

改写结果还有一处普遍变化：过去只要 schema 里有一处需要改写，`Canonical` 就可能返回空的 `{}`、丢掉整份约束；现在只有出问题的那个字段会退化，其余部分完整保留。
