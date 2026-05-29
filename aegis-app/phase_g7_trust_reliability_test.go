package main

import (
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newG7TestApp(t *testing.T, dbName string) *App {
	t.Helper()
	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), dbName))
	if err := app.initDatabase(); err != nil {
		t.Fatalf("init database: %v", err)
	}
	t.Cleanup(func() {
		_ = app.StopP2P()
		if app.db != nil {
			// Checkpoint WAL and truncate before closing.
			_, _ = app.db.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
			_ = app.db.Close()
		}
	})
	return app
}

func seedLocalIdentity(t *testing.T, app *App) Identity {
	t.Helper()
	identity, err := app.GenerateIdentity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	if err = app.saveLocalIdentity(identity); err != nil {
		t.Fatalf("save local identity: %v", err)
	}
	return identity
}

func waitForCondition(t *testing.T, timeout time.Duration, fn func() (bool, error)) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ok, err := fn()
		if err != nil {
			t.Fatalf("wait condition failed: %v", err)
		}
		if ok {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

func loopbackAddress(port int, peerID string) string {
	return fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", port, strings.TrimSpace(peerID))
}

func reserveLoopbackPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve loopback port: %v", err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok || addr.Port == 0 {
		t.Fatalf("reserve loopback port: invalid addr %v", listener.Addr())
	}
	return addr.Port
}

func waitForNoBusyError(t *testing.T, timeout time.Duration, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err := fn()
		if err == nil {
			return
		}
		if !strings.Contains(strings.ToLower(err.Error()), "database is locked") {
			t.Fatalf("unexpected operation error: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("database remained busy for %s", timeout)
}

func TestG7RejectsForgedSignedPost(t *testing.T) {
	app := newG7TestApp(t, "g7_signature.db")
	identity := seedLocalIdentity(t, app)

	message := IncomingMessage{
		Type:          "POST",
		OpType:        postOpTypeCreate,
		OpID:          "forged-post-op",
		SchemaVersion: lamportSchemaV2,
		AuthScope:     authScopeUser,
		ID:            "forged-post",
		Pubkey:        identity.PublicKey,
		Title:         "Original title",
		Body:          "Original body",
		ContentCID:    buildContentCID("Original body"),
		SubID:         defaultSubID,
		Timestamp:     101,
		Lamport:       101,
	}

	signed, err := app.signIncomingMessage(message)
	if err != nil {
		t.Fatalf("sign message: %v", err)
	}
	signed.Body = "Tampered body"

	payload, err := marshalJSON(signed)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err = app.ProcessIncomingMessage(payload); err == nil {
		t.Fatalf("expected forged payload to be rejected")
	}
}

func TestG7PublishPostFlushesOutboxAfterPeerConnect(t *testing.T) {
	nodeA := newG7TestApp(t, "g7_node_a.db")
	nodeB := newG7TestApp(t, "g7_node_b.db")
	identityA := seedLocalIdentity(t, nodeA)
	seedLocalIdentity(t, nodeB)

	if _, err := nodeA.upsertProfile(identityA.PublicKey, "alice", "", time.Now().Unix()); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	if err := nodeA.PublishPostStructuredToSub(identityA.PublicKey, "Queued post", "Replicate me", defaultSubID); err != nil {
		t.Fatalf("publish post offline: %v", err)
	}

	var opID string
	if err := nodeA.db.QueryRow(`SELECT current_op_id FROM messages WHERE title = ? LIMIT 1;`, "Queued post").Scan(&opID); err != nil {
		t.Fatalf("read local post op id: %v", err)
	}

	var pendingCount int
	if err := nodeA.db.QueryRow(`SELECT COUNT(1) FROM message_outbox WHERE id = ?;`, opID).Scan(&pendingCount); err != nil {
		t.Fatalf("count outbox entries: %v", err)
	}
	if pendingCount != 1 {
		t.Fatalf("expected queued outbox entry, got %d", pendingCount)
	}

	portB := reserveLoopbackPort(t)
	portA := reserveLoopbackPort(t)

	statusB, err := nodeB.StartP2P(portB, nil)
	if err != nil {
		t.Fatalf("start node B: %v", err)
	}
	statusA, err := nodeA.StartP2P(portA, nil)
	if err != nil {
		t.Fatalf("start node A: %v", err)
	}

	targetAddr := loopbackAddress(portB, statusB.PeerID)
	if targetAddr == "" {
		t.Fatalf("node B has no connectable address")
	}

	// Allow hosts to fully initialize and mDNS to settle before connecting.
	time.Sleep(2 * time.Second)
	connectPeerWithBackoffClear(t, nodeA, targetAddr, 5)

	waitForCondition(t, 10*time.Second, func() (bool, error) {
		return len(nodeA.GetP2PStatus().ConnectedPeers) > 0 && len(nodeB.GetP2PStatus().ConnectedPeers) > 0, nil
	})

	nodeA.flushOutgoingMessagesAsync()

	waitForCondition(t, 10*time.Second, func() (bool, error) {
		var count int
		if err := nodeA.db.QueryRow(`SELECT COUNT(1) FROM message_outbox WHERE id = ?;`, opID).Scan(&count); err != nil {
			return false, err
		}
		return count == 0, nil
	})

	waitForCondition(t, 10*time.Second, func() (bool, error) {
		var replicated int
		err := nodeB.db.QueryRow(`SELECT COUNT(1) FROM messages WHERE title = ? AND pubkey = ?;`, "Queued post", identityA.PublicKey).Scan(&replicated)
		return replicated == 1, err
	})

	if len(statusA.ListenAddrs) == 0 && len(statusA.AnnounceAddrs) == 0 {
		t.Fatalf("node A failed to expose listen address")
	}
}

func TestG7ReplicatesCommunityOperationsAcrossPeers(t *testing.T) {
	nodeA := newG7TestApp(t, "g7_community_a.db")
	nodeB := newG7TestApp(t, "g7_community_b.db")
	adminIdentity := seedLocalIdentity(t, nodeA)
	seedLocalIdentity(t, nodeB)

	if err := nodeA.AddTrustedAdmin(adminIdentity.PublicKey, "owner"); err != nil {
		t.Fatalf("trust admin on node A: %v", err)
	}
	if err := nodeB.AddTrustedAdmin(adminIdentity.PublicKey, "owner"); err != nil {
		t.Fatalf("trust admin on node B: %v", err)
	}

	portB := reserveLoopbackPort(t)
	portA := reserveLoopbackPort(t)

	statusB, err := nodeB.StartP2P(portB, nil)
	if err != nil {
		t.Fatalf("start node B: %v", err)
	}
	if _, err = nodeA.StartP2P(portA, nil); err != nil {
		t.Fatalf("start node A: %v", err)
	}
	// Allow hosts to fully initialize and mDNS to settle before connecting.
	// This avoids TLS handshake races between explicit ConnectPeer and
	// mDNS-triggered dial attempts.
	time.Sleep(2 * time.Second)
	connectPeerWithBackoffClear(t, nodeA, loopbackAddress(portB, statusB.PeerID), 5)

	waitForCondition(t, 10*time.Second, func() (bool, error) {
		return len(nodeA.GetP2PStatus().ConnectedPeers) > 0 && len(nodeB.GetP2PStatus().ConnectedPeers) > 0, nil
	})

	waitForNoBusyError(t, 5*time.Second, func() error {
		return nodeA.PublishPostStructuredToSub(adminIdentity.PublicKey, "Admin notice", "Important post", defaultSubID)
	})

	var postID string
	waitForCondition(t, 10*time.Second, func() (bool, error) {
		err := nodeA.db.QueryRow(`SELECT id FROM messages WHERE title = ? LIMIT 1;`, "Admin notice").Scan(&postID)
		return err == nil && strings.TrimSpace(postID) != "", nil
	})
	waitForCondition(t, 10*time.Second, func() (bool, error) {
		var replicated int
		err := nodeB.db.QueryRow(`SELECT COUNT(1) FROM messages WHERE id = ?;`, postID).Scan(&replicated)
		return replicated == 1, err
	})

	waitForNoBusyError(t, 5*time.Second, func() error {
		return nodeA.PublishSubSettingsUpdate(defaultSubID, []string{"Be respectful", "Stay on topic"}, "Weekly thread is live.")
	})
	waitForNoBusyError(t, 5*time.Second, func() error {
		return nodeA.PublishSetPostPinned(postID, true)
	})
	waitForNoBusyError(t, 5*time.Second, func() error {
		return nodeA.PublishSetPostLocked(postID, true)
	})

	waitForCondition(t, 10*time.Second, func() (bool, error) {
		settings, err := nodeB.GetSubSettings(defaultSubID)
		if err != nil {
			return false, err
		}
		return settings.Announcement == "Weekly thread is live." && len(settings.Rules) == 2, nil
	})

	waitForCondition(t, 10*time.Second, func() (bool, error) {
		var pinned int
		var locked int
		err := nodeB.db.QueryRow(`
			SELECT pinned, locked
			FROM post_admin_state
			WHERE post_id = ?;
		`, postID).Scan(&pinned, &locked)
		return pinned == 1 && locked == 1, err
	})

	locked, _, err := nodeB.getPostLockState(postID)
	if err != nil {
		t.Fatalf("read post lock state: %v", err)
	}
	if !locked {
		t.Fatalf("expected replicated lock state on node B")
	}
}

func marshalJSON(message IncomingMessage) ([]byte, error) {
	return json.Marshal(message)
}
