// file: pkg/scoring/selection.go
// version: 1.0.0
// guid: 3a6d1f08-9c47-4b25-8e0a-7f2c5d94b613
// last-edited: 2026-07-23

package scoring

import "github.com/jdfalk/subtitle-manager/pkg/providers/opensubtitles"

// SelectBestResult scores OpenSubtitles search results against media using
// profile and returns the highest-scoring result that meets profile.MinScore,
// together with its score. It returns (nil, nil) when there are no results or
// none clear the minimum score. Unlike SelectBest it returns the original
// search result, so the caller can download exactly that candidate without any
// fragile re-matching.
func SelectBestResult(results []opensubtitles.SearchResult, media MediaItem, profile Profile) (*opensubtitles.SearchResult, *SubtitleScore) {
	best := -1
	var bestScore SubtitleScore
	for i := range results {
		sub := FromOpenSubtitlesResult(results[i], "opensubtitles")
		sc := CalculateScore(sub, media, profile)
		if best == -1 || sc.Total > bestScore.Total {
			best = i
			bestScore = sc
		}
	}
	if best == -1 || bestScore.Total < profile.MinScore {
		return nil, nil
	}
	return &results[best], &bestScore
}
