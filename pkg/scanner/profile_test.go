// file: pkg/scanner/profile_test.go
// version: 1.0.0
// guid: 9e2b6c74-51af-4d38-a0e6-7b3c8f1d20a5
// last-edited: 2026-08-01

package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/profiles"
	providersmocks "github.com/jdfalk/subtitle-manager/pkg/providers/mocks"
)

// profileTestStore opens a throwaway Pebble store. Pebble is used rather than
// SQLite so these tests run without the sqlite build tag, which CI does not
// set.
func profileTestStore(t *testing.T) database.SubtitleStore {
	t.Helper()
	store, err := database.OpenStore(filepath.Join(t.TempDir(), "db"), "pebble")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// newProfile stores a language profile and returns its ID.
func newProfile(t *testing.T, store database.SubtitleStore, name string, isDefault bool, langs ...profiles.LanguageConfig) string {
	t.Helper()
	id := name + "-id"
	if err := store.CreateLanguageProfile(&database.LanguageProfile{
		ID:          id,
		Name:        name,
		Languages:   langs,
		CutoffScore: 75,
		IsDefault:   isDefault,
	}); err != nil {
		t.Fatalf("create profile %s: %v", name, err)
	}
	if isDefault {
		if err := store.SetDefaultLanguageProfile(id); err != nil {
			t.Fatalf("set default %s: %v", name, err)
		}
	}
	return id
}

// TestScanUsesAssignedProfileLanguages is the point of the whole change: a file
// with its own language profile must be downloaded for the languages that
// profile asks for, in priority order, instead of the single language the scan
// was started with.
//
// Before this, no download path consulted a profile at all — ProcessFile was
// always called with one language, and scanner.ProcessFileWithProfile (which
// did iterate a profile) had no callers and read its assignment from a
// different database under a different key.
func TestScanUsesAssignedProfileLanguages(t *testing.T) {
	dir := t.TempDir()
	vid := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(vid, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}
	viper.Set("media_directory", dir)
	defer viper.Reset()

	store := profileTestStore(t)
	newProfile(t, store, "default", true,
		profiles.LanguageConfig{Language: "en", Priority: 1})
	assigned := newProfile(t, store, "assigned", false,
		profiles.LanguageConfig{Language: "fr", Priority: 2},
		profiles.LanguageConfig{Language: "es", Priority: 1})

	if err := store.AssignProfileToMedia(vid, assigned); err != nil {
		t.Fatalf("assign profile: %v", err)
	}

	// The scan is started with "en". The profile asks for es then fr, so those
	// are what the provider must be asked for — and "en" must not be.
	m := providersmocks.NewMockProvider(t)
	m.On("Fetch", mock.Anything, mock.Anything, "es").Return([]byte("es-sub"), nil).Once()
	m.On("Fetch", mock.Anything, mock.Anything, "fr").Return([]byte("fr-sub"), nil).Once()

	if err := ScanDirectoryProgress(context.Background(), dir, "en", "test", m, false, 1, store, nil); err != nil {
		t.Fatalf("scan: %v", err)
	}
	m.AssertExpectations(t)

	for _, lang := range []string{"es", "fr"} {
		if _, err := os.Stat(filepath.Join(dir, "movie."+lang+".srt")); err != nil {
			t.Errorf("no %s subtitle written: %v", lang, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "movie.en.srt")); err == nil {
		t.Error("wrote an en subtitle: the scan language was used despite an assigned profile")
	}
}

// TestScanIgnoresDefaultProfile pins that an unassigned file is left alone.
//
// GetMediaProfile falls back to the default profile rather than reporting a
// miss, so treating any resolved profile as an assignment would silently
// switch every file in the library to profile-driven downloading the moment a
// default existed. That is a behaviour change nobody asked for.
func TestScanIgnoresDefaultProfile(t *testing.T) {
	dir := t.TempDir()
	vid := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(vid, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}
	viper.Set("media_directory", dir)
	defer viper.Reset()

	store := profileTestStore(t)
	newProfile(t, store, "default", true,
		profiles.LanguageConfig{Language: "de", Priority: 1})

	// No assignment for vid. The scan language must win, not the default
	// profile's "de".
	m := providersmocks.NewMockProvider(t)
	m.On("Fetch", mock.Anything, mock.Anything, "en").Return([]byte("en-sub"), nil).Once()

	if err := ScanDirectoryProgress(context.Background(), dir, "en", "test", m, false, 1, store, nil); err != nil {
		t.Fatalf("scan: %v", err)
	}
	m.AssertExpectations(t)
}

// TestAssignedProfileLanguagesOrdersByPriority covers the ordering and the
// single-language-naming interaction directly.
func TestAssignedProfileLanguagesOrdersByPriority(t *testing.T) {
	dir := t.TempDir()
	vid := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(vid, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}
	viper.Set("media_directory", dir)
	defer viper.Reset()

	store := profileTestStore(t)
	newProfile(t, store, "default", true, profiles.LanguageConfig{Language: "en", Priority: 1})
	assigned := newProfile(t, store, "assigned", false,
		profiles.LanguageConfig{Language: "fr", Priority: 3},
		profiles.LanguageConfig{Language: "es", Priority: 1},
		profiles.LanguageConfig{Language: "de", Priority: 2})
	if err := store.AssignProfileToMedia(vid, assigned); err != nil {
		t.Fatalf("assign profile: %v", err)
	}

	got, ok := assignedProfileLanguagesForTest(vid, store)
	if !ok {
		t.Fatal("assignment not seen")
	}
	want := []string{"es", "de", "fr"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v (priority order not honoured)", got, want)
		}
	}

	// Single-language naming writes every language to the same file, so only
	// the highest-priority one is fetched.
	viper.Set("subtitles.single_language", true)
	got, ok = assignedProfileLanguagesForTest(vid, store)
	if !ok {
		t.Fatal("assignment not seen with single-language naming")
	}
	if len(got) != 1 || got[0] != "es" {
		t.Errorf("got %v with single-language naming, want [es]: every language "+
			"would otherwise write to the same file and overwrite the last", got)
	}
}

// TestScanCreatesNoProfileRows pins that scanning a library with no profiles
// configured leaves the profile table empty.
//
// PebbleStore.GetDefaultLanguageProfile *writes* a new profile with ID
// "default" when the store is empty. Because a scan now consults profiles per
// file, calling it would mean a first scan on a fresh install silently
// conjures a profile row — one that cannot then be deleted, since
// handleDeleteProfile refuses to remove the default. defaultProfileID reads
// the list instead, which also makes the behaviour identical on SQLStore,
// where GetDefaultLanguageProfile does not fall back or create.
func TestScanCreatesNoProfileRows(t *testing.T) {
	dir := t.TempDir()
	vid := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(vid, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}
	viper.Set("media_directory", dir)
	defer viper.Reset()

	store := profileTestStore(t)

	m := providersmocks.NewMockProvider(t)
	m.On("Fetch", mock.Anything, mock.Anything, "en").Return([]byte("en-sub"), nil).Once()

	if err := ScanDirectoryProgress(context.Background(), dir, "en", "test", m, false, 1, store, nil); err != nil {
		t.Fatalf("scan: %v", err)
	}

	list, err := store.ListLanguageProfiles()
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	if len(list) != 0 {
		names := make([]string, 0, len(list))
		for _, p := range list {
			names = append(names, p.ID+"/"+p.Name)
		}
		t.Errorf("scanning created %d profile(s) %v; a lookup must not write rows", len(list), names)
	}
}

// assignedProfileLanguagesForTest resolves the default once and calls through,
// mirroring what the scan loop does.
func assignedProfileLanguagesForTest(path string, store database.SubtitleStore) ([]string, bool) {
	id, _ := defaultProfileID(store)
	return assignedProfileLanguages(path, store, id)
}
