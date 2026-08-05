// file: cmd/profiles.go
// version: 1.2.0
// last-edited: 2026-08-04
// guid: 3f4e5d6c-7b8a-9c0d-1e2f-4e5d6c7b8a9c

package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/profiles"
	"github.com/jdfalk/subtitle-manager/pkg/security"
	"github.com/spf13/cobra"
)

var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Manage language profiles",
	Long:  "Create, list, and manage language profiles for subtitle preferences",
}

var listProfilesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all language profiles",
	Long:  "Display all configured language profiles with their settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("profiles")

		store, err := database.OpenStore(database.GetDatabasePath(), database.GetDatabaseBackend())
		if err != nil {
			logger.Errorf("failed to open database: %v", err)
			return err
		}
		defer store.Close()

		profiles, err := store.ListLanguageProfiles()
		if err != nil {
			logger.Errorf("failed to list profiles: %v", err)
			return err
		}

		if len(profiles) == 0 {
			fmt.Println("No language profiles found")
			return nil
		}

		// Create a table writer for nice formatting
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID\tName\tDefault\tCutoff\tLanguages\n")
		fmt.Fprintf(w, "--\t----\t-------\t------\t---------\n")

		for _, profile := range profiles {
			var langStrs []string
			for _, lang := range profile.Languages {
				langStr := fmt.Sprintf("%s(%d)", lang.Language, lang.Priority)
				if lang.Forced {
					langStr += "[F]"
				}
				if lang.HI {
					langStr += "[HI]"
				}
				langStrs = append(langStrs, langStr)
			}

			defaultStr := ""
			if profile.IsDefault {
				defaultStr = "✓"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%d%%\t%s\n",
				profile.ID[:8], // Show only first 8 chars of ID
				profile.Name,
				defaultStr,
				profile.CutoffScore,
				strings.Join(langStrs, ", "))
		}

		w.Flush()
		return nil
	},
}

var createProfileCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new language profile",
	Long:  "Create a new language profile with specified languages and settings",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("profiles")
		profileName := args[0]

		languages, _ := cmd.Flags().GetStringSlice("languages")
		cutoff, _ := cmd.Flags().GetInt("cutoff")
		setDefault, _ := cmd.Flags().GetBool("default")

		if len(languages) == 0 {
			return fmt.Errorf("at least one language must be specified with --languages")
		}

		// Parse languages into LanguageConfig
		var langConfigs []profiles.LanguageConfig
		for i, lang := range languages {
			parts := strings.Split(lang, ":")
			langCode := parts[0]
			forced := false
			hi := false

			if len(parts) > 1 {
				for _, flag := range strings.Split(parts[1], ",") {
					switch strings.ToLower(flag) {
					case "forced", "f":
						forced = true
					case "hi", "hearingimpaired":
						hi = true
					}
				}
			}

			langConfigs = append(langConfigs, profiles.LanguageConfig{
				Language: langCode,
				Priority: i + 1, // 1-based priority
				Forced:   forced,
				HI:       hi,
			})
		}

		profile := &profiles.LanguageProfile{
			ID:          uuid.NewString(),
			Name:        profileName,
			Languages:   langConfigs,
			CutoffScore: cutoff,
			IsDefault:   setDefault,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		// Validate profile
		if err := profile.Validate(); err != nil {
			logger.Errorf("profile validation failed: %v", err)
			return err
		}

		store, err := database.OpenStore(database.GetDatabasePath(), database.GetDatabaseBackend())
		if err != nil {
			logger.Errorf("failed to open database: %v", err)
			return err
		}
		defer store.Close()

		if err := store.CreateLanguageProfile(profile); err != nil {
			logger.Errorf("failed to create profile: %v", err)
			return err
		}

		// Set as default if requested
		if setDefault {
			if err := store.SetDefaultLanguageProfile(profile.ID); err != nil {
				logger.Errorf("failed to set as default: %v", err)
				return err
			}
		}

		fmt.Printf("Created language profile: %s (ID: %s)\n", profile.Name, profile.ID)
		return nil
	},
}

var assignProfileCmd = &cobra.Command{
	Use:   "assign [media-path] [profile-id]",
	Short: "Assign language profile to media item",
	Long:  "Assign a language profile to a specific media file or directory",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("profiles")
		mediaPath := args[0]
		profileID := args[1]

		// Validate and sanitize the media path to prevent path traversal attacks
		validatedPath, err := security.ValidateAndSanitizePath(mediaPath)
		if err != nil {
			logger.Errorf("invalid media path: %v", err)
			return fmt.Errorf("invalid media path: %w", err)
		}

		store, err := database.OpenStore(database.GetDatabasePath(), database.GetDatabaseBackend())
		if err != nil {
			logger.Errorf("failed to open database: %v", err)
			return err
		}
		defer store.Close()

		// Verify profile exists
		// Pebble returns (nil, nil) for an unknown ID, so an error check alone
		// would accept one. See database.ProfileMissing.
		profile, err := store.GetLanguageProfile(profileID)
		if database.ProfileMissing(profile, err) {
			return fmt.Errorf("profile %s not found", profileID)
		}
		if err != nil {
			logger.Errorf("failed to look up profile: %v", err)
			return err
		}

		// Assign profile to media using the validated path
		if err := store.AssignProfileToMedia(validatedPath, profileID); err != nil {
			logger.Errorf("failed to assign profile: %v", err)
			return err
		}

		fmt.Printf("Assigned profile '%s' to media: %s\n", profile.Name, validatedPath)
		return nil
	},
}

var removeProfileCmd = &cobra.Command{
	Use:   "remove [media-path]",
	Short: "Remove language profile assignment from media item",
	Long:  "Remove language profile assignment from a media file, reverting to default",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("profiles")
		mediaPath := args[0]

		// Validate and sanitize the media path to prevent path traversal attacks
		validatedPath, err := security.ValidateAndSanitizePath(mediaPath)
		if err != nil {
			logger.Errorf("invalid media path: %v", err)
			return fmt.Errorf("invalid media path: %w", err)
		}

		store, err := database.OpenStore(database.GetDatabasePath(), database.GetDatabaseBackend())
		if err != nil {
			logger.Errorf("failed to open database: %v", err)
			return err
		}
		defer store.Close()

		if err := store.RemoveProfileFromMedia(validatedPath); err != nil {
			logger.Errorf("failed to remove profile assignment: %v", err)
			return err
		}

		fmt.Printf("Removed profile assignment from media: %s\n", validatedPath)
		return nil
	},
}

var showMediaProfileCmd = &cobra.Command{
	Use:   "show [media-path]",
	Short: "Show language profile assigned to media item",
	Long:  "Display the language profile currently assigned to a media file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("profiles")
		mediaPath := args[0]

		// Validate and sanitize the media path to prevent path traversal attacks
		validatedPath, err := security.ValidateAndSanitizePath(mediaPath)
		if err != nil {
			logger.Errorf("invalid media path: %v", err)
			return fmt.Errorf("invalid media path: %w", err)
		}

		store, err := database.OpenStore(database.GetDatabasePath(), database.GetDatabaseBackend())
		if err != nil {
			logger.Errorf("failed to open database: %v", err)
			return err
		}
		defer store.Close()

		profile, err := store.GetMediaProfile(validatedPath)
		if err != nil {
			logger.Errorf("failed to get media profile: %v", err)
			return err
		}

		fmt.Printf("Media: %s\n", validatedPath)
		fmt.Printf("Profile: %s (ID: %s)\n", profile.Name, profile.ID)
		if profile.IsDefault {
			fmt.Printf("Type: Default Profile\n")
		} else {
			fmt.Printf("Type: Custom Profile\n")
		}
		fmt.Printf("Cutoff Score: %d%%\n", profile.CutoffScore)
		fmt.Printf("Languages:\n")

		for _, lang := range profile.Languages {
			flags := ""
			if lang.Forced {
				flags += " [Forced]"
			}
			if lang.HI {
				flags += " [HI]"
			}
			fmt.Printf("  %d. %s%s\n", lang.Priority, lang.Language, flags)
		}

		return nil
	},
}

var deleteProfileCmd = &cobra.Command{
	Use:   "delete [profile-id]",
	Short: "Delete a language profile",
	Long:  "Delete a language profile (cannot delete the default profile)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("profiles")
		profileID := args[0]

		store, err := database.OpenStore(database.GetDatabasePath(), database.GetDatabaseBackend())
		if err != nil {
			logger.Errorf("failed to open database: %v", err)
			return err
		}
		defer store.Close()

		// Check if it's the default profile
		// Pebble returns (nil, nil) for an unknown ID, so an error check alone
		// would accept one. See database.ProfileMissing.
		profile, err := store.GetLanguageProfile(profileID)
		if database.ProfileMissing(profile, err) {
			return fmt.Errorf("profile %s not found", profileID)
		}
		if err != nil {
			logger.Errorf("failed to look up profile: %v", err)
			return err
		}

		if profile.IsDefault {
			return fmt.Errorf("cannot delete the default profile")
		}

		if err := store.DeleteLanguageProfile(profileID); err != nil {
			logger.Errorf("failed to delete profile: %v", err)
			return err
		}

		fmt.Printf("Deleted language profile: %s\n", profile.Name)
		return nil
	},
}

func init() {
	// Add flags for create command
	createProfileCmd.Flags().StringSlice("languages", []string{"en"}, "Languages in priority order (format: code or code:flags, e.g., en, es:forced, fr:hi)")
	createProfileCmd.Flags().Int("cutoff", 75, "Minimum quality score (0-100)")
	createProfileCmd.Flags().Bool("default", false, "Set this profile as the default")

	// Add subcommands to profiles command
	profilesCmd.AddCommand(listProfilesCmd)
	profilesCmd.AddCommand(createProfileCmd)
	profilesCmd.AddCommand(assignProfileCmd)
	profilesCmd.AddCommand(removeProfileCmd)
	profilesCmd.AddCommand(showMediaProfileCmd)
	profilesCmd.AddCommand(deleteProfileCmd)

	// Add profiles command to root
	rootCmd.AddCommand(profilesCmd)
}
