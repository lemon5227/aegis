package main

import (
	"path/filepath"
	"strings"
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

	adminPubkey := "trusted-admin-1"
	if err := app.AddTrustedAdmin(adminPubkey, "appointed"); err != nil {
		t.Fatalf("AddTrustedAdmin failed: %v", err)
	}

	payload := []byte(`{"type":"GOVERNANCE_POLICY_UPDATE","admin_pubkey":"trusted-admin-1","hide_history_on_shadowban":false}`)
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

func TestC1GovernancePolicyUpdateMessageRejectsUntrustedAdmin(t *testing.T) {
	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), "policy_sync_reject.db"))
	if err := app.initDatabase(); err != nil {
		t.Fatalf("initDatabase failed: %v", err)
	}
	defer func() {
		if app.db != nil {
			_ = app.db.Close()
		}
	}()

	payload := []byte(`{"type":"GOVERNANCE_POLICY_UPDATE","admin_pubkey":"normal-user","hide_history_on_shadowban":false}`)
	err := app.ProcessIncomingMessage(payload)
	if err == nil {
		t.Fatalf("expected untrusted admin policy update to be rejected")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "not trusted") {
		t.Fatalf("expected not trusted error, got %v", err)
	}
}
