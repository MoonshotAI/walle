"""
Regression tests for fork + libwalle CGO (historically segfault in cgofree / FreeErrString).

Scenario: parent loads libwalle via WalleValidator, then forked children call
canonical_schema / ms_tool_req_simplify. Without clearing CDLL state in the child,
this can crash.

Requires a built ``python/walle/lib/libwalle.so`` (see README / python/c-shared/build.sh).
"""

from __future__ import annotations

import os
import subprocess
import sys
import textwrap
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


# Child program for ``TestFreeErrStringUnderMmapPressure``. Fresh ``python -c``
# so ``MALLOC_MMAP_THRESHOLD_`` is honored from the child's first libc
# allocation (glibc reads it once, before unittest's own heap traffic).
_FREEERRSTRING_CHILD_CODE = textwrap.dedent(
    """
    from walle import WalleValidator

    validator = WalleValidator()
    validator.canonical_schema(
        '{"type":"object","properties":{"x":{"type":"string"}}}'
    )
    print("OK", flush=True)
    """
)


@unittest.skipUnless(sys.platform.startswith("linux"), "glibc malloc tunable, Linux only")
@unittest.skipUnless(_lib_available(), "python/walle/lib/libwalle.so missing (run python/c-shared/build.sh)")
class TestFreeErrStringUnderMmapPressure(unittest.TestCase):
    """``FreeErrString`` must keep ``argtypes=[c_void_p]`` under high-VA cgo allocs.

    When ``C.CString`` returns an address ``>= 2**31``, omitting those ctypes
    lines (historic v0.1.6) caused pointer truncation → SIGSEGV in ``free``.
    Normal desktops often allocate below ``2**31``, so CI forces mmap-heavy
    glibc behaviour via ``MALLOC_MMAP_THRESHOLD_=1``. A/B against the broken
    binding lives in ``python/c-shared/repro_freeerrstring_truncation.py``
    (--mode v016 vs head); this unittest only asserts the packaged binding survives.
    """

    def test_binding_survives_under_mmap_pressure(self) -> None:
        env = os.environ.copy()
        env["PYTHONPATH"] = os.pathsep.join(
            [p for p in sys.path if p] + [env.get("PYTHONPATH", "")]
        )
        env["MALLOC_MMAP_THRESHOLD_"] = "1"
        env["MALLOC_MMAP_MAX_"] = str((1 << 31) - 1)

        result = subprocess.run(
            [sys.executable, "-c", _FREEERRSTRING_CHILD_CODE],
            capture_output=True,
            text=True,
            env=env,
            timeout=60.0,
        )
        self.assertEqual(
            result.returncode,
            0,
            "expected success under mmap pressure; if this fails after editing "
            f"validator._open_walle_cdll FreeErrString lines, revisit ctypes. "
            f"stdout={result.stdout!r} stderr={result.stderr!r}",
        )
        self.assertIn("OK", result.stdout)


if __name__ == "__main__":
    unittest.main()