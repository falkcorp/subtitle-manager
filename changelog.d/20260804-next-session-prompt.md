### Changed

#### Next-session prompt brought up to date

`docs/NEXT_SESSION_PROMPT.md` now reflects the 2026-08-04 session: the language
profile work is complete, so the queue leads with credential-gated providers,
the two CodeQL alerts, and the remaining scoring hole (a profile's cutoff score
and Forced/HI flags still never reach the scorer).

It also records the decisions made rather than only the outcomes — why an
assigned profile beats `MonitoredItem.Languages`, why requests that name a
language directly are exempt, and why deleting the only profile is allowed —
along with several process traps that cost real time: a PR branched off an
unmerged branch goes `CONFLICTING` after that branch rebase-merges and its CI
then never queues at all, `workflow_dispatch` skips Go CI because there is no
diff to detect, actions here are SHA-pinned so a tag ref fails at job set-up,
and plain `go test` misses races in tests that mutate package globals.
