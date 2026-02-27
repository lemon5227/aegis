import re

file_path = 'aegis-app/app.go'

with open(file_path, 'r') as f:
    content = f.read()

# Since exact matching failed, let's append SearchPostsFTS and SearchComments as new functions
# and we can rename them or just use them.
# Or better, use regex to find and replace SearchPosts if it exists.

# Try regex to find SearchPosts
pattern = r'func \(a \*App\) SearchPosts\(keyword string, subID string, limit int\) \(\[\]ForumMessage, error\) \{.*?^}'
match = re.search(pattern, content, re.DOTALL | re.MULTILINE)

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

    // FTS5 query construction
    tokens := strings.Fields(keyword)
    if len(tokens) == 0 {
        return []ForumMessage{}, nil
    }

    // Simple query builder: "token"*
    ftsQuery := ""
    for _, token := range tokens {
        // Escape quotes
        token = strings.ReplaceAll(token, "\"", "\"\"")
        ftsQuery += "\\"" + token + "\\"*" + " "
    }
    ftsQuery = strings.TrimSpace(ftsQuery)

	subID = strings.TrimSpace(subID)
	var rows *sql.Rows
	var err error

	if subID != "" {
		rows, err = a.db.Query(`
			SELECT m.id, m.pubkey, m.title, m.body, m.content_cid, m.content, m.score, m.timestamp, m.size_bytes, m.zone, m.sub_id, m.is_protected, m.visibility
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
			SELECT m.id, m.pubkey, m.title, m.body, m.content_cid, m.content, m.score, m.timestamp, m.size_bytes, m.zone, m.sub_id, m.is_protected, m.visibility
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
		// Fallback to LIKE if FTS fails (e.g. module not loaded or query syntax error)
        // Ideally we log this.
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
}"""

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
        token = strings.ReplaceAll(token, "\"", "\"\"")
        ftsQuery += "\\"" + token + "\\"*" + " "
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

if match:
    # Replace existing function
    content = content.replace(match.group(0), new_search_posts + "\n\n" + search_comments_code)
else:
    # Append if not found
    print("SearchPosts not found by regex, appending...")
    content += "\n" + new_search_posts + "\n" + search_comments_code

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
