// file: cmd/verify.go
// version: 1.0.0
// guid: 203bb659-7f2f-4b3c-8d86-10b58591ec59

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/security"
	"github.com/jdfalk/subtitle-manager/pkg/subsync"
	"github.com/jdfalk/subtitle-manager/pkg/video"
)

var (
	verifyAnchors int
	verifyTrack   int
)

// verifyCmd checks whether a subtitle stays aligned with a media file's audio
// across its runtime, detecting a constant offset and linear speed-up/slow-down
// drift by transcribing several audio windows with Whisper.
var verifyCmd = &cobra.Command{
	Use:   "verify [media] [subtitle] [lang]",
	Short: "Verify a subtitle's audio alignment (offset + speed drift) using Whisper",
	Long: `Sample several audio windows across the media, transcribe each with Whisper,
and compare the spoken timing to the subtitle. Reports whether the subtitle is in
sync, has a constant offset, or drifts (sped up / slowed down) — including a
likely framerate cause (e.g. 23.976 vs 25 fps).

Requires a configured Whisper endpoint (openai_api_key, base URL overridable for a
self-hosted server) and ffmpeg on PATH.`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("verify")
		media, subPath, lang := args[0], args[1], args[2]
		if err := security.ValidateLanguageCode(lang); err != nil {
			return err
		}

		info, err := video.AnalyzeVideo(media)
		if err != nil {
			return fmt.Errorf("probe media: %w", err)
		}
		if info.Duration <= 0 {
			return fmt.Errorf("could not determine media duration")
		}

		measure, err := subsync.NewWhisperMeasurer(media, subPath, lang, viper.GetString("openai_api_key"), verifyTrack, nil)
		if err != nil {
			return err
		}

		rep, err := subsync.Verify(context.Background(), subsync.VerifyOptions{
			MediaDuration: info.Duration,
			Anchors:       verifyAnchors,
		}, measure)
		if err != nil {
			return err
		}

		logger.Infof("verified %d/%d anchors", rep.Used, len(rep.Anchors))
		fmt.Printf("In sync:        %v\n", rep.InSync)
		fmt.Printf("Constant offset:%v (%.0f ms)\n", rep.ConstantOffset, rep.InterceptMs)
		fmt.Printf("Rate drift:     %v (%.2f ms/s)\n", rep.RateDrift, rep.SlopeMsPerSec)
		if rep.LikelyCause != "" {
			fmt.Printf("Likely cause:   %s\n", rep.LikelyCause)
		}
		fmt.Printf("Fit RMSE:       %.0f ms (max residual %.0f ms)\n", rep.RMSEMs, rep.MaxResidualMs)
		if !rep.InSync {
			return fmt.Errorf("subtitle is not in sync")
		}
		return nil
	},
}

func init() {
	verifyCmd.Flags().IntVar(&verifyAnchors, "anchors", 10, "number of audio windows to sample")
	verifyCmd.Flags().IntVar(&verifyTrack, "audio-track", 0, "audio track index to transcribe")
	rootCmd.AddCommand(verifyCmd)
}
