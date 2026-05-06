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

Multiprocessing demos(recommended: `spawn`) in `python/c-shared/example.py`: **`use_multiprocessing_spawn()`**
then **`use_multiprocessing_fork()`**. 

### Python + multiprocessing (fork safety)

`libwalle.so` embeds a Go runtime via CGO. After `fork(2)`, child processes must
not reuse CGO handles inherited from the parent; crashes often appear as segfaults
inside `cgofree` / `FreeErrString`.

**1. Prefer `spawn` over `fork` (only if you use multiprocessing)**

If your program uses `multiprocessing` or `concurrent.futures.ProcessPoolExecutor`,
note that Linux often defaults to `fork`. Prefer **`spawn`**: each worker starts a
fresh Python interpreter and reloads native code, so it does not inherit the
parent’s CGO / Go runtime state. Set it once at the main entry point (stdlib only;
no walle helper):

```python
import multiprocessing as mp

if __name__ == "__main__":
    try:
        mp.set_start_method("spawn", force=True)
    except RuntimeError:
        pass  # already set
    ...
```

If you never use process-based parallelism, you can ignore this item.

**2. If you must use `fork`**

Import `walle` before starting worker pools. The package registers
`os.register_at_fork(after_in_child=...)` to drop cached `ctypes.CDLL` handles;
the next access through `WalleValidator.lib` loads a fresh `libwalle.so` in the
child. This is a safety net when you cannot control the process model (default
`fork`, third-party pools, etc.).

**3. `WalleValidator` lifetime and process boundaries**

**Rule:** any process that calls into `libwalle` should construct its **own**
`WalleValidator()` there. Do not use a validator instance that was created in the
parent (or in another interpreter) inside a worker.

### Python + threading (main thread only)

Call `WalleValidator` (and helpers that use it, e.g. `ms_tool_req_simplify`) **only
from the thread that loaded `libwalle` first — in typical scripts, the main thread**.

Do **not** invoke `validate_schema` / `canonical_schema` from
`threading.Thread` targets or `ThreadPoolExecutor` workers: that path can **SIGSEGV**
inside CGO (`FreeErrString`). A Python `threading.Lock` does **not** fix this.

If you need more throughput, add **processes** (see *Python + multiprocessing*), not
more threads into `libwalle.so`.

### Python regression tests (fork + CGO)

After `python/c-shared/./build.sh` (installs `libwalle.so` under `python/walle/lib/`):

```bash
PYTHONPATH=python python -m unittest discover -s python/tests -v
```

`tests/test_fork_cgo_regression.py` covers the historical crash: parent initializes
`libwalle`, then forked workers call `canonical_schema` / `ms_tool_req_simplify`.
