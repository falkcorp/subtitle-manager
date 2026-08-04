// file: pkg/database/store.go
// version: 1.0.0
// guid: db0c372c-d86a-47fd-ae26-7685d0d4a60f
// last-edited: 2026-08-04

package database

import (
	"time"

	common "buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go/commonpb/v2"
	"github.com/jdfalk/subtitle-manager/pkg/profiles"
)

type LanguageProfile = profiles.LanguageProfile

type LanguageProfileAssignment = profiles.MediaProfileAssignment

// SubtitleStore abstracts subtitle history storage backends.
type SubtitleStore interface {
	// InsertSubtitle stores a subtitle record.
	InsertSubtitle(rec *SubtitleRecord) error
	// ListSubtitles retrieves all subtitle records sorted by creation time.
	ListSubtitles() ([]SubtitleRecord, error)
	// ListSubtitlesByVideo returns subtitle records for a video file.
	ListSubtitlesByVideo(video string) ([]SubtitleRecord, error)
	// CountSubtitles returns the total number of subtitle records.
	CountSubtitles() (int, error)
	// DeleteSubtitle removes all records for the specified file.
	DeleteSubtitle(file string) error
	// InsertDownload stores a download record.
	InsertDownload(rec *DownloadRecord) error
	// ListDownloads retrieves all download records sorted by creation time.
	ListDownloads() ([]DownloadRecord, error)
	// ListDownloadsByVideo returns download records for a video file.
	ListDownloadsByVideo(video string) ([]DownloadRecord, error)
	// CountDownloads returns the total number of download records.
	CountDownloads() (int, error)
	// DeleteDownload removes download records for the specified subtitle file.
	DeleteDownload(file string) error
	// InsertMediaItem stores a media library record.
	InsertMediaItem(rec *MediaItem) error
	// ListMediaItems retrieves all media items sorted by creation time.
	ListMediaItems() ([]MediaItem, error)
	// CountMediaItems returns the total number of media items.
	CountMediaItems() (int, error)
	// DeleteMediaItem removes a record for the specified media path.
	DeleteMediaItem(path string) error
	// GetMediaItem retrieves a media item by path. Returns nil if not found.
	GetMediaItem(path string) (*MediaItem, error)
	// InsertTag stores a new tag value.
	InsertTag(name string) error
	// ListTags retrieves all tags.
	ListTags() ([]Tag, error)
	// DeleteTag removes a tag by ID.
	DeleteTag(id int64) error
	// UpdateTag renames a tag by ID.
	UpdateTag(id int64, name string) error
	// AssignTagToUser associates a tag with a user.
	AssignTagToUser(userID, tagID int64) error
	// RemoveTagFromUser disassociates a tag from a user.
	RemoveTagFromUser(userID, tagID int64) error
	// ListTagsForUser returns tags associated with a user.
	ListTagsForUser(userID int64) ([]Tag, error)
	// AssignTagToMedia associates a tag with a media item.
	AssignTagToMedia(mediaID, tagID int64) error
	// RemoveTagFromMedia disassociates a tag from a media item.
	RemoveTagFromMedia(mediaID, tagID int64) error
	// ListTagsForMedia returns tags associated with a media item.
	ListTagsForMedia(mediaID int64) ([]Tag, error)
	// SetMediaReleaseGroup sets the release group for a media item.
	SetMediaReleaseGroup(path, group string) error
	// SetMediaAltTitles stores alternate titles for a media item.
	SetMediaAltTitles(path string, titles []string) error
	// SetMediaFieldLocks updates locked fields for a media item.
	SetMediaFieldLocks(path, locks string) error
	// GetMediaReleaseGroup retrieves the release group for a media item.
	GetMediaReleaseGroup(path string) (string, error)
	// GetMediaAltTitles retrieves alternate titles for a media item.
	GetMediaAltTitles(path string) ([]string, error)
	// GetMediaFieldLocks retrieves locked fields for a media item.
	GetMediaFieldLocks(path string) (string, error)
	// SetMediaTitle updates the title for a media item.
	SetMediaTitle(path, title string) error
	// CreateLanguageProfile stores a new language profile.
	CreateLanguageProfile(profile *LanguageProfile) error
	// GetLanguageProfile retrieves a language profile by ID.
	GetLanguageProfile(id string) (*LanguageProfile, error)
	// ListLanguageProfiles retrieves all language profiles.
	ListLanguageProfiles() ([]LanguageProfile, error)
	// UpdateLanguageProfile updates an existing language profile.
	UpdateLanguageProfile(profile *LanguageProfile) error
	// DeleteLanguageProfile removes a language profile by ID.
	DeleteLanguageProfile(id string) error
	// SetDefaultLanguageProfile marks a profile as the default.
	SetDefaultLanguageProfile(id string) error
	// GetDefaultLanguageProfile retrieves the default language profile.
	GetDefaultLanguageProfile() (*LanguageProfile, error)
	// AssignProfileToMedia assigns a language profile to a media item.
	AssignProfileToMedia(mediaID, profileID string) error
	// RemoveProfileFromMedia removes language profile assignment from a media item.
	RemoveProfileFromMedia(mediaID string) error
	// GetMediaProfile retrieves the language profile assigned to a media item.
	GetMediaProfile(mediaID string) (*LanguageProfile, error)
	// GetAssignedProfileID reports which profile was explicitly assigned to a
	// media item, returning "" when the item has no assignment of its own.
	//
	// This exists because GetMediaProfile falls back to the default profile on a
	// miss, so it can never answer "was anything assigned here?". That is the
	// right answer for "which profile governs this file" and the wrong one for
	// deciding whether to change how a scan behaves — and it makes a file
	// explicitly assigned the default indistinguishable from an unassigned one.
	//
	// Unlike GetMediaProfile this never falls back and never creates rows.
	GetAssignedProfileID(mediaID string) (string, error)
	// Subtitle source tracking operations
	InsertSubtitleSource(src *SubtitleSource) error
	GetSubtitleSource(sourceHash string) (*SubtitleSource, error)
	UpdateSubtitleSourceStats(sourceHash string, downloadCount, successCount int, avgRating *float64) error
	ListSubtitleSources(provider string, limit int) ([]SubtitleSource, error)
	DeleteSubtitleSource(sourceHash string) error
	// Monitoring methods
	// InsertMonitoredItem stores a monitored item record.
	InsertMonitoredItem(rec *MonitoredItem) error
	// ListMonitoredItems retrieves all monitored items.
	ListMonitoredItems() ([]MonitoredItem, error)
	// UpdateMonitoredItem updates an existing monitored item.
	UpdateMonitoredItem(rec *MonitoredItem) error
	// DeleteMonitoredItem removes a monitored item by ID.
	DeleteMonitoredItem(id string) error
	// GetMonitoredItemsToCheck returns items that need monitoring.
	GetMonitoredItemsToCheck(interval time.Duration) ([]MonitoredItem, error)

	// User authentication and session management
	// CreateUser creates a new user with hashed password and returns user ID.
	CreateUser(username, passwordHash, email, role string) (string, error)
	// GetUserByUsername retrieves a user by username.
	GetUserByUsername(username string) (*common.User, error)
	// GetUserByEmail retrieves a user by email.
	GetUserByEmail(email string) (*common.User, error)
	// GetUserByID retrieves a user by ID.
	GetUserByID(id string) (*common.User, error)
	// ListUsers retrieves all users.
	ListUsers() ([]*common.User, error)
	// UpdateUserRole updates a user's role.
	UpdateUserRole(username, role string) error
	// UpdateUserPassword updates a user's password hash.
	UpdateUserPassword(userID, passwordHash string) error
	// CreateSession creates a new user session.
	CreateSession(userID, token string, duration time.Duration) error
	// ValidateSession validates a session token and returns user ID.
	ValidateSession(token string) (string, error)
	// InvalidateSession invalidates a specific session.
	InvalidateSession(token string) error
	// InvalidateUserSessions invalidates all sessions for a user.
	InvalidateUserSessions(userID string) error
	// CleanupExpiredSessions removes expired sessions.
	CleanupExpiredSessions() error
	// CreateAPIKey creates an API key for a user.
	CreateAPIKey(userID, key string) error
	// ValidateAPIKey validates an API key and returns user ID.
	ValidateAPIKey(key string) (string, error)
	// CreateOneTimeToken creates a one-time use token.
	CreateOneTimeToken(userID, token string, duration time.Duration) error
	// ValidateOneTimeToken validates a one-time token and returns user ID.
	ValidateOneTimeToken(token string) (string, error)
	// SetDashboardLayout sets dashboard layout preferences for a user.
	SetDashboardLayout(userID, layout string) error
	// GetDashboardLayout gets dashboard layout preferences for a user.
	GetDashboardLayout(userID string) (string, error)

	// Close releases any resources held by the store.
	Close() error
}
