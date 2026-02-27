import re

file_path = 'aegis-app/db.go'

with open(file_path, 'r') as f:
    content = f.read()

# Add FTS tables and triggers to schema
fts_schema = """
		`CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
			title,
			body,
			content,
			content_rowid UNINDEXED
		);`,
		`CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
			INSERT INTO messages_fts(rowid, title, body, content, content_rowid)
			VALUES (new.rowid, new.title, new.body, new.content, new.rowid);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
			DELETE FROM messages_fts WHERE content_rowid = old.rowid;
		END;`,
		`CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
			INSERT INTO messages_fts(messages_fts, rowid, title, body, content, content_rowid)
			VALUES('delete', old.rowid, old.title, old.body, old.content, old.rowid);
			INSERT INTO messages_fts(rowid, title, body, content, content_rowid)
			VALUES (new.rowid, new.title, new.body, new.content, new.rowid);
		END;`,

		`CREATE VIRTUAL TABLE IF NOT EXISTS comments_fts USING fts5(
			body,
			content_rowid UNINDEXED
		);`,
		`CREATE TRIGGER IF NOT EXISTS comments_ai AFTER INSERT ON comments BEGIN
			INSERT INTO comments_fts(rowid, body, content_rowid)
			VALUES (new.rowid, new.body, new.rowid);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS comments_ad AFTER DELETE ON comments BEGIN
			DELETE FROM comments_fts WHERE content_rowid = old.rowid;
		END;`,
		`CREATE TRIGGER IF NOT EXISTS comments_au AFTER UPDATE ON comments BEGIN
			INSERT INTO comments_fts(comments_fts, rowid, body, content_rowid)
			VALUES('delete', old.rowid, old.body, old.rowid);
			INSERT INTO comments_fts(rowid, body, content_rowid)
			VALUES (new.rowid, new.body, new.rowid);
		END;`,

        `CREATE TABLE IF NOT EXISTS hashtags (
            tag TEXT NOT NULL,
            target_id TEXT NOT NULL,
            target_type TEXT NOT NULL,
            timestamp INTEGER NOT NULL,
            PRIMARY KEY (tag, target_id)
        );`,
        `CREATE INDEX IF NOT EXISTS idx_hashtags_tag_timestamp ON hashtags(tag, timestamp DESC);`,
"""

# Insert into ensureSchema
# Hook into `CREATE TABLE IF NOT EXISTS logical_clock`
if '`CREATE TABLE IF NOT EXISTS logical_clock (' in content:
    content = content.replace('`CREATE TABLE IF NOT EXISTS logical_clock (', fts_schema + '`CREATE TABLE IF NOT EXISTS logical_clock (')
else:
    print("Could not find insertion point for FTS schema")

# Add rebuild FTS command to populate it if it was just created (optional but good for dev)
rebuild_logic = """
	// Rebuild FTS if empty (heuristic)
    // Actually, triggers handle new data. For existing data, we might need a one-time backfill.
    // For simplicity in this session, we assume new app or we can add a manual backfill step if needed.
"""

# Add reset logic for new tables
reset_stmts = """
		`DELETE FROM messages_fts;`,
		`DELETE FROM comments_fts;`,
		`DELETE FROM hashtags;`,
"""
if '`DELETE FROM logical_clock;`,' in content:
    content = content.replace('`DELETE FROM logical_clock;`,', reset_stmts + '`DELETE FROM logical_clock;`,')

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
