// file: cmd/root.go
// version: 1.1.0
// guid: 537af48f-4b60-44b5-a4a1-76a2616b9ccb
// last-edited: 2026-08-12
// Package cmd implements the CLI commands for subtitle-manager.
// It provides the root command and subcommands for all user-facing operations.
//
// This package is the entry point for the application's command-line interface.
// It initializes configuration, sets up logging, and defines the available commands
// such as convert, merge, translate, fetch, batch, sonarr, radarr, and rename.
//
// The root command, when executed, will run the associated subcommand or show
// the help information if no subcommand is specified. Configuration options can
// be provided via command-line flags or through a configuration file, and are
// managed using the viper package. Logging is handled by the logrus package,
// with log level and output configuration options available.
//
// The package also includes database path and backend configuration, with
// support for SQLite, Pebble, and Postgres databases. API keys and other
// settings for translation and transcription services are also configured
// within this package.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/captcha"
	"github.com/jdfalk/subtitle-manager/pkg/database"
	gconfig "github.com/jdfalk/subtitle-manager/pkg/gcommon/config"
	"github.com/jdfalk/subtitle-manager/pkg/i18n"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/proxy"
	"github.com/jdfalk/subtitle-manager/pkg/transcriber"
	"github.com/jdfalk/subtitle-manager/pkg/translator"
)

var cfgFile string
var dbPath string
var dbBackend string
var sqliteFilename string
var showVersion bool
var flagsOnce sync.Once
var rootCmd = &cobra.Command{
	Use:   "subtitle-manager",
	Short: "Subtitle Manager CLI",
	Long:  `A simple subtitle management tool built in Go`,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle --version flag
		if showVersion {
			printVersion()
			return
		}
		// If no subcommand is specified, show help
		cmd.Help()
	},
}

// Execute executes the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logging.GetLogger("root").Fatalf("%v", err)
	}
}

// GetDatabasePath returns the full database path using the database package helper
func GetDatabasePath() string {
	return database.GetDatabasePath()
}

// GetDatabaseBackend returns the configured database backend using the database package helper
func GetDatabaseBackend() string {
	return database.GetDatabaseBackend()
}

var rootInit sync.Once

func init() {
	if rootCmd.PersistentFlags().Lookup("s3-region") != nil {
		return
	}
	rootInit.Do(func() {
		cobra.OnInitialize(initConfig)
		rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.subtitle-manager.yaml)")
		rootCmd.Flags().BoolVar(&showVersion, "version", false, "show version information")
		rootCmd.PersistentFlags().StringVar(&dbPath, "db-path", "", "database path (default is $HOME/.subtitle-manager/db for pebble, $HOME for sqlite)")
		rootCmd.PersistentFlags().StringVar(&dbBackend, "db-backend", "pebble", "database backend: sqlite, pebble, or postgres")
		rootCmd.PersistentFlags().StringVar(&sqliteFilename, "sqlite3-filename", "subtitle-manager.db", "SQLite database filename (only used when db-backend=sqlite)")
		rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
		viper.BindPFlag("log-level", rootCmd.PersistentFlags().Lookup("log-level"))
		rootCmd.PersistentFlags().StringToString("log-levels", nil, "per component log levels")
		viper.BindPFlag("log_levels", rootCmd.PersistentFlags().Lookup("log-levels"))
		rootCmd.PersistentFlags().String("log-file", "", "log file path")
		viper.BindPFlag("log_file", rootCmd.PersistentFlags().Lookup("log-file"))
		viper.BindPFlag("db_path", rootCmd.PersistentFlags().Lookup("db-path"))
		viper.BindPFlag("db_backend", rootCmd.PersistentFlags().Lookup("db-backend"))
		viper.BindPFlag("sqlite3_filename", rootCmd.PersistentFlags().Lookup("sqlite3-filename"))
		rootCmd.PersistentFlags().String("admin-user", "", "admin username for automatic setup (Docker/env only)")
		viper.BindPFlag("admin_user", rootCmd.PersistentFlags().Lookup("admin-user"))
		rootCmd.PersistentFlags().String("admin-password", "", "admin password for automatic setup (Docker/env only)")
		viper.BindPFlag("admin_password", rootCmd.PersistentFlags().Lookup("admin-password"))
		rootCmd.PersistentFlags().String("google-key", "", "Google Translate API key")
		viper.BindPFlag("google_api_key", rootCmd.PersistentFlags().Lookup("google-key"))
		rootCmd.PersistentFlags().String("openai-key", "", "OpenAI API key")
		viper.BindPFlag("openai_api_key", rootCmd.PersistentFlags().Lookup("openai-key"))
		rootCmd.PersistentFlags().String("openai-api-url", "", "OpenAI API base URL")
		viper.BindPFlag("openai_api_url", rootCmd.PersistentFlags().Lookup("openai-api-url"))
		rootCmd.PersistentFlags().String("opensubtitles-key", "", "OpenSubtitles API key")
		viper.BindPFlag("opensubtitles.api_key", rootCmd.PersistentFlags().Lookup("opensubtitles-key"))
		rootCmd.PersistentFlags().String("tmdb-key", "", "TMDB API key")
		viper.BindPFlag("tmdb_api_key", rootCmd.PersistentFlags().Lookup("tmdb-key"))
		rootCmd.PersistentFlags().String("omdb-key", "", "OMDb API key")
		viper.BindPFlag("omdb_api_key", rootCmd.PersistentFlags().Lookup("omdb-key"))
		rootCmd.PersistentFlags().String("ffmpeg-path", "", "path to ffmpeg binary")
		viper.BindPFlag("ffmpeg_path", rootCmd.PersistentFlags().Lookup("ffmpeg-path"))
		rootCmd.PersistentFlags().Int("batch-workers", 4, "number of workers for batch translate")
		viper.BindPFlag("batch_workers", rootCmd.PersistentFlags().Lookup("batch-workers"))
		rootCmd.PersistentFlags().Int("scan-workers", 4, "number of workers for scanning")
		viper.BindPFlag("scan_workers", rootCmd.PersistentFlags().Lookup("scan-workers"))
		rootCmd.PersistentFlags().String("queue-provider", "memory", "gcommon queue provider")
		viper.BindPFlag("queue.provider", rootCmd.PersistentFlags().Lookup("queue-provider"))
		rootCmd.PersistentFlags().Int("queue-workers", 3, "number of queue workers")
		viper.BindPFlag("queue.workers", rootCmd.PersistentFlags().Lookup("queue-workers"))
		rootCmd.PersistentFlags().String("google-api-url", "", "override Google Translate API URL")
		viper.BindPFlag("google_api_url", rootCmd.PersistentFlags().Lookup("google-api-url"))
		rootCmd.PersistentFlags().String("openai-model", "gpt-3.5-turbo", "ChatGPT model")
		viper.BindPFlag("openai_model", rootCmd.PersistentFlags().Lookup("openai-model"))
		rootCmd.PersistentFlags().String("opensubtitles-api-url", "", "OpenSubtitles API base URL")
		viper.BindPFlag("opensubtitles.api_url", rootCmd.PersistentFlags().Lookup("opensubtitles-api-url"))
		rootCmd.PersistentFlags().String("opensubtitles-user-agent", "", "OpenSubtitles user agent")
		viper.BindPFlag("opensubtitles.user_agent", rootCmd.PersistentFlags().Lookup("opensubtitles-user-agent"))

		// Whisper and Anti-Captcha provider flags
		rootCmd.PersistentFlags().String("whisper-api-url", "", "Whisper service URL")
		viper.BindPFlag("providers.whisper.api_url", rootCmd.PersistentFlags().Lookup("whisper-api-url"))

		rootCmd.PersistentFlags().String("anticaptcha-key", "", "Anti-Captcha API key")
		viper.BindPFlag("anticaptcha.api_key", rootCmd.PersistentFlags().Lookup("anticaptcha-key"))
		rootCmd.PersistentFlags().String("anticaptcha-api-url", "", "Anti-Captcha API base URL")
		viper.BindPFlag("anticaptcha.api_url", rootCmd.PersistentFlags().Lookup("anticaptcha-api-url"))

		// Cache configuration flags
		// These are kept as persistent flags so all subcommands
		// share the same cache settings.
		rootCmd.PersistentFlags().String("cache-backend", "memory", "cache backend: memory or redis")
		viper.BindPFlag("cache.backend", rootCmd.PersistentFlags().Lookup("cache-backend"))
		rootCmd.PersistentFlags().String("cache-redis-address", "localhost:6379", "Redis server address")
		viper.BindPFlag("cache.redis.address", rootCmd.PersistentFlags().Lookup("cache-redis-address"))
		rootCmd.PersistentFlags().String("cache-redis-password", "", "Redis password")
		viper.BindPFlag("cache.redis.password", rootCmd.PersistentFlags().Lookup("cache-redis-password"))
		rootCmd.PersistentFlags().Int("cache-redis-database", 0, "Redis database number")
		viper.BindPFlag("cache.redis.database", rootCmd.PersistentFlags().Lookup("cache-redis-database"))
		rootCmd.PersistentFlags().String("cache-redis-key-prefix", "subtitle-manager:", "Redis key prefix")
		viper.BindPFlag("cache.redis.key_prefix", rootCmd.PersistentFlags().Lookup("cache-redis-key-prefix"))
		rootCmd.PersistentFlags().Int("cache-memory-max-entries", 10000, "Maximum number of memory cache entries")
		viper.BindPFlag("cache.memory.max_entries", rootCmd.PersistentFlags().Lookup("cache-memory-max-entries"))
		rootCmd.PersistentFlags().Int64("cache-memory-max-memory", 104857600, "Maximum memory cache size in bytes (100MB)")
		viper.BindPFlag("cache.memory.max_memory", rootCmd.PersistentFlags().Lookup("cache-memory-max-memory"))

		// Add language support
		rootCmd.PersistentFlags().String("language", "en", "language for user interface (en, es, fr)")
		viper.BindPFlag("language", rootCmd.PersistentFlags().Lookup("language"))

		// S3 configuration
		rootCmd.PersistentFlags().String("s3-region", "", "S3 region")
		viper.BindPFlag("storage.s3_region", rootCmd.PersistentFlags().Lookup("s3-region"))
		rootCmd.PersistentFlags().String("s3-bucket", "", "S3 bucket name")
		viper.BindPFlag("storage.s3_bucket", rootCmd.PersistentFlags().Lookup("s3-bucket"))
		rootCmd.PersistentFlags().String("s3-endpoint", "", "S3 endpoint URL (for S3-compatible services)")
		viper.BindPFlag("storage.s3_endpoint", rootCmd.PersistentFlags().Lookup("s3-endpoint"))
		rootCmd.PersistentFlags().String("s3-access-key", "", "S3 access key")
		viper.BindPFlag("storage.s3_access_key", rootCmd.PersistentFlags().Lookup("s3-access-key"))
		rootCmd.PersistentFlags().String("s3-secret-key", "", "S3 secret key")
		viper.BindPFlag("storage.s3_secret_key", rootCmd.PersistentFlags().Lookup("s3-secret-key"))
		// Azure configuration and others defined above; duplicates removed

		// Base URL configuration for reverse proxy support
		rootCmd.PersistentFlags().String("base-url", "", "base URL path when behind a reverse proxy")
		viper.BindPFlag("base_url", rootCmd.PersistentFlags().Lookup("base-url"))

		// Cloud storage configuration
		rootCmd.PersistentFlags().String("storage-provider", "local", "storage provider: local, s3, azure, gcs")
		viper.BindPFlag("storage.provider", rootCmd.PersistentFlags().Lookup("storage-provider"))
		rootCmd.PersistentFlags().String("storage-local-path", "subtitles", "local storage path")
		viper.BindPFlag("storage.local_path", rootCmd.PersistentFlags().Lookup("storage-local-path"))
		// Azure configuration
		rootCmd.PersistentFlags().String("azure-account", "", "Azure storage account name")
		viper.BindPFlag("storage.azure_account", rootCmd.PersistentFlags().Lookup("azure-account"))
		rootCmd.PersistentFlags().String("azure-key", "", "Azure storage account key")
		viper.BindPFlag("storage.azure_key", rootCmd.PersistentFlags().Lookup("azure-key"))
		rootCmd.PersistentFlags().String("azure-container", "", "Azure blob container name")
		viper.BindPFlag("storage.azure_container", rootCmd.PersistentFlags().Lookup("azure-container"))
		// GCS configuration
		rootCmd.PersistentFlags().String("gcs-bucket", "", "Google Cloud Storage bucket name")
		viper.BindPFlag("storage.gcs_bucket", rootCmd.PersistentFlags().Lookup("gcs-bucket"))
		rootCmd.PersistentFlags().String("gcs-credentials", "", "Google Cloud credentials JSON file path")
		viper.BindPFlag("storage.gcs_credentials", rootCmd.PersistentFlags().Lookup("gcs-credentials"))
		// Storage options
		rootCmd.PersistentFlags().Bool("storage-enable-backup", false, "enable cloud backup of subtitle files")
		viper.BindPFlag("storage.enable_backup", rootCmd.PersistentFlags().Lookup("storage-enable-backup"))
		rootCmd.PersistentFlags().Bool("storage-backup-history", false, "enable cloud backup of history data")
		viper.BindPFlag("storage.backup_history", rootCmd.PersistentFlags().Lookup("storage-backup-history"))

		// Only commands that do NOT register themselves belong here.
		//
		// fetch, batch, sonarr, radarr and rename were listed here as well as
		// in their own files' init(), so cobra held two entries for each and
		// `--help` printed them twice. Every other command in cmd/ follows the
		// self-registration pattern; these five now do too.
		rootCmd.AddCommand(convertCmd)
		rootCmd.AddCommand(mergeCmd)
		rootCmd.AddCommand(translateCmd)
		rootCmd.AddCommand(queueCmd)
		rootCmd.AddCommand(profileCmd)
	})
}

func initConfig() {
	if err := gconfig.Load(rootCmd, cfgFile); err != nil {
		rootCmd.PrintErrf("failed to load config: %v\n", err)
	}

	// Set defaults (needs to happen regardless of config file source)
	home, err := os.UserHomeDir()
	if err == nil {
		viper.SetDefault("db_path", filepath.Join(home, ".subtitle-manager", "db"))
	} else {
		viper.SetDefault("db_path", "/config/db")
	}
	viper.SetDefault("db_backend", "pebble")
	viper.SetDefault("sqlite3_filename", "subtitle-manager.db")
	viper.SetDefault("admin_user", "")
	viper.SetDefault("admin_password", "")
	viper.SetDefault("translate_service", "google")
	viper.SetDefault("google_api_key", "")
	viper.SetDefault("openai_api_key", "")
	viper.SetDefault("opensubtitles.api_key", "")
	viper.SetDefault("tmdb_api_key", "")
	viper.SetDefault("omdb_api_key", "")
	viper.SetDefault("ffmpeg_path", "ffmpeg")
	viper.SetDefault("batch_workers", 4)
	viper.SetDefault("scan_workers", 4)
	viper.SetDefault("queue.provider", "memory")
	viper.SetDefault("queue.workers", 3)
	viper.SetDefault("google_api_url", "https://translation.googleapis.com/language/translate/v2")
	viper.SetDefault("openai_model", "gpt-3.5-turbo")
	viper.SetDefault("openai_api_url", "https://api.openai.com/v1")
	viper.SetDefault("opensubtitles.api_url", "https://rest.opensubtitles.org")
	viper.SetDefault("opensubtitles.user_agent", "github.com/jdfalk/subtitle-manager/0.1")
	viper.SetDefault("anticaptcha.api_key", "")
	viper.SetDefault("anticaptcha.api_url", "https://api.anti-captcha.com")
	viper.SetDefault("providers.generic.api_url", "")
	viper.SetDefault("providers.generic.username", "")
	viper.SetDefault("providers.generic.password", "")
	viper.SetDefault("providers.generic.api_key", "")
	viper.SetDefault("providers.whisper.api_url", "http://localhost:9000")
	viper.SetDefault("whisper.container_name", "whisper-asr-service")
	viper.SetDefault("whisper.image", "onerahmet/openai-whisper-asr-webservice:latest")
	viper.SetDefault("whisper.port", "9000")
	viper.SetDefault("whisper.model", "base")
	viper.SetDefault("whisper.device", "cuda")
	viper.SetDefault("whisper.use_gpu", true)
	viper.SetDefault("log_file", "/config/logs/subtitle-manager.log")
	viper.SetDefault("plex.url", "http://localhost:32400")
	viper.SetDefault("plex.token", "")
	viper.SetDefault("server_name", "Subtitle Manager")
	viper.SetDefault("base_url", "")

	// Cache configuration defaults
	viper.SetDefault("cache.backend", "memory")
	viper.SetDefault("cache.memory.max_entries", 10000)
	viper.SetDefault("cache.memory.max_memory", 104857600) // 100MB
	viper.SetDefault("cache.memory.default_ttl", "1h")
	viper.SetDefault("cache.memory.cleanup_interval", "10m")
	viper.SetDefault("cache.redis.address", "localhost:6379")
	viper.SetDefault("cache.redis.password", "")
	viper.SetDefault("cache.redis.database", 0)
	viper.SetDefault("cache.redis.pool_size", 10)
	viper.SetDefault("cache.redis.min_idle_conns", 2)
	viper.SetDefault("cache.redis.dial_timeout", "5s")
	viper.SetDefault("cache.redis.read_timeout", "3s")
	viper.SetDefault("cache.redis.write_timeout", "3s")
	viper.SetDefault("cache.redis.key_prefix", "subtitle-manager:")
	viper.SetDefault("cache.ttls.provider_search_results", "5m")
	viper.SetDefault("cache.ttls.tmdb_metadata", "24h")
	viper.SetDefault("cache.ttls.translation_results", "0") // permanent
	viper.SetDefault("cache.ttls.user_sessions", "24h")
	viper.SetDefault("cache.ttls.api_responses", "30m")
	viper.SetDefault("reverse_proxy", false)
	viper.SetDefault("integrations.sonarr.enabled", false)
	viper.SetDefault("integrations.radarr.enabled", false)
	viper.SetDefault("integrations.bazarr.import", false)
	viper.SetDefault("integrations.plex.enabled", false)
	viper.SetDefault("integrations.notifications.enabled", false)
	viper.SetDefault("auto_update", false)
	viper.SetDefault("update_branch", "master")
	viper.SetDefault("update_frequency", "daily")
	viper.SetDefault("db_cleanup_frequency", "daily")
	viper.SetDefault("metadata_refresh_frequency", "weekly")
	viper.SetDefault("disk_scan_frequency", "weekly")
	viper.SetDefault("language", "en")
	// Backup configuration defaults
	viper.SetDefault("backup_frequency", "daily")
	viper.SetDefault("backup_type", "full")
	viper.SetDefault("backup_compression_enabled", true)
	viper.SetDefault("backup_encryption_enabled", false)
	viper.SetDefault("backup_encryption_key", "")
	viper.SetDefault("backup_retention_days", 30)
	viper.SetDefault("backup_retention_count", 10)
	viper.SetDefault("backup_path", "")
	viper.SetDefault("backup_subtitle_paths", "")
	viper.SetDefault("backup_subtitle_restore_path", "")
	viper.SetDefault("backup_cloud_enabled", false)
	viper.SetDefault("backup_cloud_type", "")
	viper.SetDefault("backup_cloud_s3_region", "")
	viper.SetDefault("backup_cloud_s3_bucket", "")
	viper.SetDefault("backup_cloud_s3_prefix", "subtitle-manager-backups")
	viper.SetDefault("backup_cloud_s3_access_key", "")
	viper.SetDefault("backup_cloud_s3_secret_key", "")
	viper.SetDefault("backup_cloud_s3_endpoint", "")
	viper.SetDefault("backup_cloud_gcs_project_id", "")
	viper.SetDefault("backup_cloud_gcs_bucket", "")
	viper.SetDefault("backup_cloud_gcs_prefix", "subtitle-manager-backups")
	viper.SetDefault("backup_cloud_gcs_key_file", "")

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	// Initialize i18n
	i18n.Initialize()
	if lang := viper.GetString("language"); lang != "" {
		if err := i18n.SetLanguage(lang); err != nil {
			fmt.Printf("Warning: failed to set language %s: %v\n", lang, err)
		}
	}

	level, err := logrus.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)
	logging.Configure()
	if u := viper.GetString("google_api_url"); u != "" {
		translator.SetGoogleAPIURL(u)
	}
	if m := viper.GetString("openai_model"); m != "" {
		translator.SetOpenAIModel(m)
	}
	if u := viper.GetString("openai_api_url"); u != "" {
		transcriber.SetBaseURL(u)
	}
	// Route all outbound HTTP through a configured proxy when set. Applies to
	// every client using the default transport (providers, integrations, ...).
	if p := viper.GetString("proxy_url"); p != "" {
		if err := proxy.Configure(p); err != nil {
			logrus.Warnf("invalid proxy_url: %v", err)
		}
	}
	// The transcription model applies to the OpenAI-compatible path only; the
	// self-hosted whisper.transcribe_url (/asr) server's model is fixed at its
	// own startup and cannot be selected per-request.
	if m := viper.GetString("whisper.transcribe_model"); m != "" {
		transcriber.SetWhisperModel(m)
	}
	// Initialize Whisper container defaults
	transcriber.SetDefaultConfig()
	if u := viper.GetString("anticaptcha.api_url"); u != "" {
		captcha.SetAPIURL(u)
	}
}
