<!-- file: changelog.d/docs-language-profile-findings.md -->
<!-- version: 1.0.0 -->
<!-- guid: 8b3d5e17-2c9a-4f60-b148-6e0a7d95c231 -->
<!-- last-edited: 2026-08-01 -->

### Changed

#### Parity matrix: language profiles corrected from built to not-wired

`BAZARR_PARITY_STATUS.md` claimed desired-languages-via-profiles and
Forced/HI-at-download were complete. `FetchWithProfile` does implement the
priority iteration, but no real download path calls it, and the profile
assignment the web UI writes is not the one the fetch path reads. Those rows
are now 🔴/🟡, with a findings section recording the write-only assignment
problem and the undeletable-last-profile bug.
