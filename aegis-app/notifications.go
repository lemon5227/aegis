package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
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
		runtime.EventsEmit(a.ctx, "notifications:updated")
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

// getCommentPostID returns the post_id that a comment belongs to.
func (a *App) getCommentPostID(commentID string) (string, error) {
	var postID string
	err := a.db.QueryRow(`SELECT post_id FROM comments WHERE id = ?`, commentID).Scan(&postID)
	return postID, err
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

// GetNotifications returns a paginated list of notifications ordered by created_at DESC.
func (a *App) GetNotifications(limit int, cursor string) (NotificationPage, error) {
	if a.db == nil {
		return NotificationPage{}, errors.New("database not initialized")
	}
	if limit <= 0 {
		limit = 20
	}

	var rows_ []Notification
	var err error

	if cursor == "" {
		r, qErr := a.db.Query(
			`SELECT id, type, source_pubkey, target_entity_id, target_type, post_id, is_read, created_at
			 FROM notifications ORDER BY created_at DESC, id DESC LIMIT ?`, limit+1)
		if qErr != nil {
			return NotificationPage{}, qErr
		}
		defer r.Close()
		for r.Next() {
			var n Notification
			var isRead int
			if err = r.Scan(&n.ID, &n.Type, &n.SourcePubkey, &n.TargetEntityID, &n.TargetType, &n.PostID, &isRead, &n.CreatedAt); err != nil {
				return NotificationPage{}, err
			}
			n.IsRead = isRead != 0
			rows_ = append(rows_, n)
		}
		if err = r.Err(); err != nil {
			return NotificationPage{}, err
		}
	} else {
		ts, cursorID, decErr := decodeNotificationCursor(cursor)
		if decErr != nil {
			return NotificationPage{}, decErr
		}
		r, qErr := a.db.Query(
			`SELECT id, type, source_pubkey, target_entity_id, target_type, post_id, is_read, created_at
			 FROM notifications
			 WHERE (created_at < ? OR (created_at = ? AND id < ?))
			 ORDER BY created_at DESC, id DESC LIMIT ?`, ts, ts, cursorID, limit+1)
		if qErr != nil {
			return NotificationPage{}, qErr
		}
		defer r.Close()
		for r.Next() {
			var n Notification
			var isRead int
			if err = r.Scan(&n.ID, &n.Type, &n.SourcePubkey, &n.TargetEntityID, &n.TargetType, &n.PostID, &isRead, &n.CreatedAt); err != nil {
				return NotificationPage{}, err
			}
			n.IsRead = isRead != 0
			rows_ = append(rows_, n)
		}
		if err = r.Err(); err != nil {
			return NotificationPage{}, err
		}
	}

	page := NotificationPage{Items: make([]Notification, 0)}
	if len(rows_) > limit {
		page.Items = rows_[:limit]
		last := rows_[limit-1]
		page.NextCursor = encodeNotificationCursor(last.CreatedAt, last.ID)
	} else {
		page.Items = rows_
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
		runtime.EventsEmit(a.ctx, "notifications:updated")
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
		runtime.EventsEmit(a.ctx, "notifications:updated")
	}
	return nil
}
