// file: pkg/backups/service_test.go
// version: 1.0.0
// guid: 123e4567-e89b-12d3-a456-426614174012

package backups

import (
	"context"
	"fmt"
	"testing"
	"time"

	common "github.com/falkcorp/gcommon/v2/pkg/commonpb/v2"
	"github.com/jdfalk/subtitle-manager/pkg/database"
	"github.com/jdfalk/subtitle-manager/pkg/logging"
)

// mockSubtitleStore implements database.SubtitleStore for testing
type mockSubtitleStore struct {
	subtitles  []database.SubtitleRecord
	downloads  []database.DownloadRecord
	mediaItems []database.MediaItem
	tags       []database.Tag
}

func newMockSubtitleStore() *mockSubtitleStore {
	return &mockSubtitleStore{
		subtitles: []database.SubtitleRecord{
			{File: "test1.srt", VideoFile: "test1.mp4", Language: "en"},
			{File: "test2.srt", VideoFile: "test2.mp4", Language: "es"},
		},
		downloads: []database.DownloadRecord{
			{File: "test1.srt", Provider: "opensubtitles"},
		},
		mediaItems: []database.MediaItem{
			{Path: "test1.mp4", Title: "Test Movie 1"},
		},
		tags: []database.Tag{
			{ID: "1", Name: "action"},
		},
	}
}

func (m *mockSubtitleStore) InsertSubtitle(rec *database.SubtitleRecord) error {
	m.subtitles = append(m.subtitles, *rec)
	return nil
}

func (m *mockSubtitleStore) ListSubtitles() ([]database.SubtitleRecord, error) {
	return m.subtitles, nil
}

func (m *mockSubtitleStore) ListSubtitlesByVideo(video string) ([]database.SubtitleRecord, error) {
	var result []database.SubtitleRecord
	for _, sub := range m.subtitles {
		if sub.VideoFile == video {
			result = append(result, sub)
		}
	}
	return result, nil
}

func (m *mockSubtitleStore) CountSubtitles() (int, error) {
	return len(m.subtitles), nil
}

func (m *mockSubtitleStore) DeleteSubtitle(file string) error {
	var filtered []database.SubtitleRecord
	for _, sub := range m.subtitles {
		if sub.File != file {
			filtered = append(filtered, sub)
		}
	}
	m.subtitles = filtered
	return nil
}

func (m *mockSubtitleStore) InsertDownload(rec *database.DownloadRecord) error {
	m.downloads = append(m.downloads, *rec)
	return nil
}

func (m *mockSubtitleStore) ListDownloads() ([]database.DownloadRecord, error) {
	return m.downloads, nil
}

func (m *mockSubtitleStore) ListDownloadsByVideo(video string) ([]database.DownloadRecord, error) {
	return []database.DownloadRecord{}, nil
}

func (m *mockSubtitleStore) CountDownloads() (int, error) {
	return len(m.downloads), nil
}

func (m *mockSubtitleStore) DeleteDownload(file string) error {
	return nil
}

func (m *mockSubtitleStore) InsertMediaItem(rec *database.MediaItem) error {
	m.mediaItems = append(m.mediaItems, *rec)
	return nil
}

func (m *mockSubtitleStore) ListMediaItems() ([]database.MediaItem, error) {
	return m.mediaItems, nil
}

func (m *mockSubtitleStore) CountMediaItems() (int, error) {
	return len(m.mediaItems), nil
}

func (m *mockSubtitleStore) DeleteMediaItem(path string) error {
	return nil
}

func (m *mockSubtitleStore) GetMediaItem(path string) (*database.MediaItem, error) {
	for i := range m.mediaItems {
		if m.mediaItems[i].Path == path {
			return &m.mediaItems[i], nil
		}
	}
	return nil, nil
}

func (m *mockSubtitleStore) InsertTag(name string) error {
	m.tags = append(m.tags, database.Tag{ID: fmt.Sprintf("%d", len(m.tags)+1), Name: name})
	return nil
}

func (m *mockSubtitleStore) ListTags() ([]database.Tag, error) {
	return m.tags, nil
}

func (m *mockSubtitleStore) DeleteTag(id int64) error                    { return nil }
func (m *mockSubtitleStore) UpdateTag(id int64, name string) error       { return nil }
func (m *mockSubtitleStore) AssignTagToUser(userID, tagID int64) error   { return nil }
func (m *mockSubtitleStore) RemoveTagFromUser(userID, tagID int64) error { return nil }
func (m *mockSubtitleStore) ListTagsForUser(userID int64) ([]database.Tag, error) {
	return []database.Tag{}, nil
}
func (m *mockSubtitleStore) AssignTagToMedia(mediaID, tagID int64) error   { return nil }
func (m *mockSubtitleStore) RemoveTagFromMedia(mediaID, tagID int64) error { return nil }
func (m *mockSubtitleStore) ListTagsForMedia(mediaID int64) ([]database.Tag, error) {
	return []database.Tag{}, nil
}
func (m *mockSubtitleStore) SetMediaReleaseGroup(path, group string) error        { return nil }
func (m *mockSubtitleStore) SetMediaAltTitles(path string, titles []string) error { return nil }
func (m *mockSubtitleStore) SetMediaFieldLocks(path, locks string) error          { return nil }
func (m *mockSubtitleStore) SetMediaTitle(path, title string) error               { return nil }
func (m *mockSubtitleStore) GetMediaReleaseGroup(path string) (string, error)     { return "", nil }
func (m *mockSubtitleStore) GetMediaAltTitles(path string) ([]string, error)      { return []string{}, nil }
func (m *mockSubtitleStore) GetMediaFieldLocks(path string) (string, error)       { return "", nil }
func (m *mockSubtitleStore) Close() error                                         { return nil }
func (m *mockSubtitleStore) CleanupExpiredSessions() error                        { return nil }

func TestService_CreateDatabaseBackup(t *testing.T) {
	store := newMockSubtitleStore()
	config := ServiceConfig{
		BackupPath:        "/tmp/test-backups",
		EnableCompression: true,
		DatabaseStore:     store,
	}

	service, err := NewService(config)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	backup, err := service.CreateDatabaseBackup(ctx)
	if err != nil {
		t.Fatalf("failed to create database backup: %v", err)
	}

	if backup.Type != BackupTypeDatabase {
		t.Errorf("expected backup type %s, got %s", BackupTypeDatabase, backup.Type)
	}

	if !backup.Compressed {
		t.Error("expected backup to be compressed")
	}

	if len(backup.Contents) != 1 || backup.Contents[0] != "database" {
		t.Errorf("expected contents [database], got %v", backup.Contents)
	}
}

func TestService_CreateConfigBackup(t *testing.T) {
	store := newMockSubtitleStore()
	config := ServiceConfig{
		BackupPath:    "/tmp/test-backups",
		DatabaseStore: store,
	}

	service, err := NewService(config)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	backup, err := service.CreateConfigBackup(ctx)
	if err != nil {
		t.Fatalf("failed to create config backup: %v", err)
	}

	if backup.Type != BackupTypeConfiguration {
		t.Errorf("expected backup type %s, got %s", BackupTypeConfiguration, backup.Type)
	}

	if len(backup.Contents) != 1 || backup.Contents[0] != "configuration" {
		t.Errorf("expected contents [configuration], got %v", backup.Contents)
	}
}

func TestService_RestoreBackup(t *testing.T) {
	store := newMockSubtitleStore()

	// Use mock storage to avoid file system issues
	storage := &mockStorage{data: make(map[string][]byte)}
	compression := NewGzipCompression()
	manager := NewBackupManager(storage, compression, nil)

	service := &Service{
		manager:     manager,
		dbBackupper: NewDatabaseBackupper(store),
		logger:      logging.GetLogger("backup"),
	}

	ctx := context.Background()

	// Create a database backup
	backup, err := service.CreateDatabaseBackup(ctx)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Clear the store to simulate empty database
	store.subtitles = []database.SubtitleRecord{}
	store.downloads = []database.DownloadRecord{}
	store.mediaItems = []database.MediaItem{}
	store.tags = []database.Tag{}

	// Restore the backup
	err = service.RestoreBackup(ctx, backup.ID)
	if err != nil {
		t.Fatalf("failed to restore backup: %v", err)
	}

	// Verify data was restored
	if len(store.subtitles) == 0 {
		t.Error("subtitles were not restored")
	}
}

func TestService_RotateBackups(t *testing.T) {
	store := newMockSubtitleStore()

	// Use mock storage to avoid file system issues
	storage := &mockStorage{data: make(map[string][]byte)}
	compression := NewGzipCompression()
	manager := NewBackupManager(storage, compression, nil)

	service := &Service{
		manager:     manager,
		dbBackupper: NewDatabaseBackupper(store),
		logger:      logging.GetLogger("backup"),
	}

	ctx := context.Background()

	// Create multiple backups
	for i := 0; i < 5; i++ {
		_, err := service.CreateDatabaseBackup(ctx)
		if err != nil {
			t.Fatalf("failed to create backup %d: %v", i, err)
		}
	}

	// Check we have 5 backups
	backups := service.ListBackups()
	if len(backups) != 5 {
		t.Errorf("expected 5 backups, got %d", len(backups))
	}

	// Rotate with limit of 3
	err := service.RotateBackups(ctx, 24*365*time.Hour, 3) // 1 year max age, 3 max count
	if err != nil {
		t.Fatalf("failed to rotate backups: %v", err)
	}

	// Check we now have 3 backups
	backups = service.ListBackups()
	if len(backups) != 3 {
		t.Errorf("expected 3 backups after rotation, got %d", len(backups))
	}
}

func (m *mockSubtitleStore) AssignProfileToMedia(mediaID, profileID string) error { return nil }
func (m *mockSubtitleStore) RemoveProfileFromMedia(mediaID string) error          { return nil }
func (m *mockSubtitleStore) GetMediaProfile(mediaID string) (*database.LanguageProfile, error) {
	return &database.LanguageProfile{}, nil
}
func (m *mockSubtitleStore) CreateLanguageProfile(profile *database.LanguageProfile) error {
	return nil
}
func (m *mockSubtitleStore) GetLanguageProfile(id string) (*database.LanguageProfile, error) {
	return &database.LanguageProfile{}, nil
}
func (m *mockSubtitleStore) ListLanguageProfiles() ([]database.LanguageProfile, error) {
	return []database.LanguageProfile{}, nil
}
func (m *mockSubtitleStore) UpdateLanguageProfile(profile *database.LanguageProfile) error {
	return nil
}
func (m *mockSubtitleStore) DeleteLanguageProfile(id string) error     { return nil }
func (m *mockSubtitleStore) SetDefaultLanguageProfile(id string) error { return nil }
func (m *mockSubtitleStore) GetDefaultLanguageProfile() (*database.LanguageProfile, error) {
	return &database.LanguageProfile{}, nil
}
func (m *mockSubtitleStore) InsertSubtitleSource(src *database.SubtitleSource) error { return nil }
func (m *mockSubtitleStore) GetSubtitleSource(sourceHash string) (*database.SubtitleSource, error) {
	return &database.SubtitleSource{}, nil
}
func (m *mockSubtitleStore) UpdateSubtitleSourceStats(sourceHash string, downloadCount, successCount int, avgRating *float64) error {
	return nil
}
func (m *mockSubtitleStore) ListSubtitleSources(provider string, limit int) ([]database.SubtitleSource, error) {
	return []database.SubtitleSource{}, nil
}
func (m *mockSubtitleStore) DeleteSubtitleSource(sourceHash string) error          { return nil }
func (m *mockSubtitleStore) InsertMonitoredItem(rec *database.MonitoredItem) error { return nil }
func (m *mockSubtitleStore) ListMonitoredItems() ([]database.MonitoredItem, error) {
	return []database.MonitoredItem{}, nil
}
func (m *mockSubtitleStore) UpdateMonitoredItem(rec *database.MonitoredItem) error { return nil }
func (m *mockSubtitleStore) DeleteMonitoredItem(id string) error                   { return nil }
func (m *mockSubtitleStore) GetMonitoredItemsToCheck(interval time.Duration) ([]database.MonitoredItem, error) {
	return []database.MonitoredItem{}, nil
}

// User authentication and session management methods
func (m *mockSubtitleStore) CreateUser(username, passwordHash, email, role string) (string, error) {
	return "user-id", nil
}
func (m *mockSubtitleStore) GetUserByUsername(username string) (*common.User, error) {
	return &common.User{}, nil
}
func (m *mockSubtitleStore) GetUserByEmail(email string) (*common.User, error) {
	return &common.User{}, nil
}
func (m *mockSubtitleStore) GetUserByID(id string) (*common.User, error) {
	return &common.User{}, nil
}
func (m *mockSubtitleStore) ListUsers() ([]*common.User, error) {
	return []*common.User{}, nil
}
func (m *mockSubtitleStore) UpdateUserRole(username, role string) error           { return nil }
func (m *mockSubtitleStore) UpdateUserPassword(userID, passwordHash string) error { return nil }
func (m *mockSubtitleStore) CreateSession(userID, token string, duration time.Duration) error {
	return nil
}
func (m *mockSubtitleStore) ValidateSession(token string) (string, error) { return "", nil }
func (m *mockSubtitleStore) InvalidateSession(token string) error         { return nil }
func (m *mockSubtitleStore) InvalidateUserSessions(userID string) error   { return nil }
func (m *mockSubtitleStore) CreateAPIKey(userID, key string) error        { return nil }
func (m *mockSubtitleStore) ValidateAPIKey(key string) (string, error)    { return "", nil }
func (m *mockSubtitleStore) CreateOneTimeToken(userID, token string, duration time.Duration) error {
	return nil
}
func (m *mockSubtitleStore) ValidateOneTimeToken(token string) (string, error) { return "", nil }
func (m *mockSubtitleStore) SetDashboardLayout(userID, layout string) error    { return nil }
func (m *mockSubtitleStore) GetDashboardLayout(userID string) (string, error)  { return "", nil }
