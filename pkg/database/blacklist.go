// file: pkg/database/blacklist.go
// version: 1.0.0
// guid: 6f3a1c94-8b02-4e57-9d16-2a7c0e5b4831
// last-edited: 2026-07-23

package database

import (
	"database/sql"
	"encoding/json"
	"sort"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/google/uuid"
)

// BlacklistItem is a persisted subtitle blacklist entry. An entry prevents
// re-downloading subtitles for a monitored item (optionally scoped to one
// language) until it is removed or, when ExpiresAt is set, until it expires.
type BlacklistItem struct {
	ID        string     `json:"id"`
	ItemID    string     `json:"item_id"`  // the MonitoredItem this applies to
	Path      string     `json:"path"`     // media path (informational)
	Language  string     `json:"language"` // "" means all languages
	Reason    string     `json:"reason"`
	Details   string     `json:"details"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Expired reports whether the entry has an expiry that has passed as of now.
func (b *BlacklistItem) Expired(now time.Time) bool {
	return b.ExpiresAt != nil && now.After(*b.ExpiresAt)
}

// BlacklistStore is an optional capability implemented by stores that can
// persist a subtitle blacklist with reasons and expiry. It is intentionally
// separate from SubtitleStore so the large mocked interface is untouched;
// callers type-assert to it and fall back gracefully when it is absent.
type BlacklistStore interface {
	// InsertBlacklist stores a blacklist entry, assigning an ID when unset.
	InsertBlacklist(item *BlacklistItem) error
	// ListBlacklist returns all blacklist entries.
	ListBlacklist() ([]BlacklistItem, error)
	// DeleteBlacklist removes an entry by ID.
	DeleteBlacklist(id string) error
	// DeleteBlacklistByItem removes all entries for a monitored item, returning
	// the number removed.
	DeleteBlacklistByItem(itemID string) (int, error)
	// DeleteExpiredBlacklist removes entries whose expiry has passed as of now,
	// returning the number removed.
	DeleteExpiredBlacklist(now time.Time) (int, error)
}

// Compile-time assertions that every concrete store implements BlacklistStore.
var (
	_ BlacklistStore = (*SQLStore)(nil)
	_ BlacklistStore = (*PostgresStore)(nil)
	_ BlacklistStore = (*PebbleStore)(nil)
)

func ensureBlacklistID(item *BlacklistItem) {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
}

// --- SQLStore (sqlite) ---

// InsertBlacklist stores a blacklist entry in the SQLite-backed store.
func (s *SQLStore) InsertBlacklist(item *BlacklistItem) error {
	ensureBlacklistID(item)
	var exp interface{}
	if item.ExpiresAt != nil {
		exp = *item.ExpiresAt
	}
	_, err := s.db.Exec(
		`INSERT INTO blacklist (id, item_id, path, language, reason, details, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.ItemID, item.Path, item.Language, item.Reason, item.Details, item.CreatedAt, exp)
	return err
}

// ListBlacklist returns all blacklist entries from the SQLite store.
func (s *SQLStore) ListBlacklist() ([]BlacklistItem, error) {
	rows, err := s.db.Query(`SELECT id, item_id, path, language, reason, details, created_at, expires_at FROM blacklist`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlacklistRows(rows)
}

// DeleteBlacklist removes a blacklist entry by ID from the SQLite store.
func (s *SQLStore) DeleteBlacklist(id string) error {
	_, err := s.db.Exec(`DELETE FROM blacklist WHERE id = ?`, id)
	return err
}

// DeleteBlacklistByItem removes all entries for itemID from the SQLite store.
func (s *SQLStore) DeleteBlacklistByItem(itemID string) (int, error) {
	res, err := s.db.Exec(`DELETE FROM blacklist WHERE item_id = ?`, itemID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// DeleteExpiredBlacklist removes expired entries from the SQLite store.
func (s *SQLStore) DeleteExpiredBlacklist(now time.Time) (int, error) {
	res, err := s.db.Exec(`DELETE FROM blacklist WHERE expires_at IS NOT NULL AND expires_at < ?`, now)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// --- PostgresStore ---

// InsertBlacklist stores a blacklist entry in the Postgres-backed store.
func (p *PostgresStore) InsertBlacklist(item *BlacklistItem) error {
	ensureBlacklistID(item)
	var exp interface{}
	if item.ExpiresAt != nil {
		exp = *item.ExpiresAt
	}
	_, err := p.db.Exec(
		`INSERT INTO blacklist (id, item_id, path, language, reason, details, created_at, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		item.ID, item.ItemID, item.Path, item.Language, item.Reason, item.Details, item.CreatedAt, exp)
	return err
}

// ListBlacklist returns all blacklist entries from the Postgres store.
func (p *PostgresStore) ListBlacklist() ([]BlacklistItem, error) {
	rows, err := p.db.Query(`SELECT id, item_id, path, language, reason, details, created_at, expires_at FROM blacklist`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBlacklistRows(rows)
}

// DeleteBlacklist removes a blacklist entry by ID from the Postgres store.
func (p *PostgresStore) DeleteBlacklist(id string) error {
	_, err := p.db.Exec(`DELETE FROM blacklist WHERE id = $1`, id)
	return err
}

// DeleteBlacklistByItem removes all entries for itemID from the Postgres store.
func (p *PostgresStore) DeleteBlacklistByItem(itemID string) (int, error) {
	res, err := p.db.Exec(`DELETE FROM blacklist WHERE item_id = $1`, itemID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// DeleteExpiredBlacklist removes expired entries from the Postgres store.
func (p *PostgresStore) DeleteExpiredBlacklist(now time.Time) (int, error) {
	res, err := p.db.Exec(`DELETE FROM blacklist WHERE expires_at IS NOT NULL AND expires_at < $1`, now)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// scanBlacklistRows converts SQL rows into BlacklistItem values, shared by the
// SQLite and Postgres stores.
func scanBlacklistRows(rows *sql.Rows) ([]BlacklistItem, error) {
	var items []BlacklistItem
	for rows.Next() {
		var it BlacklistItem
		var exp sql.NullTime
		if err := rows.Scan(&it.ID, &it.ItemID, &it.Path, &it.Language, &it.Reason, &it.Details, &it.CreatedAt, &exp); err != nil {
			return nil, err
		}
		if exp.Valid {
			t := exp.Time
			it.ExpiresAt = &t
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// --- PebbleStore ---

const blacklistPrefix = "blacklist:"

// prefixUpperBound returns the exclusive upper-bound key for iterating every
// key sharing prefix, so a Pebble range scan touches only the matching rows
// instead of the whole keyspace. (Pattern adopted from audiobook-organizer's
// pebble store.)
func prefixUpperBound(prefix string) []byte {
	b := []byte(prefix)
	upper := make([]byte, len(b))
	copy(upper, b)
	upper[len(upper)-1]++
	return upper
}

// InsertBlacklist stores a blacklist entry in the Pebble-backed store.
func (p *PebbleStore) InsertBlacklist(item *BlacklistItem) error {
	ensureBlacklistID(item)
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}
	return p.db.Set([]byte(blacklistPrefix+item.ID), b, pebble.Sync)
}

// ListBlacklist returns all blacklist entries from the Pebble store using a
// bounded prefix scan.
func (p *PebbleStore) ListBlacklist() ([]BlacklistItem, error) {
	iter, err := p.db.NewIter(&pebble.IterOptions{
		LowerBound: []byte(blacklistPrefix),
		UpperBound: prefixUpperBound(blacklistPrefix),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	var items []BlacklistItem
	for iter.First(); iter.Valid(); iter.Next() {
		var it BlacklistItem
		if err := json.Unmarshal(iter.Value(), &it); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

// DeleteBlacklist removes a blacklist entry by ID from the Pebble store.
func (p *PebbleStore) DeleteBlacklist(id string) error {
	return p.db.Delete([]byte(blacklistPrefix+id), pebble.Sync)
}

// DeleteBlacklistByItem removes all entries for itemID from the Pebble store.
func (p *PebbleStore) DeleteBlacklistByItem(itemID string) (int, error) {
	items, err := p.ListBlacklist()
	if err != nil {
		return 0, err
	}
	n := 0
	for _, it := range items {
		if it.ItemID == itemID {
			if err := p.DeleteBlacklist(it.ID); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

// DeleteExpiredBlacklist removes expired entries from the Pebble store.
func (p *PebbleStore) DeleteExpiredBlacklist(now time.Time) (int, error) {
	items, err := p.ListBlacklist()
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range items {
		if items[i].Expired(now) {
			if err := p.DeleteBlacklist(items[i].ID); err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}
