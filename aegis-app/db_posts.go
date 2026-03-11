// db_posts.go — 帖子相关的数据库操作，包括 CRUD、Feed 查询、搜索、排序等。
package main

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

func (a *App) GetFeed() ([]ForumMessage, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	rows, err := a.db.Query(`
		SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
		FROM messages
		WHERE zone = 'public' AND (visibility = 'normal' OR (pubkey = ? AND visibility != 'deleted'))
		ORDER BY timestamp DESC
		LIMIT 200;
	`, viewerPubkey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]ForumMessage, 0)
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
		messages = append(messages, message)
	}

	return messages, rows.Err()
}
func (a *App) GetFeedBySub(subID string) ([]ForumMessage, error) {
	return a.GetFeedBySubSorted(subID, "hot")
}
func normalizeFeedSortMode(sortMode string) string {
	sortMode = strings.ToLower(strings.TrimSpace(sortMode))
	switch sortMode {
	case "", "hot":
		return "hot"
	case "new", "top-day", "top-week", "top-month", "top-all":
		return sortMode
	default:
		return "hot"
	}
}

func topWindowStartUnix(sortMode string, now int64) int64 {
	switch normalizeFeedSortMode(sortMode) {
	case "top-day":
		return now - 24*60*60
	case "top-week":
		return now - 7*24*60*60
	case "top-month":
		return now - 30*24*60*60
	default:
		return 0
	}
}
func (a *App) GetFeedBySubSorted(subID string, sortMode string) ([]ForumMessage, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	subID = normalizeSubID(subID)
	sortMode = normalizeFeedSortMode(sortMode)

	orderBy := "score DESC, timestamp DESC"
	if sortMode == "new" {
		orderBy = "timestamp DESC"
	}

	if sortMode == "new" {
		query := fmt.Sprintf(`
			SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
			FROM messages
			WHERE zone = 'public' AND (visibility = 'normal' OR (pubkey = ? AND visibility != 'deleted')) AND sub_id = ?
			ORDER BY %s
			LIMIT 200;
		`, orderBy)

		rows, err := a.db.Query(query, viewerPubkey, subID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		messages := make([]ForumMessage, 0)
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
			messages = append(messages, message)
		}

		return messages, rows.Err()
	}

	query := `
		SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
		FROM messages
		WHERE zone = 'public' AND (visibility = 'normal' OR (pubkey = ? AND visibility != 'deleted')) AND sub_id = ?
		ORDER BY timestamp DESC
		LIMIT 500;
	`

	rows, err := a.db.Query(query, viewerPubkey, subID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]ForumMessage, 0)
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
		messages = append(messages, message)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	if strings.HasPrefix(sortMode, "top-") {
		windowStart := topWindowStartUnix(sortMode, now)
		if windowStart > 0 {
			filtered := make([]ForumMessage, 0, len(messages))
			for _, message := range messages {
				if message.Timestamp >= windowStart {
					filtered = append(filtered, message)
				}
			}
			messages = filtered
		}
		sort.SliceStable(messages, func(i int, j int) bool {
			if messages[i].Score == messages[j].Score {
				return messages[i].Timestamp > messages[j].Timestamp
			}
			return messages[i].Score > messages[j].Score
		})
		if len(messages) > 200 {
			messages = messages[:200]
		}
		return messages, nil
	}
	sort.SliceStable(messages, func(i int, j int) bool {
		left := computeHotScore(messages[i].Score, messages[i].Timestamp, now)
		right := computeHotScore(messages[j].Score, messages[j].Timestamp, now)
		if left == right {
			return messages[i].Timestamp > messages[j].Timestamp
		}
		return left > right
	})

	if len(messages) > 200 {
		messages = messages[:200]
	}

	return messages, nil
}
func (a *App) GetFeedIndexBySubSorted(subID string, sortMode string) ([]PostIndex, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	subID = normalizeSubID(subID)
	sortMode = normalizeFeedSortMode(sortMode)

	query := `
		SELECT id, pubkey, title, SUBSTR(body, 1, 140) AS body_preview, content_cid, image_cid, thumb_cid, image_mime, image_size, image_width, image_height, score, timestamp, zone, sub_id, visibility
		FROM messages
		WHERE zone = 'public' AND (visibility = 'normal' OR (pubkey = ? AND visibility != 'deleted')) AND sub_id = ?
		ORDER BY timestamp DESC
		LIMIT 500;
	`

	rows, err := a.db.Query(query, viewerPubkey, subID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PostIndex, 0)
	for rows.Next() {
		var item PostIndex
		if err = rows.Scan(
			&item.ID,
			&item.Pubkey,
			&item.Title,
			&item.BodyPreview,
			&item.ContentCID,
			&item.ImageCID,
			&item.ThumbCID,
			&item.ImageMIME,
			&item.ImageSize,
			&item.ImageWidth,
			&item.ImageHeight,
			&item.Score,
			&item.Timestamp,
			&item.Zone,
			&item.SubID,
			&item.Visibility,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	if strings.HasPrefix(sortMode, "top-") {
		windowStart := topWindowStartUnix(sortMode, now)
		if windowStart > 0 {
			filtered := make([]PostIndex, 0, len(items))
			for _, item := range items {
				if item.Timestamp >= windowStart {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
		sort.SliceStable(items, func(i int, j int) bool {
			if items[i].Score == items[j].Score {
				return items[i].Timestamp > items[j].Timestamp
			}
			return items[i].Score > items[j].Score
		})
	} else if sortMode == "hot" {
		sort.SliceStable(items, func(i int, j int) bool {
			left := computeHotScore(items[i].Score, items[i].Timestamp, now)
			right := computeHotScore(items[j].Score, items[j].Timestamp, now)
			if left == right {
				return items[i].Timestamp > items[j].Timestamp
			}
			return left > right
		})
	}

	if len(items) > 200 {
		items = items[:200]
	}

	return items, nil
}
func (a *App) GetPostIndexByID(postID string) (PostIndex, error) {
	if a.db == nil {
		return PostIndex{}, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return PostIndex{}, errors.New("post id is required")
	}

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	var item PostIndex
	err := a.db.QueryRow(`
		SELECT id, pubkey, title, SUBSTR(body, 1, 140) AS body_preview, content_cid, image_cid, thumb_cid, image_mime, image_size, image_width, image_height, score, timestamp, zone, sub_id, visibility
		FROM messages
		WHERE id = ?
		  AND (
			(zone = 'public' AND (visibility = 'normal' OR (pubkey = ? AND visibility != 'deleted')))
			OR (zone = 'private' AND pubkey = ?)
		  )
		LIMIT 1;
	`, postID, viewerPubkey, viewerPubkey).Scan(
		&item.ID,
		&item.Pubkey,
		&item.Title,
		&item.BodyPreview,
		&item.ContentCID,
		&item.ImageCID,
		&item.ThumbCID,
		&item.ImageMIME,
		&item.ImageSize,
		&item.ImageWidth,
		&item.ImageHeight,
		&item.Score,
		&item.Timestamp,
		&item.Zone,
		&item.SubID,
		&item.Visibility,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return PostIndex{}, errors.New("post not found")
	}
	if err != nil {
		return PostIndex{}, err
	}

	return item, nil
}
func (a *App) GetMyPosts(limit int, cursor string) (PostIndexPage, error) {
	if a.db == nil {
		return PostIndexPage{}, errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return PostIndexPage{}, err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return PostIndexPage{}, errors.New("identity pubkey is empty")
	}

	limit = normalizeMyPostsLimit(limit)
	cursorTs, cursorPostID, err := decodeMyPostsCursor(cursor)
	if err != nil {
		return PostIndexPage{}, err
	}

	args := []interface{}{pubkey}
	query := `
		SELECT
			id,
			pubkey,
			title,
			SUBSTR(body, 1, 140) AS body_preview,
			content_cid,
			image_cid,
			thumb_cid,
			image_mime,
			image_size,
			image_width,
			image_height,
			score,
			timestamp,
			zone,
			sub_id,
			visibility
		FROM messages
		WHERE pubkey = ? AND visibility != 'deleted'
	`
	if cursorTs > 0 && cursorPostID != "" {
		query += `
		  AND (timestamp < ? OR (timestamp = ? AND id < ?))
		`
		args = append(args, cursorTs, cursorTs, cursorPostID)
	}
	query += `
		ORDER BY timestamp DESC, id DESC
		LIMIT ?;
	`
	args = append(args, limit+1)

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return PostIndexPage{}, err
	}
	defer rows.Close()

	resultRows := make([]PostIndex, 0, limit+1)
	for rows.Next() {
		var row PostIndex
		if err = rows.Scan(
			&row.ID,
			&row.Pubkey,
			&row.Title,
			&row.BodyPreview,
			&row.ContentCID,
			&row.ImageCID,
			&row.ThumbCID,
			&row.ImageMIME,
			&row.ImageSize,
			&row.ImageWidth,
			&row.ImageHeight,
			&row.Score,
			&row.Timestamp,
			&row.Zone,
			&row.SubID,
			&row.Visibility,
		); err != nil {
			return PostIndexPage{}, err
		}
		resultRows = append(resultRows, row)
	}
	if err = rows.Err(); err != nil {
		return PostIndexPage{}, err
	}

	page := PostIndexPage{
		Items:      make([]PostIndex, 0, min(limit, len(resultRows))),
		NextCursor: "",
	}

	if len(resultRows) > limit {
		cursorRow := resultRows[limit-1]
		page.NextCursor = encodeMyPostsCursor(cursorRow.Timestamp, cursorRow.ID)
		resultRows = resultRows[:limit]
	}

	page.Items = append(page.Items, resultRows...)
	return page, nil
}
func (a *App) GetPostsByAuthor(pubkey string, limit int) ([]PostIndex, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return nil, errors.New("author pubkey is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	rows, err := a.db.Query(`
		SELECT
			id,
			pubkey,
			title,
			SUBSTR(body, 1, 140) AS body_preview,
			content_cid,
			image_cid,
			thumb_cid,
			image_mime,
			image_size,
			image_width,
			image_height,
			score,
			timestamp,
			zone,
			sub_id,
			visibility
		FROM messages
		WHERE pubkey = ?
		  AND visibility != 'deleted'
		  AND (
			(zone = 'public' AND (visibility = 'normal' OR pubkey = ?))
			OR (zone = 'private' AND pubkey = ? AND pubkey = ?)
		  )
		ORDER BY timestamp DESC, id DESC
		LIMIT ?;
	`, pubkey, viewerPubkey, viewerPubkey, pubkey, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PostIndex, 0, limit)
	for rows.Next() {
		var row PostIndex
		if err = rows.Scan(
			&row.ID,
			&row.Pubkey,
			&row.Title,
			&row.BodyPreview,
			&row.ContentCID,
			&row.ImageCID,
			&row.ThumbCID,
			&row.ImageMIME,
			&row.ImageSize,
			&row.ImageWidth,
			&row.ImageHeight,
			&row.Score,
			&row.Timestamp,
			&row.Zone,
			&row.SubID,
			&row.Visibility,
		); err != nil {
			return nil, err
		}
		items = append(items, row)
	}

	return items, rows.Err()
}
func computeHotScore(score int64, createdAt int64, now int64) float64 {
	ageHours := float64(now-createdAt) / 3600.0
	if ageHours < 0 {
		ageHours = 0
	}

	return float64(score) / math.Pow(ageHours+2, 1.2)
}
func (a *App) GetPrivateFeed() ([]ForumMessage, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
		FROM messages
		WHERE zone = 'private'
		ORDER BY timestamp DESC
		LIMIT 200;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]ForumMessage, 0)
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
		messages = append(messages, message)
	}

	return messages, rows.Err()
}
func (a *App) SearchPosts(keyword string, subID string, limit int) ([]ForumMessage, error) {
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

	lowerKeyword := strings.ToLower(keyword)
	pattern := "%" + lowerKeyword + "%"

	subID = strings.TrimSpace(subID)
	var rows *sql.Rows
	var err error
	if subID != "" {
		rows, err = a.db.Query(`
			SELECT m.id, m.pubkey, m.title, m.body, m.content_cid, m.content, m.score, m.timestamp, m.size_bytes, m.zone, m.sub_id, m.is_protected, m.visibility
			FROM messages m
			LEFT JOIN content_blobs cb ON cb.content_cid = m.content_cid
			WHERE m.zone = 'public'
			  AND (m.visibility = 'normal' OR m.pubkey = ?)
			  AND m.sub_id = ?
			  AND (
				LOWER(m.title) LIKE ?
				OR LOWER(m.body) LIKE ?
				OR LOWER(COALESCE(cb.body, '')) LIKE ?
			  )
			ORDER BY m.timestamp DESC
			LIMIT ?;
		`, viewerPubkey, normalizeSubID(subID), pattern, pattern, pattern, limit)
	} else {
		rows, err = a.db.Query(`
			SELECT m.id, m.pubkey, m.title, m.body, m.content_cid, m.content, m.score, m.timestamp, m.size_bytes, m.zone, m.sub_id, m.is_protected, m.visibility
			FROM messages m
			LEFT JOIN content_blobs cb ON cb.content_cid = m.content_cid
			WHERE m.zone = 'public'
			  AND (m.visibility = 'normal' OR m.pubkey = ?)
			  AND (
				LOWER(m.title) LIKE ?
				OR LOWER(m.body) LIKE ?
				OR LOWER(COALESCE(cb.body, '')) LIKE ?
			  )
			ORDER BY m.timestamp DESC
			LIMIT ?;
		`, viewerPubkey, pattern, pattern, pattern, limit)
	}
	if err != nil {
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
}
func (a *App) AddLocalPostStructured(pubkey string, title string, body string, zone string) (ForumMessage, error) {
	return a.AddLocalPostStructuredToSub(pubkey, title, body, zone, defaultSubID)
}
func (a *App) deleteLocalPostAsAuthor(pubkey string, postID string, deletedAt int64, lamport int64, opID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	if pubkey == "" || postID == "" {
		return errors.New("pubkey and post id are required")
	}
	if deletedAt <= 0 {
		deletedAt = time.Now().Unix()
	}
	if lamport <= 0 {
		lamport = deletedAt
	}
	opID = resolveOperationID(opID, postID, pubkey, lamport, postOpTypeDelete)

	var (
		author           string
		currentLamport   int64
		currentAuthorKey string
		currentOpID      string
		deletedLamport   int64
	)
	err := a.db.QueryRow(`
		SELECT pubkey, lamport, current_author_pubkey, current_op_id, deleted_at_lamport
		FROM messages
		WHERE id = ?;
	`, postID).Scan(&author, &currentLamport, &currentAuthorKey, &currentOpID, &deletedLamport)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("post not found")
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(author) != pubkey {
		return errors.New("only post author can delete this post")
	}
	if strings.TrimSpace(currentAuthorKey) == "" {
		currentAuthorKey = author
	}
	if strings.TrimSpace(currentOpID) == "" {
		currentOpID = postID
	}

	incomingVersion := LamportVersion{Lamport: lamport, Author: pubkey, OpID: opID}
	currentVersion := LamportVersion{Lamport: currentLamport, Author: currentAuthorKey, OpID: currentOpID}
	if compareLamportVersion(incomingVersion, currentVersion) <= 0 {
		return nil
	}
	if deletedLamport > 0 {
		tombstoneVersion := LamportVersion{Lamport: deletedLamport, Author: currentAuthorKey, OpID: currentOpID}
		if compareLamportVersion(incomingVersion, tombstoneVersion) <= 0 {
			return nil
		}
	}

	_, err = a.db.Exec(`
		UPDATE messages
		SET visibility = 'deleted',
		    title = CASE WHEN title = '' THEN title ELSE '[deleted]' END,
		    body = '',
		    content = '',
		    content_cid = '',
		    image_cid = '',
		    thumb_cid = '',
		    image_mime = '',
		    image_size = 0,
		    image_width = 0,
		    image_height = 0,
		    timestamp = ?,
		    lamport = ?,
		    current_author_pubkey = ?,
		    current_op_id = ?,
		    deleted_at_lamport = ?,
		    deleted_at_ts = ?,
		    deleted_by = ?
		WHERE id = ?;
	`, deletedAt, lamport, pubkey, opID, lamport, deletedAt, pubkey, postID)
	if err != nil {
		return err
	}
	return a.appendEntityOperation(
		entityTypePost,
		postID,
		postOpTypeDelete,
		opID,
		pubkey,
		lamport,
		deletedAt,
		lamportSchemaV2,
		authScopeUser,
		map[string]any{"deletedAtLamport": lamport},
	)
}
func (a *App) AddLocalPostStructuredToSub(pubkey string, title string, body string, zone string, subID string) (ForumMessage, error) {
	zone = strings.ToLower(strings.TrimSpace(zone))
	if zone != "private" && zone != "public" {
		return ForumMessage{}, errors.New("zone must be private or public")
	}

	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" {
		title = deriveTitle(body)
	}
	if title == "" || body == "" {
		return ForumMessage{}, errors.New("title and body are required")
	}

	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return ForumMessage{}, err
	}
	messageIDSeed := fmt.Sprintf("%s|%s|%d", title, body, lamport)
	messageID := buildMessageID(pubkey, messageIDSeed, now)
	message := ForumMessage{
		ID:          messageID,
		Pubkey:      pubkey,
		OpID:        generateOperationID(messageID, pubkey, lamport),
		Title:       title,
		Body:        body,
		ContentCID:  buildContentCID(body),
		Content:     "",
		Score:       0,
		Timestamp:   now,
		Lamport:     lamport,
		SizeBytes:   int64(len([]byte(body))),
		Zone:        zone,
		SubID:       normalizeSubID(subID),
		Visibility:  "normal",
		IsProtected: 0,
	}

	return a.insertMessage(message)
}
func (a *App) AddLocalPostWithImageToSub(pubkey string, title string, body string, zone string, subID string, imageBase64 string, imageMIME string) (ForumMessage, error) {
	message, err := a.AddLocalPostStructuredToSub(pubkey, title, body, zone, subID)
	if err != nil {
		return ForumMessage{}, err
	}

	imageBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(imageBase64))
	if err != nil || len(imageBytes) == 0 {
		return ForumMessage{}, errors.New("invalid image payload")
	}

	processedBytes, processedMime, width, height, thumbBytes, thumbMime, _, _, prepErr := prepareImageAssets(imageBytes, imageMIME)
	if prepErr != nil {
		return ForumMessage{}, prepErr
	}

	imageCID := buildBinaryCID(processedBytes)
	thumbCID := imageCID
	if err = a.upsertMediaBlobRaw(imageCID, processedMime, processedBytes, width, height, false); err != nil {
		return ForumMessage{}, err
	}

	if len(thumbBytes) > 0 {
		candidateThumbCID := buildBinaryCID(thumbBytes)
		thumbCID = candidateThumbCID
		if candidateThumbCID != imageCID {
			if err = a.upsertMediaBlobRaw(candidateThumbCID, thumbMime, thumbBytes, 0, 0, true); err != nil {
				return ForumMessage{}, err
			}
		}
	}

	if _, err = a.db.Exec(`
		UPDATE messages
		SET image_cid = ?, thumb_cid = ?, image_mime = ?, image_size = ?, image_width = ?, image_height = ?
		WHERE id = ?;
	`, imageCID, thumbCID, processedMime, int64(len(processedBytes)), width, height, message.ID); err != nil {
		return ForumMessage{}, err
	}

	message.ImageCID = imageCID
	message.ThumbCID = thumbCID
	message.ImageMIME = processedMime
	message.ImageSize = int64(len(processedBytes))
	message.ImageWidth = width
	message.ImageHeight = height

	return message, nil
}
func (a *App) insertMessage(message ForumMessage) (ForumMessage, error) {
	if a.db == nil {
		return ForumMessage{}, errors.New("database not initialized")
	}

	if message.Zone != "private" && message.Zone != "public" {
		return ForumMessage{}, errors.New("invalid message zone")
	}
	message.ID = strings.TrimSpace(message.ID)
	message.Pubkey = strings.TrimSpace(message.Pubkey)
	if message.ID == "" || message.Pubkey == "" {
		return ForumMessage{}, errors.New("invalid message")
	}

	if message.SizeBytes <= 0 {
		message.SizeBytes = int64(len([]byte(message.Body)))
	}
	if message.Timestamp <= 0 {
		message.Timestamp = time.Now().Unix()
	}
	if message.Lamport <= 0 {
		message.Lamport = message.Timestamp
	}
	message.OpID = resolveOperationID(message.OpID, message.ID, message.Pubkey, message.Lamport, postOpTypeCreate)

	message.Title = strings.TrimSpace(message.Title)
	message.Body = strings.TrimSpace(message.Body)
	if message.Title == "" || message.Body == "" {
		return ForumMessage{}, errors.New("message title and body are required")
	}
	fullBody := message.Body
	message.ContentCID = strings.TrimSpace(message.ContentCID)
	if message.ContentCID == "" {
		message.ContentCID = buildContentCID(fullBody)
	}
	blobSizeBytes := int64(len([]byte(fullBody)))
	message.Body = deriveBodyPreview(fullBody, 180)
	message.SizeBytes = int64(len([]byte(message.Body)))

	message.SubID = normalizeSubID(message.SubID)
	if strings.TrimSpace(message.Visibility) == "" {
		message.Visibility = "normal"
	}

	quota := publicQuotaBytes
	if message.Zone == "private" {
		quota = privateQuotaBytes
	}

	var (
		existingPubkey       string
		existingLamport      int64
		existingAuthorPubkey string
		existingOpID         string
		existingDeletedL     int64
		existingVisibility   string
	)
	err := a.db.QueryRow(`
		SELECT pubkey, lamport, current_author_pubkey, current_op_id, deleted_at_lamport, visibility
		FROM messages
		WHERE id = ?;
	`, message.ID).Scan(&existingPubkey, &existingLamport, &existingAuthorPubkey, &existingOpID, &existingDeletedL, &existingVisibility)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ForumMessage{}, err
	}
	appliedOpType := postOpTypeCreate
	if err == nil {
		appliedOpType = postOpTypeUpdate
		existingPubkey = strings.TrimSpace(existingPubkey)
		if existingPubkey != "" && existingPubkey != message.Pubkey {
			return ForumMessage{}, errors.New("only post author can mutate this post")
		}

		if strings.TrimSpace(existingAuthorPubkey) == "" {
			existingAuthorPubkey = existingPubkey
		}
		if strings.TrimSpace(existingOpID) == "" {
			existingOpID = message.ID
		}

		incomingVersion := LamportVersion{Lamport: message.Lamport, Author: message.Pubkey, OpID: message.OpID}
		currentVersion := LamportVersion{Lamport: existingLamport, Author: existingAuthorPubkey, OpID: existingOpID}
		if compareLamportVersion(incomingVersion, currentVersion) <= 0 {
			return message, nil
		}

		if existingDeletedL > 0 {
			tombstoneVersion := LamportVersion{Lamport: existingDeletedL, Author: existingAuthorPubkey, OpID: existingOpID}
			if compareLamportVersion(incomingVersion, tombstoneVersion) <= 0 {
				return message, nil
			}
		}

		if strings.EqualFold(strings.TrimSpace(existingVisibility), "deleted") && existingDeletedL >= message.Lamport {
			return message, nil
		}
	}

	a.dbMu.Lock()
	defer a.dbMu.Unlock()

	tx, err := a.db.Begin()
	if err != nil {
		return ForumMessage{}, err
	}

	if err = ensureBlobQuotaWithLRU(tx, message.Zone, quota, blobSizeBytes, message.ContentCID); err != nil {
		_ = tx.Rollback()
		return ForumMessage{}, err
	}

	if _, err = tx.Exec(`
		INSERT INTO content_blobs (content_cid, body, size_bytes, created_at, last_accessed_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(content_cid) DO UPDATE SET
			body = excluded.body,
			size_bytes = excluded.size_bytes,
			last_accessed_at = excluded.last_accessed_at;
	`, message.ContentCID, fullBody, blobSizeBytes, message.Timestamp, message.Timestamp); err != nil {
		_ = tx.Rollback()
		return ForumMessage{}, err
	}

	_, err = tx.Exec(
		`INSERT INTO messages (
			id, pubkey, current_author_pubkey, current_op_id, title, body, content_cid, image_cid, thumb_cid, image_mime, image_size,
			image_width, image_height, content, score, timestamp, lamport, size_bytes, zone, sub_id, is_protected, visibility,
			deleted_at_lamport, deleted_at_ts, deleted_by
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 0, '')
		ON CONFLICT(id) DO UPDATE SET
			pubkey = excluded.pubkey,
			current_author_pubkey = excluded.current_author_pubkey,
			current_op_id = excluded.current_op_id,
			title = excluded.title,
			body = excluded.body,
			content_cid = excluded.content_cid,
			image_cid = excluded.image_cid,
			thumb_cid = excluded.thumb_cid,
			image_mime = excluded.image_mime,
			image_size = excluded.image_size,
			image_width = excluded.image_width,
			image_height = excluded.image_height,
			content = excluded.content,
			timestamp = excluded.timestamp,
			lamport = excluded.lamport,
			size_bytes = excluded.size_bytes,
			zone = excluded.zone,
			sub_id = excluded.sub_id,
			is_protected = excluded.is_protected,
			visibility = 'normal',
			deleted_at_lamport = 0,
			deleted_at_ts = 0,
			deleted_by = '';`,
		message.ID,
		message.Pubkey,
		message.Pubkey,
		message.OpID,
		message.Title,
		message.Body,
		message.ContentCID,
		message.ImageCID,
		message.ThumbCID,
		message.ImageMIME,
		message.ImageSize,
		message.ImageWidth,
		message.ImageHeight,
		message.Content,
		message.Score,
		message.Timestamp,
		message.Lamport,
		message.SizeBytes,
		message.Zone,
		message.SubID,
		message.IsProtected,
		message.Visibility,
	)
	if err != nil {
		_ = tx.Rollback()
		return ForumMessage{}, err
	}
	if err = a.appendEntityOperationTx(
		tx,
		entityTypePost,
		message.ID,
		appliedOpType,
		message.OpID,
		message.Pubkey,
		message.Lamport,
		message.Timestamp,
		lamportSchemaV2,
		authScopeUser,
		map[string]any{"subId": message.SubID, "zone": message.Zone, "visibility": message.Visibility},
	); err != nil {
		_ = tx.Rollback()
		return ForumMessage{}, err
	}

	if err = tx.Commit(); err != nil {
		return ForumMessage{}, err
	}

	return message, nil
}
func (a *App) postExists(postID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return false, nil
	}

	var exists int
	err := a.db.QueryRow(`SELECT 1 FROM messages WHERE id = ? LIMIT 1;`, postID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
func (a *App) queryPostsBySubSet(viewerPubkey string, subIDs []string, limit int) ([]ForumMessage, error) {
	if len(subIDs) == 0 {
		return []ForumMessage{}, nil
	}
	if limit <= 0 {
		limit = 20
	}

	placeholders := makeSQLPlaceholders(len(subIDs))
	args := make([]interface{}, 0, len(subIDs)+2)
	args = append(args, viewerPubkey)
	for _, subID := range subIDs {
		args = append(args, normalizeSubID(subID))
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
		FROM messages
		WHERE zone = 'public'
		  AND (visibility = 'normal' OR (pubkey = ? AND visibility != 'deleted'))
		  AND sub_id IN (%s)
		ORDER BY timestamp DESC
		LIMIT ?;
	`, placeholders)

	return a.queryForumMessages(query, args...)
}
func (a *App) queryRecommendedPosts(viewerPubkey string, subscribedSubIDs []string, limit int) ([]ForumMessage, error) {
	if limit <= 0 {
		limit = 40
	}

	if len(subscribedSubIDs) == 0 {
		return a.queryForumMessages(`
			SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
			FROM messages
			WHERE zone = 'public'
			  AND (visibility = 'normal' OR (pubkey = ? AND visibility != 'deleted'))
			ORDER BY score DESC, timestamp DESC
			LIMIT ?;
		`, viewerPubkey, limit)
	}

	placeholders := makeSQLPlaceholders(len(subscribedSubIDs))
	args := make([]interface{}, 0, len(subscribedSubIDs)+2)
	args = append(args, viewerPubkey)
	for _, subID := range subscribedSubIDs {
		args = append(args, normalizeSubID(subID))
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT id, pubkey, title, body, content_cid, content, score, timestamp, size_bytes, zone, sub_id, is_protected, visibility
		FROM messages
		WHERE zone = 'public'
		  AND (visibility = 'normal' OR (pubkey = ? AND visibility != 'deleted'))
		  AND sub_id NOT IN (%s)
		ORDER BY score DESC, timestamp DESC
		LIMIT ?;
	`, placeholders)

	return a.queryForumMessages(query, args...)
}
func (a *App) UpdateLocalPost(pubkey string, postID string, title string, body string) (ForumMessage, error) {
	if a.db == nil {
		return ForumMessage{}, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	if pubkey == "" || postID == "" {
		return ForumMessage{}, errors.New("pubkey and post id are required")
	}

	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	if title == "" && body == "" {
		return ForumMessage{}, errors.New("title or body must be updated")
	}

	// Verify post exists and user is author
	var (
		currentTitle  string
		currentBody   string
		currentSubID  string
		currentZone   string
		currentAuthor string
	)

	err := a.db.QueryRow(`
		SELECT title, body, sub_id, zone, pubkey
		FROM messages
		WHERE id = ?;
	`, postID).Scan(&currentTitle, &currentBody, &currentSubID, &currentZone, &currentAuthor)
	if errors.Is(err, sql.ErrNoRows) {
		return ForumMessage{}, errors.New("post not found")
	}
	if err != nil {
		return ForumMessage{}, err
	}

	if currentAuthor != pubkey {
		return ForumMessage{}, errors.New("only author can update post")
	}

	if title == "" {
		title = currentTitle
	}
	if body == "" {
		body = currentBody
	}

	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return ForumMessage{}, err
	}

	updatedPost := ForumMessage{
		ID:        postID,
		Pubkey:    pubkey,
		Title:     title,
		Body:      body, // Full body
		Timestamp: now,
		Lamport:   lamport,
		Zone:      currentZone,
		SubID:     currentSubID,
	}

	var (
		imgCid  string
		thbCid  string
		imgMime string
		imgSize int64
		imgW    int
		imgH    int
	)
	_ = a.db.QueryRow(`
		SELECT image_cid, thumb_cid, image_mime, image_size, image_width, image_height
		FROM messages WHERE id = ?
	`, postID).Scan(&imgCid, &thbCid, &imgMime, &imgSize, &imgW, &imgH)

	updatedPost.ImageCID = imgCid
	updatedPost.ThumbCID = thbCid
	updatedPost.ImageMIME = imgMime
	updatedPost.ImageSize = imgSize
	updatedPost.ImageWidth = imgW
	updatedPost.ImageHeight = imgH

	return a.insertMessage(updatedPost)
}
