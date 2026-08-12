# file: tests/test_ci_smoke.py
# version: 1.1.0
# guid: 8d2c1f47-5b3e-4a90-8c16-7e4f2a9d3b58
# last-edited: 2026-08-12

"""Smoke test that keeps the standard-CI Python unit run green.

subtitle-manager is a Go application. The only remaining Python test suite is
``sdks/python``, which runs in its own package context with its own
dependencies (see ``pytest.ini``). This placeholder gives the repo-root
``pytest`` invocation at least one collectable, passing test so the Python CI
job does not fail on an empty collection.

It must not acquire third-party imports: there is no root requirements.txt, and
keeping this stdlib-only is what lets the Python CI job pass without one.
"""


def test_python_toolchain_available() -> None:
    """The CI Python interpreter can import the standard library."""
    import json  # noqa: PLC0415 - intentional inline import for a smoke check

    assert json.loads("[1, 2, 3]") == [1, 2, 3]
