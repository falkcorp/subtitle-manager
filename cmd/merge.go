// file: cmd/merge.go
// version: 1.1.0
// guid: 5d8f0a26-91c4-4e73-b2a8-6c17e93d0f4b
// last-edited: 2026-08-10

package cmd

import (
	"os"

	"github.com/asticode/go-astisub"
	"github.com/spf13/cobra"

	"github.com/jdfalk/subtitle-manager/pkg/logging"
	"github.com/jdfalk/subtitle-manager/pkg/security"
	"github.com/jdfalk/subtitle-manager/pkg/subtitles"
)

// mergeStack selects cue-by-cue stacking instead of interleaving. It is
// off by default so the existing behaviour of `merge` is unchanged.
var mergeStack bool

var mergeCmd = &cobra.Command{
	Use:   "merge [sub1] [sub2] [output]",
	Short: "Merge two subtitles into one",
	Long: `Merge two subtitles into one.

By default the two tracks are interleaved: every cue from both files is kept as
its own cue, ordered by start time. Cues that share a timestamp stay separate,
which players render as two competing overlapping subtitles.

With --stack the tracks are combined into bilingual "double subs" instead: each
cue carries the second language beneath the first, so one cue renders as two
lines. Cues are paired by greatest time overlap rather than exact timestamps,
because independently sourced tracks are rarely frame-identical, and a cue with
no counterpart is kept rather than dropped.`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.GetLogger("merge")
		sub1Path, err := security.SanitizePath(args[0])
		if err != nil {
			return err
		}
		sub1, err := astisub.OpenFile(string(sub1Path))
		if err != nil {
			return err
		}
		sub2Path, err := security.SanitizePath(args[1])
		if err != nil {
			return err
		}
		sub2, err := astisub.OpenFile(string(sub2Path))
		if err != nil {
			return err
		}
		if mergeStack {
			sub1.Items = subtitles.StackTracks(sub1.Items, sub2.Items)
		} else {
			sub1.Items = subtitles.MergeTracks(sub1.Items, sub2.Items)
		}
		outPath, err := security.SanitizePath(args[2])
		if err != nil {
			return err
		}
		f, err := os.Create(string(outPath))
		if err != nil {
			return err
		}
		defer f.Close()
		if err := sub1.WriteToSRT(f); err != nil {
			return err
		}
		if mergeStack {
			logger.Infof("Stacked %s and %s into bilingual %s", args[0], args[1], args[2])
		} else {
			logger.Infof("Merged %s and %s into %s", args[0], args[1], args[2])
		}
		return nil
	},
}

func init() {
	mergeCmd.Flags().BoolVar(&mergeStack, "stack", false,
		"combine into bilingual double-subs (one cue carrying both languages) instead of interleaving")
}
