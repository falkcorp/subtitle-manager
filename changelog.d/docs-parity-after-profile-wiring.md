<!-- file: changelog.d/docs-parity-after-profile-wiring.md -->
<!-- version: 1.0.0 -->
<!-- guid: e70a4c26-91b8-4f53-a6d7-2c845b1e30f9 -->
<!-- last-edited: 2026-08-01 -->

### Changed

#### Parity matrix updated for the library-scan profile wiring

The language-profile rows were demoted just before the wiring landed, so the
matrix understated where things stand. Desired-languages-via-profiles and the
profiles row are now 🟡 rather than 🔴/🟡-for-a-different-reason, scoped to
library scans.

Also records a gap the wiring did not close: `processWithAssignedProfile`
passes a bare language code to `ProcessFile`, which scores candidates using the
global `scoring.*` config, so a profile's **cutoff score and per-language
Forced/HI preferences are still ignored**. Only the retired `scoredProfileFetch`
ever applied them.
