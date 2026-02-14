package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	totalQuotaBytes   int64 = 100 * 1024 * 1024
	privateQuotaBytes int64 = 20 * 1024 * 1024
	publicQuotaBytes  int64 = 80 * 1024 * 1024
)

type ForumMessage struct {
	ID          string `json:"id"`
	Pubkey      string `json:"pubkey"`
	Content     string `json:"content"`
	Timestamp   int64  `json:"timestamp"`
	SizeBytes   int64  `json:"sizeBytes"`
	Zone        string `json:"zone"`
	IsProtected int    `json:"isProtected"`
	Visibility  string `json:"visibility"`
}

type ModerationState struct {
	TargetPubkey string `json:"targetPubkey"`
	Action       string `json:"action"`
	SourceAdmin  string `json:"sourceAdmin"`
	Timestamp    int64  `json:"timestamp"`
	Reason       string `json:"reason"`
}

type IdentityState struct {
	Pubkey             string `json:"pubkey"`
	State              string `json:"state"`
	StorageCommitBytes int64  `json:"storageCommitBytes"`
	PublicQuotaBytes   int64  `json:"publicQuotaBytes"`
	PrivateQuotaBytes  int64  `json:"privateQuotaBytes"`
	UpdatedAt          int64  `json:"updatedAt"`
}

type StorageUsage struct {
	PrivateUsedBytes int64 `json:"privateUsedBytes"`
	PublicUsedBytes  int64 `json:"publicUsedBytes"`
	PrivateQuota     int64 `json:"privateQuota"`
	PublicQuota      int64 `json:"publicQuota"`
	TotalQuota       int64 `json:"totalQuota"`
}

type IncomingMessage struct {
	Type         string `json:"type"`
	ID           string `json:"id"`
	Pubkey       string `json:"pubkey"`
	Content      string `json:"content"`
	Timestamp    int64  `json:"timestamp"`
	Signature    string `json:"signature"`
	TargetPubkey string `json:"target_pubkey"`
	AdminPubkey  string `json:"admin_pubkey"`
	Reason       string `json:"reason"`
}

func (a *App) initDatabase() error {
	a.dbMu.Lock()
	defer a.dbMu.Unlock()

	if a.db != nil {
		return nil
	}

	db, err := sql.Open("sqlite3", "file:aegis_node.db?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return err
	}

	if _, err = db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		_ = db.Close()
		return err
	}

	if err = a.ensureSchema(db); err != nil {
		_ = db.Close()
		return err
	}

	a.db = db
	return nil
}

func (a *App) ensureSchema(db *sql.DB) error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			pubkey TEXT NOT NULL,
			content TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			size_bytes INTEGER NOT NULL,
			zone TEXT NOT NULL CHECK (zone IN ('private', 'public')),
			is_protected INTEGER NOT NULL DEFAULT 0,
			visibility TEXT NOT NULL DEFAULT 'normal'
		);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_zone_timestamp ON messages(zone, timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_pubkey_timestamp ON messages(pubkey, timestamp);`,
		`CREATE TABLE IF NOT EXISTS moderation (
			target_pubkey TEXT PRIMARY KEY,
			action TEXT NOT NULL,
			source_admin TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			reason TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS identity_state (
			pubkey TEXT PRIMARY KEY,
			state TEXT NOT NULL,
			storage_commit_bytes INTEGER NOT NULL,
			public_quota_bytes INTEGER NOT NULL,
			private_quota_bytes INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS governance_admins (
			admin_pubkey TEXT PRIMARY KEY,
			role TEXT NOT NULL,
			active INTEGER NOT NULL DEFAULT 1
		);`,
	}

	for _, statement := range schema {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}

	return nil
}

func (a *App) GetFeed() ([]ForumMessage, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT id, pubkey, content, timestamp, size_bytes, zone, is_protected, visibility
		FROM messages
		WHERE zone = 'public' AND visibility = 'normal'
		ORDER BY timestamp DESC
		LIMIT 200;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]ForumMessage, 0)
	for rows.Next() {
		var message ForumMessage
		if err := rows.Scan(
			&message.ID,
			&message.Pubkey,
			&message.Content,
			&message.Timestamp,
			&message.SizeBytes,
			&message.Zone,
			&message.IsProtected,
			&message.Visibility,
		); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, rows.Err()
}

func (a *App) GetPrivateFeed() ([]ForumMessage, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT id, pubkey, content, timestamp, size_bytes, zone, is_protected, visibility
		FROM messages
		WHERE zone = 'private'
		ORDER BY timestamp DESC
		LIMIT 200;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]ForumMessage, 0)
	for rows.Next() {
		var message ForumMessage
		if err := rows.Scan(
			&message.ID,
			&message.Pubkey,
			&message.Content,
			&message.Timestamp,
			&message.SizeBytes,
			&message.Zone,
			&message.IsProtected,
			&message.Visibility,
		); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, rows.Err()
}

func (a *App) GetModerationState() ([]ModerationState, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT target_pubkey, action, source_admin, timestamp, reason
		FROM moderation
		ORDER BY timestamp DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ModerationState, 0)
	for rows.Next() {
		var state ModerationState
		if err := rows.Scan(&state.TargetPubkey, &state.Action, &state.SourceAdmin, &state.Timestamp, &state.Reason); err != nil {
			return nil, err
		}
		result = append(result, state)
	}

	return result, rows.Err()
}

func (a *App) GetIdentityState() ([]IdentityState, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT pubkey, state, storage_commit_bytes, public_quota_bytes, private_quota_bytes, updated_at
		FROM identity_state
		ORDER BY updated_at DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]IdentityState, 0)
	for rows.Next() {
		var state IdentityState
		if err := rows.Scan(
			&state.Pubkey,
			&state.State,
			&state.StorageCommitBytes,
			&state.PublicQuotaBytes,
			&state.PrivateQuotaBytes,
			&state.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, state)
	}

	return result, rows.Err()
}

func (a *App) GetStorageUsage() (StorageUsage, error) {
	if a.db == nil {
		return StorageUsage{}, errors.New("database not initialized")
	}

	usage := StorageUsage{
		PrivateQuota: privateQuotaBytes,
		PublicQuota:  publicQuotaBytes,
		TotalQuota:   totalQuotaBytes,
	}

	if err := a.db.QueryRow(`SELECT COALESCE(SUM(size_bytes), 0) FROM messages WHERE zone = 'private';`).Scan(&usage.PrivateUsedBytes); err != nil {
		return StorageUsage{}, err
	}

	if err := a.db.QueryRow(`SELECT COALESCE(SUM(size_bytes), 0) FROM messages WHERE zone = 'public';`).Scan(&usage.PublicUsedBytes); err != nil {
		return StorageUsage{}, err
	}

	return usage, nil
}

func (a *App) ProcessIncomingMessage(payload []byte) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	var message IncomingMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return err
	}

	message.Type = strings.ToUpper(strings.TrimSpace(message.Type))

	switch message.Type {
	case "SHADOW_BAN":
		return a.upsertModeration(message.TargetPubkey, "SHADOW_BAN", message.AdminPubkey, message.Timestamp, message.Reason)
	case "UNBAN":
		return a.upsertModeration(message.TargetPubkey, "UNBAN", message.AdminPubkey, message.Timestamp, message.Reason)
	case "POST":
		if strings.TrimSpace(message.Pubkey) == "" || strings.TrimSpace(message.Content) == "" {
			return errors.New("invalid post payload")
		}

		shadowBanned, err := a.isShadowBanned(message.Pubkey)
		if err != nil {
			return err
		}
		if shadowBanned {
			return nil
		}

		if strings.TrimSpace(message.ID) == "" {
			message.ID = buildMessageID(message.Pubkey, message.Content, message.Timestamp)
		}
		if message.Timestamp == 0 {
			message.Timestamp = time.Now().Unix()
		}

		_, err = a.insertMessage(ForumMessage{
			ID:          message.ID,
			Pubkey:      message.Pubkey,
			Content:     message.Content,
			Timestamp:   message.Timestamp,
			SizeBytes:   int64(len([]byte(message.Content))),
			Zone:        "public",
			Visibility:  "normal",
			IsProtected: 0,
		})
		return err
	default:
		return fmt.Errorf("unsupported message type: %s", message.Type)
	}
}

func (a *App) AddLocalPost(pubkey string, content string, zone string) (ForumMessage, error) {
	zone = strings.ToLower(strings.TrimSpace(zone))
	if zone != "private" && zone != "public" {
		return ForumMessage{}, errors.New("zone must be private or public")
	}

	now := time.Now().Unix()
	message := ForumMessage{
		ID:          buildMessageID(pubkey, content, now),
		Pubkey:      pubkey,
		Content:     content,
		Timestamp:   now,
		SizeBytes:   int64(len([]byte(content))),
		Zone:        zone,
		Visibility:  "normal",
		IsProtected: 0,
	}

	return a.insertMessage(message)
}

func (a *App) insertMessage(message ForumMessage) (ForumMessage, error) {
	if a.db == nil {
		return ForumMessage{}, errors.New("database not initialized")
	}

	if message.Zone != "private" && message.Zone != "public" {
		return ForumMessage{}, errors.New("invalid message zone")
	}

	if message.SizeBytes <= 0 {
		message.SizeBytes = int64(len([]byte(message.Content)))
	}

	quota := publicQuotaBytes
	if message.Zone == "private" {
		quota = privateQuotaBytes
	}

	a.dbMu.Lock()
	defer a.dbMu.Unlock()

	tx, err := a.db.Begin()
	if err != nil {
		return ForumMessage{}, err
	}

	if err = ensureZoneQuota(tx, message.Zone, quota, message.SizeBytes); err != nil {
		_ = tx.Rollback()
		return ForumMessage{}, err
	}

	_, err = tx.Exec(
		`INSERT OR REPLACE INTO messages (id, pubkey, content, timestamp, size_bytes, zone, is_protected, visibility)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?);`,
		message.ID,
		message.Pubkey,
		message.Content,
		message.Timestamp,
		message.SizeBytes,
		message.Zone,
		message.IsProtected,
		message.Visibility,
	)
	if err != nil {
		_ = tx.Rollback()
		return ForumMessage{}, err
	}

	if err = tx.Commit(); err != nil {
		return ForumMessage{}, err
	}

	return message, nil
}

func ensureZoneQuota(tx *sql.Tx, zone string, quota int64, incomingBytes int64) error {
	if incomingBytes > quota {
		return errors.New("message exceeds zone quota")
	}

	for {
		var used int64
		if err := tx.QueryRow(`SELECT COALESCE(SUM(size_bytes), 0) FROM messages WHERE zone = ?;`, zone).Scan(&used); err != nil {
			return err
		}

		if used+incomingBytes <= quota {
			return nil
		}

		result, err := tx.Exec(`
			DELETE FROM messages
			WHERE id IN (
				SELECT id
				FROM messages
				WHERE zone = ? AND is_protected = 0
				ORDER BY timestamp ASC
				LIMIT 1
			);
		`, zone)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return errors.New("quota exceeded and no evictable records")
		}
	}
}

func (a *App) upsertModeration(targetPubkey string, action string, sourceAdmin string, timestamp int64, reason string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	action = strings.ToUpper(strings.TrimSpace(action))
	if action != "SHADOW_BAN" && action != "UNBAN" {
		return errors.New("invalid moderation action")
	}

	if strings.TrimSpace(targetPubkey) == "" || strings.TrimSpace(sourceAdmin) == "" {
		return errors.New("invalid moderation payload")
	}

	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}

	_, err := a.db.Exec(`
		INSERT INTO moderation (target_pubkey, action, source_admin, timestamp, reason)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(target_pubkey) DO UPDATE SET
			action = excluded.action,
			source_admin = excluded.source_admin,
			timestamp = excluded.timestamp,
			reason = excluded.reason;
	`, targetPubkey, action, sourceAdmin, timestamp, reason)
	if err != nil {
		return err
	}

	state := "normal"
	if action == "SHADOW_BAN" {
		state = "shadow_banned"
	}

	_, err = a.db.Exec(`
		INSERT INTO identity_state (pubkey, state, storage_commit_bytes, public_quota_bytes, private_quota_bytes, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(pubkey) DO UPDATE SET
			state = excluded.state,
			storage_commit_bytes = excluded.storage_commit_bytes,
			public_quota_bytes = excluded.public_quota_bytes,
			private_quota_bytes = excluded.private_quota_bytes,
			updated_at = excluded.updated_at;
	`, targetPubkey, state, totalQuotaBytes, publicQuotaBytes, privateQuotaBytes, time.Now().Unix())

	return err
}

func (a *App) isShadowBanned(pubkey string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	var action string
	err := a.db.QueryRow(`SELECT action FROM moderation WHERE target_pubkey = ?;`, pubkey).Scan(&action)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return strings.ToUpper(action) == "SHADOW_BAN", nil
}

func buildMessageID(pubkey string, content string, timestamp int64) string {
	raw := fmt.Sprintf("%s|%d|%s", pubkey, timestamp, content)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}
