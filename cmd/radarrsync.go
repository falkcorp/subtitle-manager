// file: cmd/radarrsync.go
// version: 1.1.0
// last-edited: 2026-08-04
// guid: a013d1b8-4cd3-4f59-8e4d-0d82cd9acae7

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jdfalk/subtitle-manager/pkg/arr"
	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/radarr"
)

var radarrSyncCmd = &cobra.Command{
	Use:   "radarr-sync",
	Short: "Sync Radarr library once",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("radarr-sync")
		// arr.Resolve reads integrations.radarr.* first and falls back to the
		// deprecated flat keys, so this command sees the same configuration
		// the web server does. Reading the flat keys directly meant a setup
		// done through the settings UI or the Bazarr importer left this
		// command believing nothing was configured.
		conn, ok := arr.Resolve(arr.Radarr)
		if !ok || conn.APIKey == "" {
			logger.Warn("radarr url or api key not configured")
			return nil
		}
		url, key := conn.URL, conn.APIKey

		c := radarr.NewClient(url, key)
		store, err := database.OpenStoreWithConfig()
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer store.Close()

		ctx := context.Background()
		if err := radarr.Sync(ctx, c, store); err != nil {
			return err
		}
		logger.Info("radarr library sync complete")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(radarrSyncCmd)
}
