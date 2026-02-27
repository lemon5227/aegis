import re

file_path = 'aegis-app/app.go'

with open(file_path, 'r') as f:
    content = f.read()

# Add Notification methods
notification_methods = """
func (a *App) GetNotifications(limit int) ([]Notification, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := a.db.Query(`
		SELECT id, type, title, body, target_id, from_pubkey, created_at, read_at
		FROM notifications
		ORDER BY created_at DESC
		LIMIT ?;
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Notification, 0)
	for rows.Next() {
		var item Notification
		if err := rows.Scan(
			&item.ID,
			&item.Type,
			&item.Title,
			&item.Body,
			&item.TargetID,
			&item.FromPubkey,
			&item.CreatedAt,
			&item.ReadAt,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	return result, rows.Err()
}

func (a *App) MarkNotificationRead(id string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("notification id is required")
	}

	now := time.Now().Unix()
	_, err := a.db.Exec("UPDATE notifications SET read_at = ? WHERE id = ? AND read_at = 0", now, id)
	return err
}

func (a *App) MarkAllNotificationsRead() error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	now := time.Now().Unix()
	_, err := a.db.Exec("UPDATE notifications SET read_at = ? WHERE read_at = 0", now)
	return err
}

func (a *App) checkAndCreateNotification(comment Comment) {
    // This function checks if a comment should trigger a notification for the local user.
    // It should be called after a comment is successfully inserted.

    if a.db == nil {
        return
    }

    identity, err := a.getLocalIdentity()
    if err != nil {
        return
    }
    localPubkey := identity.PublicKey

    // Don't notify for own comments
    if comment.Pubkey == localPubkey {
        return
    }

    // Check if it's a reply to a post or comment authored by local user
    var parentAuthor string
    var notifType string
    var title string

    // First, check direct parent (comment)
    if comment.ParentID != "" {
        err := a.db.QueryRow("SELECT pubkey FROM comments WHERE id = ?", comment.ParentID).Scan(&parentAuthor)
        if err == nil && parentAuthor == localPubkey {
            notifType = "reply"
            title = "New reply to your comment"
        }
    }

    // If not a reply to a comment, or parent not found, check post author
    if notifType == "" {
        err := a.db.QueryRow("SELECT pubkey, title FROM messages WHERE id = ?", comment.PostID).Scan(&parentAuthor, &title)
        if err == nil && parentAuthor == localPubkey {
            notifType = "reply"
            if title == "" {
                title = "New reply to your post"
            } else {
                title = "Reply: " + title
            }
        }
    }

    // TODO: Mentions parsing (e.g. regex for @User)

    if notifType != "" {
        now := time.Now().Unix()
        id := buildMessageID(localPubkey, fmt.Sprintf("notif|%s|%d", comment.ID, now), now)

        _, err = a.db.Exec(`
            INSERT INTO notifications (id, type, title, body, target_id, from_pubkey, created_at, read_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, 0)
            ON CONFLICT(id) DO NOTHING;
        `, id, notifType, title, comment.Body, comment.ID, comment.Pubkey, now)

        if err == nil && a.ctx != nil {
             runtime.EventsEmit(a.ctx, "notifications:new")
        }
    }
}
"""

content += "\n" + notification_methods

# Hook into insertComment
# We need to call a.checkAndCreateNotification(comment) before return comment, nil
if 'return comment, nil' in content:
    # There are multiple returns, we want the one in insertComment
    # We can use regex to replace the specific return in insertComment
    # pattern: func (a *App) insertComment.*?return comment, nil

    # Simpler: Search for the function body and replace the last return
    # But insertComment might be used for sync too. That's actually desired (if we sync a reply to us, we want to know).

    # Let's find the insertComment function definition
    start_idx = content.find('func (a *App) insertComment(comment Comment) (Comment, error) {')
    if start_idx != -1:
        # Find the end of the function (rough heuristic or counting braces)
        # Assuming we just search for the specific return statement *after* the start_idx
        # The function likely ends with `return comment, nil` or `if err = tx.Commit(); ... return comment, nil`

        # We will replace `return comment, nil` with `a.checkAndCreateNotification(comment)\n\treturn comment, nil`
        # BEWARE: This might affect other functions if we are not careful.

        # Let's limit the search scope
        sub_content = content[start_idx:]
        end_idx = sub_content.find('\n}') # Very risky.

        # Better: Replace the specific string `if err = tx.Commit(); err != nil {\n\t\treturn Comment{}, err\n\t}\n\n\treturn comment, nil`

        target_str = """if err = tx.Commit(); err != nil {
		return Comment{}, err
	}

	return comment, nil"""

        replacement_str = """if err = tx.Commit(); err != nil {
		return Comment{}, err
	}

    a.checkAndCreateNotification(comment)
	return comment, nil"""

        if target_str in content:
            content = content.replace(target_str, replacement_str)
        else:
            print("Could not find insertComment return block to hook")
    else:
        print("Could not find insertComment function")

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
