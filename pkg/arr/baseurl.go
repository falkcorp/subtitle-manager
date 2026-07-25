// file: pkg/arr/baseurl.go
// version: 1.0.0
// guid: 5a09a63e-a737-4060-a986-8976349f7bd9
// last-edited: 2026-07-25

package arr

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// BaseURL builds a Sonarr/Radarr base URL from config under the given prefix,
// e.g. "integrations.sonarr". Returns "" when the integration is disabled or
// has no host, so callers can treat the empty string as "not configured".
func BaseURL(prefix string) string {
	if !viper.GetBool(prefix + ".enabled") {
		return ""
	}
	host := viper.GetString(prefix + ".host")
	if host == "" {
		return ""
	}
	scheme := "http"
	if viper.GetBool(prefix + ".ssl") {
		scheme = "https"
	}
	base := strings.Trim(viper.GetString(prefix+".base_url"), "/")
	return fmt.Sprintf("%s://%s:%v/%s", scheme, host, viper.GetString(prefix+".port"), base)
}
