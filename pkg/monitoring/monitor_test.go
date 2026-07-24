// file: pkg/monitoring/monitor_test.go
// version: 1.0.1
// guid: 12345678-1234-1234-1234-123456789015

package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	gcommon "buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go/commonpb/v2"
	"github.com/jdfalk/subtitle-manager/pkg/database"
)

// MockSubtitleStore implements database.SubtitleStore for testing
type MockSubtitleStore struct {
	mock.Mock
}

// Implement all required methods for database.SubtitleStore interface
func (m *MockSubtitleStore) InsertSubtitle(rec *database.SubtitleRecord) error {
	args := m.Called(rec)
	return args.Error(0)
}

func (m *MockSubtitleStore) ListSubtitles() ([]database.SubtitleRecord, error) {
	args := m.Called()
	return args.Get(0).([]database.SubtitleRecord), args.Error(1)
}

func (m *MockSubtitleStore) ListSubtitlesByVideo(video string) ([]database.SubtitleRecord, error) {
	args := m.Called(video)
	return args.Get(0).([]database.SubtitleRecord), args.Error(1)
}

func (m *MockSubtitleStore) CountSubtitles() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *MockSubtitleStore) DeleteSubtitle(file string) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *MockSubtitleStore) InsertDownload(rec *database.DownloadRecord) error {
	args := m.Called(rec)
	return args.Error(0)
}

func (m *MockSubtitleStore) ListDownloads() ([]database.DownloadRecord, error) {
	args := m.Called()
	return args.Get(0).([]database.DownloadRecord), args.Error(1)
}

func (m *MockSubtitleStore) ListDownloadsByVideo(video string) ([]database.DownloadRecord, error) {
	args := m.Called(video)
	return args.Get(0).([]database.DownloadRecord), args.Error(1)
}

func (m *MockSubtitleStore) CountDownloads() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *MockSubtitleStore) DeleteDownload(file string) error {
	args := m.Called(file)
	return args.Error(0)
}

func (m *MockSubtitleStore) InsertMediaItem(rec *database.MediaItem) error {
	args := m.Called(rec)
	return args.Error(0)
}

func (m *MockSubtitleStore) ListMediaItems() ([]database.MediaItem, error) {
	args := m.Called()
	return args.Get(0).([]database.MediaItem), args.Error(1)
}

func (m *MockSubtitleStore) CountMediaItems() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

func (m *MockSubtitleStore) DeleteMediaItem(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockSubtitleStore) GetMediaItem(path string) (*database.MediaItem, error) {
	args := m.Called(path)
	if rec, ok := args.Get(0).(*database.MediaItem); ok {
		return rec, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSubtitleStore) InsertTag(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockSubtitleStore) ListTags() ([]database.Tag, error) {
	args := m.Called()
	return args.Get(0).([]database.Tag), args.Error(1)
}

func (m *MockSubtitleStore) DeleteTag(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockSubtitleStore) UpdateTag(id int64, name string) error {
	args := m.Called(id, name)
	return args.Error(0)
}

func (m *MockSubtitleStore) AssignTagToUser(userID, tagID int64) error {
	args := m.Called(userID, tagID)
	return args.Error(0)
}

func (m *MockSubtitleStore) RemoveTagFromUser(userID, tagID int64) error {
	args := m.Called(userID, tagID)
	return args.Error(0)
}

func (m *MockSubtitleStore) ListTagsForUser(userID int64) ([]database.Tag, error) {
	args := m.Called(userID)
	return args.Get(0).([]database.Tag), args.Error(1)
}

func (m *MockSubtitleStore) AssignTagToMedia(mediaID, tagID int64) error {
	args := m.Called(mediaID, tagID)
	return args.Error(0)
}

func (m *MockSubtitleStore) RemoveTagFromMedia(mediaID, tagID int64) error {
	args := m.Called(mediaID, tagID)
	return args.Error(0)
}

func (m *MockSubtitleStore) ListTagsForMedia(mediaID int64) ([]database.Tag, error) {
	args := m.Called(mediaID)
	return args.Get(0).([]database.Tag), args.Error(1)
}

func (m *MockSubtitleStore) SetMediaReleaseGroup(path, group string) error {
	args := m.Called(path, group)
	return args.Error(0)
}

func (m *MockSubtitleStore) SetMediaAltTitles(path string, titles []string) error {
	args := m.Called(path, titles)
	return args.Error(0)
}

func (m *MockSubtitleStore) SetMediaFieldLocks(path, locks string) error {
	args := m.Called(path, locks)
	return args.Error(0)
}

func (m *MockSubtitleStore) SetMediaTitle(path, title string) error {
	args := m.Called(path, title)
	return args.Error(0)
}
func (m *MockSubtitleStore) GetMediaReleaseGroup(path string) (string, error) { return "", nil }
func (m *MockSubtitleStore) GetMediaAltTitles(path string) ([]string, error)  { return []string{}, nil }
func (m *MockSubtitleStore) GetMediaFieldLocks(path string) (string, error)   { return "", nil }

func (m *MockSubtitleStore) InsertMonitoredItem(rec *database.MonitoredItem) error {
	args := m.Called(rec)
	return args.Error(0)
}

func (m *MockSubtitleStore) ListMonitoredItems() ([]database.MonitoredItem, error) {
	args := m.Called()
	return args.Get(0).([]database.MonitoredItem), args.Error(1)
}

func (m *MockSubtitleStore) UpdateMonitoredItem(rec *database.MonitoredItem) error {
	args := m.Called(rec)
	return args.Error(0)
}

func (m *MockSubtitleStore) DeleteMonitoredItem(id string) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockSubtitleStore) GetMonitoredItemsToCheck(interval time.Duration) ([]database.MonitoredItem, error) {
	args := m.Called(interval)
	return args.Get(0).([]database.MonitoredItem), args.Error(1)
}

func (m *MockSubtitleStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Add missing method from database.SubtitleStore interface to satisfy tests
func (m *MockSubtitleStore) CleanupExpiredSessions() error { return nil }

// Auth and session related methods (stubs) to satisfy the SubtitleStore interface
func (m *MockSubtitleStore) CreateUser(username, passwordHash, email, role string) (string, error) {
	return "", nil
}
func (m *MockSubtitleStore) GetUserByUsername(username string) (*gcommon.User, error) {
	return nil, nil
}
func (m *MockSubtitleStore) GetUserByEmail(email string) (*gcommon.User, error)   { return nil, nil }
func (m *MockSubtitleStore) GetUserByID(id string) (*gcommon.User, error)         { return nil, nil }
func (m *MockSubtitleStore) ListUsers() ([]*gcommon.User, error)                  { return nil, nil }
func (m *MockSubtitleStore) UpdateUserRole(username, role string) error           { return nil }
func (m *MockSubtitleStore) UpdateUserPassword(userID, passwordHash string) error { return nil }
func (m *MockSubtitleStore) CreateSession(userID, token string, duration time.Duration) error {
	return nil
}
func (m *MockSubtitleStore) ValidateSession(token string) (string, error) { return "", nil }
func (m *MockSubtitleStore) InvalidateSession(token string) error         { return nil }
func (m *MockSubtitleStore) InvalidateUserSessions(userID string) error   { return nil }
func (m *MockSubtitleStore) CreateAPIKey(userID, key string) error        { return nil }
func (m *MockSubtitleStore) ValidateAPIKey(key string) (string, error)    { return "", nil }
func (m *MockSubtitleStore) CreateOneTimeToken(userID, token string, duration time.Duration) error {
	return nil
}
func (m *MockSubtitleStore) ValidateOneTimeToken(token string) (string, error) { return "", nil }
func (m *MockSubtitleStore) SetDashboardLayout(userID, layout string) error    { return nil }
func (m *MockSubtitleStore) GetDashboardLayout(userID string) (string, error)  { return "", nil }

func TestEpisodeMonitor_NewEpisodeMonitor(t *testing.T) {
	store := &MockSubtitleStore{}

	monitor := NewEpisodeMonitor(
		time.Hour,
		nil, // sonarr client
		nil, // radarr client
		store,
		3,    // maxRetries
		true, // qualityCheck
	)

	assert.NotNil(t, monitor)
	assert.Equal(t, time.Hour, monitor.interval)
	assert.Equal(t, 3, monitor.maxRetries)
	assert.True(t, monitor.qualityCheck)
}

func TestEpisodeMonitor_GetMonitoringStats(t *testing.T) {
	store := &MockSubtitleStore{}

	// Mock data
	mockItems := []database.MonitoredItem{
		{ID: "1", Status: "pending"},
		{ID: "2", Status: "monitoring"},
		{ID: "3", Status: "found"},
		{ID: "4", Status: "failed"},
		{ID: "5", Status: "blacklisted"},
	}

	store.On("ListMonitoredItems").Return(mockItems, nil)

	monitor := NewEpisodeMonitor(
		time.Hour,
		nil, nil,
		store,
		3, true,
	)

	stats, err := monitor.GetMonitoringStats()

	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 5, stats.Total)
	assert.Equal(t, 1, stats.Pending)
	assert.Equal(t, 1, stats.Monitoring)
	assert.Equal(t, 1, stats.Found)
	assert.Equal(t, 1, stats.Failed)
	assert.Equal(t, 1, stats.Blacklisted)

	store.AssertExpectations(t)
}

func TestEpisodeMonitor_CheckForSubtitles(t *testing.T) {
	store := &MockSubtitleStore{}

	// Mock empty items to check
	store.On("GetMonitoredItemsToCheck", mock.Anything).Return([]database.MonitoredItem{}, nil)

	monitor := NewEpisodeMonitor(
		time.Minute,
		nil, nil,
		store,
		3, true,
	)

	err := monitor.checkForSubtitles(context.Background())

	assert.NoError(t, err)
	store.AssertExpectations(t)
}

func (m *MockSubtitleStore) AssignProfileToMedia(mediaID, profileID string) error { return nil }
func (m *MockSubtitleStore) RemoveProfileFromMedia(mediaID string) error          { return nil }
func (m *MockSubtitleStore) GetMediaProfile(mediaID string) (*database.LanguageProfile, error) {
	return &database.LanguageProfile{}, nil
}
func (m *MockSubtitleStore) CreateLanguageProfile(profile *database.LanguageProfile) error {
	return nil
}
func (m *MockSubtitleStore) GetLanguageProfile(id string) (*database.LanguageProfile, error) {
	return &database.LanguageProfile{}, nil
}
func (m *MockSubtitleStore) ListLanguageProfiles() ([]database.LanguageProfile, error) {
	return []database.LanguageProfile{}, nil
}
func (m *MockSubtitleStore) UpdateLanguageProfile(profile *database.LanguageProfile) error {
	return nil
}
func (m *MockSubtitleStore) DeleteLanguageProfile(id string) error     { return nil }
func (m *MockSubtitleStore) SetDefaultLanguageProfile(id string) error { return nil }
func (m *MockSubtitleStore) GetDefaultLanguageProfile() (*database.LanguageProfile, error) {
	return &database.LanguageProfile{}, nil
}
func (m *MockSubtitleStore) InsertSubtitleSource(src *database.SubtitleSource) error { return nil }
func (m *MockSubtitleStore) GetSubtitleSource(sourceHash string) (*database.SubtitleSource, error) {
	return &database.SubtitleSource{}, nil
}
func (m *MockSubtitleStore) UpdateSubtitleSourceStats(sourceHash string, downloadCount, successCount int, avgRating *float64) error {
	return nil
}
func (m *MockSubtitleStore) ListSubtitleSources(provider string, limit int) ([]database.SubtitleSource, error) {
	return []database.SubtitleSource{}, nil
}
func (m *MockSubtitleStore) DeleteSubtitleSource(sourceHash string) error { return nil }
