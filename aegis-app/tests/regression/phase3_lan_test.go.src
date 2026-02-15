package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestPhase3P2PReplication is a long-term integration regression test.
// It verifies that a structured public post published by node A is replicated to node B over P2P.
func TestPhase3P2PReplication(t *testing.T) {
	tempDir := t.TempDir()

	nodeA := NewApp()
	nodeB := NewApp()
	nodeA.SetDatabasePath(filepath.Join(tempDir, "node_a.db"))
	nodeB.SetDatabasePath(filepath.Join(tempDir, "node_b.db"))

	if err := nodeA.initDatabase(); err != nil {
		t.Fatalf("nodeA initDatabase failed: %v", err)
	}
	if err := nodeB.initDatabase(); err != nil {
		t.Fatalf("nodeB initDatabase failed: %v", err)
	}

	defer func() {
		_ = nodeA.StopP2P()
		_ = nodeB.StopP2P()
		if nodeA.db != nil {
			_ = nodeA.db.Close()
		}
		if nodeB.db != nil {
			_ = nodeB.db.Close()
		}
	}()

	statusA, err := nodeA.StartP2P(41001, nil)
	if err != nil {
		t.Fatalf("nodeA StartP2P failed: %v", err)
	}

	statusB, err := nodeB.StartP2P(41002, nil)
	if err != nil {
		t.Fatalf("nodeB StartP2P failed: %v", err)
	}

	addrA := pickLoopbackAddr(statusA.ListenAddrs)
	if addrA == "" {
		t.Fatalf("nodeA has no listen address")
	}
	addrB := pickLoopbackAddr(statusB.ListenAddrs)
	if addrB == "" {
		t.Fatalf("nodeB has no listen address")
	}

	if err = nodeB.ConnectPeer(addrA); err != nil {
		t.Fatalf("nodeB ConnectPeer failed: %v", err)
	}
	if err = nodeA.ConnectPeer(addrB); err != nil {
		t.Fatalf("nodeA ConnectPeer failed: %v", err)
	}

	if err = waitForConnectedPeers(nodeA, 6*time.Second); err != nil {
		t.Fatalf("nodeA did not observe peer connection: %v", err)
	}
	if err = waitForConnectedPeers(nodeB, 6*time.Second); err != nil {
		t.Fatalf("nodeB did not observe peer connection: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	title := "phase3-lan-test-title"
	body := "phase3-lan-test-message"
	if err = nodeA.PublishPostStructuredToSub("node-a-pubkey", title, body, "general"); err != nil {
		t.Fatalf("nodeA PublishPostStructuredToSub failed: %v", err)
	}

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		feed, feedErr := nodeB.GetFeed()
		if feedErr == nil {
			for _, message := range feed {
				if message.Title == title && message.Body == body {
					return
				}
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	feed, _ := nodeB.GetFeed()
	t.Fatalf("nodeB did not receive published message, feed size=%d", len(feed))
}

func pickLoopbackAddr(addrs []string) string {
	for _, addr := range addrs {
		if strings.Contains(addr, "/ip4/127.0.0.1/") {
			return addr
		}
	}
	for _, addr := range addrs {
		if strings.Contains(addr, "/ip4/0.0.0.0/") {
			return strings.Replace(addr, "/ip4/0.0.0.0/", "/ip4/127.0.0.1/", 1)
		}
	}
	return ""
}

func waitForConnectedPeers(app *App, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status := app.GetP2PStatus()
		if len(status.ConnectedPeers) > 0 {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return errors.New("timeout waiting for connected peers")
}
