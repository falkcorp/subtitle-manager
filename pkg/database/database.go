// file: pkg/database/database.go
// version: 1.0.0
// guid: 6e68e15d-7acf-4121-94f7-b29e4650650d
// last-edited: 2026-08-04

// Package database provides data storage and retrieval functionality for subtitle-manager.
// It supports SQLite and other database backends for managing subtitle metadata and operations.
//
// This package is the core data layer for subtitle-manager, handling persistence and queries.
// It defines the data models, database schema, and CRUD operations for subtitles, media items,
// downloads, and user data. It also manages tag-based organization and retrieval of entities.
//
// Usage:
//
//	import (
//	    "github.com/yourusername/subtitle-manager/database"
//	)
//
//	func main() {
//	    // Open the database
//	    db, err := database.Open("path/to/your/database/file")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    defer db.Close()
//
//	    // Insert a new subtitle record
//	    err = db.InsertSubtitle(&database.SubtitleRecord{
//	        File:      "example.srt",
//	        VideoFile: "example.mp4",
//	        Release:   "Example Release",
//	        Language:  "en",
//	        Service:   "subscene",
//	        Embedded:  false,
//	    })
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//
//	    // List all subtitles
//	    subs, err := db.ListSubtitles()
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    for _, sub := range subs {
//	        fmt.Println(sub.File, sub.Language)
//	    }
//	}
package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	common "buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go/commonpb/v2"
	"github.com/spf13/viper"
)

// GetDatabasePath returns the full database path based on the backend type.
// For SQLite, it combines db_path with sqlite3_filename.
// For other backends (pebble, postgres), it returns db_path as-is.
func GetDatabasePath() string {
	backend := viper.GetString("db_backend")
	dbPath := viper.GetString("db_path")

	if backend == "sqlite" {
		filename := viper.GetString("sqlite3_filename")
		return filepath.Join(dbPath, filename)
	}

	return dbPath
}

// GetDatabaseBackend returns the configured database backend
func GetDatabaseBackend() string {
	return viper.GetString("db_backend")
}

// SubtitleRecord represents a subtitle file that has been processed.
// VideoFile is the path to the media file the subtitle belongs to.
// Release denotes the original release name including group when known.
// Enhanced with source tracking and relationship metadata.
type SubtitleRecord struct {
	ID               string
	File             string
	VideoFile        string
	Release          string
	Language         string
	Service          string
	Embedded         bool
	SourceURL        string   // Original download URL
	ProviderMetadata string   // JSON metadata from provider
	ConfidenceScore  *float64 // Quality/match confidence (0-1)
	ParentID         *string  // Parent subtitle ID for tracking modifications
	ModificationType string   // sync, translate, manual_edit, etc.
	CreatedAt        time.Time
}

// DownloadRecord represents a downloaded subtitle file.
// File is the path to the subtitle file stored on disk.
// VideoFile is the media file the subtitle corresponds to.
// Enhanced with search and performance tracking metadata.
type DownloadRecord struct {
	ID               string
	File             string
	VideoFile        string
	Provider         string
	Language         string
	SearchQuery      string   // Original search query used
	MatchScore       *float64 // How well the result matched (0-1)
	DownloadAttempts int      // Number of download attempts
	ErrorMessage     string   // Last error message if failed
	ResponseTimeMs   *int     // Provider response time in milliseconds
	CreatedAt        time.Time
}

// SubtitleSource represents metadata about subtitle sources and provider performance.
// Tracks where subtitles come from and how well providers perform over time.
type SubtitleSource struct {
	ID            string
	SourceHash    string    // Unique hash of the subtitle content/source
	OriginalURL   string    // Original download URL
	Provider      string    // Provider name
	Title         string    // Subtitle title from provider
	ReleaseInfo   string    // Release information
	FileSize      *int      // File size in bytes
	DownloadCount int       // Total download attempts
	SuccessCount  int       // Successful downloads
	AvgRating     *float64  // Average user rating (0-5)
	LastSeen      time.Time // Last time this source was seen
	Metadata      string    // JSON metadata from provider
	CreatedAt     time.Time
}

// MediaItem represents a video file discovered in the library.
// Path is the absolute location on disk. Title is the parsed show or movie name.
// Season and Episode provide optional numbering for TV episodes.
type MediaItem struct {
	ID           string
	Path         string
	Title        string
	Season       int
	Episode      int
	ReleaseGroup string
	AltTitles    string
	FieldLocks   string
	CreatedAt    time.Time
}

// Tag represents a universal tag that can be associated with any entity type.
// Supports media, users, providers, language profiles, and media profiles.
// Type indicates the tag classification (system, user, custom).
// EntityType specifies which types of entities this tag can be applied to.
// Color and Description provide additional metadata for UI display and organization.
type Tag struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`        // 'system', 'user', 'custom'
	EntityType  string    `json:"entity_type"` // 'media', 'language', 'movie', 'series', 'provider', 'user', 'all'
	Color       string    `json:"color"`       // hex color for UI display (optional)
	Description string    `json:"description"` // optional tag description
	CreatedAt   time.Time `json:"created_at"`
}

// TagAssociation represents a relationship between a tag and any entity.
// Uses polymorphic association pattern with EntityType and EntityID.
type TagAssociation struct {
	TagID      string    `json:"tag_id"`
	EntityType string    `json:"entity_type"` // 'movie', 'series', 'user', 'provider', etc.
	EntityID   string    `json:"entity_id"`   // polymorphic entity identifier
	CreatedAt  time.Time `json:"created_at"`
}

// TaggedEntity interface defines methods that all taggable entities must implement.
type TaggedEntity interface {
	GetEntityType() string
	GetEntityID() string
	GetTags() []string
	SetTags([]string)
}

// MonitoredItem represents a media item being monitored for subtitle availability.
type MonitoredItem struct {
	ID          string    `json:"id"`
	MediaID     string    `json:"media_id"`
	Path        string    `json:"path"`
	Languages   string    `json:"languages"` // JSON array of language codes
	LastChecked time.Time `json:"last_checked"`
	Status      string    `json:"status"` // pending, monitoring, found, failed, blacklisted
	RetryCount  int       `json:"retry_count"`
	MaxRetries  int       `json:"max_retries"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SQLStore implements SubtitleStore using an SQLite database.
type SQLStore struct {
	db *sql.DB
}

// InsertDownload stores a download record.
func (s *SQLStore) InsertDownload(rec *DownloadRecord) error {
	_, err := s.db.Exec(`INSERT INTO downloads (file, video_file, provider, language, search_query, match_score, download_attempts, error_message, response_time_ms, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.File, rec.VideoFile, rec.Provider, rec.Language, rec.SearchQuery, rec.MatchScore, rec.DownloadAttempts, rec.ErrorMessage, rec.ResponseTimeMs, time.Now())
	return err
}

// ListDownloads retrieves download records ordered by most recent.
func (s *SQLStore) ListDownloads() ([]DownloadRecord, error) {
	rows, err := s.db.Query(`SELECT id, file, video_file, provider, language, search_query, match_score, download_attempts, error_message, response_time_ms, created_at FROM downloads ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []DownloadRecord
	for rows.Next() {
		var r DownloadRecord
		var id int64
		var searchQuery, errorMessage sql.NullString
		var matchScore sql.NullFloat64
		var responseTimeMs sql.NullInt64
		if err := rows.Scan(&id, &r.File, &r.VideoFile, &r.Provider, &r.Language, &searchQuery, &matchScore, &r.DownloadAttempts, &errorMessage, &responseTimeMs, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		if searchQuery.Valid {
			r.SearchQuery = searchQuery.String
		}
		if matchScore.Valid {
			r.MatchScore = &matchScore.Float64
		}
		if errorMessage.Valid {
			r.ErrorMessage = errorMessage.String
		}
		if responseTimeMs.Valid {
			respTime := int(responseTimeMs.Int64)
			r.ResponseTimeMs = &respTime
		}
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// ListDownloadsByVideo retrieves download history for a specific video file.
func (s *SQLStore) ListDownloadsByVideo(video string) ([]DownloadRecord, error) {
	rows, err := s.db.Query(`SELECT id, file, video_file, provider, language, search_query, match_score, download_attempts, error_message, response_time_ms, created_at FROM downloads WHERE video_file = ? ORDER BY id DESC`, video)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []DownloadRecord
	for rows.Next() {
		var r DownloadRecord
		var id int64
		var searchQuery, errorMessage sql.NullString
		var matchScore sql.NullFloat64
		var responseTimeMs sql.NullInt64
		if err := rows.Scan(&id, &r.File, &r.VideoFile, &r.Provider, &r.Language, &searchQuery, &matchScore, &r.DownloadAttempts, &errorMessage, &responseTimeMs, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		if searchQuery.Valid {
			r.SearchQuery = searchQuery.String
		}
		if matchScore.Valid {
			r.MatchScore = &matchScore.Float64
		}
		if errorMessage.Valid {
			r.ErrorMessage = errorMessage.String
		}
		if responseTimeMs.Valid {
			respTime := int(responseTimeMs.Int64)
			r.ResponseTimeMs = &respTime
		}
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// DeleteDownload removes download records matching file from the database.
func (s *SQLStore) DeleteDownload(file string) error {
	_, err := s.db.Exec(`DELETE FROM downloads WHERE file = ?`, file)
	return err
}

// (Removed duplicate InsertMediaItem method - keeping the version with created_at)

// (Removed duplicate ListMediaItems method - keeping the version with created_at)

// (Removed duplicate DeleteMediaItem method - keeping the version with created_at)

// InsertDownload stores a download record using a raw *sql.DB.
func InsertDownload(db *sql.DB, file, video, provider, lang string) error {
	_, err := db.Exec(`INSERT INTO downloads (file, video_file, provider, language, search_query, match_score, download_attempts, error_message, response_time_ms, created_at) VALUES (?, ?, ?, ?, '', NULL, 1, '', NULL, ?)`,
		file, video, provider, lang, time.Now())
	return err
}

// ListDownloads retrieves download records using a raw *sql.DB.
func ListDownloads(db *sql.DB) ([]DownloadRecord, error) {
	rows, err := db.Query(`SELECT id, file, video_file, provider, language, search_query, match_score, download_attempts, error_message, response_time_ms, created_at FROM downloads ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []DownloadRecord
	for rows.Next() {
		var r DownloadRecord
		var id int64
		var searchQuery, errorMessage sql.NullString
		var matchScore sql.NullFloat64
		var responseTimeMs sql.NullInt64
		if err := rows.Scan(&id, &r.File, &r.VideoFile, &r.Provider, &r.Language, &searchQuery, &matchScore, &r.DownloadAttempts, &errorMessage, &responseTimeMs, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		if searchQuery.Valid {
			r.SearchQuery = searchQuery.String
		}
		if matchScore.Valid {
			r.MatchScore = &matchScore.Float64
		}
		if errorMessage.Valid {
			r.ErrorMessage = errorMessage.String
		}
		if responseTimeMs.Valid {
			respTime := int(responseTimeMs.Int64)
			r.ResponseTimeMs = &respTime
		}
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// ListDownloadsByVideo retrieves download history for a specific video file using a raw *sql.DB.
func ListDownloadsByVideo(db *sql.DB, video string) ([]DownloadRecord, error) {
	rows, err := db.Query(`SELECT id, file, video_file, provider, language, search_query, match_score, download_attempts, error_message, response_time_ms, created_at FROM downloads WHERE video_file = ? ORDER BY id DESC`, video)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []DownloadRecord
	for rows.Next() {
		var r DownloadRecord
		var id int64
		var searchQuery, errorMessage sql.NullString
		var matchScore sql.NullFloat64
		var responseTimeMs sql.NullInt64
		if err := rows.Scan(&id, &r.File, &r.VideoFile, &r.Provider, &r.Language, &searchQuery, &matchScore, &r.DownloadAttempts, &errorMessage, &responseTimeMs, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		if searchQuery.Valid {
			r.SearchQuery = searchQuery.String
		}
		if matchScore.Valid {
			r.MatchScore = &matchScore.Float64
		}
		if errorMessage.Valid {
			r.ErrorMessage = errorMessage.String
		}
		if responseTimeMs.Valid {
			respTime := int(responseTimeMs.Int64)
			r.ResponseTimeMs = &respTime
		}
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// DeleteDownload removes download records matching file using a raw *sql.DB.
func DeleteDownload(db *sql.DB, file string) error {
	_, err := db.Exec(`DELETE FROM downloads WHERE file = ?`, file)
	return err
}

// InsertMediaItem stores a media library record.
func (s *SQLStore) InsertMediaItem(rec *MediaItem) error {
	_, err := s.db.Exec(`INSERT INTO media_items (path, title, season, episode, release_group, alt_titles, field_locks, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.Path, rec.Title, rec.Season, rec.Episode, rec.ReleaseGroup, rec.AltTitles, rec.FieldLocks, time.Now())
	return err
}

// ListMediaItems retrieves all media items sorted by creation time.
func (s *SQLStore) ListMediaItems() ([]MediaItem, error) {
	rows, err := s.db.Query(`SELECT id, path, title, season, episode, release_group, alt_titles, field_locks, created_at FROM media_items ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []MediaItem
	for rows.Next() {
		var r MediaItem
		var id int64
		if err := rows.Scan(&id, &r.Path, &r.Title, &r.Season, &r.Episode, &r.ReleaseGroup, &r.AltTitles, &r.FieldLocks, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// DeleteMediaItem removes a media item matching path.
func (s *SQLStore) DeleteMediaItem(path string) error {
	_, err := s.db.Exec(`DELETE FROM media_items WHERE path = ?`, path)
	return err
}

// GetMediaItem retrieves a media item by path. Returns nil if not found.
func (s *SQLStore) GetMediaItem(path string) (*MediaItem, error) {
	row := s.db.QueryRow(`SELECT id, path, title, season, episode, release_group, alt_titles, field_locks, created_at FROM media_items WHERE path = ?`, path)
	var it MediaItem
	var id int64
	if err := row.Scan(&id, &it.Path, &it.Title, &it.Season, &it.Episode, &it.ReleaseGroup, &it.AltTitles, &it.FieldLocks, &it.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	it.ID = strconv.FormatInt(id, 10)
	return &it, nil
}

// CountSubtitles returns the number of subtitle records.
func (s *SQLStore) CountSubtitles() (int, error) {
	row := s.db.QueryRow(`SELECT COUNT(*) FROM subtitles`)
	var n int
	err := row.Scan(&n)
	return n, err
}

// CountDownloads returns the number of download records.
func (s *SQLStore) CountDownloads() (int, error) {
	row := s.db.QueryRow(`SELECT COUNT(*) FROM downloads`)
	var n int
	err := row.Scan(&n)
	return n, err
}

// CountMediaItems returns the number of media items.
func (s *SQLStore) CountMediaItems() (int, error) {
	row := s.db.QueryRow(`SELECT COUNT(*) FROM media_items`)
	var n int
	err := row.Scan(&n)
	return n, err
}

// SetMediaReleaseGroup updates the release group for a media item.
func (s *SQLStore) SetMediaReleaseGroup(path, group string) error {
	_, err := s.db.Exec(`UPDATE media_items SET release_group = ? WHERE path = ?`, group, path)
	return err
}

// SetMediaAltTitles updates alternate titles for a media item.
func (s *SQLStore) SetMediaAltTitles(path string, titles []string) error {
	data, err := json.Marshal(titles)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE media_items SET alt_titles = ? WHERE path = ?`, string(data), path)
	return err
}

// SetMediaFieldLocks updates field locks for a media item.
func (s *SQLStore) SetMediaFieldLocks(path string, locks string) error {
	_, err := s.db.Exec(`UPDATE media_items SET field_locks = ? WHERE path = ?`, locks, path)
	return err
}

// GetMediaReleaseGroup retrieves the release group for a media item.
func (s *SQLStore) GetMediaReleaseGroup(path string) (string, error) {
	row := s.db.QueryRow(`SELECT release_group FROM media_items WHERE path = ?`, path)
	var group string
	if err := row.Scan(&group); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return group, nil
}

// GetMediaAltTitles retrieves alternate titles for a media item.
func (s *SQLStore) GetMediaAltTitles(path string) ([]string, error) {
	row := s.db.QueryRow(`SELECT alt_titles FROM media_items WHERE path = ?`, path)
	var data string
	if err := row.Scan(&data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if data == "" {
		return nil, nil
	}
	var titles []string
	if err := json.Unmarshal([]byte(data), &titles); err != nil {
		return nil, err
	}
	return titles, nil
}

// GetMediaFieldLocks retrieves locked fields for a media item.
func (s *SQLStore) GetMediaFieldLocks(path string) (string, error) {
	row := s.db.QueryRow(`SELECT field_locks FROM media_items WHERE path = ?`, path)
	var locks string
	if err := row.Scan(&locks); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return locks, nil
}

// SetMediaTitle updates the title for a media item.
func (s *SQLStore) SetMediaTitle(path, title string) error {
	_, err := s.db.Exec(`UPDATE media_items SET title = ? WHERE path = ?`, title, path)
	return err
}

// DB returns the underlying *sql.DB for compatibility with existing code.
func (s *SQLStore) DB() *sql.DB {
	return s.db
}

// Close closes the underlying database connection.
func (s *SQLStore) Close() error {
	return s.db.Close()
}

// InsertSubtitle stores a new subtitle record with associated metadata.
func (s *SQLStore) InsertSubtitle(rec *SubtitleRecord) error {
	_, err := s.db.Exec(`INSERT INTO subtitles (file, video_file, release, language, service, embedded, source_url, provider_metadata, confidence_score, parent_id, modification_type, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.File, rec.VideoFile, rec.Release, rec.Language, rec.Service, boolToInt(rec.Embedded), rec.SourceURL, rec.ProviderMetadata, rec.ConfidenceScore, rec.ParentID, rec.ModificationType, time.Now())
	return err
}

// ListSubtitles retrieves subtitle records ordered by most recent.
func (s *SQLStore) ListSubtitles() ([]SubtitleRecord, error) {
	rows, err := s.db.Query(`SELECT id, file, video_file, release, language, service, embedded, source_url, provider_metadata, confidence_score, parent_id, modification_type, created_at FROM subtitles ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []SubtitleRecord
	for rows.Next() {
		var r SubtitleRecord
		var embedded int
		var id int64
		var sourceURL, providerMetadata, parentID, modificationType sql.NullString
		var confidenceScore sql.NullFloat64
		if err := rows.Scan(&id, &r.File, &r.VideoFile, &r.Release, &r.Language, &r.Service, &embedded, &sourceURL, &providerMetadata, &confidenceScore, &parentID, &modificationType, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		r.Embedded = embedded == 1
		if sourceURL.Valid {
			r.SourceURL = sourceURL.String
		}
		if providerMetadata.Valid {
			r.ProviderMetadata = providerMetadata.String
		}
		if confidenceScore.Valid {
			r.ConfidenceScore = &confidenceScore.Float64
		}
		if parentID.Valid {
			r.ParentID = &parentID.String
		}
		if modificationType.Valid {
			r.ModificationType = modificationType.String
		}
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// ListSubtitlesByVideo retrieves subtitle history for a specific video file.
func (s *SQLStore) ListSubtitlesByVideo(video string) ([]SubtitleRecord, error) {
	rows, err := s.db.Query(`SELECT id, file, video_file, release, language, service, embedded, source_url, provider_metadata, confidence_score, parent_id, modification_type, created_at FROM subtitles WHERE video_file = ? ORDER BY id DESC`, video)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []SubtitleRecord
	for rows.Next() {
		var r SubtitleRecord
		var embedded int
		var id int64
		var sourceURL, providerMetadata, parentID, modificationType sql.NullString
		var confidenceScore sql.NullFloat64
		if err := rows.Scan(&id, &r.File, &r.VideoFile, &r.Release, &r.Language, &r.Service, &embedded, &sourceURL, &providerMetadata, &confidenceScore, &parentID, &modificationType, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		r.Embedded = embedded == 1
		if sourceURL.Valid {
			r.SourceURL = sourceURL.String
		}
		if providerMetadata.Valid {
			r.ProviderMetadata = providerMetadata.String
		}
		if confidenceScore.Valid {
			r.ConfidenceScore = &confidenceScore.Float64
		}
		if parentID.Valid {
			r.ParentID = &parentID.String
		}
		if modificationType.Valid {
			r.ModificationType = modificationType.String
		}
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// DeleteSubtitle removes subtitle records matching file from the database.
func (s *SQLStore) DeleteSubtitle(file string) error {
	_, err := s.db.Exec(`DELETE FROM subtitles WHERE file = ?`, file)
	return err
}

// InsertSubtitle stores a new subtitle record with associated metadata.
func InsertSubtitle(db *sql.DB, file, video, lang, service, release string, embedded bool) error {
	_, err := db.Exec(`INSERT INTO subtitles (file, video_file, release, language, service, embedded, source_url, provider_metadata, confidence_score, parent_id, modification_type, created_at) VALUES (?, ?, ?, ?, ?, ?, '', '', NULL, NULL, '', ?)`,
		file, video, release, lang, service, boolToInt(embedded), time.Now())
	return err
}

func ListSubtitles(db *sql.DB) ([]SubtitleRecord, error) {
	rows, err := db.Query(`SELECT id, file, video_file, release, language, service, embedded, source_url, provider_metadata, confidence_score, parent_id, modification_type, created_at FROM subtitles ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []SubtitleRecord
	for rows.Next() {
		var r SubtitleRecord
		var embedded int
		var id int64
		var sourceURL, providerMetadata, parentID, modificationType sql.NullString
		var confidenceScore sql.NullFloat64
		if err := rows.Scan(&id, &r.File, &r.VideoFile, &r.Release, &r.Language, &r.Service, &embedded, &sourceURL, &providerMetadata, &confidenceScore, &parentID, &modificationType, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		r.Embedded = embedded == 1
		if sourceURL.Valid {
			r.SourceURL = sourceURL.String
		}
		if providerMetadata.Valid {
			r.ProviderMetadata = providerMetadata.String
		}
		if confidenceScore.Valid {
			r.ConfidenceScore = &confidenceScore.Float64
		}
		if parentID.Valid {
			r.ParentID = &parentID.String
		}
		if modificationType.Valid {
			r.ModificationType = modificationType.String
		}
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// ListSubtitlesByVideo retrieves subtitle records for a specific video file using a raw *sql.DB.
func ListSubtitlesByVideo(db *sql.DB, video string) ([]SubtitleRecord, error) {
	rows, err := db.Query(`SELECT id, file, video_file, release, language, service, embedded, source_url, provider_metadata, confidence_score, parent_id, modification_type, created_at FROM subtitles WHERE video_file = ? ORDER BY id DESC`, video)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []SubtitleRecord
	for rows.Next() {
		var r SubtitleRecord
		var embedded int
		var id int64
		var sourceURL, providerMetadata, parentID, modificationType sql.NullString
		var confidenceScore sql.NullFloat64
		if err := rows.Scan(&id, &r.File, &r.VideoFile, &r.Release, &r.Language, &r.Service, &embedded, &sourceURL, &providerMetadata, &confidenceScore, &parentID, &modificationType, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		r.Embedded = embedded == 1
		if sourceURL.Valid {
			r.SourceURL = sourceURL.String
		}
		if providerMetadata.Valid {
			r.ProviderMetadata = providerMetadata.String
		}
		if confidenceScore.Valid {
			r.ConfidenceScore = &confidenceScore.Float64
		}
		if parentID.Valid {
			r.ParentID = &parentID.String
		}
		if modificationType.Valid {
			r.ModificationType = modificationType.String
		}
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// DeleteSubtitle removes subtitle records matching file from the database.
func DeleteSubtitle(db *sql.DB, file string) error {
	_, err := db.Exec(`DELETE FROM subtitles WHERE file = ?`, file)
	return err
}

// InsertMediaItem stores a media item using a raw *sql.DB.
func InsertMediaItem(db *sql.DB, path, title string, season, episode int) error {
	_, err := db.Exec(`INSERT INTO media_items (path, title, season, episode, release_group, alt_titles, field_locks, created_at) VALUES (?, ?, ?, ?, '', '', '', ?)`,
		path, title, season, episode, time.Now())
	return err
}

// ListMediaItems retrieves media items using a raw *sql.DB.
func ListMediaItems(db *sql.DB) ([]MediaItem, error) {
	rows, err := db.Query(`SELECT id, path, title, season, episode, release_group, alt_titles, field_locks, created_at FROM media_items ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []MediaItem
	for rows.Next() {
		var it MediaItem
		var id int64
		if err := rows.Scan(&id, &it.Path, &it.Title, &it.Season, &it.Episode, &it.ReleaseGroup, &it.AltTitles, &it.FieldLocks, &it.CreatedAt); err != nil {
			return nil, err
		}
		it.ID = strconv.FormatInt(id, 10)
		items = append(items, it)
	}
	return items, rows.Err()
}

// DeleteMediaItem removes media items with the given path using a raw *sql.DB.
func DeleteMediaItem(db *sql.DB, path string) error {
	_, err := db.Exec(`DELETE FROM media_items WHERE path = ?`, path)
	return err
}

// Subtitle Source operations

// InsertSubtitleSource stores a new subtitle source record.
func (s *SQLStore) InsertSubtitleSource(src *SubtitleSource) error {
	_, err := s.db.Exec(`INSERT INTO subtitle_sources (source_hash, original_url, provider, title, release_info, file_size, download_count, success_count, avg_rating, last_seen, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		src.SourceHash, src.OriginalURL, src.Provider, src.Title, src.ReleaseInfo, src.FileSize, src.DownloadCount, src.SuccessCount, src.AvgRating, src.LastSeen, src.Metadata, time.Now())
	return err
}

// GetSubtitleSource retrieves a subtitle source by hash.
func (s *SQLStore) GetSubtitleSource(sourceHash string) (*SubtitleSource, error) {
	var src SubtitleSource
	var id int64
	var title, releaseInfo, metadata sql.NullString
	var fileSize sql.NullInt64
	var avgRating sql.NullFloat64

	row := s.db.QueryRow(`SELECT id, source_hash, original_url, provider, title, release_info, file_size, download_count, success_count, avg_rating, last_seen, metadata, created_at FROM subtitle_sources WHERE source_hash = ?`, sourceHash)

	err := row.Scan(&id, &src.SourceHash, &src.OriginalURL, &src.Provider, &title, &releaseInfo, &fileSize, &src.DownloadCount, &src.SuccessCount, &avgRating, &src.LastSeen, &metadata, &src.CreatedAt)
	if err != nil {
		return nil, err
	}

	src.ID = strconv.FormatInt(id, 10)
	if title.Valid {
		src.Title = title.String
	}
	if releaseInfo.Valid {
		src.ReleaseInfo = releaseInfo.String
	}
	if fileSize.Valid {
		size := int(fileSize.Int64)
		src.FileSize = &size
	}
	if avgRating.Valid {
		src.AvgRating = &avgRating.Float64
	}
	if metadata.Valid {
		src.Metadata = metadata.String
	}

	return &src, nil
}

// UpdateSubtitleSourceStats updates download statistics for a subtitle source.
func (s *SQLStore) UpdateSubtitleSourceStats(sourceHash string, downloadCount, successCount int, avgRating *float64) error {
	_, err := s.db.Exec(`UPDATE subtitle_sources SET download_count = ?, success_count = ?, avg_rating = ?, last_seen = ? WHERE source_hash = ?`,
		downloadCount, successCount, avgRating, time.Now(), sourceHash)
	return err
}

// ListSubtitleSources retrieves all subtitle sources for a provider.
func (s *SQLStore) ListSubtitleSources(provider string, limit int) ([]SubtitleSource, error) {
	query := `SELECT id, source_hash, original_url, provider, title, release_info, file_size, download_count, success_count, avg_rating, last_seen, metadata, created_at FROM subtitle_sources`
	args := []interface{}{}

	if provider != "" {
		query += ` WHERE provider = ?`
		args = append(args, provider)
	}

	query += ` ORDER BY last_seen DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []SubtitleSource
	for rows.Next() {
		var src SubtitleSource
		var id int64
		var title, releaseInfo, metadata sql.NullString
		var fileSize sql.NullInt64
		var avgRating sql.NullFloat64

		if err := rows.Scan(&id, &src.SourceHash, &src.OriginalURL, &src.Provider, &title, &releaseInfo, &fileSize, &src.DownloadCount, &src.SuccessCount, &avgRating, &src.LastSeen, &metadata, &src.CreatedAt); err != nil {
			return nil, err
		}

		src.ID = strconv.FormatInt(id, 10)
		if title.Valid {
			src.Title = title.String
		}
		if releaseInfo.Valid {
			src.ReleaseInfo = releaseInfo.String
		}
		if fileSize.Valid {
			size := int(fileSize.Int64)
			src.FileSize = &size
		}
		if avgRating.Valid {
			src.AvgRating = &avgRating.Float64
		}
		if metadata.Valid {
			src.Metadata = metadata.String
		}

		sources = append(sources, src)
	}

	return sources, rows.Err()
}

// DeleteSubtitleSource removes a subtitle source record.
func (s *SQLStore) DeleteSubtitleSource(sourceHash string) error {
	_, err := s.db.Exec(`DELETE FROM subtitle_sources WHERE source_hash = ?`, sourceHash)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// InsertTag adds a new tag to the database with enhanced metadata support.
func (s *SQLStore) InsertTag(name string) error {
	_, err := s.db.Exec(`INSERT INTO tags (name, type, entity_type, created_at) VALUES (?, 'user', 'all', ?)`, name, time.Now())
	return err
}

// InsertTagWithMetadata adds a new tag with full metadata support.
func (s *SQLStore) InsertTagWithMetadata(name, tagType, entityType, color, description string) error {
	_, err := s.db.Exec(`INSERT INTO tags (name, type, entity_type, color, description, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		name, tagType, entityType, color, description, time.Now())
	return err
}

// UpdateTag renames an existing tag and optionally updates metadata.
func (s *SQLStore) UpdateTag(id int64, name string) error {
	_, err := s.db.Exec(`UPDATE tags SET name = ? WHERE id = ?`, name, id)
	return err
}

// UpdateTagWithMetadata updates a tag with enhanced metadata.
func (s *SQLStore) UpdateTagWithMetadata(id int64, name, tagType, entityType, color, description string) error {
	query := `UPDATE tags SET name = ?, type = ?, entity_type = ?, color = ?, description = ? WHERE id = ?`
	_, err := s.db.Exec(query, name, tagType, entityType, color, description, id)
	return err
}

// ListTags returns all defined tags ordered by ID with full metadata.
func (s *SQLStore) ListTags() ([]Tag, error) {
	rows, err := s.db.Query(`SELECT id, name, type, entity_type, color, description, created_at FROM tags ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		var id int64
		var color, description sql.NullString
		if err := rows.Scan(&id, &t.Name, &t.Type, &t.EntityType, &color, &description, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.ID = strconv.FormatInt(id, 10)
		if color.Valid {
			t.Color = color.String
		}
		if description.Valid {
			t.Description = description.String
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// DeleteTag removes a tag by ID.
func (s *SQLStore) DeleteTag(id int64) error {
	_, err := s.db.Exec(`DELETE FROM tags WHERE id = ?`, id)
	return err
}

// Universal tag association methods

// AssignTagToEntity creates a universal tag association with any entity type.
func (s *SQLStore) AssignTagToEntity(tagID int64, entityType, entityID string) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO tag_associations (tag_id, entity_type, entity_id, created_at) VALUES (?, ?, ?, ?)`,
		tagID, entityType, entityID, time.Now())
	return err
}

// RemoveTagFromEntity removes a universal tag association.
func (s *SQLStore) RemoveTagFromEntity(tagID int64, entityType, entityID string) error {
	_, err := s.db.Exec(`DELETE FROM tag_associations WHERE tag_id = ? AND entity_type = ? AND entity_id = ?`,
		tagID, entityType, entityID)
	return err
}

// ListTagsForEntity returns all tags associated with a specific entity.
func (s *SQLStore) ListTagsForEntity(entityType, entityID string) ([]Tag, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.name, t.type, t.entity_type, t.color, t.description, t.created_at
		FROM tags t
		JOIN tag_associations ta ON t.id = ta.tag_id
		WHERE ta.entity_type = ? AND ta.entity_id = ?
		ORDER BY t.id`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		var id int64
		var color, description sql.NullString
		if err := rows.Scan(&id, &t.Name, &t.Type, &t.EntityType, &color, &description, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.ID = strconv.FormatInt(id, 10)
		if color.Valid {
			t.Color = color.String
		}
		if description.Valid {
			t.Description = description.String
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ListEntitiesWithTag returns all entity IDs of a specific type that have the given tag.
func (s *SQLStore) ListEntitiesWithTag(tagID int64, entityType string) ([]string, error) {
	rows, err := s.db.Query(`SELECT entity_id FROM tag_associations WHERE tag_id = ? AND entity_type = ? ORDER BY entity_id`,
		tagID, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var entityID string
		if err := rows.Scan(&entityID); err != nil {
			return nil, err
		}
		out = append(out, entityID)
	}
	return out, rows.Err()
}

// BulkAssignTags assigns multiple tags to multiple entities in a single transaction.
func (s *SQLStore) BulkAssignTags(tagIDs []int64, entities []TagAssociation) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO tag_associations (tag_id, entity_type, entity_id, created_at) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now()
	for _, tagID := range tagIDs {
		for _, entity := range entities {
			if _, err := stmt.Exec(tagID, entity.EntityType, entity.EntityID, now); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// Legacy compatibility methods (will be deprecated)

// AssignTagToUser associates a tag with a user (legacy method).
func (s *SQLStore) AssignTagToUser(userID, tagID int64) error {
	// Use both old and new systems for compatibility
	if err := s.AssignTagToEntity(tagID, "user", strconv.FormatInt(userID, 10)); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT OR IGNORE INTO user_tags (user_id, tag_id) VALUES (?, ?)`, userID, tagID)
	return err
}

// RemoveTagFromUser deletes a tag association for a user (legacy method).
func (s *SQLStore) RemoveTagFromUser(userID, tagID int64) error {
	// Remove from both old and new systems for compatibility
	_ = s.RemoveTagFromEntity(tagID, "user", strconv.FormatInt(userID, 10))
	_, err := s.db.Exec(`DELETE FROM user_tags WHERE user_id = ? AND tag_id = ?`, userID, tagID)
	return err
}

// ListTagsForUser returns tags assigned to a user (enhanced legacy method).
func (s *SQLStore) ListTagsForUser(userID int64) ([]Tag, error) {
	// Prefer new system, fallback to old system for compatibility
	tags, err := s.ListTagsForEntity("user", strconv.FormatInt(userID, 10))
	if err == nil && len(tags) > 0 {
		return tags, nil
	}

	// Fallback to legacy table
	rows, err := s.db.Query(`
		SELECT t.id, t.name, t.type, t.entity_type, t.color, t.description, t.created_at
		FROM tags t
		JOIN user_tags u ON t.id = u.tag_id
		WHERE u.user_id = ?
		ORDER BY t.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		var id int64
		var color, description sql.NullString
		if err := rows.Scan(&id, &t.Name, &t.Type, &t.EntityType, &color, &description, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.ID = strconv.FormatInt(id, 10)
		if color.Valid {
			t.Color = color.String
		}
		if description.Valid {
			t.Description = description.String
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// AssignTagToMedia associates a tag with a media item.
func (s *SQLStore) AssignTagToMedia(mediaID, tagID int64) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO media_tags (media_id, tag_id) VALUES (?, ?)`, mediaID, tagID)
	return err
}

// RemoveTagFromMedia deletes a tag association for a media item.
func (s *SQLStore) RemoveTagFromMedia(mediaID, tagID int64) error {
	_, err := s.db.Exec(`DELETE FROM media_tags WHERE media_id = ? AND tag_id = ?`, mediaID, tagID)
	return err
}

// ListTagsForMedia returns tags assigned to a media item.
func (s *SQLStore) ListTagsForMedia(mediaID int64) ([]Tag, error) {
	rows, err := s.db.Query(`SELECT t.id, t.name, t.created_at FROM tags t JOIN media_tags m ON t.id = m.tag_id WHERE m.media_id = ? ORDER BY t.id`, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		var id int64
		if err := rows.Scan(&id, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.ID = strconv.FormatInt(id, 10)
		out = append(out, t)
	}
	return out, rows.Err()
}

// InsertTag adds a tag using a raw *sql.DB (legacy compatibility).
func InsertTag(db *sql.DB, name string) error {
	_, err := db.Exec(`INSERT INTO tags (name, type, entity_type, created_at) VALUES (?, 'user', 'all', ?)`, name, time.Now())
	return err
}

// UpdateTag updates the name for an existing tag using a raw *sql.DB.
func UpdateTag(db *sql.DB, id int64, name string) error {
	_, err := db.Exec(`UPDATE tags SET name = ? WHERE id = ?`, name, id)
	return err
}

// ListTags retrieves tags using a raw *sql.DB with enhanced metadata.
func ListTags(db *sql.DB) ([]Tag, error) {
	rows, err := db.Query(`SELECT id, name, type, entity_type, color, description, created_at FROM tags ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		var id int64
		var color, description sql.NullString
		if err := rows.Scan(&id, &t.Name, &t.Type, &t.EntityType, &color, &description, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.ID = strconv.FormatInt(id, 10)
		if color.Valid {
			t.Color = color.String
		}
		if description.Valid {
			t.Description = description.String
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// DeleteTag removes a tag by ID using a raw *sql.DB.
func DeleteTag(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM tags WHERE id = ?`, id)
	return err
}

// AssignTagToUser associates a tag with a user using a raw *sql.DB.
func AssignTagToUser(db *sql.DB, userID, tagID int64) error {
	_, err := db.Exec(`INSERT OR IGNORE INTO user_tags (user_id, tag_id) VALUES (?, ?)`, userID, tagID)
	return err
}

// RemoveTagFromUser removes a tag from a user using a raw *sql.DB.
func RemoveTagFromUser(db *sql.DB, userID, tagID int64) error {
	_, err := db.Exec(`DELETE FROM user_tags WHERE user_id = ? AND tag_id = ?`, userID, tagID)
	return err
}

// ListTagsForUser retrieves tags for a user using a raw *sql.DB.
func ListTagsForUser(db *sql.DB, userID int64) ([]Tag, error) {
	rows, err := db.Query(`SELECT t.id, t.name, t.created_at FROM tags t JOIN user_tags u ON t.id = u.tag_id WHERE u.user_id = ? ORDER BY t.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		var id int64
		if err := rows.Scan(&id, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.ID = strconv.FormatInt(id, 10)
		out = append(out, t)
	}
	return out, rows.Err()
}

// AssignTagToMedia associates a tag with a media item using a raw *sql.DB.
func AssignTagToMedia(db *sql.DB, mediaID, tagID int64) error {
	_, err := db.Exec(`INSERT OR IGNORE INTO media_tags (media_id, tag_id) VALUES (?, ?)`, mediaID, tagID)
	return err
}

// RemoveTagFromMedia deletes a tag from a media item using a raw *sql.DB.
func RemoveTagFromMedia(db *sql.DB, mediaID, tagID int64) error {
	_, err := db.Exec(`DELETE FROM media_tags WHERE media_id = ? AND tag_id = ?`, mediaID, tagID)
	return err
}

// ListTagsForMedia retrieves tags for a media item using a raw *sql.DB.
func ListTagsForMedia(db *sql.DB, mediaID int64) ([]Tag, error) {
	rows, err := db.Query(`SELECT t.id, t.name, t.created_at FROM tags t JOIN media_tags m ON t.id = m.tag_id WHERE m.media_id = ? ORDER BY t.id`, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		var id int64
		if err := rows.Scan(&id, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.ID = strconv.FormatInt(id, 10)
		out = append(out, t)
	}
	return out, rows.Err()
}

// Universal tagging methods for raw *sql.DB access

// AssignTagToEntity creates a universal tag association using raw *sql.DB.
func AssignTagToEntity(db *sql.DB, tagID int64, entityType, entityID string) error {
	_, err := db.Exec(`INSERT OR IGNORE INTO tag_associations (tag_id, entity_type, entity_id, created_at) VALUES (?, ?, ?, ?)`,
		tagID, entityType, entityID, time.Now())
	return err
}

// RemoveTagFromEntity removes a universal tag association using raw *sql.DB.
func RemoveTagFromEntity(db *sql.DB, tagID int64, entityType, entityID string) error {
	_, err := db.Exec(`DELETE FROM tag_associations WHERE tag_id = ? AND entity_type = ? AND entity_id = ?`,
		tagID, entityType, entityID)
	return err
}

// ListTagsForEntity returns all tags for a specific entity using raw *sql.DB.
func ListTagsForEntity(db *sql.DB, entityType, entityID string) ([]Tag, error) {
	rows, err := db.Query(`
		SELECT t.id, t.name, t.type, t.entity_type, t.color, t.description, t.created_at
		FROM tags t
		JOIN tag_associations ta ON t.id = ta.tag_id
		WHERE ta.entity_type = ? AND ta.entity_id = ?
		ORDER BY t.id`, entityType, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		var id int64
		var color, description sql.NullString
		if err := rows.Scan(&id, &t.Name, &t.Type, &t.EntityType, &color, &description, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.ID = strconv.FormatInt(id, 10)
		if color.Valid {
			t.Color = color.String
		}
		if description.Valid {
			t.Description = description.String
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetMediaIDByPath returns the media item ID for the given file path.
// It returns sql.ErrNoRows if the path has not been indexed.
func GetMediaIDByPath(db *sql.DB, path string) (int64, error) {
	var id int64
	err := db.QueryRow(`SELECT id FROM media_items WHERE path = ?`, path).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// EnsureMediaItem retrieves the ID for path, creating a new record if needed.
func EnsureMediaItem(db *sql.DB, path string) (int64, error) {
	id, err := GetMediaIDByPath(db, path)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	res, err := db.Exec(`INSERT INTO media_items (path, title, season, episode, release_group, alt_titles, field_locks, created_at) VALUES (?, ?, 0, 0, '', '', '', ?)`,
		path, filepath.Base(path), time.Now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// SetDashboardLayout stores the widget layout JSON for a user.
//
// Parameters:
//
//	db - active database handle
//	userID - identifier for the user
//	layout - JSON string describing widget placement
//
// Returns any error encountered while saving.
func SetDashboardLayout(db *sql.DB, userID int64, layout string) error {
	_, err := db.Exec(`INSERT INTO dashboard_prefs (user_id, layout, updated_at) VALUES (?, ?, ?)
        ON CONFLICT(user_id) DO UPDATE SET layout=excluded.layout, updated_at=excluded.updated_at`,
		userID, layout, time.Now())
	return err
}

// GetDashboardLayout retrieves the widget layout JSON for a user.
//
// Parameters:
//
//	db - active database handle
//	userID - identifier for the user
//
// Returns the stored layout JSON string and any error encountered. If no layout
// exists, the string will be empty and error will be nil.
func GetDashboardLayout(db *sql.DB, userID int64) (string, error) {
	row := db.QueryRow(`SELECT layout FROM dashboard_prefs WHERE user_id = ?`, userID)
	var layout string
	if err := row.Scan(&layout); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return layout, nil
}

// Language Profile operations

// CreateLanguageProfile stores a new language profile.
func (s *SQLStore) CreateLanguageProfile(profile *LanguageProfile) error {
	config, err := profile.MarshalConfig()
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`INSERT INTO language_profiles (id, name, config, cutoff_score, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		profile.ID, profile.Name, string(config), profile.CutoffScore, profile.IsDefault,
		profile.CreatedAt, profile.UpdatedAt)
	return err
}

// GetLanguageProfile retrieves a language profile by ID.
func (s *SQLStore) GetLanguageProfile(id string) (*LanguageProfile, error) {
	var profile LanguageProfile
	var configStr string

	row := s.db.QueryRow(`SELECT id, name, config, cutoff_score, is_default, created_at, updated_at
		FROM language_profiles WHERE id = ?`, id)

	err := row.Scan(&profile.ID, &profile.Name, &configStr, &profile.CutoffScore,
		&profile.IsDefault, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := profile.UnmarshalConfig([]byte(configStr)); err != nil {
		return nil, fmt.Errorf("failed to unmarshal profile config: %w", err)
	}

	return &profile, nil
}

// ListLanguageProfiles retrieves all language profiles.
func (s *SQLStore) ListLanguageProfiles() ([]LanguageProfile, error) {
	rows, err := s.db.Query(`SELECT id, name, config, cutoff_score, is_default, created_at, updated_at
		FROM language_profiles ORDER BY is_default DESC, name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []LanguageProfile
	for rows.Next() {
		var profile LanguageProfile
		var configStr string

		err := rows.Scan(&profile.ID, &profile.Name, &configStr, &profile.CutoffScore,
			&profile.IsDefault, &profile.CreatedAt, &profile.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if err := profile.UnmarshalConfig([]byte(configStr)); err != nil {
			return nil, fmt.Errorf("failed to unmarshal profile config: %w", err)
		}

		profiles = append(profiles, profile)
	}

	return profiles, rows.Err()
}

// UpdateLanguageProfile updates an existing language profile.
func (s *SQLStore) UpdateLanguageProfile(profile *LanguageProfile) error {
	config, err := profile.MarshalConfig()
	if err != nil {
		return fmt.Errorf("failed to marshal profile config: %w", err)
	}

	_, err = s.db.Exec(`UPDATE language_profiles
		SET name = ?, config = ?, cutoff_score = ?, is_default = ?, updated_at = ?
		WHERE id = ?`,
		profile.Name, string(config), profile.CutoffScore, profile.IsDefault,
		profile.UpdatedAt, profile.ID)
	return err
}

// DeleteLanguageProfile removes a language profile by ID.
func (s *SQLStore) DeleteLanguageProfile(id string) error {
	// First remove any media assignments
	if _, err := s.db.Exec(`DELETE FROM media_profiles WHERE profile_id = ?`, id); err != nil {
		return err
	}

	// Then remove the profile itself
	_, err := s.db.Exec(`DELETE FROM language_profiles WHERE id = ?`, id)
	return err
}

// SetDefaultLanguageProfile marks a profile as the default.
func (s *SQLStore) SetDefaultLanguageProfile(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear all default flags
	if _, err := tx.Exec(`UPDATE language_profiles SET is_default = FALSE`); err != nil {
		return err
	}

	// Set the specified profile as default
	if _, err := tx.Exec(`UPDATE language_profiles SET is_default = TRUE WHERE id = ?`, id); err != nil {
		return err
	}

	return tx.Commit()
}

// GetDefaultLanguageProfile retrieves the default language profile.
func (s *SQLStore) GetDefaultLanguageProfile() (*LanguageProfile, error) {
	var profile LanguageProfile
	var configStr string

	row := s.db.QueryRow(`SELECT id, name, config, cutoff_score, is_default, created_at, updated_at
		FROM language_profiles WHERE is_default = TRUE LIMIT 1`)

	err := row.Scan(&profile.ID, &profile.Name, &configStr, &profile.CutoffScore,
		&profile.IsDefault, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		return nil, err
	}

	if err := profile.UnmarshalConfig([]byte(configStr)); err != nil {
		return nil, fmt.Errorf("failed to unmarshal profile config: %w", err)
	}

	return &profile, nil
}

// AssignProfileToMedia assigns a language profile to a media item.
func (s *SQLStore) AssignProfileToMedia(mediaID, profileID string) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO media_profiles (media_id, profile_id, created_at)
		VALUES (?, ?, ?)`, mediaID, profileID, time.Now())
	return err
}

// RemoveProfileFromMedia removes language profile assignment from a media item.
func (s *SQLStore) RemoveProfileFromMedia(mediaID string) error {
	_, err := s.db.Exec(`DELETE FROM media_profiles WHERE media_id = ?`, mediaID)
	return err
}

// GetAssignedProfileID reports which profile was explicitly assigned to a media
// item, or "" when it has none. It never falls back to the default.
func (s *SQLStore) GetAssignedProfileID(mediaID string) (string, error) {
	var profileID string
	row := s.db.QueryRow(`SELECT profile_id FROM media_profiles WHERE media_id = ?`, mediaID)
	if err := row.Scan(&profileID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return profileID, nil
}

// GetMediaProfile retrieves the language profile assigned to a media item.
func (s *SQLStore) GetMediaProfile(mediaID string) (*LanguageProfile, error) {
	var profileID string
	row := s.db.QueryRow(`SELECT profile_id FROM media_profiles WHERE media_id = ?`, mediaID)

	err := row.Scan(&profileID)
	if err != nil {
		if err == sql.ErrNoRows {
			// No profile assigned, return default profile
			return s.GetDefaultLanguageProfile()
		}
		return nil, err
	}

	return s.GetLanguageProfile(profileID)
}

// InsertMonitoredItem stores a monitored item record.
func (s *SQLStore) InsertMonitoredItem(rec *MonitoredItem) error {
	_, err := s.db.Exec(`INSERT INTO monitored_items (media_id, path, languages, last_checked, status, retry_count, max_retries, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.MediaID, rec.Path, rec.Languages, rec.LastChecked, rec.Status, rec.RetryCount, rec.MaxRetries, time.Now(), time.Now())
	return err
}

// ListMonitoredItems retrieves all monitored items.
func (s *SQLStore) ListMonitoredItems() ([]MonitoredItem, error) {
	rows, err := s.db.Query(`SELECT id, media_id, path, languages, last_checked, status, retry_count, max_retries, created_at, updated_at FROM monitored_items ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []MonitoredItem
	for rows.Next() {
		var r MonitoredItem
		var id int64
		if err := rows.Scan(&id, &r.MediaID, &r.Path, &r.Languages, &r.LastChecked, &r.Status, &r.RetryCount, &r.MaxRetries, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// UpdateMonitoredItem updates an existing monitored item.
func (s *SQLStore) UpdateMonitoredItem(rec *MonitoredItem) error {
	_, err := s.db.Exec(`UPDATE monitored_items SET last_checked = ?, status = ?, retry_count = ?, updated_at = ? WHERE id = ?`,
		rec.LastChecked, rec.Status, rec.RetryCount, time.Now(), rec.ID)
	return err
}

// DeleteMonitoredItem removes a monitored item by ID.
func (s *SQLStore) DeleteMonitoredItem(id string) error {
	_, err := s.db.Exec(`DELETE FROM monitored_items WHERE id = ?`, id)
	return err
}

// GetMonitoredItemsToCheck returns items that need monitoring.
func (s *SQLStore) GetMonitoredItemsToCheck(interval time.Duration) ([]MonitoredItem, error) {
	cutoff := time.Now().Add(-interval)
	rows, err := s.db.Query(`SELECT id, media_id, path, languages, last_checked, status, retry_count, max_retries, created_at, updated_at FROM monitored_items WHERE status IN ('pending', 'monitoring', 'failed') AND last_checked < ? AND retry_count < max_retries ORDER BY last_checked ASC`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []MonitoredItem
	for rows.Next() {
		var r MonitoredItem
		var id int64
		if err := rows.Scan(&id, &r.MediaID, &r.Path, &r.Languages, &r.LastChecked, &r.Status, &r.RetryCount, &r.MaxRetries, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// Authentication methods for SQLStore

// CreateUser creates a new user with hashed password and returns user ID (placeholder - delegates to gcommonauth).
func (s *SQLStore) CreateUser(username, passwordHash, email, role string) (string, error) {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	// which directly uses the database connection
	return "", nil
}

// GetUserByUsername retrieves a user by username (placeholder - delegates to gcommonauth).
func (s *SQLStore) GetUserByUsername(username string) (*common.User, error) {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil, nil
}

// GetUserByEmail retrieves a user by email (placeholder - delegates to gcommonauth).
func (s *SQLStore) GetUserByEmail(email string) (*common.User, error) {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil, nil
}

// GetUserByID retrieves a user by ID (placeholder - delegates to gcommonauth).
func (s *SQLStore) GetUserByID(id string) (*common.User, error) {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil, nil
}

// ListUsers retrieves all users (placeholder - delegates to gcommonauth).
func (s *SQLStore) ListUsers() ([]*common.User, error) {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil, nil
}

// UpdateUserRole updates a user's role (placeholder - delegates to gcommonauth).
func (s *SQLStore) UpdateUserRole(username, role string) error {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil
}

// UpdateUserPassword updates a user's password hash (placeholder - delegates to gcommonauth).
func (s *SQLStore) UpdateUserPassword(userID, passwordHash string) error {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil
}

// CreateSession creates a new user session (placeholder - delegates to gcommonauth).
func (s *SQLStore) CreateSession(userID, token string, duration time.Duration) error {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil
}

// ValidateSession validates a session token and returns user ID (placeholder - delegates to gcommonauth).
func (s *SQLStore) ValidateSession(token string) (string, error) {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return "", nil
}

// DeleteSession deletes a session (placeholder - delegates to gcommonauth).
func (s *SQLStore) DeleteSession(token string) error {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil
}

// InvalidateSession invalidates a specific session (placeholder - delegates to gcommonauth).
func (s *SQLStore) InvalidateSession(token string) error {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil
}

// InvalidateUserSessions invalidates all sessions for a user (placeholder - delegates to gcommonauth).
func (s *SQLStore) InvalidateUserSessions(userID string) error {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil
}

// DeleteUser deletes a user (placeholder - delegates to gcommonauth).
func (s *SQLStore) DeleteUser(userID string) error {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil
}

// CleanupExpiredSessions removes expired sessions (placeholder - delegates to gcommonauth).
func (s *SQLStore) CleanupExpiredSessions() error {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil
}

// ValidateOneTimeToken validates a one-time token (placeholder - delegates to gcommonauth).
func (s *SQLStore) ValidateOneTimeToken(token string) (string, error) {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return "", nil
}

// CreateAPIKey creates a new API key for a user (placeholder - delegates to gcommonauth).
func (s *SQLStore) CreateAPIKey(userID, key string) error {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil
}

// ValidateAPIKey validates an API key and returns user ID (placeholder - delegates to gcommonauth).
func (s *SQLStore) ValidateAPIKey(key string) (string, error) {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return "", nil
}

// CreateOneTimeToken creates a one-time use token (placeholder - delegates to gcommonauth).
func (s *SQLStore) CreateOneTimeToken(userID, token string, duration time.Duration) error {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil
}

// DeleteAPIKey deletes an API key (placeholder - delegates to gcommonauth).
func (s *SQLStore) DeleteAPIKey(key string) error {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil
}

// SetDashboardLayout sets dashboard layout preferences for a user (placeholder - delegates to gcommonauth).
func (s *SQLStore) SetDashboardLayout(userID, layout string) error {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return nil
}

// GetDashboardLayout gets dashboard layout preferences for a user (placeholder - delegates to gcommonauth).
func (s *SQLStore) GetDashboardLayout(userID string) (string, error) {
	// This is a placeholder - SQLStore authentication is handled by gcommonauth package
	return "", nil
}
