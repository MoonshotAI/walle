# walle
A Moonshot AI flavored Json schema validator.

## Entry Point
The main entry point is `walle.go`, which provides two core APIs:
- `ParseSchema`: Parses input JSON schema string and creates a walle schema instance
- `Schema.Validate`: Validates the schema with optional configurations
  - Validation levels:
    - **ultra** / **default**: Most comprehensive validation including potentially "harmless" checks (e.g., no duplicate items).
    - **strict**: The most permissive level required by Moonshot AI server for efficient structured generation.
    - **lite**: Skips a subset of rules that **strict** enforces.
    - **loose**: Skips schema validation in `Schema.Validate` and relies more on model capabilities.

## Usage

## cli
```
go install github.com/moonshotai/walle/cmd/walle@latest
```
```
walle -schema '{"type": "object"}' -level strict
walle -schema-file your_schema.json
```

### go package
```go
import "github.com/moonshotai/walle"

// Define your JSON schema
schemaStr := `{
    "type": "object",
    "properties": {
        "name": {"type": "string"},
        "age": {"type": "integer"}
    },
    "required": ["name"]
}`

// Create a schema instance
schema, err := walle.ParseSchema(schemaStr)
...

// Validate the schema with default options
err = schema.Validate()
...

// Canonical JSON string
canonicalJSON, warnErr := schema.Canonical()

// Validate the schema with custom options
err = schema.Validate(
    walle.WithValidateLevel(walle.ValidateLevelStrict),
)
...
```

### Python
The Python interface is packaged as `walle`. Release wheels include `libwalle.so`,
so users can import it directly after installing the wheel.

For local development, build the shared library before building or installing
the package:

```sh
cd python/c-shared
./build.sh
cd ../..
python -m pip wheel . -w dist --no-deps
```

Example:
```python
from walle import WalleValidator

# Initialize validator
validator = WalleValidator()

# Validate a schema
schema = '''
{
    "type": "object",
    "properties": {
        "name": {"type": "string"},
        "age": {"type": "integer"}
    },
    "required": ["name"]
}
'''
validator.validate_schema(schema)

# Canonical schema
canonical_json, warning = validator.canonical_schema(schema)

# With custom configuration
config = {
    "validateLevel": "strict",
    "maxEnumItems": 3
}
validator.validate_schema(schema, config)
```

Tool-calling schema helpers:
```python
from walle import ms_tool_req_cvt, ms_tool_req_simplify

internal_json = ms_tool_req_cvt(request)
simplified_json, warnings = ms_tool_req_simplify(request)
```
