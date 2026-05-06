import multiprocessing as mp
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

def use_default_config():
    from walle import WalleValidator

    # valid case
    validator = WalleValidator()
    valid_schema = """
    {
        "type": "object",
        "properties": {
            "name": {"type": "string"},
            "age": {"type": "integer"}
        }
    }
    """
    print("use default config")
    validator.validate_schema(valid_schema)
    print("case1: validate success!")
    canon, warn = validator.canonical_schema(valid_schema)
    print("canonical_schema:", canon)
    print("canonical warning:", warn)

    # invalid case
    try:
        validator = WalleValidator()
        invalid_schema = '{"type": "invalid"}'
        validator.validate_schema(invalid_schema)
    except ValueError as e:
        print(f"case2: validate failed: {e}")


def use_custom_config():
    from walle import WalleValidator

    try:
        validator = WalleValidator()
        schema = """
        {
            "type": "object",
            "properties": {
                "name": {
                    "type": "string",
                    "enum": ["Alice", "Bob", "Charlie", "David", "Eve"]
                }
            }
        }
        """
        custom_config = {
            "validateLevel": "strict",
        }
        print("\nuse custom config")
        validator.validate_schema(schema, custom_config)
    except ValueError as e:
        print(f"case3: validate failed: {e}")


def worker_func(_: int):
    # Import here, not in ``use_multiprocessing_spawn`` or ``use_multiprocessing_fork``: 
    # only worker processes run this body; the parent never calls ``ms_tool_req_simplify``. 
    # Avoid loading ``libwalle`` in the parent before ``Pool(spawn)`` is created in this script.
    # 尽可能晚加载 ``ms_tool_req_simplify / libwalle``
    from walle import ms_tool_req_simplify

    req = {
        "tools": [
            {
                "type": "function",
                "function": {
                    "name": "demo_mp_worker",
                    "parameters": {
                        "type": "object",
                        "properties": {"a": {"type": "string"}},
                    },
                },
            }
        ],
        "tool_choice": "auto",
    }
    out, warns = ms_tool_req_simplify(req)
    assert out is not None
    return len(out), len(warns)

def use_multiprocessing_fork():
    """
    Safe pattern for **multiprocessing + fork** (Linux default) with libwalle.

    Problem being avoided: parent loads ``libwalle`` / CGO, then ``fork()``; the
    child used to inherit broken Go runtime state and crash in ``FreeErrString``.

    """
    from walle import WalleValidator

    print("\n--- Test: multiprocessing + fork ---")
    if not sys.platform.startswith("linux"):
        print("(skipped: Linux only)")
        return

    WalleValidator().canonical_schema('{"type":"object"}')

    ctx = mp.get_context("fork")
    with ctx.Pool(processes=20) as pool:
        results = pool.map(worker_func, range(40))
    lengths = [r[0] for r in results]
    print("worker output string lengths:", lengths)


def use_multiprocessing_spawn():
    """
    **Recommended** when you use process-based parallelism: ``spawn`` start method.

    Each worker is a fresh interpreter with its own ``libwalle.so``.
    """
    print("\n--- Test: multiprocessing + spawn ---")
    print("multiprocessing: **spawn** pool (recommended)")

    ctx = mp.get_context("spawn")
    with ctx.Pool(processes=2) as pool:
        results = pool.map(worker_func, range(4))
    lengths = [r[0] for r in results]
    print("worker output string lengths:", lengths)


def main():
    # ``Pool(spawn)`` in this process after ``libwalle`` is loaded (e.g. after
    # ``use_default_config``) can SIGSEGV the parent, so the spawn demo runs first.
    use_multiprocessing_spawn()
    use_default_config()
    use_custom_config()
    use_multiprocessing_fork()


if __name__ == "__main__":
    main()
