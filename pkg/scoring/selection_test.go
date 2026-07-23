// file: pkg/scoring/selection_test.go
// version: 1.0.0
// guid: 7d1a4e08-3c95-4b26-9f10-2a8d6c5e04b7

package scoring

import (
	"testing"

	"github.com/jdfalk/subtitle-manager/pkg/providers/opensubtitles"
)

func osResult(id, release string, trusted, hd bool, downloads int, rating float64) opensubtitles.SearchResult {
	var r opensubtitles.SearchResult
	r.Attributes.SubtitleID = id
	r.Attributes.Release = release
	r.Attributes.FromTrusted = trusted
	r.Attributes.HD = hd
	r.Attributes.DownloadCount = downloads
	r.Attributes.Ratings = rating
	return r
}

// TestSelectBestResult verifies the highest-scoring result above MinScore is
// returned as its original SearchResult, and that nothing is returned below it.
func TestSelectBestResult(t *testing.T) {
	base := "Inception.2010.1080p.BluRay.x264-GROUP"
	results := []opensubtitles.SearchResult{
		osResult("weak", "", false, false, 0, 0),
		osResult("strong", base, true, true, 5000, 9),
	}
	media := FromMediaPath("/media/" + base + ".mkv")

	prof := DefaultProfile()
	prof.MinScore = 0
	best, score := SelectBestResult(results, media, prof)
	if best == nil || best.Attributes.SubtitleID != "strong" {
		t.Fatalf("expected 'strong' to be selected, got %+v", best)
	}
	if score == nil || score.Total <= 0 {
		t.Fatalf("expected a positive score, got %+v", score)
	}

	// An unreachable cutoff returns nothing.
	prof.MinScore = 100
	if best, _ := SelectBestResult(results, media, prof); best != nil {
		t.Fatalf("expected nil above-cutoff selection, got %+v", best)
	}

	// No results → nil.
	if best, _ := SelectBestResult(nil, media, prof); best != nil {
		t.Fatal("expected nil for empty results")
	}
}
