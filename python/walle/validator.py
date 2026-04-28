import ctypes
import json
from enum import Enum
from importlib import resources
from pathlib import Path
from typing import Any, Dict, Optional, Tuple


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

    def __init__(self, lib_path: Optional[str] = None):
        if lib_path is None:
            lib_path = self._default_lib_path()
        self.lib = ctypes.CDLL(str(lib_path))
        self.lib.ValidateSchema.argtypes = [ctypes.c_char_p, ctypes.c_char_p]
        self.lib.ValidateSchema.restype = ctypes.c_void_p
        self.lib.CanonicalSchema.argtypes = [ctypes.c_char_p]
        self.lib.CanonicalSchema.restype = ctypes.c_void_p

    @staticmethod
    def _default_lib_path() -> Path:
        lib_path = resources.files("walle").joinpath("lib", "libwalle.so")
        if not lib_path.is_file():
            raise FileNotFoundError(
                "libwalle.so is missing from the walle package. "
                "Run python/c-shared/build.sh before building the wheel."
            )
        return Path(str(lib_path))

    def validate_schema(
        self, schema: str, config: Optional[Dict[str, Any]] = None
    ) -> None:
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

    def canonical_schema(self, schema: str) -> Tuple[str, Optional[str]]:
        raw = self.lib.CanonicalSchema(schema.encode("utf-8"))
        try:
            payload = json.loads(ctypes.string_at(raw).decode("utf-8"))
        finally:
            self.lib.FreeErrString(raw)

        err = payload.get("error")
        if err:
            raise ValueError(err)
        canonical = payload.get("canonical")
        if canonical is None:
            raise ValueError("canonical response missing 'canonical' field")
        warn = payload.get("warning")
        return canonical, (warn if warn else None)

    def _validate_config(self, config: Dict[str, Any]) -> None:
        if not isinstance(config, dict):
            raise ValueError("config must be a dictionary")

        for key, value in config.items():
            if key not in self.CONFIG_CONSTRAINTS:
                raise ValueError(f"Unknown configuration parameter: {key}")

            constraints = self.CONFIG_CONSTRAINTS[key]
            if not isinstance(value, constraints["type"]):
                raise ValueError(
                    f"Invalid type for {key}: expected "
                    f"{constraints['type'].__name__}, got {type(value).__name__}"
                )

            if key == "validateLevel" and value not in constraints["values"]:
                raise ValueError(
                    f"Invalid value for {key}: must be one of {list(constraints['values'])}"
                )

            if isinstance(value, int) and value < constraints["min"]:
                raise ValueError(
                    f"Invalid value for {key}: must be greater than or equal to "
                    f"{constraints['min']}"
                )
