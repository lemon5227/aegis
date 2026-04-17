package main

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (a *App) getLatestFavoriteOpTimestamp(pubkey string) (int64, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return 0, nil
	}

	var latest sql.NullInt64
	if err := a.db.QueryRow(`
		SELECT MAX(created_at)
		FROM post_favorite_ops
		WHERE pubkey = ?;
	`, pubkey).Scan(&latest); err != nil {
		return 0, err
	}
	if !latest.Valid {
		return 0, nil
	}

	return latest.Int64, nil
}

func (a *App) listFavoriteOpsSince(pubkey string, sinceTimestamp int64, limit int) ([]FavoriteOpRecord, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return []FavoriteOpRecord{}, nil
	}
	if sinceTimestamp < 0 {
		sinceTimestamp = 0
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	rows, err := a.db.Query(`
		SELECT op_id, pubkey, post_id, op, created_at, signature
		FROM post_favorite_ops
		WHERE pubkey = ? AND created_at >= ?
		ORDER BY created_at ASC, op_id ASC
		LIMIT ?;
	`, pubkey, sinceTimestamp, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]FavoriteOpRecord, 0, limit)
	for rows.Next() {
		var record FavoriteOpRecord
		if err = rows.Scan(
			&record.OpID,
			&record.Pubkey,
			&record.PostID,
			&record.Op,
			&record.CreatedAt,
			&record.Signature,
		); err != nil {
			return nil, err
		}
		result = append(result, record)
	}

	return result, rows.Err()
}

func (a *App) AddFavorite(postID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return errors.New("post id is required")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return errors.New("identity pubkey is empty")
	}

	exists, err := a.postExists(postID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("post not found")
	}

	active, err := a.isFavoritedByPubkey(pubkey, postID)
	if err != nil {
		return err
	}
	if active {
		return nil
	}

	record, err := a.buildLocalFavoriteOperation(identity, postID, "ADD")
	if err != nil {
		return err
	}

	applied, err := a.applyFavoriteOperation(record, true)
	if err != nil {
		return err
	}
	if !applied {
		return nil
	}

	a.emitFavoritesUpdated(postID)
	if err = a.publishFavoriteOperation(record); err != nil && a.ctx != nil {
		a.logWarningf("favorite publish failed op_id=%s post_id=%s err=%v", record.OpID, postID, err)
	}

	return nil
}

func (a *App) RemoveFavorite(postID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return errors.New("post id is required")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return errors.New("identity pubkey is empty")
	}

	active, err := a.isFavoritedByPubkey(pubkey, postID)
	if err != nil {
		return err
	}
	if !active {
		return nil
	}

	record, err := a.buildLocalFavoriteOperation(identity, postID, "REMOVE")
	if err != nil {
		return err
	}

	applied, err := a.applyFavoriteOperation(record, true)
	if err != nil {
		return err
	}
	if !applied {
		return nil
	}

	a.emitFavoritesUpdated(postID)
	if err = a.publishFavoriteOperation(record); err != nil && a.ctx != nil {
		a.logWarningf("favorite publish failed op_id=%s post_id=%s err=%v", record.OpID, postID, err)
	}

	return nil
}

func (a *App) IsFavorited(postID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return false, errors.New("post id is required")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return false, err
	}

	return a.isFavoritedByPubkey(identity.PublicKey, postID)
}

func (a *App) GetFavoritePostIDs() ([]string, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return nil, err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return nil, errors.New("identity pubkey is empty")
	}

	rows, err := a.db.Query(`
		SELECT post_id
		FROM post_favorites_state
		WHERE pubkey = ? AND state = 'active'
		ORDER BY updated_at DESC, post_id DESC;
	`, pubkey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]string, 0)
	for rows.Next() {
		var postID string
		if err = rows.Scan(&postID); err != nil {
			return nil, err
		}
		result = append(result, strings.TrimSpace(postID))
	}

	return result, rows.Err()
}

func (a *App) GetFavorites(limit int, cursor string) (PostIndexPage, error) {
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

	limit = normalizeFavoriteLimit(limit)
	cursorTs, cursorPostID, err := decodeFavoriteCursor(cursor)
	if err != nil {
		return PostIndexPage{}, err
	}

	args := []interface{}{pubkey, pubkey, pubkey}
	query := `
		SELECT
			m.id,
			m.pubkey,
			m.title,
			SUBSTR(m.body, 1, 140) AS body_preview,
			m.content_cid,
			m.image_cid,
			m.thumb_cid,
			m.image_mime,
			m.image_size,
			m.image_width,
			m.image_height,
			m.score,
			m.timestamp,
			m.zone,
			m.sub_id,
			m.visibility,
			s.updated_at,
			s.post_id
		FROM post_favorites_state s
		INNER JOIN messages m ON m.id = s.post_id
		WHERE s.pubkey = ?
		  AND s.state = 'active'
		  AND (
			(m.zone = 'public' AND (m.visibility = 'normal' OR m.pubkey = ?))
			OR (m.zone = 'private' AND m.pubkey = ?)
		  )
	`
	if cursorTs > 0 && cursorPostID != "" {
		query += `
		  AND (s.updated_at < ? OR (s.updated_at = ? AND s.post_id < ?))
		`
		args = append(args, cursorTs, cursorTs, cursorPostID)
	}
	query += `
		ORDER BY s.updated_at DESC, s.post_id DESC
		LIMIT ?;
	`
	args = append(args, limit+1)

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return PostIndexPage{}, err
	}
	defer rows.Close()

	type favoriteRow struct {
		item      PostIndex
		updatedAt int64
		postID    string
	}

	resultRows := make([]favoriteRow, 0, limit+1)
	for rows.Next() {
		var row favoriteRow
		if err = rows.Scan(
			&row.item.ID,
			&row.item.Pubkey,
			&row.item.Title,
			&row.item.BodyPreview,
			&row.item.ContentCID,
			&row.item.ImageCID,
			&row.item.ThumbCID,
			&row.item.ImageMIME,
			&row.item.ImageSize,
			&row.item.ImageWidth,
			&row.item.ImageHeight,
			&row.item.Score,
			&row.item.Timestamp,
			&row.item.Zone,
			&row.item.SubID,
			&row.item.Visibility,
			&row.updatedAt,
			&row.postID,
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
		page.NextCursor = encodeFavoriteCursor(cursorRow.updatedAt, cursorRow.postID)
		resultRows = resultRows[:limit]
	}

	for _, row := range resultRows {
		page.Items = append(page.Items, row.item)
	}

	return page, nil
}

func (a *App) isFavoritedByPubkey(pubkey string, postID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	if pubkey == "" || postID == "" {
		return false, nil
	}

	var state string
	err := a.db.QueryRow(`
		SELECT state
		FROM post_favorites_state
		WHERE pubkey = ? AND post_id = ?;
	`, pubkey, postID).Scan(&state)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return strings.EqualFold(strings.TrimSpace(state), "active"), nil
}

func (a *App) buildLocalFavoriteOperation(identity Identity, postID string, op string) (FavoriteOpRecord, error) {
	pubkey := strings.TrimSpace(identity.PublicKey)
	mnemonic := strings.TrimSpace(identity.Mnemonic)
	postID = strings.TrimSpace(postID)
	normalizedOp, err := normalizeFavoriteOperation(op)
	if err != nil {
		return FavoriteOpRecord{}, err
	}
	if pubkey == "" || mnemonic == "" || postID == "" {
		return FavoriteOpRecord{}, errors.New("invalid favorite operation identity")
	}

	now := time.Now()
	createdAt := now.Unix()
	opID := buildMessageID(pubkey, fmt.Sprintf("favorite|%s|%s|%d", normalizedOp, postID, now.UnixNano()), createdAt)
	signaturePayload := buildFavoriteSignaturePayload(pubkey, postID, normalizedOp, createdAt, opID)
	signature, err := a.SignMessage(mnemonic, signaturePayload)
	if err != nil {
		return FavoriteOpRecord{}, err
	}

	return FavoriteOpRecord{
		OpID:      opID,
		Pubkey:    pubkey,
		PostID:    postID,
		Op:        normalizedOp,
		CreatedAt: createdAt,
		Signature: signature,
	}, nil
}

func (a *App) applyFavoriteOperation(record FavoriteOpRecord, verifySignature bool) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	record.OpID = strings.TrimSpace(record.OpID)
	record.Pubkey = strings.TrimSpace(record.Pubkey)
	record.PostID = strings.TrimSpace(record.PostID)
	record.Signature = strings.TrimSpace(record.Signature)

	normalizedOp, err := normalizeFavoriteOperation(record.Op)
	if err != nil {
		return false, err
	}
	record.Op = normalizedOp

	if record.Pubkey == "" || record.PostID == "" {
		return false, errors.New("invalid favorite operation payload")
	}
	if record.CreatedAt <= 0 {
		record.CreatedAt = time.Now().Unix()
	}
	if record.OpID == "" {
		record.OpID = buildMessageID(record.Pubkey, fmt.Sprintf("favorite|%s|%s", record.Op, record.PostID), record.CreatedAt)
	}
	if verifySignature {
		if record.Signature == "" {
			return false, errors.New("favorite operation signature is required")
		}
		valid, verifyErr := a.verifyFavoriteOperationSignature(record)
		if verifyErr != nil {
			return false, verifyErr
		}
		if !valid {
			return false, errors.New("invalid favorite operation signature")
		}
	}

	nextState := favoriteStateForOperation(record.Op)
	tx, err := a.db.Begin()
	if err != nil {
		return false, err
	}

	result, err := tx.Exec(`
		INSERT INTO post_favorite_ops (op_id, pubkey, post_id, op, created_at, signature)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(op_id) DO NOTHING;
	`, record.OpID, record.Pubkey, record.PostID, record.Op, record.CreatedAt, record.Signature)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}

	insertedCount, err := result.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}

	if insertedCount == 0 {
		if err = tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}

	var existingUpdatedAt int64
	var existingLastOpID string
	err = tx.QueryRow(`
		SELECT updated_at, last_op_id
		FROM post_favorites_state
		WHERE pubkey = ? AND post_id = ?;
	`, record.Pubkey, record.PostID).Scan(&existingUpdatedAt, &existingLastOpID)

	shouldApply := true
	if err == nil {
		shouldApply = favoriteOperationWins(existingUpdatedAt, strings.TrimSpace(existingLastOpID), record.CreatedAt, record.OpID)
	} else if !errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		return false, err
	}

	if shouldApply {
		if _, err = tx.Exec(`
			INSERT INTO post_favorites_state (pubkey, post_id, state, updated_at, last_op_id)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(pubkey, post_id) DO UPDATE SET
				state = excluded.state,
				updated_at = excluded.updated_at,
				last_op_id = excluded.last_op_id;
		`, record.Pubkey, record.PostID, nextState, record.CreatedAt, record.OpID); err != nil {
			_ = tx.Rollback()
			return false, err
		}
	}

	if err = tx.Commit(); err != nil {
		return false, err
	}

	return shouldApply, nil
}

func (a *App) verifyFavoriteOperationSignature(record FavoriteOpRecord) (bool, error) {
	pubkey := strings.TrimSpace(record.Pubkey)
	postID := strings.TrimSpace(record.PostID)
	opID := strings.TrimSpace(record.OpID)
	signature := strings.TrimSpace(record.Signature)
	if pubkey == "" || postID == "" || opID == "" || signature == "" {
		return false, nil
	}

	normalizedOp, err := normalizeFavoriteOperation(record.Op)
	if err != nil {
		return false, err
	}
	payload := buildFavoriteSignaturePayload(pubkey, postID, normalizedOp, record.CreatedAt, opID)
	return a.VerifyMessage(pubkey, payload, signature)
}

func (a *App) emitFavoritesUpdated(postID string) {
	if a.ctx == nil {
		return
	}

	payload := map[string]string{
		"postId": strings.TrimSpace(postID),
	}
	a.emitEvent("favorites:updated", payload)
	a.emitEvent("feed:updated")
}

func normalizeFavoriteLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func normalizeFavoriteOperation(op string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(op))
	if normalized == "ADD" || normalized == "REMOVE" {
		return normalized, nil
	}

	return "", errors.New("invalid favorite operation")
}

func favoriteStateForOperation(op string) string {
	if strings.ToUpper(strings.TrimSpace(op)) == "ADD" {
		return "active"
	}

	return "removed"
}

func favoriteOperationWins(existingUpdatedAt int64, existingOpID string, incomingUpdatedAt int64, incomingOpID string) bool {
	if incomingUpdatedAt > existingUpdatedAt {
		return true
	}
	if incomingUpdatedAt < existingUpdatedAt {
		return false
	}

	return strings.Compare(strings.TrimSpace(incomingOpID), strings.TrimSpace(existingOpID)) > 0
}

func buildFavoriteSignaturePayload(pubkey string, postID string, op string, createdAt int64, opID string) string {
	return fmt.Sprintf(
		"favorite|%s|%s|%s|%d|%s",
		strings.TrimSpace(pubkey),
		strings.TrimSpace(postID),
		strings.ToUpper(strings.TrimSpace(op)),
		createdAt,
		strings.TrimSpace(opID),
	)
}

func encodeFavoriteCursor(updatedAt int64, postID string) string {
	raw := fmt.Sprintf("%d|%s", updatedAt, strings.TrimSpace(postID))
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func decodeFavoriteCursor(cursor string) (int64, string, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, "", nil
	}

	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, "", errors.New("invalid favorite cursor")
	}

	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return 0, "", errors.New("invalid favorite cursor")
	}

	updatedAt, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || updatedAt <= 0 {
		return 0, "", errors.New("invalid favorite cursor")
	}
	postID := strings.TrimSpace(parts[1])
	if postID == "" {
		return 0, "", errors.New("invalid favorite cursor")
	}

	return updatedAt, postID, nil
}
