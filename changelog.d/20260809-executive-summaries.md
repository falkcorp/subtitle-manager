### Added

#### Executive summaries backfilled for the whole project history

`docs/executive-summaries/` is new, adopting the house format already used in
the `audiobook-organizer` repository. These documents exist so that someone who
makes funding and priority decisions, and who does not read code, can understand
what was done, what it cost, what risk it removed, and what is still unresolved.

A reusable `TEMPLATE-executive-summary.md` captures the conventions: filename
dated by when the document was written rather than the period it covers, a
`Shipped:` line linking an auditable GitHub pull-request search, a bulleted
executive summary followed by plain-language detail in the same order, an
explicit highest-risk block, and a required section on what remains unproven.

Five roundups cover the project from its first commit to the present, derived
from the merge record (913 merged pull requests):

- **June 2025** — the build-out month; 270 pull requests, more than the following
  eleven months combined.
- **July 2025** — the Bazarr feature-parity push, followed by a large internal
  library migration that produced no user-visible capability.
- **August 2025 – June 2026** — eleven months in which 296 of 340 merged pull
  requests were automated dependency updates. Documented briefly and honestly
  rather than inflated; covers two genuine supply-chain security events.
- **July 2026** — the restart, which delivered real parity features and
  discovered that several completed features had never been connected to a
  reachable address.
- **August 2026 (to date)** — language profiles finally wired end to end, two
  security fixes, a year of broken Windows builds repaired, and a verified
  inventory of what is still broken.

The summaries deliberately record failures alongside delivery, including work
lost before it was committed, features reported complete three times before
being connected, and a passing test suite that proved nothing because its
fixtures disagreed with the real server.
