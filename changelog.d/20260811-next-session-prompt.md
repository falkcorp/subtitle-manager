### Changed

#### Next-session prompt rewritten around a working product and one open decision

`docs/NEXT_SESSION_PROMPT.md` no longer opens with unfinished work on disk —
everything is merged and there are no open PRs. It now leads with what has been
verified in a real browser (library detection, navigation, mass edit, and
combining two subtitles into one bilingual file) and with the reproduction steps
that are easy to get wrong: the web server needs a `-tags sqlite` build, and the
frontend must be built first or the binary silently serves a stale shell.

The top item is now a decision rather than a task: published release binaries
cannot start the web server at all, and the three ways out differ enormously in
cost.

It also records the CI-reading rules that cost real time this session — stale
0-second failures appear twice while a genuine 3-second CodeQL failure appears
once; a skipped check is not a passing check; and a wait-loop filtered by job
name reports success before the job exists.
