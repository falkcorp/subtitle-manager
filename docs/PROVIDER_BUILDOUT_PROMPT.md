<!-- file: docs/PROVIDER_BUILDOUT_PROMPT.md -->
<!-- version: 1.3.0 -->
<!-- guid: 0d3f8b21-5c47-4e90-a2b6-9f1c0e7d4a38 -->
<!-- last-edited: 2026-08-04 -->

# Next-session prompt: provider build-out

Paste the block below into a fresh Claude Code session (in the
`falkcorp/subtitle-manager` repo). Check the boxes for providers you have
accounts/keys for, and put the **actual secrets in your local
`~/.subtitle-manager.yaml`** (never in the repo or chat) before you run it.

---

Continue the subtitle-manager **Bazarr backend feature-parity** work. Everything
tractable without external accounts is already merged (see
`docs/BAZARR_PARITY_STATUS.md` and the `project_subtitle_manager_bazarr_parity`
auto-memory). The one big remaining gap is the **provider ecosystem**: ~50 of
~55 registered providers are non-functional stubs (they GET a fictional
`api.<name>.com`). Real ones today: `opensubtitles`, `napiprojekt`, `gestdown`,
`podnapisi`, `embedded`.

> **Read this before trusting any "already implemented" note below.** This
> document is a prompt, not an audit — at least one of its original claims was
> wrong (it said opensubtitles.com was done; the `opensubtitlescom` package is
> still a fictional stub). **Verify each line against the code before acting on
> it.** The same rule applies to provider protocols: confirm against the
> provider's live docs *and* subliminal's source and VCR cassettes before
> writing any client. The stubs' plausible-looking URLs are fiction.

**Work the phases below, one focused PR per provider, using the established
workflow** (worktree off `origin/main`; `changelog.d/` fragment; mandatory file
version headers; httptest-mocked tests — you never need my real secrets, only
the config-key wiring; `gh pr merge <N> --rebase --admin`). For each provider,
**confirm the real API/protocol from its current docs before implementing — do
not guess** — and skip pure-HTML-scraping providers unless I say otherwise
(note any you skip and why).

## 2026-08-04: OpenSubtitles works; ~22 registered providers are stub shells

### OpenSubtitles is done (#2248, #2250, #2252)

`opensubtitlescom` was a stub pointing at `api.opensubtitlescom.com` — not a
real host — hitting `GET /subtitles/{name}/{lang}`, an endpoint in no version of
the API. It is now a thin wrapper over `pkg/providers/opensubtitles`, which is a
genuine REST v1 client.

That client had three defects on the download path, all fixed and all confirmed
against the published contract rather than inferred:

- `GET {api}/download?file_id=` instead of `POST /download {"file_id":N}`
  followed by a GET of the returned `link`
- the id read from `attributes.subtitle_id`; it lives at
  `attributes.files[].file_id`, and they are different values
- the required `Api-Key` header never sent, though `opensubtitles.api_key` was
  read by the CLI

Credentials are now read from `opensubtitles.*` **and** the
`providers.opensubtitles.*` / `providers.opensubtitlescom.*` spellings the
Bazarr importer writes — an imported config previously left the provider
unauthenticated with the username and password sitting in the file unread.

### The stub-shell problem, with data

**22 of 51 hostnames referenced by provider packages do not resolve at all**,
measured 2026-08-04. They follow the same fabricated `api.<name>.com` pattern as
the `opensubtitlescom` stub:

```
api.addic7ed.com      api.animekalesi.com   api.avistaz.com       api.bsplayer.com
api.hosszupuska.com   api.karagarga.com     api.legendasdivx.com  api.legendasnet.com
api.subdivx.com       api.subs4series.com   api.subssabbz.com     api.subsynchro.com
api.subtitrarinoi.com api.subtitriidlv.com  api.subtitulamos.com  api.titrariro.com
api.tusubtitulo.com   api.yavka.com         api.yifysubtitles.com api.zimuku.com
```

This is not cosmetic. With no provider instances configured, `FetchFromAll`
falls back to **every registered provider**, so each shell takes a slot in a
fetch wave and costs a DNS failure before the real providers are reached. It is
also why a failed search used to be so uninformative — the reported error was
whichever shell happened to fail last (`no subtitle found: ... lookup
api.zimuku.com: no such host`).

**Deliberately not acted on.** Deciding which of these to implement, retire, or
repoint is a product call, and a wrong guess silently removes a provider someone
relies on. Two candidates are worth checking individually rather than in bulk:
`www.podnapisi.net` also failed to resolve in this sweep despite podnapisi being
implemented and reported working, and `api.opensubtitles.org` resolves but is a
legacy endpoint the v1 client cannot speak.

Suggested next step: audit the registry rather than the hostnames — for each
registered name, does the package implement a real protocol, or is it a
`GET {host}/subtitles/{name}/{lang}` shell? The latter can be unregistered as a
group with no behaviour loss, since they have never returned a subtitle.

## Phase 1 — keyless providers
- [x] **Gestdown** (`gestdown.info`) — done, PR #2201. Keyless REST proxy for
      Addic7ed, **TV only**. Indexes by show/season/episode rather than by hash,
      so it fits the filename-only `Fetch` interface via `metadata.ParseFileName`.
      Verified live: it treats `en`, `eng` and `English` as equivalent, but
      regional variants are distinct (`pt` ≠ `pt-BR`).
- [x] **Podnapisi** — done, PR #2202. Covers **movies and TV**. The parity doc's
      old "fragile HTML scraping" label was wrong: it has a real advanced-search
      JSON API. Language is **alpha-2**, and downloads are **zip-wrapped** — both
      facts came from subliminal's VCR cassette, not from guessing.
- [x] **Wizdom** (`wizdom.xyz`) — done. **Hebrew only**, movies and TV. Keyless
      JSON API: `/api/search?action=by_id&imdb=…` plus `season`/`episode`, and
      `/api/files/sub/{id}` returning a zip. Two facts worth carrying forward:
      it indexes strictly by **IMDb ID**, so the provider resolves title→ID
      itself against IMDb's keyless suggestion endpoint; and its files are
      **Windows-1255**, not UTF-8.

      This entry is also the correction to what this section used to claim.
      "Phase 1 COMPLETE / EXHAUSTED — no keyless candidates remain" was wrong:
      wizdom had never been assessed at all. Treat "exhausted" as meaning
      "nothing left that I looked at", and re-probe before believing it.

### Assessed and skipped, with the evidence (probed 2026-07-31)

| Candidate | Result | Verdict |
|---|---|---|
| `yifysubtitles.ch` | 200, `text/html`, no API | scraping — skip (old label was right) |
| `subscene.com` | 403 Cloudflare interstitial | blocked — skip |
| `subdl.com` | 403 `{"error":"not_authorized"}` | Phase 2, needs key |
| `titlovi.com` | 403 on `/gettoken` | Phase 2, needs credentials |
| `animetosho.org` | keyless JSON, but keys on **AniDB episode IDs** | needs an AniDB mapping layer first |
| `subsource.net` | `/v1` is live JSON but every search param returns `{"error":"Movie ID is required"}` | undocumented; revisit if the API is published |
| `subtitulamos.tv` | `/search/query?q=` returns `[]` for every query tried | undocumented; revisit |
| `feliratok.eu` | HTML only | scraping — skip |

## Phase 2 — credential-gated providers (I've put secrets in my local config)
Implement + wire config for the ones I check. Config lives under the provider's
namespace in `~/.subtitle-manager.yaml`; you only wire the key names + protocol.
- [ ] **opensubtitles.com** — ⚠️ **NOT implemented**, despite this line's original
      claim. The `opensubtitlescom` package is still a fictional stub hitting
      `api.opensubtitlescom.com`. The real, working integration is the *classic*
      `opensubtitles` package — a different provider. Treat this as a
      from-scratch REST v1 implementation
      (`opensubtitles.api_key` / `.username` / `.password`).
- [ ] **subdl.com** — free API key (`subdl.api_key`).
- [ ] **addic7ed** — username/password (+ cookies/anti-captcha caveats; note if blocked).
- [ ] **titlovi** — username/password.
- [ ] **betaseries** — API key.
- [ ] **assrt** — API token.
- [ ] _(add any others you want and have creds for)_

Providers requiring private-tracker accounts (`avistaz`, `hdbits`, `karagarga`)
or that are scraping-only stay stubs unless I explicitly ask — keep the
provider-boundary section of `docs/BAZARR_PARITY_STATUS.md` accurate as you go.

## Phase 3 — optional infra gaps (from the changelog cross-reference)
- [x] Notify Sonarr/Radarr to rescan after a download — done, PR #2205. The
      "media→*arr-item-id mapping" was **not** the blocker it looked like: the id
      is resolved from the media path at event time against the *arr's own folder
      listing, so no schema change was needed. See the design note in
      `docs/BAZARR_PARITY_STATUS.md`.
- [ ] Split Whisper connection vs read timeouts; provider-throttling depth.
- [ ] Real-time SignalR sync with Sonarr/Radarr — still open, still large.

**Security:** put real secrets only in the local (uncommitted) config file. Just
tell me which providers you configured — I don't need the secret values.

---
