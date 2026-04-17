package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

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
