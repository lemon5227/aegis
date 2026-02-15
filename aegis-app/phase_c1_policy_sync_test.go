package main

import (
	"path/filepath"
	"testing"
)

func TestC1GovernancePolicyUpdateMessageApplied(t *testing.T) {
	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), "policy_sync.db"))
	if err := app.initDatabase(); err != nil {
		t.Fatalf("initDatabase failed: %v", err)
	}
	defer func() {
		if app.db != nil {
			_ = app.db.Close()
		}
	}()

	if _, err := app.SetGovernancePolicy(true); err != nil {
		t.Fatalf("SetGovernancePolicy(true) failed: %v", err)
	}

	payload := []byte(`{"type":"GOVERNANCE_POLICY_UPDATE","hide_history_on_shadowban":false}`)
	if err := app.ProcessIncomingMessage(payload); err != nil {
		t.Fatalf("ProcessIncomingMessage failed: %v", err)
	}

	policy, err := app.GetGovernancePolicy()
	if err != nil {
		t.Fatalf("GetGovernancePolicy failed: %v", err)
	}
	if policy.HideHistoryOnShadowBan {
		t.Fatalf("expected policy hideHistoryOnShadowBan=false after message sync")
	}
}
