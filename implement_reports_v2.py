import re

file_path = 'aegis-app/app.go'

with open(file_path, 'r') as f:
    content = f.read()

submit_report_code = """
func (a *App) SubmitReport(targetID string, targetType string, reason string) error {
	if a.db == nil {
		return errors.New("database not initialized")
	}

	targetID = strings.TrimSpace(targetID)
	targetType = strings.ToLower(strings.TrimSpace(targetType))
	reason = strings.TrimSpace(reason)

	if targetID == "" || reason == "" {
		return errors.New("target id and reason are required")
	}
	if targetType != "post" && targetType != "comment" {
		return errors.New("invalid target type")
	}

	identity, err := a.getLocalIdentity()
	if err != nil {
		return err
	}
	reporterPubkey := identity.PublicKey

	now := time.Now().Unix()
	reportID := buildMessageID(reporterPubkey, fmt.Sprintf("report|%s|%s|%d", targetID, reason, now), now)

	_, err = a.db.Exec(`
		INSERT INTO reports (id, target_id, target_type, reason, reporter_pubkey, timestamp, status)
		VALUES (?, ?, ?, ?, ?, ?, 'pending')
		ON CONFLICT(id) DO NOTHING;
	`, reportID, targetID, targetType, reason, reporterPubkey, now)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) GetReports(limit int, status string) ([]Report, error) {
	if a.db == nil {
		return nil, errors.New("database not initialized")
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

    status = strings.TrimSpace(strings.ToLower(status))

    query := `
		SELECT id, target_id, target_type, reason, reporter_pubkey, timestamp, status
		FROM reports
    `
    args := []interface{}{}

    if status != "" && status != "all" {
        query += " WHERE status = ?"
        args = append(args, status)
    }

    query += " ORDER BY timestamp DESC LIMIT ?;"
    args = append(args, limit)

	rows, err := a.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Report, 0)
	for rows.Next() {
		var item Report
		if err := rows.Scan(
			&item.ID,
			&item.TargetID,
			&item.TargetType,
			&item.Reason,
			&item.ReporterPubkey,
			&item.Timestamp,
			&item.Status,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	return result, rows.Err()
}

func (a *App) UpdateReportStatus(reportID string, status string) error {
    if a.db == nil {
		return errors.New("database not initialized")
	}

    status = strings.ToLower(strings.TrimSpace(status))
    if status != "pending" && status != "resolved" && status != "ignored" {
        return errors.New("invalid status")
    }

    _, err := a.db.Exec("UPDATE reports SET status = ? WHERE id = ?", status, reportID)
    return err
}
"""

content = content + "\n" + submit_report_code

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
