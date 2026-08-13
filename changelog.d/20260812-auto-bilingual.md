<!-- file: changelog.d/20260812-auto-bilingual.md -->
<!-- version: 1.0.0 -->
<!-- guid: e2704b91-8c3d-45fa-b016-79d5c8213ea4 -->
<!-- last-edited: 2026-08-12 -->

### Added

#### Bilingual subtitles are produced automatically during scan and download

Combining two languages into one "double subs" file previously required doing
it by hand: tick two subtitles in the Media Library and press **Combine**. Now a
language profile that asks for two or more languages produces the bilingual file
as part of the ordinary download, stacking the two highest-priority languages
the profile actually obtained.

**It is purely additive.** The per-language sidecars are written exactly as
before and left untouched, so a player can still select English or Spanish on
its own. Nothing is replaced, and no existing file is ever overwritten.

**Two output files, because they serve different readers:**

| File | Purpose |
| --- | --- |
| `Episode.en-es.srt` | Self-describing and collision-free. Cannot clash with `Episode.en.srt` or `Episode.es.srt`, and the order states which language is on top. |
| `Episode.eo.srt` | What media servers actually surface. Plex/Jellyfin/Emby map a filename language tag to a track label, and `en-es` is not a language they recognise — without this the bilingual file would never appear in the subtitle menu. |

The second is a **reflink** of the first where the filesystem supports it
(APFS `clonefile`, or the `FICLONE` ioctl on btrfs, XFS and OpenZFS ≥2.2), so it
costs no additional space; elsewhere it degrades to a plain copy. Deliberately
not a hardlink — a hardlink shares an inode, so a tool rewriting one file in
place would silently rewrite the other.

The Esperanto sentinel follows the convention already used by `dualsub` and
`POST /api/subtitles/stack`, and reads the same `dualsub.sentinel_language`
setting.

**Partial results behave sensibly.** If only one language is found you get that
normal sidecar and no bilingual file, exactly as today; the pair appears once
the second language arrives. Both sidecars must exist on disk before anything is
combined, so post-processing that renames or converts a file cannot produce
something that looks bilingual but carries a single track.

Skipped under single-language naming, where every language is written to the
same `<video>.srt` and no second sidecar exists to stack.

### Fixed

#### Language codes may now contain a hyphen

`ValidateLanguageCode` was alphanumeric-only, so it rejected every BCP-47
regional tag — `pt-BR`, `zh-Hans`, `sr-Latn` — even though those are ordinary
subtitle languages. It also blocked the `en-es` form the bilingual filename
needs.

An interior hyphen is now accepted. It is safe in this position: the value is
interpolated into a filename component, a hyphen is not a path separator and
cannot form `..`, and the finished path is still validated. Edge positions stay
rejected, so a code can never begin a filename component with `-` or leave a
dangling separator. `ValidateProviderName` has always permitted a hyphen, so
this also removes an inconsistency between the two validators.

The existing `{"with dash", "en-us", true}` case grouped a hyphen with genuinely
dangerous input (slashes, null bytes, traversal); it is replaced by an
acceptance case plus explicit leading/trailing/alone rejections, so the boundary
is still pinned rather than merely loosened.

### Verification

- `TestWriteBilingualPairProducesBothFiles` asserts both files exist, are
  identical, contain both languages, and that the primary language appears
  *above* the secondary in the stacked cue.
- `TestWriteBilingualPairNeverOverwrites` puts a real Esperanto subtitle at the
  sentinel name: it survives, and the combined file is still written.
- `TestWriteBilingualPairRequiresBothSidecars` removes one source and confirms
  neither output is produced.
- `TestReflinkBackendIsWired` calls the platform backend directly, so a backend
  that failed on every call cannot hide behind the copy fallback. On APFS it
  reports a real reflink; where unsupported it asserts the error is the expected
  "unsupported" signal and that no stray file is left behind.
- `TestCloneFileProducesAnIndependentCopy` writes to the clone and checks the
  source is unchanged — which is what rules out a hardlink.
- All five release targets cross-compile, including the Windows build that has
  no reflink backend at all.
