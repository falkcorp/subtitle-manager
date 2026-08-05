// file: cmd/sonarrsync.go
// version: 1.1.0
// last-edited: 2026-08-04
// guid: 1c9e3f2a-d0ff-4d6f-bd5a-385aaf8b9416

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jdfalk/subtitle-manager/pkg/arr"
	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/sonarr"
)

var sonarrSyncCmd = &cobra.Command{
	Use:   "sonarr-sync",
	Short: "Sync Sonarr library once",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("sonarr-sync")
		// arr.Resolve reads integrations.sonarr.* first and falls back to the
		// deprecated flat keys, so this command sees the same configuration
		// the web server does. Reading the flat keys directly meant a setup
		// done through the settings UI or the Bazarr importer left this
		// command believing nothing was configured.
		conn, ok := arr.Resolve(arr.Sonarr)
		if !ok || conn.APIKey == "" {
			logger.Warn("sonarr url or api key not configured")
			return nil
		}
		url, key := conn.URL, conn.APIKey

		c := sonarr.NewClient(url, key)
		store, err := database.OpenStoreWithConfig()
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer store.Close()

		ctx := context.Background()
		if err := sonarr.Sync(ctx, c, store); err != nil {
			return err
		}
		logger.Info("sonarr library sync complete")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sonarrSyncCmd)
}
