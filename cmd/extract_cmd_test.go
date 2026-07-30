package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/testutil"
)

// TestExtractCmdStoresRecord verifies that running the extract command
// records the generated subtitle in the database.
func TestExtractCmdStoresRecord(t *testing.T) {
	// This test uses SQLite file database, skip if not available
	if err := testutil.CheckSQLiteSupport(); err != nil {
		t.Skipf("SQLite support not available: %v", err)
	}

	dir := t.TempDir()
	script := filepath.Join(dir, "ffmpeg")
	data := "#!/bin/sh\ncp ../testdata/simple.srt \"$6\"\n"
	if err := os.WriteFile(script, []byte(data), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	out := filepath.Join(dir, "out.srt")
	dbPath := filepath.Join(dir, "test.db")

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	origDB := viper.GetString("db_path")
	origBackend := viper.GetString("db_backend")
	viper.Set("db_path", dbPath)
	viper.Set("db_backend", "sqlite")
	defer func() {
		viper.Set("db_path", origDB)
		viper.Set("db_backend", origBackend)
	}()

	// An absolute path: the extract command validates its input through
	// security.ValidateAndSanitizePath, which rejects relative paths. The test
	// predates that hardening and still passed a bare "video.mkv".
	media := filepath.Join(dir, "video.mkv")
	if err := os.WriteFile(media, []byte("fake media"), 0644); err != nil {
		t.Fatalf("write media: %v", err)
	}
	if err := extractCmd.RunE(extractCmd, []string{media, out}); err != nil {
		t.Fatalf("run: %v", err)
	}

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	recs, err := database.ListSubtitles(db)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recs))
	}
	r := recs[0]
	if r.File != out || r.VideoFile != media || !r.Embedded {
		t.Fatalf("unexpected record %+v", r)
	}
}
