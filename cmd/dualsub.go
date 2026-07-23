// file: cmd/dualsub.go
// version: 1.0.0
// guid: 7b9d1e2a-4c86-4f30-9b57-1e8a2d6c3f45

package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/security"
	"github.com/jdfalk/subtitle-manager/pkg/subtitles"
)

var (
	dualSubOutput   string
	dualSubSentinel string
)

// dualSubCmd generates a bilingual ("double subs") subtitle: the original text
// with a translation stacked beneath each cue, tagged with a sentinel language
// code (Esperanto "eo" by default) so players do not treat it as a real track.
var dualSubCmd = &cobra.Command{
	Use:   "dualsub [input] [target-lang]",
	Short: "Generate bilingual double-subs (original + translation) tagged as a sentinel language",
	Long: `Generate a bilingual "double subs" subtitle from an existing subtitle file:
each cue keeps its original line(s) and gains the translation stacked beneath.

The output is written with a sentinel language suffix (default "eo", Esperanto)
so Plex/Jellyfin/Emby treat it as a distinct, unused-language track rather than
overwriting a real subtitle. target-lang defaults to zh (Mandarin).`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("dualsub")
		in := args[0]
		targetLang := "zh"
		if len(args) == 2 {
			targetLang = args[1]
		}
		if err := security.ValidateLanguageCode(targetLang); err != nil {
			return fmt.Errorf("invalid target language: %w", err)
		}
		if err := security.ValidateLanguageCode(dualSubSentinel); err != nil {
			return fmt.Errorf("invalid sentinel language: %w", err)
		}

		out := dualSubOutput
		if out == "" {
			base := strings.TrimSuffix(in, filepath.Ext(in))
			out = base + "." + dualSubSentinel + ".srt"
		}

		service := viper.GetString("translate_service")
		gKey := viper.GetString("google_api_key")
		gptKey := viper.GetString("openai_api_key")
		grpcAddr := viper.GetString("grpc_addr")

		logger.Infof("generating double-subs %s -> %s (target=%s, tag=%s)", in, out, targetLang, dualSubSentinel)
		return subtitles.GenerateDualSubtitles(in, out, targetLang, service, gKey, gptKey, grpcAddr)
	},
}

func init() {
	dualSubCmd.Flags().StringVarP(&dualSubOutput, "output", "o", "", "output path (default: <input>.<sentinel>.srt)")
	dualSubCmd.Flags().StringVar(&dualSubSentinel, "sentinel", "eo", "sentinel language code used to tag the double-subs output")
	rootCmd.AddCommand(dualSubCmd)
}
