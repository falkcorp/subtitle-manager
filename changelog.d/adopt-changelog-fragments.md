### Changed

#### Adopt changelog fragments (`changelog.d/`) for assembling CHANGELOG.md

`CHANGELOG.md` is now assembled from per-change Markdown fragments under
`changelog.d/` by `scriv`, instead of being edited by hand. Contributors add a
fragment with `scriv create`; a CI check requires one on each PR, and the
fragments are folded into `CHANGELOG.md` when a release is published. This
removes changelog merge conflicts across parallel PRs.
