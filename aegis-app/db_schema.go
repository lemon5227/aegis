// db_schema.go — 数据库初始化和全部 Schema DDL（CREATE TABLE、ALTER TABLE 迁移、数据回填）。
package main

import (
	"database/sql"
	"strings"
	"time"
)

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

	// Use connection-string pragmas so settings apply to ALL pooled connections,
	// not just the first one. This fixes intermittent SQLITE_BUSY errors under
	// concurrent test load and high-contention production usage.
	connURL := databasePath + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)"

	db, err := sql.Open("sqlite", connURL)
	if err != nil {
		return err
	}

	// Limit connection pool size for SQLite to reduce write contention.
	// SQLite WAL mode allows concurrent reads but only one writer at a time.
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)

	// Verify pragmas applied (sanity check).
	if _, err = db.Exec("PRAGMA busy_timeout = 10000;"); err != nil {
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
			current_author_pubkey TEXT NOT NULL DEFAULT '',
			current_op_id TEXT NOT NULL DEFAULT '',
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
			visibility TEXT NOT NULL DEFAULT 'normal',
			deleted_at_lamport INTEGER NOT NULL DEFAULT 0,
			deleted_at_ts INTEGER NOT NULL DEFAULT 0,
			deleted_by TEXT NOT NULL DEFAULT ''
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
		`CREATE TABLE IF NOT EXISTS sub_settings (
			sub_id TEXT PRIMARY KEY,
			rules_json TEXT NOT NULL DEFAULT '[]',
			announcement TEXT NOT NULL DEFAULT '',
			updated_at INTEGER NOT NULL DEFAULT 0,
			current_admin_pubkey TEXT NOT NULL DEFAULT '',
			current_op_id TEXT NOT NULL DEFAULT '',
			lamport INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY(sub_id) REFERENCES subs(id) ON DELETE CASCADE
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
			current_author_pubkey TEXT NOT NULL DEFAULT '',
			current_op_id TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL,
			attachments_json TEXT NOT NULL DEFAULT '[]',
			score INTEGER NOT NULL DEFAULT 0,
			timestamp INTEGER NOT NULL,
			lamport INTEGER NOT NULL DEFAULT 0,
			deleted_at_lamport INTEGER NOT NULL DEFAULT 0,
			deleted_at INTEGER NOT NULL DEFAULT 0,
			deleted_by TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE INDEX IF NOT EXISTS idx_comments_post_timestamp ON comments(post_id, timestamp);`,
		`CREATE TABLE IF NOT EXISTS entity_ops (
			op_id TEXT PRIMARY KEY,
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			op_type TEXT NOT NULL,
			author_pubkey TEXT NOT NULL,
			lamport INTEGER NOT NULL,
			timestamp INTEGER NOT NULL,
			schema_version INTEGER NOT NULL DEFAULT 1,
			auth_scope TEXT NOT NULL DEFAULT 'user',
			payload_json TEXT NOT NULL DEFAULT '{}'
		);`,
		`CREATE INDEX IF NOT EXISTS idx_entity_ops_entity_lamport ON entity_ops(entity_type, entity_id, lamport DESC, op_id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_entity_ops_timestamp ON entity_ops(timestamp DESC);`,
		`CREATE TABLE IF NOT EXISTS tombstone_gc_marks (
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			deleted_at_lamport INTEGER NOT NULL,
			stable_passes INTEGER NOT NULL DEFAULT 0,
			first_seen_at INTEGER NOT NULL,
			last_seen_at INTEGER NOT NULL,
			PRIMARY KEY (entity_type, entity_id)
		);`,
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
		`CREATE TABLE IF NOT EXISTS post_downvotes (
			post_id TEXT NOT NULL,
			voter_pubkey TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			PRIMARY KEY (post_id, voter_pubkey)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_post_downvotes_post ON post_downvotes(post_id);`,
		`CREATE TABLE IF NOT EXISTS comment_downvotes (
			comment_id TEXT NOT NULL,
			voter_pubkey TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			PRIMARY KEY (comment_id, voter_pubkey)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_comment_downvotes_comment ON comment_downvotes(comment_id);`,
		`CREATE TABLE IF NOT EXISTS vote_ops (
			op_id TEXT PRIMARY KEY,
			created_at INTEGER NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS post_admin_state (
			post_id TEXT PRIMARY KEY,
			pinned INTEGER NOT NULL DEFAULT 0,
			pinned_by TEXT NOT NULL DEFAULT '',
			pinned_updated_at INTEGER NOT NULL DEFAULT 0,
			pinned_lamport INTEGER NOT NULL DEFAULT 0,
			pinned_op_id TEXT NOT NULL DEFAULT '',
			locked INTEGER NOT NULL DEFAULT 0,
			locked_by TEXT NOT NULL DEFAULT '',
			locked_updated_at INTEGER NOT NULL DEFAULT 0,
			locked_lamport INTEGER NOT NULL DEFAULT 0,
			locked_op_id TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE INDEX IF NOT EXISTS idx_post_admin_state_pinned ON post_admin_state(pinned, pinned_updated_at DESC);`,
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
		`CREATE TABLE IF NOT EXISTS message_outbox (
			id TEXT PRIMARY KEY,
			message_type TEXT NOT NULL,
			payload_json TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			available_at INTEGER NOT NULL,
			attempt_count INTEGER NOT NULL DEFAULT 0,
			last_error TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			source_pubkey TEXT NOT NULL,
			target_entity_id TEXT NOT NULL,
			target_type TEXT NOT NULL DEFAULT 'post',
			post_id TEXT NOT NULL DEFAULT '',
			is_read INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			UNIQUE(type, source_pubkey, target_entity_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);`,
	}

	for _, statement := range schema {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}

	// --- notifications table migrations (for DBs created before notification center) ---
	if _, err := db.Exec(`ALTER TABLE notifications ADD COLUMN target_type TEXT NOT NULL DEFAULT 'post';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE notifications ADD COLUMN post_id TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE notifications ADD COLUMN is_read INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications(is_read, created_at DESC);`); err != nil {
		return err
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

	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN current_author_pubkey TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN current_op_id TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN deleted_at_lamport INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN deleted_at_ts INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE messages ADD COLUMN deleted_by TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE sub_settings ADD COLUMN rules_json TEXT NOT NULL DEFAULT '[]';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE sub_settings ADD COLUMN announcement TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE sub_settings ADD COLUMN updated_at INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE sub_settings ADD COLUMN current_admin_pubkey TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE sub_settings ADD COLUMN current_op_id TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE sub_settings ADD COLUMN lamport INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE post_admin_state ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE post_admin_state ADD COLUMN pinned_by TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE post_admin_state ADD COLUMN pinned_updated_at INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE post_admin_state ADD COLUMN pinned_lamport INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE post_admin_state ADD COLUMN pinned_op_id TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE post_admin_state ADD COLUMN locked INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE post_admin_state ADD COLUMN locked_by TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE post_admin_state ADD COLUMN locked_updated_at INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE post_admin_state ADD COLUMN locked_lamport INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE post_admin_state ADD COLUMN locked_op_id TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_post_admin_state_pinned ON post_admin_state(pinned, pinned_updated_at DESC);`); err != nil {
		return err
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
	if _, err := db.Exec(`ALTER TABLE comments ADD COLUMN current_author_pubkey TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE comments ADD COLUMN current_op_id TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE comments ADD COLUMN deleted_at_lamport INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE comments ADD COLUMN deleted_at INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE comments ADD COLUMN deleted_by TEXT NOT NULL DEFAULT '';`); err != nil {
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

	if _, err := db.Exec(`ALTER TABLE message_outbox ADD COLUMN available_at INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE message_outbox ADD COLUMN attempt_count INTEGER NOT NULL DEFAULT 0;`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE message_outbox ADD COLUMN last_error TEXT NOT NULL DEFAULT '';`); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return err
		}
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_message_outbox_available_at ON message_outbox(available_at, created_at);`); err != nil {
		return err
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
	if _, err := db.Exec(`UPDATE messages SET current_author_pubkey = pubkey WHERE COALESCE(TRIM(current_author_pubkey), '') = '';`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE messages SET current_op_id = id WHERE COALESCE(TRIM(current_op_id), '') = '';`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE messages SET deleted_at_lamport = lamport, deleted_at_ts = timestamp, deleted_by = pubkey WHERE visibility = 'deleted' AND deleted_at_lamport = 0;`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE comments SET lamport = timestamp WHERE lamport = 0;`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE comments SET current_author_pubkey = pubkey WHERE COALESCE(TRIM(current_author_pubkey), '') = '';`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE comments SET current_op_id = id WHERE COALESCE(TRIM(current_op_id), '') = '';`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE comments SET deleted_at_lamport = lamport WHERE deleted_at > 0 AND deleted_at_lamport = 0;`); err != nil {
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

	// --- Personal mute users (client-side only, not synced via P2P) ---
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS muted_users (
			pubkey TEXT PRIMARY KEY,
			reason TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_muted_users_created_at ON muted_users(created_at DESC);`); err != nil {
		return err
	}

	// --- Post read tracking (client-side only, not synced via P2P) ---
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS post_reads (
			post_id TEXT PRIMARY KEY,
			read_at INTEGER NOT NULL
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_post_reads_read_at ON post_reads(read_at DESC);`); err != nil {
		return err
	}

	// --- User preferences (client-side key-value store) ---
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_preferences (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at INTEGER NOT NULL
		);
	`); err != nil {
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
