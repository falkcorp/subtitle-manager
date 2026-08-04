// file: pkg/scanner/profile_test.go
// version: 1.2.0
// guid: 9e2b6c74-51af-4d38-a0e6-7b3c8f1d20a5
// last-edited: 2026-08-04

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
// conjures a profile row — one that could not then be deleted, since
// handleDeleteProfile refuses to remove the default. Pebble no longer creates
// on read, and anyLanguageProfiles reads the list, so the behaviour is now
// identical on SQLStore.
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

// assignedProfileLanguagesForTest calls through, mirroring the scan loop.
func assignedProfileLanguagesForTest(path string, store database.SubtitleStore) ([]string, bool) {
	return assignedProfileLanguages(path, store)
}

// TestExplicitDefaultProfileAssignmentIsHonoured is the bug this change exists
// to fix. A file explicitly assigned the *default* profile used to be
// indistinguishable from an unassigned one, because assignment was inferred
// from "the resolved profile is not the default" rather than from whether an
// assignment row existed. Such a file was scanned with the scan's own language
// instead of the profile's list — the user's explicit choice silently ignored.
//
// It is deliberately the mirror image of TestScanIgnoresDefaultProfile: same
// default profile, same resolved profile, and the only difference is that
// AssignProfileToMedia was called. The two must reach opposite conclusions.
func TestExplicitDefaultProfileAssignmentIsHonoured(t *testing.T) {
	dir := t.TempDir()
	vid := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(vid, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}
	viper.Set("media_directory", dir)
	defer viper.Reset()

	store := profileTestStore(t)
	defaultID := newProfile(t, store, "default", true,
		profiles.LanguageConfig{Language: "fr", Priority: 2},
		profiles.LanguageConfig{Language: "es", Priority: 1})

	if err := store.AssignProfileToMedia(vid, defaultID); err != nil {
		t.Fatalf("assign default profile: %v", err)
	}

	m := providersmocks.NewMockProvider(t)
	m.On("Fetch", mock.Anything, mock.Anything, "es").Return([]byte("es-sub"), nil).Once()
	m.On("Fetch", mock.Anything, mock.Anything, "fr").Return([]byte("fr-sub"), nil).Once()

	if err := ScanDirectoryProgress(context.Background(), dir, "en", "test", m, false, 1, store, nil); err != nil {
		t.Fatalf("scan: %v", err)
	}
	m.AssertExpectations(t)

	if _, err := os.Stat(filepath.Join(dir, "movie.en.srt")); err == nil {
		t.Error("wrote an en subtitle: an explicit assignment to the default profile was ignored")
	}
}

// TestDanglingAssignmentFallsBackToScanLanguage covers deleting a profile that
// files still reference. The assignment row survives the profile, and resolving
// it must not fail the scan — the file falls back to the scan's own language.
func TestDanglingAssignmentFallsBackToScanLanguage(t *testing.T) {
	dir := t.TempDir()
	vid := filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(vid, []byte("x"), 0644); err != nil {
		t.Fatalf("create video: %v", err)
	}
	viper.Set("media_directory", dir)
	defer viper.Reset()

	store := profileTestStore(t)
	newProfile(t, store, "keep", true, profiles.LanguageConfig{Language: "en", Priority: 1})
	gone := newProfile(t, store, "gone", false, profiles.LanguageConfig{Language: "fr", Priority: 1})
	if err := store.AssignProfileToMedia(vid, gone); err != nil {
		t.Fatalf("assign profile: %v", err)
	}
	if err := store.DeleteLanguageProfile(gone); err != nil {
		t.Fatalf("delete profile: %v", err)
	}

	m := providersmocks.NewMockProvider(t)
	m.On("Fetch", mock.Anything, mock.Anything, "en").Return([]byte("en-sub"), nil).Once()

	if err := ScanDirectoryProgress(context.Background(), dir, "en", "test", m, false, 1, store, nil); err != nil {
		t.Fatalf("scan: %v", err)
	}
	m.AssertExpectations(t)
}

// TestDeletingTheLastProfileStaysDeleted pins the resurrection bug.
//
// PebbleStore's GetDefaultLanguageProfile used to *create* a profile when the
// store was empty, and GetMediaProfile called it on every miss. So deleting the
// last profile only lasted until the next lookup, which wrote it back — flagged
// default, and therefore refused by the delete handler for good. Fixing only
// the delete guard would have left the bug reappearing one request later.
func TestDeletingTheLastProfileStaysDeleted(t *testing.T) {
	store := profileTestStore(t)

	// Pebble seeds a default profile on init, so clear whatever is there.
	list, err := store.ListLanguageProfiles()
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	for _, p := range list {
		if err := store.DeleteLanguageProfile(p.ID); err != nil {
			t.Fatalf("delete profile %s: %v", p.ID, err)
		}
	}

	// The lookup that used to resurrect it.
	if _, err := store.GetMediaProfile("/media/movie.mkv"); err == nil {
		t.Log("GetMediaProfile succeeded on an empty store; only the write-back matters")
	}

	after, err := store.ListLanguageProfiles()
	if err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	if len(after) != 0 {
		names := make([]string, 0, len(after))
		for _, p := range after {
			names = append(names, p.ID+"/"+p.Name)
		}
		t.Errorf("a lookup recreated %d profile(s) %v after the last was deleted", len(after), names)
	}
}

// TestProcessWithProfileIfAssignedReportsHandled pins the contract every wired
// download path depends on: the helper must say whether it took over, so the
// caller knows whether to fall through to its own single language.
//
// Getting this backwards in either direction is silent: reporting handled when
// it did nothing loses the download entirely, and reporting unhandled after
// downloading duplicates it.
func TestProcessWithProfileIfAssignedReportsHandled(t *testing.T) {
	dir := t.TempDir()
	assignedVid := filepath.Join(dir, "assigned.mkv")
	plainVid := filepath.Join(dir, "plain.mkv")
	for _, v := range []string{assignedVid, plainVid} {
		if err := os.WriteFile(v, []byte("x"), 0644); err != nil {
			t.Fatalf("create video: %v", err)
		}
	}
	viper.Set("media_directory", dir)
	defer viper.Reset()

	store := profileTestStore(t)
	newProfile(t, store, "default", true, profiles.LanguageConfig{Language: "en", Priority: 1})
	assigned := newProfile(t, store, "assigned", false,
		profiles.LanguageConfig{Language: "fr", Priority: 2},
		profiles.LanguageConfig{Language: "es", Priority: 1})
	if err := store.AssignProfileToMedia(assignedVid, assigned); err != nil {
		t.Fatalf("assign profile: %v", err)
	}

	m := providersmocks.NewMockProvider(t)
	m.On("Fetch", mock.Anything, mock.Anything, "es").Return([]byte("es-sub"), nil).Once()
	m.On("Fetch", mock.Anything, mock.Anything, "fr").Return([]byte("fr-sub"), nil).Once()

	handled, err := ProcessWithProfileIfAssigned(context.Background(), assignedVid, "test", m, false, store)
	if err != nil {
		t.Fatalf("assigned file: %v", err)
	}
	if !handled {
		t.Fatal("assigned file reported unhandled; the caller would download its own language instead")
	}
	m.AssertExpectations(t)

	// The unassigned file must be declined, not silently downloaded with the
	// default profile's languages — otherwise every file in the library becomes
	// profile-driven the moment a default exists.
	unassigned := providersmocks.NewMockProvider(t)
	handled, err = ProcessWithProfileIfAssigned(context.Background(), plainVid, "test", unassigned, false, store)
	if err != nil {
		t.Fatalf("unassigned file: %v", err)
	}
	if handled {
		t.Error("unassigned file reported handled; the caller would skip its own download")
	}
	unassigned.AssertExpectations(t)
}
