// file: cmd/monitor.go
// version: 1.2.0
// last-edited: 2026-08-04
// guid: 12345678-1234-1234-1234-123456789014

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/jdfalk/subtitle-manager/pkg/arr"
	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/monitoring"
	"github.com/jdfalk/subtitle-manager/pkg/radarr"
	"github.com/jdfalk/subtitle-manager/pkg/sonarr"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Manage automatic subtitle monitoring",
	Long:  `Manage automatic subtitle monitoring for TV episodes and movies from Sonarr/Radarr`,
}

var monitorStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the monitoring daemon",
	Long:  `Start the automatic subtitle monitoring daemon`,
	RunE:  runMonitorStart,
}

var monitorSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync media from Sonarr/Radarr",
	Long:  `Synchronize media library from Sonarr/Radarr and add to monitoring`,
	RunE:  runMonitorSync,
}

var monitorAutoSyncCmd = &cobra.Command{
	Use:   "autosync",
	Short: "Run continuous Sonarr/Radarr sync",
	Long:  `Periodically synchronize media libraries from Sonarr/Radarr`,
	RunE:  runMonitorAutoSync,
}

var monitorStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show monitoring status",
	Long:  `Show current monitoring statistics and status`,
	RunE:  runMonitorStatus,
}

var monitorListCmd = &cobra.Command{
	Use:   "list",
	Short: "List monitored items",
	Long:  `List all items currently being monitored`,
	RunE:  runMonitorList,
}

var monitorBlacklistCmd = &cobra.Command{
	Use:   "blacklist",
	Short: "Manage blacklisted items",
	Long:  `Manage items that are blacklisted from monitoring`,
}

var monitorBlacklistListCmd = &cobra.Command{
	Use:   "list",
	Short: "List blacklisted items",
	Long:  `List all items currently blacklisted from monitoring`,
	RunE:  runMonitorBlacklistList,
}

var monitorBlacklistRemoveCmd = &cobra.Command{
	Use:   "remove [item-id]",
	Short: "Remove item from blacklist",
	Long:  `Remove an item from the blacklist to resume monitoring`,
	Args:  cobra.ExactArgs(1),
	RunE:  runMonitorBlacklistRemove,
}

// Flags for monitoring commands
var (
	monitorInterval     string
	monitorSyncInterval string
	monitorSyncSchedule string
	monitorLanguages    []string
	monitorMaxRetries   int
	monitorQualityCheck bool
	monitorForceRefresh bool
	monitorSource       string
)

func init() {
	// Add subcommands
	monitorCmd.AddCommand(monitorStartCmd)
	monitorCmd.AddCommand(monitorSyncCmd)
	monitorCmd.AddCommand(monitorAutoSyncCmd)
	monitorCmd.AddCommand(monitorStatusCmd)
	monitorCmd.AddCommand(monitorListCmd)
	monitorCmd.AddCommand(monitorBlacklistCmd)

	// Blacklist subcommands
	monitorBlacklistCmd.AddCommand(monitorBlacklistListCmd)
	monitorBlacklistCmd.AddCommand(monitorBlacklistRemoveCmd)

	// Start command flags
	monitorStartCmd.Flags().StringVar(&monitorInterval, "interval", "1h", "Monitoring interval (e.g. 30m, 1h, 2h)")
	monitorStartCmd.Flags().IntVar(&monitorMaxRetries, "max-retries", 3, "Maximum retry attempts per item")
	monitorStartCmd.Flags().BoolVar(&monitorQualityCheck, "quality-check", true, "Enable quality upgrade monitoring")

	// Sync command flags
	monitorSyncCmd.Flags().StringSliceVar(&monitorLanguages, "languages", []string{"en"}, "Languages to monitor (comma-separated)")
	monitorSyncCmd.Flags().IntVar(&monitorMaxRetries, "max-retries", 3, "Maximum retry attempts per item")
	monitorSyncCmd.Flags().BoolVar(&monitorForceRefresh, "force-refresh", false, "Refresh existing monitored items")
	monitorSyncCmd.Flags().StringVar(&monitorSource, "source", "both", "Source to sync from: sonarr, radarr, or both")

	// AutoSync command flags
	monitorAutoSyncCmd.Flags().StringVar(&monitorSyncInterval, "interval", "24h", "Sync interval (e.g. 12h, 24h)")
	monitorAutoSyncCmd.Flags().StringSliceVar(&monitorLanguages, "languages", []string{"en"}, "Languages to monitor (comma-separated)")
	monitorAutoSyncCmd.Flags().IntVar(&monitorMaxRetries, "max-retries", 3, "Maximum retry attempts per item")
	monitorAutoSyncCmd.Flags().StringVar(&monitorSource, "source", "both", "Source to sync from: sonarr, radarr, or both")
	monitorAutoSyncCmd.Flags().StringVar(&monitorSyncSchedule, "schedule", "", "Cron schedule expression")

	rootCmd.AddCommand(monitorCmd)
}

func runMonitorStart(cmd *cobra.Command, args []string) error {
	// Parse interval
	interval, err := time.ParseDuration(monitorInterval)
	if err != nil {
		return fmt.Errorf("invalid interval: %v", err)
	}

	// Open database
	store, err := database.OpenStoreWithConfig()
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer store.Close()

	sonarrClient, radarrClient := arrClients()

	// Create monitor
	monitor := monitoring.NewEpisodeMonitor(
		interval,
		sonarrClient,
		radarrClient,
		store,
		monitorMaxRetries,
		monitorQualityCheck,
	)

	fmt.Printf("Starting monitoring daemon (interval: %v)\n", interval)

	// Start monitoring
	ctx := context.Background()
	return monitor.Start(ctx)
}

func runMonitorSync(cmd *cobra.Command, args []string) error {
	// Open database
	store, err := database.OpenStoreWithConfig()
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer store.Close()

	sonarrClient, radarrClient := arrClients()

	// Create monitor
	monitor := monitoring.NewEpisodeMonitor(
		time.Hour, // Not used for sync
		sonarrClient,
		radarrClient,
		store,
		monitorMaxRetries,
		false, // Quality check not used for sync
	)

	// Sync options
	opts := monitoring.SyncOptions{
		Languages:    monitorLanguages,
		MaxRetries:   monitorMaxRetries,
		ForceRefresh: monitorForceRefresh,
	}

	ctx := context.Background()

	// Sync from Sonarr
	if (monitorSource == "both" || monitorSource == "sonarr") && sonarrClient != nil {
		fmt.Println("Syncing from Sonarr...")
		if err := monitor.SyncFromSonarr(ctx, opts); err != nil {
			return fmt.Errorf("failed to sync from Sonarr: %v", err)
		}
		fmt.Println("Sonarr sync complete")
	}

	// Sync from Radarr
	if (monitorSource == "both" || monitorSource == "radarr") && radarrClient != nil {
		fmt.Println("Syncing from Radarr...")
		if err := monitor.SyncFromRadarr(ctx, opts); err != nil {
			return fmt.Errorf("failed to sync from Radarr: %v", err)
		}
		fmt.Println("Radarr sync complete")
	}

	return nil
}

func runMonitorAutoSync(cmd *cobra.Command, args []string) error {
	interval, err := time.ParseDuration(monitorSyncInterval)
	if err != nil {
		return fmt.Errorf("invalid interval: %v", err)
	}

	store, err := database.OpenStoreWithConfig()
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer store.Close()

	sonarrClient, radarrClient := arrClients()

	sched := monitoring.NewScheduledMonitor(
		sonarrClient,
		radarrClient,
		store,
		monitorMaxRetries,
		false,
	)

	opts := monitoring.SyncOptions{
		Languages:  monitorLanguages,
		MaxRetries: monitorMaxRetries,
	}

	ctx := context.Background()
	if monitorSyncSchedule != "" {
		fmt.Printf("Starting scheduled sync using cron '%s'\n", monitorSyncSchedule)
		return monitoring.RunCronSync(ctx, monitorSyncSchedule, sched, opts)
	}

	fmt.Printf("Starting scheduled sync every %v\n", interval)
	return sched.StartScheduledSync(ctx, interval, opts)
}

func runMonitorStatus(cmd *cobra.Command, args []string) error {
	// Open database
	store, err := database.OpenStoreWithConfig()
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer store.Close()

	// Create monitor
	monitor := monitoring.NewEpisodeMonitor(
		time.Hour, // Not used for status
		nil, nil,  // Clients not needed for status
		store,
		0, false, // Not used for status
	)

	// Get stats
	stats, err := monitor.GetMonitoringStats()
	if err != nil {
		return fmt.Errorf("failed to get monitoring stats: %v", err)
	}

	// Display stats
	fmt.Printf("Monitoring Status:\n")
	fmt.Printf("  Total items:    %d\n", stats.Total)
	fmt.Printf("  Pending:        %d\n", stats.Pending)
	fmt.Printf("  Monitoring:     %d\n", stats.Monitoring)
	fmt.Printf("  Found:          %d\n", stats.Found)
	fmt.Printf("  Failed:         %d\n", stats.Failed)
	fmt.Printf("  Blacklisted:    %d\n", stats.Blacklisted)

	return nil
}

func runMonitorList(cmd *cobra.Command, args []string) error {
	// Open database
	store, err := database.OpenStoreWithConfig()
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer store.Close()

	// Get monitored items
	items, err := store.ListMonitoredItems()
	if err != nil {
		return fmt.Errorf("failed to list monitored items: %v", err)
	}

	if len(items) == 0 {
		fmt.Println("No items are currently being monitored.")
		return nil
	}

	// Display items
	fmt.Printf("Monitored Items (%d total):\n\n", len(items))
	for _, item := range items {
		fmt.Printf("ID: %s\n", item.ID)
		fmt.Printf("  Path: %s\n", item.Path)
		fmt.Printf("  Languages: %s\n", item.Languages)
		fmt.Printf("  Status: %s\n", item.Status)
		fmt.Printf("  Retries: %d/%d\n", item.RetryCount, item.MaxRetries)
		fmt.Printf("  Last Checked: %s\n", item.LastChecked.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Created: %s\n", item.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	return nil
}

func runMonitorBlacklistList(cmd *cobra.Command, args []string) error {
	// Open database
	store, err := database.OpenStoreWithConfig()
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer store.Close()

	// Create monitor
	monitor := monitoring.NewEpisodeMonitor(
		time.Hour, // Not used for blacklist ops
		nil, nil,  // Clients not needed
		store,
		0, false, // Not used
	)

	// Get blacklisted items
	items, err := monitor.GetBlacklistedItems()
	if err != nil {
		return fmt.Errorf("failed to get blacklisted items: %v", err)
	}

	if len(items) == 0 {
		fmt.Println("No items are currently blacklisted.")
		return nil
	}

	// Display blacklisted items
	fmt.Printf("Blacklisted Items (%d total):\n\n", len(items))
	for _, item := range items {
		fmt.Printf("ID: %s\n", item.ID)
		fmt.Printf("  Path: %s\n", item.Path)
		fmt.Printf("  Languages: %s\n", item.Languages)
		fmt.Printf("  Status: %s\n", item.Status)
		fmt.Printf("  Retries: %d/%d\n", item.RetryCount, item.MaxRetries)
		fmt.Printf("  Last Checked: %s\n", item.LastChecked.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Updated: %s\n", item.UpdatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	return nil
}

func runMonitorBlacklistRemove(cmd *cobra.Command, args []string) error {
	itemID := args[0]

	// Open database
	store, err := database.OpenStoreWithConfig()
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer store.Close()

	// Create monitor
	monitor := monitoring.NewEpisodeMonitor(
		time.Hour, // Not used for blacklist ops
		nil, nil,  // Clients not needed
		store,
		0, false, // Not used
	)

	// Remove from blacklist
	if err := monitor.RemoveFromBlacklist(itemID); err != nil {
		return fmt.Errorf("failed to remove from blacklist: %v", err)
	}

	fmt.Printf("Removed item %s from blacklist\n", itemID)
	return nil
}

// arrClients builds the Sonarr and Radarr clients from whichever config scheme
// is in use, returning nil for a service that is not configured.
//
// All three monitor entry points construct the same pair, and each used to read
// the flat sonarr_url/radarr_url keys directly — so a setup done through the
// settings UI or the Bazarr importer, which write integrations.*, left the
// monitor loop with no clients and nothing said so. arr.Resolve reads the
// canonical keys first and falls back to the flat ones.
func arrClients() (*sonarr.Client, *radarr.Client) {
	var sc *sonarr.Client
	if conn, ok := arr.Resolve(arr.Sonarr); ok {
		sc = sonarr.NewClient(conn.URL, conn.APIKey)
	}
	var rc *radarr.Client
	if conn, ok := arr.Resolve(arr.Radarr); ok {
		rc = radarr.NewClient(conn.URL, conn.APIKey)
	}
	return sc, rc
}
