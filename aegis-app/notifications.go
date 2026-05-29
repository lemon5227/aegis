package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)



// Notification represents a single notification record.
type Notification struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	SourcePubkey   string `json:"sourcePubkey"`
	TargetEntityID string `json:"targetEntityId"`
	TargetType     string `json:"targetType"`
	PostID         string `json:"postId"`
	IsRead         bool   `json:"isRead"`
	CreatedAt      int64  `json:"createdAt"`
}

// NotificationPage represents a paginated list of notifications.
type NotificationPage struct {
	Items      []Notification `json:"items"`
	NextCursor string         `json:"nextCursor"`
}

// buildNotificationID generates a deterministic notification ID from the dedup key.
func buildNotificationID(notifType, sourcePubkey, targetEntityID string) string {
	raw := fmt.Sprintf("%s|%s|%s", notifType, sourcePubkey, targetEntityID)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])[:16]
}

// tryGenerateNotification attempts to insert a notification. Failures are logged, never returned.
func (a *App) tryGenerateNotification(notifType, sourcePubkey, targetEntityID, targetType, postID string, createdAt int64) {
	identity, err := a.getLocalIdentity()
	if err != nil {
		return // no local identity — nothing to notify
	}
	localPubkey := strings.TrimSpace(identity.PublicKey)
	if localPubkey == "" {
		return
	}
	// Never notify yourself.
	if sourcePubkey == localPubkey {
		return
	}

	id := buildNotificationID(notifType, sourcePubkey, targetEntityID)

	res, err := a.db.Exec(
		`INSERT OR IGNORE INTO notifications (id, type, source_pubkey, target_entity_id, target_type, post_id, is_read, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?)`,
		id, notifType, sourcePubkey, targetEntityID, targetType, postID, createdAt,
	)
	if err != nil {
		log.Printf("[notifications] insert error: %v", err)
		return
	}
	rows, _ := res.RowsAffected()
	if rows > 0 && a.ctx != nil {
		a.emitEvent("notifications:updated")
	}
}

// getPostAuthor returns the pubkey of a post's author.
func (a *App) getPostAuthor(postID string) (string, error) {
	var pubkey string
	err := a.db.QueryRow(`SELECT pubkey FROM messages WHERE id = ?`, postID).Scan(&pubkey)
	return pubkey, err
}

// getCommentAuthor returns the pubkey of a comment's author.
func (a *App) getCommentAuthor(commentID string) (string, error) {
	var pubkey string
	err := a.db.QueryRow(`SELECT pubkey FROM comments WHERE id = ?`, commentID).Scan(&pubkey)
	return pubkey, err
}

// maybeNotifyPostVote emits a post-vote notification when the post's author
// is the local node. It is a no-op when the post author is unknown, the local
// identity is missing, or the local node is not the post author. Errors are
// swallowed so notification failures never abort message processing.
func (a *App) maybeNotifyPostVote(postID, voterPubkey, voteState string, timestamp int64) {
	postID = strings.TrimSpace(postID)
	if postID == "" {
		return
	}
	author, err := a.getPostAuthor(postID)
	if err != nil || strings.TrimSpace(author) == "" {
		return
	}
	localID, liErr := a.getLocalIdentity()
	if liErr != nil || strings.TrimSpace(localID.PublicKey) != strings.TrimSpace(author) {
		return
	}
	notifType := NotifTypePostUpvote
	if strings.TrimSpace(strings.ToLower(voteState)) == "down" {
		notifType = NotifTypePostDownvote
	}
	a.tryGenerateNotification(notifType, voterPubkey, postID, "post", postID, timestamp)
}

// maybeNotifyCommentVote emits a comment-vote notification when the comment's
// author is the local node. Same no-op semantics as maybeNotifyPostVote.
func (a *App) maybeNotifyCommentVote(commentID, postID, voterPubkey, voteState string, timestamp int64) {
	commentID = strings.TrimSpace(commentID)
	if commentID == "" {
		return
	}
	author, err := a.getCommentAuthor(commentID)
	if err != nil || strings.TrimSpace(author) == "" {
		return
	}
	localID, liErr := a.getLocalIdentity()
	if liErr != nil || strings.TrimSpace(localID.PublicKey) != strings.TrimSpace(author) {
		return
	}
	notifType := NotifTypeCommentUpvote
	if strings.TrimSpace(strings.ToLower(voteState)) == "down" {
		notifType = NotifTypeCommentDownvote
	}
	a.tryGenerateNotification(notifType, voterPubkey, commentID, "comment", strings.TrimSpace(postID), timestamp)
}

// encodeNotificationCursor encodes a cursor from created_at and id.
func encodeNotificationCursor(createdAt int64, id string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d|%s", createdAt, id)))
}

// decodeNotificationCursor decodes a cursor into created_at and id.
func decodeNotificationCursor(cursor string) (int64, string, error) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, "", fmt.Errorf("invalid cursor encoding: %w", err)
	}
	parts := strings.SplitN(string(data), "|", 2)
	if len(parts) != 2 {
		return 0, "", errors.New("invalid cursor format")
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid cursor timestamp: %w", err)
	}
	return ts, parts[1], nil
}

// scanNotifications consumes a *sql.Rows that selects the canonical notification
// columns and returns the decoded list. The caller owns rows and must close it.
func scanNotifications(rows *sql.Rows) ([]Notification, error) {
	var out []Notification
	for rows.Next() {
		var (
			n      Notification
			isRead int
		)
		if err := rows.Scan(&n.ID, &n.Type, &n.SourcePubkey, &n.TargetEntityID, &n.TargetType, &n.PostID, &isRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		n.IsRead = isRead != 0
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// GetNotifications returns a paginated list of notifications ordered by created_at DESC.
func (a *App) GetNotifications(limit int, cursor string) (NotificationPage, error) {
	if a.db == nil {
		return NotificationPage{}, errors.New("database not initialized")
	}
	if limit <= 0 {
		limit = 20
	}

	const baseSelect = `SELECT id, type, source_pubkey, target_entity_id, target_type, post_id, is_read, created_at
		 FROM notifications`

	var (
		rows *sql.Rows
		err  error
	)
	if cursor == "" {
		rows, err = a.db.Query(baseSelect+` ORDER BY created_at DESC, id DESC LIMIT ?`, limit+1)
	} else {
		ts, cursorID, decErr := decodeNotificationCursor(cursor)
		if decErr != nil {
			return NotificationPage{}, decErr
		}
		rows, err = a.db.Query(
			baseSelect+` WHERE (created_at < ? OR (created_at = ? AND id < ?))
			 ORDER BY created_at DESC, id DESC LIMIT ?`,
			ts, ts, cursorID, limit+1,
		)
	}
	if err != nil {
		return NotificationPage{}, err
	}
	defer rows.Close()

	items, err := scanNotifications(rows)
	if err != nil {
		return NotificationPage{}, err
	}

	page := NotificationPage{Items: make([]Notification, 0)}
	if len(items) > limit {
		page.Items = items[:limit]
		last := items[limit-1]
		page.NextCursor = encodeNotificationCursor(last.CreatedAt, last.ID)
	} else if items != nil {
		page.Items = items
	}
	return page, nil
}

// GetUnreadNotificationCount returns the number of unread notifications.
func (a *App) GetUnreadNotificationCount() (int, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}
	var count int
	err := a.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE is_read = 0`).Scan(&count)
	return count, err
}

// MarkNotificationRead marks a single notification as read (idempotent).
func (a *App) MarkNotificationRead(notificationID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	_, err := a.db.Exec(`UPDATE notifications SET is_read = 1 WHERE id = ?`, notificationID)
	if err != nil {
		return err
	}
	if a.ctx != nil {
		a.emitEvent("notifications:updated")
	}
	return nil
}

// MarkAllNotificationsRead marks all unread notifications as read.
func (a *App) MarkAllNotificationsRead() error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	_, err := a.db.Exec(`UPDATE notifications SET is_read = 1 WHERE is_read = 0`)
	if err != nil {
		return err
	}
	if a.ctx != nil {
		a.emitEvent("notifications:updated")
	}
	return nil
}
