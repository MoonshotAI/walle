> 🌐 **语言 (Language):** [English](./walle.md) | 简体中文

# Json Schema Validator（walle）

本文档说明 **walle** 对 JSON Schema 的校验规则与错误分类，遵守 [**Moonshot Flavored JSON Schema Spec**（MFJS）](./mfjs-spec.zh.md)。

## 术语与前置信息

| 术语 | 含义 |
| --- | --- |
| **walle** | MFJS 的 schema 校验器（本仓库） |
| **MFJS** | Moonshot Flavored JSON Schema Spec |
| **root** | Schema 根层，对应 JSON Pointer `/` |
| **ANY** | `null` / `boolean` / `object` / `array` / `number` / `integer` / `string` 之一 |

## 错误检查类型

| Categories/错误分类 | rules | details |
| --- | --- | --- |
| Structural Errors/结构错误 | subschema需要显式指定type字段。与 anyOf 或 $ref 同级时 type 合法（2020-12 下同级关键字按 AND 生效），lite 放行：与 anyOf 同级时 Canonical 把 type 分发进每个分支，与 $ref 同级时折叠进被引用的 schema。与被引用 schema 的 type 交集为空时属恒假，lite 仍放行但 Canonical 把该子 schema 退化为 {}，strict 及以上拒绝 | 两种例外:<br>情况1：完整schema == {} 代表ANY<br>情况2："additionalProperties" : {} 代表ANY<br>注意：非上面两种情况暂时不支持 {} 自动推导为ANY，比如<br><br><pre><code class="language-json">"properties": {&#10;  "key1": {},&#10;  "key2": {}&#10;}</code></pre> |
|  | 只支持null/boolean/object/array/number/integer/string 这7种types, root schema必须是dict |  |
|  | 只支持MFJS约定的范围的keywords | 分为两种情况：非法的keyword、合法但是MFJS不支持的keyword |
|  | 各种keywords的位置要符合json schema规范 | object类型中只允许type/properties/required/additionalProperties/anyOf/$ref 这些keywords |
|  | Object: required 列举的字段必须是 properties 中声明过的 |  |
|  | Object: properties中的keys不能重复 |  |
|  | Object: Objects have limitations on nesting depth and size | 默认上限：所有 object 累计 3000 个 properties 键、嵌套 30 层、整份 schema 120000 字节。调用方可用 WithMaxTotalPropertiesKeysNum / WithMaxSchemaDepth / WithMaxSchemaSize 调整 |
|  | Object: properties的key的名字不能是"$defs"/"$ref"/"anyOf"/"required"/"additionalProperties" |  |
|  | type keyword 与 anyOf 或 $ref 同级都合法，lite 放行；ultra 报出来，由 Canonical 分发进 anyOf 各分支、或折叠进被引用的 schema | 分级行为详见 [validation-principles.zh.md](./validation-principles.zh.md) |
|  | anyOf 元素个数>=1, <= 500（默认，可用 WithMaxAnyOfItems 调整） |  |
|  | $defs/$id 只能定义在root level |  |
|  | $ref 需要指向本schema自身或者本schema的subschema 的合法$defs，不支持remote/跨文件/url | 指向自身: "$ref": "#" |
|  | $ref/$defs 需要有合理终止条件，禁止infinite recursive loop | 判据是能否构造有限实例：可选属性成环、数组成环且未设 minItems 都可终止；必填属性成环、minItems >= 1 的数组成环则拒绝 |
|  | $ref的位置要合法 | 可以存在于："properties", "defs", "additionalProperties", "anyOf", "items", "root" 相关位置处 |
|  | Array: Limitations on enum size | 单个 enum 默认最多 1000 项。另有一条字符串总长限制（上限 75000 字符），但它只在 enum 项数超过 2500 时才检查——该阈值高于 1000 的项数上限，因此默认配置下不会触发，只有调用方用 WithMaxEnumItems 放宽项数后才生效 |
|  | Array:  "items"可以不定义，如果定义则内容不能为空 |  |
|  | Limitations on total string size | 同上：enum 项数默认上限 1000，字符串总长上限 75000 字符、触发阈值 2500 项。长度基于 go 标准库 json.Marshal 统计，因此空白字符不被计入字符串大小 |
|  | anyOf 与 $ref 同级都允许约束关键字，root 额外允许"$defs"/"$id" | 按 2020-12 的 AND 语义合法：lite 放行。ultra 报出来，由 Canonical 把 anyOf 同级的约束分发进每个分支、把 $ref 同级的内联展开，同名关键字都取更严的一侧 |
|  | default关键字只支持boolean/number/string/integer/null这些类型，默认值需要与类型匹配 |  |
| Data Type Errors/类型错误 | type和enum value类型不匹配，如integer vs 3.67 |  |
|  | type的value类型必须是字符串 |  |
|  | required items的类型必须是字符串 |  |
|  | enum items的类型必须是数组 |  |
|  | type如果是数组，与enum组合使用的时候有限制 | 情况1：type不与enum共同使用时：则支持任意合法类型，但是只允许同subSchema出现description/title关键字，如果是root，那么额外允许$id/$defs情况2：type与enum共同使用：1、type 数组的大小只能是1或者2 2、如果是2，只支持 [number/integer/string/boolean] + null的组合方式，类型只能是这4种之一和null的组合3、对应enum的数值类型必须一致 |
|  | enum 枚举值元素类型必须相同 | 如果是number，那么[2, 3.33] 是合法的 |
|  | integer/number对应的enum/min/max的值需要在合理数值范围之内 | 整数只支持十进制，不支持其它进制 浮点数不支持科学计数法double:  (-1.8e308, 1.8e308)int: [-2**53 + 1, 2**53 - 1] |
|  | properties 类型必须是结构体 |  |
|  | description/title/$id 类型必须是字符串 |  |
|  | anyOf 类型必须是数组，每个items必须是合法的subSchema |  |
|  | additionalPropertis value的类型只能是boolean或者object | 如果不指定，默认值是true |
|  | 各类型存在的min/max需要满足 min <= max 的限制条件 | min > max 属恒假：lite 放行但 Canonical 把该子 schema 退化为 {}，strict 及以上拒绝。另："type": "integer" 而 mininum/maxinum 为浮点时 schema 视为合法，但 enforcer 会进行 Rounding，算法选择 Rounding toward to zero (Truncate) |
|  | $defs 的key name不能包括 / 字符 |  |
