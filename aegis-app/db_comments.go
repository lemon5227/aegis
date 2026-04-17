package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func (a *App) getLatestPublicCommentTimestamp() (int64, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}

	var latest sql.NullInt64
	if err := a.db.QueryRow(`
		SELECT MAX(c.timestamp)
		FROM comments c
		JOIN messages m ON m.id = c.post_id
		WHERE m.zone = 'public' AND m.visibility = 'normal' AND c.deleted_at = 0;
	`).Scan(&latest); err != nil {
		return 0, err
	}

	if !latest.Valid {
		return 0, nil
	}

	return latest.Int64, nil
}

func (a *App) listPublicCommentDigestsSince(sinceTimestamp int64, limit int) ([]SyncCommentDigest, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 500 {
		limit = 200
	}

	policy, policyErr := a.GetGovernancePolicy()
	if policyErr != nil {
		return nil, policyErr
	}

	var (
		rows *sql.Rows
		err  error
	)
	if sinceTimestamp > 0 {
		if policy.HideHistoryOnShadowBan {
			rows, err = a.db.Query(`
				SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.current_op_id, c.body, c.attachments_json, c.score, c.timestamp, c.lamport, c.deleted_at_lamport, c.deleted_at,
				       COALESCE(p.display_name, ''), COALESCE(p.avatar_url, '')
				FROM comments c
				JOIN messages m ON m.id = c.post_id
				LEFT JOIN profiles p ON p.pubkey = c.pubkey
				LEFT JOIN moderation mo ON mo.target_pubkey = c.pubkey
				WHERE m.zone = 'public' AND c.timestamp >= ?
				  AND (mo.action IS NULL OR UPPER(mo.action) != 'SHADOW_BAN')
				ORDER BY c.timestamp ASC
				LIMIT ?;
			`, sinceTimestamp, limit)
		} else {
			rows, err = a.db.Query(`
				SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.current_op_id, c.body, c.attachments_json, c.score, c.timestamp, c.lamport, c.deleted_at_lamport, c.deleted_at,
				       COALESCE(p.display_name, ''), COALESCE(p.avatar_url, '')
				FROM comments c
				JOIN messages m ON m.id = c.post_id
				LEFT JOIN profiles p ON p.pubkey = c.pubkey
				LEFT JOIN moderation mo ON mo.target_pubkey = c.pubkey
				WHERE m.zone = 'public' AND c.timestamp >= ?
				  AND (
					mo.action IS NULL
					OR UPPER(mo.action) != 'SHADOW_BAN'
					OR c.lamport < mo.lamport
					OR (c.lamport = 0 OR mo.lamport = 0) AND c.timestamp < mo.timestamp
				  )
				ORDER BY c.timestamp ASC
				LIMIT ?;
			`, sinceTimestamp, limit)
		}
	} else {
		if policy.HideHistoryOnShadowBan {
			rows, err = a.db.Query(`
				SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.current_op_id, c.body, c.attachments_json, c.score, c.timestamp, c.lamport, c.deleted_at_lamport, c.deleted_at,
				       COALESCE(p.display_name, ''), COALESCE(p.avatar_url, '')
				FROM comments c
				JOIN messages m ON m.id = c.post_id
				LEFT JOIN profiles p ON p.pubkey = c.pubkey
				LEFT JOIN moderation mo ON mo.target_pubkey = c.pubkey
				WHERE m.zone = 'public'
				  AND (mo.action IS NULL OR UPPER(mo.action) != 'SHADOW_BAN')
				ORDER BY c.timestamp DESC
				LIMIT ?;
			`, limit)
		} else {
			rows, err = a.db.Query(`
				SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.current_op_id, c.body, c.attachments_json, c.score, c.timestamp, c.lamport, c.deleted_at_lamport, c.deleted_at,
				       COALESCE(p.display_name, ''), COALESCE(p.avatar_url, '')
				FROM comments c
				JOIN messages m ON m.id = c.post_id
				LEFT JOIN profiles p ON p.pubkey = c.pubkey
				LEFT JOIN moderation mo ON mo.target_pubkey = c.pubkey
				WHERE m.zone = 'public'
				  AND (
					mo.action IS NULL
					OR UPPER(mo.action) != 'SHADOW_BAN'
					OR c.lamport < mo.lamport
					OR (c.lamport = 0 OR mo.lamport = 0) AND c.timestamp < mo.timestamp
				  )
				ORDER BY c.timestamp DESC
				LIMIT ?;
			`, limit)
		}
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SyncCommentDigest, 0, limit)
	for rows.Next() {
		var item SyncCommentDigest
		var attachmentsJSON string
		var deletedAtLamport int64
		var deletedAt int64
		if err = rows.Scan(
			&item.ID,
			&item.PostID,
			&item.ParentID,
			&item.Pubkey,
			&item.OpID,
			&item.Body,
			&attachmentsJSON,
			&item.Score,
			&item.Timestamp,
			&item.Lamport,
			&deletedAtLamport,
			&deletedAt,
			&item.DisplayName,
			&item.AvatarURL,
		); err != nil {
			return nil, err
		}
		item.Deleted = deletedAt > 0
		item.DeletedAtLamport = deletedAtLamport
		if item.Deleted {
			item.OpType = postOpTypeDelete
			item.Body = ""
			attachmentsJSON = "[]"
		} else {
			item.OpType = postOpTypeCreate
		}
		item.Attachments = decodeCommentAttachmentsJSON(attachmentsJSON)
		result = append(result, item)
	}

	return result, rows.Err()
}

func (a *App) listPublicCommentDigestsByPostSince(postID string, sinceTimestamp int64, limit int) ([]SyncCommentDigest, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return []SyncCommentDigest{}, nil
	}

	if limit <= 0 || limit > 500 {
		limit = 200
	}

	policy, err := a.GetGovernancePolicy()
	if err != nil {
		return nil, err
	}

	var rows *sql.Rows
	if policy.HideHistoryOnShadowBan {
		rows, err = a.db.Query(`
			SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.current_op_id, c.body, c.attachments_json, c.score, c.timestamp, c.lamport, c.deleted_at_lamport, c.deleted_at,
			       COALESCE(p.display_name, ''), COALESCE(p.avatar_url, '')
			FROM comments c
			JOIN messages m ON m.id = c.post_id
			LEFT JOIN profiles p ON p.pubkey = c.pubkey
			LEFT JOIN moderation mo ON mo.target_pubkey = c.pubkey
			WHERE m.zone = 'public' AND c.post_id = ? AND c.timestamp >= ?
			  AND (mo.action IS NULL OR UPPER(mo.action) != 'SHADOW_BAN')
			ORDER BY c.timestamp ASC
			LIMIT ?;
		`, postID, sinceTimestamp, limit)
	} else {
		rows, err = a.db.Query(`
			SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.current_op_id, c.body, c.attachments_json, c.score, c.timestamp, c.lamport, c.deleted_at_lamport, c.deleted_at,
			       COALESCE(p.display_name, ''), COALESCE(p.avatar_url, '')
			FROM comments c
			JOIN messages m ON m.id = c.post_id
			LEFT JOIN profiles p ON p.pubkey = c.pubkey
			LEFT JOIN moderation mo ON mo.target_pubkey = c.pubkey
			WHERE m.zone = 'public' AND c.post_id = ? AND c.timestamp >= ?
			  AND (
				mo.action IS NULL
				OR UPPER(mo.action) != 'SHADOW_BAN'
				OR c.lamport < mo.lamport
				OR (c.lamport = 0 OR mo.lamport = 0) AND c.timestamp < mo.timestamp
			  )
			ORDER BY c.timestamp ASC
			LIMIT ?;
		`, postID, sinceTimestamp, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SyncCommentDigest, 0, limit)
	for rows.Next() {
		var item SyncCommentDigest
		var attachmentsJSON string
		var deletedAtLamport int64
		var deletedAt int64
		if err = rows.Scan(
			&item.ID,
			&item.PostID,
			&item.ParentID,
			&item.Pubkey,
			&item.OpID,
			&item.Body,
			&attachmentsJSON,
			&item.Score,
			&item.Timestamp,
			&item.Lamport,
			&deletedAtLamport,
			&deletedAt,
			&item.DisplayName,
			&item.AvatarURL,
		); err != nil {
			return nil, err
		}
		item.Deleted = deletedAt > 0
		item.DeletedAtLamport = deletedAtLamport
		if item.Deleted {
			item.OpType = postOpTypeDelete
			item.Body = ""
			attachmentsJSON = "[]"
		} else {
			item.OpType = postOpTypeCreate
		}
		item.Attachments = decodeCommentAttachmentsJSON(attachmentsJSON)
		result = append(result, item)
	}

	return result, rows.Err()
}

func (a *App) GetCommentsByPost(postID string) ([]Comment, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return nil, errors.New("post id is required")
	}

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	policy, policyErr := a.GetGovernancePolicy()
	hideHistoryOnShadowBan := true
	if policyErr == nil {
		hideHistoryOnShadowBan = policy.HideHistoryOnShadowBan
	}

	query := `
		SELECT c.id, c.post_id, c.parent_id, c.pubkey, c.body, c.attachments_json, c.score, c.timestamp, c.lamport
		FROM comments c
		LEFT JOIN moderation m ON m.target_pubkey = c.pubkey
	`
	args := []interface{}{}
	query += `
		WHERE c.post_id = ? AND c.deleted_at = 0
	`
	args = append(args, postID)
	if hideHistoryOnShadowBan {
		query += `
		  AND (
			m.action IS NULL
			OR UPPER(m.action) != 'SHADOW_BAN'
			OR c.pubkey = ?
		  )
		`
		args = append(args, viewerPubkey)
	} else {
		query += `
		  AND (
			m.action IS NULL
			OR UPPER(m.action) != 'SHADOW_BAN'
			OR c.pubkey = ?
			OR c.lamport < m.lamport
			OR (c.lamport = 0 OR m.lamport = 0) AND c.timestamp < m.timestamp
		  )
		`
		args = append(args, viewerPubkey)
	}
	query += `
		ORDER BY c.timestamp ASC;
	`

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Comment, 0)
	for rows.Next() {
		var comment Comment
		var attachmentsJSON string
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.ParentID, &comment.Pubkey, &comment.Body, &attachmentsJSON, &comment.Score, &comment.Timestamp, &comment.Lamport); err != nil {
			return nil, err
		}
		comment.Attachments = decodeCommentAttachmentsJSON(attachmentsJSON)
		result = append(result, comment)
	}

	return result, rows.Err()
}

func (a *App) AddLocalComment(pubkey string, postID string, parentID string, body string) (Comment, error) {
	return a.AddLocalCommentWithAttachments(pubkey, postID, parentID, body, nil)
}

func (a *App) AddLocalCommentWithAttachments(pubkey string, postID string, parentID string, body string, attachments []CommentAttachment) (Comment, error) {
	if a.db == nil {
		return Comment{}, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	postID = strings.TrimSpace(postID)
	parentID = strings.TrimSpace(parentID)
	body = strings.TrimSpace(body)

	attachments = normalizeCommentAttachments(attachments)
	if pubkey == "" || postID == "" {
		return Comment{}, errors.New("pubkey and post id are required")
	}
	if body == "" && len(attachments) == 0 {
		return Comment{}, errors.New("comment content is required")
	}
	if locked, _, err := a.getPostLockState(postID); err != nil {
		return Comment{}, err
	} else if locked {
		return Comment{}, errors.New("post is locked")
	}

	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return Comment{}, err
	}
	attachmentsJSON, err := encodeCommentAttachmentsJSON(attachments)
	if err != nil {
		return Comment{}, err
	}
	raw := fmt.Sprintf("%s|%s|%s|%s|%d", postID, parentID, body, attachmentsJSON, lamport)
	commentID := buildMessageID(pubkey, raw, now)
	comment := Comment{
		ID:          commentID,
		PostID:      postID,
		ParentID:    parentID,
		Pubkey:      pubkey,
		OpID:        generateOperationID(commentID, pubkey, lamport),
		Body:        body,
		Attachments: attachments,
		Score:       0,
		Timestamp:   now,
		Lamport:     lamport,
	}

	return a.insertComment(comment)
}

func (a *App) deleteLocalCommentAsAuthor(pubkey string, commentID string, deletedAt int64, lamport int64, opID string) (string, error) {
	if a.db == nil {
		return "", errors.New("database not initialized")
	}
	pubkey = strings.TrimSpace(pubkey)
	commentID = strings.TrimSpace(commentID)
	if pubkey == "" || commentID == "" {
		return "", errors.New("pubkey and comment id are required")
	}
	if deletedAt <= 0 {
		deletedAt = time.Now().Unix()
	}
	if lamport <= 0 {
		lamport = deletedAt
	}
	opID = resolveOperationID(opID, commentID, pubkey, lamport, postOpTypeDelete)

	var (
		author           string
		postID           string
		currentLamport   int64
		currentAuthorKey string
		currentOpID      string
		deletedLamport   int64
	)
	err := a.db.QueryRow(`
		SELECT pubkey, post_id, lamport, current_author_pubkey, current_op_id, deleted_at_lamport
		FROM comments
		WHERE id = ?;
	`, commentID).Scan(&author, &postID, &currentLamport, &currentAuthorKey, &currentOpID, &deletedLamport)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errors.New("comment not found")
	}
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(author) != pubkey {
		return "", errors.New("only comment author can delete this comment")
	}
	if strings.TrimSpace(currentAuthorKey) == "" {
		currentAuthorKey = author
	}
	if strings.TrimSpace(currentOpID) == "" {
		currentOpID = commentID
	}

	incomingVersion := LamportVersion{Lamport: lamport, Author: pubkey, OpID: opID}
	currentVersion := LamportVersion{Lamport: currentLamport, Author: currentAuthorKey, OpID: currentOpID}
	if compareLamportVersion(incomingVersion, currentVersion) <= 0 {
		return strings.TrimSpace(postID), nil
	}
	if deletedLamport > 0 {
		tombstoneVersion := LamportVersion{Lamport: deletedLamport, Author: currentAuthorKey, OpID: currentOpID}
		if compareLamportVersion(incomingVersion, tombstoneVersion) <= 0 {
			return strings.TrimSpace(postID), nil
		}
	}

	_, err = a.db.Exec(`
		UPDATE comments
		SET body = '',
		    attachments_json = '[]',
		    deleted_at_lamport = ?,
		    deleted_at = ?,
		    deleted_by = ?,
		    timestamp = ?,
		    lamport = ?,
		    current_author_pubkey = ?,
		    current_op_id = ?
		WHERE id = ?;
	`, lamport, deletedAt, pubkey, deletedAt, lamport, pubkey, opID, commentID)
	if err != nil {
		return "", err
	}
	if err = a.appendEntityOperation(
		entityTypeComment,
		commentID,
		postOpTypeDelete,
		opID,
		pubkey,
		lamport,
		deletedAt,
		lamportSchemaV2,
		authScopeUser,
		map[string]any{"postId": strings.TrimSpace(postID), "deletedAtLamport": lamport},
	); err != nil {
		return "", err
	}

	return strings.TrimSpace(postID), nil
}

func (a *App) UpvoteComment(commentID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return err
	}

	return a.applyCommentUpvote(identity.PublicKey, commentID, "", generateOperationID(commentID, identity.PublicKey, time.Now().UnixNano()))
}

func (a *App) DownvoteComment(commentID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return err
	}

	return a.applyCommentDownvote(identity.PublicKey, commentID, "", generateOperationID(commentID, identity.PublicKey, time.Now().UnixNano()))
}

func (a *App) StoreCommentImageDataURL(dataURL string) (CommentAttachment, error) {
	if a.db == nil {
		return CommentAttachment{}, errors.New("database not initialized")
	}

	dataURL = strings.TrimSpace(dataURL)
	if dataURL == "" {
		return CommentAttachment{}, errors.New("image payload is required")
	}
	if !strings.HasPrefix(dataURL, "data:") {
		return CommentAttachment{}, errors.New("invalid data URL")
	}
	commaIndex := strings.Index(dataURL, ",")
	if commaIndex <= 0 {
		return CommentAttachment{}, errors.New("invalid data URL")
	}

	header := dataURL[5:commaIndex]
	body := dataURL[commaIndex+1:]
	if !strings.Contains(strings.ToLower(header), ";base64") {
		return CommentAttachment{}, errors.New("comment image must be base64 encoded")
	}
	hintMIME := strings.TrimSpace(strings.SplitN(header, ";", 2)[0])
	raw, err := base64.StdEncoding.DecodeString(body)
	if err != nil || len(raw) == 0 {
		return CommentAttachment{}, errors.New("invalid image payload")
	}

	processedBytes, processedMime, width, height, _, _, _, _, prepErr := prepareImageAssets(raw, hintMIME)
	if prepErr != nil {
		return CommentAttachment{}, prepErr
	}

	contentCID := buildBinaryCID(processedBytes)
	if err = a.upsertMediaBlobRaw(contentCID, processedMime, processedBytes, width, height, false); err != nil {
		return CommentAttachment{}, err
	}

	return CommentAttachment{
		Kind:      "media_cid",
		Ref:       contentCID,
		Mime:      processedMime,
		Width:     width,
		Height:    height,
		SizeBytes: int64(len(processedBytes)),
	}, nil
}

func (a *App) insertComment(comment Comment) (Comment, error) {
	if a.db == nil {
		return Comment{}, errors.New("database not initialized")
	}

	comment.ID = strings.TrimSpace(comment.ID)
	comment.PostID = strings.TrimSpace(comment.PostID)
	comment.ParentID = strings.TrimSpace(comment.ParentID)
	comment.Pubkey = strings.TrimSpace(comment.Pubkey)
	comment.Body = strings.TrimSpace(comment.Body)

	if comment.ID == "" || comment.PostID == "" || comment.Pubkey == "" {
		return Comment{}, errors.New("invalid comment")
	}
	if comment.DeletedAt <= 0 && comment.Body == "" && len(comment.Attachments) == 0 {
		return Comment{}, errors.New("invalid comment")
	}
	if comment.Timestamp == 0 {
		comment.Timestamp = time.Now().Unix()
	}
	if comment.Lamport <= 0 {
		comment.Lamport = comment.Timestamp
	}
	comment.OpID = resolveOperationID(comment.OpID, comment.ID, comment.Pubkey, comment.Lamport, postOpTypeCreate)
	comment.DeletedBy = strings.TrimSpace(comment.DeletedBy)
	if comment.DeletedAt < 0 {
		comment.DeletedAt = 0
	}

	var (
		existingPubkey       string
		existingLamport      int64
		existingAuthorPubkey string
		existingOpID         string
		existingDeletedL     int64
	)
	err := a.db.QueryRow(`
		SELECT pubkey, lamport, current_author_pubkey, current_op_id, deleted_at_lamport
		FROM comments
		WHERE id = ?;
	`, comment.ID).Scan(&existingPubkey, &existingLamport, &existingAuthorPubkey, &existingOpID, &existingDeletedL)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Comment{}, err
	}
	appliedOpType := postOpTypeCreate
	if err == nil {
		appliedOpType = postOpTypeUpdate
		existingPubkey = strings.TrimSpace(existingPubkey)
		if existingPubkey != "" && existingPubkey != comment.Pubkey {
			return Comment{}, errors.New("only comment author can mutate this comment")
		}
		if strings.TrimSpace(existingAuthorPubkey) == "" {
			existingAuthorPubkey = existingPubkey
		}
		if strings.TrimSpace(existingOpID) == "" {
			existingOpID = comment.ID
		}

		incomingVersion := LamportVersion{Lamport: comment.Lamport, Author: comment.Pubkey, OpID: comment.OpID}
		currentVersion := LamportVersion{Lamport: existingLamport, Author: existingAuthorPubkey, OpID: existingOpID}
		if compareLamportVersion(incomingVersion, currentVersion) <= 0 {
			return comment, nil
		}
		if existingDeletedL > 0 {
			tombstoneVersion := LamportVersion{Lamport: existingDeletedL, Author: existingAuthorPubkey, OpID: existingOpID}
			if compareLamportVersion(incomingVersion, tombstoneVersion) <= 0 {
				return comment, nil
			}
		}
	}

	comment.Attachments = normalizeCommentAttachments(comment.Attachments)
	attachmentsJSON, err := encodeCommentAttachmentsJSON(comment.Attachments)
	if err != nil {
		return Comment{}, err
	}
	mediaCIDs := mediaCIDsFromAttachments(comment.Attachments)

	tx, err := a.db.Begin()
	if err != nil {
		return Comment{}, err
	}

	_, err = tx.Exec(`
		INSERT INTO comments (
			id, post_id, parent_id, pubkey, current_author_pubkey, current_op_id,
			body, attachments_json, score, timestamp, lamport, deleted_at_lamport, deleted_at, deleted_by
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			post_id = excluded.post_id,
			parent_id = excluded.parent_id,
			pubkey = excluded.pubkey,
			current_author_pubkey = excluded.current_author_pubkey,
			current_op_id = excluded.current_op_id,
			body = excluded.body,
			attachments_json = excluded.attachments_json,
			score = excluded.score,
			timestamp = excluded.timestamp,
			lamport = excluded.lamport,
			deleted_at_lamport = 0,
			deleted_at = 0,
			deleted_by = '';
	`, comment.ID, comment.PostID, comment.ParentID, comment.Pubkey, comment.Pubkey, comment.OpID, comment.Body, attachmentsJSON, comment.Score, comment.Timestamp, comment.Lamport, comment.DeletedAt, comment.DeletedBy)
	if err != nil {
		_ = tx.Rollback()
		return Comment{}, err
	}

	if _, err = tx.Exec(`DELETE FROM comment_media_refs WHERE comment_id = ?;`, comment.ID); err != nil {
		_ = tx.Rollback()
		return Comment{}, err
	}
	for _, cid := range mediaCIDs {
		if _, err = tx.Exec(`
			INSERT INTO comment_media_refs (comment_id, content_cid)
			VALUES (?, ?)
			ON CONFLICT(comment_id, content_cid) DO NOTHING;
		`, comment.ID, cid); err != nil {
			_ = tx.Rollback()
			return Comment{}, err
		}
	}
	if err = a.appendEntityOperationTx(
		tx,
		entityTypeComment,
		comment.ID,
		appliedOpType,
		comment.OpID,
		comment.Pubkey,
		comment.Lamport,
		comment.Timestamp,
		lamportSchemaV2,
		authScopeUser,
		map[string]any{"postId": comment.PostID, "deleted": false},
	); err != nil {
		_ = tx.Rollback()
		return Comment{}, err
	}

	if err = tx.Commit(); err != nil {
		return Comment{}, err
	}

	return comment, nil
}

func (a *App) upsertCommentTombstone(commentID string, postID string, pubkey string, deletedAt int64, lamport int64, opID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	commentID = strings.TrimSpace(commentID)
	postID = strings.TrimSpace(postID)
	pubkey = strings.TrimSpace(pubkey)
	if commentID == "" || postID == "" || pubkey == "" {
		return errors.New("invalid comment tombstone")
	}
	if deletedAt <= 0 {
		deletedAt = time.Now().Unix()
	}
	if lamport <= 0 {
		lamport = deletedAt
	}
	opID = resolveOperationID(opID, commentID, pubkey, lamport, postOpTypeDelete)

	var (
		existingPubkey       string
		existingLamport      int64
		existingAuthorPubkey string
		existingOpID         string
		existingDeletedL     int64
	)
	err := a.db.QueryRow(`
		SELECT pubkey, lamport, current_author_pubkey, current_op_id, deleted_at_lamport
		FROM comments
		WHERE id = ?;
	`, commentID).Scan(&existingPubkey, &existingLamport, &existingAuthorPubkey, &existingOpID, &existingDeletedL)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		existingPubkey = strings.TrimSpace(existingPubkey)
		if existingPubkey != "" && existingPubkey != pubkey {
			return errors.New("unauthorized comment operation")
		}
		if strings.TrimSpace(existingAuthorPubkey) == "" {
			existingAuthorPubkey = existingPubkey
		}
		if strings.TrimSpace(existingOpID) == "" {
			existingOpID = commentID
		}
		incomingVersion := LamportVersion{Lamport: lamport, Author: pubkey, OpID: opID}
		currentVersion := LamportVersion{Lamport: existingLamport, Author: existingAuthorPubkey, OpID: existingOpID}
		if compareLamportVersion(incomingVersion, currentVersion) <= 0 {
			return nil
		}
		if existingDeletedL > 0 {
			tombstoneVersion := LamportVersion{Lamport: existingDeletedL, Author: existingAuthorPubkey, OpID: existingOpID}
			if compareLamportVersion(incomingVersion, tombstoneVersion) <= 0 {
				return nil
			}
		}
	}

	result, err := a.db.Exec(`
		INSERT INTO comments (
			id, post_id, parent_id, pubkey, current_author_pubkey, current_op_id,
			body, attachments_json, score, timestamp, lamport, deleted_at_lamport, deleted_at, deleted_by
		)
		VALUES (?, ?, '', ?, ?, ?, '', '[]', 0, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			post_id = excluded.post_id,
			pubkey = excluded.pubkey,
			current_author_pubkey = excluded.current_author_pubkey,
			current_op_id = excluded.current_op_id,
			body = '',
			attachments_json = '[]',
			timestamp = excluded.timestamp,
			lamport = excluded.lamport,
			deleted_at_lamport = excluded.deleted_at_lamport,
			deleted_at = excluded.deleted_at,
			deleted_by = excluded.deleted_by;
	`, commentID, postID, pubkey, pubkey, opID, deletedAt, lamport, lamport, deletedAt, pubkey)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return nil
	}
	return a.appendEntityOperation(
		entityTypeComment,
		commentID,
		postOpTypeDelete,
		opID,
		pubkey,
		lamport,
		deletedAt,
		lamportSchemaV2,
		authScopeUser,
		map[string]any{"postId": postID, "deletedAtLamport": lamport, "source": "digest_or_delete"},
	)
}

func (a *App) applyCommentUpvote(voterPubkey string, commentID string, postID string, opID string) error {
	current, err := a.getCommentVoteState(voterPubkey, commentID)
	if err != nil {
		return err
	}
	target := voteStateUp
	if current == voteStateUp {
		target = voteStateNone
	}
	return a.applyCommentVoteState(voterPubkey, commentID, postID, target, opID)
}

func (a *App) applyCommentDownvote(voterPubkey string, commentID string, postID string, opID string) error {
	current, err := a.getCommentVoteState(voterPubkey, commentID)
	if err != nil {
		return err
	}
	target := voteStateDown
	if current == voteStateDown {
		target = voteStateNone
	}
	return a.applyCommentVoteState(voterPubkey, commentID, postID, target, opID)
}

func (a *App) currentCommentVoteStateTx(tx *sql.Tx, voterPubkey string, commentID string) (string, error) {
	var exists int
	err := tx.QueryRow(`SELECT 1 FROM comment_votes WHERE comment_id = ? AND voter_pubkey = ? LIMIT 1;`, commentID, voterPubkey).Scan(&exists)
	if err == nil {
		return voteStateUp, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return voteStateNone, err
	}
	err = tx.QueryRow(`SELECT 1 FROM comment_downvotes WHERE comment_id = ? AND voter_pubkey = ? LIMIT 1;`, commentID, voterPubkey).Scan(&exists)
	if err == nil {
		return voteStateDown, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return voteStateNone, err
	}
	return voteStateNone, nil
}

func (a *App) getCommentVoteState(voterPubkey string, commentID string) (string, error) {
	voterPubkey = strings.TrimSpace(voterPubkey)
	commentID = strings.TrimSpace(commentID)
	if voterPubkey == "" || commentID == "" {
		return voteStateNone, errors.New("voter pubkey and comment id are required")
	}
	tx, err := a.db.Begin()
	if err != nil {
		return voteStateNone, err
	}
	defer func() { _ = tx.Rollback() }()
	state, err := a.currentCommentVoteStateTx(tx, voterPubkey, commentID)
	if err != nil {
		return voteStateNone, err
	}
	if err = tx.Commit(); err != nil {
		return voteStateNone, err
	}
	return state, nil
}

func (a *App) applyCommentVoteState(voterPubkey string, commentID string, postID string, targetState string, opID string) error {
	targetState = normalizeVoteState(targetState)
	voterPubkey = strings.TrimSpace(voterPubkey)
	commentID = strings.TrimSpace(commentID)
	postID = strings.TrimSpace(postID)
	opID = strings.TrimSpace(opID)
	if voterPubkey == "" || commentID == "" {
		return errors.New("voter pubkey and comment id are required")
	}
	if postID == "" {
		if err := a.db.QueryRow(`SELECT post_id FROM comments WHERE id = ?;`, commentID).Scan(&postID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("comment not found")
			}
			return err
		}
	}
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if opID != "" {
		result, err := tx.Exec(`INSERT INTO vote_ops (op_id, created_at) VALUES (?, ?) ON CONFLICT(op_id) DO NOTHING;`, opID, time.Now().Unix())
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return tx.Commit()
		}
	}

	current, err := a.currentCommentVoteStateTx(tx, voterPubkey, commentID)
	if err != nil {
		return err
	}
	if current == targetState {
		return tx.Commit()
	}
	if _, err = tx.Exec(`DELETE FROM comment_votes WHERE comment_id = ? AND voter_pubkey = ?;`, commentID, voterPubkey); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM comment_downvotes WHERE comment_id = ? AND voter_pubkey = ?;`, commentID, voterPubkey); err != nil {
		return err
	}
	switch targetState {
	case voteStateUp:
		if _, err = tx.Exec(`INSERT INTO comment_votes (comment_id, voter_pubkey, timestamp) VALUES (?, ?, ?);`, commentID, voterPubkey, time.Now().Unix()); err != nil {
			return err
		}
	case voteStateDown:
		if _, err = tx.Exec(`INSERT INTO comment_downvotes (comment_id, voter_pubkey, timestamp) VALUES (?, ?, ?);`, commentID, voterPubkey, time.Now().Unix()); err != nil {
			return err
		}
	}
	delta := voteDelta(current, targetState)
	if delta != 0 {
		result, err := tx.Exec(`UPDATE comments SET score = score + ? WHERE id = ? AND post_id = ?;`, delta, commentID, postID)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.New("comment not found")
		}
	}
	return tx.Commit()
}

func normalizeCommentAttachments(input []CommentAttachment) []CommentAttachment {
	if len(input) == 0 {
		return []CommentAttachment{}
	}

	result := make([]CommentAttachment, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, item := range input {
		kind := strings.ToLower(strings.TrimSpace(item.Kind))
		ref := strings.TrimSpace(item.Ref)
		if ref == "" {
			continue
		}
		if kind != "media_cid" && kind != "external_url" {
			continue
		}
		if kind == "external_url" {
			u, err := url.Parse(ref)
			if err != nil {
				continue
			}
			proto := strings.ToLower(strings.TrimSpace(u.Scheme))
			if proto != "http" && proto != "https" {
				continue
			}
			ref = u.String()
		}

		key := kind + "|" + ref
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, CommentAttachment{
			Kind:      kind,
			Ref:       ref,
			Mime:      strings.TrimSpace(item.Mime),
			Width:     item.Width,
			Height:    item.Height,
			SizeBytes: item.SizeBytes,
		})
	}

	if len(result) > 8 {
		result = result[:8]
	}

	return result
}

func encodeCommentAttachmentsJSON(items []CommentAttachment) (string, error) {
	normalized := normalizeCommentAttachments(items)
	if len(normalized) == 0 {
		return "[]", nil
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func decodeCommentAttachmentsJSON(raw string) []CommentAttachment {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []CommentAttachment{}
	}

	var items []CommentAttachment
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []CommentAttachment{}
	}
	return normalizeCommentAttachments(items)
}

func mediaCIDsFromAttachments(items []CommentAttachment) []string {
	if len(items) == 0 {
		return []string{}
	}

	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.Kind)) != "media_cid" {
			continue
		}
		cid := strings.TrimSpace(item.Ref)
		if cid == "" {
			continue
		}
		if _, exists := seen[cid]; exists {
			continue
		}
		seen[cid] = struct{}{}
		result = append(result, cid)
	}
	return result
}

func (a *App) UpdateLocalComment(pubkey string, commentID string, body string) (Comment, error) {
	if a.db == nil {
		return Comment{}, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	commentID = strings.TrimSpace(commentID)
	if pubkey == "" || commentID == "" {
		return Comment{}, errors.New("pubkey and comment id are required")
	}

	body = strings.TrimSpace(body)
	if body == "" {
		// If body is empty, we might want to check attachments.
		// For now, let's assume we update the text body.
		// But insertComment validates body+attachments non-empty.
		// If user clears body, attachments must persist.
	}

	var (
		currentBody        string
		currentAuthor      string
		currentPostID      string
		currentParentID    string
		currentAttachments string
	)

	err := a.db.QueryRow(`
		SELECT body, pubkey, post_id, parent_id, attachments_json
		FROM comments
		WHERE id = ?;
	`, commentID).Scan(&currentBody, &currentAuthor, &currentPostID, &currentParentID, &currentAttachments)
	if errors.Is(err, sql.ErrNoRows) {
		return Comment{}, errors.New("comment not found")
	}
	if err != nil {
		return Comment{}, err
	}

	if currentAuthor != pubkey {
		return Comment{}, errors.New("only author can update comment")
	}

	if body == "" {
		// If body passed is empty, and user meant to keep existing?
		// Or user meant to clear it?
		// insertComment checks: if body == "" && len(attachments) == 0 -> invalid.
		// If we use currentAttachments, it might be valid.
		// Let's assume passed body is the NEW body.
	}

	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return Comment{}, err
	}

	attachments := decodeCommentAttachmentsJSON(currentAttachments)

	updatedComment := Comment{
		ID:          commentID,
		PostID:      currentPostID,
		ParentID:    currentParentID,
		Pubkey:      pubkey,
		Body:        body,
		Attachments: attachments, // Preserve attachments
		Timestamp:   now,
		Lamport:     lamport,
	}

	return a.insertComment(updatedComment)
}

