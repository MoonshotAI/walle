# walle
A Moonshot AI flavored Json schema validator.

## Entry Point
The main entry point is `walle.go`, which provides two core APIs:
- `ParseSchema`: Parses input JSON schema string and creates a walle schema instance
- `Schema.Validate`: Validates the schema with optional configurations
  - Three validation levels:
    - **strict**: Most comprehensive validation including potentially "harmless" checks (e.g., no duplicate items). Aligns with model training behavior.
    - **lite**: Semantic correctness validation. The most permissive level required by Moonshot AI server for efficient structured generation.
    - **loose**: Minimal validation that relies more on model capabilities. (Not recommended for production use)

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

// Validate the schema with custom options
err = schema.Validate(
    walle.WithValidateLevel(walle.ValidateLevelStrict),
)
...
```

### Python
The Python interface provides a simple wrapper around the walle Go package. To use it, first build the shared library by running `./build.sh` in the `c-shared` directory, then refer to the implementation in `c-shared/walle.py` for usage details.

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
...

# With custom configuration
config = {
    "validateLevel": "strict",
    "maxEnumItems": 3
}
validator.validate_schema(schema, config)
...
```
