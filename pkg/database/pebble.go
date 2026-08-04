// file: pkg/database/pebble.go
// version: 1.0.0
// guid: f376d786-9298-4568-a0d8-7d4167d0f7c0
// last-edited: 2026-08-04

package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	common "buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go/commonpb/v2"
	"github.com/cockroachdb/pebble"
	"github.com/google/uuid"
	profilesPkg "github.com/jdfalk/subtitle-manager/pkg/profiles"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PebbleStore wraps a Pebble database and implements basic CRUD operations
// for SubtitleRecord documents.
// Keys are generated using UUIDs with the "subtitle:" prefix.
// Values are stored as JSON encoded SubtitleRecord structures.
type PebbleStore struct {
	db *pebble.DB
}

// InitPebbleStore seeds default permissions and language profile if missing.
func (p *PebbleStore) InitPebbleStore() error {
	// Seed default permissions if missing
	if err := p.InitializeDefaultPermissions(); err != nil {
		return err
	}

	// Seed default language profile if missing
	profiles, err := p.ListLanguageProfiles()
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		defaultProfile := &LanguageProfile{
			ID:          "default",
			Name:        "Default English",
			Languages:   []profilesPkg.LanguageConfig{{Language: "en", Priority: 1, Forced: false, HI: false}},
			CutoffScore: 75,
			IsDefault:   true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		if err := p.CreateLanguageProfile(defaultProfile); err != nil {
			return err
		}
	}
	return nil
}

func mediaPathKey(path string) []byte {
	return []byte("media_path:" + path)
}

func mediaKey(id string) []byte {
	return []byte("media:" + id)
}

func (p *PebbleStore) getMediaByPath(path string) (*MediaItem, string, error) {
	val, closer, err := p.db.Get(mediaPathKey(path))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, "", nil
		}
		return nil, "", err
	}
	closer.Close()
	id := string(val)
	data, closer, err := p.db.Get(mediaKey(id))
	if err != nil {
		return nil, "", err
	}
	defer closer.Close()
	var it MediaItem
	if err := json.Unmarshal(data, &it); err != nil {
		return nil, "", err
	}
	return &it, id, nil
}

// OpenPebble opens a Pebble database at path and returns a PebbleStore.
func OpenPebble(path string) (*PebbleStore, error) {
	db, err := pebble.Open(path, &pebble.Options{})
	if err != nil {
		return nil, err
	}
	return &PebbleStore{db: db}, nil
}

// Close closes the underlying Pebble database.
func (p *PebbleStore) Close() error { return p.db.Close() }

// InsertSubtitle stores a subtitle translation record.
// The ID field of rec will be filled with a generated UUID if empty.
func (p *PebbleStore) InsertSubtitle(rec *SubtitleRecord) error {
	if rec.ID == "" {
		rec.ID = uuid.NewString()
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	key := []byte("subtitle:" + rec.ID)
	return p.db.Set(key, b, pebble.Sync)
}

// ListSubtitles returns all stored subtitle records sorted by creation time
// in descending order.
func (p *PebbleStore) ListSubtitles() ([]SubtitleRecord, error) {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	var recs []SubtitleRecord
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), "subtitle:") {
			continue
		}
		var r SubtitleRecord
		if err := json.Unmarshal(iter.Value(), &r); err != nil {
			return nil, err
		}
		recs = append(recs, r)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].CreatedAt.After(recs[j].CreatedAt)
	})
	return recs, nil
}

// ListSubtitlesByVideo filters subtitle records by video file path.
func (p *PebbleStore) ListSubtitlesByVideo(video string) ([]SubtitleRecord, error) {
	recs, err := p.ListSubtitles()
	if err != nil {
		return nil, err
	}
	out := recs[:0]
	for _, r := range recs {
		if r.VideoFile == video {
			out = append(out, r)
		}
	}
	return out, nil
}

// DeleteSubtitle removes all records matching file from the store.
func (p *PebbleStore) DeleteSubtitle(file string) error {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), "subtitle:") {
			continue
		}
		var r SubtitleRecord
		if err := json.Unmarshal(iter.Value(), &r); err != nil {
			return err
		}
		if r.File == file {
			if err := p.db.Delete(iter.Key(), pebble.Sync); err != nil {
				return err
			}
		}
	}
	return iter.Error()
}

// InsertDownload stores a download record in PebbleDB.
// The ID field is generated when empty.
func (p *PebbleStore) InsertDownload(rec *DownloadRecord) error {
	if rec.ID == "" {
		rec.ID = uuid.NewString()
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	key := []byte("download:" + rec.ID)
	return p.db.Set(key, b, pebble.Sync)
}

// ListDownloads returns all download records sorted by creation time descending.
func (p *PebbleStore) ListDownloads() ([]DownloadRecord, error) {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	var recs []DownloadRecord
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), "download:") {
			continue
		}
		var r DownloadRecord
		if err := json.Unmarshal(iter.Value(), &r); err != nil {
			return nil, err
		}
		recs = append(recs, r)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sort.Slice(recs, func(i, j int) bool {
		return recs[i].CreatedAt.After(recs[j].CreatedAt)
	})
	return recs, nil
}

// ListDownloadsByVideo filters download records by video file path.
func (p *PebbleStore) ListDownloadsByVideo(video string) ([]DownloadRecord, error) {
	recs, err := p.ListDownloads()
	if err != nil {
		return nil, err
	}
	out := recs[:0]
	for _, r := range recs {
		if r.VideoFile == video {
			out = append(out, r)
		}
	}
	return out, nil
}

// DeleteDownload removes download records for the given subtitle file.
func (p *PebbleStore) DeleteDownload(file string) error {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), "download:") {
			continue
		}
		var r DownloadRecord
		if err := json.Unmarshal(iter.Value(), &r); err != nil {
			return err
		}
		if r.File == file {
			if err := p.db.Delete(iter.Key(), pebble.Sync); err != nil {
				return err
			}
		}
	}
	return iter.Error()
}

// InsertMediaItem stores a media item.
func (p *PebbleStore) InsertMediaItem(rec *MediaItem) error {
	if rec.ID == "" {
		rec.ID = uuid.NewString()
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	batch := p.db.NewBatch()
	if err := batch.Set(mediaKey(rec.ID), b, nil); err != nil {
		batch.Close()
		return err
	}
	if err := batch.Set(mediaPathKey(rec.Path), []byte(rec.ID), nil); err != nil {
		batch.Close()
		return err
	}
	return batch.Commit(pebble.Sync)
}

// ListMediaItems returns stored media items sorted by creation time.
func (p *PebbleStore) ListMediaItems() ([]MediaItem, error) {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	var recs []MediaItem
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), "media:") {
			continue
		}
		var r MediaItem
		if err := json.Unmarshal(iter.Value(), &r); err != nil {
			return nil, err
		}
		recs = append(recs, r)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].CreatedAt.After(recs[j].CreatedAt) })
	return recs, nil
}

// DeleteMediaItem removes records with matching path.
func (p *PebbleStore) DeleteMediaItem(path string) error {
	item, id, err := p.getMediaByPath(path)
	if err != nil || item == nil {
		return err
	}
	batch := p.db.NewBatch()
	if err := batch.Delete(mediaKey(id), nil); err != nil {
		batch.Close()
		return err
	}
	if err := batch.Delete(mediaPathKey(path), nil); err != nil {
		batch.Close()
		return err
	}
	return batch.Commit(pebble.Sync)
}

// CountSubtitles returns the number of subtitle records.
func (p *PebbleStore) CountSubtitles() (int, error) {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return 0, err
	}
	defer iter.Close()
	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		if strings.HasPrefix(string(iter.Key()), "subtitle:") {
			count++
		}
	}
	if err := iter.Error(); err != nil {
		return 0, err
	}
	return count, nil
}

// CountDownloads returns the number of download records.
func (p *PebbleStore) CountDownloads() (int, error) {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return 0, err
	}
	defer iter.Close()
	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		if strings.HasPrefix(string(iter.Key()), "download:") {
			count++
		}
	}
	if err := iter.Error(); err != nil {
		return 0, err
	}
	return count, nil
}

// CountMediaItems returns the number of media item records.
func (p *PebbleStore) CountMediaItems() (int, error) {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return 0, err
	}
	defer iter.Close()
	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		if strings.HasPrefix(string(iter.Key()), "media:") {
			count++
		}
	}
	if err := iter.Error(); err != nil {
		return 0, err
	}
	return count, nil
}

// SetMediaReleaseGroup stores the release group in the media item record.
func (p *PebbleStore) SetMediaReleaseGroup(path, group string) error {
	item, _, err := p.getMediaByPath(path)
	if err != nil || item == nil {
		return err
	}
	item.ReleaseGroup = group
	return p.InsertMediaItem(item)
}

// SetMediaAltTitles stores alternate titles in the media item record.
func (p *PebbleStore) SetMediaAltTitles(path string, titles []string) error {
	item, _, err := p.getMediaByPath(path)
	if err != nil || item == nil {
		return err
	}
	data, err := json.Marshal(titles)
	if err != nil {
		return err
	}
	item.AltTitles = string(data)
	return p.InsertMediaItem(item)
}

// SetMediaFieldLocks stores locked fields in the media item record.
func (p *PebbleStore) SetMediaFieldLocks(path, locks string) error {
	item, _, err := p.getMediaByPath(path)
	if err != nil || item == nil {
		return err
	}
	item.FieldLocks = locks
	return p.InsertMediaItem(item)
}

// GetMediaReleaseGroup retrieves the release group for a media item.
func (p *PebbleStore) GetMediaReleaseGroup(path string) (string, error) {
	item, _, err := p.getMediaByPath(path)
	if err != nil || item == nil {
		return "", err
	}
	return item.ReleaseGroup, nil
}

// GetMediaAltTitles retrieves alternate titles for a media item.
func (p *PebbleStore) GetMediaAltTitles(path string) ([]string, error) {
	item, _, err := p.getMediaByPath(path)
	if err != nil || item == nil {
		return nil, err
	}
	if item.AltTitles == "" {
		return nil, nil
	}
	var titles []string
	if err := json.Unmarshal([]byte(item.AltTitles), &titles); err != nil {
		return nil, err
	}
	return titles, nil
}

// GetMediaFieldLocks retrieves locked fields for a media item.
func (p *PebbleStore) GetMediaFieldLocks(path string) (string, error) {
	item, _, err := p.getMediaByPath(path)
	if err != nil || item == nil {
		return "", err
	}
	return item.FieldLocks, nil
}

// SetMediaTitle updates the title in the media item record.
func (p *PebbleStore) SetMediaTitle(path, title string) error {
	item, _, err := p.getMediaByPath(path)
	if err != nil || item == nil {
		return err
	}
	item.Title = title
	return p.InsertMediaItem(item)
}

// GetMediaItem retrieves a media item by path. Returns nil if not found.
func (p *PebbleStore) GetMediaItem(path string) (*MediaItem, error) {
	item, _, err := p.getMediaByPath(path)
	if err != nil {
		return nil, err
	}
	return item, nil
}

// ==================== USER AUTHENTICATION FUNCTIONS ====================

// UserWithPassword represents an account in the system with password hash for Pebble storage.
type UserWithPassword struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// ToUser converts UserWithPassword to common.User.
func (u UserWithPassword) ToUser() *common.User {
	user := &common.User{}
	user.SetId(u.ID)
	user.SetUsername(u.Username)
	user.SetEmail(u.Email)
	user.SetCreatedAt(timestamppb.New(u.CreatedAt))
	// Note: protobuf User doesn't have Role field - may need to handle roles differently
	return user
}

// Session represents a user session for Pebble storage.
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ApiKey represents an API key for Pebble storage.
type ApiKey struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginToken represents a one-time login token for Pebble storage.
type LoginToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

// Permission represents a role-based permission for Pebble storage.
type Permission struct {
	ID         string `json:"id"`
	Role       string `json:"role"`
	Permission string `json:"permission"`
}

// DashboardPref represents user dashboard preferences for Pebble storage.
type DashboardPref struct {
	UserID    string    `json:"user_id"`
	Layout    string    `json:"layout"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Helper functions for key generation
func userKey(id string) []byte               { return []byte("user:" + id) }
func userUsernameKey(username string) []byte { return []byte("user_username:" + username) }
func userEmailKey(email string) []byte       { return []byte("user_email:" + email) }
func sessionKey(id string) []byte            { return []byte("session:" + id) }
func sessionTokenKey(token string) []byte    { return []byte("session_token:" + token) }
func apiKeyKey(id string) []byte             { return []byte("api_key:" + id) }
func apiKeyValueKey(key string) []byte       { return []byte("api_key_value:" + key) }
func loginTokenKey(id string) []byte         { return []byte("login_token:" + id) }
func loginTokenValueKey(token string) []byte { return []byte("login_token_value:" + token) }
func permissionKey(id string) []byte         { return []byte("permission:" + id) }
func dashboardKey(userID string) []byte      { return []byte("dashboard:" + userID) }
func tagKey(id string) []byte                { return []byte("tag:" + id) }
func tagNameKey(name string) []byte          { return []byte("tag_name:" + name) }
func tagAssocKey(tagID, entityType, entityID string) []byte {
	return []byte("tag_assoc:" + tagID + ":" + entityType + ":" + entityID)
}

// CreateUser inserts a new user with hashed password.
func (p *PebbleStore) CreateUser(username, passwordHash, email, role string) (string, error) {
	id := uuid.NewString()
	user := UserWithPassword{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		Email:        email,
		Role:         role,
		CreatedAt:    time.Now(),
	}

	userData, err := json.Marshal(user)
	if err != nil {
		return "", err
	}

	batch := p.db.NewBatch()
	if err := batch.Set(userKey(id), userData, nil); err != nil {
		batch.Close()
		return "", err
	}
	if err := batch.Set(userUsernameKey(username), []byte(id), nil); err != nil {
		batch.Close()
		return "", err
	}
	if email != "" {
		if err := batch.Set(userEmailKey(email), []byte(id), nil); err != nil {
			batch.Close()
			return "", err
		}
	}
	if err := batch.Commit(pebble.Sync); err != nil {
		return "", err
	}
	return id, nil
}

// GetUserByUsername retrieves a user by username.
func (p *PebbleStore) GetUserByUsername(username string) (*common.User, error) {
	val, closer, err := p.db.Get(userUsernameKey(username))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	closer.Close()

	id := string(val)
	userData, closer, err := p.db.Get(userKey(id))
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	var user UserWithPassword
	if err := json.Unmarshal(userData, &user); err != nil {
		return nil, err
	}
	return user.ToUser(), nil
}

// GetUserByEmail retrieves a user by email address.
func (p *PebbleStore) GetUserByEmail(email string) (*common.User, error) {
	val, closer, err := p.db.Get(userEmailKey(email))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	closer.Close()

	id := string(val)
	userData, closer, err := p.db.Get(userKey(id))
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	var user UserWithPassword
	if err := json.Unmarshal(userData, &user); err != nil {
		return nil, err
	}
	return user.ToUser(), nil
}

// GetUserByID retrieves a user by ID.
func (p *PebbleStore) GetUserByID(id string) (*common.User, error) {
	userData, closer, err := p.db.Get(userKey(id))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var user UserWithPassword
	if err := json.Unmarshal(userData, &user); err != nil {
		return nil, err
	}
	return user.ToUser(), nil
}

// ListUsers returns all users sorted by creation time.
func (p *PebbleStore) ListUsers() ([]*common.User, error) {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var users []*common.User
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), "user:") {
			continue
		}
		var user UserWithPassword
		if err := json.Unmarshal(iter.Value(), &user); err != nil {
			return nil, err
		}
		users = append(users, user.ToUser())
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}

	sort.Slice(users, func(i, j int) bool {
		return users[i].GetCreatedAt().AsTime().Before(users[j].GetCreatedAt().AsTime())
	})
	return users, nil
}

// UpdateUserRole updates a user's role.
func (p *PebbleStore) UpdateUserRole(username, role string) error {
	// Get ID first
	val, closer, err := p.db.Get(userUsernameKey(username))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil
		}
		return err
	}
	closer.Close()

	id := string(val)
	user, err := p.getUserWithPasswordByID(id)
	if err != nil || user == nil {
		return err
	}

	user.Role = role
	userData, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return p.db.Set(userKey(user.ID), userData, pebble.Sync)
}

// getUserWithPasswordByID retrieves the internal UserWithPassword by ID.
func (p *PebbleStore) getUserWithPasswordByID(id string) (*UserWithPassword, error) {
	userData, closer, err := p.db.Get(userKey(id))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var user UserWithPassword
	if err := json.Unmarshal(userData, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUserPassword updates a user's password hash.
func (p *PebbleStore) UpdateUserPassword(userID, passwordHash string) error {
	user, err := p.getUserWithPasswordByID(userID)
	if err != nil || user == nil {
		return err
	}

	user.PasswordHash = passwordHash
	userData, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return p.db.Set(userKey(userID), userData, pebble.Sync)
}

// CreateSession creates a new user session.
func (p *PebbleStore) CreateSession(userID, token string, duration time.Duration) error {
	id := uuid.NewString()
	session := Session{
		ID:        id,
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(duration),
		CreatedAt: time.Now(),
	}

	sessionData, err := json.Marshal(session)
	if err != nil {
		return err
	}

	batch := p.db.NewBatch()
	if err := batch.Set(sessionKey(id), sessionData, nil); err != nil {
		batch.Close()
		return err
	}
	if err := batch.Set(sessionTokenKey(token), []byte(id), nil); err != nil {
		batch.Close()
		return err
	}
	return batch.Commit(pebble.Sync)
}

// ValidateSession validates a session token and returns the user ID.
func (p *PebbleStore) ValidateSession(token string) (string, error) {
	val, closer, err := p.db.Get(sessionTokenKey(token))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	closer.Close()

	sessionID := string(val)
	sessionData, closer, err := p.db.Get(sessionKey(sessionID))
	if err != nil {
		return "", err
	}
	defer closer.Close()

	var session Session
	if err := json.Unmarshal(sessionData, &session); err != nil {
		return "", err
	}

	if time.Now().After(session.ExpiresAt) {
		// Session expired, clean it up
		if err := p.InvalidateSession(token); err != nil {
			// Optionally log or handle the error
			return "", err
		}
		return "", nil
	}

	return session.UserID, nil
}

// InvalidateSession removes a session token.
func (p *PebbleStore) InvalidateSession(token string) error {
	val, closer, err := p.db.Get(sessionTokenKey(token))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil // Already gone
		}
		return err
	}
	closer.Close()

	sessionID := string(val)
	batch := p.db.NewBatch()
	if err := batch.Delete(sessionKey(sessionID), nil); err != nil {
		batch.Close()
		return err
	}
	if err := batch.Delete(sessionTokenKey(token), nil); err != nil {
		batch.Close()
		return err
	}
	return batch.Commit(pebble.Sync)
}

// InvalidateUserSessions removes all sessions for a user.
func (p *PebbleStore) InvalidateUserSessions(userID string) error {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	batch := p.db.NewBatch()
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), "session:") {
			continue
		}
		var session Session
		if err := json.Unmarshal(iter.Value(), &session); err != nil {
			continue
		}
		if session.UserID == userID {
			if err := batch.Delete(iter.Key(), nil); err != nil {
				batch.Close()
				return err
			}
			if err := batch.Delete(sessionTokenKey(session.Token), nil); err != nil {
				batch.Close()
				return err
			}
		}
	}
	return batch.Commit(pebble.Sync)
}

// CleanupExpiredSessions removes expired sessions.
func (p *PebbleStore) CleanupExpiredSessions() error {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	now := time.Now()
	batch := p.db.NewBatch()
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), "session:") {
			continue
		}
		var session Session
		if err := json.Unmarshal(iter.Value(), &session); err != nil {
			continue
		}
		if now.After(session.ExpiresAt) {
			if err := batch.Delete(iter.Key(), nil); err != nil {
				batch.Close()
				return err
			}
			if err := batch.Delete(sessionTokenKey(session.Token), nil); err != nil {
				batch.Close()
				return err
			}
		}
	}
	return batch.Commit(pebble.Sync)
}

// ValidateOneTimeToken validates a one-time token (placeholder implementation).
func (p *PebbleStore) ValidateOneTimeToken(token string) (string, error) {
	// TODO: Implement one-time token storage and validation
	// For now, return empty string to indicate invalid token
	return "", nil
}

// CreateAPIKey creates a new API key for a user.
func (p *PebbleStore) CreateAPIKey(userID, key string) error {
	id := uuid.NewString()
	apiKey := ApiKey{
		ID:        id,
		UserID:    userID,
		Key:       key,
		CreatedAt: time.Now(),
	}

	keyData, err := json.Marshal(apiKey)
	if err != nil {
		return err
	}

	batch := p.db.NewBatch()
	if err := batch.Set(apiKeyKey(id), keyData, nil); err != nil {
		batch.Close()
		return err
	}
	if err := batch.Set(apiKeyValueKey(key), []byte(userID), nil); err != nil {
		batch.Close()
		return err
	}
	return batch.Commit(pebble.Sync)
}

// ValidateAPIKey validates an API key and returns the user ID.
func (p *PebbleStore) ValidateAPIKey(key string) (string, error) {
	val, closer, err := p.db.Get(apiKeyValueKey(key))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	defer closer.Close()

	return string(val), nil
}

// CreateOneTimeToken creates a one-time login token.
func (p *PebbleStore) CreateOneTimeToken(userID, token string, duration time.Duration) error {
	id := uuid.NewString()
	loginToken := LoginToken{
		ID:        id,
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(duration),
		Used:      false,
		CreatedAt: time.Now(),
	}

	tokenData, err := json.Marshal(loginToken)
	if err != nil {
		return err
	}

	batch := p.db.NewBatch()
	if err := batch.Set(loginTokenKey(id), tokenData, nil); err != nil {
		batch.Close()
		return err
	}
	if err := batch.Set(loginTokenValueKey(token), []byte(id), nil); err != nil {
		batch.Close()
		return err
	}
	return batch.Commit(pebble.Sync)
}

// ConsumeOneTimeToken validates and marks a token as used.
func (p *PebbleStore) ConsumeOneTimeToken(token string) (string, error) {
	val, closer, err := p.db.Get(loginTokenValueKey(token))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	closer.Close()

	tokenID := string(val)
	tokenData, closer, err := p.db.Get(loginTokenKey(tokenID))
	if err != nil {
		return "", err
	}
	defer closer.Close()

	var loginToken LoginToken
	if err := json.Unmarshal(tokenData, &loginToken); err != nil {
		return "", err
	}

	if loginToken.Used || time.Now().After(loginToken.ExpiresAt) {
		return "", nil
	}

	loginToken.Used = true
	updatedData, err := json.Marshal(loginToken)
	if err != nil {
		return "", err
	}

	if err := p.db.Set(loginTokenKey(tokenID), updatedData, pebble.Sync); err != nil {
		return "", err
	}

	return loginToken.UserID, nil
}

// SetDashboardLayout sets the dashboard layout for a user.
func (p *PebbleStore) SetDashboardLayout(userID, layout string) error {
	pref := DashboardPref{
		UserID:    userID,
		Layout:    layout,
		UpdatedAt: time.Now(),
	}

	prefData, err := json.Marshal(pref)
	if err != nil {
		return err
	}

	return p.db.Set(dashboardKey(userID), prefData, pebble.Sync)
}

// GetDashboardLayout gets the dashboard layout for a user.
func (p *PebbleStore) GetDashboardLayout(userID string) (string, error) {
	prefData, closer, err := p.db.Get(dashboardKey(userID))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	defer closer.Close()

	var pref DashboardPref
	if err := json.Unmarshal(prefData, &pref); err != nil {
		return "", err
	}

	return pref.Layout, nil
}

// ==================== ENHANCED TAG SYSTEM ====================

// InsertTag creates a new tag (overrides the stub implementation).
func (p *PebbleStore) InsertTag(name string) error {
	return p.CreateTag(name, "user", "all", "", "")
}

// CreateTag creates a new tag with full metadata.
func (p *PebbleStore) CreateTag(name, tagType, entityType, color, description string) error {
	id := uuid.NewString()
	tag := Tag{
		ID:          id,
		Name:        name,
		Type:        tagType,
		EntityType:  entityType,
		Color:       color,
		Description: description,
		CreatedAt:   time.Now(),
	}

	tagData, err := json.Marshal(tag)
	if err != nil {
		return err
	}

	batch := p.db.NewBatch()
	if err := batch.Set(tagKey(id), tagData, nil); err != nil {
		batch.Close()
		return err
	}
	if err := batch.Set(tagNameKey(name), []byte(id), nil); err != nil {
		batch.Close()
		return err
	}
	return batch.Commit(pebble.Sync)
}

// ListTags returns all tags (overrides the stub implementation).
func (p *PebbleStore) ListTags() ([]Tag, error) {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var tags []Tag
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), "tag:") {
			continue
		}
		var tag Tag
		if err := json.Unmarshal(iter.Value(), &tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})
	return tags, nil
}

// GetTagByName retrieves a tag by name.
func (p *PebbleStore) GetTagByName(name string) (*Tag, error) {
	val, closer, err := p.db.Get(tagNameKey(name))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	closer.Close()

	id := string(val)
	tagData, closer, err := p.db.Get(tagKey(id))
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	var tag Tag
	if err := json.Unmarshal(tagData, &tag); err != nil {
		return nil, err
	}
	return &tag, nil
}

// UpdateTag updates an existing tag.
func (p *PebbleStore) UpdateTag(id int64, name string) error {
	// For legacy compatibility, we need to store a mapping of int64 IDs to string UUIDs
	// First, find all tags and check which one corresponds to this int64 ID
	tags, err := p.ListTags()
	if err != nil {
		return err
	}

	// Find the tag by matching the legacy ID (we'll use array index for now)
	if id <= 0 || int(id-1) >= len(tags) {
		return errors.New("tag not found")
	}

	tag := tags[id-1] // Use array index as legacy ID
	oldName := tag.Name

	// Remove old name mapping
	if err := p.db.Delete(tagNameKey(oldName), nil); err != nil {
		return err
	}

	tag.Name = name
	updatedData, err := json.Marshal(tag)
	if err != nil {
		return err
	}

	batch := p.db.NewBatch()
	if err := batch.Set(tagKey(tag.ID), updatedData, nil); err != nil {
		batch.Close()
		return err
	}
	if err := batch.Set(tagNameKey(name), []byte(tag.ID), nil); err != nil {
		batch.Close()
		return err
	}
	return batch.Commit(pebble.Sync)
}

// DeleteTag removes a tag and all its associations.
func (p *PebbleStore) DeleteTag(id int64) error {
	// For legacy compatibility, find tag by index
	tags, err := p.ListTags()
	if err != nil {
		return err
	}

	if id <= 0 || int(id-1) >= len(tags) {
		return errors.New("tag not found")
	}

	tag := tags[id-1] // Use array index as legacy ID
	tagID := tag.ID

	// Remove all associations for this tag
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	batch := p.db.NewBatch()
	prefix := "tag_assoc:" + tagID + ":"
	for iter.First(); iter.Valid(); iter.Next() {
		if strings.HasPrefix(string(iter.Key()), prefix) {
			if err := batch.Delete(iter.Key(), nil); err != nil {
				batch.Close()
				return err
			}
		}
	}

	// Remove tag and name mapping
	if err := batch.Delete(tagKey(tagID), nil); err != nil {
		batch.Close()
		return err
	}
	if err := batch.Delete(tagNameKey(tag.Name), nil); err != nil {
		batch.Close()
		return err
	}

	return batch.Commit(pebble.Sync)
}

// AssignTagToEntity creates a tag association.
func (p *PebbleStore) AssignTagToEntity(tagID, entityType, entityID string) error {
	assoc := TagAssociation{
		TagID:      tagID,
		EntityType: entityType,
		EntityID:   entityID,
		CreatedAt:  time.Now(),
	}

	assocData, err := json.Marshal(assoc)
	if err != nil {
		return err
	}

	key := tagAssocKey(tagID, entityType, entityID)
	return p.db.Set(key, assocData, pebble.Sync)
}

// RemoveTagFromEntity removes a tag association.
func (p *PebbleStore) RemoveTagFromEntity(tagID, entityType, entityID string) error {
	key := tagAssocKey(tagID, entityType, entityID)
	return p.db.Delete(key, pebble.Sync)
}

// ListTagsForEntity returns tags associated with an entity.
func (p *PebbleStore) ListTagsForEntity(entityType, entityID string) ([]Tag, error) {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var tagIDs []string
	prefix := "tag_assoc:"
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), prefix) {
			continue
		}
		var assoc TagAssociation
		if err := json.Unmarshal(iter.Value(), &assoc); err != nil {
			continue
		}
		if assoc.EntityType == entityType && assoc.EntityID == entityID {
			tagIDs = append(tagIDs, assoc.TagID)
		}
	}

	var tags []Tag
	for _, tagID := range tagIDs {
		tagData, closer, err := p.db.Get(tagKey(tagID))
		if err != nil {
			continue
		}
		var tag Tag
		if err := json.Unmarshal(tagData, &tag); err != nil {
			closer.Close()
			continue
		}
		closer.Close()
		tags = append(tags, tag)
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})
	return tags, nil
}

// Legacy compatibility functions for the existing tag interface

// AssignTagToUser assigns a tag to a user (legacy compatibility).
func (p *PebbleStore) AssignTagToUser(userID, tagID int64) error {
	// Convert legacy int64 IDs to string format
	userIDStr := strconv.FormatInt(userID, 10)

	// For tagID, we need to map from legacy int64 to our UUID strings
	tags, err := p.ListTags()
	if err != nil {
		return err
	}

	if tagID <= 0 || int(tagID-1) >= len(tags) {
		return errors.New("tag not found")
	}

	tag := tags[tagID-1] // Use array index as legacy ID
	return p.AssignTagToEntity(tag.ID, "user", userIDStr)
}

// RemoveTagFromUser removes a tag from a user (legacy compatibility).
func (p *PebbleStore) RemoveTagFromUser(userID, tagID int64) error {
	userIDStr := strconv.FormatInt(userID, 10)

	tags, err := p.ListTags()
	if err != nil {
		return err
	}

	if tagID <= 0 || int(tagID-1) >= len(tags) {
		return errors.New("tag not found")
	}

	tag := tags[tagID-1]
	return p.RemoveTagFromEntity(tag.ID, "user", userIDStr)
}

// ListTagsForUser returns tags for a user (legacy compatibility).
func (p *PebbleStore) ListTagsForUser(userID int64) ([]Tag, error) {
	userIDStr := strconv.FormatInt(userID, 10)
	return p.ListTagsForEntity("user", userIDStr)
}

// AssignTagToMedia assigns a tag to media (legacy compatibility).
func (p *PebbleStore) AssignTagToMedia(mediaID, tagID int64) error {
	mediaIDStr := strconv.FormatInt(mediaID, 10)

	tags, err := p.ListTags()
	if err != nil {
		return err
	}

	if tagID <= 0 || int(tagID-1) >= len(tags) {
		return errors.New("tag not found")
	}

	tag := tags[tagID-1]
	return p.AssignTagToEntity(tag.ID, "media", mediaIDStr)
}

// RemoveTagFromMedia removes a tag from media (legacy compatibility).
func (p *PebbleStore) RemoveTagFromMedia(mediaID, tagID int64) error {
	mediaIDStr := strconv.FormatInt(mediaID, 10)

	tags, err := p.ListTags()
	if err != nil {
		return err
	}

	if tagID <= 0 || int(tagID-1) >= len(tags) {
		return errors.New("tag not found")
	}

	tag := tags[tagID-1]
	return p.RemoveTagFromEntity(tag.ID, "media", mediaIDStr)
}

// ListTagsForMedia returns tags for media (legacy compatibility).
func (p *PebbleStore) ListTagsForMedia(mediaID int64) ([]Tag, error) {
	mediaIDStr := strconv.FormatInt(mediaID, 10)
	return p.ListTagsForEntity("media", mediaIDStr)
}

// ==================== PERMISSION SYSTEM ====================

// CreatePermission creates a new permission.
func (p *PebbleStore) CreatePermission(role, permission string) error {
	id := uuid.NewString()
	perm := Permission{
		ID:         id,
		Role:       role,
		Permission: permission,
	}

	permData, err := json.Marshal(perm)
	if err != nil {
		return err
	}

	return p.db.Set(permissionKey(id), permData, pebble.Sync)
}

// ListPermissions returns all permissions.
func (p *PebbleStore) ListPermissions() ([]Permission, error) {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var permissions []Permission
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), "permission:") {
			continue
		}
		var perm Permission
		if err := json.Unmarshal(iter.Value(), &perm); err != nil {
			return nil, err
		}
		permissions = append(permissions, perm)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}

	return permissions, nil
}

// GetPermissionsForRole returns permissions for a specific role.
func (p *PebbleStore) GetPermissionsForRole(role string) ([]string, error) {
	perms, err := p.ListPermissions()
	if err != nil {
		return nil, err
	}

	var permissions []string
	for _, perm := range perms {
		if perm.Role == role {
			permissions = append(permissions, perm.Permission)
		}
	}

	return permissions, nil
}

// InitializeDefaultPermissions seeds default permissions if none exist.
func (p *PebbleStore) InitializeDefaultPermissions() error {
	perms, err := p.ListPermissions()
	if err != nil {
		return err
	}

	if len(perms) == 0 {
		defaults := []struct{ role, permission string }{
			{"admin", "all"},
			{"user", "read"},
			{"user", "download"},
		}

		for _, def := range defaults {
			if err := p.CreatePermission(def.role, def.permission); err != nil {
				return err
			}
		}
	}

	return nil
}

// Language Profile operations for PebbleStore

func languageProfileKey(id string) []byte   { return []byte("language_profile:" + id) }
func mediaProfileKey(mediaID string) []byte { return []byte("media_profile:" + mediaID) }

// CreateLanguageProfile stores a new language profile.
func (p *PebbleStore) CreateLanguageProfile(profile *LanguageProfile) error {
	profileData, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	return p.db.Set(languageProfileKey(profile.ID), profileData, pebble.Sync)
}

// GetLanguageProfile retrieves a language profile by ID.
func (p *PebbleStore) GetLanguageProfile(id string) (*LanguageProfile, error) {
	profileData, closer, err := p.db.Get(languageProfileKey(id))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	defer closer.Close()

	var profile LanguageProfile
	if err := json.Unmarshal(profileData, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// ListLanguageProfiles retrieves all language profiles.
func (p *PebbleStore) ListLanguageProfiles() ([]LanguageProfile, error) {
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var profiles []LanguageProfile
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), "language_profile:") {
			continue
		}
		var profile LanguageProfile
		if err := json.Unmarshal(iter.Value(), &profile); err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}

	// Sort by default first, then by name
	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].IsDefault != profiles[j].IsDefault {
			return profiles[i].IsDefault
		}
		return profiles[i].Name < profiles[j].Name
	})
	return profiles, nil
}

// UpdateLanguageProfile updates an existing language profile.
func (p *PebbleStore) UpdateLanguageProfile(profile *LanguageProfile) error {
	profile.UpdatedAt = time.Now()
	profileData, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	return p.db.Set(languageProfileKey(profile.ID), profileData, pebble.Sync)
}

// DeleteLanguageProfile removes a language profile by ID.
func (p *PebbleStore) DeleteLanguageProfile(id string) error {
	// First remove any media assignments
	iter, err := p.db.NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	batch := p.db.NewBatch()
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), "media_profile:") {
			continue
		}
		var assignment profilesPkg.MediaProfileAssignment
		if err := json.Unmarshal(iter.Value(), &assignment); err != nil {
			continue
		}
		if assignment.ProfileID == id {
			if err := batch.Delete(iter.Key(), nil); err != nil {
				batch.Close()
				return err
			}
		}
	}

	// Remove the profile itself
	if err := batch.Delete(languageProfileKey(id), nil); err != nil {
		batch.Close()
		return err
	}

	return batch.Commit(pebble.Sync)
}

// SetDefaultLanguageProfile marks a profile as the default.
func (p *PebbleStore) SetDefaultLanguageProfile(id string) error {
	// First get all profiles and clear default flags
	profilesList, err := p.ListLanguageProfiles()
	if err != nil {
		return err
	}

	batch := p.db.NewBatch()
	for _, profile := range profilesList {
		profile.IsDefault = (profile.ID == id)
		profile.UpdatedAt = time.Now()
		profileData, err := json.Marshal(profile)
		if err != nil {
			batch.Close()
			return err
		}
		if err := batch.Set(languageProfileKey(profile.ID), profileData, nil); err != nil {
			batch.Close()
			return err
		}
	}

	return batch.Commit(pebble.Sync)
}

// GetDefaultLanguageProfile retrieves the default language profile.
func (p *PebbleStore) GetDefaultLanguageProfile() (*LanguageProfile, error) {
	profilesList, err := p.ListLanguageProfiles()
	if err != nil {
		return nil, err
	}

	for _, profile := range profilesList {
		if profile.IsDefault {
			return &profile, nil
		}
	}

	// No profile is flagged default; fall back to the first one so callers that
	// merely want "some governing profile" still get an answer.
	if len(profilesList) > 0 {
		return &profilesList[0], nil
	}

	// An empty store used to be answered by *creating* a "default" profile here.
	// That made a read mutate the database, and GetMediaProfile calls this on
	// every miss — so deleting the last profile only lasted until the next
	// lookup, which resurrected it, flagged default, and therefore undeletable
	// again. SQLStore has always reported a miss instead; this now matches it.
	return nil, fmt.Errorf("no language profiles exist")
}

// AssignProfileToMedia assigns a language profile to a media item.
func (p *PebbleStore) AssignProfileToMedia(mediaID, profileID string) error {
	assignment := profilesPkg.MediaProfileAssignment{
		MediaID:   mediaID,
		ProfileID: profileID,
		CreatedAt: time.Now(),
	}

	assignmentData, err := json.Marshal(assignment)
	if err != nil {
		return err
	}

	return p.db.Set(mediaProfileKey(mediaID), assignmentData, pebble.Sync)
}

// RemoveProfileFromMedia removes language profile assignment from a media item.
func (p *PebbleStore) RemoveProfileFromMedia(mediaID string) error {
	return p.db.Delete(mediaProfileKey(mediaID), pebble.Sync)
}

// GetAssignedProfileID reports which profile was explicitly assigned to a media
// item, or "" when it has none. It never falls back to the default and never
// creates rows.
func (p *PebbleStore) GetAssignedProfileID(mediaID string) (string, error) {
	assignmentData, closer, err := p.db.Get(mediaProfileKey(mediaID))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	defer closer.Close()

	var assignment profilesPkg.MediaProfileAssignment
	if err := json.Unmarshal(assignmentData, &assignment); err != nil {
		return "", err
	}
	return assignment.ProfileID, nil
}

// GetMediaProfile retrieves the language profile assigned to a media item.
func (p *PebbleStore) GetMediaProfile(mediaID string) (*LanguageProfile, error) {
	assignmentData, closer, err := p.db.Get(mediaProfileKey(mediaID))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			// No profile assigned, return default profile
			return p.GetDefaultLanguageProfile()
		}
		return nil, err
	}
	defer closer.Close()

	var assignment profilesPkg.MediaProfileAssignment
	if err := json.Unmarshal(assignmentData, &assignment); err != nil {
		return nil, err
	}

	return p.GetLanguageProfile(assignment.ProfileID)
}

// Subtitle Source operations for PebbleStore

// InsertSubtitleSource stores a new subtitle source record.
func (p *PebbleStore) InsertSubtitleSource(src *SubtitleSource) error {
	if src.ID == "" {
		src.ID = uuid.NewString()
	}
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return p.db.Set(subtitleSourceKey(src.SourceHash), data, pebble.Sync)
}

// GetSubtitleSource retrieves a subtitle source by hash.
func (p *PebbleStore) GetSubtitleSource(sourceHash string) (*SubtitleSource, error) {
	data, closer, err := p.db.Get(subtitleSourceKey(sourceHash))
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	var src SubtitleSource
	if err := json.Unmarshal(data, &src); err != nil {
		return nil, err
	}

	return &src, nil
}

// UpdateSubtitleSourceStats updates download statistics for a subtitle source.
func (p *PebbleStore) UpdateSubtitleSourceStats(sourceHash string, downloadCount, successCount int, avgRating *float64) error {
	src, err := p.GetSubtitleSource(sourceHash)
	if err != nil {
		return err
	}

	src.DownloadCount = downloadCount
	src.SuccessCount = successCount
	src.AvgRating = avgRating
	src.LastSeen = time.Now()

	return p.InsertSubtitleSource(src)
}

// ListSubtitleSources retrieves all subtitle sources for a provider.
func (p *PebbleStore) ListSubtitleSources(provider string, limit int) ([]SubtitleSource, error) {
	iter, err := p.db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("subtitle_source:"),
		UpperBound: []byte("subtitle_source;"),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var sources []SubtitleSource
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		if limit > 0 && count >= limit {
			break
		}

		var src SubtitleSource
		if err := json.Unmarshal(iter.Value(), &src); err != nil {
			continue
		}

		if provider == "" || src.Provider == provider {
			sources = append(sources, src)
			count++
		}
	}

	return sources, iter.Error()
}

// DeleteSubtitleSource removes a subtitle source record.
func (p *PebbleStore) DeleteSubtitleSource(sourceHash string) error {
	return p.db.Delete(subtitleSourceKey(sourceHash), pebble.Sync)
}

func subtitleSourceKey(sourceHash string) []byte {
	return []byte("subtitle_source:" + sourceHash)
}

// ==================== MONITORING FUNCTIONS ====================

// InsertMonitoredItem stores a monitored item record.
func (p *PebbleStore) InsertMonitoredItem(rec *MonitoredItem) error {
	if rec.ID == "" {
		rec.ID = uuid.NewString()
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	rec.UpdatedAt = time.Now()

	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return p.db.Set([]byte("monitored:"+rec.ID), b, nil)
}

// ListMonitoredItems retrieves all monitored items.
func (p *PebbleStore) ListMonitoredItems() ([]MonitoredItem, error) {
	iter, err := p.db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("monitored:"),
		UpperBound: []byte("monitored;"),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var items []MonitoredItem
	for iter.First(); iter.Valid(); iter.Next() {
		var item MonitoredItem
		if err := json.Unmarshal(iter.Value(), &item); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, iter.Error()
}

// UpdateMonitoredItem updates an existing monitored item.
func (p *PebbleStore) UpdateMonitoredItem(rec *MonitoredItem) error {
	rec.UpdatedAt = time.Now()
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return p.db.Set([]byte("monitored:"+rec.ID), b, nil)
}

// DeleteMonitoredItem removes a monitored item by ID.
func (p *PebbleStore) DeleteMonitoredItem(id string) error {
	return p.db.Delete([]byte("monitored:"+id), nil)
}

// GetMonitoredItemsToCheck returns items that need monitoring.
func (p *PebbleStore) GetMonitoredItemsToCheck(interval time.Duration) ([]MonitoredItem, error) {
	cutoff := time.Now().Add(-interval)

	iter, err := p.db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("monitored:"),
		UpperBound: []byte("monitored;"),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var items []MonitoredItem
	for iter.First(); iter.Valid(); iter.Next() {
		var item MonitoredItem
		if err := json.Unmarshal(iter.Value(), &item); err != nil {
			continue
		}

		// Check if item needs monitoring
		if (item.Status == "pending" || item.Status == "monitoring" || item.Status == "failed") &&
			item.LastChecked.Before(cutoff) &&
			item.RetryCount < item.MaxRetries {
			items = append(items, item)
		}
	}
	return items, iter.Error()
}
