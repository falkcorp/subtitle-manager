### Security

#### `allowed_base_dirs` is no longer voided by a temp-directory bypass

`security.ValidateAndSanitizePath` accepted **any** absolute path under
`os.TempDir()`, unconditionally — no build tag, no config key, no comment
explaining the exposure. Because the system temp directory is world-writable,
that meant a path an attacker could steer under `/tmp` was accepted regardless
of how the operator had configured the allowed roots, so the setting protected
nothing against that one case.

The escape hatch existed for a real reason: the test suite builds fixtures under
`t.TempDir()`, and around 68 test files depend on those paths validating. It is
now gated on `testing.Testing()`, which is true only in a test binary — so the
hatch stays open exactly where it was needed and is closed in every shipped
binary, with no change required to any of those tests.

The gate is exposed as a package variable rather than an inline call, so the
production behaviour is actually testable: a test can close the hatch and assert
that a temp path is refused while a configured media root still validates.
Otherwise the closed case would be untestable by construction, since every test
runs with it open.
