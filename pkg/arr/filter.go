// file: pkg/arr/filter.go
// version: 1.0.0
// guid: 94ab65c2-d217-4214-9857-eb91161a7dff

// Package arr holds shared filtering used by the Sonarr and Radarr clients to
// reach Bazarr parity: monitored-only mode, excluded series-types and tags, and
// path mappings (when *arr and subtitle-manager see files under different
// roots).
package arr

import (
	"strings"

	"github.com/spf13/viper"
)

// Filters describes which Sonarr/Radarr items to ingest and how to remap their
// paths. The zero value applies no filtering (backwards compatible).
type Filters struct {
	// MonitoredOnly drops items that are not monitored.
	MonitoredOnly bool
	// ExcludedSeriesTypes drops Sonarr series whose type matches (e.g. "anime",
	// "daily", "standard"). Case-insensitive. Ignored by Radarr.
	ExcludedSeriesTypes []string
	// ExcludedTagIDs drops items carrying any of these Sonarr/Radarr tag IDs.
	ExcludedTagIDs []int
	// PathMappings rewrites a leading path prefix {from → to} so subtitle-manager
	// can find files that *arr reports under a different root.
	PathMappings [][2]string
}

// IsZero reports whether no filtering/mapping is configured.
func (f Filters) IsZero() bool {
	return !f.MonitoredOnly && len(f.ExcludedSeriesTypes) == 0 &&
		len(f.ExcludedTagIDs) == 0 && len(f.PathMappings) == 0
}

// TypeExcluded reports whether seriesType is in the excluded set.
func (f Filters) TypeExcluded(seriesType string) bool {
	for _, t := range f.ExcludedSeriesTypes {
		if strings.EqualFold(strings.TrimSpace(t), strings.TrimSpace(seriesType)) {
			return true
		}
	}
	return false
}

// TagsExcluded reports whether any of tags is in the excluded set.
func (f Filters) TagsExcluded(tags []int) bool {
	if len(f.ExcludedTagIDs) == 0 {
		return false
	}
	set := make(map[int]struct{}, len(f.ExcludedTagIDs))
	for _, id := range f.ExcludedTagIDs {
		set[id] = struct{}{}
	}
	for _, id := range tags {
		if _, ok := set[id]; ok {
			return true
		}
	}
	return false
}

// MapPath applies the first matching path prefix mapping to p, or returns p
// unchanged. The longest "from" prefix wins so nested mappings are stable.
func (f Filters) MapPath(p string) string {
	best := -1
	var bestTo string
	for _, m := range f.PathMappings {
		from, to := m[0], m[1]
		if from != "" && strings.HasPrefix(p, from) && len(from) > best {
			best = len(from)
			bestTo = to + strings.TrimPrefix(p, from)
		}
	}
	if best >= 0 {
		return bestTo
	}
	return p
}

// FiltersFromViper reads filters from config under the given prefix, e.g.
// "integrations.sonarr" or "integrations.radarr".
func FiltersFromViper(prefix string) Filters {
	f := Filters{
		MonitoredOnly:       viper.GetBool(prefix + ".monitored_only"),
		ExcludedSeriesTypes: viper.GetStringSlice(prefix + ".excluded_series_types"),
		ExcludedTagIDs:      viper.GetIntSlice(prefix + ".excluded_tags"),
	}
	for from, to := range viper.GetStringMapString(prefix + ".path_mappings") {
		f.PathMappings = append(f.PathMappings, [2]string{from, to})
	}
	return f
}
