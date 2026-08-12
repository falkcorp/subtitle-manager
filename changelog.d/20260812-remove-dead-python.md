<!-- file: changelog.d/20260812-remove-dead-python.md -->
<!-- version: 1.0.0 -->
<!-- guid: 6a09d5f3-72be-4c81-b3e7-0f4d29c85a16 -->
<!-- last-edited: 2026-08-12 -->

### Removed

#### The root `requirements.txt`, which was Bazarr's dependency list

`Python CI (3.13)` failed on every PR that touched Python dependencies and
reported **skipped** on `main`, so the Python build was broken while `main`
looked green.

The cause was the manifest itself. `requirements.txt` pinned ~110 packages —
Flask, Flask-SocketIO, Flask-SQLAlchemy, SQLAlchemy, alembic, subliminal,
guessit, enzyme, babelfish, pysubs2, openai, mypy, pylint — inherited from
Bazarr. **Nothing in this repository imports any of them.** subtitle-manager is
a Go service with a React UI; the only Python that CI actually runs is
`scripts/assemble_todo.py` and `.github/workflows/scripts/sync_receiver.py`,
and both are **stdlib-only**.

Meanwhile the one dependency something did need — `selenium`, for the E2E suite
below — was never declared, which is what produced the install failures.

Deleting the file also removes the `requirements*.txt` path trigger, so the
Python job stops firing on dependency churn it has no dependencies for.

Unaffected: `sdks/python/` ships its own `requirements.txt` and `setup.py`
resolves it relative to its own directory, so the published
`subtitle_manager_sdk` is untouched. The `scripts/` tooling is untouched.

#### `tests/e2e/` — a Selenium suite that no workflow ran

Seven test modules plus a runner, a conftest and their own pytest.ini, wired to
nothing. `pytest.ini` at the repo root explicitly excluded them
(`--ignore=tests/e2e`), no workflow invoked them, and their `selenium`
dependency was never installed anywhere — so they had never run.

Browser verification in this project is done with Playwright, which is already a
working dev dependency and was used to verify the Media Library, the bilingual
Combine flow, and Settings → Providers against a real server. Two unmaintained
browser-test stacks is one too many; this removes the one that never worked.

### Verification

- Root `pytest` passes in a clean venv with **only `pytest` installed** — the
  exact condition CI now runs under, with no dependency manifest present.
- `ruff check .github/scripts` and `ruff format --check .github/scripts`, the
  only Python lint CI gates on, both pass.
- `reusable-ci.yml` already detects a missing dependency manifest and skips the
  pip cache rather than failing, so removing the file needs no workflow change.
- Go build and full suite unaffected.
