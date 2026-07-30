// file: cmd/root_commands_test.go
// version: 1.0.0
// guid: 5b8e1074-2c93-4a6f-b015-7e2d94c3a681
// last-edited: 2026-07-30

package cmd

import "testing"

// TestNoDuplicateCommands guards against a command being registered twice.
//
// Five commands were listed both in root.go's setup block and in their own
// file's init(), so cobra held two entries for each and `--help` printed
// batch, fetch, radarr, sonarr and rename twice. Nothing broke functionally —
// which is why it survived — but a help listing that repeats itself reads as
// an unfinished tool, and it is the kind of thing a user notices in the first
// thirty seconds of evaluating the software.
func TestNoDuplicateCommands(t *testing.T) {
	seen := map[string]int{}
	for _, c := range rootCmd.Commands() {
		seen[c.Name()]++
	}
	for name, n := range seen {
		if n > 1 {
			t.Errorf("command %q is registered %d times; it will appear %d times "+
				"in --help. Register it in its own file's init() only.", name, n, n)
		}
	}
	if len(seen) == 0 {
		t.Fatal("no commands registered; this guard would pass vacuously")
	}
}
