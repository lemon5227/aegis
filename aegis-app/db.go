package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"strconv"
	"strings"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	xdraw "golang.org/x/image/draw"
	_ "modernc.org/sqlite"
)

func compareLamportVersion(left LamportVersion, right LamportVersion) int {
	if left.Lamport > right.Lamport {
		return 1
	}
	if left.Lamport < right.Lamport {
		return -1
	}

	left.Author = strings.TrimSpace(left.Author)
	right.Author = strings.TrimSpace(right.Author)
	if left.Author > right.Author {
		return 1
	}
	if left.Author < right.Author {
		return -1
	}

	left.OpID = strings.TrimSpace(left.OpID)
	right.OpID = strings.TrimSpace(right.OpID)
	if left.OpID > right.OpID {
		return 1
	}
	if left.OpID < right.OpID {
		return -1
	}

	return 0
}

func normalizeOperationType(opType string, fallback string) string {
	normalized := strings.ToUpper(strings.TrimSpace(opType))
	switch normalized {
	case postOpTypeCreate, postOpTypeUpdate, postOpTypeDelete:
		return normalized
	}

	fallback = strings.ToUpper(strings.TrimSpace(fallback))
	switch fallback {
	case postOpTypeCreate, postOpTypeUpdate, postOpTypeDelete:
		return fallback
	default:
		return postOpTypeCreate
	}
}

func normalizeVoteState(state string) string {
	state = strings.ToUpper(strings.TrimSpace(state))
	switch state {
	case voteStateUp, voteStateDown, voteStateNone:
		return state
	default:
		return voteStateNone
	}
}

func generateOperationID(entityID string, author string, lamport int64) string {
	entityID = strings.TrimSpace(entityID)
	author = strings.TrimSpace(author)
	if lamport < 0 {
		lamport = 0
	}

	nonce := make([]byte, defaultOpNonceBytes)
	if _, err := rand.Read(nonce); err != nil {
		nonceSeed := fmt.Sprintf("%s|%s|%d|%d", entityID, author, lamport, time.Now().UnixNano())
		hash := sha256.Sum256([]byte(nonceSeed))
		nonce = hash[:defaultOpNonceBytes]
	}

	return fmt.Sprintf("%s:%s:%d:%s", entityID, author, lamport, hex.EncodeToString(nonce))
}

func fallbackOperationID(entityID string, author string, lamport int64, opType string) string {
	if lamport < 0 {
		lamport = 0
	}
	return fmt.Sprintf("%s:%s:%d:%s", strings.TrimSpace(entityID), strings.TrimSpace(author), lamport, strings.ToLower(strings.TrimSpace(opType)))
}

func resolveOperationID(opID string, entityID string, author string, lamport int64, opType string) string {
	opID = strings.TrimSpace(opID)
	if opID != "" {
		return opID
	}
	return fallbackOperationID(entityID, author, lamport, opType)
}

func resolveCurrentVersion(lamport int64, author string, opID string, entityID string) LamportVersion {
	author = strings.TrimSpace(author)
	if author == "" {
		author = strings.TrimSpace(entityID)
	}
	opID = strings.TrimSpace(opID)
	if opID == "" {
		opID = strings.TrimSpace(entityID)
	}
	return LamportVersion{Lamport: lamport, Author: author, OpID: opID}
}

func normalizeAuthScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		return authScopeUser
	}
	return scope
}

func isDevModeEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("AEGIS_DEV_MODE")))
	switch value {
	case "1", "true", "yes", "on", "dev":
		return true
	default:
		return false
	}
}

func (a *App) IsDevMode() bool {
	return isDevModeEnabled()
}

func marshalOperationPayload(payload any) string {
	if payload == nil {
		return "{}"
	}
	encoded, err := json.Marshal(payload)
	if err != nil || len(encoded) == 0 {
		return "{}"
	}
	return string(encoded)
}

func (a *App) appendEntityOperationTx(tx *sql.Tx, entityType string, entityID string, opType string, opID string, authorPubkey string, lamport int64, timestamp int64, schemaVersion int, authScope string, payload any) error {
	if tx == nil {
		return errors.New("nil transaction")
	}
	entityType = strings.TrimSpace(entityType)
	entityID = strings.TrimSpace(entityID)
	opType = normalizeOperationType(opType, postOpTypeCreate)
	opID = strings.TrimSpace(opID)
	authorPubkey = strings.TrimSpace(authorPubkey)
	if entityType == "" || entityID == "" || opID == "" || authorPubkey == "" {
		return errors.New("invalid entity operation")
	}
	if lamport < 0 {
		lamport = 0
	}
	if timestamp <= 0 {
		timestamp = time.Now().Unix()
	}
	if schemaVersion <= 0 {
		schemaVersion = lamportSchemaV2
	}

	_, err := tx.Exec(`
		INSERT INTO entity_ops (
			op_id, entity_type, entity_id, op_type, author_pubkey, lamport, timestamp, schema_version, auth_scope, payload_json
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(op_id) DO NOTHING;
	`, opID, entityType, entityID, opType, authorPubkey, lamport, timestamp, schemaVersion, normalizeAuthScope(authScope), marshalOperationPayload(payload))
	return err
}

func (a *App) appendEntityOperation(entityType string, entityID string, opType string, opID string, authorPubkey string, lamport int64, timestamp int64, schemaVersion int, authScope string, payload any) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	entityType = strings.TrimSpace(entityType)
	entityID = strings.TrimSpace(entityID)
	opType = normalizeOperationType(opType, postOpTypeCreate)
	opID = strings.TrimSpace(opID)
	authorPubkey = strings.TrimSpace(authorPubkey)
	if entityType == "" || entityID == "" || opID == "" || authorPubkey == "" {
		return errors.New("invalid entity operation")
	}
	if lamport < 0 {
		lamport = 0
	}
	if timestamp <= 0 {
		timestamp = time.Now().Unix()
	}
	if schemaVersion <= 0 {
		schemaVersion = lamportSchemaV2
	}

	_, err := a.db.Exec(`
		INSERT INTO entity_ops (
			op_id, entity_type, entity_id, op_type, author_pubkey, lamport, timestamp, schema_version, auth_scope, payload_json
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(op_id) DO NOTHING;
	`, opID, entityType, entityID, opType, authorPubkey, lamport, timestamp, schemaVersion, normalizeAuthScope(authScope), marshalOperationPayload(payload))
	return err
}

func (a *App) ListEntityOps(entityType string, entityID string, limit int) ([]EntityOpRecord, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}
	if !isDevModeEnabled() {
		return nil, errors.New("timeline is available in dev mode only")
	}

	entityType = strings.TrimSpace(entityType)
	entityID = strings.TrimSpace(entityID)
	if limit <= 0 || limit > 5000 {
		limit = 200
	}

	query := `
		SELECT op_id, entity_type, entity_id, op_type, author_pubkey, lamport, timestamp, schema_version, auth_scope, payload_json
		FROM entity_ops
	`
	args := make([]any, 0, 3)
	if entityType != "" || entityID != "" {
		conditions := make([]string, 0, 2)
		if entityType != "" {
			conditions = append(conditions, "entity_type = ?")
			args = append(args, entityType)
		}
		if entityID != "" {
			conditions = append(conditions, "entity_id = ?")
			args = append(args, entityID)
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY lamport DESC, author_pubkey DESC, op_id DESC LIMIT ?;"
	args = append(args, limit)

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]EntityOpRecord, 0, limit)
	for rows.Next() {
		var item EntityOpRecord
		if err = rows.Scan(
			&item.OpID,
			&item.EntityType,
			&item.EntityID,
			&item.OpType,
			&item.AuthorPubkey,
			&item.Lamport,
			&item.Timestamp,
			&item.SchemaVersion,
			&item.AuthScope,
			&item.PayloadJSON,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	return result, rows.Err()
}

func (a *App) RunTombstoneGC(retentionDays int, requiredStablePasses int, batchSize int) (TombstoneGCResult, error) {
	if a.db == nil {
		return TombstoneGCResult{}, errors.New("database not initialized")
	}
	if retentionDays <= 0 {
		retentionDays = 30
	}
	if requiredStablePasses <= 0 {
		requiredStablePasses = 2
	}
	if batchSize <= 0 || batchSize > 2000 {
		batchSize = 200
	}

	now := time.Now().Unix()
	cutoff := now - int64(retentionDays)*24*3600

	a.dbMu.Lock()
	defer a.dbMu.Unlock()

	tx, err := a.db.Begin()
	if err != nil {
		return TombstoneGCResult{}, err
	}

	result := TombstoneGCResult{}

	processCandidate := func(entityType string, entityID string, deletedAtLamport int64) (bool, error) {
		var markLamport int64
		var stablePasses int
		err := tx.QueryRow(`
			SELECT deleted_at_lamport, stable_passes
			FROM tombstone_gc_marks
			WHERE entity_type = ? AND entity_id = ?;
		`, entityType, entityID).Scan(&markLamport, &stablePasses)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, err
		}

		if errors.Is(err, sql.ErrNoRows) || markLamport != deletedAtLamport {
			_, err = tx.Exec(`
				INSERT INTO tombstone_gc_marks (entity_type, entity_id, deleted_at_lamport, stable_passes, first_seen_at, last_seen_at)
				VALUES (?, ?, ?, 1, ?, ?)
				ON CONFLICT(entity_type, entity_id) DO UPDATE SET
					deleted_at_lamport = excluded.deleted_at_lamport,
					stable_passes = 1,
					first_seen_at = excluded.first_seen_at,
					last_seen_at = excluded.last_seen_at;
			`, entityType, entityID, deletedAtLamport, now, now)
			if err != nil {
				return false, err
			}
			return false, nil
		}

		stablePasses++
		_, err = tx.Exec(`
			UPDATE tombstone_gc_marks
			SET stable_passes = ?, last_seen_at = ?
			WHERE entity_type = ? AND entity_id = ?;
		`, stablePasses, now, entityType, entityID)
		if err != nil {
			return false, err
		}

		return stablePasses >= requiredStablePasses, nil
	}

	postRows, err := tx.Query(`
		SELECT id, deleted_at_lamport
		FROM messages
		WHERE visibility = 'deleted' AND deleted_at_ts > 0 AND deleted_at_ts <= ?
		ORDER BY deleted_at_ts ASC
		LIMIT ?;
	`, cutoff, batchSize)
	if err != nil {
		_ = tx.Rollback()
		return TombstoneGCResult{}, err
	}
	for postRows.Next() {
		var postID string
		var deletedAtLamport int64
		if err = postRows.Scan(&postID, &deletedAtLamport); err != nil {
			postRows.Close()
			_ = tx.Rollback()
			return TombstoneGCResult{}, err
		}
		result.ScannedPosts++
		ready, decideErr := processCandidate(entityTypePost, strings.TrimSpace(postID), deletedAtLamport)
		if decideErr != nil {
			postRows.Close()
			_ = tx.Rollback()
			return TombstoneGCResult{}, decideErr
		}
		if !ready {
			continue
		}
		if _, err = tx.Exec(`DELETE FROM messages WHERE id = ? AND visibility = 'deleted';`, postID); err != nil {
			postRows.Close()
			_ = tx.Rollback()
			return TombstoneGCResult{}, err
		}
		if _, err = tx.Exec(`DELETE FROM tombstone_gc_marks WHERE entity_type = ? AND entity_id = ?;`, entityTypePost, postID); err != nil {
			postRows.Close()
			_ = tx.Rollback()
			return TombstoneGCResult{}, err
		}
		result.DeletedPosts++
	}
	if err = postRows.Err(); err != nil {
		postRows.Close()
		_ = tx.Rollback()
		return TombstoneGCResult{}, err
	}
	postRows.Close()

	commentRows, err := tx.Query(`
		SELECT id, deleted_at_lamport
		FROM comments
		WHERE deleted_at > 0 AND deleted_at <= ?
		ORDER BY deleted_at ASC
		LIMIT ?;
	`, cutoff, batchSize)
	if err != nil {
		_ = tx.Rollback()
		return TombstoneGCResult{}, err
	}
	for commentRows.Next() {
		var commentID string
		var deletedAtLamport int64
		if err = commentRows.Scan(&commentID, &deletedAtLamport); err != nil {
			commentRows.Close()
			_ = tx.Rollback()
			return TombstoneGCResult{}, err
		}
		result.ScannedComments++
		ready, decideErr := processCandidate(entityTypeComment, strings.TrimSpace(commentID), deletedAtLamport)
		if decideErr != nil {
			commentRows.Close()
			_ = tx.Rollback()
			return TombstoneGCResult{}, decideErr
		}
		if !ready {
			continue
		}
		if _, err = tx.Exec(`DELETE FROM comment_media_refs WHERE comment_id = ?;`, commentID); err != nil {
			commentRows.Close()
			_ = tx.Rollback()
			return TombstoneGCResult{}, err
		}
		if _, err = tx.Exec(`DELETE FROM comments WHERE id = ? AND deleted_at > 0;`, commentID); err != nil {
			commentRows.Close()
			_ = tx.Rollback()
			return TombstoneGCResult{}, err
		}
		if _, err = tx.Exec(`DELETE FROM tombstone_gc_marks WHERE entity_type = ? AND entity_id = ?;`, entityTypeComment, commentID); err != nil {
			commentRows.Close()
			_ = tx.Rollback()
			return TombstoneGCResult{}, err
		}
		result.DeletedComments++
	}
	if err = commentRows.Err(); err != nil {
		commentRows.Close()
		_ = tx.Rollback()
		return TombstoneGCResult{}, err
	}
	commentRows.Close()

	if err = tx.Commit(); err != nil {
		return TombstoneGCResult{}, err
	}

	return result, nil
}

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

func (a *App) GetPostBodyByCID(contentCID string) (PostBodyBlob, error) {
	if a.db == nil {
		return PostBodyBlob{}, errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return PostBodyBlob{}, errors.New("content cid is required")
	}

	body, err := a.getContentBlobLocal(contentCID)
	if err == nil {
		a.noteBlobCacheHit()
		return body, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return PostBodyBlob{}, err
	}
	a.noteBlobCacheMiss()

	status := a.GetP2PStatus()
	if !status.Started {
		return PostBodyBlob{}, errors.New("content not found")
	}

	maxAttempts := resolveFetchRetryAttempts()
	var fetchErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fetchErr = a.fetchContentBlobFromNetwork(contentCID, 4*time.Second)
		if fetchErr == nil {
			break
		}
		if attempt < maxAttempts {
			time.Sleep(150 * time.Millisecond)
		}
	}
	if fetchErr != nil {
		return PostBodyBlob{}, fetchErr
	}

	body, err = a.getContentBlobLocal(contentCID)
	if errors.Is(err, sql.ErrNoRows) {
		return PostBodyBlob{}, errors.New("content not found")
	}
	if err != nil {
		return PostBodyBlob{}, err
	}

	return body, nil
}

func (a *App) GetPostBodyByID(postID string) (PostBodyBlob, error) {
	if a.db == nil {
		return PostBodyBlob{}, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return PostBodyBlob{}, errors.New("post id is required")
	}

	var contentCID string
	err := a.db.QueryRow(`SELECT content_cid FROM messages WHERE id = ?;`, postID).Scan(&contentCID)
	if errors.Is(err, sql.ErrNoRows) {
		return PostBodyBlob{}, errors.New("post not found")
	}
	if err != nil {
		return PostBodyBlob{}, err
	}

	return a.GetPostBodyByCID(contentCID)
}

func (a *App) GetMediaByCID(contentCID string) (MediaBlob, error) {
	if a.db == nil {
		return MediaBlob{}, errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return MediaBlob{}, errors.New("media cid is required")
	}

	media, err := a.getMediaBlobLocal(contentCID)
	if err == nil {
		a.noteBlobCacheHit()
		return media, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return MediaBlob{}, err
	}
	a.noteBlobCacheMiss()

	status := a.GetP2PStatus()
	if !status.Started {
		return MediaBlob{}, errors.New("media not found")
	}

	maxAttempts := resolveFetchRetryAttempts()
	var fetchErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fetchErr = a.fetchMediaBlobFromNetwork(contentCID, 5*time.Second)
		if fetchErr == nil {
			break
		}
		if attempt < maxAttempts {
			time.Sleep(150 * time.Millisecond)
		}
	}
	if fetchErr != nil {
		return MediaBlob{}, fetchErr
	}

	media, err = a.getMediaBlobLocal(contentCID)
	if errors.Is(err, sql.ErrNoRows) {
		return MediaBlob{}, errors.New("media not found")
	}
	if err != nil {
		return MediaBlob{}, err
	}

	return media, nil
}

func (a *App) GetPostMediaByID(postID string) (MediaBlob, error) {
	if a.db == nil {
		return MediaBlob{}, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return MediaBlob{}, errors.New("post id is required")
	}

	var imageCID string
	err := a.db.QueryRow(`SELECT image_cid FROM messages WHERE id = ?;`, postID).Scan(&imageCID)
	if errors.Is(err, sql.ErrNoRows) {
		return MediaBlob{}, errors.New("post not found")
	}
	if err != nil {
		return MediaBlob{}, err
	}
	imageCID = strings.TrimSpace(imageCID)
	if imageCID == "" {
		return MediaBlob{}, errors.New("post has no image")
	}

	return a.GetMediaByCID(imageCID)
}

func (a *App) getMediaBlobRawLocal(contentCID string) (MediaBlob, []byte, error) {
	var media MediaBlob
	var raw []byte
	var isThumb int
	err := a.db.QueryRow(`
		SELECT content_cid, data, mime, size_bytes, width, height, is_thumbnail
		FROM media_blobs
		WHERE content_cid = ?;
	`, contentCID).Scan(&media.ContentCID, &raw, &media.Mime, &media.SizeBytes, &media.Width, &media.Height, &isThumb)
	if err != nil {
		return MediaBlob{}, nil, err
	}

	media.IsThumbnail = isThumb == 1
	media.DataBase64 = base64.StdEncoding.EncodeToString(raw)

	if _, err = a.db.Exec(`UPDATE media_blobs SET last_accessed_at = ? WHERE content_cid = ?;`, time.Now().Unix(), contentCID); err != nil {
		return MediaBlob{}, nil, err
	}

	return media, raw, nil
}

func (a *App) getMediaBlobLocal(contentCID string) (MediaBlob, error) {
	media, _, err := a.getMediaBlobRawLocal(contentCID)
	if err != nil {
		return MediaBlob{}, err
	}
	return media, nil
}

func (a *App) upsertMediaBlobRaw(contentCID string, mime string, data []byte, width int, height int, isThumbnail bool) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	mime = strings.TrimSpace(mime)
	if contentCID == "" || len(data) == 0 {
		return errors.New("invalid media blob")
	}

	thumbFlag := 0
	if isThumbnail {
		thumbFlag = 1
	}

	now := time.Now().Unix()
	_, err := a.db.Exec(`
		INSERT INTO media_blobs (content_cid, data, mime, size_bytes, width, height, is_thumbnail, created_at, last_accessed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(content_cid) DO UPDATE SET
			data = excluded.data,
			mime = excluded.mime,
			size_bytes = excluded.size_bytes,
			width = excluded.width,
			height = excluded.height,
			is_thumbnail = excluded.is_thumbnail,
			last_accessed_at = excluded.last_accessed_at;
	`, contentCID, data, mime, int64(len(data)), width, height, thumbFlag, now, now)
	return err
}

func (a *App) getContentBlobLocal(contentCID string) (PostBodyBlob, error) {
	var body PostBodyBlob
	err := a.db.QueryRow(`
		SELECT content_cid, body, size_bytes
		FROM content_blobs
		WHERE content_cid = ?;
	`, contentCID).Scan(&body.ContentCID, &body.Body, &body.SizeBytes)
	if err != nil {
		return PostBodyBlob{}, err
	}

	if _, err = a.db.Exec(`UPDATE content_blobs SET last_accessed_at = ? WHERE content_cid = ?;`, time.Now().Unix(), contentCID); err != nil {
		return PostBodyBlob{}, err
	}

	return body, nil
}

func (a *App) canServeContentBlobToNetwork(contentCID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return false, nil
	}

	rows, err := a.db.Query(`
		SELECT id, pubkey, timestamp, lamport
		FROM messages
		WHERE zone = 'public' AND visibility = 'normal' AND content_cid = ?;
	`, contentCID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id        string
			pubkey    string
			timestamp int64
			lamport   int64
		)
		if err = rows.Scan(&id, &pubkey, &timestamp, &lamport); err != nil {
			return false, err
		}

		allowed, allowErr := a.shouldAcceptPublicContent(pubkey, lamport, timestamp, id, "")
		if allowErr != nil {
			return false, allowErr
		}
		if allowed {
			return true, nil
		}
	}

	if err = rows.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func (a *App) canServeMediaBlobToNetwork(contentCID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return false, nil
	}

	rows, err := a.db.Query(`
		SELECT id, pubkey, timestamp, lamport
		FROM messages
		WHERE zone = 'public' AND visibility = 'normal' AND (image_cid = ? OR thumb_cid = ?);
	`, contentCID, contentCID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id        string
			pubkey    string
			timestamp int64
			lamport   int64
		)
		if err = rows.Scan(&id, &pubkey, &timestamp, &lamport); err != nil {
			return false, err
		}

		allowed, allowErr := a.shouldAcceptPublicContent(pubkey, lamport, timestamp, id, "")
		if allowErr != nil {
			return false, allowErr
		}
		if allowed {
			return true, nil
		}
	}

	if err = rows.Err(); err != nil {
		return false, err
	}

	commentRows, err := a.db.Query(`
		SELECT c.id, c.pubkey, c.timestamp, c.lamport
		FROM comment_media_refs r
		JOIN comments c ON c.id = r.comment_id
		JOIN messages m ON m.id = c.post_id
		WHERE r.content_cid = ? AND m.zone = 'public' AND m.visibility = 'normal';
	`, contentCID)
	if err != nil {
		return false, err
	}
	defer commentRows.Close()

	for commentRows.Next() {
		var (
			id        string
			pubkey    string
			timestamp int64
			lamport   int64
		)
		if err = commentRows.Scan(&id, &pubkey, &timestamp, &lamport); err != nil {
			return false, err
		}

		allowed, allowErr := a.shouldAcceptPublicContent(pubkey, lamport, timestamp, id, "")
		if allowErr != nil {
			return false, allowErr
		}
		if allowed {
			return true, nil
		}
	}

	if err = commentRows.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func (a *App) upsertContentBlob(contentCID string, body string, sizeBytes int64) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	body = strings.TrimSpace(body)
	if contentCID == "" || body == "" {
		return errors.New("invalid content blob")
	}
	if sizeBytes <= 0 {
		sizeBytes = int64(len([]byte(body)))
	}

	now := time.Now().Unix()
	_, err := a.db.Exec(`
		INSERT INTO content_blobs (content_cid, body, size_bytes, created_at, last_accessed_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(content_cid) DO UPDATE SET
			body = excluded.body,
			size_bytes = excluded.size_bytes,
			last_accessed_at = excluded.last_accessed_at;
	`, contentCID, body, sizeBytes, now, now)
	return err
}

func (a *App) listRecentPublicPostDigests(limit int) ([]SyncPostDigest, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 500 {
		limit = 200
	}

	rows, err := a.db.Query(`
		SELECT id, pubkey, current_op_id, visibility, deleted_at_lamport, title, content_cid, image_cid, thumb_cid, image_mime, image_size, image_width, image_height, timestamp, lamport, sub_id
		FROM messages
		WHERE zone = 'public' AND (visibility = 'normal' OR visibility = 'deleted')
		ORDER BY timestamp DESC
		LIMIT ?;
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SyncPostDigest, 0, limit)
	for rows.Next() {
		var digest SyncPostDigest
		var visibility string
		if err = rows.Scan(&digest.ID, &digest.Pubkey, &digest.OpID, &visibility, &digest.DeletedAtLamport, &digest.Title, &digest.ContentCID, &digest.ImageCID, &digest.ThumbCID, &digest.ImageMIME, &digest.ImageSize, &digest.ImageWidth, &digest.ImageHeight, &digest.Timestamp, &digest.Lamport, &digest.SubID); err != nil {
			return nil, err
		}
		digest.Deleted = strings.EqualFold(strings.TrimSpace(visibility), "deleted")
		if digest.Deleted {
			digest.OpType = postOpTypeDelete
		} else {
			digest.OpType = postOpTypeCreate
		}
		result = append(result, digest)
	}

	return result, rows.Err()
}

func (a *App) getLatestPublicPostTimestamp() (int64, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}

	var latest sql.NullInt64
	if err := a.db.QueryRow(`
		SELECT MAX(timestamp)
		FROM messages
		WHERE zone = 'public' AND visibility = 'normal';
	`).Scan(&latest); err != nil {
		return 0, err
	}

	if !latest.Valid {
		return 0, nil
	}

	return latest.Int64, nil
}

func (a *App) listPublicPostDigestsSince(sinceTimestamp int64, limit int) ([]SyncPostDigest, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 500 {
		limit = 200
	}

	policy, err := a.GetGovernancePolicy()
	if err != nil {
		return nil, err
	}

	var rows *sql.Rows
	err = nil
	if sinceTimestamp > 0 {
		if policy.HideHistoryOnShadowBan {
			rows, err = a.db.Query(`
				SELECT m.id, m.pubkey, m.current_op_id, m.visibility, m.deleted_at_lamport, m.title, m.content_cid, m.image_cid, m.thumb_cid, m.image_mime, m.image_size, m.image_width, m.image_height, m.timestamp, m.lamport, m.sub_id
				FROM messages m
				LEFT JOIN moderation mo ON mo.target_pubkey = m.pubkey
				WHERE m.zone = 'public' AND m.timestamp >= ?
				  AND (mo.action IS NULL OR UPPER(mo.action) != 'SHADOW_BAN')
				ORDER BY m.timestamp ASC
				LIMIT ?;
			`, sinceTimestamp, limit)
		} else {
			rows, err = a.db.Query(`
				SELECT m.id, m.pubkey, m.current_op_id, m.visibility, m.deleted_at_lamport, m.title, m.content_cid, m.image_cid, m.thumb_cid, m.image_mime, m.image_size, m.image_width, m.image_height, m.timestamp, m.lamport, m.sub_id
				FROM messages m
				LEFT JOIN moderation mo ON mo.target_pubkey = m.pubkey
				WHERE m.zone = 'public' AND m.timestamp >= ?
				  AND (
					mo.action IS NULL
					OR UPPER(mo.action) != 'SHADOW_BAN'
					OR m.lamport < mo.lamport
					OR (m.lamport = 0 OR mo.lamport = 0) AND m.timestamp < mo.timestamp
				  )
				ORDER BY m.timestamp ASC
				LIMIT ?;
			`, sinceTimestamp, limit)
		}
	} else {
		if policy.HideHistoryOnShadowBan {
			rows, err = a.db.Query(`
				SELECT m.id, m.pubkey, m.current_op_id, m.visibility, m.deleted_at_lamport, m.title, m.content_cid, m.image_cid, m.thumb_cid, m.image_mime, m.image_size, m.image_width, m.image_height, m.timestamp, m.lamport, m.sub_id
				FROM messages m
				LEFT JOIN moderation mo ON mo.target_pubkey = m.pubkey
				WHERE m.zone = 'public'
				  AND (mo.action IS NULL OR UPPER(mo.action) != 'SHADOW_BAN')
				ORDER BY m.timestamp DESC
				LIMIT ?;
			`, limit)
		} else {
			rows, err = a.db.Query(`
				SELECT m.id, m.pubkey, m.current_op_id, m.visibility, m.deleted_at_lamport, m.title, m.content_cid, m.image_cid, m.thumb_cid, m.image_mime, m.image_size, m.image_width, m.image_height, m.timestamp, m.lamport, m.sub_id
				FROM messages m
				LEFT JOIN moderation mo ON mo.target_pubkey = m.pubkey
				WHERE m.zone = 'public'
				  AND (
					mo.action IS NULL
					OR UPPER(mo.action) != 'SHADOW_BAN'
					OR m.lamport < mo.lamport
					OR (m.lamport = 0 OR mo.lamport = 0) AND m.timestamp < mo.timestamp
				  )
				ORDER BY m.timestamp DESC
				LIMIT ?;
			`, limit)
		}
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]SyncPostDigest, 0, limit)
	for rows.Next() {
		var digest SyncPostDigest
		var visibility string
		if err = rows.Scan(&digest.ID, &digest.Pubkey, &digest.OpID, &visibility, &digest.DeletedAtLamport, &digest.Title, &digest.ContentCID, &digest.ImageCID, &digest.ThumbCID, &digest.ImageMIME, &digest.ImageSize, &digest.ImageWidth, &digest.ImageHeight, &digest.Timestamp, &digest.Lamport, &digest.SubID); err != nil {
			return nil, err
		}
		digest.Deleted = strings.EqualFold(strings.TrimSpace(visibility), "deleted")
		if digest.Deleted {
			digest.OpType = postOpTypeDelete
			digest.ContentCID = ""
		} else {
			digest.OpType = postOpTypeCreate
		}
		result = append(result, digest)
	}

	return result, rows.Err()
}

func (a *App) upsertPublicPostIndexFromDigest(digest SyncPostDigest) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	digest.ID = strings.TrimSpace(digest.ID)
	digest.Pubkey = strings.TrimSpace(digest.Pubkey)
	digest.OpType = normalizeOperationType(digest.OpType, postOpTypeCreate)
	digest.OpID = resolveOperationID(digest.OpID, digest.ID, digest.Pubkey, digest.Lamport, digest.OpType)
	digest.Title = strings.TrimSpace(digest.Title)
	digest.ContentCID = strings.TrimSpace(digest.ContentCID)
	digest.ImageCID = strings.TrimSpace(digest.ImageCID)
	digest.ThumbCID = strings.TrimSpace(digest.ThumbCID)
	digest.ImageMIME = strings.TrimSpace(digest.ImageMIME)
	digest.SubID = normalizeSubID(digest.SubID)
	if digest.Timestamp <= 0 {
		digest.Timestamp = time.Now().Unix()
	}
	if digest.Lamport <= 0 {
		digest.Lamport = digest.Timestamp
	}
	if digest.DeletedAtLamport > digest.Lamport {
		digest.Lamport = digest.DeletedAtLamport
	}
	digest.OpID = resolveOperationID(digest.OpID, digest.ID, digest.Pubkey, digest.Lamport, digest.OpType)

	if digest.ID == "" || digest.Pubkey == "" {
		return false, errors.New("invalid sync digest")
	}
	if !digest.Deleted && digest.ContentCID == "" {
		return false, errors.New("invalid sync digest")
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
		FROM messages
		WHERE id = ?;
	`, digest.ID).Scan(&existingPubkey, &existingLamport, &existingAuthorPubkey, &existingOpID, &existingDeletedL)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	existingRow := err == nil
	if err == nil {
		existingPubkey = strings.TrimSpace(existingPubkey)
		if existingPubkey != "" && existingPubkey != digest.Pubkey {
			return false, errors.New("unauthorized post operation")
		}
		if strings.TrimSpace(existingAuthorPubkey) == "" {
			existingAuthorPubkey = existingPubkey
		}
		if strings.TrimSpace(existingOpID) == "" {
			existingOpID = digest.ID
		}

		incomingVersion := LamportVersion{Lamport: digest.Lamport, Author: digest.Pubkey, OpID: digest.OpID}
		currentVersion := LamportVersion{Lamport: existingLamport, Author: existingAuthorPubkey, OpID: existingOpID}
		if compareLamportVersion(incomingVersion, currentVersion) <= 0 {
			return false, nil
		}
		if existingDeletedL > 0 {
			tombstoneVersion := LamportVersion{Lamport: existingDeletedL, Author: existingAuthorPubkey, OpID: existingOpID}
			if compareLamportVersion(incomingVersion, tombstoneVersion) <= 0 {
				return false, nil
			}
		}
	}

	if digest.Title == "" {
		digest.Title = "Untitled"
	}

	if digest.Deleted || digest.OpType == postOpTypeDelete {
		if digest.DeletedAtLamport <= 0 {
			digest.DeletedAtLamport = digest.Lamport
		}
		result, execErr := a.db.Exec(`
			INSERT INTO messages (
				id, pubkey, current_author_pubkey, current_op_id, title, body, content_cid, image_cid, thumb_cid, image_mime,
				image_size, image_width, image_height, content, score, timestamp, lamport, size_bytes, zone, sub_id, is_protected,
				visibility, deleted_at_lamport, deleted_at_ts, deleted_by
			)
			VALUES (?, ?, ?, ?, '[deleted]', '', '', '', '', '', 0, 0, 0, '', 0, ?, ?, 0, 'public', ?, 0, 'deleted', ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				pubkey = excluded.pubkey,
				current_author_pubkey = excluded.current_author_pubkey,
				current_op_id = excluded.current_op_id,
				title = '[deleted]',
				body = '',
				content = '',
				content_cid = '',
				image_cid = '',
				thumb_cid = '',
				image_mime = '',
				image_size = 0,
				image_width = 0,
				image_height = 0,
				timestamp = excluded.timestamp,
				lamport = excluded.lamport,
				sub_id = excluded.sub_id,
				visibility = 'deleted',
				deleted_at_lamport = excluded.deleted_at_lamport,
				deleted_at_ts = excluded.deleted_at_ts,
				deleted_by = excluded.deleted_by;
		`, digest.ID, digest.Pubkey, digest.Pubkey, digest.OpID, digest.Timestamp, digest.Lamport, digest.SubID, digest.DeletedAtLamport, digest.Timestamp, digest.Pubkey)
		if execErr != nil {
			return false, execErr
		}
		affected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return false, rowsErr
		}
		if affected > 0 {
			if logErr := a.appendEntityOperation(
				entityTypePost,
				digest.ID,
				postOpTypeDelete,
				digest.OpID,
				digest.Pubkey,
				digest.Lamport,
				digest.Timestamp,
				lamportSchemaV2,
				authScopeUser,
				map[string]any{"deletedAtLamport": digest.DeletedAtLamport, "source": "digest"},
			); logErr != nil {
				return false, logErr
			}
		}
		return affected > 0, nil
	}

	result, err := a.db.Exec(`
		INSERT INTO messages (
			id, pubkey, current_author_pubkey, current_op_id, title, body, content_cid, image_cid, thumb_cid, image_mime,
			image_size, image_width, image_height, content, score, timestamp, lamport, size_bytes, zone, sub_id, is_protected,
			visibility, deleted_at_lamport, deleted_at_ts, deleted_by
		)
		VALUES (?, ?, ?, ?, ?, '', ?, ?, ?, ?, ?, ?, ?, '', 0, ?, ?, 0, 'public', ?, 0, 'normal', 0, 0, '')
		ON CONFLICT(id) DO UPDATE SET
			pubkey = excluded.pubkey,
			current_author_pubkey = excluded.current_author_pubkey,
			current_op_id = excluded.current_op_id,
			title = excluded.title,
			content_cid = excluded.content_cid,
			image_cid = excluded.image_cid,
			thumb_cid = excluded.thumb_cid,
			image_mime = excluded.image_mime,
			image_size = excluded.image_size,
			image_width = excluded.image_width,
			image_height = excluded.image_height,
			timestamp = excluded.timestamp,
			lamport = excluded.lamport,
			sub_id = excluded.sub_id,
			visibility = 'normal',
			deleted_at_lamport = 0,
			deleted_at_ts = 0,
			deleted_by = '';
	`, digest.ID, digest.Pubkey, digest.Pubkey, digest.OpID, digest.Title, digest.ContentCID, digest.ImageCID, digest.ThumbCID, digest.ImageMIME, digest.ImageSize, digest.ImageWidth, digest.ImageHeight, digest.Timestamp, digest.Lamport, digest.SubID)
	if err != nil {
		return false, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected > 0 {
		opType := normalizeOperationType(digest.OpType, postOpTypeCreate)
		if opType != postOpTypeDelete {
			if existingRow {
				opType = postOpTypeUpdate
			} else {
				opType = postOpTypeCreate
			}
		}
		if logErr := a.appendEntityOperation(
			entityTypePost,
			digest.ID,
			opType,
			digest.OpID,
			digest.Pubkey,
			digest.Lamport,
			digest.Timestamp,
			lamportSchemaV2,
			authScopeUser,
			map[string]any{"subId": digest.SubID, "source": "digest"},
		); logErr != nil {
			return false, logErr
		}
	}

	return affected > 0, nil
}

func (a *App) hasContentBlobLocal(contentCID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return false, nil
	}

	var exists int
	err := a.db.QueryRow(`SELECT 1 FROM content_blobs WHERE content_cid = ? LIMIT 1;`, contentCID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (a *App) hasMediaBlobLocal(contentCID string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	contentCID = strings.TrimSpace(contentCID)
	if contentCID == "" {
		return false, nil
	}

	var exists int
	err := a.db.QueryRow(`SELECT 1 FROM media_blobs WHERE content_cid = ? LIMIT 1;`, contentCID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
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

func (a *App) UpdateProfile(displayName string, avatarURL string) (Profile, error) {
	if a.db == nil {
		return Profile{}, errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return Profile{}, err
	}

	return a.upsertProfile(identity.PublicKey, displayName, avatarURL, time.Now().Unix())
}

func (a *App) UpdateProfileDetails(displayName string, avatarURL string, bio string) (ProfileDetails, error) {
	if a.db == nil {
		return ProfileDetails{}, errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return ProfileDetails{}, err
	}

	updatedAt := time.Now().Unix()
	profile, err := a.upsertProfile(identity.PublicKey, displayName, avatarURL, updatedAt)
	if err != nil {
		return ProfileDetails{}, err
	}

	bio = strings.TrimSpace(bio)
	if len([]rune(bio)) > 160 {
		bio = string([]rune(bio)[:160])
	}

	_, err = a.db.Exec(`
		INSERT INTO profile_details (pubkey, bio, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(pubkey) DO UPDATE SET
			bio = excluded.bio,
			updated_at = excluded.updated_at;
	`, profile.Pubkey, bio, updatedAt)
	if err != nil {
		return ProfileDetails{}, err
	}

	return ProfileDetails{
		Pubkey:      profile.Pubkey,
		DisplayName: profile.DisplayName,
		AvatarURL:   profile.AvatarURL,
		Bio:         bio,
		UpdatedAt:   updatedAt,
	}, nil
}

func (a *App) GetProfile(pubkey string) (Profile, error) {
	if a.db == nil {
		return Profile{}, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return Profile{}, errors.New("pubkey is required")
	}

	var profile Profile
	err := a.db.QueryRow(`
		SELECT pubkey, display_name, avatar_url, updated_at
		FROM profiles
		WHERE pubkey = ?;
	`, pubkey).Scan(&profile.Pubkey, &profile.DisplayName, &profile.AvatarURL, &profile.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{Pubkey: pubkey}, nil
	}
	if err != nil {
		return Profile{}, err
	}

	return profile, nil
}

func (a *App) GetProfileDetails(pubkey string) (ProfileDetails, error) {
	if a.db == nil {
		return ProfileDetails{}, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return ProfileDetails{}, errors.New("pubkey is required")
	}

	profile, err := a.GetProfile(pubkey)
	if err != nil {
		return ProfileDetails{}, err
	}

	var (
		bio              string
		detailsUpdatedAt int64
	)
	err = a.db.QueryRow(`
		SELECT bio, updated_at
		FROM profile_details
		WHERE pubkey = ?;
	`, pubkey).Scan(&bio, &detailsUpdatedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ProfileDetails{}, err
	}

	updatedAt := profile.UpdatedAt
	if detailsUpdatedAt > updatedAt {
		updatedAt = detailsUpdatedAt
	}

	return ProfileDetails{
		Pubkey:      profile.Pubkey,
		DisplayName: profile.DisplayName,
		AvatarURL:   profile.AvatarURL,
		Bio:         bio,
		UpdatedAt:   updatedAt,
	}, nil
}

func (a *App) GetPrivacySettings() (PrivacySettings, error) {
	if a.db == nil {
		return PrivacySettings{}, errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return PrivacySettings{}, err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return PrivacySettings{}, errors.New("identity pubkey is empty")
	}

	var (
		showOnlineStatus int
		allowSearch      int
		updatedAt        int64
	)
	err = a.db.QueryRow(`
		SELECT show_online_status, allow_search, updated_at
		FROM privacy_settings
		WHERE pubkey = ?;
	`, pubkey).Scan(&showOnlineStatus, &allowSearch, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return PrivacySettings{
			ShowOnlineStatus: true,
			AllowSearch:      true,
			UpdatedAt:        0,
		}, nil
	}
	if err != nil {
		return PrivacySettings{}, err
	}

	return PrivacySettings{
		ShowOnlineStatus: showOnlineStatus == 1,
		AllowSearch:      allowSearch == 1,
		UpdatedAt:        updatedAt,
	}, nil
}

func (a *App) SetPrivacySettings(showOnlineStatus bool, allowSearch bool) (PrivacySettings, error) {
	if a.db == nil {
		return PrivacySettings{}, errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return PrivacySettings{}, err
	}
	pubkey := strings.TrimSpace(identity.PublicKey)
	if pubkey == "" {
		return PrivacySettings{}, errors.New("identity pubkey is empty")
	}

	updatedAt := time.Now().Unix()
	showOnlineStatusInt := 0
	if showOnlineStatus {
		showOnlineStatusInt = 1
	}
	allowSearchInt := 0
	if allowSearch {
		allowSearchInt = 1
	}

	_, err = a.db.Exec(`
		INSERT INTO privacy_settings (pubkey, show_online_status, allow_search, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(pubkey) DO UPDATE SET
			show_online_status = excluded.show_online_status,
			allow_search = excluded.allow_search,
			updated_at = excluded.updated_at;
	`, pubkey, showOnlineStatusInt, allowSearchInt, updatedAt)
	if err != nil {
		return PrivacySettings{}, err
	}

	return PrivacySettings{
		ShowOnlineStatus: showOnlineStatus,
		AllowSearch:      allowSearch,
		UpdatedAt:        updatedAt,
	}, nil
}

func (a *App) GetModerationState() ([]ModerationState, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT target_pubkey, action, source_admin, timestamp, lamport, reason
		FROM moderation
		ORDER BY timestamp DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ModerationState, 0)
	for rows.Next() {
		var state ModerationState
		if err := rows.Scan(&state.TargetPubkey, &state.Action, &state.SourceAdmin, &state.Timestamp, &state.Lamport, &state.Reason); err != nil {
			return nil, err
		}
		result = append(result, state)
	}

	return result, rows.Err()
}

func (a *App) getLatestModerationTimestamp() (int64, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}

	var latest sql.NullInt64
	if err := a.db.QueryRow(`SELECT MAX(timestamp) FROM moderation;`).Scan(&latest); err != nil {
		return 0, err
	}
	if !latest.Valid {
		return 0, nil
	}
	return latest.Int64, nil
}

func (a *App) listModerationSince(sinceTimestamp int64, limit int) ([]ModerationState, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 500 {
		limit = 200
	}
	if sinceTimestamp < 0 {
		sinceTimestamp = 0
	}

	rows, err := a.db.Query(`
		SELECT target_pubkey, action, source_admin, timestamp, lamport, reason
		FROM moderation
		WHERE timestamp >= ?
		ORDER BY timestamp ASC
		LIMIT ?;
	`, sinceTimestamp, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ModerationState, 0, limit)
	for rows.Next() {
		var row ModerationState
		if err = rows.Scan(&row.TargetPubkey, &row.Action, &row.SourceAdmin, &row.Timestamp, &row.Lamport, &row.Reason); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

func (a *App) getLatestAppliedModerationLogTimestamp() (int64, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}

	var latest sql.NullInt64
	if err := a.db.QueryRow(`SELECT MAX(timestamp) FROM moderation_logs WHERE result = 'applied';`).Scan(&latest); err != nil {
		return 0, err
	}
	if !latest.Valid {
		return 0, nil
	}
	return latest.Int64, nil
}

func (a *App) listAppliedModerationLogsSince(sinceTimestamp int64, limit int) ([]ModerationLog, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 500 {
		limit = 200
	}
	if sinceTimestamp < 0 {
		sinceTimestamp = 0
	}

	rows, err := a.db.Query(`
		SELECT id, target_pubkey, action, source_admin, timestamp, lamport, reason, result
		FROM moderation_logs
		WHERE result = 'applied' AND timestamp >= ?
		ORDER BY timestamp ASC, id ASC
		LIMIT ?;
	`, sinceTimestamp, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ModerationLog, 0, limit)
	for rows.Next() {
		var row ModerationLog
		if err = rows.Scan(&row.ID, &row.TargetPubkey, &row.Action, &row.SourceAdmin, &row.Timestamp, &row.Lamport, &row.Reason, &row.Result); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

func (a *App) insertModerationLogIfAbsent(log ModerationLog) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	log.TargetPubkey = strings.TrimSpace(log.TargetPubkey)
	log.Action = strings.ToUpper(strings.TrimSpace(log.Action))
	log.SourceAdmin = strings.TrimSpace(log.SourceAdmin)
	log.Reason = strings.TrimSpace(log.Reason)
	log.Result = strings.TrimSpace(log.Result)
	if log.Result == "" {
		log.Result = "applied"
	}
	if log.TargetPubkey == "" || log.SourceAdmin == "" || log.Action == "" {
		return false, errors.New("invalid moderation log payload")
	}
	if log.Timestamp <= 0 {
		log.Timestamp = time.Now().Unix()
	}
	if log.Lamport <= 0 {
		log.Lamport = log.Timestamp
	}

	var exists int
	err := a.db.QueryRow(`
		SELECT 1
		FROM moderation_logs
		WHERE target_pubkey = ?
		  AND action = ?
		  AND source_admin = ?
		  AND timestamp = ?
		  AND reason = ?
		  AND result = ?
		LIMIT 1;
	`, log.TargetPubkey, log.Action, log.SourceAdmin, log.Timestamp, log.Reason, log.Result).Scan(&exists)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}

	_, err = a.db.Exec(`
		INSERT INTO moderation_logs (target_pubkey, action, source_admin, timestamp, lamport, reason, result)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`, log.TargetPubkey, log.Action, log.SourceAdmin, log.Timestamp, log.Lamport, log.Reason, log.Result)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (a *App) GetModerationLogs(limit int) ([]ModerationLog, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 500 {
		limit = 100
	}

	rows, err := a.db.Query(`
		SELECT id, target_pubkey, action, source_admin, timestamp, lamport, reason, result
		FROM moderation_logs
		ORDER BY timestamp DESC, id DESC
		LIMIT ?;
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ModerationLog, 0)
	for rows.Next() {
		var row ModerationLog
		if err = rows.Scan(&row.ID, &row.TargetPubkey, &row.Action, &row.SourceAdmin, &row.Timestamp, &row.Lamport, &row.Reason, &row.Result); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

func (a *App) GetGovernancePolicy() (GovernancePolicy, error) {
	if a.db == nil {
		return GovernancePolicy{}, errors.New("database not initialized")
	}

	var value string
	err := a.db.QueryRow(`SELECT value FROM governance_config WHERE key = 'hide_history_on_shadowban';`).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return GovernancePolicy{HideHistoryOnShadowBan: true}, nil
	}
	if err != nil {
		return GovernancePolicy{}, err
	}

	value = strings.TrimSpace(strings.ToLower(value))
	hide := value == "1" || value == "true" || value == "yes"

	return GovernancePolicy{HideHistoryOnShadowBan: hide}, nil
}

func (a *App) SetGovernancePolicy(hideHistoryOnShadowBan bool) (GovernancePolicy, error) {
	if a.db == nil {
		return GovernancePolicy{}, errors.New("database not initialized")
	}

	value := "0"
	if hideHistoryOnShadowBan {
		value = "1"
	}

	_, err := a.db.Exec(`
		INSERT INTO governance_config (key, value, updated_at)
		VALUES ('hide_history_on_shadowban', ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = excluded.updated_at;
	`, value, time.Now().Unix())
	if err != nil {
		return GovernancePolicy{}, err
	}

	if hideHistoryOnShadowBan {
		if _, err = a.db.Exec(`
			UPDATE messages
			SET visibility = 'shadowed'
			WHERE zone = 'public'
			  AND pubkey IN (
				SELECT target_pubkey FROM moderation WHERE action = 'SHADOW_BAN'
			  );
		`); err != nil {
			return GovernancePolicy{}, err
		}
	} else {
		if _, err = a.db.Exec(`
			UPDATE messages
			SET visibility = 'normal'
			WHERE zone = 'public'
			  AND visibility = 'shadowed'
			  AND pubkey IN (
				SELECT target_pubkey FROM moderation WHERE action = 'SHADOW_BAN'
			  );
		`); err != nil {
			return GovernancePolicy{}, err
		}
	}

	return GovernancePolicy{HideHistoryOnShadowBan: hideHistoryOnShadowBan}, nil
}

func (a *App) GetIdentityState() ([]IdentityState, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT pubkey, state, storage_commit_bytes, public_quota_bytes, private_quota_bytes, updated_at
		FROM identity_state
		ORDER BY updated_at DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]IdentityState, 0)
	for rows.Next() {
		var state IdentityState
		if err := rows.Scan(
			&state.Pubkey,
			&state.State,
			&state.StorageCommitBytes,
			&state.PublicQuotaBytes,
			&state.PrivateQuotaBytes,
			&state.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, state)
	}

	return result, rows.Err()
}

func (a *App) GetStorageUsage() (StorageUsage, error) {
	if a.db == nil {
		return StorageUsage{}, errors.New("database not initialized")
	}

	usage := StorageUsage{
		PrivateQuota: privateQuotaBytes,
		PublicQuota:  publicQuotaBytes,
		TotalQuota:   totalQuotaBytes,
	}

	rows, err := a.db.Query(`
		SELECT content_cid, size_bytes
		FROM content_blobs;
	`)
	if err != nil {
		return StorageUsage{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid  string
			size int64
		)
		if err = rows.Scan(&cid, &size); err != nil {
			return StorageUsage{}, err
		}

		shareable, shareErr := a.canServeContentBlobToNetwork(cid)
		if shareErr != nil {
			return StorageUsage{}, shareErr
		}
		if shareable {
			usage.PublicUsedBytes += size
		} else {
			usage.PrivateUsedBytes += size
		}
	}
	if err = rows.Err(); err != nil {
		return StorageUsage{}, err
	}

	mediaRows, err := a.db.Query(`
		SELECT content_cid, size_bytes
		FROM media_blobs;
	`)
	if err != nil {
		return StorageUsage{}, err
	}
	defer mediaRows.Close()

	for mediaRows.Next() {
		var (
			cid  string
			size int64
		)
		if err = mediaRows.Scan(&cid, &size); err != nil {
			return StorageUsage{}, err
		}

		shareable, shareErr := a.canServeMediaBlobToNetwork(cid)
		if shareErr != nil {
			return StorageUsage{}, shareErr
		}
		if shareable {
			usage.PublicUsedBytes += size
		} else {
			usage.PrivateUsedBytes += size
		}
	}
	if err = mediaRows.Err(); err != nil {
		return StorageUsage{}, err
	}

	return usage, nil
}

func (a *App) ResetLocalTestData() error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	tx, err := a.db.Begin()
	if err != nil {
		return err
	}

	statements := []string{
		`DELETE FROM comment_media_refs;`,
		`DELETE FROM comments;`,
		`DELETE FROM comment_votes;`,
		`DELETE FROM post_votes;`,
		`DELETE FROM comment_downvotes;`,
		`DELETE FROM post_downvotes;`,
		`DELETE FROM vote_ops;`,
		`DELETE FROM post_favorite_ops;`,
		`DELETE FROM post_favorites_state;`,
		`DELETE FROM entity_ops;`,
		`DELETE FROM tombstone_gc_marks;`,
		`DELETE FROM messages;`,
		`DELETE FROM content_blobs;`,
		`DELETE FROM media_blobs;`,
		`DELETE FROM sub_subscriptions;`,
		`DELETE FROM subs;`,
		`DELETE FROM moderation_logs;`,
		`DELETE FROM moderation;`,
		`DELETE FROM known_peers;`,
		`DELETE FROM identity_state;`,
		`DELETE FROM profiles;`,
		`DELETE FROM profile_details;`,
		`DELETE FROM logical_clock;`,
	}

	for _, statement := range statements {
		if _, execErr := tx.Exec(statement); execErr != nil {
			_ = tx.Rollback()
			return execErr
		}
	}

	now := time.Now().Unix()
	if _, err = tx.Exec(`
		INSERT INTO subs (id, title, description, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			description = excluded.description,
			created_at = excluded.created_at;
	`, defaultSubID, "General", "Default public community", now); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	if a.ctx != nil {
		a.emitEvent("feed:updated")
		a.emitEvent("subs:updated")
		a.emitEvent("p2p:updated")
	}

	a.releaseAlertMu.Lock()
	a.releaseAlertState = make(map[string]int64)
	a.releaseAlertActive = make(map[string]ReleaseAlert)
	a.releaseAlertMu.Unlock()

	a.antiEntropyMu.Lock()
	a.antiEntropyStats = AntiEntropyStats{}
	a.antiEntropyMu.Unlock()

	return nil
}

func (a *App) nextLamport() (int64, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}

	now := time.Now().Unix()
	if _, err := a.db.Exec(`
		INSERT INTO logical_clock (scope, value, updated_at)
		VALUES ('global', 0, ?)
		ON CONFLICT(scope) DO NOTHING;
	`, now); err != nil {
		return 0, err
	}

	var lamport int64
	if err := a.db.QueryRow(`
		UPDATE logical_clock
		SET value = value + 1,
		    updated_at = ?
		WHERE scope = 'global'
		RETURNING value;
	`, now).Scan(&lamport); err != nil {
		return 0, err
	}

	return lamport, nil
}

func (a *App) observeLamport(incomingLamport int64) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	if incomingLamport < 0 {
		incomingLamport = 0
	}
	now := time.Now().Unix()
	if _, err := a.db.Exec(`
		INSERT INTO logical_clock (scope, value, updated_at)
		VALUES ('global', 0, ?)
		ON CONFLICT(scope) DO NOTHING;
	`, now); err != nil {
		return err
	}

	_, err := a.db.Exec(`
		UPDATE logical_clock
		SET value = CASE
			WHEN value > ? THEN value + 1
			ELSE ? + 1
		END,
		updated_at = ?
		WHERE scope = 'global';
	`, incomingLamport, incomingLamport, now)
	return err
}

func (a *App) normalizeIncomingLamport(incomingLamport int64, timestamp int64) (int64, error) {
	if timestamp <= 0 {
		timestamp = time.Now().Unix()
	}
	if incomingLamport <= 0 {
		incomingLamport = timestamp
	}
	if err := a.observeLamport(incomingLamport); err != nil {
		return 0, err
	}

	return incomingLamport, nil
}

func (a *App) ProcessIncomingMessage(payload []byte) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	var message IncomingMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return err
	}

	message.Type = strings.ToUpper(strings.TrimSpace(message.Type))
	if signedIncomingMessageType(message.Type) {
		if err := a.verifyIncomingMessageSignature(message); err != nil {
			return err
		}
	}

	switch message.Type {
	case "SUB_CREATE":
		_, err := a.upsertSub(message.SubID, message.SubTitle, message.SubDesc, message.Timestamp)
		if err != nil {
			return err
		}
		if a.ctx != nil {
			a.emitEvent("subs:updated")
		}
		return nil
	case "GOVERNANCE_POLICY_UPDATE":
		trusted, trustErr := a.isTrustedAdmin(message.AdminPubkey)
		if trustErr != nil {
			return trustErr
		}
		if !trusted {
			return errors.New("admin pubkey is not trusted")
		}
		_, policyErr := a.SetGovernancePolicy(message.HideHistoryOnShadowBan)
		return policyErr
	case messageTypeSubSettingsUpdate:
		trusted, trustErr := a.isTrustedAdmin(message.AdminPubkey)
		if trustErr != nil {
			return trustErr
		}
		if !trusted {
			return errors.New("admin pubkey is not trusted")
		}
		if message.Timestamp <= 0 {
			message.Timestamp = time.Now().Unix()
		}
		lamport, err := a.normalizeIncomingLamport(message.Lamport, message.Timestamp)
		if err != nil {
			return err
		}
		return a.applySubSettingsUpdate(message.SubID, message.Rules, message.Announcement, message.AdminPubkey, message.Timestamp, lamport, message.OpID)
	case messageTypePostPinSet:
		trusted, trustErr := a.isTrustedAdmin(message.AdminPubkey)
		if trustErr != nil {
			return trustErr
		}
		if !trusted {
			return errors.New("admin pubkey is not trusted")
		}
		if message.Timestamp <= 0 {
			message.Timestamp = time.Now().Unix()
		}
		lamport, err := a.normalizeIncomingLamport(message.Lamport, message.Timestamp)
		if err != nil {
			return err
		}
		return a.applyPostPinnedState(message.PostID, message.Pinned, message.AdminPubkey, message.Timestamp, lamport, message.OpID)
	case messageTypePostLockSet:
		trusted, trustErr := a.isTrustedAdmin(message.AdminPubkey)
		if trustErr != nil {
			return trustErr
		}
		if !trusted {
			return errors.New("admin pubkey is not trusted")
		}
		if message.Timestamp <= 0 {
			message.Timestamp = time.Now().Unix()
		}
		lamport, err := a.normalizeIncomingLamport(message.Lamport, message.Timestamp)
		if err != nil {
			return err
		}
		return a.applyPostLockedState(message.PostID, message.Locked, message.AdminPubkey, message.Timestamp, lamport, message.OpID)
	case "PROFILE_UPDATE":
		_, err := a.upsertProfile(message.Pubkey, message.DisplayName, message.AvatarURL, message.Timestamp)
		return err
	case "POST_DELETE":
		if scope := strings.TrimSpace(strings.ToLower(message.AuthScope)); scope != "" && scope != authScopeUser {
			return errors.New("invalid post delete auth scope")
		}
		if strings.TrimSpace(message.Pubkey) == "" || strings.TrimSpace(message.PostID) == "" {
			return errors.New("invalid post delete payload")
		}
		if message.Timestamp <= 0 {
			message.Timestamp = time.Now().Unix()
		}
		lamport, err := a.normalizeIncomingLamport(message.Lamport, message.Timestamp)
		if err != nil {
			return err
		}
		deleteLamport := lamport
		if message.DeletedAtLamport > deleteLamport {
			deleteLamport = message.DeletedAtLamport
		}
		err = a.deleteLocalPostAsAuthor(message.Pubkey, message.PostID, message.Timestamp, deleteLamport, message.OpID)
		if err == nil {
			return nil
		}
		if strings.Contains(strings.ToLower(err.Error()), "post not found") {
			_, upsertErr := a.upsertPublicPostIndexFromDigest(SyncPostDigest{
				ID:               strings.TrimSpace(message.PostID),
				Pubkey:           strings.TrimSpace(message.Pubkey),
				OpID:             resolveOperationID(message.OpID, message.PostID, message.Pubkey, deleteLamport, postOpTypeDelete),
				OpType:           postOpTypeDelete,
				Deleted:          true,
				Timestamp:        message.Timestamp,
				Lamport:          deleteLamport,
				DeletedAtLamport: deleteLamport,
				SubID:            defaultSubID,
			})
			return upsertErr
		}
		return err
	case "COMMENT_DELETE":
		if scope := strings.TrimSpace(strings.ToLower(message.AuthScope)); scope != "" && scope != authScopeUser {
			return errors.New("invalid comment delete auth scope")
		}
		if strings.TrimSpace(message.Pubkey) == "" || strings.TrimSpace(message.CommentID) == "" {
			return errors.New("invalid comment delete payload")
		}
		if message.Timestamp <= 0 {
			message.Timestamp = time.Now().Unix()
		}
		lamport, err := a.normalizeIncomingLamport(message.Lamport, message.Timestamp)
		if err != nil {
			return err
		}
		deleteLamport := lamport
		if message.DeletedAtLamport > deleteLamport {
			deleteLamport = message.DeletedAtLamport
		}
		_, err = a.deleteLocalCommentAsAuthor(message.Pubkey, message.CommentID, message.Timestamp, deleteLamport, message.OpID)
		if err == nil {
			return nil
		}
		if strings.Contains(strings.ToLower(err.Error()), "comment not found") {
			return a.upsertCommentTombstone(message.CommentID, strings.TrimSpace(message.PostID), message.Pubkey, message.Timestamp, deleteLamport, message.OpID)
		}
		return err
	case "POST_UPVOTE":
		voterPubkey := strings.TrimSpace(message.VoterPubkey)
		if voterPubkey == "" {
			voterPubkey = strings.TrimSpace(message.Pubkey)
		}
		if voterPubkey == "" || strings.TrimSpace(message.PostID) == "" {
			return errors.New("invalid post upvote payload")
		}

		if err := a.applyPostUpvote(voterPubkey, message.PostID, message.OpID); err != nil {
			return err
		}
		if postAuthor, paErr := a.getPostAuthor(message.PostID); paErr == nil && postAuthor != "" {
			if localID, liErr := a.getLocalIdentity(); liErr == nil && strings.TrimSpace(localID.PublicKey) == postAuthor {
				a.tryGenerateNotification(NotifTypePostUpvote, voterPubkey, message.PostID, "post", message.PostID, message.Timestamp)
			}
		}
		return nil
	case "POST_DOWNVOTE":
		voterPubkey := strings.TrimSpace(message.VoterPubkey)
		if voterPubkey == "" {
			voterPubkey = strings.TrimSpace(message.Pubkey)
		}
		if voterPubkey == "" || strings.TrimSpace(message.PostID) == "" {
			return errors.New("invalid post downvote payload")
		}

		if err := a.applyPostDownvote(voterPubkey, message.PostID, message.OpID); err != nil {
			return err
		}
		if postAuthor, paErr := a.getPostAuthor(message.PostID); paErr == nil && postAuthor != "" {
			if localID, liErr := a.getLocalIdentity(); liErr == nil && strings.TrimSpace(localID.PublicKey) == postAuthor {
				a.tryGenerateNotification(NotifTypePostDownvote, voterPubkey, message.PostID, "post", message.PostID, message.Timestamp)
			}
		}
		return nil
	case "POST_VOTE_SET":
		voterPubkey := strings.TrimSpace(message.VoterPubkey)
		if voterPubkey == "" {
			voterPubkey = strings.TrimSpace(message.Pubkey)
		}
		if voterPubkey == "" || strings.TrimSpace(message.PostID) == "" {
			return errors.New("invalid post vote set payload")
		}
		if err := a.applyPostVoteState(voterPubkey, message.PostID, message.VoteState, message.OpID); err != nil {
			return err
		}
		if postAuthor, paErr := a.getPostAuthor(message.PostID); paErr == nil && postAuthor != "" {
			if localID, liErr := a.getLocalIdentity(); liErr == nil && strings.TrimSpace(localID.PublicKey) == postAuthor {
				notifType := NotifTypePostUpvote
				if strings.TrimSpace(message.VoteState) == "down" {
					notifType = NotifTypePostDownvote
				}
				a.tryGenerateNotification(notifType, voterPubkey, message.PostID, "post", message.PostID, message.Timestamp)
			}
		}
		return nil
	case "COMMENT_UPVOTE":
		voterPubkey := strings.TrimSpace(message.VoterPubkey)
		if voterPubkey == "" {
			voterPubkey = strings.TrimSpace(message.Pubkey)
		}
		if voterPubkey == "" || strings.TrimSpace(message.CommentID) == "" || strings.TrimSpace(message.PostID) == "" {
			return errors.New("invalid comment upvote payload")
		}

		if err := a.applyCommentUpvote(voterPubkey, message.CommentID, message.PostID, message.OpID); err != nil {
			return err
		}
		if commentAuthor, caErr := a.getCommentAuthor(message.CommentID); caErr == nil && commentAuthor != "" {
			if localID, liErr := a.getLocalIdentity(); liErr == nil && strings.TrimSpace(localID.PublicKey) == commentAuthor {
				a.tryGenerateNotification(NotifTypeCommentUpvote, voterPubkey, message.CommentID, "comment", message.PostID, message.Timestamp)
			}
		}
		return nil
	case "COMMENT_DOWNVOTE":
		voterPubkey := strings.TrimSpace(message.VoterPubkey)
		if voterPubkey == "" {
			voterPubkey = strings.TrimSpace(message.Pubkey)
		}
		if voterPubkey == "" || strings.TrimSpace(message.CommentID) == "" || strings.TrimSpace(message.PostID) == "" {
			return errors.New("invalid comment downvote payload")
		}

		if err := a.applyCommentDownvote(voterPubkey, message.CommentID, message.PostID, message.OpID); err != nil {
			return err
		}
		if commentAuthor, caErr := a.getCommentAuthor(message.CommentID); caErr == nil && commentAuthor != "" {
			if localID, liErr := a.getLocalIdentity(); liErr == nil && strings.TrimSpace(localID.PublicKey) == commentAuthor {
				a.tryGenerateNotification(NotifTypeCommentDownvote, voterPubkey, message.CommentID, "comment", message.PostID, message.Timestamp)
			}
		}
		return nil
	case "COMMENT_VOTE_SET":
		voterPubkey := strings.TrimSpace(message.VoterPubkey)
		if voterPubkey == "" {
			voterPubkey = strings.TrimSpace(message.Pubkey)
		}
		if voterPubkey == "" || strings.TrimSpace(message.CommentID) == "" || strings.TrimSpace(message.PostID) == "" {
			return errors.New("invalid comment vote set payload")
		}
		if err := a.applyCommentVoteState(voterPubkey, message.CommentID, message.PostID, message.VoteState, message.OpID); err != nil {
			return err
		}
		if commentAuthor, caErr := a.getCommentAuthor(message.CommentID); caErr == nil && commentAuthor != "" {
			if localID, liErr := a.getLocalIdentity(); liErr == nil && strings.TrimSpace(localID.PublicKey) == commentAuthor {
				notifType := NotifTypeCommentUpvote
				if strings.TrimSpace(message.VoteState) == "down" {
					notifType = NotifTypeCommentDownvote
				}
				a.tryGenerateNotification(notifType, voterPubkey, message.CommentID, "comment", message.PostID, message.Timestamp)
			}
		}
		return nil
	case messageTypeFavoriteOp:
		localIdentity, err := a.getLocalIdentity()
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "identity not found") {
				return nil
			}
			return err
		}
		localPubkey := strings.TrimSpace(localIdentity.PublicKey)
		if localPubkey == "" {
			return nil
		}

		record := FavoriteOpRecord{
			OpID:      strings.TrimSpace(message.FavoriteOpID),
			Pubkey:    strings.TrimSpace(message.Pubkey),
			PostID:    strings.TrimSpace(message.PostID),
			Op:        strings.TrimSpace(message.FavoriteOp),
			CreatedAt: message.Timestamp,
			Signature: strings.TrimSpace(message.Signature),
		}
		if record.Pubkey != localPubkey {
			return nil
		}

		applied, applyErr := a.applyFavoriteOperation(record, true)
		if applyErr != nil {
			return applyErr
		}
		if applied {
			a.emitFavoritesUpdated(record.PostID)
		}
		return nil
	case "COMMENT":
		if scope := strings.TrimSpace(strings.ToLower(message.AuthScope)); scope != "" && scope != authScopeUser {
			return errors.New("invalid comment auth scope")
		}
		if strings.TrimSpace(message.Pubkey) == "" || strings.TrimSpace(message.PostID) == "" {
			return errors.New("invalid comment payload")
		}
		if message.Timestamp == 0 {
			message.Timestamp = time.Now().Unix()
		}
		lamport, err := a.normalizeIncomingLamport(message.Lamport, message.Timestamp)
		if err != nil {
			return err
		}
		message.Lamport = lamport

		viewerPubkey := ""
		if identity, idErr := a.getLocalIdentity(); idErr == nil {
			viewerPubkey = strings.TrimSpace(identity.PublicKey)
		}
		allowed, err := a.shouldAcceptPublicContent(message.Pubkey, message.Lamport, message.Timestamp, message.ID, viewerPubkey)
		if err != nil {
			return err
		}
		if !allowed {
			return nil
		}
		if locked, lockLamport, lockErr := a.getPostLockState(message.PostID); lockErr != nil {
			return lockErr
		} else if locked && (lockLamport == 0 || message.Lamport > lockLamport) {
			return errors.New("post is locked")
		}

		commentBody := strings.TrimSpace(message.Body)
		attachments := normalizeCommentAttachments(message.CommentAttachments)
		if commentBody == "" && len(attachments) == 0 {
			return errors.New("invalid comment payload")
		}

		if strings.TrimSpace(message.DisplayName) != "" || strings.TrimSpace(message.AvatarURL) != "" {
			if _, err := a.upsertProfile(message.Pubkey, message.DisplayName, message.AvatarURL, message.Timestamp); err != nil {
				return err
			}
		}

		if strings.TrimSpace(message.ID) == "" {
			attachmentsJSON, err := encodeCommentAttachmentsJSON(attachments)
			if err != nil {
				return err
			}
			raw := fmt.Sprintf("%s|%s|%s|%s|%d", message.PostID, strings.TrimSpace(message.ParentID), commentBody, attachmentsJSON, message.Lamport)
			message.ID = buildMessageID(message.Pubkey, raw, message.Timestamp)
		}

		_, err = a.insertComment(Comment{
			ID:          message.ID,
			PostID:      strings.TrimSpace(message.PostID),
			ParentID:    strings.TrimSpace(message.ParentID),
			Pubkey:      message.Pubkey,
			OpID:        resolveOperationID(message.OpID, message.ID, message.Pubkey, message.Lamport, normalizeOperationType(message.OpType, postOpTypeCreate)),
			Body:        commentBody,
			Attachments: attachments,
			Timestamp:   message.Timestamp,
			Lamport:     message.Lamport,
		})
		if err == nil {
			postID := strings.TrimSpace(message.PostID)
			// Notify post author about new comment (only if post author is local user).
			if postAuthor, paErr := a.getPostAuthor(postID); paErr == nil && postAuthor != "" {
				localID, liErr := a.getLocalIdentity()
				if liErr == nil && strings.TrimSpace(localID.PublicKey) == postAuthor {
					a.tryGenerateNotification(NotifTypePostComment, message.Pubkey, message.ID, "comment", postID, message.Timestamp)
				}
			}
			// Notify parent comment author about reply (only if parent author is local user).
			parentID := strings.TrimSpace(message.ParentID)
			if parentID != "" {
				if parentAuthor, pcErr := a.getCommentAuthor(parentID); pcErr == nil && parentAuthor != "" {
					localID, liErr := a.getLocalIdentity()
					if liErr == nil && strings.TrimSpace(localID.PublicKey) == parentAuthor {
						a.tryGenerateNotification(NotifTypeCommentReply, message.Pubkey, message.ID, "comment", postID, message.Timestamp)
					}
				}
			}
		}
		return err
	case "SHADOW_BAN":
		trusted, err := a.isTrustedAdmin(message.AdminPubkey)
		if err != nil {
			return err
		}
		if !trusted {
			return errors.New("admin pubkey is not trusted")
		}
		if message.Timestamp == 0 {
			message.Timestamp = time.Now().Unix()
		}
		lamport, err := a.normalizeIncomingLamport(message.Lamport, message.Timestamp)
		if err != nil {
			return err
		}
		if err := a.upsertModeration(message.TargetPubkey, "SHADOW_BAN", message.AdminPubkey, message.Timestamp, lamport, message.Reason); err != nil {
			return err
		}
		if localID, liErr := a.getLocalIdentity(); liErr == nil && strings.TrimSpace(localID.PublicKey) == strings.TrimSpace(message.TargetPubkey) {
			a.tryGenerateNotification(NotifTypeGovernance, message.AdminPubkey, strings.TrimSpace(message.TargetPubkey), "user", "", message.Timestamp)
		}
		return nil
	case "UNBAN":
		trusted, err := a.isTrustedAdmin(message.AdminPubkey)
		if err != nil {
			return err
		}
		if !trusted {
			return errors.New("admin pubkey is not trusted")
		}
		if message.Timestamp == 0 {
			message.Timestamp = time.Now().Unix()
		}
		lamport, err := a.normalizeIncomingLamport(message.Lamport, message.Timestamp)
		if err != nil {
			return err
		}
		if err := a.upsertModeration(message.TargetPubkey, "UNBAN", message.AdminPubkey, message.Timestamp, lamport, message.Reason); err != nil {
			return err
		}
		if localID, liErr := a.getLocalIdentity(); liErr == nil && strings.TrimSpace(localID.PublicKey) == strings.TrimSpace(message.TargetPubkey) {
			a.tryGenerateNotification(NotifTypeGovernance, message.AdminPubkey, strings.TrimSpace(message.TargetPubkey), "user", "", message.Timestamp)
		}
		return nil
	case "POST":
		if scope := strings.TrimSpace(strings.ToLower(message.AuthScope)); scope != "" && scope != authScopeUser {
			return errors.New("invalid post auth scope")
		}
		if strings.TrimSpace(message.Pubkey) == "" {
			return errors.New("invalid post payload")
		}
		if message.Timestamp == 0 {
			message.Timestamp = time.Now().Unix()
		}
		lamport, err := a.normalizeIncomingLamport(message.Lamport, message.Timestamp)
		if err != nil {
			return err
		}
		message.Lamport = lamport

		if strings.TrimSpace(message.DisplayName) != "" || strings.TrimSpace(message.AvatarURL) != "" {
			if _, err := a.upsertProfile(message.Pubkey, message.DisplayName, message.AvatarURL, message.Timestamp); err != nil {
				return err
			}
		}

		body := strings.TrimSpace(message.Body)
		if body == "" {
			body = strings.TrimSpace(message.Content)
		}

		title := strings.TrimSpace(message.Title)

		if title == "" || body == "" {
			return errors.New("invalid post payload")
		}

		viewerPubkey := ""
		if identity, idErr := a.getLocalIdentity(); idErr == nil {
			viewerPubkey = strings.TrimSpace(identity.PublicKey)
		}
		allowed, err := a.shouldAcceptPublicContent(message.Pubkey, message.Lamport, message.Timestamp, message.ID, viewerPubkey)
		if err != nil {
			return err
		}
		if !allowed {
			return nil
		}

		if strings.TrimSpace(message.ID) == "" {
			seed := fmt.Sprintf("%s|%s|%d", title, body, message.Lamport)
			message.ID = buildMessageID(message.Pubkey, seed, message.Timestamp)
		}

		insertedMessage, err := a.insertMessage(ForumMessage{
			ID:          message.ID,
			Pubkey:      message.Pubkey,
			OpID:        resolveOperationID(message.OpID, message.ID, message.Pubkey, message.Lamport, normalizeOperationType(message.OpType, postOpTypeCreate)),
			Title:       title,
			Body:        body,
			ContentCID:  strings.TrimSpace(message.ContentCID),
			ImageCID:    strings.TrimSpace(message.ImageCID),
			ThumbCID:    strings.TrimSpace(message.ThumbCID),
			ImageMIME:   strings.TrimSpace(message.ImageMIME),
			ImageSize:   message.ImageSize,
			ImageWidth:  message.ImageWidth,
			ImageHeight: message.ImageHeight,
			Content:     "",
			Score:       0,
			Timestamp:   message.Timestamp,
			Lamport:     message.Lamport,
			SizeBytes:   int64(len([]byte(body))),
			Zone:        "public",
			SubID:       normalizeSubID(message.SubID),
			Visibility:  "normal",
			IsProtected: 0,
		})
		if err != nil {
			return err
		}

		a.emitSubscribedSubUpdate(insertedMessage)
		return nil
	default:
		return fmt.Errorf("unsupported message type: %s", message.Type)
	}
}

func (a *App) UpvotePost(postID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return err
	}

	return a.applyPostUpvote(identity.PublicKey, postID, generateOperationID(postID, identity.PublicKey, time.Now().UnixNano()))
}

func (a *App) DownvotePost(postID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return err
	}

	return a.applyPostDownvote(identity.PublicKey, postID, generateOperationID(postID, identity.PublicKey, time.Now().UnixNano()))
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

func (a *App) upsertProfile(pubkey string, displayName string, avatarURL string, updatedAt int64) (Profile, error) {
	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return Profile{}, errors.New("pubkey is required")
	}

	displayName = strings.TrimSpace(displayName)
	avatarURL = strings.TrimSpace(avatarURL)
	if len([]rune(displayName)) > 64 {
		displayName = string([]rune(displayName)[:64])
	}

	if updatedAt <= 0 {
		updatedAt = time.Now().Unix()
	}

	_, err := a.db.Exec(`
		INSERT INTO profiles (pubkey, display_name, avatar_url, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(pubkey) DO UPDATE SET
			display_name = excluded.display_name,
			avatar_url = excluded.avatar_url,
			updated_at = excluded.updated_at;
	`, pubkey, displayName, avatarURL, updatedAt)
	if err != nil {
		return Profile{}, err
	}

	return Profile{Pubkey: pubkey, DisplayName: displayName, AvatarURL: avatarURL, UpdatedAt: updatedAt}, nil
}

func prepareImageAssets(source []byte, hintMIME string) (mainBytes []byte, mainMIME string, width int, height int, thumbBytes []byte, thumbMIME string, thumbWidth int, thumbHeight int, err error) {
	hintMIME = strings.TrimSpace(strings.ToLower(hintMIME))
	decoded, format, decodeErr := image.Decode(bytes.NewReader(source))
	if decodeErr != nil {
		fallbackMIME := hintMIME
		if fallbackMIME == "" {
			fallbackMIME = "application/octet-stream"
		}
		return source, fallbackMIME, 0, 0, nil, "", 0, 0, nil
	}

	bounds := decoded.Bounds()
	width = bounds.Dx()
	height = bounds.Dy()

	mainMIME = normalizedImageMIME(hintMIME, format)

	compressedImage := resizeImageIfNeeded(decoded, 1920)
	compressedBytes, compressedMIME, encodeErr := encodeImageForStorage(compressedImage, mainMIME)
	if encodeErr != nil {
		return nil, "", 0, 0, nil, "", 0, 0, encodeErr
	}

	thumbImage := resizeImageIfNeeded(decoded, 320)
	thumbWidth = thumbImage.Bounds().Dx()
	thumbHeight = thumbImage.Bounds().Dy()
	thumbBytes, thumbMIME, err = encodeImageForStorage(thumbImage, "image/jpeg")
	if err != nil {
		return nil, "", 0, 0, nil, "", 0, 0, err
	}

	return compressedBytes, compressedMIME, width, height, thumbBytes, thumbMIME, thumbWidth, thumbHeight, nil
}

func normalizedImageMIME(hint string, format string) string {
	hint = strings.TrimSpace(strings.ToLower(hint))
	if strings.HasPrefix(hint, "image/") {
		return hint
	}

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

func encodeImageForStorage(img image.Image, preferredMIME string) ([]byte, string, error) {
	preferredMIME = strings.TrimSpace(strings.ToLower(preferredMIME))
	if preferredMIME == "image/png" && !hasTransparency(img) {
		preferredMIME = "image/jpeg"
	}

	var buffer bytes.Buffer
	switch preferredMIME {
	case "image/png":
		encoder := png.Encoder{CompressionLevel: png.BestSpeed}
		if err := encoder.Encode(&buffer, img); err != nil {
			return nil, "", err
		}
		return buffer.Bytes(), "image/png", nil
	default:
		if err := jpeg.Encode(&buffer, img, &jpeg.Options{Quality: 82}); err != nil {
			return nil, "", err
		}
		return buffer.Bytes(), "image/jpeg", nil
	}
}

func resizeImageIfNeeded(src image.Image, maxEdge int) image.Image {
	if maxEdge <= 0 {
		return src
	}

	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= maxEdge && height <= maxEdge {
		return src
	}

	scale := float64(maxEdge) / float64(width)
	if height > width {
		scale = float64(maxEdge) / float64(height)
	}
	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)
	if newWidth < 1 {
		newWidth = 1
	}
	if newHeight < 1 {
		newHeight = 1
	}

	destination := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	xdraw.ApproxBiLinear.Scale(destination, destination.Bounds(), src, bounds, xdraw.Over, nil)
	return destination
}

func hasTransparency(img image.Image) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, alpha := img.At(x, y).RGBA()
			if alpha < 0xffff {
				return true
			}
		}
	}
	return false
}

func (a *App) ApplyShadowBan(targetPubkey string, adminPubkey string, reason string) error {
	trusted, err := a.isTrustedAdmin(adminPubkey)
	if err != nil {
		return err
	}
	if !trusted {
		return errors.New("admin pubkey is not trusted")
	}

	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return err
	}
	return a.upsertModeration(targetPubkey, "SHADOW_BAN", adminPubkey, now, lamport, reason)
}

func (a *App) ApplyUnban(targetPubkey string, adminPubkey string, reason string) error {
	trusted, err := a.isTrustedAdmin(adminPubkey)
	if err != nil {
		return err
	}
	if !trusted {
		return errors.New("admin pubkey is not trusted")
	}

	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return err
	}
	return a.upsertModeration(targetPubkey, "UNBAN", adminPubkey, now, lamport, reason)
}

func (a *App) AddTrustedAdmin(pubkey string, role string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return errors.New("admin pubkey is required")
	}

	role = strings.TrimSpace(strings.ToLower(role))
	if role == "" {
		role = "appointed"
	}

	_, err := a.db.Exec(`
		INSERT INTO governance_admins (admin_pubkey, role, active)
		VALUES (?, ?, 1)
		ON CONFLICT(admin_pubkey) DO UPDATE SET
			role = excluded.role,
			active = 1;
	`, pubkey, role)
	return err
}

func (a *App) GetTrustedAdmins() ([]GovernanceAdmin, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	rows, err := a.db.Query(`
		SELECT admin_pubkey, role, active
		FROM governance_admins
		WHERE active = 1
		ORDER BY role, admin_pubkey;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]GovernanceAdmin, 0)
	for rows.Next() {
		var admin GovernanceAdmin
		var active int
		if err = rows.Scan(&admin.AdminPubkey, &admin.Role, &active); err != nil {
			return nil, err
		}
		admin.Active = active == 1
		result = append(result, admin)
	}

	return result, rows.Err()
}

func (a *App) isTrustedAdmin(pubkey string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return false, nil
	}

	var count int
	err := a.db.QueryRow(`
		SELECT COUNT(1)
		FROM governance_admins
		WHERE admin_pubkey = ? AND active = 1;
	`, pubkey).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (a *App) applyPostUpvote(voterPubkey string, postID string, opID string) error {
	current, err := a.getPostVoteState(voterPubkey, postID)
	if err != nil {
		return err
	}
	target := voteStateUp
	if current == voteStateUp {
		target = voteStateNone
	}
	return a.applyPostVoteState(voterPubkey, postID, target, opID)
}

func (a *App) applyPostDownvote(voterPubkey string, postID string, opID string) error {
	current, err := a.getPostVoteState(voterPubkey, postID)
	if err != nil {
		return err
	}
	target := voteStateDown
	if current == voteStateDown {
		target = voteStateNone
	}
	return a.applyPostVoteState(voterPubkey, postID, target, opID)
}

func (a *App) currentPostVoteStateTx(tx *sql.Tx, voterPubkey string, postID string) (string, error) {
	var exists int
	err := tx.QueryRow(`SELECT 1 FROM post_votes WHERE post_id = ? AND voter_pubkey = ? LIMIT 1;`, postID, voterPubkey).Scan(&exists)
	if err == nil {
		return voteStateUp, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return voteStateNone, err
	}
	err = tx.QueryRow(`SELECT 1 FROM post_downvotes WHERE post_id = ? AND voter_pubkey = ? LIMIT 1;`, postID, voterPubkey).Scan(&exists)
	if err == nil {
		return voteStateDown, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return voteStateNone, err
	}
	return voteStateNone, nil
}

func (a *App) getPostVoteState(voterPubkey string, postID string) (string, error) {
	voterPubkey = strings.TrimSpace(voterPubkey)
	postID = strings.TrimSpace(postID)
	if voterPubkey == "" || postID == "" {
		return voteStateNone, errors.New("voter pubkey and post id are required")
	}
	tx, err := a.db.Begin()
	if err != nil {
		return voteStateNone, err
	}
	defer func() { _ = tx.Rollback() }()
	state, err := a.currentPostVoteStateTx(tx, voterPubkey, postID)
	if err != nil {
		return voteStateNone, err
	}
	if err = tx.Commit(); err != nil {
		return voteStateNone, err
	}
	return state, nil
}

func voteDelta(before string, after string) int64 {
	value := func(state string) int64 {
		switch normalizeVoteState(state) {
		case voteStateUp:
			return 1
		case voteStateDown:
			return -1
		default:
			return 0
		}
	}
	return value(after) - value(before)
}

func (a *App) applyPostVoteState(voterPubkey string, postID string, targetState string, opID string) error {
	targetState = normalizeVoteState(targetState)
	voterPubkey = strings.TrimSpace(voterPubkey)
	postID = strings.TrimSpace(postID)
	opID = strings.TrimSpace(opID)
	if voterPubkey == "" || postID == "" {
		return errors.New("voter pubkey and post id are required")
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

	current, err := a.currentPostVoteStateTx(tx, voterPubkey, postID)
	if err != nil {
		return err
	}
	if current == targetState {
		return tx.Commit()
	}
	if _, err = tx.Exec(`DELETE FROM post_votes WHERE post_id = ? AND voter_pubkey = ?;`, postID, voterPubkey); err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM post_downvotes WHERE post_id = ? AND voter_pubkey = ?;`, postID, voterPubkey); err != nil {
		return err
	}
	switch targetState {
	case voteStateUp:
		if _, err = tx.Exec(`INSERT INTO post_votes (post_id, voter_pubkey, timestamp) VALUES (?, ?, ?);`, postID, voterPubkey, time.Now().Unix()); err != nil {
			return err
		}
	case voteStateDown:
		if _, err = tx.Exec(`INSERT INTO post_downvotes (post_id, voter_pubkey, timestamp) VALUES (?, ?, ?);`, postID, voterPubkey, time.Now().Unix()); err != nil {
			return err
		}
	}
	delta := voteDelta(current, targetState)
	if delta != 0 {
		result, err := tx.Exec(`UPDATE messages SET score = score + ? WHERE id = ?;`, delta, postID)
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.New("post not found")
		}
	}
	return tx.Commit()
}

func ensureZoneQuota(tx *sql.Tx, zone string, quota int64, incomingBytes int64) error {
	if incomingBytes > quota {
		return errors.New("message exceeds zone quota")
	}

	for {
		var used int64
		if err := tx.QueryRow(`SELECT COALESCE(SUM(size_bytes), 0) FROM messages WHERE zone = ?;`, zone).Scan(&used); err != nil {
			return err
		}

		if used+incomingBytes <= quota {
			return nil
		}

		result, err := tx.Exec(`
			DELETE FROM messages
			WHERE id IN (
				SELECT id
				FROM messages
				WHERE zone = ? AND is_protected = 0
				ORDER BY timestamp ASC
				LIMIT 1
			);
		`, zone)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return errors.New("quota exceeded and no evictable records")
		}
	}
}

func ensureBlobQuotaWithLRU(tx *sql.Tx, zone string, quota int64, incomingBytes int64, incomingContentCID string) error {
	if incomingBytes > quota {
		return errors.New("content blob exceeds zone quota")
	}

	incomingContentCID = strings.TrimSpace(incomingContentCID)
	if incomingContentCID == "" {
		return errors.New("content cid is required")
	}

	var existingCount int
	if err := tx.QueryRow(`
		SELECT COUNT(1)
		FROM content_blobs cb
		WHERE cb.content_cid = ?
			AND EXISTS (
				SELECT 1 FROM messages m
				WHERE m.zone = ? AND m.content_cid = cb.content_cid
			);
	`, incomingContentCID, zone).Scan(&existingCount); err != nil {
		return err
	}

	effectiveIncoming := incomingBytes
	if existingCount > 0 {
		effectiveIncoming = 0
	}

	var total int64
	if err := tx.QueryRow(`
		SELECT COALESCE(SUM(cb.size_bytes), 0)
		FROM content_blobs cb
		WHERE EXISTS (
			SELECT 1 FROM messages m
			WHERE m.zone = ? AND m.content_cid = cb.content_cid
		);
	`, zone).Scan(&total); err != nil {
		return err
	}

	if total+effectiveIncoming <= quota {
		return nil
	}

	rows, err := tx.Query(`
		SELECT cb.content_cid, cb.size_bytes
		FROM content_blobs cb
		WHERE cb.pinned = 0
			AND EXISTS (
				SELECT 1 FROM messages m
				WHERE m.zone = ? AND m.content_cid = cb.content_cid
			)
		ORDER BY cb.last_accessed_at ASC, cb.created_at ASC;
	`, zone)
	if err != nil {
		return err
	}
	defer rows.Close()

	type candidate struct {
		cid  string
		size int64
	}
	candidates := make([]candidate, 0)
	for rows.Next() {
		var c candidate
		if scanErr := rows.Scan(&c.cid, &c.size); scanErr != nil {
			return scanErr
		}
		candidates = append(candidates, c)
	}
	if err = rows.Err(); err != nil {
		return err
	}

	for _, c := range candidates {
		if total+effectiveIncoming <= quota {
			return nil
		}
		if c.cid == incomingContentCID {
			continue
		}

		var zoneCount int
		if err = tx.QueryRow(`SELECT COUNT(DISTINCT zone) FROM messages WHERE content_cid = ?;`, c.cid).Scan(&zoneCount); err != nil {
			return err
		}
		if zoneCount > 1 {
			continue
		}

		result, delErr := tx.Exec(`DELETE FROM content_blobs WHERE content_cid = ? AND pinned = 0;`, c.cid)
		if delErr != nil {
			return delErr
		}
		affected, affectedErr := result.RowsAffected()
		if affectedErr != nil {
			return affectedErr
		}
		if affected > 0 {
			total -= c.size
		}
	}

	if total+effectiveIncoming > quota {
		return errors.New("content blob quota exceeded and no evictable records")
	}

	return nil
}

func (a *App) upsertModeration(targetPubkey string, action string, sourceAdmin string, timestamp int64, lamport int64, reason string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	action = strings.ToUpper(strings.TrimSpace(action))
	if action != "SHADOW_BAN" && action != "UNBAN" {
		return errors.New("invalid moderation action")
	}

	if strings.TrimSpace(targetPubkey) == "" || strings.TrimSpace(sourceAdmin) == "" {
		return errors.New("invalid moderation payload")
	}

	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}
	if lamport <= 0 {
		lamport = timestamp
	}

	var existingTimestamp int64
	err := a.db.QueryRow(`
		SELECT timestamp
		FROM moderation
		WHERE target_pubkey = ?;
	`, targetPubkey).Scan(&existingTimestamp)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if err == nil && existingTimestamp > timestamp {
		if _, logErr := a.db.Exec(`
			INSERT INTO moderation_logs (target_pubkey, action, source_admin, timestamp, lamport, reason, result)
			VALUES (?, ?, ?, ?, ?, ?, 'ignored_older');
		`, targetPubkey, action, sourceAdmin, timestamp, lamport, reason); logErr != nil {
			return logErr
		}
		return nil
	}

	_, err = a.db.Exec(`
		INSERT INTO moderation (target_pubkey, action, source_admin, timestamp, lamport, reason)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(target_pubkey) DO UPDATE SET
			action = excluded.action,
			source_admin = excluded.source_admin,
			timestamp = excluded.timestamp,
			lamport = excluded.lamport,
			reason = excluded.reason;
	`, targetPubkey, action, sourceAdmin, timestamp, lamport, reason)
	if err != nil {
		return err
	}

	if _, err = a.db.Exec(`
		INSERT INTO moderation_logs (target_pubkey, action, source_admin, timestamp, lamport, reason, result)
		VALUES (?, ?, ?, ?, ?, ?, 'applied');
	`, targetPubkey, action, sourceAdmin, timestamp, lamport, reason); err != nil {
		return err
	}

	policy, err := a.GetGovernancePolicy()
	if err != nil {
		return err
	}

	if action == "SHADOW_BAN" && policy.HideHistoryOnShadowBan {
		if _, err = a.db.Exec(`
			UPDATE messages
			SET visibility = 'shadowed'
			WHERE pubkey = ? AND zone = 'public';
		`, targetPubkey); err != nil {
			return err
		}
	}

	if action == "UNBAN" {
		if _, err = a.db.Exec(`
			UPDATE messages
			SET visibility = 'normal'
			WHERE pubkey = ? AND zone = 'public';
		`, targetPubkey); err != nil {
			return err
		}
	}

	state := "normal"
	if action == "SHADOW_BAN" {
		state = "shadow_banned"
	}

	_, err = a.db.Exec(`
		INSERT INTO identity_state (pubkey, state, storage_commit_bytes, public_quota_bytes, private_quota_bytes, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(pubkey) DO UPDATE SET
			state = excluded.state,
			storage_commit_bytes = excluded.storage_commit_bytes,
			public_quota_bytes = excluded.public_quota_bytes,
			private_quota_bytes = excluded.private_quota_bytes,
			updated_at = excluded.updated_at;
	`, targetPubkey, state, totalQuotaBytes, publicQuotaBytes, privateQuotaBytes, time.Now().Unix())

	return err
}

func (a *App) isShadowBanned(pubkey string) (bool, error) {
	if a.db == nil {
		return false, errors.New("database not initialized")
	}

	var action string
	err := a.db.QueryRow(`SELECT action FROM moderation WHERE target_pubkey = ?;`, pubkey).Scan(&action)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return strings.ToUpper(action) == "SHADOW_BAN", nil
}

func (a *App) getModerationSnapshot(pubkey string) (string, int64, int64, string, error) {
	if a.db == nil {
		return "", 0, 0, "", errors.New("database not initialized")
	}

	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" {
		return "", 0, 0, "", nil
	}

	var action string
	var timestamp int64
	var lamport int64
	var sourceAdmin string
	err := a.db.QueryRow(`
		SELECT action, timestamp, lamport, source_admin
		FROM moderation
		WHERE target_pubkey = ?;
	`, pubkey).Scan(&action, &timestamp, &lamport, &sourceAdmin)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, 0, "", nil
	}
	if err != nil {
		return "", 0, 0, "", err
	}

	return strings.ToUpper(strings.TrimSpace(action)), timestamp, lamport, strings.TrimSpace(sourceAdmin), nil
}

func (a *App) shouldAcceptPublicContent(authorPubkey string, contentLamport int64, contentTimestamp int64, contentID string, viewerPubkey string) (bool, error) {
	authorPubkey = strings.TrimSpace(authorPubkey)
	contentID = strings.TrimSpace(contentID)
	viewerPubkey = strings.TrimSpace(viewerPubkey)
	if authorPubkey == "" {
		return false, errors.New("author pubkey is required")
	}

	if viewerPubkey != "" && authorPubkey == viewerPubkey {
		return true, nil
	}

	action, moderationTimestamp, moderationLamport, moderationAdmin, err := a.getModerationSnapshot(authorPubkey)
	if err != nil {
		return false, err
	}
	if action != "SHADOW_BAN" {
		return true, nil
	}

	policy, err := a.GetGovernancePolicy()
	if err != nil {
		return false, err
	}
	if policy.HideHistoryOnShadowBan {
		return false, nil
	}

	if contentTimestamp <= 0 {
		contentTimestamp = time.Now().Unix()
	}
	if contentLamport <= 0 {
		contentLamport = contentTimestamp
	}

	if moderationLamport > 0 && contentLamport > 0 {
		contentVersion := LamportVersion{
			Lamport: contentLamport,
			Author:  authorPubkey,
			OpID:    contentID,
		}
		moderationVersion := LamportVersion{
			Lamport: moderationLamport,
			Author:  moderationAdmin,
			OpID:    fmt.Sprintf("moderation|%s|%d|%s", authorPubkey, moderationTimestamp, action),
		}
		return compareLamportVersion(contentVersion, moderationVersion) < 0, nil
	}

	return contentTimestamp < moderationTimestamp, nil
}

func buildMessageID(pubkey string, content string, timestamp int64) string {
	raw := fmt.Sprintf("%s|%d|%s", pubkey, timestamp, content)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func buildContentCID(body string) string {
	trimmed := strings.TrimSpace(body)
	hash := sha256.Sum256([]byte(trimmed))
	return "cidv1-" + hex.EncodeToString(hash[:])
}

func buildBinaryCID(data []byte) string {
	hash := sha256.Sum256(data)
	return "cidv1-bin-" + hex.EncodeToString(hash[:])
}

func deriveTitle(body string) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return ""
	}

	runes := []rune(trimmed)
	if len(runes) <= 20 {
		return string(runes)
	}

	return string(runes[:20])
}

func deriveBodyPreview(body string, maxRunes int) string {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return ""
	}
	if maxRunes <= 0 {
		maxRunes = 180
	}
	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes])
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

func normalizeSearchLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func normalizeMyPostsLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func encodeMyPostsCursor(timestamp int64, postID string) string {
	raw := fmt.Sprintf("%d|%s", timestamp, strings.TrimSpace(postID))
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func decodeMyPostsCursor(cursor string) (int64, string, error) {
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		return 0, "", nil
	}

	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, "", errors.New("invalid my posts cursor")
	}

	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return 0, "", errors.New("invalid my posts cursor")
	}

	timestamp, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	if err != nil || timestamp <= 0 {
		return 0, "", errors.New("invalid my posts cursor")
	}
	postID := strings.TrimSpace(parts[1])
	if postID == "" {
		return 0, "", errors.New("invalid my posts cursor")
	}

	return timestamp, postID, nil
}

func normalizeFeedStreamLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func normalizeFeedStreamAlgorithm(algorithm string) string {
	normalized := strings.TrimSpace(strings.ToLower(algorithm))
	if normalized == "" {
		return "hot-v1"
	}
	return normalized
}

func scoreFeedRecommendation(message ForumMessage, now int64, algorithm string) float64 {
	switch strings.TrimSpace(strings.ToLower(algorithm)) {
	case "hot-v1":
		return computeHotScore(message.Score, message.Timestamp, now)
	default:
		return computeHotScore(message.Score, message.Timestamp, now)
	}
}

func countFeedItemsByReason(items []FeedStreamItem, reason string) int {
	total := 0
	for _, item := range items {
		if item.Reason == reason {
			total++
		}
	}
	return total
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

func (a *App) queryForumMessages(query string, args ...interface{}) ([]ForumMessage, error) {
	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]ForumMessage, 0)
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

func makeSQLPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}

	parts := make([]string, count)
	for i := range count {
		parts[i] = "?"
	}
	return strings.Join(parts, ",")
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

func (a *App) saveLocalIdentity(identity Identity) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	mnemonic := strings.TrimSpace(identity.Mnemonic)
	pubkey := strings.TrimSpace(identity.PublicKey)
	if mnemonic == "" || pubkey == "" {
		return errors.New("identity is incomplete")
	}

	_, err := a.db.Exec(`
		INSERT INTO local_identity (id, mnemonic, pubkey, updated_at)
		VALUES (1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			mnemonic = excluded.mnemonic,
			pubkey = excluded.pubkey,
			updated_at = excluded.updated_at;
	`, mnemonic, pubkey, time.Now().Unix())

	return err
}

func (a *App) getLocalIdentity() (Identity, error) {
	if a.db == nil {
		return Identity{}, errors.New("database not initialized")
	}

	var identity Identity
	err := a.db.QueryRow(`SELECT mnemonic, pubkey FROM local_identity WHERE id = 1;`).Scan(&identity.Mnemonic, &identity.PublicKey)
	if errors.Is(err, sql.ErrNoRows) {
		return Identity{}, errors.New("identity not found")
	}
	if err != nil {
		return Identity{}, err
	}

	return identity, nil
}

