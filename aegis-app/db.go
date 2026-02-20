package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	xdraw "golang.org/x/image/draw"
	_ "modernc.org/sqlite"
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
	ImageCID    string `json:"imageCid"`
	ThumbCID    string `json:"thumbCid"`
	ImageMIME   string `json:"imageMime"`
	ImageSize   int64  `json:"imageSize"`
	ImageWidth  int    `json:"imageWidth"`
	ImageHeight int    `json:"imageHeight"`
	Content     string `json:"content"`
	Score       int64  `json:"score"`
	Timestamp   int64  `json:"timestamp"`
	Lamport     int64  `json:"lamport"`
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
	ImageCID    string `json:"imageCid"`
	ThumbCID    string `json:"thumbCid"`
	ImageMIME   string `json:"imageMime"`
	ImageSize   int64  `json:"imageSize"`
	ImageWidth  int    `json:"imageWidth"`
	ImageHeight int    `json:"imageHeight"`
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

type MediaBlob struct {
	ContentCID  string `json:"contentCid"`
	DataBase64  string `json:"dataBase64"`
	Mime        string `json:"mime"`
	SizeBytes   int64  `json:"sizeBytes"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	IsThumbnail bool   `json:"isThumbnail"`
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

type ProfileDetails struct {
	Pubkey      string `json:"pubkey"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarURL"`
	Bio         string `json:"bio"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type Comment struct {
	ID          string              `json:"id"`
	PostID      string              `json:"postId"`
	ParentID    string              `json:"parentId"`
	Pubkey      string              `json:"pubkey"`
	Body        string              `json:"body"`
	Attachments []CommentAttachment `json:"attachments,omitempty"`
	Score       int64               `json:"score"`
	Timestamp   int64               `json:"timestamp"`
	Lamport     int64               `json:"lamport"`
}

type CommentAttachment struct {
	Kind      string `json:"kind"`
	Ref       string `json:"ref"`
	Mime      string `json:"mime,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
}

type ModerationState struct {
	TargetPubkey string `json:"targetPubkey"`
	Action       string `json:"action"`
	SourceAdmin  string `json:"sourceAdmin"`
	Timestamp    int64  `json:"timestamp"`
	Lamport      int64  `json:"lamport"`
	Reason       string `json:"reason"`
}

type ModerationLog struct {
	ID           int64  `json:"id"`
	TargetPubkey string `json:"targetPubkey"`
	Action       string `json:"action"`
	SourceAdmin  string `json:"sourceAdmin"`
	Timestamp    int64  `json:"timestamp"`
	Lamport      int64  `json:"lamport"`
	Reason       string `json:"reason"`
	Result       string `json:"result"`
}

type GovernancePolicy struct {
	HideHistoryOnShadowBan bool `json:"hideHistoryOnShadowBan"`
}

type P2PConfig struct {
	ListenPort int      `json:"listenPort"`
	RelayPeers []string `json:"relayPeers"`
	AutoStart  bool     `json:"autoStart"`
	UpdatedAt  int64    `json:"updatedAt"`
}

type PrivacySettings struct {
	ShowOnlineStatus bool  `json:"showOnlineStatus"`
	AllowSearch      bool  `json:"allowSearch"`
	UpdatedAt        int64 `json:"updatedAt"`
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

type FeedStreamItem struct {
	Post                ForumMessage `json:"post"`
	Reason              string       `json:"reason"`
	IsSubscribed        bool         `json:"isSubscribed"`
	RecommendationScore float64      `json:"recommendationScore"`
}

type FeedStream struct {
	Items       []FeedStreamItem `json:"items"`
	Algorithm   string           `json:"algorithm"`
	GeneratedAt int64            `json:"generatedAt"`
}

type PostIndexPage struct {
	Items      []PostIndex `json:"items"`
	NextCursor string      `json:"nextCursor"`
}

type FavoriteOpRecord struct {
	OpID      string `json:"opId"`
	Pubkey    string `json:"pubkey"`
	PostID    string `json:"postId"`
	Op        string `json:"op"`
	CreatedAt int64  `json:"createdAt"`
	Signature string `json:"signature"`
}

type GovernanceAdmin struct {
	AdminPubkey string `json:"adminPubkey"`
	Role        string `json:"role"`
	Active      bool   `json:"active"`
}

type SyncPostDigest struct {
	ID          string `json:"id"`
	Pubkey      string `json:"pubkey"`
	Title       string `json:"title"`
	ContentCID  string `json:"content_cid"`
	ImageCID    string `json:"image_cid"`
	ThumbCID    string `json:"thumb_cid"`
	ImageMIME   string `json:"image_mime"`
	ImageSize   int64  `json:"image_size"`
	ImageWidth  int    `json:"image_width"`
	ImageHeight int    `json:"image_height"`
	Timestamp   int64  `json:"timestamp"`
	Lamport     int64  `json:"lamport"`
	SubID       string `json:"sub_id"`
}

type SyncCommentDigest struct {
	ID          string              `json:"id"`
	PostID      string              `json:"post_id"`
	ParentID    string              `json:"parent_id"`
	Pubkey      string              `json:"pubkey"`
	DisplayName string              `json:"display_name"`
	AvatarURL   string              `json:"avatar_url"`
	Body        string              `json:"body"`
	Attachments []CommentAttachment `json:"attachments,omitempty"`
	Score       int64               `json:"score"`
	Timestamp   int64               `json:"timestamp"`
	Lamport     int64               `json:"lamport"`
}

type IncomingMessage struct {
	Type                   string              `json:"type"`
	ID                     string              `json:"id"`
	Pubkey                 string              `json:"pubkey"`
	VoterPubkey            string              `json:"voter_pubkey"`
	PostID                 string              `json:"post_id"`
	CommentID              string              `json:"comment_id"`
	ParentID               string              `json:"parent_id"`
	DisplayName            string              `json:"display_name"`
	AvatarURL              string              `json:"avatar_url"`
	Title                  string              `json:"title"`
	Body                   string              `json:"body"`
	CommentAttachments     []CommentAttachment `json:"comment_attachments,omitempty"`
	ContentCID             string              `json:"content_cid"`
	ImageCID               string              `json:"image_cid"`
	ThumbCID               string              `json:"thumb_cid"`
	ImageMIME              string              `json:"image_mime"`
	ImageSize              int64               `json:"image_size"`
	ImageWidth             int                 `json:"image_width"`
	ImageHeight            int                 `json:"image_height"`
	ImageDataBase64        string              `json:"image_data_base64,omitempty"`
	IsThumbnail            bool                `json:"is_thumbnail,omitempty"`
	RequestID              string              `json:"request_id"`
	RequesterPeerID        string              `json:"requester_peer_id"`
	ResponderPeerID        string              `json:"responder_peer_id"`
	SyncSinceTimestamp     int64               `json:"sync_since_timestamp,omitempty"`
	SyncWindowSeconds      int64               `json:"sync_window_seconds,omitempty"`
	SyncBatchSize          int                 `json:"sync_batch_size,omitempty"`
	CommentSinceTs         int64               `json:"comment_since_ts,omitempty"`
	CommentBatchSize       int                 `json:"comment_batch_size,omitempty"`
	GovernanceSinceTs      int64               `json:"governance_since_ts,omitempty"`
	GovernanceBatchSize    int                 `json:"governance_batch_size,omitempty"`
	GovernanceLogSinceTs   int64               `json:"governance_log_since_ts,omitempty"`
	GovernanceLogLimit     int                 `json:"governance_log_limit,omitempty"`
	GovernanceStates       []ModerationState   `json:"governance_states,omitempty"`
	GovernanceLogs         []ModerationLog     `json:"governance_logs,omitempty"`
	FavoriteOpID           string              `json:"favorite_op_id,omitempty"`
	FavoriteOp             string              `json:"favorite_op,omitempty"`
	FavoriteSinceTs        int64               `json:"favorite_since_ts,omitempty"`
	FavoriteBatchSize      int                 `json:"favorite_batch_size,omitempty"`
	FavoriteOps            []FavoriteOpRecord  `json:"favorite_ops,omitempty"`
	Found                  bool                `json:"found"`
	SizeBytes              int64               `json:"size_bytes"`
	Content                string              `json:"content"`
	SubID                  string              `json:"sub_id"`
	SubTitle               string              `json:"sub_title"`
	SubDesc                string              `json:"sub_desc"`
	Timestamp              int64               `json:"timestamp"`
	Lamport                int64               `json:"lamport,omitempty"`
	Signature              string              `json:"signature"`
	TargetPubkey           string              `json:"target_pubkey"`
	AdminPubkey            string              `json:"admin_pubkey"`
	Reason                 string              `json:"reason"`
	Summaries              []SyncPostDigest    `json:"summaries,omitempty"`
	CommentSummaries       []SyncCommentDigest `json:"comment_summaries,omitempty"`
	KnownPeers             []KnownPeerExchange `json:"known_peers,omitempty"`
	RelayCapable           bool                `json:"relay_capable,omitempty"`
	PublicReachable        bool                `json:"public_reachable,omitempty"`
	HideHistoryOnShadowBan bool                `json:"hide_history_on_shadowban"`
}

type KnownPeerExchange struct {
	PeerID          string   `json:"peer_id"`
	Addrs           []string `json:"addrs"`
	RelayCapable    bool     `json:"relay_capable"`
	PublicReachable bool     `json:"public_reachable"`
	LastSeen        int64    `json:"last_seen"`
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

	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return err
	}

	if _, err = db.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		_ = db.Close()
		return err
	}

	if _, err = db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		_ = db.Close()
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
			image_cid TEXT NOT NULL DEFAULT '',
			thumb_cid TEXT NOT NULL DEFAULT '',
			image_mime TEXT NOT NULL DEFAULT '',
			image_size INTEGER NOT NULL DEFAULT 0,
			image_width INTEGER NOT NULL DEFAULT 0,
			image_height INTEGER NOT NULL DEFAULT 0,
			content TEXT NOT NULL,
			score INTEGER NOT NULL DEFAULT 0,
			timestamp INTEGER NOT NULL,
			lamport INTEGER NOT NULL DEFAULT 0,
			size_bytes INTEGER NOT NULL,
			zone TEXT NOT NULL CHECK (zone IN ('private', 'public')),
			sub_id TEXT NOT NULL DEFAULT 'general',
			is_protected INTEGER NOT NULL DEFAULT 0,
			visibility TEXT NOT NULL DEFAULT 'normal'
		);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_zone_timestamp ON messages(zone, timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_messages_pubkey_timestamp ON messages(pubkey, timestamp);`,
		`CREATE TABLE IF NOT EXISTS content_blobs (
			content_cid TEXT PRIMARY KEY,
			body TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			last_accessed_at INTEGER NOT NULL DEFAULT 0,
			pinned INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS media_blobs (
			content_cid TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			mime TEXT NOT NULL DEFAULT '',
			size_bytes INTEGER NOT NULL,
			width INTEGER NOT NULL DEFAULT 0,
			height INTEGER NOT NULL DEFAULT 0,
			is_thumbnail INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			last_accessed_at INTEGER NOT NULL DEFAULT 0,
			pinned INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS subs (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sub_subscriptions (
			sub_id TEXT PRIMARY KEY,
			subscribed_at INTEGER NOT NULL,
			FOREIGN KEY(sub_id) REFERENCES subs(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_sub_subscriptions_subscribed_at ON sub_subscriptions(subscribed_at DESC);`,
		`CREATE TABLE IF NOT EXISTS post_favorites_state (
			pubkey TEXT NOT NULL,
			post_id TEXT NOT NULL,
			state TEXT NOT NULL,
			updated_at INTEGER NOT NULL,
			last_op_id TEXT NOT NULL,
			PRIMARY KEY (pubkey, post_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_post_favorites_state_pubkey_updated_at ON post_favorites_state(pubkey, updated_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_post_favorites_state_post_id ON post_favorites_state(post_id);`,
		`CREATE TABLE IF NOT EXISTS post_favorite_ops (
			op_id TEXT PRIMARY KEY,
			pubkey TEXT NOT NULL,
			post_id TEXT NOT NULL,
			op TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			signature TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE INDEX IF NOT EXISTS idx_post_favorite_ops_pubkey_created_at ON post_favorite_ops(pubkey, created_at, op_id);`,
		`CREATE TABLE IF NOT EXISTS profiles (
			pubkey TEXT PRIMARY KEY,
			display_name TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			updated_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS profile_details (
			pubkey TEXT PRIMARY KEY,
			bio TEXT NOT NULL DEFAULT '',
			updated_at INTEGER NOT NULL,
			FOREIGN KEY(pubkey) REFERENCES profiles(pubkey) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS comments (
			id TEXT PRIMARY KEY,
			post_id TEXT NOT NULL,
			parent_id TEXT NOT NULL DEFAULT '',
			pubkey TEXT NOT NULL,
			body TEXT NOT NULL,
			attachments_json TEXT NOT NULL DEFAULT '[]',
			score INTEGER NOT NULL DEFAULT 0,
			timestamp INTEGER NOT NULL,
			lamport INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_post_timestamp ON comments(post_id, timestamp);`,
		`CREATE TABLE IF NOT EXISTS comment_media_refs (
			comment_id TEXT NOT NULL,
			content_cid TEXT NOT NULL,
			PRIMARY KEY (comment_id, content_cid)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_comment_media_refs_cid ON comment_media_refs(content_cid);`,
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
			lamport INTEGER NOT NULL DEFAULT 0,
			reason TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS moderation_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target_pubkey TEXT NOT NULL,
			action TEXT NOT NULL,
			source_admin TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			lamport INTEGER NOT NULL DEFAULT 0,
			reason TEXT NOT NULL DEFAULT '',
			result TEXT NOT NULL DEFAULT 'applied'
		);`,
		`CREATE INDEX IF NOT EXISTS idx_moderation_logs_timestamp ON moderation_logs(timestamp DESC);`,
		`CREATE TABLE IF NOT EXISTS governance_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS privacy_settings (
			pubkey TEXT PRIMARY KEY,
			show_online_status INTEGER NOT NULL,
			allow_search INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS p2p_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			listen_port INTEGER NOT NULL,
			relay_peers_json TEXT NOT NULL,
			auto_start INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS known_peers (
			peer_id TEXT PRIMARY KEY,
			addrs_json TEXT NOT NULL,
			last_seen INTEGER NOT NULL,
			success_count INTEGER NOT NULL DEFAULT 0,
			fail_count INTEGER NOT NULL DEFAULT 0,
			relay_capable INTEGER NOT NULL DEFAULT 0,
			public_reachable INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_known_peers_updated_at ON known_peers(updated_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_known_peers_last_seen ON known_peers(last_seen DESC);`,
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
		`CREATE TABLE IF NOT EXISTS logical_clock (
			scope TEXT PRIMARY KEY,
			value INTEGER NOT NULL,
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

	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN image_cid TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN thumb_cid TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN image_mime TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN image_size INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN image_width INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN image_height INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_content_cid ON messages(content_cid);`); err != nil {
		return err
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_image_cid ON messages(image_cid);`); err != nil {
		return err
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_media_blobs_last_accessed ON media_blobs(last_accessed_at);`); err != nil {
		return err
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

	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN lamport INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE comments ADD COLUMN lamport INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE comments ADD COLUMN attachments_json TEXT NOT NULL DEFAULT '[]';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS comment_media_refs (
			comment_id TEXT NOT NULL,
			content_cid TEXT NOT NULL,
			PRIMARY KEY (comment_id, content_cid)
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_comment_media_refs_cid ON comment_media_refs(content_cid);`); err != nil {
		return err
	}

	if _, err := db.Exec(`ALTER TABLE moderation ADD COLUMN lamport INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}

	if _, err := db.Exec(`ALTER TABLE moderation_logs ADD COLUMN lamport INTEGER NOT NULL DEFAULT 0;`); err != nil {
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

	if _, err := db.Exec(`UPDATE messages SET lamport = timestamp WHERE lamport = 0;`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE comments SET lamport = timestamp WHERE lamport = 0;`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE moderation SET lamport = timestamp WHERE lamport = 0;`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE moderation_logs SET lamport = timestamp WHERE lamport = 0;`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		INSERT INTO logical_clock (scope, value, updated_at)
		SELECT 'global',
			MAX(COALESCE(mv, 0), COALESCE(cv, 0), COALESCE(gv, 0), COALESCE(glv, 0)),
			?
		FROM (
			SELECT (SELECT MAX(lamport) FROM messages) AS mv,
			       (SELECT MAX(lamport) FROM comments) AS cv,
			       (SELECT MAX(lamport) FROM moderation) AS gv,
			       (SELECT MAX(lamport) FROM moderation_logs) AS glv
		)
		WHERE 1 = 1
		ON CONFLICT(scope) DO NOTHING;
	`, time.Now().Unix()); err != nil {
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
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_messages_lamport ON messages(lamport);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_comments_lamport ON comments(lamport);`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_moderation_lamport ON moderation(lamport);`); err != nil {
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

func (a *App) GetFeedStream(limit int) (FeedStream, error) {
	return a.GetFeedStreamWithStrategy(limit, "hot-v1")
}

func (a *App) GetFeedStreamWithStrategy(limit int, algorithm string) (FeedStream, error) {
	if a.db == nil {
		return FeedStream{}, errors.New("database not initialized")
	}

	now := time.Now().Unix()
	limit = normalizeFeedStreamLimit(limit)
	algorithm = normalizeFeedStreamAlgorithm(algorithm)

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	subscribedSubIDs, err := a.listSubscribedSubIDs()
	if err != nil {
		return FeedStream{}, err
	}

	subscribedQuota := int(math.Ceil(float64(limit) * 0.7))
	if subscribedQuota < 1 {
		subscribedQuota = 1
	}
	recommendedQuota := limit - subscribedQuota
	if recommendedQuota < 0 {
		recommendedQuota = 0
	}

	subscribedPosts := make([]ForumMessage, 0)
	if len(subscribedSubIDs) > 0 {
		subscribedPosts, err = a.queryPostsBySubSet(viewerPubkey, subscribedSubIDs, subscribedQuota*3)
		if err != nil {
			return FeedStream{}, err
		}
	}

	recommendedPosts, err := a.queryRecommendedPosts(viewerPubkey, subscribedSubIDs, max(limit*4, 40))
	if err != nil {
		return FeedStream{}, err
	}

	items := make([]FeedStreamItem, 0, limit)
	seen := make(map[string]struct{}, limit)

	si := 0
	ri := 0
	for len(items) < limit && (si < len(subscribedPosts) || ri < len(recommendedPosts)) {
		appendedSubscribed := 0
		for appendedSubscribed < 2 && si < len(subscribedPosts) && len(items) < limit && countFeedItemsByReason(items, "subscribed") < subscribedQuota {
			post := subscribedPosts[si]
			si++
			if _, exists := seen[post.ID]; exists {
				continue
			}
			seen[post.ID] = struct{}{}
			items = append(items, FeedStreamItem{
				Post:                post,
				Reason:              "subscribed",
				IsSubscribed:        true,
				RecommendationScore: scoreFeedRecommendation(post, now, algorithm),
			})
			appendedSubscribed++
		}

		for ri < len(recommendedPosts) && len(items) < limit && countFeedItemsByReason(items, "recommended_hot") < recommendedQuota {
			post := recommendedPosts[ri]
			ri++
			if _, exists := seen[post.ID]; exists {
				continue
			}
			seen[post.ID] = struct{}{}
			items = append(items, FeedStreamItem{
				Post:                post,
				Reason:              "recommended_hot",
				IsSubscribed:        false,
				RecommendationScore: scoreFeedRecommendation(post, now, algorithm),
			})
			break
		}

		if si >= len(subscribedPosts) && ri < len(recommendedPosts) {
			for ri < len(recommendedPosts) && len(items) < limit {
				post := recommendedPosts[ri]
				ri++
				if _, exists := seen[post.ID]; exists {
					continue
				}
				seen[post.ID] = struct{}{}
				items = append(items, FeedStreamItem{
					Post:                post,
					Reason:              "recommended_hot",
					IsSubscribed:        false,
					RecommendationScore: scoreFeedRecommendation(post, now, algorithm),
				})
			}
		}

		if ri >= len(recommendedPosts) && si < len(subscribedPosts) {
			for si < len(subscribedPosts) && len(items) < limit {
				post := subscribedPosts[si]
				si++
				if _, exists := seen[post.ID]; exists {
					continue
				}
				seen[post.ID] = struct{}{}
				items = append(items, FeedStreamItem{
					Post:                post,
					Reason:              "subscribed",
					IsSubscribed:        true,
					RecommendationScore: scoreFeedRecommendation(post, now, algorithm),
				})
			}
		}
	}

	return FeedStream{
		Items:       items,
		Algorithm:   algorithm,
		GeneratedAt: now,
	}, nil
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
		SELECT id, pubkey, title, SUBSTR(body, 1, 140) AS body_preview, content_cid, image_cid, thumb_cid, image_mime, image_size, image_width, image_height, score, timestamp, zone, sub_id, visibility
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
			&item.ImageCID,
			&item.ThumbCID,
			&item.ImageMIME,
			&item.ImageSize,
			&item.ImageWidth,
			&item.ImageHeight,
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

func (a *App) GetPostIndexByID(postID string) (PostIndex, error) {
	if a.db == nil {
		return PostIndex{}, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return PostIndex{}, errors.New("post id is required")
	}

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	var item PostIndex
	err := a.db.QueryRow(`
		SELECT id, pubkey, title, SUBSTR(body, 1, 140) AS body_preview, content_cid, image_cid, thumb_cid, image_mime, image_size, image_width, image_height, score, timestamp, zone, sub_id, visibility
		FROM messages
		WHERE id = ?
		  AND (
			(zone = 'public' AND (visibility = 'normal' OR pubkey = ?))
			OR (zone = 'private' AND pubkey = ?)
		  )
		LIMIT 1;
	`, postID, viewerPubkey, viewerPubkey).Scan(
		&item.ID,
		&item.Pubkey,
		&item.Title,
		&item.BodyPreview,
		&item.ContentCID,
		&item.ImageCID,
		&item.ThumbCID,
		&item.ImageMIME,
		&item.ImageSize,
		&item.ImageWidth,
		&item.ImageHeight,
		&item.Score,
		&item.Timestamp,
		&item.Zone,
		&item.SubID,
		&item.Visibility,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return PostIndex{}, errors.New("post not found")
	}
	if err != nil {
		return PostIndex{}, err
	}

	return item, nil
}

func (a *App) GetMyPosts(limit int, cursor string) (PostIndexPage, error) {
	if a.db == nil {
		return PostIndexPage{}, errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return PostIndexPage{}, err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return PostIndexPage{}, errors.New("identity pubkey is empty")
	}

	limit = normalizeMyPostsLimit(limit)
	cursorTs, cursorPostID, err := decodeMyPostsCursor(cursor)
	if err != nil {
		return PostIndexPage{}, err
	}

	args := []interface{}{pubkey}
	query := `
		SELECT
			id,
			pubkey,
			title,
			SUBSTR(body, 1, 140) AS body_preview,
			content_cid,
			image_cid,
			thumb_cid,
			image_mime,
			image_size,
			image_width,
			image_height,
			score,
			timestamp,
			zone,
			sub_id,
			visibility
		FROM messages
		WHERE pubkey = ?
	`
	if cursorTs > 0 && cursorPostID != "" {
		query += `
		  AND (timestamp < ? OR (timestamp = ? AND id < ?))
		`
		args = append(args, cursorTs, cursorTs, cursorPostID)
	}
	query += `
		ORDER BY timestamp DESC, id DESC
		LIMIT ?;
	`
	args = append(args, limit+1)

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return PostIndexPage{}, err
	}
	defer rows.Close()

	resultRows := make([]PostIndex, 0, limit+1)
	for rows.Next() {
		var row PostIndex
		if err = rows.Scan(
			&row.ID,
			&row.Pubkey,
			&row.Title,
			&row.BodyPreview,
			&row.ContentCID,
			&row.ImageCID,
			&row.ThumbCID,
			&row.ImageMIME,
			&row.ImageSize,
			&row.ImageWidth,
			&row.ImageHeight,
			&row.Score,
			&row.Timestamp,
			&row.Zone,
			&row.SubID,
			&row.Visibility,
		); err != nil {
			return PostIndexPage{}, err
		}
		resultRows = append(resultRows, row)
	}
	if err = rows.Err(); err != nil {
		return PostIndexPage{}, err
	}

	page := PostIndexPage{
		Items:      make([]PostIndex, 0, min(limit, len(resultRows))),
		NextCursor: "",
	}

	if len(resultRows) > limit {
		cursorRow := resultRows[limit-1]
		page.NextCursor = encodeMyPostsCursor(cursorRow.Timestamp, cursorRow.ID)
		resultRows = resultRows[:limit]
	}

	page.Items = append(page.Items, resultRows...)
	return page, nil
}

func (a *App) GetPostBodyByCID(contentCID string) (PostBodyBlob, error) {
	if a.db == nil {
		return PostBodyBlob{}, errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return PostBodyBlob{}, errors.New("content cid is required")
	}

	body, err := a.getContentBlobLocal(contentCID)
	if err == nil {
		a.noteBlobCacheHit()
		return body, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return PostBodyBlob{}, err
	}
	a.noteBlobCacheMiss()

	status := a.GetP2PStatus()
	if !status.Started {
		return PostBodyBlob{}, errors.New("content not found")
	}

	maxAttempts := resolveFetchRetryAttempts()
	var fetchErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fetchErr = a.fetchContentBlobFromNetwork(contentCID, 4*time.Second)
		if fetchErr == nil {
			break
		}
		if attempt < maxAttempts {
			time.Sleep(150 * time.Millisecond)
		}
	}
	if fetchErr != nil {
		return PostBodyBlob{}, fetchErr
	}

	body, err = a.getContentBlobLocal(contentCID)
	if errors.Is(err, sql.ErrNoRows) {
		return PostBodyBlob{}, errors.New("content not found")
	}
	if err != nil {
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

func (a *App) GetMediaByCID(contentCID string) (MediaBlob, error) {
	if a.db == nil {
		return MediaBlob{}, errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return MediaBlob{}, errors.New("media cid is required")
	}

	media, err := a.getMediaBlobLocal(contentCID)
	if err == nil {
		a.noteBlobCacheHit()
		return media, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return MediaBlob{}, err
	}
	a.noteBlobCacheMiss()

	status := a.GetP2PStatus()
	if !status.Started {
		return MediaBlob{}, errors.New("media not found")
	}

	maxAttempts := resolveFetchRetryAttempts()
	var fetchErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fetchErr = a.fetchMediaBlobFromNetwork(contentCID, 5*time.Second)
		if fetchErr == nil {
			break
		}
		if attempt < maxAttempts {
			time.Sleep(150 * time.Millisecond)
		}
	}
	if fetchErr != nil {
		return MediaBlob{}, fetchErr
	}

	media, err = a.getMediaBlobLocal(contentCID)
	if errors.Is(err, sql.ErrNoRows) {
		return MediaBlob{}, errors.New("media not found")
	}
	if err != nil {
		return MediaBlob{}, err
	}

	return media, nil
}

func (a *App) GetPostMediaByID(postID string) (MediaBlob, error) {
	if a.db == nil {
		return MediaBlob{}, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return MediaBlob{}, errors.New("post id is required")
	}

	var imageCID string
	err := a.db.QueryRow(`SELECT image_cid FROM messages WHERE id = ?;`, postID).Scan(&imageCID)
	if errors.Is(err, sql.ErrNoRows) {
		return MediaBlob{}, errors.New("post not found")
	}
	if err != nil {
		return MediaBlob{}, err
	}
	imageCID = strings.TrimSpace(imageCID)
	if imageCID == "" {
		return MediaBlob{}, errors.New("post has no image")
	}

	return a.GetMediaByCID(imageCID)
}

func (a *App) getMediaBlobRawLocal(contentCID string) (MediaBlob, []byte, error) {
	var media MediaBlob
	var raw []byte
	var isThumb int
	err := a.db.QueryRow(`
		SELECT content_cid, data, mime, size_bytes, width, height, is_thumbnail
		FROM media_blobs
		WHERE content_cid = ?;
	`, contentCID).Scan(&media.ContentCID, &raw, &media.Mime, &media.SizeBytes, &media.Width, &media.Height, &isThumb)
	if err != nil {
		return MediaBlob{}, nil, err
	}

	media.IsThumbnail = isThumb == 1
	media.DataBase64 = base64.StdEncoding.EncodeToString(raw)

	if _, err = a.db.Exec(`UPDATE media_blobs SET last_accessed_at = ? WHERE content_cid = ?;`, time.Now().Unix(), contentCID); err != nil {
		return MediaBlob{}, nil, err
	}

	return media, raw, nil
}

func (a *App) getMediaBlobLocal(contentCID string) (MediaBlob, error) {
	media, _, err := a.getMediaBlobRawLocal(contentCID)
	if err != nil {
		return MediaBlob{}, err
	}
	return media, nil
}

func (a *App) upsertMediaBlobRaw(contentCID string, mime string, data []byte, width int, height int, isThumbnail bool) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	mime = strings.TrimSpace(mime)
	if contentCID == "" || len(data) == 0 {
		return errors.New("invalid media blob")
	}

	thumbFlag := 0
	if isThumbnail {
		thumbFlag = 1
	}

	now := time.Now().Unix()
	_, err := a.db.Exec(`
		INSERT INTO media_blobs (content_cid, data, mime, size_bytes, width, height, is_thumbnail, created_at, last_accessed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(content_cid) DO UPDATE SET
			data = excluded.data,
			mime = excluded.mime,
			size_bytes = excluded.size_bytes,
			width = excluded.width,
			height = excluded.height,
			is_thumbnail = excluded.is_thumbnail,
			last_accessed_at = excluded.last_accessed_at;
	`, contentCID, data, mime, int64(len(data)), width, height, thumbFlag, now, now)
	return err
}

func (a *App) getContentBlobLocal(contentCID string) (PostBodyBlob, error) {
	var body PostBodyBlob
	err := a.db.QueryRow(`
		SELECT content_cid, body, size_bytes
		FROM content_blobs
		WHERE content_cid = ?;
	`, contentCID).Scan(&body.ContentCID, &body.Body, &body.SizeBytes)
	if err != nil {
		return PostBodyBlob{}, err
	}

	if _, err = a.db.Exec(`UPDATE content_blobs SET last_accessed_at = ? WHERE content_cid = ?;`, time.Now().Unix(), contentCID); err != nil {
		return PostBodyBlob{}, err
	}

	return body, nil
}

func (a *App) canServeContentBlobToNetwork(contentCID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return false, nil
	}

	rows, err := a.db.Query(`
		SELECT id, pubkey, timestamp, lamport
		FROM messages
		WHERE zone = 'public' AND visibility = 'normal' AND content_cid = ?;
	`, contentCID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id        string
			pubkey    string
			timestamp int64
			lamport   int64
		)
		if err = rows.Scan(&id, &pubkey, &timestamp, &lamport); err != nil {
			return false, err
		}

		allowed, allowErr := a.shouldAcceptPublicContent(pubkey, lamport, timestamp, id, "")
		if allowErr != nil {
			return false, allowErr
		}
		if allowed {
			return true, nil
		}
	}

	if err = rows.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func (a *App) canServeMediaBlobToNetwork(contentCID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return false, nil
	}

	rows, err := a.db.Query(`
		SELECT id, pubkey, timestamp, lamport
		FROM messages
		WHERE zone = 'public' AND visibility = 'normal' AND (image_cid = ? OR thumb_cid = ?);
	`, contentCID, contentCID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id        string
			pubkey    string
			timestamp int64
			lamport   int64
		)
		if err = rows.Scan(&id, &pubkey, &timestamp, &lamport); err != nil {
			return false, err
		}

		allowed, allowErr := a.shouldAcceptPublicContent(pubkey, lamport, timestamp, id, "")
		if allowErr != nil {
			return false, allowErr
		}
		if allowed {
			return true, nil
		}
	}

	if err = rows.Err(); err != nil {
		return false, err
	}

	commentRows, err := a.db.Query(`
		SELECT c.id, c.pubkey, c.timestamp, c.lamport
		FROM comment_media_refs r
		JOIN comments c ON c.id = r.comment_id
		JOIN messages m ON m.id = c.post_id
		WHERE r.content_cid = ? AND m.zone = 'public' AND m.visibility = 'normal';
	`, contentCID)
	if err != nil {
		return false, err
	}
	defer commentRows.Close()

	for commentRows.Next() {
		var (
			id        string
			pubkey    string
			timestamp int64
			lamport   int64
		)
		if err = commentRows.Scan(&id, &pubkey, &timestamp, &lamport); err != nil {
			return false, err
		}

		allowed, allowErr := a.shouldAcceptPublicContent(pubkey, lamport, timestamp, id, "")
		if allowErr != nil {
			return false, allowErr
		}
		if allowed {
			return true, nil
		}
	}

	if err = commentRows.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func (a *App) upsertContentBlob(contentCID string, body string, sizeBytes int64) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	body = strings.TrimSpace(body)
	if contentCID == "" || body == "" {
		return errors.New("invalid content blob")
	}
	if sizeBytes <= 0 {
		sizeBytes = int64(len([]byte(body)))
	}

	now := time.Now().Unix()
	_, err := a.db.Exec(`
		INSERT INTO content_blobs (content_cid, body, size_bytes, created_at, last_accessed_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(content_cid) DO UPDATE SET
			body = excluded.body,
			size_bytes = excluded.size_bytes,
			last_accessed_at = excluded.last_accessed_at;
	`, contentCID, body, sizeBytes, now, now)
	return err
}

func (a *App) listRecentPublicPostDigests(limit int) ([]SyncPostDigest, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 500 {
		limit = 200
	}

	rows, err := a.db.Query(`
		SELECT id, pubkey, title, content_cid, image_cid, thumb_cid, image_mime, image_size, image_width, image_height, timestamp, sub_id
		FROM messages
		WHERE zone = 'public' AND visibility = 'normal' AND content_cid != ''
		ORDER BY timestamp DESC
		LIMIT ?;
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SyncPostDigest, 0, limit)
	for rows.Next() {
		var digest SyncPostDigest
		if err = rows.Scan(&digest.ID, &digest.Pubkey, &digest.Title, &digest.ContentCID, &digest.ImageCID, &digest.ThumbCID, &digest.ImageMIME, &digest.ImageSize, &digest.ImageWidth, &digest.ImageHeight, &digest.Timestamp, &digest.SubID); err != nil {
			return nil, err
		}
		result = append(result, digest)
	}

	return result, rows.Err()
}

func (a *App) getLatestPublicPostTimestamp() (int64, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}

	var latest sql.NullInt64
	if err := a.db.QueryRow(`
		SELECT MAX(timestamp)
		FROM messages
		WHERE zone = 'public' AND visibility = 'normal';
	`).Scan(&latest); err != nil {
		return 0, err
	}

	if !latest.Valid {
		return 0, nil
	}

	return latest.Int64, nil
}

func (a *App) getLatestPublicCommentTimestamp() (int64, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}

	var latest sql.NullInt64
	if err := a.db.QueryRow(`
		SELECT MAX(c.timestamp)
		FROM comments c
		JOIN messages m ON m.id = c.post_id
		WHERE m.zone = 'public' AND m.visibility = 'normal';
	`).Scan(&latest); err != nil {
		return 0, err
	}

	if !latest.Valid {
		return 0, nil
	}

	return latest.Int64, nil
}

func (a *App) listPublicPostDigestsSince(sinceTimestamp int64, limit int) ([]SyncPostDigest, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 500 {
		limit = 200
	}

	policy, err := a.GetGovernancePolicy()
	if err != nil {
		return nil, err
	}

	var rows *sql.Rows
	err = nil
	if sinceTimestamp > 0 {
		if policy.HideHistoryOnShadowBan {
			rows, err = a.db.Query(`
				SELECT m.id, m.pubkey, m.title, m.content_cid, m.image_cid, m.thumb_cid, m.image_mime, m.image_size, m.image_width, m.image_height, m.timestamp, m.lamport, m.sub_id
				FROM messages m
				LEFT JOIN moderation mo ON mo.target_pubkey = m.pubkey
				WHERE m.zone = 'public' AND m.visibility = 'normal' AND m.content_cid != '' AND m.timestamp >= ?
				  AND (mo.action IS NULL OR UPPER(mo.action) != 'SHADOW_BAN')
				ORDER BY m.timestamp ASC
				LIMIT ?;
			`, sinceTimestamp, limit)
		} else {
			rows, err = a.db.Query(`
				SELECT m.id, m.pubkey, m.title, m.content_cid, m.image_cid, m.thumb_cid, m.image_mime, m.image_size, m.image_width, m.image_height, m.timestamp, m.lamport, m.sub_id
				FROM messages m
				LEFT JOIN moderation mo ON mo.target_pubkey = m.pubkey
				WHERE m.zone = 'public' AND m.visibility = 'normal' AND m.content_cid != '' AND m.timestamp >= ?
				  AND (
					mo.action IS NULL
					OR UPPER(mo.action) != 'SHADOW_BAN'
					OR m.lamport < mo.lamport
					OR (m.lamport = 0 OR mo.lamport = 0) AND m.timestamp < mo.timestamp
				  )
				ORDER BY m.timestamp ASC
				LIMIT ?;
			`, sinceTimestamp, limit)
		}
	} else {
		if policy.HideHistoryOnShadowBan {
			rows, err = a.db.Query(`
				SELECT m.id, m.pubkey, m.title, m.content_cid, m.image_cid, m.thumb_cid, m.image_mime, m.image_size, m.image_width, m.image_height, m.timestamp, m.lamport, m.sub_id
				FROM messages m
				LEFT JOIN moderation mo ON mo.target_pubkey = m.pubkey
				WHERE m.zone = 'public' AND m.visibility = 'normal' AND m.content_cid != ''
				  AND (mo.action IS NULL OR UPPER(mo.action) != 'SHADOW_BAN')
				ORDER BY m.timestamp DESC
				LIMIT ?;
			`, limit)
		} else {
			rows, err = a.db.Query(`
				SELECT m.id, m.pubkey, m.title, m.content_cid, m.image_cid, m.thumb_cid, m.image_mime, m.image_size, m.image_width, m.image_height, m.timestamp, m.lamport, m.sub_id
				FROM messages m
				LEFT JOIN moderation mo ON mo.target_pubkey = m.pubkey
				WHERE m.zone = 'public' AND m.visibility = 'normal' AND m.content_cid != ''
				  AND (
					mo.action IS NULL
					OR UPPER(mo.action) != 'SHADOW_BAN'
					OR m.lamport < mo.lamport
					OR (m.lamport = 0 OR mo.lamport = 0) AND m.timestamp < mo.timestamp
				  )
				ORDER BY m.timestamp DESC
				LIMIT ?;
			`, limit)
		}
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SyncPostDigest, 0, limit)
	for rows.Next() {
		var digest SyncPostDigest
		if err = rows.Scan(&digest.ID, &digest.Pubkey, &digest.Title, &digest.ContentCID, &digest.ImageCID, &digest.ThumbCID, &digest.ImageMIME, &digest.ImageSize, &digest.ImageWidth, &digest.ImageHeight, &digest.Timestamp, &digest.Lamport, &digest.SubID); err != nil {
			return nil, err
		}
		result = append(result, digest)
	}

	return result, rows.Err()
}

func (a *App) listPublicCommentDigestsSince(sinceTimestamp int64, limit int) ([]SyncCommentDigest, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 500 {
		limit = 200
	}

	policy, policyErr := a.GetGovernancePolicy()
	if policyErr != nil {
		return nil, policyErr
	}

	var (
		rows *sql.Rows
		err  error
	)
	if sinceTimestamp > 0 {
		if policy.HideHistoryOnShadowBan {
			rows, err = a.db.Query(`
				SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.body, c.attachments_json, c.score, c.timestamp, c.lamport,
				       COALESCE(p.display_name, ''), COALESCE(p.avatar_url, '')
				FROM comments c
				JOIN messages m ON m.id = c.post_id
				LEFT JOIN profiles p ON p.pubkey = c.pubkey
				LEFT JOIN moderation mo ON mo.target_pubkey = c.pubkey
				WHERE m.zone = 'public' AND m.visibility = 'normal' AND c.timestamp >= ?
				  AND (mo.action IS NULL OR UPPER(mo.action) != 'SHADOW_BAN')
				ORDER BY c.timestamp ASC
				LIMIT ?;
			`, sinceTimestamp, limit)
		} else {
			rows, err = a.db.Query(`
				SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.body, c.attachments_json, c.score, c.timestamp, c.lamport,
				       COALESCE(p.display_name, ''), COALESCE(p.avatar_url, '')
				FROM comments c
				JOIN messages m ON m.id = c.post_id
				LEFT JOIN profiles p ON p.pubkey = c.pubkey
				LEFT JOIN moderation mo ON mo.target_pubkey = c.pubkey
				WHERE m.zone = 'public' AND m.visibility = 'normal' AND c.timestamp >= ?
				  AND (
					mo.action IS NULL
					OR UPPER(mo.action) != 'SHADOW_BAN'
					OR c.lamport < mo.lamport
					OR (c.lamport = 0 OR mo.lamport = 0) AND c.timestamp < mo.timestamp
				  )
				ORDER BY c.timestamp ASC
				LIMIT ?;
			`, sinceTimestamp, limit)
		}
	} else {
		if policy.HideHistoryOnShadowBan {
			rows, err = a.db.Query(`
				SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.body, c.attachments_json, c.score, c.timestamp, c.lamport,
				       COALESCE(p.display_name, ''), COALESCE(p.avatar_url, '')
				FROM comments c
				JOIN messages m ON m.id = c.post_id
				LEFT JOIN profiles p ON p.pubkey = c.pubkey
				LEFT JOIN moderation mo ON mo.target_pubkey = c.pubkey
				WHERE m.zone = 'public' AND m.visibility = 'normal'
				  AND (mo.action IS NULL OR UPPER(mo.action) != 'SHADOW_BAN')
				ORDER BY c.timestamp DESC
				LIMIT ?;
			`, limit)
		} else {
			rows, err = a.db.Query(`
				SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.body, c.attachments_json, c.score, c.timestamp, c.lamport,
				       COALESCE(p.display_name, ''), COALESCE(p.avatar_url, '')
				FROM comments c
				JOIN messages m ON m.id = c.post_id
				LEFT JOIN profiles p ON p.pubkey = c.pubkey
				LEFT JOIN moderation mo ON mo.target_pubkey = c.pubkey
				WHERE m.zone = 'public' AND m.visibility = 'normal'
				  AND (
					mo.action IS NULL
					OR UPPER(mo.action) != 'SHADOW_BAN'
					OR c.lamport < mo.lamport
					OR (c.lamport = 0 OR mo.lamport = 0) AND c.timestamp < mo.timestamp
				  )
				ORDER BY c.timestamp DESC
				LIMIT ?;
			`, limit)
		}
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SyncCommentDigest, 0, limit)
	for rows.Next() {
		var item SyncCommentDigest
		var attachmentsJSON string
		if err = rows.Scan(
			&item.ID,
			&item.PostID,
			&item.ParentID,
			&item.Pubkey,
			&item.Body,
			&attachmentsJSON,
			&item.Score,
			&item.Timestamp,
			&item.Lamport,
			&item.DisplayName,
			&item.AvatarURL,
		); err != nil {
			return nil, err
		}
		item.Attachments = decodeCommentAttachmentsJSON(attachmentsJSON)
		result = append(result, item)
	}

	return result, rows.Err()
}

func (a *App) listPublicCommentDigestsByPostSince(postID string, sinceTimestamp int64, limit int) ([]SyncCommentDigest, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return []SyncCommentDigest{}, nil
	}

	if limit <= 0 || limit > 500 {
		limit = 200
	}

	policy, err := a.GetGovernancePolicy()
	if err != nil {
		return nil, err
	}

	var rows *sql.Rows
	if policy.HideHistoryOnShadowBan {
		rows, err = a.db.Query(`
			SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.body, c.attachments_json, c.score, c.timestamp, c.lamport,
			       COALESCE(p.display_name, ''), COALESCE(p.avatar_url, '')
			FROM comments c
			JOIN messages m ON m.id = c.post_id
			LEFT JOIN profiles p ON p.pubkey = c.pubkey
			LEFT JOIN moderation mo ON mo.target_pubkey = c.pubkey
			WHERE m.zone = 'public' AND m.visibility = 'normal' AND c.post_id = ? AND c.timestamp >= ?
			  AND (mo.action IS NULL OR UPPER(mo.action) != 'SHADOW_BAN')
			ORDER BY c.timestamp ASC
			LIMIT ?;
		`, postID, sinceTimestamp, limit)
	} else {
		rows, err = a.db.Query(`
			SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.body, c.attachments_json, c.score, c.timestamp, c.lamport,
			       COALESCE(p.display_name, ''), COALESCE(p.avatar_url, '')
			FROM comments c
			JOIN messages m ON m.id = c.post_id
			LEFT JOIN profiles p ON p.pubkey = c.pubkey
			LEFT JOIN moderation mo ON mo.target_pubkey = c.pubkey
			WHERE m.zone = 'public' AND m.visibility = 'normal' AND c.post_id = ? AND c.timestamp >= ?
			  AND (
				mo.action IS NULL
				OR UPPER(mo.action) != 'SHADOW_BAN'
				OR c.lamport < mo.lamport
				OR (c.lamport = 0 OR mo.lamport = 0) AND c.timestamp < mo.timestamp
			  )
			ORDER BY c.timestamp ASC
			LIMIT ?;
		`, postID, sinceTimestamp, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SyncCommentDigest, 0, limit)
	for rows.Next() {
		var item SyncCommentDigest
		var attachmentsJSON string
		if err = rows.Scan(
			&item.ID,
			&item.PostID,
			&item.ParentID,
			&item.Pubkey,
			&item.Body,
			&attachmentsJSON,
			&item.Score,
			&item.Timestamp,
			&item.Lamport,
			&item.DisplayName,
			&item.AvatarURL,
		); err != nil {
			return nil, err
		}
		item.Attachments = decodeCommentAttachmentsJSON(attachmentsJSON)
		result = append(result, item)
	}

	return result, rows.Err()
}

func (a *App) getLatestFavoriteOpTimestamp(pubkey string) (int64, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return 0, nil
	}

	var latest sql.NullInt64
	if err := a.db.QueryRow(`
		SELECT MAX(created_at)
		FROM post_favorite_ops
		WHERE pubkey = ?;
	`, pubkey).Scan(&latest); err != nil {
		return 0, err
	}
	if !latest.Valid {
		return 0, nil
	}

	return latest.Int64, nil
}

func (a *App) listFavoriteOpsSince(pubkey string, sinceTimestamp int64, limit int) ([]FavoriteOpRecord, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return []FavoriteOpRecord{}, nil
	}
	if sinceTimestamp < 0 {
		sinceTimestamp = 0
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	rows, err := a.db.Query(`
		SELECT op_id, pubkey, post_id, op, created_at, signature
		FROM post_favorite_ops
		WHERE pubkey = ? AND created_at >= ?
		ORDER BY created_at ASC, op_id ASC
		LIMIT ?;
	`, pubkey, sinceTimestamp, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]FavoriteOpRecord, 0, limit)
	for rows.Next() {
		var record FavoriteOpRecord
		if err = rows.Scan(
			&record.OpID,
			&record.Pubkey,
			&record.PostID,
			&record.Op,
			&record.CreatedAt,
			&record.Signature,
		); err != nil {
			return nil, err
		}
		result = append(result, record)
	}

	return result, rows.Err()
}

func (a *App) upsertPublicPostIndexFromDigest(digest SyncPostDigest) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	digest.ID = strings.TrimSpace(digest.ID)
	digest.Pubkey = strings.TrimSpace(digest.Pubkey)
	digest.Title = strings.TrimSpace(digest.Title)
	digest.ContentCID = strings.TrimSpace(digest.ContentCID)
	digest.ImageCID = strings.TrimSpace(digest.ImageCID)
	digest.ThumbCID = strings.TrimSpace(digest.ThumbCID)
	digest.ImageMIME = strings.TrimSpace(digest.ImageMIME)
	digest.SubID = normalizeSubID(digest.SubID)
	if digest.Timestamp <= 0 {
		digest.Timestamp = time.Now().Unix()
	}
	if digest.Lamport <= 0 {
		digest.Lamport = digest.Timestamp
	}

	if digest.ID == "" || digest.Pubkey == "" || digest.ContentCID == "" {
		return false, errors.New("invalid sync digest")
	}
	if digest.Title == "" {
		digest.Title = "Untitled"
	}

	result, err := a.db.Exec(`
		INSERT INTO messages (
			id, pubkey, title, body, content_cid, image_cid, thumb_cid, image_mime, image_size, image_width, image_height, content, score, timestamp, lamport, size_bytes, zone, sub_id, is_protected, visibility
		)
		VALUES (?, ?, ?, '', ?, ?, ?, ?, ?, ?, ?, '', 0, ?, ?, 0, 'public', ?, 0, 'normal')
		ON CONFLICT(id) DO UPDATE SET
			content_cid = CASE WHEN COALESCE(messages.content_cid, '') = '' THEN excluded.content_cid ELSE messages.content_cid END,
			image_cid = CASE WHEN COALESCE(messages.image_cid, '') = '' THEN excluded.image_cid ELSE messages.image_cid END,
			thumb_cid = CASE WHEN COALESCE(messages.thumb_cid, '') = '' THEN excluded.thumb_cid ELSE messages.thumb_cid END,
			image_mime = CASE WHEN COALESCE(messages.image_mime, '') = '' THEN excluded.image_mime ELSE messages.image_mime END,
			image_size = CASE WHEN COALESCE(messages.image_size, 0) = 0 THEN excluded.image_size ELSE messages.image_size END,
			image_width = CASE WHEN COALESCE(messages.image_width, 0) = 0 THEN excluded.image_width ELSE messages.image_width END,
			image_height = CASE WHEN COALESCE(messages.image_height, 0) = 0 THEN excluded.image_height ELSE messages.image_height END,
			lamport = CASE WHEN COALESCE(messages.lamport, 0) = 0 THEN excluded.lamport ELSE messages.lamport END,
			sub_id = CASE WHEN COALESCE(messages.sub_id, '') = '' THEN excluded.sub_id ELSE messages.sub_id END;
	`, digest.ID, digest.Pubkey, digest.Title, digest.ContentCID, digest.ImageCID, digest.ThumbCID, digest.ImageMIME, digest.ImageSize, digest.ImageWidth, digest.ImageHeight, digest.Timestamp, digest.Lamport, digest.SubID)
	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (a *App) hasContentBlobLocal(contentCID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return false, nil
	}

	var exists int
	err := a.db.QueryRow(`SELECT 1 FROM content_blobs WHERE content_cid = ? LIMIT 1;`, contentCID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (a *App) hasMediaBlobLocal(contentCID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return false, nil
	}

	var exists int
	err := a.db.QueryRow(`SELECT 1 FROM media_blobs WHERE content_cid = ? LIMIT 1;`, contentCID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
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

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	policy, policyErr := a.GetGovernancePolicy()
	hideHistoryOnShadowBan := true
	if policyErr == nil {
		hideHistoryOnShadowBan = policy.HideHistoryOnShadowBan
	}

	query := `
		SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.body, c.attachments_json, c.score, c.timestamp, c.lamport
		FROM comments c
		LEFT JOIN moderation m ON m.target_pubkey = c.pubkey
	`
	args := []interface{}{}
	query += `
		WHERE c.post_id = ?
	`
	args = append(args, postID)
	if hideHistoryOnShadowBan {
		query += `
		  AND (
			m.action IS NULL
			OR UPPER(m.action) != 'SHADOW_BAN'
			OR c.pubkey = ?
		  )
		`
		args = append(args, viewerPubkey)
	} else {
		query += `
		  AND (
			m.action IS NULL
			OR UPPER(m.action) != 'SHADOW_BAN'
			OR c.pubkey = ?
			OR c.lamport < m.lamport
			OR (c.lamport = 0 OR m.lamport = 0) AND c.timestamp < m.timestamp
		  )
		`
		args = append(args, viewerPubkey)
	}
	query += `
		ORDER BY c.timestamp ASC;
	`

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Comment, 0)
	for rows.Next() {
		var comment Comment
		var attachmentsJSON string
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.ParentID, &comment.Pubkey, &comment.Body, &attachmentsJSON, &comment.Score, &comment.Timestamp, &comment.Lamport); err != nil {
			return nil, err
		}
		comment.Attachments = decodeCommentAttachmentsJSON(attachmentsJSON)
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

func (a *App) SubscribeSub(subID string) (Sub, error) {
	if a.db == nil {
		return Sub{}, errors.New("database not initialized")
	}

	subID = normalizeSubID(subID)
	now := time.Now().Unix()

	var sub Sub
	err := a.db.QueryRow(`
		SELECT id, title, description, created_at
		FROM subs
		WHERE id = ?;
	`, subID).Scan(&sub.ID, &sub.Title, &sub.Description, &sub.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		sub, err = a.upsertSub(subID, subID, "", now)
		if err != nil {
			return Sub{}, err
		}
	} else if err != nil {
		return Sub{}, err
	}

	result, err := a.db.Exec(`
		INSERT INTO sub_subscriptions (sub_id, subscribed_at)
		VALUES (?, ?)
		ON CONFLICT(sub_id) DO NOTHING;
	`, subID, now)
	if err != nil {
		return Sub{}, err
	}

	if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 && a.ctx != nil {
		runtime.EventsEmit(a.ctx, "subs:subscriptions_updated")
	}

	return sub, nil
}

func (a *App) UnsubscribeSub(subID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	subID = normalizeSubID(subID)
	result, err := a.db.Exec(`DELETE FROM sub_subscriptions WHERE sub_id = ?;`, subID)
	if err != nil {
		return err
	}

	if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 && a.ctx != nil {
		runtime.EventsEmit(a.ctx, "subs:subscriptions_updated")
	}

	return nil
}

func (a *App) GetSubscribedSubs() ([]Sub, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT s.id, s.title, s.description, s.created_at
		FROM sub_subscriptions ss
		INNER JOIN subs s ON s.id = ss.sub_id
		ORDER BY ss.subscribed_at DESC, s.id ASC;
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

func (a *App) SearchSubs(keyword string, limit int) ([]Sub, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []Sub{}, nil
	}
	limit = normalizeSearchLimit(limit)

	lowerKeyword := strings.ToLower(keyword)
	pattern := "%" + lowerKeyword + "%"

	rows, err := a.db.Query(`
		SELECT id, title, description, created_at
		FROM subs
		WHERE LOWER(id) LIKE ?
		   OR LOWER(title) LIKE ?
		   OR LOWER(description) LIKE ?
		ORDER BY
			CASE
				WHEN LOWER(id) = ? THEN 0
				WHEN LOWER(title) = ? THEN 1
				ELSE 2
			END,
			created_at DESC
		LIMIT ?;
	`, pattern, pattern, pattern, lowerKeyword, lowerKeyword, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Sub, 0, limit)
	for rows.Next() {
		var sub Sub
		if err := rows.Scan(&sub.ID, &sub.Title, &sub.Description, &sub.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, sub)
	}

	return result, rows.Err()
}

func (a *App) SearchPosts(keyword string, subID string, limit int) ([]ForumMessage, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []ForumMessage{}, nil
	}
	limit = normalizeSearchLimit(limit)

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	lowerKeyword := strings.ToLower(keyword)
	pattern := "%" + lowerKeyword + "%"

	subID = strings.TrimSpace(subID)
	var rows *sql.Rows
	var err error
	if subID != "" {
		rows, err = a.db.Query(`
			SELECT m.id, m.pubkey, m.title, m.body, m.content_cid, m.content, m.score, m.timestamp, m.size_bytes, m.zone, m.sub_id, m.is_protected, m.visibility
			FROM messages m
			LEFT JOIN content_blobs cb ON cb.content_cid = m.content_cid
			WHERE m.zone = 'public'
			  AND (m.visibility = 'normal' OR m.pubkey = ?)
			  AND m.sub_id = ?
			  AND (
				LOWER(m.title) LIKE ?
				OR LOWER(m.body) LIKE ?
				OR LOWER(COALESCE(cb.body, '')) LIKE ?
			  )
			ORDER BY m.timestamp DESC
			LIMIT ?;
		`, viewerPubkey, normalizeSubID(subID), pattern, pattern, pattern, limit)
	} else {
		rows, err = a.db.Query(`
			SELECT m.id, m.pubkey, m.title, m.body, m.content_cid, m.content, m.score, m.timestamp, m.size_bytes, m.zone, m.sub_id, m.is_protected, m.visibility
			FROM messages m
			LEFT JOIN content_blobs cb ON cb.content_cid = m.content_cid
			WHERE m.zone = 'public'
			  AND (m.visibility = 'normal' OR m.pubkey = ?)
			  AND (
				LOWER(m.title) LIKE ?
				OR LOWER(m.body) LIKE ?
				OR LOWER(COALESCE(cb.body, '')) LIKE ?
			  )
			ORDER BY m.timestamp DESC
			LIMIT ?;
		`, viewerPubkey, pattern, pattern, pattern, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ForumMessage, 0, limit)
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
		result = append(result, message)
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

func (a *App) UpdateProfileDetails(displayName string, avatarURL string, bio string) (ProfileDetails, error) {
	if a.db == nil {
		return ProfileDetails{}, errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return ProfileDetails{}, err
	}

	updatedAt := time.Now().Unix()
	profile, err := a.upsertProfile(identity.PublicKey, displayName, avatarURL, updatedAt)
	if err != nil {
		return ProfileDetails{}, err
	}

	bio = strings.TrimSpace(bio)
	if len([]rune(bio)) > 160 {
		bio = string([]rune(bio)[:160])
	}

	_, err = a.db.Exec(`
		INSERT INTO profile_details (pubkey, bio, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(pubkey) DO UPDATE SET
			bio = excluded.bio,
			updated_at = excluded.updated_at;
	`, profile.Pubkey, bio, updatedAt)
	if err != nil {
		return ProfileDetails{}, err
	}

	return ProfileDetails{
		Pubkey:      profile.Pubkey,
		DisplayName: profile.DisplayName,
		AvatarURL:   profile.AvatarURL,
		Bio:         bio,
		UpdatedAt:   updatedAt,
	}, nil
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

func (a *App) GetProfileDetails(pubkey string) (ProfileDetails, error) {
	if a.db == nil {
		return ProfileDetails{}, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return ProfileDetails{}, errors.New("pubkey is required")
	}

	profile, err := a.GetProfile(pubkey)
	if err != nil {
		return ProfileDetails{}, err
	}

	var (
		bio              string
		detailsUpdatedAt int64
	)
	err = a.db.QueryRow(`
		SELECT bio, updated_at
		FROM profile_details
		WHERE pubkey = ?;
	`, pubkey).Scan(&bio, &detailsUpdatedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ProfileDetails{}, err
	}

	updatedAt := profile.UpdatedAt
	if detailsUpdatedAt > updatedAt {
		updatedAt = detailsUpdatedAt
	}

	return ProfileDetails{
		Pubkey:      profile.Pubkey,
		DisplayName: profile.DisplayName,
		AvatarURL:   profile.AvatarURL,
		Bio:         bio,
		UpdatedAt:   updatedAt,
	}, nil
}

func (a *App) GetPrivacySettings() (PrivacySettings, error) {
	if a.db == nil {
		return PrivacySettings{}, errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return PrivacySettings{}, err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return PrivacySettings{}, errors.New("identity pubkey is empty")
	}

	var (
		showOnlineStatus int
		allowSearch      int
		updatedAt        int64
	)
	err = a.db.QueryRow(`
		SELECT show_online_status, allow_search, updated_at
		FROM privacy_settings
		WHERE pubkey = ?;
	`, pubkey).Scan(&showOnlineStatus, &allowSearch, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return PrivacySettings{
			ShowOnlineStatus: true,
			AllowSearch:      true,
			UpdatedAt:        0,
		}, nil
	}
	if err != nil {
		return PrivacySettings{}, err
	}

	return PrivacySettings{
		ShowOnlineStatus: showOnlineStatus == 1,
		AllowSearch:      allowSearch == 1,
		UpdatedAt:        updatedAt,
	}, nil
}

func (a *App) SetPrivacySettings(showOnlineStatus bool, allowSearch bool) (PrivacySettings, error) {
	if a.db == nil {
		return PrivacySettings{}, errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return PrivacySettings{}, err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return PrivacySettings{}, errors.New("identity pubkey is empty")
	}

	updatedAt := time.Now().Unix()
	showOnlineStatusInt := 0
	if showOnlineStatus {
		showOnlineStatusInt = 1
	}
	allowSearchInt := 0
	if allowSearch {
		allowSearchInt = 1
	}

	_, err = a.db.Exec(`
		INSERT INTO privacy_settings (pubkey, show_online_status, allow_search, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(pubkey) DO UPDATE SET
			show_online_status = excluded.show_online_status,
			allow_search = excluded.allow_search,
			updated_at = excluded.updated_at;
	`, pubkey, showOnlineStatusInt, allowSearchInt, updatedAt)
	if err != nil {
		return PrivacySettings{}, err
	}

	return PrivacySettings{
		ShowOnlineStatus: showOnlineStatus,
		AllowSearch:      allowSearch,
		UpdatedAt:        updatedAt,
	}, nil
}

func (a *App) GetModerationState() ([]ModerationState, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT target_pubkey, action, source_admin, timestamp, lamport, reason
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
		if err := rows.Scan(&state.TargetPubkey, &state.Action, &state.SourceAdmin, &state.Timestamp, &state.Lamport, &state.Reason); err != nil {
			return nil, err
		}
		result = append(result, state)
	}

	return result, rows.Err()
}

func (a *App) getLatestModerationTimestamp() (int64, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}

	var latest sql.NullInt64
	if err := a.db.QueryRow(`SELECT MAX(timestamp) FROM moderation;`).Scan(&latest); err != nil {
		return 0, err
	}
	if !latest.Valid {
		return 0, nil
	}
	return latest.Int64, nil
}

func (a *App) listModerationSince(sinceTimestamp int64, limit int) ([]ModerationState, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 500 {
		limit = 200
	}
	if sinceTimestamp < 0 {
		sinceTimestamp = 0
	}

	rows, err := a.db.Query(`
		SELECT target_pubkey, action, source_admin, timestamp, lamport, reason
		FROM moderation
		WHERE timestamp >= ?
		ORDER BY timestamp ASC
		LIMIT ?;
	`, sinceTimestamp, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ModerationState, 0, limit)
	for rows.Next() {
		var row ModerationState
		if err = rows.Scan(&row.TargetPubkey, &row.Action, &row.SourceAdmin, &row.Timestamp, &row.Lamport, &row.Reason); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

func (a *App) getLatestAppliedModerationLogTimestamp() (int64, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}

	var latest sql.NullInt64
	if err := a.db.QueryRow(`SELECT MAX(timestamp) FROM moderation_logs WHERE result = 'applied';`).Scan(&latest); err != nil {
		return 0, err
	}
	if !latest.Valid {
		return 0, nil
	}
	return latest.Int64, nil
}

func (a *App) listAppliedModerationLogsSince(sinceTimestamp int64, limit int) ([]ModerationLog, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 500 {
		limit = 200
	}
	if sinceTimestamp < 0 {
		sinceTimestamp = 0
	}

	rows, err := a.db.Query(`
		SELECT id, target_pubkey, action, source_admin, timestamp, lamport, reason, result
		FROM moderation_logs
		WHERE result = 'applied' AND timestamp >= ?
		ORDER BY timestamp ASC, id ASC
		LIMIT ?;
	`, sinceTimestamp, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ModerationLog, 0, limit)
	for rows.Next() {
		var row ModerationLog
		if err = rows.Scan(&row.ID, &row.TargetPubkey, &row.Action, &row.SourceAdmin, &row.Timestamp, &row.Lamport, &row.Reason, &row.Result); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

func (a *App) insertModerationLogIfAbsent(log ModerationLog) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	log.TargetPubkey = strings.TrimSpace(log.TargetPubkey)
	log.Action = strings.ToUpper(strings.TrimSpace(log.Action))
	log.SourceAdmin = strings.TrimSpace(log.SourceAdmin)
	log.Reason = strings.TrimSpace(log.Reason)
	log.Result = strings.TrimSpace(log.Result)
	if log.Result == "" {
		log.Result = "applied"
	}
	if log.TargetPubkey == "" || log.SourceAdmin == "" || log.Action == "" {
		return false, errors.New("invalid moderation log payload")
	}
	if log.Timestamp <= 0 {
		log.Timestamp = time.Now().Unix()
	}
	if log.Lamport <= 0 {
		log.Lamport = log.Timestamp
	}

	var exists int
	err := a.db.QueryRow(`
		SELECT 1
		FROM moderation_logs
		WHERE target_pubkey = ?
		  AND action = ?
		  AND source_admin = ?
		  AND timestamp = ?
		  AND reason = ?
		  AND result = ?
		LIMIT 1;
	`, log.TargetPubkey, log.Action, log.SourceAdmin, log.Timestamp, log.Reason, log.Result).Scan(&exists)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}

	_, err = a.db.Exec(`
		INSERT INTO moderation_logs (target_pubkey, action, source_admin, timestamp, lamport, reason, result)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`, log.TargetPubkey, log.Action, log.SourceAdmin, log.Timestamp, log.Lamport, log.Reason, log.Result)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (a *App) GetModerationLogs(limit int) ([]ModerationLog, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 500 {
		limit = 100
	}

	rows, err := a.db.Query(`
		SELECT id, target_pubkey, action, source_admin, timestamp, lamport, reason, result
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
		if err = rows.Scan(&row.ID, &row.TargetPubkey, &row.Action, &row.SourceAdmin, &row.Timestamp, &row.Lamport, &row.Reason, &row.Result); err != nil {
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

	if hideHistoryOnShadowBan {
		if _, err = a.db.Exec(`
			UPDATE messages
			SET visibility = 'shadowed'
			WHERE zone = 'public'
			  AND pubkey IN (
				SELECT target_pubkey FROM moderation WHERE action = 'SHADOW_BAN'
			  );
		`); err != nil {
			return GovernancePolicy{}, err
		}
	} else {
		if _, err = a.db.Exec(`
			UPDATE messages
			SET visibility = 'normal'
			WHERE zone = 'public'
			  AND visibility = 'shadowed'
			  AND pubkey IN (
				SELECT target_pubkey FROM moderation WHERE action = 'SHADOW_BAN'
			  );
		`); err != nil {
			return GovernancePolicy{}, err
		}
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

	rows, err := a.db.Query(`
		SELECT content_cid, size_bytes
		FROM content_blobs;
	`)
	if err != nil {
		return StorageUsage{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid  string
			size int64
		)
		if err = rows.Scan(&cid, &size); err != nil {
			return StorageUsage{}, err
		}

		shareable, shareErr := a.canServeContentBlobToNetwork(cid)
		if shareErr != nil {
			return StorageUsage{}, shareErr
		}
		if shareable {
			usage.PublicUsedBytes += size
		} else {
			usage.PrivateUsedBytes += size
		}
	}
	if err = rows.Err(); err != nil {
		return StorageUsage{}, err
	}

	mediaRows, err := a.db.Query(`
		SELECT content_cid, size_bytes
		FROM media_blobs;
	`)
	if err != nil {
		return StorageUsage{}, err
	}
	defer mediaRows.Close()

	for mediaRows.Next() {
		var (
			cid  string
			size int64
		)
		if err = mediaRows.Scan(&cid, &size); err != nil {
			return StorageUsage{}, err
		}

		shareable, shareErr := a.canServeMediaBlobToNetwork(cid)
		if shareErr != nil {
			return StorageUsage{}, shareErr
		}
		if shareable {
			usage.PublicUsedBytes += size
		} else {
			usage.PrivateUsedBytes += size
		}
	}
	if err = mediaRows.Err(); err != nil {
		return StorageUsage{}, err
	}

	return usage, nil
}

func (a *App) nextLamport() (int64, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}

	now := time.Now().Unix()
	if _, err := a.db.Exec(`
		INSERT INTO logical_clock (scope, value, updated_at)
		VALUES ('global', 0, ?)
		ON CONFLICT(scope) DO NOTHING;
	`, now); err != nil {
		return 0, err
	}

	var lamport int64
	if err := a.db.QueryRow(`
		UPDATE logical_clock
		SET value = value + 1,
		    updated_at = ?
		WHERE scope = 'global'
		RETURNING value;
	`, now).Scan(&lamport); err != nil {
		return 0, err
	}

	return lamport, nil
}

func (a *App) observeLamport(incomingLamport int64) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	if incomingLamport < 0 {
		incomingLamport = 0
	}
	now := time.Now().Unix()
	if _, err := a.db.Exec(`
		INSERT INTO logical_clock (scope, value, updated_at)
		VALUES ('global', 0, ?)
		ON CONFLICT(scope) DO NOTHING;
	`, now); err != nil {
		return err
	}

	_, err := a.db.Exec(`
		UPDATE logical_clock
		SET value = CASE
			WHEN value > ? THEN value + 1
			ELSE ? + 1
		END,
		updated_at = ?
		WHERE scope = 'global';
	`, incomingLamport, incomingLamport, now)
	return err
}

func (a *App) normalizeIncomingLamport(incomingLamport int64, timestamp int64) (int64, error) {
	if timestamp <= 0 {
		timestamp = time.Now().Unix()
	}
	if incomingLamport <= 0 {
		incomingLamport = timestamp
	}
	if err := a.observeLamport(incomingLamport); err != nil {
		return 0, err
	}

	return incomingLamport, nil
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
		if err != nil {
			return err
		}
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "subs:updated")
		}
		return nil
	case "GOVERNANCE_POLICY_UPDATE":
		trusted, trustErr := a.isTrustedAdmin(message.AdminPubkey)
		if trustErr != nil {
			return trustErr
		}
		if !trusted {
			return errors.New("admin pubkey is not trusted")
		}
		_, policyErr := a.SetGovernancePolicy(message.HideHistoryOnShadowBan)
		return policyErr
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
	case messageTypeFavoriteOp:
		localIdentity, err := a.getLocalIdentity()
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "identity not found") {
				return nil
			}
			return err
		}
		localPubkey := strings.TrimSpace(localIdentity.PublicKey)
		if localPubkey == "" {
			return nil
		}

		record := FavoriteOpRecord{
			OpID:      strings.TrimSpace(message.FavoriteOpID),
			Pubkey:    strings.TrimSpace(message.Pubkey),
			PostID:    strings.TrimSpace(message.PostID),
			Op:        strings.TrimSpace(message.FavoriteOp),
			CreatedAt: message.Timestamp,
			Signature: strings.TrimSpace(message.Signature),
		}
		if record.Pubkey != localPubkey {
			return nil
		}

		applied, applyErr := a.applyFavoriteOperation(record, true)
		if applyErr != nil {
			return applyErr
		}
		if applied {
			a.emitFavoritesUpdated(record.PostID)
		}
		return nil
	case "COMMENT":
		if strings.TrimSpace(message.Pubkey) == "" || strings.TrimSpace(message.PostID) == "" {
			return errors.New("invalid comment payload")
		}
		if message.Timestamp == 0 {
			message.Timestamp = time.Now().Unix()
		}
		lamport, err := a.normalizeIncomingLamport(message.Lamport, message.Timestamp)
		if err != nil {
			return err
		}
		message.Lamport = lamport

		viewerPubkey := ""
		if identity, idErr := a.getLocalIdentity(); idErr == nil {
			viewerPubkey = strings.TrimSpace(identity.PublicKey)
		}
		allowed, err := a.shouldAcceptPublicContent(message.Pubkey, message.Lamport, message.Timestamp, message.ID, viewerPubkey)
		if err != nil {
			return err
		}
		if !allowed {
			return nil
		}

		commentBody := strings.TrimSpace(message.Body)
		attachments := normalizeCommentAttachments(message.CommentAttachments)
		if commentBody == "" && len(attachments) == 0 {
			return errors.New("invalid comment payload")
		}

		if strings.TrimSpace(message.DisplayName) != "" || strings.TrimSpace(message.AvatarURL) != "" {
			if _, err := a.upsertProfile(message.Pubkey, message.DisplayName, message.AvatarURL, message.Timestamp); err != nil {
				return err
			}
		}

		if strings.TrimSpace(message.ID) == "" {
			attachmentsJSON, err := encodeCommentAttachmentsJSON(attachments)
			if err != nil {
				return err
			}
			raw := fmt.Sprintf("%s|%s|%s|%s|%d", message.PostID, strings.TrimSpace(message.ParentID), commentBody, attachmentsJSON, message.Lamport)
			message.ID = buildMessageID(message.Pubkey, raw, message.Timestamp)
		}

		_, err = a.insertComment(Comment{
			ID:          message.ID,
			PostID:      strings.TrimSpace(message.PostID),
			ParentID:    strings.TrimSpace(message.ParentID),
			Pubkey:      message.Pubkey,
			Body:        commentBody,
			Attachments: attachments,
			Timestamp:   message.Timestamp,
			Lamport:     message.Lamport,
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
		if message.Timestamp == 0 {
			message.Timestamp = time.Now().Unix()
		}
		lamport, err := a.normalizeIncomingLamport(message.Lamport, message.Timestamp)
		if err != nil {
			return err
		}
		return a.upsertModeration(message.TargetPubkey, "SHADOW_BAN", message.AdminPubkey, message.Timestamp, lamport, message.Reason)
	case "UNBAN":
		trusted, err := a.isTrustedAdmin(message.AdminPubkey)
		if err != nil {
			return err
		}
		if !trusted {
			return errors.New("admin pubkey is not trusted")
		}
		if message.Timestamp == 0 {
			message.Timestamp = time.Now().Unix()
		}
		lamport, err := a.normalizeIncomingLamport(message.Lamport, message.Timestamp)
		if err != nil {
			return err
		}
		return a.upsertModeration(message.TargetPubkey, "UNBAN", message.AdminPubkey, message.Timestamp, lamport, message.Reason)
	case "POST":
		if strings.TrimSpace(message.Pubkey) == "" {
			return errors.New("invalid post payload")
		}
		if message.Timestamp == 0 {
			message.Timestamp = time.Now().Unix()
		}
		lamport, err := a.normalizeIncomingLamport(message.Lamport, message.Timestamp)
		if err != nil {
			return err
		}
		message.Lamport = lamport

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

		viewerPubkey := ""
		if identity, idErr := a.getLocalIdentity(); idErr == nil {
			viewerPubkey = strings.TrimSpace(identity.PublicKey)
		}
		allowed, err := a.shouldAcceptPublicContent(message.Pubkey, message.Lamport, message.Timestamp, message.ID, viewerPubkey)
		if err != nil {
			return err
		}
		if !allowed {
			return nil
		}

		if strings.TrimSpace(message.ID) == "" {
			seed := fmt.Sprintf("%s|%s|%d", title, body, message.Lamport)
			message.ID = buildMessageID(message.Pubkey, seed, message.Timestamp)
		}

		insertedMessage, err := a.insertMessage(ForumMessage{
			ID:          message.ID,
			Pubkey:      message.Pubkey,
			Title:       title,
			Body:        body,
			ContentCID:  strings.TrimSpace(message.ContentCID),
			ImageCID:    strings.TrimSpace(message.ImageCID),
			ThumbCID:    strings.TrimSpace(message.ThumbCID),
			ImageMIME:   strings.TrimSpace(message.ImageMIME),
			ImageSize:   message.ImageSize,
			ImageWidth:  message.ImageWidth,
			ImageHeight: message.ImageHeight,
			Content:     "",
			Score:       0,
			Timestamp:   message.Timestamp,
			Lamport:     message.Lamport,
			SizeBytes:   int64(len([]byte(body))),
			Zone:        "public",
			SubID:       normalizeSubID(message.SubID),
			Visibility:  "normal",
			IsProtected: 0,
		})
		if err != nil {
			return err
		}

		a.emitSubscribedSubUpdate(insertedMessage)
		return nil
	default:
		return fmt.Errorf("unsupported message type: %s", message.Type)
	}
}

func (a *App) AddLocalPostStructured(pubkey string, title string, body string, zone string) (ForumMessage, error) {
	return a.AddLocalPostStructuredToSub(pubkey, title, body, zone, defaultSubID)
}

func (a *App) AddLocalComment(pubkey string, postID string, parentID string, body string) (Comment, error) {
	return a.AddLocalCommentWithAttachments(pubkey, postID, parentID, body, nil)
}

func (a *App) AddLocalCommentWithAttachments(pubkey string, postID string, parentID string, body string, attachments []CommentAttachment) (Comment, error) {
	if a.db == nil {
		return Comment{}, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	parentID = strings.TrimSpace(parentID)
	body = strings.TrimSpace(body)

	attachments = normalizeCommentAttachments(attachments)
	if pubkey == "" || postID == "" {
		return Comment{}, errors.New("pubkey and post id are required")
	}
	if body == "" && len(attachments) == 0 {
		return Comment{}, errors.New("comment content is required")
	}

	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return Comment{}, err
	}
	attachmentsJSON, err := encodeCommentAttachmentsJSON(attachments)
	if err != nil {
		return Comment{}, err
	}
	raw := fmt.Sprintf("%s|%s|%s|%s|%d", postID, parentID, body, attachmentsJSON, lamport)
	comment := Comment{
		ID:          buildMessageID(pubkey, raw, now),
		PostID:      postID,
		ParentID:    parentID,
		Pubkey:      pubkey,
		Body:        body,
		Attachments: attachments,
		Score:       0,
		Timestamp:   now,
		Lamport:     lamport,
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

func (a *App) AddFavorite(postID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return errors.New("post id is required")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return errors.New("identity pubkey is empty")
	}

	exists, err := a.postExists(postID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("post not found")
	}

	active, err := a.isFavoritedByPubkey(pubkey, postID)
	if err != nil {
		return err
	}
	if active {
		return nil
	}

	record, err := a.buildLocalFavoriteOperation(identity, postID, "ADD")
	if err != nil {
		return err
	}

	applied, err := a.applyFavoriteOperation(record, true)
	if err != nil {
		return err
	}
	if !applied {
		return nil
	}

	a.emitFavoritesUpdated(postID)
	if err = a.publishFavoriteOperation(record); err != nil && a.ctx != nil {
		runtime.LogWarningf(a.ctx, "favorite publish failed op_id=%s post_id=%s err=%v", record.OpID, postID, err)
	}

	return nil
}

func (a *App) RemoveFavorite(postID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return errors.New("post id is required")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return errors.New("identity pubkey is empty")
	}

	active, err := a.isFavoritedByPubkey(pubkey, postID)
	if err != nil {
		return err
	}
	if !active {
		return nil
	}

	record, err := a.buildLocalFavoriteOperation(identity, postID, "REMOVE")
	if err != nil {
		return err
	}

	applied, err := a.applyFavoriteOperation(record, true)
	if err != nil {
		return err
	}
	if !applied {
		return nil
	}

	a.emitFavoritesUpdated(postID)
	if err = a.publishFavoriteOperation(record); err != nil && a.ctx != nil {
		runtime.LogWarningf(a.ctx, "favorite publish failed op_id=%s post_id=%s err=%v", record.OpID, postID, err)
	}

	return nil
}

func (a *App) IsFavorited(postID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return false, errors.New("post id is required")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return false, err
	}

	return a.isFavoritedByPubkey(identity.PublicKey, postID)
}

func (a *App) GetFavoritePostIDs() ([]string, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return nil, err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return nil, errors.New("identity pubkey is empty")
	}

	rows, err := a.db.Query(`
		SELECT post_id
		FROM post_favorites_state
		WHERE pubkey = ? AND state = 'active'
		ORDER BY updated_at DESC, post_id DESC;
	`, pubkey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]string, 0)
	for rows.Next() {
		var postID string
		if err = rows.Scan(&postID); err != nil {
			return nil, err
		}
		result = append(result, strings.TrimSpace(postID))
	}

	return result, rows.Err()
}

func (a *App) GetFavorites(limit int, cursor string) (PostIndexPage, error) {
	if a.db == nil {
		return PostIndexPage{}, errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return PostIndexPage{}, err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return PostIndexPage{}, errors.New("identity pubkey is empty")
	}

	limit = normalizeFavoriteLimit(limit)
	cursorTs, cursorPostID, err := decodeFavoriteCursor(cursor)
	if err != nil {
		return PostIndexPage{}, err
	}

	args := []interface{}{pubkey, pubkey, pubkey}
	query := `
		SELECT
			m.id,
			m.pubkey,
			m.title,
			SUBSTR(m.body, 1, 140) AS body_preview,
			m.content_cid,
			m.image_cid,
			m.thumb_cid,
			m.image_mime,
			m.image_size,
			m.image_width,
			m.image_height,
			m.score,
			m.timestamp,
			m.zone,
			m.sub_id,
			m.visibility,
			s.updated_at,
			s.post_id
		FROM post_favorites_state s
		INNER JOIN messages m ON m.id = s.post_id
		WHERE s.pubkey = ?
		  AND s.state = 'active'
		  AND (
			(m.zone = 'public' AND (m.visibility = 'normal' OR m.pubkey = ?))
			OR (m.zone = 'private' AND m.pubkey = ?)
		  )
	`
	if cursorTs > 0 && cursorPostID != "" {
		query += `
		  AND (s.updated_at < ? OR (s.updated_at = ? AND s.post_id < ?))
		`
		args = append(args, cursorTs, cursorTs, cursorPostID)
	}
	query += `
		ORDER BY s.updated_at DESC, s.post_id DESC
		LIMIT ?;
	`
	args = append(args, limit+1)

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return PostIndexPage{}, err
	}
	defer rows.Close()

	type favoriteRow struct {
		item      PostIndex
		updatedAt int64
		postID    string
	}

	resultRows := make([]favoriteRow, 0, limit+1)
	for rows.Next() {
		var row favoriteRow
		if err = rows.Scan(
			&row.item.ID,
			&row.item.Pubkey,
			&row.item.Title,
			&row.item.BodyPreview,
			&row.item.ContentCID,
			&row.item.ImageCID,
			&row.item.ThumbCID,
			&row.item.ImageMIME,
			&row.item.ImageSize,
			&row.item.ImageWidth,
			&row.item.ImageHeight,
			&row.item.Score,
			&row.item.Timestamp,
			&row.item.Zone,
			&row.item.SubID,
			&row.item.Visibility,
			&row.updatedAt,
			&row.postID,
		); err != nil {
			return PostIndexPage{}, err
		}
		resultRows = append(resultRows, row)
	}
	if err = rows.Err(); err != nil {
		return PostIndexPage{}, err
	}

	page := PostIndexPage{
		Items:      make([]PostIndex, 0, min(limit, len(resultRows))),
		NextCursor: "",
	}

	if len(resultRows) > limit {
		cursorRow := resultRows[limit-1]
		page.NextCursor = encodeFavoriteCursor(cursorRow.updatedAt, cursorRow.postID)
		resultRows = resultRows[:limit]
	}

	for _, row := range resultRows {
		page.Items = append(page.Items, row.item)
	}

	return page, nil
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
	lamport, err := a.nextLamport()
	if err != nil {
		return ForumMessage{}, err
	}
	messageIDSeed := fmt.Sprintf("%s|%s|%d", title, body, lamport)
	message := ForumMessage{
		ID:          buildMessageID(pubkey, messageIDSeed, now),
		Pubkey:      pubkey,
		Title:       title,
		Body:        body,
		ContentCID:  buildContentCID(body),
		Content:     "",
		Score:       0,
		Timestamp:   now,
		Lamport:     lamport,
		SizeBytes:   int64(len([]byte(body))),
		Zone:        zone,
		SubID:       normalizeSubID(subID),
		Visibility:  "normal",
		IsProtected: 0,
	}

	return a.insertMessage(message)
}

func (a *App) AddLocalPostWithImageToSub(pubkey string, title string, body string, zone string, subID string, imageBase64 string, imageMIME string) (ForumMessage, error) {
	message, err := a.AddLocalPostStructuredToSub(pubkey, title, body, zone, subID)
	if err != nil {
		return ForumMessage{}, err
	}

	imageBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(imageBase64))
	if err != nil || len(imageBytes) == 0 {
		return ForumMessage{}, errors.New("invalid image payload")
	}

	processedBytes, processedMime, width, height, thumbBytes, thumbMime, _, _, prepErr := prepareImageAssets(imageBytes, imageMIME)
	if prepErr != nil {
		return ForumMessage{}, prepErr
	}

	imageCID := buildBinaryCID(processedBytes)
	thumbCID := imageCID
	if err = a.upsertMediaBlobRaw(imageCID, processedMime, processedBytes, width, height, false); err != nil {
		return ForumMessage{}, err
	}

	if len(thumbBytes) > 0 {
		candidateThumbCID := buildBinaryCID(thumbBytes)
		thumbCID = candidateThumbCID
		if candidateThumbCID != imageCID {
			if err = a.upsertMediaBlobRaw(candidateThumbCID, thumbMime, thumbBytes, 0, 0, true); err != nil {
				return ForumMessage{}, err
			}
		}
	}

	if _, err = a.db.Exec(`
		UPDATE messages
		SET image_cid = ?, thumb_cid = ?, image_mime = ?, image_size = ?, image_width = ?, image_height = ?
		WHERE id = ?;
	`, imageCID, thumbCID, processedMime, int64(len(processedBytes)), width, height, message.ID); err != nil {
		return ForumMessage{}, err
	}

	message.ImageCID = imageCID
	message.ThumbCID = thumbCID
	message.ImageMIME = processedMime
	message.ImageSize = int64(len(processedBytes))
	message.ImageWidth = width
	message.ImageHeight = height

	return message, nil
}

func (a *App) StoreCommentImageDataURL(dataURL string) (CommentAttachment, error) {
	if a.db == nil {
		return CommentAttachment{}, errors.New("database not initialized")
	}

	dataURL = strings.TrimSpace(dataURL)
	if dataURL == "" {
		return CommentAttachment{}, errors.New("image payload is required")
	}
	if !strings.HasPrefix(dataURL, "data:") {
		return CommentAttachment{}, errors.New("invalid data URL")
	}
	commaIndex := strings.Index(dataURL, ",")
	if commaIndex <= 0 {
		return CommentAttachment{}, errors.New("invalid data URL")
	}

	header := dataURL[5:commaIndex]
	body := dataURL[commaIndex+1:]
	if !strings.Contains(strings.ToLower(header), ";base64") {
		return CommentAttachment{}, errors.New("comment image must be base64 encoded")
	}
	hintMIME := strings.TrimSpace(strings.SplitN(header, ";", 2)[0])
	raw, err := base64.StdEncoding.DecodeString(body)
	if err != nil || len(raw) == 0 {
		return CommentAttachment{}, errors.New("invalid image payload")
	}

	processedBytes, processedMime, width, height, _, _, _, _, prepErr := prepareImageAssets(raw, hintMIME)
	if prepErr != nil {
		return CommentAttachment{}, prepErr
	}

	contentCID := buildBinaryCID(processedBytes)
	if err = a.upsertMediaBlobRaw(contentCID, processedMime, processedBytes, width, height, false); err != nil {
		return CommentAttachment{}, err
	}

	return CommentAttachment{
		Kind:      "media_cid",
		Ref:       contentCID,
		Mime:      processedMime,
		Width:     width,
		Height:    height,
		SizeBytes: int64(len(processedBytes)),
	}, nil
}

func prepareImageAssets(source []byte, hintMIME string) (mainBytes []byte, mainMIME string, width int, height int, thumbBytes []byte, thumbMIME string, thumbWidth int, thumbHeight int, err error) {
	hintMIME = strings.TrimSpace(strings.ToLower(hintMIME))
	decoded, format, decodeErr := image.Decode(bytes.NewReader(source))
	if decodeErr != nil {
		fallbackMIME := hintMIME
		if fallbackMIME == "" {
			fallbackMIME = "application/octet-stream"
		}
		return source, fallbackMIME, 0, 0, nil, "", 0, 0, nil
	}

	bounds := decoded.Bounds()
	width = bounds.Dx()
	height = bounds.Dy()

	mainMIME = normalizedImageMIME(hintMIME, format)

	compressedImage := resizeImageIfNeeded(decoded, 1920)
	compressedBytes, compressedMIME, encodeErr := encodeImageForStorage(compressedImage, mainMIME)
	if encodeErr != nil {
		return nil, "", 0, 0, nil, "", 0, 0, encodeErr
	}

	thumbImage := resizeImageIfNeeded(decoded, 320)
	thumbWidth = thumbImage.Bounds().Dx()
	thumbHeight = thumbImage.Bounds().Dy()
	thumbBytes, thumbMIME, err = encodeImageForStorage(thumbImage, "image/jpeg")
	if err != nil {
		return nil, "", 0, 0, nil, "", 0, 0, err
	}

	return compressedBytes, compressedMIME, width, height, thumbBytes, thumbMIME, thumbWidth, thumbHeight, nil
}

func normalizedImageMIME(hint string, format string) string {
	hint = strings.TrimSpace(strings.ToLower(hint))
	if strings.HasPrefix(hint, "image/") {
		return hint
	}

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

func encodeImageForStorage(img image.Image, preferredMIME string) ([]byte, string, error) {
	preferredMIME = strings.TrimSpace(strings.ToLower(preferredMIME))
	if preferredMIME == "image/png" && !hasTransparency(img) {
		preferredMIME = "image/jpeg"
	}

	var buffer bytes.Buffer
	switch preferredMIME {
	case "image/png":
		encoder := png.Encoder{CompressionLevel: png.BestSpeed}
		if err := encoder.Encode(&buffer, img); err != nil {
			return nil, "", err
		}
		return buffer.Bytes(), "image/png", nil
	default:
		if err := jpeg.Encode(&buffer, img, &jpeg.Options{Quality: 82}); err != nil {
			return nil, "", err
		}
		return buffer.Bytes(), "image/jpeg", nil
	}
}

func resizeImageIfNeeded(src image.Image, maxEdge int) image.Image {
	if maxEdge <= 0 {
		return src
	}

	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= maxEdge && height <= maxEdge {
		return src
	}

	scale := float64(maxEdge) / float64(width)
	if height > width {
		scale = float64(maxEdge) / float64(height)
	}
	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)
	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	destination := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	xdraw.ApproxBiLinear.Scale(destination, destination.Bounds(), src, bounds, xdraw.Over, nil)
	return destination
}

func hasTransparency(img image.Image) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, alpha := img.At(x, y).RGBA()
			if alpha < 0xffff {
				return true
			}
		}
	}
	return false
}

func (a *App) ApplyShadowBan(targetPubkey string, adminPubkey string, reason string) error {
	trusted, err := a.isTrustedAdmin(adminPubkey)
	if err != nil {
		return err
	}
	if !trusted {
		return errors.New("admin pubkey is not trusted")
	}

	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return err
	}
	return a.upsertModeration(targetPubkey, "SHADOW_BAN", adminPubkey, now, lamport, reason)
}

func (a *App) ApplyUnban(targetPubkey string, adminPubkey string, reason string) error {
	trusted, err := a.isTrustedAdmin(adminPubkey)
	if err != nil {
		return err
	}
	if !trusted {
		return errors.New("admin pubkey is not trusted")
	}

	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return err
	}
	return a.upsertModeration(targetPubkey, "UNBAN", adminPubkey, now, lamport, reason)
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
	if message.Timestamp <= 0 {
		message.Timestamp = time.Now().Unix()
	}
	if message.Lamport <= 0 {
		message.Lamport = message.Timestamp
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
		`INSERT OR REPLACE INTO messages (id, pubkey, title, body, content_cid, image_cid, thumb_cid, image_mime, image_size, image_width, image_height, content, score, timestamp, lamport, size_bytes, zone, sub_id, is_protected, visibility)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		message.ID,
		message.Pubkey,
		message.Title,
		message.Body,
		message.ContentCID,
		message.ImageCID,
		message.ThumbCID,
		message.ImageMIME,
		message.ImageSize,
		message.ImageWidth,
		message.ImageHeight,
		message.Content,
		message.Score,
		message.Timestamp,
		message.Lamport,
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

	if comment.ID == "" || comment.PostID == "" || comment.Pubkey == "" {
		return Comment{}, errors.New("invalid comment")
	}
	if comment.Body == "" && len(comment.Attachments) == 0 {
		return Comment{}, errors.New("invalid comment")
	}
	if comment.Timestamp == 0 {
		comment.Timestamp = time.Now().Unix()
	}
	if comment.Lamport <= 0 {
		comment.Lamport = comment.Timestamp
	}
	comment.Attachments = normalizeCommentAttachments(comment.Attachments)
	attachmentsJSON, err := encodeCommentAttachmentsJSON(comment.Attachments)
	if err != nil {
		return Comment{}, err
	}
	mediaCIDs := mediaCIDsFromAttachments(comment.Attachments)

	tx, err := a.db.Begin()
	if err != nil {
		return Comment{}, err
	}

	_, err = tx.Exec(`
		INSERT OR REPLACE INTO comments (id, post_id, parent_id, pubkey, body, attachments_json, score, timestamp, lamport)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
	`, comment.ID, comment.PostID, comment.ParentID, comment.Pubkey, comment.Body, attachmentsJSON, comment.Score, comment.Timestamp, comment.Lamport)
	if err != nil {
		_ = tx.Rollback()
		return Comment{}, err
	}

	if _, err = tx.Exec(`DELETE FROM comment_media_refs WHERE comment_id = ?;`, comment.ID); err != nil {
		_ = tx.Rollback()
		return Comment{}, err
	}
	for _, cid := range mediaCIDs {
		if _, err = tx.Exec(`
			INSERT INTO comment_media_refs (comment_id, content_cid)
			VALUES (?, ?)
			ON CONFLICT(comment_id, content_cid) DO NOTHING;
		`, comment.ID, cid); err != nil {
			_ = tx.Rollback()
			return Comment{}, err
		}
	}

	if err = tx.Commit(); err != nil {
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

func (a *App) postExists(postID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return false, nil
	}

	var exists int
	err := a.db.QueryRow(`SELECT 1 FROM messages WHERE id = ? LIMIT 1;`, postID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (a *App) isFavoritedByPubkey(pubkey string, postID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	if pubkey == "" || postID == "" {
		return false, nil
	}

	var state string
	err := a.db.QueryRow(`
		SELECT state
		FROM post_favorites_state
		WHERE pubkey = ? AND post_id = ?;
	`, pubkey, postID).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return strings.EqualFold(strings.TrimSpace(state), "active"), nil
}

func (a *App) buildLocalFavoriteOperation(identity Identity, postID string, op string) (FavoriteOpRecord, error) {
	pubkey := strings.TrimSpace(identity.PublicKey)
	mnemonic := strings.TrimSpace(identity.Mnemonic)
	postID = strings.TrimSpace(postID)
	normalizedOp, err := normalizeFavoriteOperation(op)
	if err != nil {
		return FavoriteOpRecord{}, err
	}
	if pubkey == "" || mnemonic == "" || postID == "" {
		return FavoriteOpRecord{}, errors.New("invalid favorite operation identity")
	}

	now := time.Now()
	createdAt := now.Unix()
	opID := buildMessageID(pubkey, fmt.Sprintf("favorite|%s|%s|%d", normalizedOp, postID, now.UnixNano()), createdAt)
	signaturePayload := buildFavoriteSignaturePayload(pubkey, postID, normalizedOp, createdAt, opID)
	signature, err := a.SignMessage(mnemonic, signaturePayload)
	if err != nil {
		return FavoriteOpRecord{}, err
	}

	return FavoriteOpRecord{
		OpID:      opID,
		Pubkey:    pubkey,
		PostID:    postID,
		Op:        normalizedOp,
		CreatedAt: createdAt,
		Signature: signature,
	}, nil
}

func (a *App) applyFavoriteOperation(record FavoriteOpRecord, verifySignature bool) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	record.OpID = strings.TrimSpace(record.OpID)
	record.Pubkey = strings.TrimSpace(record.Pubkey)
	record.PostID = strings.TrimSpace(record.PostID)
	record.Signature = strings.TrimSpace(record.Signature)

	normalizedOp, err := normalizeFavoriteOperation(record.Op)
	if err != nil {
		return false, err
	}
	record.Op = normalizedOp

	if record.Pubkey == "" || record.PostID == "" {
		return false, errors.New("invalid favorite operation payload")
	}
	if record.CreatedAt <= 0 {
		record.CreatedAt = time.Now().Unix()
	}
	if record.OpID == "" {
		record.OpID = buildMessageID(record.Pubkey, fmt.Sprintf("favorite|%s|%s", record.Op, record.PostID), record.CreatedAt)
	}
	if verifySignature {
		if record.Signature == "" {
			return false, errors.New("favorite operation signature is required")
		}
		valid, verifyErr := a.verifyFavoriteOperationSignature(record)
		if verifyErr != nil {
			return false, verifyErr
		}
		if !valid {
			return false, errors.New("invalid favorite operation signature")
		}
	}

	nextState := favoriteStateForOperation(record.Op)
	tx, err := a.db.Begin()
	if err != nil {
		return false, err
	}

	result, err := tx.Exec(`
		INSERT INTO post_favorite_ops (op_id, pubkey, post_id, op, created_at, signature)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(op_id) DO NOTHING;
	`, record.OpID, record.Pubkey, record.PostID, record.Op, record.CreatedAt, record.Signature)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}

	insertedCount, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}

	if insertedCount == 0 {
		if err = tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}

	var existingUpdatedAt int64
	var existingLastOpID string
	err = tx.QueryRow(`
		SELECT updated_at, last_op_id
		FROM post_favorites_state
		WHERE pubkey = ? AND post_id = ?;
	`, record.Pubkey, record.PostID).Scan(&existingUpdatedAt, &existingLastOpID)

	shouldApply := true
	if err == nil {
		shouldApply = favoriteOperationWins(existingUpdatedAt, strings.TrimSpace(existingLastOpID), record.CreatedAt, record.OpID)
	} else if !errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		return false, err
	}

	if shouldApply {
		if _, err = tx.Exec(`
			INSERT INTO post_favorites_state (pubkey, post_id, state, updated_at, last_op_id)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(pubkey, post_id) DO UPDATE SET
				state = excluded.state,
				updated_at = excluded.updated_at,
				last_op_id = excluded.last_op_id;
		`, record.Pubkey, record.PostID, nextState, record.CreatedAt, record.OpID); err != nil {
			_ = tx.Rollback()
			return false, err
		}
	}

	if err = tx.Commit(); err != nil {
		return false, err
	}

	return shouldApply, nil
}

func (a *App) verifyFavoriteOperationSignature(record FavoriteOpRecord) (bool, error) {
	pubkey := strings.TrimSpace(record.Pubkey)
	postID := strings.TrimSpace(record.PostID)
	opID := strings.TrimSpace(record.OpID)
	signature := strings.TrimSpace(record.Signature)
	if pubkey == "" || postID == "" || opID == "" || signature == "" {
		return false, nil
	}

	normalizedOp, err := normalizeFavoriteOperation(record.Op)
	if err != nil {
		return false, err
	}
	payload := buildFavoriteSignaturePayload(pubkey, postID, normalizedOp, record.CreatedAt, opID)
	return a.VerifyMessage(pubkey, payload, signature)
}

func (a *App) emitFavoritesUpdated(postID string) {
	if a.ctx == nil {
		return
	}

	payload := map[string]string{
		"postId": strings.TrimSpace(postID),
	}
	runtime.EventsEmit(a.ctx, "favorites:updated", payload)
	runtime.EventsEmit(a.ctx, "feed:updated")
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

func (a *App) upsertModeration(targetPubkey string, action string, sourceAdmin string, timestamp int64, lamport int64, reason string) error {
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
	if lamport <= 0 {
		lamport = timestamp
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
			INSERT INTO moderation_logs (target_pubkey, action, source_admin, timestamp, lamport, reason, result)
			VALUES (?, ?, ?, ?, ?, ?, 'ignored_older');
		`, targetPubkey, action, sourceAdmin, timestamp, lamport, reason); logErr != nil {
			return logErr
		}
		return nil
	}

	_, err = a.db.Exec(`
		INSERT INTO moderation (target_pubkey, action, source_admin, timestamp, lamport, reason)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(target_pubkey) DO UPDATE SET
			action = excluded.action,
			source_admin = excluded.source_admin,
			timestamp = excluded.timestamp,
			lamport = excluded.lamport,
			reason = excluded.reason;
	`, targetPubkey, action, sourceAdmin, timestamp, lamport, reason)
	if err != nil {
		return err
	}

	if _, err = a.db.Exec(`
		INSERT INTO moderation_logs (target_pubkey, action, source_admin, timestamp, lamport, reason, result)
		VALUES (?, ?, ?, ?, ?, ?, 'applied');
	`, targetPubkey, action, sourceAdmin, timestamp, lamport, reason); err != nil {
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

func (a *App) getModerationSnapshot(pubkey string) (string, int64, int64, string, error) {
	if a.db == nil {
		return "", 0, 0, "", errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return "", 0, 0, "", nil
	}

	var action string
	var timestamp int64
	var lamport int64
	var sourceAdmin string
	err := a.db.QueryRow(`
		SELECT action, timestamp, lamport, source_admin
		FROM moderation
		WHERE target_pubkey = ?;
	`, pubkey).Scan(&action, &timestamp, &lamport, &sourceAdmin)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, 0, "", nil
	}
	if err != nil {
		return "", 0, 0, "", err
	}

	return strings.ToUpper(strings.TrimSpace(action)), timestamp, lamport, strings.TrimSpace(sourceAdmin), nil
}

func (a *App) shouldAcceptPublicContent(authorPubkey string, contentLamport int64, contentTimestamp int64, contentID string, viewerPubkey string) (bool, error) {
	authorPubkey = strings.TrimSpace(authorPubkey)
	contentID = strings.TrimSpace(contentID)
	viewerPubkey = strings.TrimSpace(viewerPubkey)
	if authorPubkey == "" {
		return false, errors.New("author pubkey is required")
	}

	if viewerPubkey != "" && authorPubkey == viewerPubkey {
		return true, nil
	}

	action, moderationTimestamp, moderationLamport, moderationAdmin, err := a.getModerationSnapshot(authorPubkey)
	if err != nil {
		return false, err
	}
	if action != "SHADOW_BAN" {
		return true, nil
	}

	policy, err := a.GetGovernancePolicy()
	if err != nil {
		return false, err
	}
	if policy.HideHistoryOnShadowBan {
		return false, nil
	}

	if contentTimestamp <= 0 {
		contentTimestamp = time.Now().Unix()
	}
	if contentLamport <= 0 {
		contentLamport = contentTimestamp
	}

	if moderationLamport > 0 && contentLamport > 0 {
		if contentLamport < moderationLamport {
			return true, nil
		}
		if contentLamport > moderationLamport {
			return false, nil
		}

		if contentTimestamp < moderationTimestamp {
			return true, nil
		}
		if contentTimestamp > moderationTimestamp {
			return false, nil
		}

		if contentID == "" {
			return false, nil
		}
		moderationKey := fmt.Sprintf("%s|%d|%s", moderationAdmin, moderationTimestamp, action)
		return strings.Compare(contentID, moderationKey) < 0, nil
	}

	return contentTimestamp < moderationTimestamp, nil
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

func buildBinaryCID(data []byte) string {
	hash := sha256.Sum256(data)
	return "cidv1-bin-" + hex.EncodeToString(hash[:])
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

func normalizeCommentAttachments(input []CommentAttachment) []CommentAttachment {
	if len(input) == 0 {
		return []CommentAttachment{}
	}

	result := make([]CommentAttachment, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, item := range input {
		kind := strings.ToLower(strings.TrimSpace(item.Kind))
		ref := strings.TrimSpace(item.Ref)
		if ref == "" {
			continue
		}
		if kind != "media_cid" && kind != "external_url" {
			continue
		}
		if kind == "external_url" {
			u, err := url.Parse(ref)
			if err != nil {
				continue
			}
			proto := strings.ToLower(strings.TrimSpace(u.Scheme))
			if proto != "http" && proto != "https" {
				continue
			}
			ref = u.String()
		}

		key := kind + "|" + ref
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, CommentAttachment{
			Kind:      kind,
			Ref:       ref,
			Mime:      strings.TrimSpace(item.Mime),
			Width:     item.Width,
			Height:    item.Height,
			SizeBytes: item.SizeBytes,
		})
	}

	if len(result) > 8 {
		result = result[:8]
	}

	return result
}

func encodeCommentAttachmentsJSON(items []CommentAttachment) (string, error) {
	normalized := normalizeCommentAttachments(items)
	if len(normalized) == 0 {
		return "[]", nil
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func decodeCommentAttachmentsJSON(raw string) []CommentAttachment {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []CommentAttachment{}
	}

	var items []CommentAttachment
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []CommentAttachment{}
	}
	return normalizeCommentAttachments(items)
}

func mediaCIDsFromAttachments(items []CommentAttachment) []string {
	if len(items) == 0 {
		return []string{}
	}

	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Kind)) != "media_cid" {
			continue
		}
		cid := strings.TrimSpace(item.Ref)
		if cid == "" {
			continue
		}
		if _, exists := seen[cid]; exists {
			continue
		}
		seen[cid] = struct{}{}
		result = append(result, cid)
	}
	return result
}

func normalizeSearchLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func normalizeFavoriteLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func normalizeMyPostsLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func normalizeFavoriteOperation(op string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(op))
	if normalized == "ADD" || normalized == "REMOVE" {
		return normalized, nil
	}

	return "", errors.New("invalid favorite operation")
}

func favoriteStateForOperation(op string) string {
	if strings.ToUpper(strings.TrimSpace(op)) == "ADD" {
		return "active"
	}

	return "removed"
}

func favoriteOperationWins(existingUpdatedAt int64, existingOpID string, incomingUpdatedAt int64, incomingOpID string) bool {
	if incomingUpdatedAt > existingUpdatedAt {
		return true
	}
	if incomingUpdatedAt < existingUpdatedAt {
		return false
	}

	return strings.Compare(strings.TrimSpace(incomingOpID), strings.TrimSpace(existingOpID)) > 0
}

func buildFavoriteSignaturePayload(pubkey string, postID string, op string, createdAt int64, opID string) string {
	return fmt.Sprintf(
		"favorite|%s|%s|%s|%d|%s",
		strings.TrimSpace(pubkey),
		strings.TrimSpace(postID),
		strings.ToUpper(strings.TrimSpace(op)),
		createdAt,
		strings.TrimSpace(opID),
	)
}

func encodeFavoriteCursor(updatedAt int64, postID string) string {
	raw := fmt.Sprintf("%d|%s", updatedAt, strings.TrimSpace(postID))
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func decodeFavoriteCursor(cursor string) (int64, string, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, "", nil
	}

	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, "", errors.New("invalid favorite cursor")
	}

	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return 0, "", errors.New("invalid favorite cursor")
	}

	updatedAt, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || updatedAt <= 0 {
		return 0, "", errors.New("invalid favorite cursor")
	}
	postID := strings.TrimSpace(parts[1])
	if postID == "" {
		return 0, "", errors.New("invalid favorite cursor")
	}

	return updatedAt, postID, nil
}

func encodeMyPostsCursor(timestamp int64, postID string) string {
	raw := fmt.Sprintf("%d|%s", timestamp, strings.TrimSpace(postID))
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func decodeMyPostsCursor(cursor string) (int64, string, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, "", nil
	}

	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, "", errors.New("invalid my posts cursor")
	}

	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return 0, "", errors.New("invalid my posts cursor")
	}

	timestamp, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || timestamp <= 0 {
		return 0, "", errors.New("invalid my posts cursor")
	}
	postID := strings.TrimSpace(parts[1])
	if postID == "" {
		return 0, "", errors.New("invalid my posts cursor")
	}

	return timestamp, postID, nil
}

func normalizeFeedStreamLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func normalizeFeedStreamAlgorithm(algorithm string) string {
	normalized := strings.TrimSpace(strings.ToLower(algorithm))
	if normalized == "" {
		return "hot-v1"
	}
	return normalized
}

func scoreFeedRecommendation(message ForumMessage, now int64, algorithm string) float64 {
	switch strings.TrimSpace(strings.ToLower(algorithm)) {
	case "hot-v1":
		return computeHotScore(message.Score, message.Timestamp, now)
	default:
		return computeHotScore(message.Score, message.Timestamp, now)
	}
}

func countFeedItemsByReason(items []FeedStreamItem, reason string) int {
	total := 0
	for _, item := range items {
		if item.Reason == reason {
			total++
		}
	}
	return total
}

func (a *App) listSubscribedSubIDs() ([]string, error) {
	rows, err := a.db.Query(`SELECT sub_id FROM sub_subscriptions ORDER BY subscribed_at DESC;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]string, 0)
	for rows.Next() {
		var subID string
		if err := rows.Scan(&subID); err != nil {
			return nil, err
		}
		result = append(result, normalizeSubID(subID))
	}

	return result, rows.Err()
}

func (a *App) queryPostsBySubSet(viewerPubkey string, subIDs []string, limit int) ([]ForumMessage, error) {
	if len(subIDs) == 0 {
		return []ForumMessage{}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	placeholders := makeSQLPlaceholders(len(subIDs))
	args := make([]interface{}, 0, len(subIDs)+2)
	args = append(args, viewerPubkey)
	for _, subID := range subIDs {
		args = append(args, normalizeSubID(subID))
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
		FROM messages
		WHERE zone = 'public'
		  AND (visibility = 'normal' OR pubkey = ?)
		  AND sub_id IN (%s)
		ORDER BY timestamp DESC
		LIMIT ?;
	`, placeholders)

	return a.queryForumMessages(query, args...)
}

func (a *App) queryRecommendedPosts(viewerPubkey string, subscribedSubIDs []string, limit int) ([]ForumMessage, error) {
	if limit <= 0 {
		limit = 40
	}

	if len(subscribedSubIDs) == 0 {
		return a.queryForumMessages(`
			SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
			FROM messages
			WHERE zone = 'public'
			  AND (visibility = 'normal' OR pubkey = ?)
			ORDER BY score DESC, timestamp DESC
			LIMIT ?;
		`, viewerPubkey, limit)
	}

	placeholders := makeSQLPlaceholders(len(subscribedSubIDs))
	args := make([]interface{}, 0, len(subscribedSubIDs)+2)
	args = append(args, viewerPubkey)
	for _, subID := range subscribedSubIDs {
		args = append(args, normalizeSubID(subID))
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
		FROM messages
		WHERE zone = 'public'
		  AND (visibility = 'normal' OR pubkey = ?)
		  AND sub_id NOT IN (%s)
		ORDER BY score DESC, timestamp DESC
		LIMIT ?;
	`, placeholders)

	return a.queryForumMessages(query, args...)
}

func (a *App) queryForumMessages(query string, args ...interface{}) ([]ForumMessage, error) {
	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ForumMessage, 0)
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
		result = append(result, message)
	}

	return result, rows.Err()
}

func makeSQLPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}

	parts := make([]string, count)
	for i := range count {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
}

func (a *App) isSubSubscribed(subID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	subID = normalizeSubID(subID)

	var count int
	if err := a.db.QueryRow(`
		SELECT COUNT(1)
		FROM sub_subscriptions
		WHERE sub_id = ?;
	`, subID).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func (a *App) emitSubscribedSubUpdate(message ForumMessage) {
	if a.ctx == nil {
		return
	}
	if strings.TrimSpace(message.Zone) != "public" {
		return
	}

	subscribed, err := a.isSubSubscribed(message.SubID)
	if err != nil || !subscribed {
		return
	}

	runtime.EventsEmit(a.ctx, "sub:updated", map[string]interface{}{
		"subId":     message.SubID,
		"postId":    message.ID,
		"title":     message.Title,
		"timestamp": message.Timestamp,
		"pubkey":    message.Pubkey,
	})
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
