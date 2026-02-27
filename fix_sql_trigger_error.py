import re

file_path = 'aegis-app/db.go'

with open(file_path, 'r') as f:
    content = f.read()

# "SQL logic error (1)" during delete usually means a trigger failed.
# The trigger I added:
# `CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
#     INSERT INTO messages_fts(messages_fts, rowid, title, body, content, content_rowid)
#     VALUES('delete', old.rowid, old.title, old.body, old.content, old.rowid);
#     INSERT INTO messages_fts(rowid, title, body, content, content_rowid)
#     VALUES (new.rowid, new.title, new.body, new.content, new.rowid);
# END;`

# FTS5 external content triggers need to handle 'delete' carefully.
# If `content_rowid` column doesn't exist in FTS5 table, using it in INSERT might fail if FTS5 table structure is wrong.
# I defined:
# `CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
#     title,
#     body,
#     content,
#     content_rowid UNINDEXED
# );`
# This seems correct.

# However, the error might be because I didn't enable FTS5 properly or some columns are NULL.
# FTS5 doesn't like NULLs in indexed columns sometimes?
# Or maybe the trigger syntax for FTS5 'delete' command is slightly different?
# `INSERT INTO messages_fts(messages_fts, rowid, ...) VALUES('delete', ...)` is correct for 'delete' command.

# BUT, I am using `content_rowid` as a user column in FTS5.
# If I use `rowid` in the VALUES list, I am explicitly setting the FTS5 rowid.
# `messages` table is `id TEXT PRIMARY KEY`. Does it have a stable `rowid`? Yes, unless WITHOUT ROWID.
# It seems `messages` is NOT `WITHOUT ROWID`.

# Wait, `messages_au` (Update Trigger) tries to do a delete and insert.
# The error happens in `delete post: SQL logic error` in test `TestPostDeleteRejectsStaleReplay`.
# The test does an `UPDATE messages SET ...` to mark it deleted.
# So `messages_au` is firing.
# The error might be `old.content` being NULL if the row was inserted with defaults?
# `content` TEXT NOT NULL. `body` TEXT NOT NULL DEFAULT ''.
# So they shouldn't be NULL.

# Let's try to simplify the trigger or debug it.
# Maybe `messages_fts` table name usage inside INSERT is redundant?
# `INSERT INTO messages_fts(messages_fts, ...)`
# Yes, `messages_fts` column is the magic column.

# Another possibility: `modernc.org/sqlite` might not support FTS5 by default or requires a build tag?
# Usually it supports it.

# Let's check `comments_au` as well.

# One potential issue: When `deleteLocalPostAsAuthor` runs, it updates `body` to `''`.
# `old.body` exists. `new.body` is `''`.
# This is fine.

# What if `content_rowid` is NOT unique or causes issues?
# Or `messages` table rowid changes? (It shouldn't).

# Actually, the error might be simpler: `no such column: rowid` if I used `WITHOUT ROWID`?
# I checked schema, it's `CREATE TABLE IF NOT EXISTS messages (...)`. Default is ROWID.

# Let's try to DROP the triggers and see if tests pass, to confirm it's the triggers.
# But I can't easily drop them in the file without replacing code.

# Wait, `TestPostDeleteRejectsStaleReplay` fails at `db_lamport_consistency_test.go:44`: `delete post: SQL logic error (1)`
# This line calls `app.deleteLocalPostAsAuthor`.
# This executes an UPDATE.
# So `messages_au` is definitely the culprit.

# Syntax check:
# `INSERT INTO messages_fts(messages_fts, rowid, title, body, content, content_rowid)`
# `VALUES('delete', old.rowid, old.title, old.body, old.content, old.rowid);`
# If I use `rowid` column in the insert list, I am specifying the rowid of the entry to be deleted.
# For 'delete' command, we need to provide the values of the columns to remove them from index.
# Do we need `rowid` in the column list?
# "To remove a row from the FTS5 table... execute an INSERT statement... with the special column... set to 'delete'."
# "The values inserted into the other columns... must match the values currently stored..."
# "If the rowid field is specified... then the rowid must also match."

# Maybe the issue is providing `content_rowid` in the column list for the 'delete' command?
# `content_rowid` is UNINDEXED. FTS5 ignores UNINDEXED columns for query matching, but they are stored.
# Does 'delete' require unindexed columns to match?
# Documentation says: "For each column in the FTS5 table, the value...".
# So yes, I must provide it.

# Is it possible `old.rowid` is unavailable in the trigger? No.

# Let's try a safer trigger that doesn't use the 'delete' magic but just does DELETE?
# `DELETE FROM messages_fts WHERE rowid = old.rowid;`
# But FTS5 standard `DELETE` is often implemented as `INSERT ... 'delete'`.
# If I use standard `DELETE`, does it work?
# "A DELETE statement... is equivalent to: INSERT INTO ft(ft, rowid, ...) VALUES('delete', rowid, ...)"
# So `DELETE FROM messages_fts WHERE rowid = old.rowid` should work and might be safer if I messed up the column list.

# Let's rewrite the UPDATE triggers to use DELETE + INSERT instead of magic INSERT 'delete'.

# Replacement:
# `CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
#   DELETE FROM messages_fts WHERE rowid = old.rowid;
#   INSERT INTO messages_fts(rowid, title, body, content, content_rowid) VALUES (new.rowid, new.title, new.body, new.content, new.rowid);
# END;`

# Same for comments.

old_msg_trigger = """		`CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
			INSERT INTO messages_fts(messages_fts, rowid, title, body, content, content_rowid)
			VALUES('delete', old.rowid, old.title, old.body, old.content, old.rowid);
			INSERT INTO messages_fts(rowid, title, body, content, content_rowid)
			VALUES (new.rowid, new.title, new.body, new.content, new.rowid);
		END;`,"""

new_msg_trigger = """		`CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
			DELETE FROM messages_fts WHERE rowid = old.rowid;
			INSERT INTO messages_fts(rowid, title, body, content, content_rowid)
			VALUES (new.rowid, new.title, new.body, new.content, new.rowid);
		END;`,"""

old_cmt_trigger = """		`CREATE TRIGGER IF NOT EXISTS comments_au AFTER UPDATE ON comments BEGIN
			INSERT INTO comments_fts(comments_fts, rowid, body, content_rowid)
			VALUES('delete', old.rowid, old.body, old.rowid);
			INSERT INTO comments_fts(rowid, body, content_rowid)
			VALUES (new.rowid, new.body, new.rowid);
		END;`,"""

new_cmt_trigger = """		`CREATE TRIGGER IF NOT EXISTS comments_au AFTER UPDATE ON comments BEGIN
			DELETE FROM comments_fts WHERE rowid = old.rowid;
			INSERT INTO comments_fts(rowid, body, content_rowid)
			VALUES (new.rowid, new.body, new.rowid);
		END;`,"""

# Remove whitespace sensitivity
content = content.replace(old_msg_trigger.strip(), new_msg_trigger.strip())
content = content.replace(old_cmt_trigger.strip(), new_cmt_trigger.strip())

# The previous replace might fail due to indentation/newlines matching.
# Let's try regex.

content = re.sub(r'CREATE TRIGGER IF NOT EXISTS messages_au.*?END;`,', new_msg_trigger.strip(), content, flags=re.DOTALL)
content = re.sub(r'CREATE TRIGGER IF NOT EXISTS comments_au.*?END;`,', new_cmt_trigger.strip(), content, flags=re.DOTALL)

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
