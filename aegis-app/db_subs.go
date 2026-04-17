package main

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

func (a *App) GetSubStats(subID string) (SubStats, error) {
	if a.db == nil {
		return SubStats{}, errors.New("database not initialized")
	}

	subID = normalizeSubID(subID)
	if subID == "" {
		return SubStats{}, errors.New("sub id is required")
	}

	stats := SubStats{SubID: subID}
	if err := a.db.QueryRow(`
		SELECT COUNT(1)
		FROM sub_subscriptions
		WHERE sub_id = ?;
	`, subID).Scan(&stats.SubscriberCount); err != nil {
		return SubStats{}, err
	}

	if err := a.db.QueryRow(`
		SELECT COUNT(m.sub_id), COUNT(DISTINCT m.pubkey), COALESCE(MAX(subs.created_at), 0)
		FROM subs
		LEFT JOIN (
			SELECT sub_id, pubkey, timestamp
			FROM messages
			WHERE zone = 'public' AND visibility = 'normal'
		) m ON m.sub_id = subs.id
		WHERE subs.id = ?;
	`, subID).Scan(&stats.PostCount, &stats.ActiveAuthors, &stats.CreatedAt); err != nil {
		return SubStats{}, err
	}

	since := time.Now().Unix() - 24*60*60
	if err := a.db.QueryRow(`
		SELECT COUNT(1)
		FROM messages
		WHERE sub_id = ?
		  AND zone = 'public'
		  AND visibility = 'normal'
		  AND timestamp >= ?;
	`, subID, since).Scan(&stats.RecentPosts24h); err != nil {
		return SubStats{}, err
	}

	return stats, nil
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
		a.emitEvent("subs:subscriptions_updated")
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
		a.emitEvent("subs:subscriptions_updated")
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

	a.emitEvent("sub:updated", map[string]interface{}{
		"subId":     message.SubID,
		"postId":    message.ID,
		"title":     message.Title,
		"timestamp": message.Timestamp,
		"pubkey":    message.Pubkey,
	})
}

