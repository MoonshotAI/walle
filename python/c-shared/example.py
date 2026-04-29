import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from walle import WalleValidator


def use_default_config():
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


def main():
    use_default_config()
    use_custom_config()

if __name__ == "__main__":
    main()
