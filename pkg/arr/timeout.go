// file: pkg/arr/timeout.go
// version: 1.0.0
// guid: 8b2f0e4c-6a19-4d73-9c50-1e7a3d6b8042
// last-edited: 2026-07-23

package arr

import (
	"time"

	"github.com/spf13/viper"
)

// DefaultTimeout is the request timeout used for Sonarr/Radarr calls when none
// is configured.
const DefaultTimeout = 30 * time.Second

// Timeout returns the configured HTTP request timeout for the given config
// prefix (e.g. "integrations.sonarr"), read from "<prefix>.timeout" in seconds.
// It falls back to DefaultTimeout when unset or non-positive, matching Bazarr's
// configurable Sonarr/Radarr request timeout.
func Timeout(prefix string) time.Duration {
	if secs := viper.GetInt(prefix + ".timeout"); secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return DefaultTimeout
}
