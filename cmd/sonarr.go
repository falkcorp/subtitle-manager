// file: cmd/sonarr.go
// version: 1.0.0
// guid: 74705a61-aef7-400c-b5a5-23c2276a28ee
// last-edited: 2026-08-04

package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/scanner"
)

var sonarrCmd = &cobra.Command{
	Use:   "sonarr [lang]",
	Short: "Handle Sonarr download event",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("sonarr")
		lang := args[0]
		path := os.Getenv("SONARR_EPISODEFILE_PATH")
		if p := viper.GetString("path"); p != "" {
			path = p
		}
		if path == "" {
			return cmd.Usage()
		}
		ctx := context.Background()
		logger.Infof("processing %s", path)
		var store database.SubtitleStore
		if dbPath := viper.GetString("db_path"); dbPath != "" {
			backend := viper.GetString("db_backend")
			if s, err := database.OpenStore(dbPath, backend); err == nil {
				store = s
				defer s.Close()
			} else {
				logger.Warnf("db open: %v", err)
			}
		}
		// An assigned language profile governs an *arr import: nothing here
		// names a language for this particular file, so a choice made in the
		// UI should win over the connector's default.
		if handled, err := scanner.ProcessWithProfileIfAssigned(ctx, path, "", nil, true, store); handled {
			return err
		}
		return scanner.ProcessFile(ctx, path, lang, "", nil, true, store)
	},
}

func init() {
	sonarrCmd.Flags().String("path", "", "path to downloaded episode")
	rootCmd.AddCommand(sonarrCmd)
}
