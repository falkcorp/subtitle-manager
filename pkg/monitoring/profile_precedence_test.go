// file: pkg/monitoring/profile_precedence_test.go
// version: 1.0.0
// guid: 7c1d40b9-3e28-4a56-9f7d-2b8e0c4a6135
// last-edited: 2026-08-04

package monitoring

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/profiles"
	"github.com/jdfalk/subtitle-manager/pkg/providers"
	providersmocks "github.com/jdfalk/subtitle-manager/pkg/providers/mocks"
)

// monitorProfileStore opens a throwaway Pebble store. Pebble rather than SQLite
// so this runs without the sqlite build tag, which CI does not set.
func monitorProfileStore(t *testing.T) database.SubtitleStore {
	t.Helper()
	store, err := database.OpenStore(filepath.Join(t.TempDir(), "db"), "pebble")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// srtFor builds a minimal but genuinely well-formed SRT cue.
func srtFor(lang string) []byte {
	return []byte("1\n00:00:01,000 --> 00:00:02,000\n" + lang + " line\n")
}

// TestAssignedProfileBeatsMonitoredItemLanguages pins the precedence decision.
//
// MonitoredItem.Languages and a language profile are two independent
// desired-languages mechanisms, and something has to win. The profile does,
// because Languages is *accumulated* rather than curated — sync.go unions in
// every language it is ever asked for and never removes one, so it cannot
// express "stop wanting German". A profile assignment is a deliberate per-file
// choice made in the UI, and an append-only accumulation should not override
// a deliberate choice.
//
// The item below asks for "de" while the assigned profile asks for es then fr.
// Exactly the profile's languages must be fetched, and "de" must not be — if
// the two were merged instead of one winning, this test would catch that too.
func TestAssignedProfileBeatsMonitoredItemLanguages(t *testing.T) {
	dir := t.TempDir()
	vid := filepath.Join(dir, "episode.mkv")
	if err := os.WriteFile(vid, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}
	viper.Set("media_directory", dir)
	defer viper.Reset()

	store := monitorProfileStore(t)
	if err := store.CreateLanguageProfile(&database.LanguageProfile{
		ID:   "assigned",
		Name: "Assigned",
		Languages: []profiles.LanguageConfig{
			{Language: "fr", Priority: 2},
			{Language: "es", Priority: 1},
		},
		CutoffScore: 75,
	}); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := store.AssignProfileToMedia(vid, "assigned"); err != nil {
		t.Fatalf("assign profile: %v", err)
	}

	// The monitor calls the pipeline with a nil provider, so it resolves one
	// through the registry. Registering a factory and pinning the instance list
	// keeps this deterministic — otherwise FetchFromAll falls back to *every*
	// registered provider and the test makes real network calls.
	p := providersmocks.NewMockProvider(t)
	// The bytes must be real SRT: the multi-provider path rejects anything that
	// does not look like a subtitle (providers.looksLikeSubtitle), because stub
	// providers were answering 200 to everything and ending the search.
	p.On("Fetch", mock.Anything, mock.Anything, "es").Return(srtFor("es"), nil).Once()
	p.On("Fetch", mock.Anything, mock.Anything, "fr").Return(srtFor("fr"), nil).Once()
	// "de" is allowed but never expected. Without this the regression case —
	// item.Languages winning — makes testify call FailNow inside the provider
	// goroutine, which deadlocks against the wave's WaitGroup and turns a clear
	// assertion failure into a ten-minute hang. Declaring it lets the missing
	// es/fr calls and the stray de subtitle do the reporting instead.
	p.On("Fetch", mock.Anything, mock.Anything, "de").
		Return(srtFor("de"), nil).Maybe()
	providers.RegisterFactory("monitortest", func() providers.Provider { return p })
	prev := providers.ListInstances()
	providers.SetInstances([]providers.Instance{
		{ID: "monitortest", Name: "monitortest", Enabled: true},
	})
	t.Cleanup(func() { providers.SetInstances(prev) })

	m := NewEpisodeMonitor(time.Hour, nil, nil, store, 3, false)
	item := &MonitoredItem{
		ID:         "item-1",
		Path:       vid,
		Languages:  []string{"de"},
		MaxRetries: 3,
	}
	if err := m.processItem(context.Background(), item); err != nil {
		t.Fatalf("process item: %v", err)
	}

	p.AssertExpectations(t)
	if _, err := os.Stat(filepath.Join(dir, "episode.de.srt")); err == nil {
		t.Error("wrote a de subtitle: MonitoredItem.Languages overrode the assigned profile")
	}
	for _, lang := range []string{"es", "fr"} {
		if _, err := os.Stat(filepath.Join(dir, "episode."+lang+".srt")); err != nil {
			t.Errorf("no %s subtitle written: %v", lang, err)
		}
	}
	if item.Status != StatusFound {
		t.Errorf("status = %q, want %q: the profile path must still record the outcome",
			item.Status, StatusFound)
	}
}
