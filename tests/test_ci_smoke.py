# file: tests/test_ci_smoke.py
# version: 1.0.0
# guid: 8d2c1f47-5b3e-4a90-8c16-7e4f2a9d3b58

"""Smoke test that keeps the standard-CI Python unit run green.

subtitle-manager is a Go application; its real Python test suites
(``sdks/python`` and ``tests/e2e``) run in their own package contexts with
their own dependencies (see ``pytest.ini``). This placeholder gives the
repo-root ``pytest`` invocation at least one collectable, passing test so the
Python CI job does not fail on an empty collection.
"""


def test_python_toolchain_available() -> None:
    """The CI Python interpreter can import the standard library."""
    import json  # noqa: PLC0415 - intentional inline import for a smoke check

    assert json.loads("[1, 2, 3]") == [1, 2, 3]
