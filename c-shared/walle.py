import json
import ctypes
from enum import Enum
from typing import Any, Dict, Optional
from pathlib import Path


class ValidateLevel(str, Enum):
    LOOSE = "loose"
    LITE = "lite"
    STRICT = "strict"
    ULTRA = "ultra"


class WalleValidator:
    CONFIG_CONSTRAINTS = {
        "validateLevel": {"type": str, "values": set(ValidateLevel)},
        "maxEnumItems": {"type": int, "min": 1},
        "maxEnumStringLength": {"type": int, "min": 1},
        "maxEnumStringCheckThreshold": {"type": int, "min": 1},
        "maxAnyOfItems": {"type": int, "min": 1},
        "maxSchemaDepth": {"type": int, "min": 1},
        "maxSchemaSize": {"type": int, "min": 1},
        "maxTotalPropertiesKeysNum": {"type": int, "min": 1},
    }

    def __init__(self, lib_path=None):
        if lib_path is None:
            current_dir = Path(__file__).parent
            lib_path = current_dir / "libwalle.so"
        self.lib = ctypes.CDLL(lib_path)
        self.lib.ValidateSchema.argtypes = [ctypes.c_char_p, ctypes.c_char_p]
        self.lib.ValidateSchema.restype = ctypes.c_void_p

    def validate_schema(
        self, schema: str, config: Optional[Dict[str, Any]] = None
    ) -> None:
        """
        validate Json schema is valid

        Args:
            schema: Json schema string
            config: Optional configuration parameters, supporting the following fields:
                {
                    "validateLevel": str,  # Validation level: "loose"/"lite"/"strict"/"ultra"
                                           # Default: "strict"
                    "maxEnumItems": int,  # Maximum number of enum items
                                          # Default: 500
                    "maxEnumStringLength": int,  # Maximum length of enum strings
                                                 # Default: 7500
                    "maxEnumStringCheckThreshold": int,  # Threshold for enum string checks
                                                         # Default: 250
                    "maxAnyOfItems": int,  # Maximum number of anyOf items
                                           # Default: 100
                    "maxSchemaDepth": int,  # Maximum schema depth
                                            # Default: 10
                    "maxSchemaSize": int,  # Maximum schema size
                                           # Default: 15000
                    "maxTotalPropertiesKeysNum": int,  # Maximum total number of property keys
                                                       # Default: 1000
                }

        Raises:
            ValueError: When schema validation fails
        """
        schema_bytes = schema.encode("utf-8")

        if config is not None:
            self._validate_config(config)
            config_bytes = json.dumps(config).encode("utf-8")
        else:
            config_bytes = b""

        result = self.lib.ValidateSchema(schema_bytes, config_bytes)
        
        error_msg = ctypes.string_at(result).decode("utf-8")
        self.lib.FreeErrString(result)
        if error_msg:
            raise ValueError(error_msg)

    def _validate_config(self, config: Dict[str, Any]) -> None:
        """
        Validate configuration parameters before sending to C interface

        Args:
            config: Configuration dictionary to validate

        Raises:
            ValueError: If any configuration parameter is invalid
        """
        if not isinstance(config, dict):
            raise ValueError("config must be a dictionary")

        for key, value in config.items():
            if key not in self.CONFIG_CONSTRAINTS:
                raise ValueError(f"Unknown configuration parameter: {key}")

            constraints = self.CONFIG_CONSTRAINTS[key]

            # Type check
            if not isinstance(value, constraints["type"]):
                raise ValueError(
                    f"Invalid type for {key}: expected {constraints['type'].__name__}, got {type(value).__name__}"
                )

            # Value check for validateLevel
            if key == "validateLevel" and value not in constraints["values"]:
                raise ValueError(
                    f"Invalid value for {key}: must be one of {list(constraints['values'])}"
                )

            # Range check for numeric values
            if isinstance(value, int):
                if value < constraints["min"]:
                    raise ValueError(
                        f"Invalid value for {key}: must be greater than or equal to {constraints['min']}"
                    )
