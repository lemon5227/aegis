package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type OutgoingMessageRecord struct {
	ID           string `json:"id"`
	MessageType  string `json:"messageType"`
	PayloadJSON  string `json:"payloadJson"`
	CreatedAt    int64  `json:"createdAt"`
	AvailableAt  int64  `json:"availableAt"`
	AttemptCount int    `json:"attemptCount"`
	LastError    string `json:"lastError"`
}

func normalizeOutboxMessageID(message IncomingMessage) string {
	if opID := strings.TrimSpace(message.OpID); opID != "" {
		return opID
	}
	if favoriteOpID := strings.TrimSpace(message.FavoriteOpID); favoriteOpID != "" {
		return favoriteOpID
	}
	return ""
}

func (a *App) enqueueOutgoingMessage(messageType string, message IncomingMessage) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	messageType = strings.ToUpper(strings.TrimSpace(messageType))
	messageID := normalizeOutboxMessageID(message)
	if messageType == "" || messageID == "" {
		return errors.New("outbox message id is required")
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	_, err = a.db.Exec(`
		INSERT INTO message_outbox (id, message_type, payload_json, created_at, available_at, attempt_count, last_error)
		VALUES (?, ?, ?, ?, ?, 0, '')
		ON CONFLICT(id) DO NOTHING;
	`, messageID, messageType, string(payload), now, now)
	return err
}

func (a *App) deleteOutgoingMessage(messageID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return errors.New("outbox message id is required")
	}
	_, err := a.db.Exec(`DELETE FROM message_outbox WHERE id = ?;`, messageID)
	return err
}

func nextOutboxAttemptAt(attemptCount int) int64 {
	if attemptCount < 0 {
		attemptCount = 0
	}
	delay := time.Second
	for i := 0; i < attemptCount && delay < 30*time.Second; i++ {
		delay *= 2
	}
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	return time.Now().Add(delay).Unix()
}

func (a *App) markOutgoingMessageRetry(messageID string, attemptCount int, err error) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return errors.New("outbox message id is required")
	}
	lastError := ""
	if err != nil {
		lastError = strings.TrimSpace(err.Error())
	}
	_, execErr := a.db.Exec(`
		UPDATE message_outbox
		SET attempt_count = ?, available_at = ?, last_error = ?
		WHERE id = ?;
	`, attemptCount+1, nextOutboxAttemptAt(attemptCount), lastError, messageID)
	return execErr
}

func (a *App) listDueOutgoingMessages(limit int) ([]OutgoingMessageRecord, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}
	if limit <= 0 || limit > 256 {
		limit = 64
	}

	rows, err := a.db.Query(`
		SELECT id, message_type, payload_json, created_at, available_at, attempt_count, last_error
		FROM message_outbox
		WHERE available_at <= ?
		ORDER BY created_at ASC
		LIMIT ?;
	`, time.Now().Unix(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]OutgoingMessageRecord, 0, limit)
	for rows.Next() {
		var record OutgoingMessageRecord
		if scanErr := rows.Scan(
			&record.ID,
			&record.MessageType,
			&record.PayloadJSON,
			&record.CreatedAt,
			&record.AvailableAt,
			&record.AttemptCount,
			&record.LastError,
		); scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (a *App) getPublishTopicState() (context.Context, *pubsub.Topic, int) {
	a.p2pMu.Lock()
	defer a.p2pMu.Unlock()

	if a.p2pCtx == nil || a.p2pTopic == nil || a.p2pHost == nil {
		return nil, nil, 0
	}
	return a.p2pCtx, a.p2pTopic, len(a.p2pHost.Network().Peers())
}

func (a *App) flushOutgoingMessages() error {
	a.outboxFlushMu.Lock()
	defer a.outboxFlushMu.Unlock()

	records, err := a.listDueOutgoingMessages(64)
	if err != nil || len(records) == 0 {
		return err
	}

	baseCtx, topic, peerCount := a.getPublishTopicState()
	if baseCtx == nil || topic == nil || peerCount == 0 {
		for _, record := range records {
			_ = a.markOutgoingMessageRetry(record.ID, record.AttemptCount, errors.New("network unavailable"))
		}
		return nil
	}

	for _, record := range records {
		publishCtx, cancel := context.WithTimeout(baseCtx, 3*time.Second)
		publishErr := topic.Publish(publishCtx, []byte(record.PayloadJSON))
		cancel()
		if publishErr != nil {
			_ = a.markOutgoingMessageRetry(record.ID, record.AttemptCount, publishErr)
			if a.ctx != nil {
				runtime.LogWarningf(a.ctx, "outbox publish failed id=%s type=%s err=%v", record.ID, record.MessageType, publishErr)
			}
			continue
		}
		if err = a.deleteOutgoingMessage(record.ID); err != nil && a.ctx != nil {
			runtime.LogWarningf(a.ctx, "outbox cleanup failed id=%s err=%v", record.ID, err)
		}
	}

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "p2p:updated")
	}
	return nil
}

func (a *App) flushOutgoingMessagesAsync() {
	go func() {
		if err := a.flushOutgoingMessages(); err != nil && a.ctx != nil && !errors.Is(err, sql.ErrNoRows) {
			runtime.LogWarningf(a.ctx, "outbox flush failed: %v", err)
		}
	}()
}

func (a *App) queueOutgoingMessage(messageType string, message IncomingMessage) error {
	if err := a.enqueueOutgoingMessage(messageType, message); err != nil {
		return err
	}
	a.flushOutgoingMessagesAsync()
	return nil
}

func (a *App) runOutboxWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	a.flushOutgoingMessagesAsync()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.flushOutgoingMessages(); err != nil && a.ctx != nil {
				runtime.LogWarningf(a.ctx, "outbox worker flush failed: %v", err)
			}
		}
	}
}
