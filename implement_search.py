import re

file_path = 'aegis-app/app.go'

with open(file_path, 'r') as f:
    content = f.read()

# Update SearchPosts to use FTS
# We need to find the `SearchPosts` function and replace its query logic.

search_posts_pattern = r'func \(a \*App\) SearchPosts\(keyword string, subID string, limit int\) \(\[\]ForumMessage, error\) \{'
# We will replace the body.

# New SearchPosts body using FTS
new_search_posts = """func (a *App) SearchPosts(keyword string, subID string, limit int) ([]ForumMessage, error) {
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

    // Sanitize keyword for FTS
    // Remove characters that might break FTS syntax like ":", "*", etc if strictly typed,
    // but usually we just wrap in quotes or use it as tokens.
    // Simple approach: replace non-alphanumeric with spaces, or just allow standard syntax.
    // Let's assume standard "keyword" search.

    // SQLite FTS5 query syntax:
    // "keyword" -> looks for exact phrase or tokens.
    // keyword* -> prefix match.
    // We will simple append * to tokens.
    tokens := strings.Fields(keyword)
    if len(tokens) == 0 {
        return []ForumMessage{}, nil
    }

    ftsQuery := ""
    for _, token := range tokens {
        ftsQuery += "\"" + token + "\"* "
    }
    ftsQuery = strings.TrimSpace(ftsQuery)

	subID = strings.TrimSpace(subID)
	var rows *sql.Rows
	var err error

    // We need to join messages with messages_fts.
    // messages_fts.rowid = messages.rowid
    // But wait, `messages.id` is TEXT PRIMARY KEY, not INTEGER ROWID.
    // SQLite tables have a hidden `rowid` unless WITHOUT ROWID.
    // `messages` is `id TEXT PRIMARY KEY`, so it has a `rowid`.

	if subID != "" {
		rows, err = a.db.Query(`
			SELECT m.id, m.pubkey, m.title, m.body, m.content_cid, m.content, m.score, m.timestamp, m.size_bytes, m.zone, m.sub_id, m.is_protected, m.visibility,
                   snippet(messages_fts, 0, '<b>', '</b>', '...', 10) as title_snip,
                   snippet(messages_fts, 1, '<b>', '</b>', '...', 30) as body_snip
			FROM messages m
			JOIN messages_fts fts ON fts.rowid = m.rowid
			WHERE messages_fts MATCH ?
			  AND m.zone = 'public'
			  AND (m.visibility = 'normal' OR m.pubkey = ?)
			  AND m.sub_id = ?
			ORDER BY rank
			LIMIT ?;
		`, ftsQuery, viewerPubkey, normalizeSubID(subID), limit)
	} else {
		rows, err = a.db.Query(`
			SELECT m.id, m.pubkey, m.title, m.body, m.content_cid, m.content, m.score, m.timestamp, m.size_bytes, m.zone, m.sub_id, m.is_protected, m.visibility,
                   snippet(messages_fts, 0, '<b>', '</b>', '...', 10) as title_snip,
                   snippet(messages_fts, 1, '<b>', '</b>', '...', 30) as body_snip
			FROM messages m
			JOIN messages_fts fts ON fts.rowid = m.rowid
			WHERE messages_fts MATCH ?
			  AND m.zone = 'public'
			  AND (m.visibility = 'normal' OR m.pubkey = ?)
			ORDER BY rank
			LIMIT ?;
		`, ftsQuery, viewerPubkey, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ForumMessage, 0, limit)
	for rows.Next() {
		var message ForumMessage
        var titleSnip, bodySnip string
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
            &titleSnip,
            &bodySnip,
		); err != nil {
			return nil, err
		}
        // Use snippets if available and relevant
        // For API simplicity, we might return snippets in a new field or overwrite Title/Body for display?
        // Let's overwrite Body with snippet if Body is long?
        // Or just keep it standard. For now, let's just return the standard fields.
        // We could add `Highlight` fields to `ForumMessage` struct but that requires struct update.
        // For now, let's stick to standard `ForumMessage` but enjoy the better ranking.
		result = append(result, message)
	}

	return result, rows.Err()
}
"""

# Implement SearchComments
search_comments_code = """
func (a *App) SearchComments(keyword string, limit int) ([]Comment, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

    keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []Comment{}, nil
	}
    limit = normalizeSearchLimit(limit)

    tokens := strings.Fields(keyword)
    if len(tokens) == 0 {
        return []Comment{}, nil
    }
    ftsQuery := ""
    for _, token := range tokens {
        ftsQuery += "\"" + token + "\"* "
    }
    ftsQuery = strings.TrimSpace(ftsQuery)

    rows, err := a.db.Query(`
        SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.body, c.attachments_json, c.score, c.timestamp, c.lamport
        FROM comments c
        JOIN comments_fts fts ON fts.rowid = c.rowid
        WHERE comments_fts MATCH ? AND c.deleted_at = 0
        ORDER BY rank
        LIMIT ?;
    `, ftsQuery, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    result := make([]Comment, 0, limit)
    for rows.Next() {
        var comment Comment
        var attachmentsJSON string
        if err := rows.Scan(
            &comment.ID,
            &comment.PostID,
            &comment.ParentID,
            &comment.Pubkey,
            &comment.Body,
            &attachmentsJSON,
            &comment.Score,
            &comment.Timestamp,
            &comment.Lamport,
        ); err != nil {
            return nil, err
        }
        comment.Attachments = decodeCommentAttachmentsJSON(attachmentsJSON)
        result = append(result, comment)
    }

    return result, rows.Err()
}
"""

# Replace SearchPosts
# We identify the function by signature
start_idx = content.find('func (a *App) SearchPosts(keyword string, subID string, limit int) ([]ForumMessage, error) {')
if start_idx != -1:
    # Find the end of this function.
    # It ends before `func (a *App) UpdateProfile`
    end_idx = content.find('func (a *App) UpdateProfile', start_idx)
    if end_idx != -1:
        # Check if there's any other function in between?
        # `SearchPosts` calls `normalizeSearchLimit` but that's a helper func at end of file usually.
        # Let's count braces to be safe.
        brace_count = 0
        actual_end = -1
        for i, char in enumerate(content[start_idx:]):
            if char == '{':
                brace_count += 1
            elif char == '}':
                brace_count -= 1
                if brace_count == 0:
                    actual_end = start_idx + i + 1
                    break

        if actual_end != -1:
            content = content[:start_idx] + new_search_posts + "\n\n" + search_comments_code + "\n\n" + content[actual_end:]
        else:
            print("Could not find end of SearchPosts")
    else:
        # Maybe it's the last function?
        print("Could not find next function anchor")
else:
    print("Could not find SearchPosts function")

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
