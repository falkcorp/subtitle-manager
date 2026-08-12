<!-- file: changelog.d/20260812-config-hierarchy-guard.md -->
<!-- version: 1.0.0 -->
<!-- guid: 2c68f409-7b31-4de5-9a02-84cf1e7360db -->
<!-- last-edited: 2026-08-12 -->

### Fixed

#### A settings save can no longer delete a whole group of settings

Configuration is a hierarchy — `providers.<name>.<field>`,
`integrations.<name>.<field>` — and settings pages write into it by dotted path.
Saving one option must never disturb another page's subtree.

That property holds, and is now pinned down by `TestConfigHierarchyIsPreserved`
rather than assumed. Verified against a config containing two providers and a
Sonarr integration: posting `providers.gestdown.enabled` as a dotted leaf, as a
nested object at the parent key, and with mixed-case spelling all deep-merge —
`gestdown`'s `api_key` and `priority`, the sibling `opensubtitles` block, and
`integrations.sonarr` survive every one.

There was one shape that did not merge. Assigning a **non-map to a key that
currently holds a map** replaces the whole subtree: `{"providers.gestdown":
"x"}` deleted that provider's `api_key` and `priority`, and the endpoint
answered `204`. Nothing in the UI sends that shape, which is exactly why it
needed to fail loudly instead of quietly destroying credentials — a settings
page with a wrong-shaped key would otherwise wipe an operator's provider
configuration and report success.

`POST /api/config` now rejects such a request with `400` and an explanatory
message, and **validates before mutating anything**: the check runs ahead of
`viper.Set`, so a rejected request leaves both the config file and the running
process's in-memory settings completely untouched. Otherwise the server would
have been left disagreeing with its own config file.

### Note

This was checked because of a concern about whether viper can express real
option hierarchies without entries clobbering each other. It can — no
replacement of viper or cobra is needed. Two behaviours are worth knowing:
viper **lowercases keys**, so `providers.GestDown` and `providers.gestdown` are
the same entry and cannot fork into duplicate sections; and `viper.Set`
populates the *override* layer while `viper.InConfig` reads the *file* layer,
which is why the config is re-read after each write.
