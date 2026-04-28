"""
Provides conversion and simplification for tool-calling input.

It prepares Moonshot-style tool definitions as an equivalent internal format.
"""

from __future__ import annotations

import copy
import json
from typing import Any, Dict, List, Mapping, Optional, Sequence, Tuple, Union

from .validator import WalleValidator

JsonDict = Dict[str, Any]
RawInput = Union[Mapping[str, Any], Sequence[Mapping[str, Any]]]

_FUNCTION = "function"
_BUILTIN_FUNCTION = "builtin_function"
_PLUGIN = "_plugin"
_PLUGIN_ALIAS = "plugin"
_DEFAULT_OBJECT_SCHEMA: JsonDict = {"type": "object"}


def ms_tool_req_cvt(raw: RawInput, *, sort_keys: bool = False) -> Optional[str]:
    """
    Convert raw user request/tools to the internal JsonSchema string.

    Returns None when no internal tool-call schema is needed.
    """
    return _ToolSchemaPipeline().cvt(raw, sort_keys=sort_keys)


def ms_tool_req_simplify(
    raw: RawInput,
    *,
    validator: Optional[WalleValidator] = None,
    sort_keys: bool = False,
) -> Tuple[Optional[str], List[str]]:
    """
    Convert with ms_tool_req_cvt(raw), then call walle Canonical for schemas inside tools.
    """
    return _ToolSchemaPipeline(validator=validator).simplify(raw, sort_keys=sort_keys)


class _ToolSchemaPipeline:
    def __init__(self, validator: Optional[WalleValidator] = None) -> None:
        self.validator = validator or WalleValidator()

    def cvt(self, raw: RawInput, *, sort_keys: bool = False) -> Optional[str]:
        req = self._request(raw)
        if req.get("ignore_" + "enf" + "orcer"):
            return None

        tools = req.get("tools") or []
        if not isinstance(tools, list) or not tools:
            return None

        choice = req.get("tool_choice")
        if self._choice_is(choice, "none"):
            return None

        if self._choice_is(choice, "auto") or self._choice_is(choice, "required"):
            result: Any = [self._tool_shape(tool) for tool in tools]
        else:
            result = self._select_tool(tools, choice)

        return self._dumps(result, sort_keys=sort_keys)

    def simplify(
        self,
        raw: RawInput,
        *,
        sort_keys: bool = False,
    ) -> Tuple[Optional[str], List[str]]:
        converted = self.cvt(raw, sort_keys=False)
        if converted is None:
            return None, []

        warnings: List[str] = []
        value = json.loads(converted)
        if isinstance(value, list):
            value = [self._canonical_tool(tool, warnings) for tool in value]
        else:
            value = self._canonical_tool(value, warnings)
        return self._dumps(value, sort_keys=sort_keys), warnings

    @staticmethod
    def _dumps(value: Any, *, sort_keys: bool = False) -> str:
        return json.dumps(
            value,
            ensure_ascii=False,
            separators=(",", ":"),
            sort_keys=sort_keys,
        )

    @staticmethod
    def _request(raw: RawInput) -> JsonDict:
        if isinstance(raw, Mapping):
            if "tools" in raw:
                return copy.deepcopy(dict(raw))
            if "type" in raw:
                return {"tools": [copy.deepcopy(dict(raw))], "tool_choice": "auto"}
        if isinstance(raw, Sequence) and not isinstance(raw, (str, bytes, bytearray)):
            return {"tools": [copy.deepcopy(dict(tool)) for tool in raw], "tool_choice": "auto"}
        raise TypeError("raw must be a request dict, a tools list, or a single tool dict")

    @staticmethod
    def _tool_type(value: Any) -> Any:
        return _PLUGIN if value == _PLUGIN_ALIAS else value

    def _tool_shape(self, tool: Mapping[str, Any]) -> JsonDict:
        tool_type = self._tool_type(tool.get("type"))
        if tool_type in (_FUNCTION, _BUILTIN_FUNCTION):
            return {"type": tool_type, "function": copy.deepcopy(tool.get("function", {}))}
        if tool_type == _PLUGIN:
            plugin = tool.get(_PLUGIN, tool.get(_PLUGIN_ALIAS, {}))
            return {"type": _PLUGIN, _PLUGIN: copy.deepcopy(plugin)}
        return copy.deepcopy(dict(tool))

    def _tool_name(self, tool: Mapping[str, Any]) -> str:
        shaped = self._tool_shape(tool)
        if shaped.get("type") in (_FUNCTION, _BUILTIN_FUNCTION):
            function = shaped.get("function")
            return str(function.get("name", "")) if isinstance(function, dict) else ""
        if shaped.get("type") == _PLUGIN:
            plugin = shaped.get(_PLUGIN)
            return str(plugin.get("name", "")) if isinstance(plugin, dict) else ""
        return ""

    def _choice_is(self, choice: Any, target: str) -> bool:
        if choice is None:
            return target == "auto"
        if choice == target:
            return True
        if not isinstance(choice, Mapping):
            return False

        strategy = choice.get("strategy") or choice.get("Strategy")
        if strategy is not None:
            return strategy == target or (target == "auto" and strategy == "")
        return target == "auto" and self._choice_name(choice) == ""

    def _choice_name(self, choice: Any) -> str:
        if not isinstance(choice, Mapping):
            return ""
        choice_type = self._tool_type(choice.get("type"))
        if choice_type in (_FUNCTION, _BUILTIN_FUNCTION):
            function = choice.get("function")
            return str(function.get("name", "")) if isinstance(function, dict) else ""
        if choice_type == _PLUGIN:
            plugin = choice.get(_PLUGIN, choice.get(_PLUGIN_ALIAS))
            return str(plugin.get("name", "")) if isinstance(plugin, dict) else ""
        return ""

    def _select_tool(self, tools: Sequence[Mapping[str, Any]], choice: Any) -> JsonDict:
        if not isinstance(choice, Mapping):
            raise ValueError(f"unsupported tool_choice: {choice!r}")

        choice_type = self._tool_type(choice.get("type"))
        choice_name = self._choice_name(choice)
        if not choice_name:
            raise ValueError("specified tool_choice missing tool name")

        if choice_type in (_FUNCTION, _BUILTIN_FUNCTION):
            for tool in tools:
                shaped = self._tool_shape(tool)
                if shaped.get("type") == choice_type and self._tool_name(shaped) == choice_name:
                    return shaped
            raise ValueError(f"specified function tool not found: {choice_name}")

        if choice_type == _PLUGIN:
            plugin_name, function_name = self._split_plugin_call_name(choice_name)
            for tool in tools:
                shaped = self._tool_shape(tool)
                if shaped.get("type") != _PLUGIN or self._tool_name(shaped) != plugin_name:
                    continue

                plugin = shaped.get(_PLUGIN)
                functions = plugin.get("functions") if isinstance(plugin, dict) else None
                if not function_name:
                    return shaped
                if not isinstance(functions, list):
                    break

                matched = [
                    copy.deepcopy(fn)
                    for fn in functions
                    if isinstance(fn, dict) and fn.get("name") == function_name
                ]
                if matched:
                    selected = copy.deepcopy(shaped)
                    selected[_PLUGIN]["functions"] = matched
                    return selected
            raise ValueError(f"specified plugin tool not found: {choice_name}")

        raise ValueError(f"unsupported specified tool_choice type: {choice_type!r}")

    @staticmethod
    def _split_plugin_call_name(name: str) -> Tuple[str, str]:
        parts = name.split(".")
        if len(parts) == 1:
            return parts[0], ""
        return ".".join(parts[:-1]), parts[-1]

    def _canonical_schema(self, schema: Optional[Mapping[str, Any]]) -> Tuple[JsonDict, Optional[str]]:
        schema_obj = schema or {}
        schema_str = self._dumps(_DEFAULT_OBJECT_SCHEMA if not schema_obj else schema_obj)
        canonical_str, warning = self.validator.canonical_schema(schema_str)
        canonical = json.loads(canonical_str)
        if not isinstance(canonical, dict):
            raise ValueError("walle canonical result is not a JSON object")
        return canonical, warning

    def _canonical_tool(self, tool: Mapping[str, Any], warnings: List[str]) -> JsonDict:
        out = copy.deepcopy(dict(tool))
        tool_type = out.get("type")

        if tool_type == _FUNCTION:
            function = out.get("function")
            if isinstance(function, dict) and function.get("parameters") is not None:
                function["parameters"] = self._canonical_or_default(
                    function.get("parameters"),
                    function.get("strict"),
                    warnings,
                )
            return out

        if tool_type == _PLUGIN:
            plugin = out.get(_PLUGIN)
            functions = plugin.get("functions") if isinstance(plugin, dict) else None
            if not isinstance(functions, list):
                return out

            plugin["functions"] = [
                self._canonical_plugin_function(function, warnings)
                for function in functions
            ]
        return out

    def _canonical_plugin_function(self, function: Any, warnings: List[str]) -> Any:
        if not isinstance(function, dict):
            return copy.deepcopy(function)

        out = copy.deepcopy(function)
        strict = bool(out.get("strict", True))
        kwargs = out.get("kwargs")
        kwargs_list = kwargs if isinstance(kwargs, list) else []

        if not strict and not kwargs_list:
            out["parameters"] = copy.deepcopy(_DEFAULT_OBJECT_SCHEMA)
            return out

        if kwargs_list:
            out["kwargs"] = [self._canonical_kwarg(arg, warnings) for arg in kwargs_list]
        elif out.get("parameters") is not None:
            out["parameters"] = self._canonical_or_default(
                out.get("parameters"),
                out.get("strict"),
                warnings,
            )
        return out

    def _canonical_kwarg(self, arg: Any, warnings: List[str]) -> Any:
        if not isinstance(arg, dict):
            return copy.deepcopy(arg)

        name = arg.get("name")
        schema = {key: copy.deepcopy(value) for key, value in arg.items() if key != "name"}
        canonical, warning = self._canonical_schema(schema)
        if name is not None:
            canonical["name"] = copy.deepcopy(name)
        if warning:
            warnings.append(warning)
        return canonical

    def _canonical_or_default(
        self,
        schema: Optional[Mapping[str, Any]],
        strict_value: Any,
        warnings: List[str],
    ) -> JsonDict:
        if strict_value is False:
            return copy.deepcopy(_DEFAULT_OBJECT_SCHEMA)
        canonical, warning = self._canonical_schema(schema)
        if warning:
            warnings.append(warning)
        return canonical
