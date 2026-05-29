package main

import (
	"testing"
)

// -----------------------------------------------------------------------------
// GetTrustedAdmins
// -----------------------------------------------------------------------------

func TestGetTrustedAdminsEmpty(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	admins, err := app.GetTrustedAdmins()
	if err != nil {
		t.Fatalf("GetTrustedAdmins on empty db: %v", err)
	}
	if len(admins) != 0 {
		t.Errorf("expected 0 admins, got %d", len(admins))
	}
	if admins == nil {
		t.Error("result should be non-nil empty slice (frontend expects [])")
	}
}

func TestGetTrustedAdminsIncludesAdded(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.AddTrustedAdmin("admin-1", "owner"); err != nil {
		t.Fatalf("add owner: %v", err)
	}
	if err := app.AddTrustedAdmin("admin-2", "appointed"); err != nil {
		t.Fatalf("add appointed: %v", err)
	}

	admins, err := app.GetTrustedAdmins()
	if err != nil {
		t.Fatalf("GetTrustedAdmins: %v", err)
	}
	if len(admins) != 2 {
		t.Fatalf("expected 2 admins, got %d", len(admins))
	}

	// Ordered alphabetically by role, then pubkey: appointed < owner.
	if admins[0].Role != "appointed" || admins[0].AdminPubkey != "admin-2" {
		t.Errorf("expected first row appointed/admin-2, got %+v", admins[0])
	}
	if admins[1].Role != "owner" || admins[1].AdminPubkey != "admin-1" {
		t.Errorf("expected second row owner/admin-1, got %+v", admins[1])
	}
	for _, a := range admins {
		if !a.Active {
			t.Errorf("expected admin %s to be active", a.AdminPubkey)
		}
	}
}

func TestAddTrustedAdminUpsertsRole(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// First call: appointed.
	if err := app.AddTrustedAdmin("admin-X", "appointed"); err != nil {
		t.Fatalf("first add: %v", err)
	}
	// Second call with the same pubkey but a different role: should update.
	if err := app.AddTrustedAdmin("admin-X", "owner"); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	admins, _ := app.GetTrustedAdmins()
	if len(admins) != 1 {
		t.Fatalf("upsert should keep single row, got %d", len(admins))
	}
	if admins[0].Role != "owner" {
		t.Errorf("role should have been upgraded to owner, got %q", admins[0].Role)
	}
}

func TestAddTrustedAdminTrimsAndLowercasesRole(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.AddTrustedAdmin("  admin-Y  ", "  OWNER  "); err != nil {
		t.Fatalf("add: %v", err)
	}
	admins, _ := app.GetTrustedAdmins()
	if len(admins) != 1 {
		t.Fatalf("expected 1 admin, got %d", len(admins))
	}
	if admins[0].AdminPubkey != "admin-Y" {
		t.Errorf("pubkey should be trimmed: %q", admins[0].AdminPubkey)
	}
	if admins[0].Role != "owner" {
		t.Errorf("role should be lowercased: %q", admins[0].Role)
	}
}

func TestAddTrustedAdminDefaultsRoleWhenEmpty(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.AddTrustedAdmin("admin-empty-role", ""); err != nil {
		t.Fatalf("add: %v", err)
	}
	admins, _ := app.GetTrustedAdmins()
	if len(admins) != 1 || admins[0].Role != "appointed" {
		t.Errorf("empty role should default to appointed, got %+v", admins)
	}
}

func TestAddTrustedAdminRejectsEmptyPubkey(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.AddTrustedAdmin("   ", "owner"); err == nil {
		t.Error("expected error for whitespace-only pubkey")
	}
	admins, _ := app.GetTrustedAdmins()
	if len(admins) != 0 {
		t.Errorf("expected 0 admins after rejected add, got %d", len(admins))
	}
}

// -----------------------------------------------------------------------------
// seedTrustedAdminsFromEnv
// -----------------------------------------------------------------------------

func TestSeedTrustedAdminsFromEnvEmpty(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)
	t.Setenv("AEGIS_TRUSTED_ADMINS", "")

	app.seedTrustedAdminsFromEnv()

	admins, _ := app.GetTrustedAdmins()
	if len(admins) != 0 {
		t.Errorf("empty env should not seed admins, got %d", len(admins))
	}
}

func TestSeedTrustedAdminsFromEnvSeedsMultiple(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)
	t.Setenv("AEGIS_TRUSTED_ADMINS", "admin-A, admin-B ,admin-C")

	app.seedTrustedAdminsFromEnv()

	admins, _ := app.GetTrustedAdmins()
	if len(admins) != 3 {
		t.Fatalf("expected 3 seeded admins, got %d", len(admins))
	}

	keys := map[string]bool{}
	for _, a := range admins {
		keys[a.AdminPubkey] = true
		if a.Role != "genesis" {
			t.Errorf("seeded admins must have role=genesis, got %q for %s", a.Role, a.AdminPubkey)
		}
	}
	for _, want := range []string{"admin-A", "admin-B", "admin-C"} {
		if !keys[want] {
			t.Errorf("missing seeded admin %s; got keys %v", want, keys)
		}
	}
}

func TestSeedTrustedAdminsFromEnvSkipsWhitespaceEntries(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)
	t.Setenv("AEGIS_TRUSTED_ADMINS", "admin-1,,   ,admin-2,")

	app.seedTrustedAdminsFromEnv()

	admins, _ := app.GetTrustedAdmins()
	if len(admins) != 2 {
		t.Errorf("blank entries should be ignored, got %d admins (%v)", len(admins), admins)
	}
}

func TestSeedTrustedAdminsFromEnvIsIdempotent(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)
	t.Setenv("AEGIS_TRUSTED_ADMINS", "admin-A,admin-B")

	// Running twice must not duplicate (AddTrustedAdmin uses upsert).
	app.seedTrustedAdminsFromEnv()
	app.seedTrustedAdminsFromEnv()

	admins, _ := app.GetTrustedAdmins()
	if len(admins) != 2 {
		t.Errorf("idempotent seed expected 2 admins, got %d", len(admins))
	}
}

// -----------------------------------------------------------------------------
// ApplyShadowBan / ApplyUnban
// -----------------------------------------------------------------------------

func TestApplyShadowBanRequiresTrustedAdmin(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// No admin seeded yet — admin pubkey is not trusted.
	err := app.ApplyShadowBan("target-pubkey", "untrusted-admin", "spam")
	if err == nil {
		t.Fatal("expected error when admin is not trusted")
	}
	if err.Error() != "admin pubkey is not trusted" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyShadowBanRecordsModeration(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)
	if err := app.AddTrustedAdmin("admin-1", "owner"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	if err := app.ApplyShadowBan("target-pubkey", "admin-1", "spam"); err != nil {
		t.Fatalf("apply ban: %v", err)
	}

	// Verify via the public moderation state API.
	state, err := app.GetModerationState()
	if err != nil {
		t.Fatalf("get moderation state: %v", err)
	}
	found := false
	for _, m := range state {
		if m.TargetPubkey == "target-pubkey" && m.Action == "SHADOW_BAN" {
			found = true
			if m.Reason != "spam" {
				t.Errorf("reason mismatch: got %q", m.Reason)
			}
			if m.SourceAdmin != "admin-1" {
				t.Errorf("admin mismatch: got %q", m.SourceAdmin)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected moderation row for target-pubkey, got %+v", state)
	}
}

func TestApplyUnbanRequiresTrustedAdmin(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	if err := app.ApplyUnban("target-pubkey", "untrusted-admin", "reformed"); err == nil {
		t.Fatal("expected error when admin is not trusted")
	}
}

func TestApplyUnbanReplacesShadowBan(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)
	if err := app.AddTrustedAdmin("admin-1", "owner"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	if err := app.ApplyShadowBan("target-pubkey", "admin-1", "spam"); err != nil {
		t.Fatalf("ban: %v", err)
	}
	if err := app.ApplyUnban("target-pubkey", "admin-1", "reformed"); err != nil {
		t.Fatalf("unban: %v", err)
	}

	state, err := app.GetModerationState()
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	for _, m := range state {
		if m.TargetPubkey == "target-pubkey" && m.Action == "SHADOW_BAN" {
			t.Errorf("expected SHADOW_BAN to be replaced by UNBAN, got %+v", m)
		}
	}
}

// -----------------------------------------------------------------------------
// GetGovernancePolicy / SetGovernancePolicy
// -----------------------------------------------------------------------------

func TestGetGovernancePolicyDefaultsToTrueWhenUnset(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	policy, err := app.GetGovernancePolicy()
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if !policy.HideHistoryOnShadowBan {
		t.Error("default policy should hide history on shadow-ban")
	}
}

func TestSetGovernancePolicyRoundTrip(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	policy, err := app.SetGovernancePolicy(false)
	if err != nil {
		t.Fatalf("set policy false: %v", err)
	}
	if policy.HideHistoryOnShadowBan {
		t.Error("returned policy should reflect false")
	}

	got, _ := app.GetGovernancePolicy()
	if got.HideHistoryOnShadowBan {
		t.Error("persisted policy should reflect false")
	}

	if _, err := app.SetGovernancePolicy(true); err != nil {
		t.Fatalf("set policy true: %v", err)
	}
	got, _ = app.GetGovernancePolicy()
	if !got.HideHistoryOnShadowBan {
		t.Error("persisted policy should reflect true after toggle")
	}
}

func TestSetGovernancePolicyTrueShadowsExistingMessages(t *testing.T) {
	app, identity := newTestAppWithIdentity(t)
	if err := app.AddTrustedAdmin(identity.PublicKey, "owner"); err != nil {
		t.Fatalf("seed admin: %v", err)
	}

	// Publish a post by the user we are going to ban.
	bannedAuthor, _ := generateRemoteIdentity(t)
	now := int64(1_700_000_000)
	if _, err := app.db.Exec(
		`INSERT INTO messages (id, pubkey, content, score, timestamp, lamport, size_bytes, zone, sub_id, visibility)
		 VALUES (?, ?, ?, 0, ?, 1, ?, 'public', ?, 'normal')`,
		"mod-post-1", bannedAuthor, "body", now, int64(len("body")), defaultSubID,
	); err != nil {
		t.Fatalf("insert remote post: %v", err)
	}

	if err := app.ApplyShadowBan(bannedAuthor, identity.PublicKey, "spam"); err != nil {
		t.Fatalf("ban: %v", err)
	}

	if _, err := app.SetGovernancePolicy(true); err != nil {
		t.Fatalf("set true: %v", err)
	}

	// Toggling true must shadow the existing message in place.
	var visibility string
	if err := app.db.QueryRow(`SELECT visibility FROM messages WHERE id = ?`, "mod-post-1").Scan(&visibility); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if visibility != "shadowed" {
		t.Errorf("expected message to be shadowed, got %q", visibility)
	}

	// Toggling back to false must restore visibility to normal.
	if _, err := app.SetGovernancePolicy(false); err != nil {
		t.Fatalf("set false: %v", err)
	}
	if err := app.db.QueryRow(`SELECT visibility FROM messages WHERE id = ?`, "mod-post-1").Scan(&visibility); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if visibility != "normal" {
		t.Errorf("expected message to be unshadowed, got %q", visibility)
	}
}

// -----------------------------------------------------------------------------
// GetIdentityState
// -----------------------------------------------------------------------------

func TestGetIdentityStateEmpty(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	state, err := app.GetIdentityState()
	if err != nil {
		t.Fatalf("get identity state: %v", err)
	}
	if len(state) != 0 {
		t.Errorf("expected empty state, got %d entries", len(state))
	}
	if state == nil {
		t.Error("result should be non-nil empty slice")
	}
}

func TestGetIdentityStateScansRows(t *testing.T) {
	app, _ := newTestAppWithIdentity(t)

	// Seed two rows so we can verify ordering by updated_at DESC.
	if _, err := app.db.Exec(
		`INSERT INTO identity_state (pubkey, state, storage_commit_bytes, public_quota_bytes, private_quota_bytes, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"pk-old", "active", int64(100), int64(1000), int64(500), int64(1_700_000_000),
	); err != nil {
		t.Fatalf("seed old: %v", err)
	}
	if _, err := app.db.Exec(
		`INSERT INTO identity_state (pubkey, state, storage_commit_bytes, public_quota_bytes, private_quota_bytes, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		"pk-new", "active", int64(200), int64(2000), int64(800), int64(1_700_000_100),
	); err != nil {
		t.Fatalf("seed new: %v", err)
	}

	state, err := app.GetIdentityState()
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if len(state) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(state))
	}
	// Most recent first.
	if state[0].Pubkey != "pk-new" || state[1].Pubkey != "pk-old" {
		t.Errorf("expected DESC ordering by updated_at, got %+v", state)
	}
	if state[0].StorageCommitBytes != 200 {
		t.Errorf("storage commit: got %d, want 200", state[0].StorageCommitBytes)
	}
	if state[0].PublicQuotaBytes != 2000 || state[0].PrivateQuotaBytes != 800 {
		t.Errorf("quota fields: got pub=%d priv=%d", state[0].PublicQuotaBytes, state[0].PrivateQuotaBytes)
	}
}
