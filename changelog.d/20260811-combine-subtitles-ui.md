### Added

#### Combine two subtitles into a bilingual track from the Media Library

Bilingual "double subs" no longer require the command line. In the Media
Library, each media file now lists the subtitles the server found for it. Tick
two and press **Combine**: the result is one file where every cue carries both
languages and renders as two lines.

The browse endpoint has always reported each file's sidecars — the interface
simply never displayed them.

Selection order is meaningful and is preserved: the first subtitle ticked
becomes the primary and renders on top of each stacked cue, which the UI says
out loud once two are selected. Combine stays disabled until exactly two are
ticked, since stacking combines a pair.

A new `POST /api/subtitles/stack` takes the two subtitle paths and writes the
bilingual file. It is deliberately separate from `/api/dualsub`, which uploads a
single file and machine-translates it: this endpoint works on files already in
the library and does not translate, so no translation service or credentials are
involved. Paths are validated before anything is opened, like every other
path-taking endpoint. A failure is reported to the user rather than being
swallowed.

### Fixed

#### Generated bilingual subtitles were reported as English

`extractLanguageFromFilename` falls back to English for any language code it
does not recognise, and `eo` — the sentinel suffix `dualsub` uses so
Plex/Jellyfin/Emby see the double-subs file as a distinct track — was missing
from the table. A combined file therefore appeared in the Media Library as a
second **English** subtitle, indistinguishable from the real one, and anything
keyed on language saw two English tracks for one episode.

`eo` now reports as "Bilingual (double subs)", which describes what the product
actually generated. The trade is that a genuine Esperanto subtitle would be
labelled the same way; that is much the rarer case, and `eo` is this product's
documented sentinel.
