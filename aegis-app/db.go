package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
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
	Title       string `json:"title"`
	Body        string `json:"body"`
	ContentCID  string `json:"contentCid"`
	Content     string `json:"content"`
	Score       int64  `json:"score"`
	Timestamp   int64  `json:"timestamp"`
	SizeBytes   int64  `json:"sizeBytes"`
	Zone        string `json:"zone"`
	SubID       string `json:"subId"`
	IsProtected int    `json:"isProtected"`
	Visibility  string `json:"visibility"`
}

type PostIndex struct {
	ID          string `json:"id"`
	Pubkey      string `json:"pubkey"`
	Title       string `json:"title"`
	BodyPreview string `json:"bodyPreview"`
	ContentCID  string `json:"contentCid"`
	Score       int64  `json:"score"`
	Timestamp   int64  `json:"timestamp"`
	Zone        string `json:"zone"`
	SubID       string `json:"subId"`
	Visibility  string `json:"visibility"`
}

type PostBodyBlob struct {
	ContentCID string `json:"contentCid"`
	Body       string `json:"body"`
	SizeBytes  int64  `json:"sizeBytes"`
}

type Sub struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"createdAt"`
}

type Profile struct {
	Pubkey      string `json:"pubkey"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarURL"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type Comment struct {
	ID        string `json:"id"`
	PostID    string `json:"postId"`
	ParentID  string `json:"parentId"`
	Pubkey    string `json:"pubkey"`
	Body      string `json:"body"`
	Score     int64  `json:"score"`
	Timestamp int64  `json:"timestamp"`
}

type ModerationState struct {
	TargetPubkey string `json:"targetPubkey"`
	Action       string `json:"action"`
	SourceAdmin  string `json:"sourceAdmin"`
	Timestamp    int64  `json:"timestamp"`
	Reason       string `json:"reason"`
}

type ModerationLog struct {
	ID           int64  `json:"id"`
	TargetPubkey string `json:"targetPubkey"`
	Action       string `json:"action"`
	SourceAdmin  string `json:"sourceAdmin"`
	Timestamp    int64  `json:"timestamp"`
	Reason       string `json:"reason"`
	Result       string `json:"result"`
}

type GovernancePolicy struct {
	HideHistoryOnShadowBan bool `json:"hideHistoryOnShadowBan"`
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

type GovernanceAdmin struct {
	AdminPubkey string `json:"adminPubkey"`
	Role        string `json:"role"`
	Active      bool   `json:"active"`
}

type IncomingMessage struct {
	Type                   string `json:"type"`
	ID                     string `json:"id"`
	Pubkey                 string `json:"pubkey"`
	VoterPubkey            string `json:"voter_pubkey"`
	PostID                 string `json:"post_id"`
	CommentID              string `json:"comment_id"`
	ParentID               string `json:"parent_id"`
	DisplayName            string `json:"display_name"`
	AvatarURL              string `json:"avatar_url"`
	Title                  string `json:"title"`
	Body                   string `json:"body"`
	ContentCID             string `json:"content_cid"`
	Content                string `json:"content"`
	SubID                  string `json:"sub_id"`
	SubTitle               string `json:"sub_title"`
	SubDesc                string `json:"sub_desc"`
	Timestamp              int64  `json:"timestamp"`
	Signature              string `json:"signature"`
	TargetPubkey           string `json:"target_pubkey"`
	AdminPubkey            string `json:"admin_pubkey"`
	Reason                 string `json:"reason"`
	HideHistoryOnShadowBan bool   `json:"hide_history_on_shadowban"`
}

const defaultSubID = "general"

func (a *App) initDatabase() error {
	a.dbMu.Lock()
	defer a.dbMu.Unlock()

	if a.db != nil {
		return nil
	}

	databasePath := strings.TrimSpace(a.dbPath)
	if databasePath == "" {
		databasePath = "aegis_node.db"
	}

	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL", databasePath)
	db, err := sql.Open("sqlite3", dsn)
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
			title TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL DEFAULT '',
			content_cid TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			score INTEGER NOT NULL DEFAULT 0,
			timestamp INTEGER NOT NULL,
			size_bytes INTEGER NOT NULL,
			zone TEXT NOT NULL CHECK (zone IN ('private', 'public')),
			sub_id TEXT NOT NULL DEFAULT 'general',
			is_protected INTEGER NOT NULL DEFAULT 0,
			visibility TEXT NOT NULL DEFAULT 'normal'
		);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_zone_timestamp ON messages(zone, timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_pubkey_timestamp ON messages(pubkey, timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_content_cid ON messages(content_cid);`,
		`CREATE TABLE IF NOT EXISTS content_blobs (
			content_cid TEXT PRIMARY KEY,
			body TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			last_accessed_at INTEGER NOT NULL,
			pinned INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS subs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS profiles (
			pubkey TEXT PRIMARY KEY,
			display_name TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			updated_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS comments (
			id TEXT PRIMARY KEY,
			post_id TEXT NOT NULL,
			parent_id TEXT NOT NULL DEFAULT '',
			pubkey TEXT NOT NULL,
			body TEXT NOT NULL,
			score INTEGER NOT NULL DEFAULT 0,
			timestamp INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_post_timestamp ON comments(post_id, timestamp);`,
		`CREATE TABLE IF NOT EXISTS post_votes (
			post_id TEXT NOT NULL,
			voter_pubkey TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			PRIMARY KEY (post_id, voter_pubkey)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_post_votes_post ON post_votes(post_id);`,
		`CREATE TABLE IF NOT EXISTS comment_votes (
			comment_id TEXT NOT NULL,
			voter_pubkey TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			PRIMARY KEY (comment_id, voter_pubkey)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_comment_votes_comment ON comment_votes(comment_id);`,
		`CREATE TABLE IF NOT EXISTS moderation (
			target_pubkey TEXT PRIMARY KEY,
			action TEXT NOT NULL,
			source_admin TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			reason TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS moderation_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target_pubkey TEXT NOT NULL,
			action TEXT NOT NULL,
			source_admin TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			result TEXT NOT NULL DEFAULT 'applied'
		);`,
		`CREATE INDEX IF NOT EXISTS idx_moderation_logs_timestamp ON moderation_logs(timestamp DESC);`,
		`CREATE TABLE IF NOT EXISTS governance_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at INTEGER NOT NULL
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
		`CREATE TABLE IF NOT EXISTS local_identity (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			mnemonic TEXT NOT NULL,
			pubkey TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
	}

	for _, statement := range schema {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN sub_id TEXT NOT NULL DEFAULT 'general';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN title TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN body TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN content_cid TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE content_blobs ADD COLUMN last_accessed_at INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE content_blobs ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN score INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE comments ADD COLUMN score INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`UPDATE messages SET sub_id = ? WHERE COALESCE(TRIM(sub_id), '') = '';`, defaultSubID); err != nil {
		return err
	}

	if _, err := db.Exec(`UPDATE messages SET body = content WHERE COALESCE(TRIM(body), '') = '';`); err != nil {
		return err
	}

	if _, err := db.Exec(`UPDATE messages SET title = SUBSTR(body, 1, 20) WHERE COALESCE(TRIM(title), '') = '';`); err != nil {
		return err
	}

	rows, err := db.Query(`SELECT id, body, content, content_cid, size_bytes, timestamp FROM messages;`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type cidBackfill struct {
		id        string
		body      string
		content   string
		contentID string
		sizeBytes int64
		timestamp int64
	}
	updates := make([]cidBackfill, 0)
	for rows.Next() {
		var item cidBackfill
		if err = rows.Scan(&item.id, &item.body, &item.content, &item.contentID, &item.sizeBytes, &item.timestamp); err != nil {
			return err
		}
		updates = append(updates, item)
	}
	if err = rows.Err(); err != nil {
		return err
	}

	for _, row := range updates {
		payload := strings.TrimSpace(row.body)
		if payload == "" {
			payload = strings.TrimSpace(row.content)
		}
		if payload == "" {
			continue
		}

		cid := strings.TrimSpace(row.contentID)
		if cid == "" {
			cid = buildContentCID(payload)
			if _, err = db.Exec(`UPDATE messages SET content_cid = ? WHERE id = ?;`, cid, row.id); err != nil {
				return err
			}
		}

		sizeBytes := row.sizeBytes
		if sizeBytes <= 0 {
			sizeBytes = int64(len([]byte(payload)))
		}
		createdAt := row.timestamp
		if createdAt <= 0 {
			createdAt = time.Now().Unix()
		}

		if _, err = db.Exec(`
			INSERT INTO content_blobs (content_cid, body, size_bytes, created_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(content_cid) DO UPDATE SET
				body = excluded.body,
				size_bytes = excluded.size_bytes;
		`, cid, payload, sizeBytes, createdAt); err != nil {
			return err
		}

		if _, err = db.Exec(`
			UPDATE content_blobs
			SET last_accessed_at = CASE
				WHEN COALESCE(last_accessed_at, 0) <= 0 THEN ?
				ELSE last_accessed_at
			END
			WHERE content_cid = ?;
		`, createdAt, cid); err != nil {
			return err
		}
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_zone_sub_timestamp ON messages(zone, sub_id, timestamp);`); err != nil {
		return err
	}

	now := time.Now().Unix()
	if _, err := db.Exec(`
		INSERT INTO subs (id, title, description, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING;
	`, defaultSubID, "General", "Default public space", now); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO governance_config (key, value, updated_at)
		VALUES ('hide_history_on_shadowban', '1', ?)
		ON CONFLICT(key) DO NOTHING;
	`, now); err != nil {
		return err
	}

	return nil
}

func (a *App) GetFeed() ([]ForumMessage, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	rows, err := a.db.Query(`
		SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
		FROM messages
		WHERE zone = 'public' AND (visibility = 'normal' OR pubkey = ?)
		ORDER BY timestamp DESC
		LIMIT 200;
	`, viewerPubkey)
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
			&message.Title,
			&message.Body,
			&message.ContentCID,
			&message.Content,
			&message.Score,
			&message.Timestamp,
			&message.SizeBytes,
			&message.Zone,
			&message.SubID,
			&message.IsProtected,
			&message.Visibility,
		); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, rows.Err()
}

func (a *App) GetFeedBySub(subID string) ([]ForumMessage, error) {
	return a.GetFeedBySubSorted(subID, "hot")
}

func (a *App) GetFeedBySubSorted(subID string, sortMode string) ([]ForumMessage, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	subID = normalizeSubID(subID)
	sortMode = strings.ToLower(strings.TrimSpace(sortMode))
	if sortMode == "" {
		sortMode = "hot"
	}

	orderBy := "score DESC, timestamp DESC"
	if sortMode == "new" {
		orderBy = "timestamp DESC"
	}

	if sortMode != "hot" {
		query := fmt.Sprintf(`
			SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
			FROM messages
			WHERE zone = 'public' AND (visibility = 'normal' OR pubkey = ?) AND sub_id = ?
			ORDER BY %s
			LIMIT 200;
		`, orderBy)

		rows, err := a.db.Query(query, viewerPubkey, subID)
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
				&message.Title,
				&message.Body,
				&message.ContentCID,
				&message.Content,
				&message.Score,
				&message.Timestamp,
				&message.SizeBytes,
				&message.Zone,
				&message.SubID,
				&message.IsProtected,
				&message.Visibility,
			); err != nil {
				return nil, err
			}
			messages = append(messages, message)
		}

		return messages, rows.Err()
	}

	query := `
		SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
		FROM messages
		WHERE zone = 'public' AND (visibility = 'normal' OR pubkey = ?) AND sub_id = ?
		ORDER BY timestamp DESC
		LIMIT 500;
	`

	rows, err := a.db.Query(query, viewerPubkey, subID)
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
			&message.Title,
			&message.Body,
			&message.ContentCID,
			&message.Content,
			&message.Score,
			&message.Timestamp,
			&message.SizeBytes,
			&message.Zone,
			&message.SubID,
			&message.IsProtected,
			&message.Visibility,
		); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	sort.SliceStable(messages, func(i int, j int) bool {
		left := computeHotScore(messages[i].Score, messages[i].Timestamp, now)
		right := computeHotScore(messages[j].Score, messages[j].Timestamp, now)
		if left == right {
			return messages[i].Timestamp > messages[j].Timestamp
		}
		return left > right
	})

	if len(messages) > 200 {
		messages = messages[:200]
	}

	return messages, nil
}

func (a *App) GetFeedIndexBySubSorted(subID string, sortMode string) ([]PostIndex, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	subID = normalizeSubID(subID)
	sortMode = strings.ToLower(strings.TrimSpace(sortMode))
	if sortMode == "" {
		sortMode = "hot"
	}

	query := `
		SELECT id, pubkey, title, SUBSTR(body, 1, 140) AS body_preview, content_cid, score, timestamp, zone, sub_id, visibility
		FROM messages
		WHERE zone = 'public' AND (visibility = 'normal' OR pubkey = ?) AND sub_id = ?
		ORDER BY timestamp DESC
		LIMIT 500;
	`

	rows, err := a.db.Query(query, viewerPubkey, subID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PostIndex, 0)
	for rows.Next() {
		var item PostIndex
		if err = rows.Scan(
			&item.ID,
			&item.Pubkey,
			&item.Title,
			&item.BodyPreview,
			&item.ContentCID,
			&item.Score,
			&item.Timestamp,
			&item.Zone,
			&item.SubID,
			&item.Visibility,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if sortMode == "hot" {
		now := time.Now().Unix()
		sort.SliceStable(items, func(i int, j int) bool {
			left := computeHotScore(items[i].Score, items[i].Timestamp, now)
			right := computeHotScore(items[j].Score, items[j].Timestamp, now)
			if left == right {
				return items[i].Timestamp > items[j].Timestamp
			}
			return left > right
		})
	}

	if len(items) > 200 {
		items = items[:200]
	}

	return items, nil
}

func (a *App) GetPostBodyByCID(contentCID string) (PostBodyBlob, error) {
	if a.db == nil {
		return PostBodyBlob{}, errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return PostBodyBlob{}, errors.New("content cid is required")
	}

	var body PostBodyBlob
	err := a.db.QueryRow(`
		SELECT content_cid, body, size_bytes
		FROM content_blobs
		WHERE content_cid = ?;
	`, contentCID).Scan(&body.ContentCID, &body.Body, &body.SizeBytes)
	if errors.Is(err, sql.ErrNoRows) {
		return PostBodyBlob{}, errors.New("content not found")
	}
	if err != nil {
		return PostBodyBlob{}, err
	}

	if _, err = a.db.Exec(`UPDATE content_blobs SET last_accessed_at = ? WHERE content_cid = ?;`, time.Now().Unix(), contentCID); err != nil {
		return PostBodyBlob{}, err
	}

	return body, nil
}

func (a *App) GetPostBodyByID(postID string) (PostBodyBlob, error) {
	if a.db == nil {
		return PostBodyBlob{}, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return PostBodyBlob{}, errors.New("post id is required")
	}

	var contentCID string
	err := a.db.QueryRow(`SELECT content_cid FROM messages WHERE id = ?;`, postID).Scan(&contentCID)
	if errors.Is(err, sql.ErrNoRows) {
		return PostBodyBlob{}, errors.New("post not found")
	}
	if err != nil {
		return PostBodyBlob{}, err
	}

	return a.GetPostBodyByCID(contentCID)
}

func computeHotScore(score int64, createdAt int64, now int64) float64 {
	ageHours := float64(now-createdAt) / 3600.0
	if ageHours < 0 {
		ageHours = 0
	}

	return float64(score) / math.Pow(ageHours+2, 1.2)
}

func (a *App) GetPrivateFeed() ([]ForumMessage, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
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
			&message.Title,
			&message.Body,
			&message.ContentCID,
			&message.Content,
			&message.Score,
			&message.Timestamp,
			&message.SizeBytes,
			&message.Zone,
			&message.SubID,
			&message.IsProtected,
			&message.Visibility,
		); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, rows.Err()
}

func (a *App) GetCommentsByPost(postID string) ([]Comment, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return nil, errors.New("post id is required")
	}

	rows, err := a.db.Query(`
		SELECT id, post_id, parent_id, pubkey, body, score, timestamp
		FROM comments
		WHERE post_id = ?
		ORDER BY timestamp ASC;
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Comment, 0)
	for rows.Next() {
		var comment Comment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.ParentID, &comment.Pubkey, &comment.Body, &comment.Score, &comment.Timestamp); err != nil {
			return nil, err
		}
		result = append(result, comment)
	}

	return result, rows.Err()
}

func (a *App) CreateSub(id string, title string, description string) (Sub, error) {
	if a.db == nil {
		return Sub{}, errors.New("database not initialized")
	}

	return a.upsertSub(id, title, description, time.Now().Unix())
}

func (a *App) GetSubs() ([]Sub, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT id, title, description, created_at
		FROM subs
		ORDER BY id ASC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Sub, 0)
	for rows.Next() {
		var sub Sub
		if err := rows.Scan(&sub.ID, &sub.Title, &sub.Description, &sub.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, sub)
	}

	return result, rows.Err()
}

func (a *App) UpdateProfile(displayName string, avatarURL string) (Profile, error) {
	if a.db == nil {
		return Profile{}, errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return Profile{}, err
	}

	return a.upsertProfile(identity.PublicKey, displayName, avatarURL, time.Now().Unix())
}

func (a *App) GetProfile(pubkey string) (Profile, error) {
	if a.db == nil {
		return Profile{}, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return Profile{}, errors.New("pubkey is required")
	}

	var profile Profile
	err := a.db.QueryRow(`
		SELECT pubkey, display_name, avatar_url, updated_at
		FROM profiles
		WHERE pubkey = ?;
	`, pubkey).Scan(&profile.Pubkey, &profile.DisplayName, &profile.AvatarURL, &profile.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{Pubkey: pubkey}, nil
	}
	if err != nil {
		return Profile{}, err
	}

	return profile, nil
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

func (a *App) GetModerationLogs(limit int) ([]ModerationLog, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 500 {
		limit = 100
	}

	rows, err := a.db.Query(`
		SELECT id, target_pubkey, action, source_admin, timestamp, reason, result
		FROM moderation_logs
		ORDER BY timestamp DESC, id DESC
		LIMIT ?;
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ModerationLog, 0)
	for rows.Next() {
		var row ModerationLog
		if err = rows.Scan(&row.ID, &row.TargetPubkey, &row.Action, &row.SourceAdmin, &row.Timestamp, &row.Reason, &row.Result); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

func (a *App) GetGovernancePolicy() (GovernancePolicy, error) {
	if a.db == nil {
		return GovernancePolicy{}, errors.New("database not initialized")
	}

	var value string
	err := a.db.QueryRow(`SELECT value FROM governance_config WHERE key = 'hide_history_on_shadowban';`).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return GovernancePolicy{HideHistoryOnShadowBan: true}, nil
	}
	if err != nil {
		return GovernancePolicy{}, err
	}

	value = strings.TrimSpace(strings.ToLower(value))
	hide := value == "1" || value == "true" || value == "yes"

	return GovernancePolicy{HideHistoryOnShadowBan: hide}, nil
}

func (a *App) SetGovernancePolicy(hideHistoryOnShadowBan bool) (GovernancePolicy, error) {
	if a.db == nil {
		return GovernancePolicy{}, errors.New("database not initialized")
	}

	value := "0"
	if hideHistoryOnShadowBan {
		value = "1"
	}

	_, err := a.db.Exec(`
		INSERT INTO governance_config (key, value, updated_at)
		VALUES ('hide_history_on_shadowban', ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = excluded.updated_at;
	`, value, time.Now().Unix())
	if err != nil {
		return GovernancePolicy{}, err
	}

	return GovernancePolicy{HideHistoryOnShadowBan: hideHistoryOnShadowBan}, nil
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

	if err := a.db.QueryRow(`
		SELECT COALESCE(SUM(cb.size_bytes), 0)
		FROM content_blobs cb
		WHERE EXISTS (
			SELECT 1 FROM messages m
			WHERE m.zone = 'private' AND m.content_cid = cb.content_cid
		);
	`).Scan(&usage.PrivateUsedBytes); err != nil {
		return StorageUsage{}, err
	}

	if err := a.db.QueryRow(`
		SELECT COALESCE(SUM(cb.size_bytes), 0)
		FROM content_blobs cb
		WHERE EXISTS (
			SELECT 1 FROM messages m
			WHERE m.zone = 'public' AND m.content_cid = cb.content_cid
		);
	`).Scan(&usage.PublicUsedBytes); err != nil {
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
	case "SUB_CREATE":
		_, err := a.upsertSub(message.SubID, message.SubTitle, message.SubDesc, message.Timestamp)
		return err
	case "GOVERNANCE_POLICY_UPDATE":
		_, err := a.SetGovernancePolicy(message.HideHistoryOnShadowBan)
		return err
	case "PROFILE_UPDATE":
		_, err := a.upsertProfile(message.Pubkey, message.DisplayName, message.AvatarURL, message.Timestamp)
		return err
	case "POST_UPVOTE":
		voterPubkey := strings.TrimSpace(message.VoterPubkey)
		if voterPubkey == "" {
			voterPubkey = strings.TrimSpace(message.Pubkey)
		}
		if voterPubkey == "" || strings.TrimSpace(message.PostID) == "" {
			return errors.New("invalid post upvote payload")
		}

		return a.applyPostUpvote(voterPubkey, message.PostID)
	case "COMMENT_UPVOTE":
		voterPubkey := strings.TrimSpace(message.VoterPubkey)
		if voterPubkey == "" {
			voterPubkey = strings.TrimSpace(message.Pubkey)
		}
		if voterPubkey == "" || strings.TrimSpace(message.CommentID) == "" || strings.TrimSpace(message.PostID) == "" {
			return errors.New("invalid comment upvote payload")
		}

		return a.applyCommentUpvote(voterPubkey, message.CommentID, message.PostID)
	case "COMMENT":
		if strings.TrimSpace(message.Pubkey) == "" || strings.TrimSpace(message.PostID) == "" {
			return errors.New("invalid comment payload")
		}

		shadowBanned, err := a.isShadowBanned(message.Pubkey)
		if err != nil {
			return err
		}
		if shadowBanned {
			return nil
		}

		commentBody := strings.TrimSpace(message.Body)
		if commentBody == "" {
			return errors.New("invalid comment payload")
		}

		if strings.TrimSpace(message.ID) == "" {
			raw := fmt.Sprintf("%s|%s|%s", message.PostID, strings.TrimSpace(message.ParentID), commentBody)
			message.ID = buildMessageID(message.Pubkey, raw, message.Timestamp)
		}
		if message.Timestamp == 0 {
			message.Timestamp = time.Now().Unix()
		}

		_, err = a.insertComment(Comment{
			ID:        message.ID,
			PostID:    strings.TrimSpace(message.PostID),
			ParentID:  strings.TrimSpace(message.ParentID),
			Pubkey:    message.Pubkey,
			Body:      commentBody,
			Timestamp: message.Timestamp,
		})
		return err
	case "SHADOW_BAN":
		trusted, err := a.isTrustedAdmin(message.AdminPubkey)
		if err != nil {
			return err
		}
		if !trusted {
			return errors.New("admin pubkey is not trusted")
		}
		return a.upsertModeration(message.TargetPubkey, "SHADOW_BAN", message.AdminPubkey, message.Timestamp, message.Reason)
	case "UNBAN":
		trusted, err := a.isTrustedAdmin(message.AdminPubkey)
		if err != nil {
			return err
		}
		if !trusted {
			return errors.New("admin pubkey is not trusted")
		}
		return a.upsertModeration(message.TargetPubkey, "UNBAN", message.AdminPubkey, message.Timestamp, message.Reason)
	case "POST":
		if strings.TrimSpace(message.Pubkey) == "" {
			return errors.New("invalid post payload")
		}

		if strings.TrimSpace(message.DisplayName) != "" || strings.TrimSpace(message.AvatarURL) != "" {
			if _, err := a.upsertProfile(message.Pubkey, message.DisplayName, message.AvatarURL, message.Timestamp); err != nil {
				return err
			}
		}

		body := strings.TrimSpace(message.Body)
		if body == "" {
			body = strings.TrimSpace(message.Content)
		}

		title := strings.TrimSpace(message.Title)

		if title == "" || body == "" {
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
			message.ID = buildMessageID(message.Pubkey, body, message.Timestamp)
		}
		if message.Timestamp == 0 {
			message.Timestamp = time.Now().Unix()
		}

		_, err = a.insertMessage(ForumMessage{
			ID:          message.ID,
			Pubkey:      message.Pubkey,
			Title:       title,
			Body:        body,
			ContentCID:  strings.TrimSpace(message.ContentCID),
			Content:     "",
			Score:       0,
			Timestamp:   message.Timestamp,
			SizeBytes:   int64(len([]byte(body))),
			Zone:        "public",
			SubID:       normalizeSubID(message.SubID),
			Visibility:  "normal",
			IsProtected: 0,
		})
		return err
	default:
		return fmt.Errorf("unsupported message type: %s", message.Type)
	}
}

func (a *App) AddLocalPostStructured(pubkey string, title string, body string, zone string) (ForumMessage, error) {
	return a.AddLocalPostStructuredToSub(pubkey, title, body, zone, defaultSubID)
}

func (a *App) AddLocalComment(pubkey string, postID string, parentID string, body string) (Comment, error) {
	if a.db == nil {
		return Comment{}, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	parentID = strings.TrimSpace(parentID)
	body = strings.TrimSpace(body)

	if pubkey == "" || postID == "" || body == "" {
		return Comment{}, errors.New("pubkey, post id and body are required")
	}

	now := time.Now().Unix()
	raw := fmt.Sprintf("%s|%s|%s", postID, parentID, body)
	comment := Comment{
		ID:        buildMessageID(pubkey, raw, now),
		PostID:    postID,
		ParentID:  parentID,
		Pubkey:    pubkey,
		Body:      body,
		Score:     0,
		Timestamp: now,
	}

	return a.insertComment(comment)
}

func (a *App) UpvotePost(postID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return err
	}

	return a.applyPostUpvote(identity.PublicKey, postID)
}

func (a *App) UpvoteComment(commentID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return err
	}

	return a.applyCommentUpvote(identity.PublicKey, commentID, "")
}

func (a *App) upsertSub(id string, title string, description string, createdAt int64) (Sub, error) {
	id = normalizeSubID(id)
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if title == "" {
		title = id
	}
	if createdAt <= 0 {
		createdAt = time.Now().Unix()
	}

	_, err := a.db.Exec(`
		INSERT INTO subs (id, title, description, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			description = excluded.description;
	`, id, title, description, createdAt)
	if err != nil {
		return Sub{}, err
	}

	return Sub{ID: id, Title: title, Description: description, CreatedAt: createdAt}, nil
}

func (a *App) upsertProfile(pubkey string, displayName string, avatarURL string, updatedAt int64) (Profile, error) {
	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return Profile{}, errors.New("pubkey is required")
	}

	displayName = strings.TrimSpace(displayName)
	avatarURL = strings.TrimSpace(avatarURL)
	if len([]rune(displayName)) > 64 {
		displayName = string([]rune(displayName)[:64])
	}

	if updatedAt <= 0 {
		updatedAt = time.Now().Unix()
	}

	_, err := a.db.Exec(`
		INSERT INTO profiles (pubkey, display_name, avatar_url, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(pubkey) DO UPDATE SET
			display_name = excluded.display_name,
			avatar_url = excluded.avatar_url,
			updated_at = excluded.updated_at;
	`, pubkey, displayName, avatarURL, updatedAt)
	if err != nil {
		return Profile{}, err
	}

	return Profile{Pubkey: pubkey, DisplayName: displayName, AvatarURL: avatarURL, UpdatedAt: updatedAt}, nil
}

func (a *App) AddLocalPostStructuredToSub(pubkey string, title string, body string, zone string, subID string) (ForumMessage, error) {
	zone = strings.ToLower(strings.TrimSpace(zone))
	if zone != "private" && zone != "public" {
		return ForumMessage{}, errors.New("zone must be private or public")
	}

	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		title = deriveTitle(body)
	}
	if title == "" || body == "" {
		return ForumMessage{}, errors.New("title and body are required")
	}

	now := time.Now().Unix()
	message := ForumMessage{
		ID:          buildMessageID(pubkey, body, now),
		Pubkey:      pubkey,
		Title:       title,
		Body:        body,
		ContentCID:  buildContentCID(body),
		Content:     "",
		Score:       0,
		Timestamp:   now,
		SizeBytes:   int64(len([]byte(body))),
		Zone:        zone,
		SubID:       normalizeSubID(subID),
		Visibility:  "normal",
		IsProtected: 0,
	}

	return a.insertMessage(message)
}

func (a *App) ApplyShadowBan(targetPubkey string, adminPubkey string, reason string) error {
	trusted, err := a.isTrustedAdmin(adminPubkey)
	if err != nil {
		return err
	}
	if !trusted {
		return errors.New("admin pubkey is not trusted")
	}

	return a.upsertModeration(targetPubkey, "SHADOW_BAN", adminPubkey, time.Now().Unix(), reason)
}

func (a *App) ApplyUnban(targetPubkey string, adminPubkey string, reason string) error {
	trusted, err := a.isTrustedAdmin(adminPubkey)
	if err != nil {
		return err
	}
	if !trusted {
		return errors.New("admin pubkey is not trusted")
	}

	return a.upsertModeration(targetPubkey, "UNBAN", adminPubkey, time.Now().Unix(), reason)
}

func (a *App) AddTrustedAdmin(pubkey string, role string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return errors.New("admin pubkey is required")
	}

	role = strings.TrimSpace(strings.ToLower(role))
	if role == "" {
		role = "appointed"
	}

	_, err := a.db.Exec(`
		INSERT INTO governance_admins (admin_pubkey, role, active)
		VALUES (?, ?, 1)
		ON CONFLICT(admin_pubkey) DO UPDATE SET
			role = excluded.role,
			active = 1;
	`, pubkey, role)
	return err
}

func (a *App) GetTrustedAdmins() ([]GovernanceAdmin, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT admin_pubkey, role, active
		FROM governance_admins
		WHERE active = 1
		ORDER BY role, admin_pubkey;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]GovernanceAdmin, 0)
	for rows.Next() {
		var admin GovernanceAdmin
		var active int
		if err = rows.Scan(&admin.AdminPubkey, &admin.Role, &active); err != nil {
			return nil, err
		}
		admin.Active = active == 1
		result = append(result, admin)
	}

	return result, rows.Err()
}

func (a *App) isTrustedAdmin(pubkey string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return false, nil
	}

	var count int
	err := a.db.QueryRow(`
		SELECT COUNT(1)
		FROM governance_admins
		WHERE admin_pubkey = ? AND active = 1;
	`, pubkey).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (a *App) insertMessage(message ForumMessage) (ForumMessage, error) {
	if a.db == nil {
		return ForumMessage{}, errors.New("database not initialized")
	}

	if message.Zone != "private" && message.Zone != "public" {
		return ForumMessage{}, errors.New("invalid message zone")
	}

	if message.SizeBytes <= 0 {
		message.SizeBytes = int64(len([]byte(message.Body)))
	}

	message.Title = strings.TrimSpace(message.Title)
	message.Body = strings.TrimSpace(message.Body)
	if message.Title == "" || message.Body == "" {
		return ForumMessage{}, errors.New("message title and body are required")
	}
	fullBody := message.Body
	message.ContentCID = strings.TrimSpace(message.ContentCID)
	if message.ContentCID == "" {
		message.ContentCID = buildContentCID(fullBody)
	}
	blobSizeBytes := int64(len([]byte(fullBody)))
	message.Body = deriveBodyPreview(fullBody, 180)
	message.SizeBytes = int64(len([]byte(message.Body)))

	message.SubID = normalizeSubID(message.SubID)

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

	if err = ensureBlobQuotaWithLRU(tx, message.Zone, quota, blobSizeBytes, message.ContentCID); err != nil {
		_ = tx.Rollback()
		return ForumMessage{}, err
	}

	if _, err = tx.Exec(`
		INSERT INTO content_blobs (content_cid, body, size_bytes, created_at, last_accessed_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(content_cid) DO UPDATE SET
			body = excluded.body,
			size_bytes = excluded.size_bytes,
			last_accessed_at = excluded.last_accessed_at;
	`, message.ContentCID, fullBody, blobSizeBytes, message.Timestamp, message.Timestamp); err != nil {
		_ = tx.Rollback()
		return ForumMessage{}, err
	}

	_, err = tx.Exec(
		`INSERT OR REPLACE INTO messages (id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		message.ID,
		message.Pubkey,
		message.Title,
		message.Body,
		message.ContentCID,
		message.Content,
		message.Score,
		message.Timestamp,
		message.SizeBytes,
		message.Zone,
		message.SubID,
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

func (a *App) insertComment(comment Comment) (Comment, error) {
	if a.db == nil {
		return Comment{}, errors.New("database not initialized")
	}

	comment.ID = strings.TrimSpace(comment.ID)
	comment.PostID = strings.TrimSpace(comment.PostID)
	comment.ParentID = strings.TrimSpace(comment.ParentID)
	comment.Pubkey = strings.TrimSpace(comment.Pubkey)
	comment.Body = strings.TrimSpace(comment.Body)

	if comment.ID == "" || comment.PostID == "" || comment.Pubkey == "" || comment.Body == "" {
		return Comment{}, errors.New("invalid comment")
	}
	if comment.Timestamp == 0 {
		comment.Timestamp = time.Now().Unix()
	}

	_, err := a.db.Exec(`
		INSERT OR REPLACE INTO comments (id, post_id, parent_id, pubkey, body, score, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`, comment.ID, comment.PostID, comment.ParentID, comment.Pubkey, comment.Body, comment.Score, comment.Timestamp)
	if err != nil {
		return Comment{}, err
	}

	return comment, nil
}

func (a *App) applyPostUpvote(voterPubkey string, postID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	voterPubkey = strings.TrimSpace(voterPubkey)
	postID = strings.TrimSpace(postID)
	if voterPubkey == "" || postID == "" {
		return errors.New("voter pubkey and post id are required")
	}

	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	result, err := tx.Exec(`
		INSERT INTO post_votes (post_id, voter_pubkey, timestamp)
		VALUES (?, ?, ?)
		ON CONFLICT(post_id, voter_pubkey) DO NOTHING;
	`, postID, voterPubkey, time.Now().Unix())
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	inserted, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if inserted == 0 {
		return tx.Commit()
	}

	result, err = tx.Exec(`UPDATE messages SET score = score + 1 WHERE id = ?;`, postID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	updated, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if updated == 0 {
		_ = tx.Rollback()
		return errors.New("post not found")
	}

	return tx.Commit()
}

func (a *App) applyCommentUpvote(voterPubkey string, commentID string, postID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	voterPubkey = strings.TrimSpace(voterPubkey)
	commentID = strings.TrimSpace(commentID)
	postID = strings.TrimSpace(postID)
	if voterPubkey == "" || commentID == "" {
		return errors.New("voter pubkey and comment id are required")
	}

	if postID == "" {
		if err := a.db.QueryRow(`SELECT post_id FROM comments WHERE id = ?;`, commentID).Scan(&postID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("comment not found")
			}
			return err
		}
	}

	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	result, err := tx.Exec(`
		INSERT INTO comment_votes (comment_id, voter_pubkey, timestamp)
		VALUES (?, ?, ?)
		ON CONFLICT(comment_id, voter_pubkey) DO NOTHING;
	`, commentID, voterPubkey, time.Now().Unix())
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	inserted, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if inserted == 0 {
		return tx.Commit()
	}

	result, err = tx.Exec(`UPDATE comments SET score = score + 1 WHERE id = ? AND post_id = ?;`, commentID, postID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	updated, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if updated == 0 {
		_ = tx.Rollback()
		return errors.New("comment not found")
	}

	return tx.Commit()
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

func ensureBlobQuotaWithLRU(tx *sql.Tx, zone string, quota int64, incomingBytes int64, incomingContentCID string) error {
	if incomingBytes > quota {
		return errors.New("content blob exceeds zone quota")
	}

	incomingContentCID = strings.TrimSpace(incomingContentCID)
	if incomingContentCID == "" {
		return errors.New("content cid is required")
	}

	var existingCount int
	if err := tx.QueryRow(`
		SELECT COUNT(1)
		FROM content_blobs cb
		WHERE cb.content_cid = ?
			AND EXISTS (
				SELECT 1 FROM messages m
				WHERE m.zone = ? AND m.content_cid = cb.content_cid
			);
	`, incomingContentCID, zone).Scan(&existingCount); err != nil {
		return err
	}

	effectiveIncoming := incomingBytes
	if existingCount > 0 {
		effectiveIncoming = 0
	}

	var total int64
	if err := tx.QueryRow(`
		SELECT COALESCE(SUM(cb.size_bytes), 0)
		FROM content_blobs cb
		WHERE EXISTS (
			SELECT 1 FROM messages m
			WHERE m.zone = ? AND m.content_cid = cb.content_cid
		);
	`, zone).Scan(&total); err != nil {
		return err
	}

	if total+effectiveIncoming <= quota {
		return nil
	}

	rows, err := tx.Query(`
		SELECT cb.content_cid, cb.size_bytes
		FROM content_blobs cb
		WHERE cb.pinned = 0
			AND EXISTS (
				SELECT 1 FROM messages m
				WHERE m.zone = ? AND m.content_cid = cb.content_cid
			)
		ORDER BY cb.last_accessed_at ASC, cb.created_at ASC;
	`, zone)
	if err != nil {
		return err
	}
	defer rows.Close()

	type candidate struct {
		cid  string
		size int64
	}
	candidates := make([]candidate, 0)
	for rows.Next() {
		var c candidate
		if scanErr := rows.Scan(&c.cid, &c.size); scanErr != nil {
			return scanErr
		}
		candidates = append(candidates, c)
	}
	if err = rows.Err(); err != nil {
		return err
	}

	for _, c := range candidates {
		if total+effectiveIncoming <= quota {
			return nil
		}
		if c.cid == incomingContentCID {
			continue
		}

		var zoneCount int
		if err = tx.QueryRow(`SELECT COUNT(DISTINCT zone) FROM messages WHERE content_cid = ?;`, c.cid).Scan(&zoneCount); err != nil {
			return err
		}
		if zoneCount > 1 {
			continue
		}

		result, delErr := tx.Exec(`DELETE FROM content_blobs WHERE content_cid = ? AND pinned = 0;`, c.cid)
		if delErr != nil {
			return delErr
		}
		affected, affectedErr := result.RowsAffected()
		if affectedErr != nil {
			return affectedErr
		}
		if affected > 0 {
			total -= c.size
		}
	}

	if total+effectiveIncoming > quota {
		return errors.New("content blob quota exceeded and no evictable records")
	}

	return nil
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

	var existingTimestamp int64
	err := a.db.QueryRow(`
		SELECT timestamp
		FROM moderation
		WHERE target_pubkey = ?;
	`, targetPubkey).Scan(&existingTimestamp)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if err == nil && existingTimestamp > timestamp {
		if _, logErr := a.db.Exec(`
			INSERT INTO moderation_logs (target_pubkey, action, source_admin, timestamp, reason, result)
			VALUES (?, ?, ?, ?, ?, 'ignored_older');
		`, targetPubkey, action, sourceAdmin, timestamp, reason); logErr != nil {
			return logErr
		}
		return nil
	}

	_, err = a.db.Exec(`
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

	if _, err = a.db.Exec(`
		INSERT INTO moderation_logs (target_pubkey, action, source_admin, timestamp, reason, result)
		VALUES (?, ?, ?, ?, ?, 'applied');
	`, targetPubkey, action, sourceAdmin, timestamp, reason); err != nil {
		return err
	}

	policy, err := a.GetGovernancePolicy()
	if err != nil {
		return err
	}

	if action == "SHADOW_BAN" && policy.HideHistoryOnShadowBan {
		if _, err = a.db.Exec(`
			UPDATE messages
			SET visibility = 'shadowed'
			WHERE pubkey = ? AND zone = 'public';
		`, targetPubkey); err != nil {
			return err
		}
	}

	if action == "UNBAN" {
		if _, err = a.db.Exec(`
			UPDATE messages
			SET visibility = 'normal'
			WHERE pubkey = ? AND zone = 'public';
		`, targetPubkey); err != nil {
			return err
		}
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

func buildContentCID(body string) string {
	trimmed := strings.TrimSpace(body)
	hash := sha256.Sum256([]byte(trimmed))
	return "cidv1-" + hex.EncodeToString(hash[:])
}

func deriveTitle(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return ""
	}

	runes := []rune(trimmed)
	if len(runes) <= 20 {
		return string(runes)
	}

	return string(runes[:20])
}

func deriveBodyPreview(body string, maxRunes int) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return ""
	}
	if maxRunes <= 0 {
		maxRunes = 180
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes])
}

func normalizeSubID(subID string) string {
	normalized := strings.ToLower(strings.TrimSpace(subID))
	if normalized == "" {
		return defaultSubID
	}

	builder := strings.Builder{}
	for _, ch := range normalized {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			builder.WriteRune(ch)
			continue
		}
		if ch == ' ' {
			builder.WriteRune('-')
		}
	}

	clean := strings.Trim(builder.String(), "-_")
	if clean == "" {
		return defaultSubID
	}

	if len(clean) > 32 {
		clean = clean[:32]
	}

	return clean
}

func (a *App) saveLocalIdentity(identity Identity) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	mnemonic := strings.TrimSpace(identity.Mnemonic)
	pubkey := strings.TrimSpace(identity.PublicKey)
	if mnemonic == "" || pubkey == "" {
		return errors.New("identity is incomplete")
	}

	_, err := a.db.Exec(`
		INSERT INTO local_identity (id, mnemonic, pubkey, updated_at)
		VALUES (1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			mnemonic = excluded.mnemonic,
			pubkey = excluded.pubkey,
			updated_at = excluded.updated_at;
	`, mnemonic, pubkey, time.Now().Unix())

	return err
}

func (a *App) getLocalIdentity() (Identity, error) {
	if a.db == nil {
		return Identity{}, errors.New("database not initialized")
	}

	var identity Identity
	err := a.db.QueryRow(`SELECT mnemonic, pubkey FROM local_identity WHERE id = 1;`).Scan(&identity.Mnemonic, &identity.PublicKey)
	if errors.Is(err, sql.ErrNoRows) {
		return Identity{}, errors.New("identity not found")
	}
	if err != nil {
		return Identity{}, err
	}

	return identity, nil
}
