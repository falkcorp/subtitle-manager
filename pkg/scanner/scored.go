// file: pkg/scanner/scored.go
// version: 1.0.0
// guid: 9f2c8b41-6d0e-4a7c-b3f5-1e8a0c2d4b6a
// last-edited: 2026-07-23

package scanner

import (
	"context"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/providers/opensubtitles"
	"github.com/jdfalk/subtitle-manager/pkg/scoring"
)

// scoredProvider is the optional capability a provider implements when it can
// return candidate results carrying quality metadata and then download a
// specific candidate. Only the OpenSubtitles client implements it today; the
// scanner type-asserts to it and falls back to the plain byte path otherwise.
type scoredProvider interface {
	SearchWithResults(ctx context.Context, mediaPath, lang string) ([]opensubtitles.SearchResult, error)
	FetchByResult(ctx context.Context, result opensubtitles.SearchResult) ([]byte, error)
}

// scoringEnabled reports whether score-gated downloading is turned on. When it
// is off (the default) the scanner keeps its previous behaviour exactly.
func scoringEnabled() bool {
	return viper.GetBool("scoring.enabled")
}

// singleLanguageNaming reports whether subtitles should be written without the
// language code in the filename ("<base>.srt" rather than "<base>.<lang>.srt"),
// matching Bazarr's single-language naming option.
func singleLanguageNaming() bool {
	return viper.GetBool("subtitles.single_language")
}

// scoredResult is a downloaded subtitle together with its computed score.
type scoredResult struct {
	data    []byte
	total   int // 0-100
	release string
}

// fetchBestScored searches sp for candidate subtitles, scores them against the
// media, and downloads the best candidate that clears the profile's minimum
// score. It returns (nil, nil) when no candidate clears the threshold (a normal
// "nothing good enough" outcome, not an error) and a non-nil error only on a
// search or download failure.
func fetchBestScored(ctx context.Context, sp scoredProvider, mediaPath, lang string) (*scoredResult, error) {
	results, err := sp.SearchWithResults(ctx, mediaPath, lang)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}

	profile := scoring.LoadProfileFromConfig()
	if err := scoring.ValidateProfile(profile); err != nil {
		profile = scoring.DefaultProfile()
	}

	subs := make([]scoring.Subtitle, len(results))
	for i, r := range results {
		subs[i] = scoring.FromOpenSubtitlesResult(r, "opensubtitles")
	}
	media := scoring.FromMediaPath(mediaPath)

	best, score := scoring.SelectBest(subs, media, profile)
	if best == nil || score == nil {
		return nil, nil
	}
	// SelectBest falls back to the highest-scoring candidate even when it is
	// below MinScore, so enforce the threshold here: below it, download nothing.
	if score.Total < profile.MinScore {
		return nil, nil
	}

	idx := selectedIndex(subs, best)
	if idx < 0 {
		return nil, nil
	}

	data, err := sp.FetchByResult(ctx, results[idx])
	if err != nil {
		return nil, err
	}
	return &scoredResult{data: data, total: score.Total, release: best.Release}, nil
}

// selectedIndex finds the candidate matching the chosen subtitle by its
// distinguishing metadata. Returns -1 when no match is found.
func selectedIndex(subs []scoring.Subtitle, best *scoring.Subtitle) int {
	for i := range subs {
		if subs[i].Release == best.Release &&
			subs[i].DownloadCount == best.DownloadCount &&
			subs[i].Rating == best.Rating {
			return i
		}
	}
	return -1
}

// priorMatchScore returns the highest match score previously recorded for the
// given video/language pair, or nil when none is on file. It is used to decide
// whether a freshly scored candidate is actually an upgrade.
func priorMatchScore(store database.SubtitleStore, video, lang string) *float64 {
	if store == nil {
		return nil
	}
	recs, err := store.ListDownloadsByVideo(video)
	if err != nil {
		return nil
	}
	var best *float64
	for i := range recs {
		if recs[i].Language != lang || recs[i].MatchScore == nil {
			continue
		}
		if best == nil || *recs[i].MatchScore > *best {
			v := *recs[i].MatchScore
			best = &v
		}
	}
	return best
}
