// file: pkg/security/tempdir_gate_test.go
// version: 1.1.0
// guid: 5a3e91c7-06b4-4d82-bf15-9e7c3a2048d6
// last-edited: 2026-08-04

package security

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// withProductionPathRules closes the temp-directory escape hatch for the
// duration of a test, so assertions can be made about how a real binary
// behaves. Every test otherwise runs with the hatch open, which makes the
// closed case untestable by construction.
func withProductionPathRules(t *testing.T) {
	t.Helper()
	prev := allowTempDirPaths
	allowTempDirPaths = false
	t.Cleanup(func() { allowTempDirPaths = prev })
}

// TestTempDirIsRejectedInProduction is the point of the change.
//
// ValidateAndSanitizePath used to accept *any* absolute path under
// os.TempDir(), with no build tag and no config key gating it. Since /tmp is
// world-writable, that voided allowed_base_dirs entirely: a path an attacker
// could steer under the temp directory was accepted no matter how the operator
// had configured the allowed roots.
func TestTempDirIsRejectedInProduction(t *testing.T) {
	withProductionPathRules(t)

	allowed := t.TempDir()
	viper.Set("media_directory", allowed)
	t.Cleanup(viper.Reset)

	// Somewhere under the system temp dir but outside the configured root.
	outside := filepath.Join(os.TempDir(), "attacker-controlled", "payload.mkv")
	if got, err := ValidateAndSanitizePath(outside); err == nil {
		t.Errorf("accepted %q (returned %q); a world-writable temp path must not "+
			"override allowed_base_dirs", outside, got)
	}
}

// TestConfiguredRootStillWorksInProduction is the control: closing the hatch
// must not break the paths an operator actually configured. Without this, a fix
// that simply refused everything would look correct.
func TestConfiguredRootStillWorksInProduction(t *testing.T) {
	withProductionPathRules(t)

	allowed := t.TempDir()
	viper.Set("media_directory", allowed)
	t.Cleanup(viper.Reset)

	inside := filepath.Join(allowed, "show", "episode.mkv")
	got, err := ValidateAndSanitizePath(inside)
	if err != nil {
		t.Fatalf("rejected a path inside the configured media directory: %v", err)
	}
	if got != inside {
		t.Errorf("got %q, want %q", got, inside)
	}
}

// TestTempDirIsAllowedInTests pins the other half of the contract: the hatch is
// open in a test binary, which is what keeps the ~68 test files that build
// fixtures under t.TempDir() working without each configuring an allowed root.
//
// If this ever fails, the gate has been tightened in a way that will break the
// suite rather than only production behaviour.
func TestTempDirIsAllowedInTests(t *testing.T) {
	if !allowTempDirPaths {
		t.Fatal("allowTempDirPaths is false inside a test binary; testing.Testing() should be true here")
	}

	viper.Set("media_directory", "/nonexistent-root-for-this-test")
	t.Cleanup(viper.Reset)

	fixture := filepath.Join(t.TempDir(), "fixture.mkv")
	if _, err := ValidateAndSanitizePath(fixture); err != nil {
		t.Errorf("rejected a t.TempDir() fixture path: %v", err)
	}
}

// TestTraversalStillRejectedUnderTempDir pins that the hatch never became a way
// to smuggle traversal through: the ".." check inside it is load-bearing even
// while the hatch is open.
func TestTraversalStillRejectedUnderTempDir(t *testing.T) {
	viper.Set("media_directory", "/nonexistent-root-for-this-test")
	t.Cleanup(viper.Reset)

	// filepath.Clean resolves the "..", so this lands outside the temp dir and
	// must not validate.
	escaping := filepath.Join(os.TempDir(), "..", "..", "etc", "passwd")
	if got, err := ValidateAndSanitizePath(escaping); err == nil {
		t.Errorf("accepted %q (returned %q)", escaping, got)
	}
}

// TestRelativeMediaDirectoryIsHonoured pins that a relative media_directory
// actually protects (and permits) the files under it.
//
// ValidateAndSanitizePath compares absolute paths, so a configured "./media"
// could never match and rejected every file beneath it. The unconditional
// temp-directory hatch masked this whenever the library sat under /tmp; closing
// that hatch in production exposed it, and `profiles assign` began failing with
// "path not in allowed directories" on a perfectly ordinary config.
func TestRelativeMediaDirectoryIsHonoured(t *testing.T) {
	withProductionPathRules(t)

	// EvalSymlinks because on macOS t.TempDir() hands back /var/folders/... while
	// the working directory resolves to /private/var/folders/... . Without this
	// the test compares two spellings of the same directory and fails for a
	// reason that has nothing to do with what it is checking.
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	media := filepath.Join(root, "media")
	if err := os.MkdirAll(media, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Configure it the way a user in that directory would.
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	viper.Set("media_directory", "./media")
	t.Cleanup(viper.Reset)

	inside := filepath.Join(media, "show", "episode.mkv")
	if _, err := ValidateAndSanitizePath(inside); err != nil {
		t.Errorf("rejected a file inside a relative media_directory: %v", err)
	}

	// The relative form must not become a way past the boundary either.
	outside := filepath.Join(root, "elsewhere", "secret.mkv")
	if got, err := ValidateAndSanitizePath(outside); err == nil {
		t.Errorf("accepted %q, which is outside the configured directory", got)
	}
}
