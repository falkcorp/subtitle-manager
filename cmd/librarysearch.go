// file: cmd/librarysearch.go
// version: 1.0.0
// guid: 0e522c71-dd7d-4e80-af62-1fe449641282

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/scanner"
	"github.com/jdfalk/subtitle-manager/pkg/security"
)

var librarySearchUpgrade bool

// librarySearchCmd downloads subtitles for every item already recorded in the
// media library (populated by `scanlib` or the Sonarr/Radarr sync), rather than
// walking a directory. This is the bridge that turns a Sonarr/Radarr pull into
// actual subtitle downloads.
var librarySearchCmd = &cobra.Command{
	Use:   "library-search [lang]",
	Short: "Download subtitles for every item in the persisted media library",
	Long: `Iterate the media library stored in the database (as populated by the
scanlib command or the Sonarr/Radarr sync) and download subtitles for each item
using all configured providers. Items whose file is missing on disk are skipped.

This differs from 'scan', which walks a filesystem directory: library-search
operates on the authoritative file paths pulled from Sonarr/Radarr.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("library-search")
		lang := args[0]
		if err := security.ValidateLanguageCode(lang); err != nil {
			return err
		}

		dbPath := viper.GetString("db_path")
		if dbPath == "" {
			return fmt.Errorf("library-search requires a configured database (set db_path)")
		}
		store, err := database.OpenStore(dbPath, viper.GetString("db_backend"))
		if err != nil {
			return err
		}
		defer store.Close()

		workers := viper.GetInt("scan_workers")
		if workers < 1 {
			workers = 4
		}
		logger.Infof("searching subtitles for library items (lang=%s, workers=%d)", lang, workers)
		return scanner.ProcessLibrary(context.Background(), lang, "", nil, librarySearchUpgrade, workers, store)
	},
}

func init() {
	librarySearchCmd.Flags().BoolVarP(&librarySearchUpgrade, "upgrade", "u", false, "replace existing subtitles when a better one is found")
	rootCmd.AddCommand(librarySearchCmd)
}
