"""
Regression tests for fork + libwalle CGO (historically segfault in cgofree / FreeErrString).

Scenario: parent loads libwalle via WalleValidator, then forked children call
canonical_schema / ms_tool_req_simplify. Without clearing CDLL state in the child,
this can crash.

Requires a built ``python/walle/lib/libwalle.so`` (see README / python/c-shared/build.sh).
"""

from __future__ import annotations

import os
import sys
import unittest
from typing import Tuple


def pool_demo_worker(_: int) -> Tuple[int, int]:
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


def _lib_available() -> bool:
    try:
        from importlib.resources import files

        p = files("walle").joinpath("lib", "libwalle.so")
        return p.is_file()
    except Exception:
        return False


def _prime_parent_cgo() -> None:
    from walle import WalleValidator

    WalleValidator().canonical_schema(
        '{"type":"object","properties":{"x":{"type":"string"}}}'
    )


def _child_canonical_schema() -> None:
    from walle import WalleValidator

    WalleValidator().canonical_schema('{"type":"object"}')


@unittest.skipUnless(sys.platform.startswith("linux"), "Linux fork + CGO regression")
@unittest.skipUnless(_lib_available(), "python/walle/lib/libwalle.so missing (run python/c-shared/build.sh)")
class TestForkAfterLibwalleInit(unittest.TestCase):
    def test_os_fork_child_calls_canonical_after_parent_primed(self) -> None:
        """Minimal repro: parent initializes CGO, child runs canonical_schema."""
        _prime_parent_cgo()
        pid = os.fork()
        if pid == 0:
            status = 0
            try:
                _child_canonical_schema()
            except Exception:
                status = 1
            os._exit(status)

        _pid, sts = os.waitpid(pid, 0)
        self.assertTrue(os.WIFEXITED(sts), f"unexpected wait status: {sts}")
        self.assertEqual(os.WEXITSTATUS(sts), 0)

    def test_multiprocessing_fork_pool_after_parent_primed(self) -> None:
        """Typical crash site: ProcessPool / Pool(fork) after parent used walle."""
        import multiprocessing as mp

        _prime_parent_cgo()
        ctx = mp.get_context("fork")
        with ctx.Pool(processes=2) as pool:
            results = pool.map(pool_demo_worker, range(4))

        self.assertEqual(len(results), 4)
        self.assertTrue(all(r[0] > 0 for r in results))


if __name__ == "__main__":
    unittest.main()
