// file: cmd/fetch.go
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/providers"
	"github.com/jdfalk/subtitle-manager/pkg/scanner"
	"github.com/jdfalk/subtitle-manager/pkg/tagging"
)

var tags string
var useProfile bool

var fetchCmd = &cobra.Command{
	Use:   "fetch [media] [lang] [output]",
	Short: "Download subtitles using all providers",
	Long: `Download subtitles using all providers.

When --profile is specified, the language parameter is ignored and the system 
uses language preferences from the media item's assigned language profile.`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("fetch")
		media, lang, out := args[0], args[1], args[2]
		key := viper.GetString("opensubtitles.api_key")
		tagNames := []string{}
		if tags != "" {
			tagNames = strings.Split(tags, ",")
		}

		var data []byte
		var name string
		var actualLang string
		var err error

		// --profile routes through the scanner, which is the single place that
		// resolves an assigned language profile.
		//
		// It used to call providers.FetchWithProfile against a raw *sql.DB
		// opened with database.OpenSQLStore regardless of the configured
		// backend, so on Pebble — the default — it died with "unable to open
		// database file: is a directory". That path also read the second,
		// integer-keyed media_profiles implementation, which is not where the
		// web UI or the CLI write assignments, so even on SQLite it looked up
		// the wrong table.
		//
		// The scanner path downloads every language the profile asks for in
		// priority order and applies the profile's cutoff score and Forced/HI
		// flags, none of which the old call did.
		if useProfile {
			store, errStore := database.OpenStore(database.GetDatabasePath(), database.GetDatabaseBackend())
			if errStore != nil {
				return errStore
			}
			defer store.Close()

			handled, perr := scanner.ProcessWithProfileIfAssigned(context.Background(), media, "", nil, false, store)
			if perr != nil {
				return perr
			}
			if !handled {
				return fmt.Errorf("no language profile assigned to %s; assign one with `subtitle-manager profiles assign`, or drop --profile", media)
			}
			// The scanner names each file per language, so the positional
			// output argument does not apply here.
			logger.Infof("downloaded subtitles for %s using its assigned language profile", media)
			return nil
		}

		if len(tagNames) > 0 {
			dbPath := database.GetDatabasePath()
			store, errStore := database.OpenSQLStore(dbPath)
			if errStore != nil {
				return errStore
			}
			defer store.Close()

			{
				// Use tags only (existing behavior)
				tm := tagging.NewTagManager(store.DB())
				data, name, err = providers.FetchFromTagged(context.Background(), media, lang, key, tagNames, tm)
				actualLang = lang
			}
		} else {
			// Standard fetch without profiles or tags
			data, name, err = providers.FetchFromAll(context.Background(), media, lang, key)
			actualLang = lang
		}
		if err != nil {
			return err
		}
		if err := os.WriteFile(out, data, 0644); err != nil {
			return err
		}
		if dbPath := viper.GetString("db_path"); dbPath != "" {
			backend := viper.GetString("db_backend")
			if store, err := database.OpenStore(dbPath, backend); err == nil {
				_ = store.InsertDownload(&database.DownloadRecord{File: out, VideoFile: media, Provider: name, Language: actualLang})
				store.Close()
			} else {
				logger.Warnf("db open: %v", err)
			}
		}
		if useProfile {
			logger.Infof("downloaded %s subtitle using profile-based search to %s", actualLang, out)
		} else {
			logger.Infof("downloaded subtitle to %s", out)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)
	fetchCmd.Flags().StringVar(&tags, "tags", "", "comma separated provider tags")
	fetchCmd.Flags().BoolVar(&useProfile, "profile", false, "use language profile preferences for the media item")
}
