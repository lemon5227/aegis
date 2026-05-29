// db_personal.go - Personal client-side features (not synced via P2P)
//
// This file implements features that are local to a single device and do NOT
// participate in P2P consensus or governance:
//
//   - Personal mute users: client-side filtering of unwanted users without
//     participating in the network's shadow-ban governance.
//   - Post read tracking: marking posts as read/unread for the local user.
//   - User preferences: arbitrary key-value preferences (theme, layout, etc).
//
// Unlike governance bans (which propagate via libp2p pubsub and require
// trusted-admin signatures), all data here stays on the local node.
package main

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// MutedUser represents a personally muted user (not a network-wide ban).
type MutedUser struct {
	Pubkey    string `json:"pubkey"`
	Reason    string `json:"reason"`
	CreatedAt int64  `json:"createdAt"`
}

// UserPreference represents a single key-value preference entry.
type UserPreference struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt int64  `json:"updatedAt"`
}

// MuteUser adds a pubkey to the local mute list. Idempotent - existing entries
// are updated with the new reason and timestamp.
func (a *App) MuteUser(pubkey string, reason string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return errors.New("pubkey is required")
	}

	// Validate reason length to prevent abuse
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) > 200 {
		reason = string([]rune(reason)[:200])
	}

	a.dbMu.Lock()
	defer a.dbMu.Unlock()

	_, err := a.db.Exec(`
		INSERT INTO muted_users (pubkey, reason, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(pubkey) DO UPDATE SET
			reason = excluded.reason,
			created_at = excluded.created_at;
	`, pubkey, reason, time.Now().Unix())
	if err != nil {
		return err
	}

	a.emitEvent("mute:updated")
	return nil
}

// UnmuteUser removes a pubkey from the local mute list. Idempotent - returns
// nil if the user was not muted.
func (a *App) UnmuteUser(pubkey string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return errors.New("pubkey is required")
	}

	a.dbMu.Lock()
	defer a.dbMu.Unlock()

	if _, err := a.db.Exec(`DELETE FROM muted_users WHERE pubkey = ?;`, pubkey); err != nil {
		return err
	}

	a.emitEvent("mute:updated")
	return nil
}

// IsMuted returns true if the given pubkey is on the local mute list.
func (a *App) IsMuted(pubkey string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}
	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return false, nil
	}

	var count int
	err := a.db.QueryRow(`SELECT COUNT(1) FROM muted_users WHERE pubkey = ?;`, pubkey).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetMutedUsers returns all muted users sorted by most-recently muted first.
func (a *App) GetMutedUsers() ([]MutedUser, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT pubkey, reason, created_at
		FROM muted_users
		ORDER BY created_at DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]MutedUser, 0)
	for rows.Next() {
		var m MutedUser
		if err := rows.Scan(&m.Pubkey, &m.Reason, &m.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// GetMutedPubkeys returns just the pubkeys of muted users as a set-like map.
// Useful for fast in-memory filtering of feed results.
func (a *App) GetMutedPubkeys() (map[string]struct{}, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`SELECT pubkey FROM muted_users;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]struct{})
	for rows.Next() {
		var pubkey string
		if err := rows.Scan(&pubkey); err != nil {
			return nil, err
		}
		result[pubkey] = struct{}{}
	}
	return result, rows.Err()
}

// MarkPostRead records that the local user has read a post. Idempotent -
// re-reading just updates the read timestamp.
func (a *App) MarkPostRead(postID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	postID = strings.TrimSpace(postID)
	if postID == "" {
		return errors.New("post id is required")
	}

	a.dbMu.Lock()
	defer a.dbMu.Unlock()

	_, err := a.db.Exec(`
		INSERT INTO post_reads (post_id, read_at)
		VALUES (?, ?)
		ON CONFLICT(post_id) DO UPDATE SET
			read_at = excluded.read_at;
	`, postID, time.Now().Unix())
	return err
}

// MarkPostsRead marks multiple posts as read in a single transaction. More
// efficient than calling MarkPostRead in a loop.
func (a *App) MarkPostsRead(postIDs []string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	if len(postIDs) == 0 {
		return nil
	}

	a.dbMu.Lock()
	defer a.dbMu.Unlock()

	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO post_reads (post_id, read_at)
		VALUES (?, ?)
		ON CONFLICT(post_id) DO UPDATE SET
			read_at = excluded.read_at;
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for _, postID := range postIDs {
		postID = strings.TrimSpace(postID)
		if postID == "" {
			continue
		}
		if _, err := stmt.Exec(postID, now); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// IsPostRead returns true if the given post has been marked as read.
func (a *App) IsPostRead(postID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}
	postID = strings.TrimSpace(postID)
	if postID == "" {
		return false, nil
	}

	var count int
	err := a.db.QueryRow(`SELECT COUNT(1) FROM post_reads WHERE post_id = ?;`, postID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetReadPostIDs returns the set of post IDs that have been marked as read.
// Useful for batch filtering when rendering feed UIs.
func (a *App) GetReadPostIDs() (map[string]struct{}, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`SELECT post_id FROM post_reads;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]struct{})
	for rows.Next() {
		var postID string
		if err := rows.Scan(&postID); err != nil {
			return nil, err
		}
		result[postID] = struct{}{}
	}
	return result, rows.Err()
}

// ClearReadHistory removes all post-read records. Useful for users who want to
// reset their reading history.
func (a *App) ClearReadHistory() error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	a.dbMu.Lock()
	defer a.dbMu.Unlock()

	_, err := a.db.Exec(`DELETE FROM post_reads;`)
	return err
}

// GetUnreadPostCount returns the number of unread posts in a sub.
func (a *App) GetUnreadPostCount(subID string) (int, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}
	subID = strings.TrimSpace(subID)
	if subID == "" {
		subID = defaultSubID
	}

	var count int
	err := a.db.QueryRow(`
		SELECT COUNT(1)
		FROM messages m
		LEFT JOIN post_reads r ON r.post_id = m.id
		WHERE m.zone = 'public'
		  AND m.sub_id = ?
		  AND m.visibility = 'normal'
		  AND r.post_id IS NULL;
	`, subID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// SetUserPreference stores a key-value preference. Values are limited to 4 KiB.
func (a *App) SetUserPreference(key string, value string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("preference key is required")
	}
	if len(key) > 64 {
		return errors.New("preference key too long (max 64 chars)")
	}
	if len(value) > 4096 {
		return errors.New("preference value too large (max 4 KiB)")
	}

	a.dbMu.Lock()
	defer a.dbMu.Unlock()

	_, err := a.db.Exec(`
		INSERT INTO user_preferences (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = excluded.updated_at;
	`, key, value, time.Now().Unix())
	return err
}

// GetUserPreference retrieves a stored preference by key. Returns empty string
// if the key does not exist.
func (a *App) GetUserPreference(key string) (string, error) {
	if a.db == nil {
		return "", errors.New("database not initialized")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", nil
	}

	var value string
	err := a.db.QueryRow(`SELECT value FROM user_preferences WHERE key = ?;`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// DeleteUserPreference removes a stored preference. Idempotent.
func (a *App) DeleteUserPreference(key string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}

	a.dbMu.Lock()
	defer a.dbMu.Unlock()

	_, err := a.db.Exec(`DELETE FROM user_preferences WHERE key = ?;`, key)
	return err
}

// GetAllUserPreferences returns all stored preferences as a slice.
func (a *App) GetAllUserPreferences() ([]UserPreference, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT key, value, updated_at
		FROM user_preferences
		ORDER BY key ASC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]UserPreference, 0)
	for rows.Next() {
		var p UserPreference
		if err := rows.Scan(&p.Key, &p.Value, &p.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
