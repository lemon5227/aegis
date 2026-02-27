import re

file_path = 'aegis-app/db.go'

with open(file_path, 'r') as f:
    content = f.read()

# Add new structs
new_structs = """
type Report struct {
	ID             string `json:"id"`
	TargetID       string `json:"targetId"`
	TargetType     string `json:"targetType"` // 'post' or 'comment'
	Reason         string `json:"reason"`
	ReporterPubkey string `json:"reporterPubkey"`
	Timestamp      int64  `json:"timestamp"`
	Status         string `json:"status"` // 'pending', 'resolved', 'ignored'
}

type Notification struct {
	ID        string `json:"id"`
	Type      string `json:"type"` // 'reply', 'mention', 'system'
	Title     string `json:"title"`
	Body      string `json:"body"`
	TargetID  string `json:"targetId"` // ID of post or comment
	FromPubkey string `json:"fromPubkey"`
	CreatedAt int64  `json:"createdAt"`
	ReadAt    int64  `json:"readAt"`
}
"""

# Insert new structs after existing structs (e.g., after KnownPeerExchange)
if 'type KnownPeerExchange struct' in content:
    content = content.replace('type KnownPeerExchange struct {', new_structs + '\ntype KnownPeerExchange struct {')
else:
    print("Could not find insertion point for structs")

# Add schema definitions
reports_table = """
		`CREATE TABLE IF NOT EXISTS reports (
			id TEXT PRIMARY KEY,
			target_id TEXT NOT NULL,
			target_type TEXT NOT NULL,
			reason TEXT NOT NULL,
			reporter_pubkey TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending'
		);`,
		`CREATE INDEX IF NOT EXISTS idx_reports_timestamp ON reports(timestamp DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_reports_target ON reports(target_id);`,
"""

notifications_table = """
		`CREATE TABLE IF NOT EXISTS notifications (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL DEFAULT '',
			target_id TEXT NOT NULL,
			from_pubkey TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			read_at INTEGER NOT NULL DEFAULT 0
		);`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_read_at ON notifications(read_at);`,
"""

# Insert table definitions into ensureSchema
if '`CREATE TABLE IF NOT EXISTS logical_clock (' in content:
    content = content.replace('`CREATE TABLE IF NOT EXISTS logical_clock (', reports_table + notifications_table + '`CREATE TABLE IF NOT EXISTS logical_clock (')
else:
    print("Could not find insertion point for schema")

# Add reset logic
reset_stmts = """
		`DELETE FROM notifications;`,
		`DELETE FROM reports;`,
"""
if '`DELETE FROM logical_clock;`,' in content:
    content = content.replace('`DELETE FROM logical_clock;`,', reset_stmts + '`DELETE FROM logical_clock;`,')

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
