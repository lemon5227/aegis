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

	if err = nodeA.ConnectPeer(targetAddr); err != nil {
		t.Fatalf("connect node A to node B: %v", err)
	}

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

func marshalJSON(message IncomingMessage) ([]byte, error) {
	return json.Marshal(message)
}
