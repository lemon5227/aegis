package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const (
	messageTypeSubSettingsUpdate = "SUB_SETTINGS_UPDATE"
	messageTypePostPinSet        = "POST_PIN_SET"
	messageTypePostLockSet       = "POST_LOCK_SET"
)

func normalizeSubRules(rules []string) []string {
	result := make([]string, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		trimmed := strings.TrimSpace(rule)
		if trimmed == "" {
			continue
		}
		if len([]rune(trimmed)) > 240 {
			trimmed = string([]rune(trimmed)[:240])
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
		if len(result) >= 12 {
			break
		}
	}
	return result
}

func encodeSubRulesJSON(rules []string) (string, error) {
	normalized := normalizeSubRules(rules)
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func decodeSubRulesJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var decoded []string
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil
	}
	return normalizeSubRules(decoded)
}

func (a *App) GetSubSettings(subID string) (SubSettings, error) {
	if a.db == nil {
		return SubSettings{}, errors.New("database not initialized")
	}

	subID = normalizeSubID(subID)
	settings := SubSettings{SubID: subID, Rules: []string{}}

	var rulesJSON string
	err := a.db.QueryRow(`
		SELECT rules_json, announcement, updated_at
		FROM sub_settings
		WHERE sub_id = ?;
	`, subID).Scan(&rulesJSON, &settings.Announcement, &settings.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return settings, nil
	}
	if err != nil {
		return SubSettings{}, err
	}
	settings.Rules = decodeSubRulesJSON(rulesJSON)
	return settings, nil
}

func (a *App) applySubSettingsUpdate(subID string, rules []string, announcement string, adminPubkey string, timestamp int64, lamport int64, opID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	subID = normalizeSubID(subID)
	adminPubkey = strings.TrimSpace(adminPubkey)
	announcement = strings.TrimSpace(announcement)
	if subID == "" || adminPubkey == "" {
		return errors.New("invalid sub settings payload")
	}
	if len([]rune(announcement)) > 400 {
		announcement = string([]rune(announcement)[:400])
	}
	if timestamp <= 0 {
		timestamp = time.Now().Unix()
	}
	if lamport <= 0 {
		lamport = timestamp
	}
	opID = resolveOperationID(opID, subID, adminPubkey, lamport, postOpTypeUpdate)

	rules = normalizeSubRules(rules)
	rulesJSON, err := encodeSubRulesJSON(rules)
	if err != nil {
		return err
	}

	var currentUpdatedAt int64
	var currentLamport int64
	var currentAdmin string
	var currentOpID string
	err = a.db.QueryRow(`
		SELECT updated_at, lamport, current_admin_pubkey, current_op_id
		FROM sub_settings
		WHERE sub_id = ?;
	`, subID).Scan(&currentUpdatedAt, &currentLamport, &currentAdmin, &currentOpID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		currentVersion := LamportVersion{Lamport: currentLamport, Author: strings.TrimSpace(currentAdmin), OpID: strings.TrimSpace(currentOpID)}
		incomingVersion := LamportVersion{Lamport: lamport, Author: adminPubkey, OpID: opID}
		if compareLamportVersion(incomingVersion, currentVersion) <= 0 {
			return nil
		}
	}

	_, err = a.db.Exec(`
		INSERT INTO sub_settings (sub_id, rules_json, announcement, updated_at, current_admin_pubkey, current_op_id, lamport)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(sub_id) DO UPDATE SET
			rules_json = excluded.rules_json,
			announcement = excluded.announcement,
			updated_at = excluded.updated_at,
			current_admin_pubkey = excluded.current_admin_pubkey,
			current_op_id = excluded.current_op_id,
			lamport = excluded.lamport;
	`, subID, rulesJSON, announcement, timestamp, adminPubkey, opID, lamport)
	return err
}

func (a *App) currentAdminIdentity() (Identity, error) {
	identity, err := a.getLocalIdentity()
	if err != nil {
		return Identity{}, err
	}
	trusted, err := a.isTrustedAdmin(identity.PublicKey)
	if err != nil {
		return Identity{}, err
	}
	if !trusted {
		return Identity{}, errors.New("admin pubkey is not trusted")
	}
	return identity, nil
}

func (a *App) UpdateSubSettings(subID string, rules []string, announcement string) (SubSettings, error) {
	identity, err := a.currentAdminIdentity()
	if err != nil {
		return SubSettings{}, err
	}
	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return SubSettings{}, err
	}
	opID := generateOperationID(subID, identity.PublicKey, lamport)
	if err = a.applySubSettingsUpdate(subID, rules, announcement, identity.PublicKey, now, lamport, opID); err != nil {
		return SubSettings{}, err
	}
	return a.GetSubSettings(subID)
}

func (a *App) applyPostPinnedState(postID string, pinned bool, adminPubkey string, timestamp int64, lamport int64, opID string) error {
	return a.applyPostAdminState(postID, "pinned", pinned, adminPubkey, timestamp, lamport, opID)
}

func (a *App) applyPostLockedState(postID string, locked bool, adminPubkey string, timestamp int64, lamport int64, opID string) error {
	return a.applyPostAdminState(postID, "locked", locked, adminPubkey, timestamp, lamport, opID)
}

func (a *App) applyPostAdminState(postID string, field string, enabled bool, adminPubkey string, timestamp int64, lamport int64, opID string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}
	postID = strings.TrimSpace(postID)
	adminPubkey = strings.TrimSpace(adminPubkey)
	field = strings.ToLower(strings.TrimSpace(field))
	if postID == "" || adminPubkey == "" {
		return errors.New("invalid post admin payload")
	}
	if field != "pinned" && field != "locked" {
		return errors.New("invalid post admin field")
	}
	if timestamp <= 0 {
		timestamp = time.Now().Unix()
	}
	if lamport <= 0 {
		lamport = timestamp
	}
	opID = resolveOperationID(opID, postID, adminPubkey, lamport, postOpTypeUpdate)

	columnPrefix := field
	var currentLamport int64
	var currentAdmin string
	var currentOpID string
	query := `
		SELECT ` + columnPrefix + `_lamport, ` + columnPrefix + `_by, ` + columnPrefix + `_op_id
		FROM post_admin_state
		WHERE post_id = ?;
	`
	err := a.db.QueryRow(query, postID).Scan(&currentLamport, &currentAdmin, &currentOpID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		currentVersion := LamportVersion{Lamport: currentLamport, Author: strings.TrimSpace(currentAdmin), OpID: strings.TrimSpace(currentOpID)}
		incomingVersion := LamportVersion{Lamport: lamport, Author: adminPubkey, OpID: opID}
		if compareLamportVersion(incomingVersion, currentVersion) <= 0 {
			return nil
		}
	}

	enabledValue := 0
	if enabled {
		enabledValue = 1
	}

	statement := `
		INSERT INTO post_admin_state (
			post_id, ` + columnPrefix + `, ` + columnPrefix + `_by, ` + columnPrefix + `_updated_at, ` + columnPrefix + `_lamport, ` + columnPrefix + `_op_id
		)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(post_id) DO UPDATE SET
			` + columnPrefix + ` = excluded.` + columnPrefix + `,
			` + columnPrefix + `_by = excluded.` + columnPrefix + `_by,
			` + columnPrefix + `_updated_at = excluded.` + columnPrefix + `_updated_at,
			` + columnPrefix + `_lamport = excluded.` + columnPrefix + `_lamport,
			` + columnPrefix + `_op_id = excluded.` + columnPrefix + `_op_id;
	`
	_, err = a.db.Exec(statement, postID, enabledValue, adminPubkey, timestamp, lamport, opID)
	return err
}

func (a *App) SetPostPinned(postID string, pinned bool) error {
	identity, err := a.currentAdminIdentity()
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return err
	}
	return a.applyPostPinnedState(postID, pinned, identity.PublicKey, now, lamport, generateOperationID(postID, identity.PublicKey, lamport))
}

func (a *App) SetPostLocked(postID string, locked bool) error {
	identity, err := a.currentAdminIdentity()
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	lamport, err := a.nextLamport()
	if err != nil {
		return err
	}
	return a.applyPostLockedState(postID, locked, identity.PublicKey, now, lamport, generateOperationID(postID, identity.PublicKey, lamport))
}

func (a *App) getPostLockState(postID string) (bool, int64, error) {
	if a.db == nil {
		return false, 0, errors.New("database not initialized")
	}
	postID = strings.TrimSpace(postID)
	if postID == "" {
		return false, 0, nil
	}

	var locked int
	var lockLamport int64
	err := a.db.QueryRow(`
		SELECT locked, locked_lamport
		FROM post_admin_state
		WHERE post_id = ?;
	`, postID).Scan(&locked, &lockLamport)
	if errors.Is(err, sql.ErrNoRows) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	return locked == 1, lockLamport, nil
}
