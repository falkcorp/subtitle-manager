// file: cmd/fetch_scored.go
// version: 1.0.0
// guid: fedcba98-7654-3210-fedc-ba9876543210
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/providers/opensubtitles"
	"github.com/jdfalk/subtitle-manager/pkg/scoring"
)

var (
	minScore    int
	providerWgt float64
	releaseWgt  float64
	formatWgt   float64
	metadataWgt float64
	allowHI     bool
	preferHI    bool
)

var fetchScoredCmd = &cobra.Command{
	Use:   "fetch-scored [media] [lang] [output]",
	Short: "Download subtitles using quality-based scoring",
	Long: `Search for subtitles and automatically select the best match based on quality scoring.
This command evaluates subtitles based on:
- Provider reliability
- Release match quality
- Subtitle format preferences
- Upload date and popularity
- User preferences`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("fetch-scored")
		media, lang, out := args[0], args[1], args[2]
		key := viper.GetString("opensubtitles.api_key")

		if key == "" {
			return fmt.Errorf("opensubtitles.api_key is required for scored fetching")
		}

		logger.Infof("searching for subtitles for %s (language: %s)", media, lang)

		// Create OpenSubtitles client
		client := opensubtitles.New(key)

		// Search for subtitles
		ctx := context.Background()
		results, err := client.SearchWithResults(ctx, media, lang)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if len(results) == 0 {
			return fmt.Errorf("no subtitles found")
		}

		logger.Infof("found %d subtitle candidates", len(results))

		// Convert search results to scoring format
		subtitles := make([]scoring.Subtitle, len(results))
		for i, result := range results {
			subtitles[i] = scoring.FromOpenSubtitlesResult(result, "opensubtitles")
		}

		// Extract media information from path
		mediaItem := scoring.FromMediaPath(media)

		// Load scoring profile from configuration
		profile := scoring.LoadProfileFromConfig()

		// Validate profile
		if err := scoring.ValidateProfile(profile); err != nil {
			logger.Warnf("invalid scoring configuration: %v, using defaults", err)
			profile = scoring.DefaultProfile()
		}

		// Apply command-line overrides
		if cmd.Flags().Changed("min-score") {
			profile.MinScore = minScore
		}
		if cmd.Flags().Changed("provider-weight") {
			profile.ProviderWeight = providerWgt
		}
		if cmd.Flags().Changed("release-weight") {
			profile.ReleaseWeight = releaseWgt
		}
		if cmd.Flags().Changed("format-weight") {
			profile.FormatWeight = formatWgt
		}
		if cmd.Flags().Changed("metadata-weight") {
			profile.MetadataWeight = metadataWgt
		}
		if cmd.Flags().Changed("allow-hi") {
			profile.AllowHI = allowHI
		}
		if cmd.Flags().Changed("prefer-hi") {
			profile.PreferHI = preferHI
		}

		// Score and select best subtitle
		best, score := scoring.SelectBest(subtitles, mediaItem, profile)
		if best == nil {
			return fmt.Errorf("no suitable subtitle found")
		}

		logger.Infof("selected subtitle with score %d (provider: %d, release: %d, format: %d, metadata: %d)",
			score.Total, score.ProviderScore, score.ReleaseScore, score.FormatScore, score.MetadataScore)

		// Find corresponding search result for download
		var selectedResult *opensubtitles.SearchResult
		for i, subtitle := range subtitles {
			if subtitle.Release == best.Release &&
				subtitle.DownloadCount == best.DownloadCount &&
				subtitle.Rating == best.Rating {
				selectedResult = &results[i]
				break
			}
		}

		if selectedResult == nil {
			return fmt.Errorf("failed to find corresponding search result")
		}

		// Download the specific selected candidate (not merely the first search
		// result), so the scoring actually determines what gets written.
		data, err := client.FetchByResult(ctx, *selectedResult)
		if err != nil {
			return fmt.Errorf("download failed: %w", err)
		}

		// Write to output file
		if err := os.WriteFile(out, data, 0644); err != nil {
			return fmt.Errorf("write failed: %w", err)
		}

		// Record in database if configured
		if dbPath := viper.GetString("db_path"); dbPath != "" {
			backend := viper.GetString("db_backend")
			if store, err := database.OpenStore(dbPath, backend); err == nil {
				_ = store.InsertDownload(&database.DownloadRecord{
					File:      out,
					VideoFile: media,
					Provider:  "opensubtitles",
					Language:  lang,
				})
				store.Close()
			} else {
				logger.Warnf("db open: %v", err)
			}
		}

		logger.Infof("downloaded best-scored subtitle to %s", out)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(fetchScoredCmd)

	// Scoring configuration flags
	fetchScoredCmd.Flags().IntVar(&minScore, "min-score", 50, "Minimum score threshold (0-100)")
	fetchScoredCmd.Flags().Float64Var(&providerWgt, "provider-weight", 0.25, "Provider reliability weight (0.0-1.0)")
	fetchScoredCmd.Flags().Float64Var(&releaseWgt, "release-weight", 0.35, "Release match weight (0.0-1.0)")
	fetchScoredCmd.Flags().Float64Var(&formatWgt, "format-weight", 0.15, "Format preference weight (0.0-1.0)")
	fetchScoredCmd.Flags().Float64Var(&metadataWgt, "metadata-weight", 0.25, "Metadata quality weight (0.0-1.0)")
	fetchScoredCmd.Flags().BoolVar(&allowHI, "allow-hi", true, "Allow hearing impaired subtitles")
	fetchScoredCmd.Flags().BoolVar(&preferHI, "prefer-hi", false, "Prefer hearing impaired subtitles")
}
