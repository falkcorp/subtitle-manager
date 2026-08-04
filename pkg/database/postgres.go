// file: pkg/database/postgres.go
// version: 1.2.0
// guid: 29e61686-4676-4b4a-b4be-b0cb294239a3
// last-edited: 2026-08-04

package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	common "buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go/commonpb/v2"
	_ "github.com/lib/pq"
)

// PostgresStore implements SubtitleStore backed by PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// OpenPostgresStore opens a connection to a PostgreSQL database using dsn.
// The database schema is created if it does not already exist.
func OpenPostgresStore(dsn string) (*PostgresStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := initPostgresSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return &PostgresStore{db: db}, nil
}

func initPostgresSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS subtitles (
			id SERIAL PRIMARY KEY,
			file TEXT NOT NULL,
			video_file TEXT,
			release TEXT,
			language TEXT NOT NULL,
			service TEXT NOT NULL,
			embedded BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS downloads (
			id SERIAL PRIMARY KEY,
			file TEXT NOT NULL,
			video_file TEXT NOT NULL,
			provider TEXT NOT NULL,
			language TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS blacklist (
			id TEXT PRIMARY KEY,
			item_id TEXT,
			path TEXT,
			language TEXT,
			reason TEXT,
			details TEXT,
			created_at TIMESTAMP NOT NULL,
			expires_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS search_history (
			id SERIAL PRIMARY KEY,
			query TEXT NOT NULL,
			results INTEGER NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS media_items (
			id SERIAL PRIMARY KEY,
			path TEXT NOT NULL,
			title TEXT NOT NULL,
			season INTEGER,
			episode INTEGER,
			release_group TEXT,
			alt_titles TEXT,
			field_locks TEXT,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS dashboard_prefs (
			user_id INTEGER PRIMARY KEY,
			layout TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS language_profiles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			config TEXT NOT NULL,
			cutoff_score INTEGER DEFAULT 80,
			is_default BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS media_profiles (
			media_id TEXT NOT NULL PRIMARY KEY,
			profile_id TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			FOREIGN KEY (profile_id) REFERENCES language_profiles(id) ON DELETE CASCADE
		)`,
		// Add missing language_profile_assignments table for Bazarr-style language management
		`CREATE TABLE IF NOT EXISTS language_profile_assignments (
			media_path TEXT PRIMARY KEY,
			profile_id TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}

	// Indexes for improved query performance
	idxStmts := []string{
		`CREATE INDEX IF NOT EXISTS idx_subtitles_file ON subtitles(file)`,
		`CREATE INDEX IF NOT EXISTS idx_subtitles_video ON subtitles(video_file)`,
		`CREATE INDEX IF NOT EXISTS idx_downloads_video_lang ON downloads(video_file, language)`,
		`CREATE INDEX IF NOT EXISTS idx_media_items_path ON media_items(path)`,
		`CREATE INDEX IF NOT EXISTS idx_downloads_file ON downloads(file)`,
		`CREATE INDEX IF NOT EXISTS idx_subtitle_sources_hash ON subtitle_sources(source_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_media_profiles_media ON media_profiles(media_id)`,
		`CREATE INDEX IF NOT EXISTS idx_media_profiles_profile ON media_profiles(profile_id)`,
		`CREATE INDEX IF NOT EXISTS idx_monitored_items_status_checked ON monitored_items(status, last_checked)`,
	}
	for _, s := range idxStmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

// User authentication and session management methods (placeholders - delegates to gcommonauth)

// CreateUser creates a new user (placeholder).
func (p *PostgresStore) CreateUser(username, passwordHash, email, role string) (string, error) {
	return "", nil
}

// GetUserByUsername retrieves a user by username (placeholder).
func (p *PostgresStore) GetUserByUsername(username string) (*common.User, error) {
	return nil, nil
}

// GetUserByEmail retrieves a user by email (placeholder).
func (p *PostgresStore) GetUserByEmail(email string) (*common.User, error) {
	return nil, nil
}

// GetUserByID retrieves a user by ID (placeholder).
func (p *PostgresStore) GetUserByID(id string) (*common.User, error) {
	return nil, nil
}

// ListUsers retrieves all users (placeholder).
func (p *PostgresStore) ListUsers() ([]*common.User, error) {
	return nil, nil
}

// UpdateUserRole updates a user's role (placeholder).
func (p *PostgresStore) UpdateUserRole(username, role string) error {
	return nil
}

// UpdateUserPassword updates a user's password hash (placeholder).
func (p *PostgresStore) UpdateUserPassword(userID, passwordHash string) error {
	return nil
}

// CreateSession creates a new user session (placeholder).
func (p *PostgresStore) CreateSession(userID, token string, duration time.Duration) error {
	return nil
}

// ValidateSession validates a session token (placeholder).
func (p *PostgresStore) ValidateSession(token string) (string, error) {
	return "", nil
}

// InvalidateSession invalidates a specific session (placeholder).
func (p *PostgresStore) InvalidateSession(token string) error {
	return nil
}

// InvalidateUserSessions invalidates all sessions for a user (placeholder).
func (p *PostgresStore) InvalidateUserSessions(userID string) error {
	return nil
}

// CleanupExpiredSessions removes expired sessions (placeholder).
func (p *PostgresStore) CleanupExpiredSessions() error {
	return nil
}

// CreateAPIKey creates an API key for a user (placeholder).
func (p *PostgresStore) CreateAPIKey(userID, key string) error {
	return nil
}

// ValidateAPIKey validates an API key (placeholder).
func (p *PostgresStore) ValidateAPIKey(key string) (string, error) {
	return "", nil
}

// CreateOneTimeToken creates a one-time use token (placeholder).
func (p *PostgresStore) CreateOneTimeToken(userID, token string, duration time.Duration) error {
	return nil
}

// ValidateOneTimeToken validates a one-time token (placeholder).
func (p *PostgresStore) ValidateOneTimeToken(token string) (string, error) {
	return "", nil
}

// SetDashboardLayout sets dashboard layout preferences (placeholder).
func (p *PostgresStore) SetDashboardLayout(userID, layout string) error {
	return nil
}

// GetDashboardLayout gets dashboard layout preferences (placeholder).
func (p *PostgresStore) GetDashboardLayout(userID string) (string, error) {
	return "", nil
}

// Close closes the underlying PostgreSQL connection.
func (p *PostgresStore) Close() error { return p.db.Close() }

// InsertSubtitle stores a subtitle record.
func (p *PostgresStore) InsertSubtitle(rec *SubtitleRecord) error {
	_, err := p.db.Exec(`INSERT INTO subtitles (file, video_file, release, language, service, embedded, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		rec.File, rec.VideoFile, rec.Release, rec.Language, rec.Service, rec.Embedded, time.Now())
	return err
}

// ListSubtitles retrieves stored subtitle records ordered by most recent.
func (p *PostgresStore) ListSubtitles() ([]SubtitleRecord, error) {
	rows, err := p.db.Query(`SELECT id, file, video_file, release, language, service, embedded, created_at FROM subtitles ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []SubtitleRecord
	for rows.Next() {
		var r SubtitleRecord
		var id int64
		if err := rows.Scan(&id, &r.File, &r.VideoFile, &r.Release, &r.Language, &r.Service, &r.Embedded, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// ListSubtitlesByVideo retrieves subtitle records for a specific video file.
func (p *PostgresStore) ListSubtitlesByVideo(video string) ([]SubtitleRecord, error) {
	rows, err := p.db.Query(`SELECT id, file, video_file, release, language, service, embedded, created_at FROM subtitles WHERE video_file = $1 ORDER BY id DESC`, video)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []SubtitleRecord
	for rows.Next() {
		var r SubtitleRecord
		var id int64
		if err := rows.Scan(&id, &r.File, &r.VideoFile, &r.Release, &r.Language, &r.Service, &r.Embedded, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// DeleteSubtitle removes subtitle records matching file.
func (p *PostgresStore) DeleteSubtitle(file string) error {
	_, err := p.db.Exec(`DELETE FROM subtitles WHERE file = $1`, file)
	return err
}

// InsertDownload stores a download record.
func (p *PostgresStore) InsertDownload(rec *DownloadRecord) error {
	_, err := p.db.Exec(`INSERT INTO downloads (file, video_file, provider, language, created_at) VALUES ($1,$2,$3,$4,$5)`,
		rec.File, rec.VideoFile, rec.Provider, rec.Language, time.Now())
	return err
}

// ListDownloads retrieves download records ordered by most recent.
func (p *PostgresStore) ListDownloads() ([]DownloadRecord, error) {
	rows, err := p.db.Query(`SELECT id, file, video_file, provider, language, created_at FROM downloads ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []DownloadRecord
	for rows.Next() {
		var r DownloadRecord
		var id int64
		if err := rows.Scan(&id, &r.File, &r.VideoFile, &r.Provider, &r.Language, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// ListDownloadsByVideo retrieves download records for a specific video file.
func (p *PostgresStore) ListDownloadsByVideo(video string) ([]DownloadRecord, error) {
	rows, err := p.db.Query(`SELECT id, file, video_file, provider, language, created_at FROM downloads WHERE video_file = $1 ORDER BY id DESC`, video)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []DownloadRecord
	for rows.Next() {
		var r DownloadRecord
		var id int64
		if err := rows.Scan(&id, &r.File, &r.VideoFile, &r.Provider, &r.Language, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.ID = strconv.FormatInt(id, 10)
		recs = append(recs, r)
	}
	return recs, rows.Err()
}

// DeleteDownload removes download records matching file.
func (p *PostgresStore) DeleteDownload(file string) error {
	_, err := p.db.Exec(`DELETE FROM downloads WHERE file = $1`, file)
	return err
}

// InsertMediaItem stores a media library record.
func (p *PostgresStore) InsertMediaItem(rec *MediaItem) error {
	_, err := p.db.Exec(`INSERT INTO media_items (path, title, season, episode, release_group, alt_titles, field_locks, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		rec.Path, rec.Title, rec.Season, rec.Episode, rec.ReleaseGroup, rec.AltTitles, rec.FieldLocks, time.Now())
	return err
}

// ListMediaItems retrieves media items ordered by most recent.
func (p *PostgresStore) ListMediaItems() ([]MediaItem, error) {
	rows, err := p.db.Query(`SELECT id, path, title, season, episode, release_group, alt_titles, field_locks, created_at FROM media_items ORDER BY id DESC`)
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

// DeleteMediaItem removes records matching path.
func (p *PostgresStore) DeleteMediaItem(path string) error {
	_, err := p.db.Exec(`DELETE FROM media_items WHERE path = $1`, path)
	return err
}

// CountSubtitles returns the number of subtitle records.
func (p *PostgresStore) CountSubtitles() (int, error) {
	row := p.db.QueryRow(`SELECT COUNT(*) FROM subtitles`)
	var n int
	err := row.Scan(&n)
	return n, err
}

// CountDownloads returns the number of download records.
func (p *PostgresStore) CountDownloads() (int, error) {
	row := p.db.QueryRow(`SELECT COUNT(*) FROM downloads`)
	var n int
	err := row.Scan(&n)
	return n, err
}

// CountMediaItems returns the number of media items.
func (p *PostgresStore) CountMediaItems() (int, error) {
	row := p.db.QueryRow(`SELECT COUNT(*) FROM media_items`)
	var n int
	err := row.Scan(&n)
	return n, err
}

// SetMediaReleaseGroup updates the release group for a media item.
func (p *PostgresStore) SetMediaReleaseGroup(path, group string) error {
	_, err := p.db.Exec(`UPDATE media_items SET release_group = $1 WHERE path = $2`, group, path)
	return err
}

// SetMediaAltTitles updates alternate titles for a media item.
func (p *PostgresStore) SetMediaAltTitles(path string, titles []string) error {
	data, err := json.Marshal(titles)
	if err != nil {
		return err
	}
	_, err = p.db.Exec(`UPDATE media_items SET alt_titles = $1 WHERE path = $2`, string(data), path)
	return err
}

// SetMediaFieldLocks updates field locks for a media item.
func (p *PostgresStore) SetMediaFieldLocks(path, locks string) error {
	_, err := p.db.Exec(`UPDATE media_items SET field_locks = $1 WHERE path = $2`, locks, path)
	return err
}

// GetMediaReleaseGroup retrieves the release group for a media item.
func (p *PostgresStore) GetMediaReleaseGroup(path string) (string, error) {
	row := p.db.QueryRow(`SELECT release_group FROM media_items WHERE path = $1`, path)
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
func (p *PostgresStore) GetMediaAltTitles(path string) ([]string, error) {
	row := p.db.QueryRow(`SELECT alt_titles FROM media_items WHERE path = $1`, path)
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
func (p *PostgresStore) GetMediaFieldLocks(path string) (string, error) {
	row := p.db.QueryRow(`SELECT field_locks FROM media_items WHERE path = $1`, path)
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
func (p *PostgresStore) SetMediaTitle(path, title string) error {
	_, err := p.db.Exec(`UPDATE media_items SET title = $1 WHERE path = $2`, title, path)
	return err
}

// GetMediaItem retrieves a media item by path. Returns nil if not found.
func (p *PostgresStore) GetMediaItem(path string) (*MediaItem, error) {
	row := p.db.QueryRow(`SELECT id, path, title, season, episode, release_group, alt_titles, field_locks, created_at FROM media_items WHERE path = $1`, path)
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

// InsertTag adds a tag to the database.
func (p *PostgresStore) InsertTag(name string) error {
	_, err := p.db.Exec(`INSERT INTO tags (name, created_at) VALUES ($1, $2)`, name, time.Now())
	return err
}

// UpdateTag renames a tag.
func (p *PostgresStore) UpdateTag(id int64, name string) error {
	_, err := p.db.Exec(`UPDATE tags SET name = $1 WHERE id = $2`, name, id)
	return err
}

// ListTags returns all tags.
func (p *PostgresStore) ListTags() ([]Tag, error) {
	rows, err := p.db.Query(`SELECT id, name, created_at FROM tags ORDER BY id`)
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

// DeleteTag removes a tag.
func (p *PostgresStore) DeleteTag(id int64) error {
	_, err := p.db.Exec(`DELETE FROM tags WHERE id = $1`, id)
	return err
}

// AssignTagToUser associates a tag with a user.
func (p *PostgresStore) AssignTagToUser(userID, tagID int64) error {
	_, err := p.db.Exec(`INSERT INTO user_tags (user_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, tagID)
	return err
}

// RemoveTagFromUser removes a tag from a user.
func (p *PostgresStore) RemoveTagFromUser(userID, tagID int64) error {
	_, err := p.db.Exec(`DELETE FROM user_tags WHERE user_id = $1 AND tag_id = $2`, userID, tagID)
	return err
}

// ListTagsForUser returns tags associated with a user.
func (p *PostgresStore) ListTagsForUser(userID int64) ([]Tag, error) {
	rows, err := p.db.Query(`SELECT t.id, t.name, t.created_at FROM tags t JOIN user_tags u ON t.id = u.tag_id WHERE u.user_id = $1 ORDER BY t.id`, userID)
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

// AssignTagToMedia associates a tag with a media item.
func (p *PostgresStore) AssignTagToMedia(mediaID, tagID int64) error {
	_, err := p.db.Exec(`INSERT INTO media_tags (media_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, mediaID, tagID)
	return err
}

// RemoveTagFromMedia removes a tag from a media item.
func (p *PostgresStore) RemoveTagFromMedia(mediaID, tagID int64) error {
	_, err := p.db.Exec(`DELETE FROM media_tags WHERE media_id = $1 AND tag_id = $2`, mediaID, tagID)
	return err
}

// ListTagsForMedia returns tags associated with a media item.
func (p *PostgresStore) ListTagsForMedia(mediaID int64) ([]Tag, error) {
	rows, err := p.db.Query(`SELECT t.id, t.name, t.created_at FROM tags t JOIN media_tags m ON t.id = m.tag_id WHERE m.media_id = $1 ORDER BY t.id`, mediaID)
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

// Language Profile operations for PostgresStore

// CreateLanguageProfile stores a new language profile.
func (p *PostgresStore) CreateLanguageProfile(profile *LanguageProfile) error {
	config, err := profile.MarshalConfig()
	if err != nil {
		return err
	}

	_, err = p.db.Exec(`INSERT INTO language_profiles (id, name, config, cutoff_score, is_default, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		profile.ID, profile.Name, string(config), profile.CutoffScore, profile.IsDefault,
		profile.CreatedAt, profile.UpdatedAt)
	return err
}

// GetLanguageProfile retrieves a language profile by ID.
func (p *PostgresStore) GetLanguageProfile(id string) (*LanguageProfile, error) {
	var profile LanguageProfile
	var configStr string

	row := p.db.QueryRow(`SELECT id, name, config, cutoff_score, is_default, created_at, updated_at
FROM language_profiles WHERE id = $1`, id)

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
func (p *PostgresStore) ListLanguageProfiles() ([]LanguageProfile, error) {
	rows, err := p.db.Query(`SELECT id, name, config, cutoff_score, is_default, created_at, updated_at
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
func (p *PostgresStore) UpdateLanguageProfile(profile *LanguageProfile) error {
	config, err := profile.MarshalConfig()
	if err != nil {
		return fmt.Errorf("failed to marshal profile config: %w", err)
	}

	_, err = p.db.Exec(`UPDATE language_profiles
SET name = $1, config = $2, cutoff_score = $3, is_default = $4, updated_at = $5
WHERE id = $6`,
		profile.Name, string(config), profile.CutoffScore, profile.IsDefault,
		profile.UpdatedAt, profile.ID)
	return err
}

// DeleteLanguageProfile removes a language profile by ID.
func (p *PostgresStore) DeleteLanguageProfile(id string) error {
	// First remove any media assignments
	if _, err := p.db.Exec(`DELETE FROM media_profiles WHERE profile_id = $1`, id); err != nil {
		return err
	}

	// Then remove the profile itself
	_, err := p.db.Exec(`DELETE FROM language_profiles WHERE id = $1`, id)
	return err
}

// SetDefaultLanguageProfile marks a profile as the default.
func (p *PostgresStore) SetDefaultLanguageProfile(id string) error {
	tx, err := p.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear all default flags
	if _, err := tx.Exec(`UPDATE language_profiles SET is_default = FALSE`); err != nil {
		return err
	}

	// Set the specified profile as default
	if _, err := tx.Exec(`UPDATE language_profiles SET is_default = TRUE WHERE id = $1`, id); err != nil {
		return err
	}

	return tx.Commit()
}

// GetDefaultLanguageProfile retrieves the default language profile.
func (p *PostgresStore) GetDefaultLanguageProfile() (*LanguageProfile, error) {
	var profile LanguageProfile
	var configStr string

	row := p.db.QueryRow(`SELECT id, name, config, cutoff_score, is_default, created_at, updated_at
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
func (p *PostgresStore) AssignProfileToMedia(mediaID, profileID string) error {
	_, err := p.db.Exec(`INSERT INTO media_profiles (media_id, profile_id, created_at)
VALUES ($1, $2, $3) ON CONFLICT (media_id) DO UPDATE SET profile_id = EXCLUDED.profile_id, created_at = EXCLUDED.created_at`,
		mediaID, profileID, time.Now())
	return err
}

// RemoveProfileFromMedia removes language profile assignment from a media item.
func (p *PostgresStore) RemoveProfileFromMedia(mediaID string) error {
	_, err := p.db.Exec(`DELETE FROM media_profiles WHERE media_id = $1`, mediaID)
	return err
}

// GetAssignedProfileID reports which profile was explicitly assigned to a media
// item, or "" when it has none. It never falls back to the default.
func (p *PostgresStore) GetAssignedProfileID(mediaID string) (string, error) {
	var profileID string
	row := p.db.QueryRow(`SELECT profile_id FROM media_profiles WHERE media_id = $1`, mediaID)
	if err := row.Scan(&profileID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return profileID, nil
}

// GetMediaProfile retrieves the language profile assigned to a media item.
func (p *PostgresStore) GetMediaProfile(mediaID string) (*LanguageProfile, error) {
	var profileID string
	row := p.db.QueryRow(`SELECT profile_id FROM media_profiles WHERE media_id = $1`, mediaID)

	err := row.Scan(&profileID)
	if err != nil {
		if err == sql.ErrNoRows {
			// No profile assigned, return default profile
			return p.GetDefaultLanguageProfile()
		}
		return nil, err
	}

	return p.GetLanguageProfile(profileID)
}

// Subtitle Source operations for PostgresStore

// InsertSubtitleSource stores a new subtitle source record.
func (p *PostgresStore) InsertSubtitleSource(src *SubtitleSource) error {
	_, err := p.db.Exec(`INSERT INTO subtitle_sources (source_hash, original_url, provider, title, release_info, file_size, download_count, success_count, avg_rating, last_seen, metadata, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		src.SourceHash, src.OriginalURL, src.Provider, src.Title, src.ReleaseInfo, src.FileSize, src.DownloadCount, src.SuccessCount, src.AvgRating, src.LastSeen, src.Metadata, time.Now())
	return err
}

// GetSubtitleSource retrieves a subtitle source by hash.
func (p *PostgresStore) GetSubtitleSource(sourceHash string) (*SubtitleSource, error) {
	var src SubtitleSource
	var id int64
	var title, releaseInfo, metadata sql.NullString
	var fileSize sql.NullInt64
	var avgRating sql.NullFloat64

	row := p.db.QueryRow(`SELECT id, source_hash, original_url, provider, title, release_info, file_size, download_count, success_count, avg_rating, last_seen, metadata, created_at FROM subtitle_sources WHERE source_hash = $1`, sourceHash)

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
func (p *PostgresStore) UpdateSubtitleSourceStats(sourceHash string, downloadCount, successCount int, avgRating *float64) error {
	_, err := p.db.Exec(`UPDATE subtitle_sources SET download_count = $1, success_count = $2, avg_rating = $3, last_seen = $4 WHERE source_hash = $5`,
		downloadCount, successCount, avgRating, time.Now(), sourceHash)
	return err
}

// ListSubtitleSources retrieves all subtitle sources for a provider.
func (p *PostgresStore) ListSubtitleSources(provider string, limit int) ([]SubtitleSource, error) {
	query := `SELECT id, source_hash, original_url, provider, title, release_info, file_size, download_count, success_count, avg_rating, last_seen, metadata, created_at FROM subtitle_sources`
	args := []interface{}{}

	if provider != "" {
		query += ` WHERE provider = $1`
		args = append(args, provider)
	}

	query += ` ORDER BY last_seen DESC`
	if limit > 0 {
		if provider != "" {
			query += ` LIMIT $2`
		} else {
			query += ` LIMIT $1`
		}
		args = append(args, limit)
	}

	rows, err := p.db.Query(query, args...)
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
func (p *PostgresStore) DeleteSubtitleSource(sourceHash string) error {
	_, err := p.db.Exec(`DELETE FROM subtitle_sources WHERE source_hash = $1`, sourceHash)
	return err
}

// ==================== MONITORING FUNCTIONS ====================

// InsertMonitoredItem stores a monitored item record.
func (p *PostgresStore) InsertMonitoredItem(rec *MonitoredItem) error {
	_, err := p.db.Exec(`INSERT INTO monitored_items (media_id, path, languages, last_checked, status, retry_count, max_retries, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		rec.MediaID, rec.Path, rec.Languages, rec.LastChecked, rec.Status, rec.RetryCount, rec.MaxRetries, time.Now(), time.Now())
	return err
}

// ListMonitoredItems retrieves all monitored items.
func (p *PostgresStore) ListMonitoredItems() ([]MonitoredItem, error) {
	rows, err := p.db.Query(`SELECT id, media_id, path, languages, last_checked, status, retry_count, max_retries, created_at, updated_at FROM monitored_items ORDER BY id DESC`)
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
func (p *PostgresStore) UpdateMonitoredItem(rec *MonitoredItem) error {
	_, err := p.db.Exec(`UPDATE monitored_items SET last_checked = $2, status = $3, retry_count = $4, updated_at = $5 WHERE id = $1`,
		rec.ID, rec.LastChecked, rec.Status, rec.RetryCount, time.Now())
	return err
}

// DeleteMonitoredItem removes a monitored item by ID.
func (p *PostgresStore) DeleteMonitoredItem(id string) error {
	_, err := p.db.Exec(`DELETE FROM monitored_items WHERE id = $1`, id)
	return err
}

// GetMonitoredItemsToCheck returns items that need monitoring.
func (p *PostgresStore) GetMonitoredItemsToCheck(interval time.Duration) ([]MonitoredItem, error) {
	cutoff := time.Now().Add(-interval)
	rows, err := p.db.Query(`SELECT id, media_id, path, languages, last_checked, status, retry_count, max_retries, created_at, updated_at FROM monitored_items WHERE status IN ('pending', 'monitoring', 'failed') AND last_checked < $1 AND retry_count < max_retries ORDER BY last_checked ASC`, cutoff)
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
