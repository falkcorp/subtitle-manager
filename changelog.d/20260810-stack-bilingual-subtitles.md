### Added

#### `merge --stack` combines two existing subtitles into one bilingual file

Two subtitle tracks the user already has on disk — an English sidecar and a
Spanish one, say — can now be combined into a single "double subs" file where
each cue carries both languages and renders as two lines.

Neither existing path could do this:

- `merge` interleaves. `MergeTracks` concatenates both cue lists and sorts by
  start time, so three English cues plus three Spanish cues became six cues with
  duplicated timestamps. Two cues at identical timing render as competing,
  overlapping subtitles rather than as a bilingual pair.
- `dualsub` produces the correct stacked shape, but only from a machine
  translation of a single input. It cannot take a second subtitle file, and it
  requires translation-service credentials.

The new `subtitles.StackTracks` pairs cues by greatest time overlap rather than
by exact timestamp equality, because independently sourced tracks are rarely
frame-identical. Each secondary cue is consumed at most once, so one long cue
cannot be pasted onto several primary cues. A cue with no counterpart is kept as
its own cue rather than dropped — losing dialogue silently would be worse than
the interleaving this replaces.

The existing interleaving behaviour of `merge` is unchanged; stacking is opt-in
via `--stack`.

Verified end to end against a running server: two real sidecars for the same
episode combined into one file of three bilingual cues, and the library browse
endpoint then reported the result as an additional subtitle track for that
episode.
