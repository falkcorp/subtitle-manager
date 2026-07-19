// file: pkg/gcommonauth/auth.go
// version: 2.0.0
// guid: ec8f6814-42cc-41ab-956a-a0ee798664a4
package gcommonauth

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/google/uuid"
	common "github.com/falkcorp/gcommon/v2/pkg/commonpb/v2"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateUser inserts a new user with a hashed password.
// CreateUser inserts a new user with a hashed password and automatically
// generates an API key for the user. The generated API key is stored in the
// database but not returned to the caller.
func CreateUser(db *sql.DB, username, password, email, role string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	res, err := db.Exec(`INSERT INTO users (username, password_hash, email, role, created_at) VALUES (?, ?, ?, ?, ?)`,
		username, string(hash), email, role, time.Now())
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	_, err = GenerateAPIKey(db, id)
	return err
}

// AuthenticateUser verifies credentials and returns the user ID.
func AuthenticateUser(db *sql.DB, username, password string) (int64, error) {
	var id int64
	var hash string
	row := db.QueryRow(`SELECT id, password_hash FROM users WHERE username = ?`, username)
	if err := row.Scan(&id, &hash); err != nil {
		return 0, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return 0, err
	}
	return id, nil
}

// GenerateSession creates a new session token for the user and returns a gcommon Session.
func GenerateSession(db *sql.DB, userID int64, duration time.Duration) (*common.Session, error) {
	token := uuid.NewString()
	expires := time.Now().Add(duration)
	createdAt := time.Now()
	_, err := db.Exec(`INSERT INTO sessions (user_id, token, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		userID, token, expires, createdAt)
	if err != nil {
		return nil, err
	}

	// Create gcommon Session with opaque API
	session := &common.Session{}
	session.SetId(token) // Use token as session ID
	session.SetUserId(strconv.FormatInt(userID, 10))
	session.SetCreatedAt(timestamppb.New(createdAt))
	session.SetExpiresAt(timestamppb.New(expires))
	session.SetStatus(common.SessionStatus_SESSION_STATUS_ACTIVE)

	return session, nil
}

// ValidateSession returns a gcommon Session for a valid session token.
func ValidateSession(db *sql.DB, token string) (*common.Session, error) {
	var userID int64
	var expires, createdAt time.Time
	row := db.QueryRow(`SELECT user_id, expires_at, created_at FROM sessions WHERE token = ?`, token)
	if err := row.Scan(&userID, &expires, &createdAt); err != nil {
		return nil, err
	}
	if time.Now().After(expires) {
		return nil, sql.ErrNoRows
	}

	// Create gcommon Session with opaque API
	session := &common.Session{}
	session.SetId(token)
	session.SetUserId(strconv.FormatInt(userID, 10))
	session.SetCreatedAt(timestamppb.New(createdAt))
	session.SetExpiresAt(timestamppb.New(expires))
	session.SetStatus(common.SessionStatus_SESSION_STATUS_ACTIVE)

	return session, nil
}

// GenerateAPIKey creates a new API key for the user and returns a gcommon APIKey.
func GenerateAPIKey(db *sql.DB, userID int64) (*common.APIKey, error) {
	key := uuid.NewString()
	createdAt := time.Now()
	_, err := db.Exec(`INSERT INTO api_keys (user_id, key, created_at) VALUES (?, ?, ?)`,
		userID, key, createdAt)
	if err != nil {
		return nil, err
	}

	// Create gcommon APIKey with opaque API
	apiKey := &common.APIKey{}
	apiKey.SetId(key)      // Use key as ID
	apiKey.SetKeyHash(key) // In a real implementation, this should be a hash
	apiKey.SetUserId(strconv.FormatInt(userID, 10))
	apiKey.SetCreatedAt(timestamppb.New(createdAt))
	apiKey.SetActive(true)

	return apiKey, nil
}

// ValidateAPIKey returns a gcommon APIKey for a valid API key.
func ValidateAPIKey(db *sql.DB, key string) (*common.APIKey, error) {
	var userID int64
	var createdAt time.Time
	row := db.QueryRow(`SELECT user_id, created_at FROM api_keys WHERE key = ?`, key)
	if err := row.Scan(&userID, &createdAt); err != nil {
		return nil, err
	}

	// Create gcommon APIKey with opaque API
	apiKey := &common.APIKey{}
	apiKey.SetId(key)      // Use key as ID
	apiKey.SetKeyHash(key) // In a real implementation, this should be a hash
	apiKey.SetUserId(strconv.FormatInt(userID, 10))
	apiKey.SetCreatedAt(timestamppb.New(createdAt))
	apiKey.SetActive(true)

	return apiKey, nil
}

// ResetPassword generates a new password for the specified user ID, updates the
// stored hash and returns the plaintext password. A new API key is also
// generated and returned as a string.
func ResetPassword(db *sql.DB, userID int64) (string, string, error) {
	pass := uuid.NewString()[:12]
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	if _, err = db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, string(hash), userID); err != nil {
		return "", "", err
	}
	apiKeyObj, err := GenerateAPIKey(db, userID)
	if err != nil {
		return "", "", err
	}
	// Extract the key string from the gcommon APIKey object
	keyString := apiKeyObj.GetId() // Use ID as the key string
	return pass, keyString, nil
}

// GetOrCreateUser returns the existing user ID for email or inserts a new user
// with the provided username, email and role if none exists. The password is
// left empty for OAuth2 users.
func GetOrCreateUser(db *sql.DB, username, email, role string) (int64, error) {
	var id int64
	row := db.QueryRow(`SELECT id FROM users WHERE email = ?`, email)
	err := row.Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	res, err := db.Exec(`INSERT INTO users (username, password_hash, email, role, created_at) VALUES (?, '', ?, ?, ?)`,
		username, email, role, time.Now())
	if err != nil {
		return 0, err
	}
	id, err = res.LastInsertId()
	return id, err
}

// SetUserRole updates the role for the specified username.
func SetUserRole(db *sql.DB, username, role string) error {
	_, err := db.Exec(`UPDATE users SET role = ? WHERE username = ?`, role, username)
	return err
}

// GenerateOneTimeToken creates a single-use login token for the user.
// The token expires after the provided duration.
func GenerateOneTimeToken(db *sql.DB, userID int64, duration time.Duration) (string, error) {
	token := uuid.NewString()
	expires := time.Now().Add(duration)
	_, err := db.Exec(`INSERT INTO login_tokens (user_id, token, expires_at, used, created_at) VALUES (?, ?, ?, 0, ?)`,
		userID, token, expires, time.Now())
	if err != nil {
		return "", err
	}
	return token, nil
}

// ConsumeOneTimeToken validates and marks the token as used.
// It returns the associated user ID when successful.
func ConsumeOneTimeToken(db *sql.DB, token string) (int64, error) {
	var userID int64
	var expires time.Time
	var used int
	row := db.QueryRow(`SELECT user_id, expires_at, used FROM login_tokens WHERE token = ?`, token)
	if err := row.Scan(&userID, &expires, &used); err != nil {
		return 0, err
	}
	if used == 1 || time.Now().After(expires) {
		return 0, sql.ErrNoRows
	}
	if _, err := db.Exec(`UPDATE login_tokens SET used = 1 WHERE token = ?`, token); err != nil {
		return 0, err
	}
	return userID, nil
}

// InvalidateSession removes a session token from the database, effectively logging out the user.
func InvalidateSession(db *sql.DB, token string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

// InvalidateUserSessions removes all session tokens for a specific user.
// This is useful for force-logout scenarios or when a user's permissions change.
func InvalidateUserSessions(db *sql.DB, userID int64) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

// CleanupExpiredSessions removes all expired session tokens from the database.
// This should be called periodically to prevent the sessions table from growing indefinitely.
func CleanupExpiredSessions(db *sql.DB) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now())
	return err
}

// User represents an account in the system. ID is stored as a string for
// ListUsers returns all users ordered by ID using gcommon User types.
func ListUsers(db *sql.DB) ([]*common.User, error) {
	rows, err := db.Query(`SELECT id, username, email, role, created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*common.User
	for rows.Next() {
		u := &common.User{}
		var id int64
		var username, email, role string
		var createdAt time.Time
		if err := rows.Scan(&id, &username, &email, &role, &createdAt); err != nil {
			return nil, err
		}
		// Use opaque API to set fields
		u.SetId(strconv.FormatInt(id, 10))
		u.SetUsername(username)
		u.SetEmail(email)
		u.SetCreatedAt(timestamppb.New(createdAt))

		// Store role in metadata since User doesn't have a role field
		metadata := make(map[string]string)
		metadata["role"] = role
		u.SetMetadata(metadata)

		out = append(out, u)
	}
	return out, rows.Err()
}
