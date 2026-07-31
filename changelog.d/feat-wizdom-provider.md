### Added

#### Wizdom is now a real subtitle provider (Hebrew)

`wizdom` was one of the fictional stubs that GET `api.wizdom.com/subtitles/…`,
a hostname nobody controls. It is now a working keyless provider against
wizdom.xyz's public JSON API — no account and no API key — covering both movies
and TV.

Three things about this provider are worth knowing:

- **It indexes strictly by IMDb ID.** The `Provider` interface only supplies a
  file path, so the provider resolves the parsed title to an IMDb ID itself
  against IMDb's keyless suggestion service, filtered on title kind and year and
  memoised per client so scanning a series does not re-resolve it once per
  episode. Resolution fails closed: no confident match is an error, never a
  guess, because a fuzzy match does not fail — it downloads the wrong show's
  subtitles and reports success.
- **It is Hebrew-only, and its API takes no language parameter.** A request for
  any other language is refused outright rather than served Hebrew bytes, which
  the scanner would have written to `movie.en.srt` and counted as a success —
  ending the search before any provider that actually had English was consulted.
- **Its files are Windows-1255, not UTF-8**, and are transcoded on download.
  The encoding is named rather than sniffed because sniffing gets it wrong:
  `charset.DetermineEncoding` reports ISO-8859-1 for these files, which decodes
  the same bytes into Latin-1 mojibake.

Candidates are ranked by how many release tokens they share with the media file
name — the API returns a `score` field, but it is 0 on every entry of every
response observed, so ranking is entirely ours.

Verified end-to-end against the live service: real subtitles for *The Shawshank
Redemption*, *Game of Thrones* S01E01 and *Inception*, correctly transcoded, and
an English request correctly refused.
