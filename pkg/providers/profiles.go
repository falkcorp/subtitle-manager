// file: pkg/providers/profiles.go
// version: 1.1.0
// guid: f1a2b3c4-d5e6-7f8a-9b0c-1d2e3f4a5b6c

package providers

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/profiles"
	"github.com/jdfalk/subtitle-manager/pkg/providers/opensubtitles"
	"github.com/jdfalk/subtitle-manager/pkg/scoring"
)

// FetchWithProfile attempts to fetch subtitles using language profile preferences.
// It tries languages in priority order from the profile associated with the media file.
func FetchWithProfile(ctx context.Context, db *sql.DB, mediaPath, key string) ([]byte, string, string, error) {
	service := profiles.NewService(db)

	// Get the language profile for this media file
	profile, err := service.GetMediaProfileByPath(mediaPath)
	if err != nil {
		// Fallback to default fetch if no profile found
		data, provider, fetchErr := FetchFromAll(ctx, mediaPath, "en", key)
		if fetchErr != nil {
			return nil, "", "", fmt.Errorf("failed to get profile (%v) and fallback fetch failed: %w", err, fetchErr)
		}
		return data, provider, "en", nil
	}

	// Sort languages by priority (lower number = higher priority)
	languages := make([]profiles.LanguageConfig, len(profile.Languages))
	copy(languages, profile.Languages)
	sort.Slice(languages, func(i, j int) bool {
		return languages[i].Priority < languages[j].Priority
	})

	// Try each language in priority order
	var lastErr error
	for _, langConfig := range languages {
		// Prefer a score-gated fetch that honors this language's Forced/HI
		// preferences and the profile cutoff score, when enabled.
		if data, ok := scoredProfileFetch(ctx, mediaPath, langConfig, profile.CutoffScore); ok {
			return data, "opensubtitles", langConfig.Language, nil
		}

		data, provider, err := FetchFromAll(ctx, mediaPath, langConfig.Language, key)
		if err == nil {
			return data, provider, langConfig.Language, nil
		}
		lastErr = err

		// Check if context was cancelled
		if ctx.Err() != nil {
			return nil, "", "", ctx.Err()
		}
	}

	return nil, "", "", fmt.Errorf("no subtitles found for any language in profile '%s': %w", profile.Name, lastErr)
}

// scoredProfileFetch attempts a score-gated OpenSubtitles download that honors
// the language config's Forced/HI preferences and the profile's cutoff score.
// It returns (data, true) on success, or (nil, false) to fall through to the
// default fetch. It is active only when scoring.enabled is set, so default
// behaviour is unchanged when scoring is off.
func scoredProfileFetch(ctx context.Context, mediaPath string, lang profiles.LanguageConfig, cutoff int) ([]byte, bool) {
	if !viper.GetBool("scoring.enabled") {
		return nil, false
	}
	client := opensubtitles.New("")
	results, err := client.SearchWithResults(ctx, mediaPath, lang.Language)
	if err != nil || len(results) == 0 {
		return nil, false
	}

	prof := scoring.LoadProfileFromConfig()
	prof.AllowHI = true
	prof.AllowForced = true
	prof.PreferHI = lang.HI
	prof.PreferForced = lang.Forced
	if cutoff > 0 {
		prof.MinScore = cutoff
	}

	best, _ := scoring.SelectBestResult(results, scoring.FromMediaPath(mediaPath), prof)
	if best == nil {
		return nil, false
	}
	data, err := client.FetchByResult(ctx, *best)
	if err != nil {
		return nil, false
	}
	return data, true
}

// FetchWithProfileTagged attempts to fetch subtitles using both profile preferences and provider tags.
func FetchWithProfileTagged(ctx context.Context, db *sql.DB, mediaPath, key string, tags []string, tm interface {
	FilterByTags(string, []string) ([]string, error)
}) ([]byte, string, string, error) {
	service := profiles.NewService(db)

	// Get the language profile for this media file
	profile, err := service.GetMediaProfileByPath(mediaPath)
	if err != nil {
		// Fallback to default tagged fetch if no profile found
		data, provider, fetchErr := FetchFromTagged(ctx, mediaPath, "en", key, tags, tm)
		if fetchErr != nil {
			return nil, "", "", fmt.Errorf("failed to get profile (%v) and fallback fetch failed: %w", err, fetchErr)
		}
		return data, provider, "en", nil
	}

	// Sort languages by priority (lower number = higher priority)
	languages := make([]profiles.LanguageConfig, len(profile.Languages))
	copy(languages, profile.Languages)
	sort.Slice(languages, func(i, j int) bool {
		return languages[i].Priority < languages[j].Priority
	})

	// Try each language in priority order with tagged providers
	var lastErr error
	for _, langConfig := range languages {
		data, provider, err := FetchFromTagged(ctx, mediaPath, langConfig.Language, key, tags, tm)
		if err == nil {
			return data, provider, langConfig.Language, nil
		}
		lastErr = err

		// Check if context was cancelled
		if ctx.Err() != nil {
			return nil, "", "", ctx.Err()
		}
	}

	return nil, "", "", fmt.Errorf("no subtitles found for any language in profile '%s' with tags %v: %w", profile.Name, tags, lastErr)
}

// GetLanguagesFromProfile extracts an ordered list of language codes from a profile.
func GetLanguagesFromProfile(ctx context.Context, db *sql.DB, mediaPath string) ([]string, error) {
	service := profiles.NewService(db)

	profile, err := service.GetMediaProfileByPath(mediaPath)
	if err != nil {
		return []string{"en"}, err // fallback to English
	}

	// Sort languages by priority
	languages := make([]profiles.LanguageConfig, len(profile.Languages))
	copy(languages, profile.Languages)
	sort.Slice(languages, func(i, j int) bool {
		return languages[i].Priority < languages[j].Priority
	})

	// Extract language codes
	codes := make([]string, len(languages))
	for i, lang := range languages {
		codes[i] = lang.Language
	}

	return codes, nil
}
