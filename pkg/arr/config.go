// file: pkg/arr/config.go
// version: 1.0.0
// guid: 4e7b1c95-8a63-40d2-b7f1-9c05e28d3a64
// last-edited: 2026-08-04

package arr

import (
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/logging"
)

// Services are the *arr integrations this package knows about.
const (
	Sonarr = "sonarr"
	Radarr = "radarr"
)

// DefaultSyncInterval matches the fallback the web server has always used.
const DefaultSyncInterval = 60 * time.Minute

// warnOnce keeps the deprecation notice to one line per service per process
// rather than one per resolution, since the monitor loop resolves repeatedly.
var warnOnce sync.Map

// Connection is a resolved *arr endpoint.
type Connection struct {
	// URL is the fully composed base URL, scheme and base path included.
	URL string
	// APIKey is the X-Api-Key value.
	APIKey string
	// Legacy reports that this came from the deprecated flat keys rather than
	// the integrations.* block.
	Legacy bool
}

// Resolve returns the connection details for "sonarr" or "radarr", and whether
// one is configured at all.
//
// # Why this exists
//
// There were two independent config schemes and neither read the other:
//
//	integrations.<service>.*                  -> the web server, wanted list,
//	                                             *arr rescan notifications, and
//	                                             both clients' HTTP timeouts
//	<service>_url / <service>_api_key         -> cmd/monitor, cmd/sonarrsync,
//	                                             cmd/radarrsync
//
// So configuring the app through the settings UI or the Bazarr importer (both
// of which write integrations.*) left the monitor loop and the sync commands
// seeing nothing at all, and configuring the flat keys left the server blind.
// Each half looked correct in isolation, which is exactly how the notifications
// config bug behaved.
//
// integrations.* wins because it is what most of the codebase already reads,
// what the Bazarr importer writes, and the only one of the two that can express
// ssl, base_url and timeout. The flat keys still work so existing configs do
// not break, but they are reported as deprecated.
//
// Every caller now resolves through here, which means a config that only has
// the flat keys starts driving the web server's background sync too. That is
// the point of the change — the config finally means the same thing everywhere
// — but it is a behaviour change on upgrade for anyone who had set the flat
// keys and relied on the server ignoring them.
func Resolve(service string) (Connection, bool) {
	prefix := "integrations." + service

	if url := BaseURL(prefix); url != "" {
		conn := Connection{URL: url, APIKey: viper.GetString(prefix + ".api_key")}
		// Both schemes set is worth saying out loud: the flat keys are silently
		// doing nothing, which is the failure mode this change exists to end.
		if viper.GetString(service+"_url") != "" {
			warnf(service+"-shadowed",
				"%s is configured under integrations.%s and also via the deprecated %s_url; the flat keys are ignored",
				service, service, service)
		}
		return conn, true
	}

	// An integrations block with a host but enabled=false is an explicit
	// opt-out, so stale flat keys must not resurrect it. Only a completely
	// absent integrations block falls through to the legacy scheme.
	if viper.GetString(prefix+".host") != "" {
		return Connection{}, false
	}

	if url := viper.GetString(service + "_url"); url != "" {
		warnf(service+"-legacy",
			"%s is configured with the deprecated %s_url/%s_api_key keys; move it under integrations.%s so the web server sees it too",
			service, service, service, service)
		return Connection{
			URL:    url,
			APIKey: viper.GetString(service + "_api_key"),
			Legacy: true,
		}, true
	}

	return Connection{}, false
}

// SyncInterval returns how often to sync the given service.
//
// Sonarr's interval was read from "episode_sync_interval" while Radarr's was
// read from "sync_interval", so setting the obvious-looking
// integrations.sonarr.sync_interval was silently ignored and fell back to the
// default. Both spellings are now accepted for both services; the
// service-specific one wins when both are present, since that is what the
// Bazarr importer writes.
func SyncInterval(service string) time.Duration {
	prefix := "integrations." + service

	if service == Sonarr {
		if m := viper.GetInt(prefix + ".episode_sync_interval"); m > 0 {
			return time.Duration(m) * time.Minute
		}
	}
	if m := viper.GetInt(prefix + ".sync_interval"); m > 0 {
		return time.Duration(m) * time.Minute
	}
	return DefaultSyncInterval
}

// warnf logs once per key for the life of the process.
func warnf(key, format string, args ...any) {
	if _, loaded := warnOnce.LoadOrStore(key, true); loaded {
		return
	}
	logging.GetLogger("arr").Warnf(format, args...)
}
