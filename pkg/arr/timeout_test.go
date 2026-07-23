// file: pkg/arr/timeout_test.go
// version: 1.0.0
// guid: 1f7c3a90-8d24-4b56-9e01-6a2f5d8c0473

package arr

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestTimeout(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	// Unset → default.
	if got := Timeout("integrations.sonarr"); got != DefaultTimeout {
		t.Fatalf("expected default %v, got %v", DefaultTimeout, got)
	}

	// Configured value in seconds.
	viper.Set("integrations.sonarr.timeout", 5)
	if got := Timeout("integrations.sonarr"); got != 5*time.Second {
		t.Fatalf("expected 5s, got %v", got)
	}

	// Non-positive → default.
	viper.Set("integrations.radarr.timeout", 0)
	if got := Timeout("integrations.radarr"); got != DefaultTimeout {
		t.Fatalf("expected default for 0, got %v", got)
	}
}
