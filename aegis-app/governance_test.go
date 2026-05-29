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
