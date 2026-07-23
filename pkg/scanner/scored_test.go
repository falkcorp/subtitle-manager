// file: pkg/scanner/scored_test.go
// version: 1.0.0
// guid: c4e91a72-8b3d-4f60-a1c7-5d2e9f0b3a48
// last-edited: 2026-07-23

package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/providers/opensubtitles"
)

// fakeScored implements both providers.Provider and the scanner's scoredProvider
// capability so ProcessFile takes the scored path.
type fakeScored struct {
	results []opensubtitles.SearchResult
	byID    map[string][]byte
}

func (f *fakeScored) Fetch(ctx context.Context, mediaPath, lang string) ([]byte, error) {
	return []byte("plain"), nil
}

func (f *fakeScored) SearchWithResults(ctx context.Context, mediaPath, lang string) ([]opensubtitles.SearchResult, error) {
	return f.results, nil
}

func (f *fakeScored) FetchByResult(ctx context.Context, result opensubtitles.SearchResult) ([]byte, error) {
	return f.byID[result.Attributes.SubtitleID], nil
}

func srResult(id, release string, trusted, hd bool, downloads, votes int, rating float64) opensubtitles.SearchResult {
	var r opensubtitles.SearchResult
	r.Attributes.SubtitleID = id
	r.Attributes.Release = release
	r.Attributes.FromTrusted = trusted
	r.Attributes.HD = hd
	r.Attributes.DownloadCount = downloads
	r.Attributes.Votes = votes
	r.Attributes.Ratings = rating
	return r
}

// newFakeScored builds a provider offering a strong candidate ("good") matching
// the release and a weak one ("bad").
func newFakeScored(release string) *fakeScored {
	good := srResult("good", release, true, true, 5000, 200, 9.0)
	bad := srResult("bad", "", false, false, 0, 0, 0)
	return &fakeScored{
		results: []opensubtitles.SearchResult{bad, good}, // out of order on purpose
		byID:    map[string][]byte{"good": []byte("good sub"), "bad": []byte("bad sub")},
	}
}

// TestProcessFileScoredSelectsBest verifies that when scoring is enabled the
// highest-scoring candidate is the one downloaded and its score is persisted.
func TestProcessFileScoredSelectsBest(t *testing.T) {
	dir := t.TempDir()
	viper.Reset()
	viper.Set("media_directory", dir)
	viper.Set("scoring.enabled", true)
	viper.Set("scoring.min_score", 0)
	defer viper.Reset()

	base := "Inception.2010.1080p.BluRay.x264-GROUP"
	vid := filepath.Join(dir, base+".mkv")
	if err := os.WriteFile(vid, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}
	store, err := database.OpenPebble(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	p := newFakeScored(base)
	if err := ProcessFile(context.Background(), vid, "en", "opensubtitles", p, false, store); err != nil {
		t.Fatalf("process: %v", err)
	}

	sub := filepath.Join(dir, base+".en.srt")
	data, err := os.ReadFile(sub)
	if err != nil {
		t.Fatalf("read subtitle: %v", err)
	}
	if string(data) != "good sub" {
		t.Fatalf("expected best candidate downloaded, got %q", data)
	}

	recs, err := store.ListDownloadsByVideo(vid)
	if err != nil {
		t.Fatalf("list downloads: %v", err)
	}
	if len(recs) != 1 || recs[0].MatchScore == nil || *recs[0].MatchScore <= 0 {
		t.Fatalf("expected persisted positive match score, got %+v", recs)
	}
}

// TestProcessFileScoredGate verifies that when no candidate clears the minimum
// score, nothing is written and no error is returned.
func TestProcessFileScoredGate(t *testing.T) {
	dir := t.TempDir()
	viper.Reset()
	viper.Set("media_directory", dir)
	viper.Set("scoring.enabled", true)
	viper.Set("scoring.min_score", 100) // unreachable
	defer viper.Reset()

	base := "Inception.2010.1080p.BluRay.x264-GROUP"
	vid := filepath.Join(dir, base+".mkv")
	if err := os.WriteFile(vid, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}

	p := newFakeScored(base)
	if err := ProcessFile(context.Background(), vid, "en", "opensubtitles", p, false, nil); err != nil {
		t.Fatalf("process: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, base+".en.srt")); !os.IsNotExist(err) {
		t.Fatalf("expected no subtitle written when below min score")
	}
}

// TestPriorMatchScore verifies the helper returns the highest recorded score for
// the video/language pair.
func TestPriorMatchScore(t *testing.T) {
	store, err := database.OpenPebble(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	lo, hi := 0.4, 0.8
	_ = store.InsertDownload(&database.DownloadRecord{File: "a.srt", VideoFile: "v.mkv", Language: "en", MatchScore: &lo})
	_ = store.InsertDownload(&database.DownloadRecord{File: "b.srt", VideoFile: "v.mkv", Language: "en", MatchScore: &hi})
	_ = store.InsertDownload(&database.DownloadRecord{File: "c.srt", VideoFile: "v.mkv", Language: "fr", MatchScore: &hi})

	got := priorMatchScore(store, "v.mkv", "en")
	if got == nil || *got != hi {
		t.Fatalf("expected prior score %.2f, got %v", hi, got)
	}
	if priorMatchScore(store, "v.mkv", "de") != nil {
		t.Fatal("expected nil for language with no records")
	}
}
