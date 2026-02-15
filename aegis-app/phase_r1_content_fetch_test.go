package main

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestR1ContentFetchFromPeerOnLocalMiss(t *testing.T) {
	tempDir := t.TempDir()

	nodeA := NewApp()
	nodeB := NewApp()
	nodeA.SetDatabasePath(filepath.Join(tempDir, "r1_node_a.db"))
	nodeB.SetDatabasePath(filepath.Join(tempDir, "r1_node_b.db"))

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

	statusA, err := nodeA.StartP2P(41101, nil)
	if err != nil {
		t.Fatalf("nodeA StartP2P failed: %v", err)
	}
	statusB, err := nodeB.StartP2P(41102, nil)
	if err != nil {
		t.Fatalf("nodeB StartP2P failed: %v", err)
	}

	addrA := pickLoopbackAddr(statusA.ListenAddrs)
	addrB := pickLoopbackAddr(statusB.ListenAddrs)
	if addrA == "" || addrB == "" {
		t.Fatalf("failed to resolve loopback addresses")
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

	title := "r1-fetch-title"
	body := "r1-fetch-body-full-content"
	if err = nodeA.PublishPostStructuredToSub("node-a", title, body, "general"); err != nil {
		t.Fatalf("nodeA publish failed: %v", err)
	}

	var postID string
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		feed, feedErr := nodeB.GetFeedBySubSorted("general", "new")
		if feedErr == nil {
			for _, msg := range feed {
				if msg.Title == title {
					postID = msg.ID
					break
				}
			}
		}
		if postID != "" {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if postID == "" {
		t.Fatalf("nodeB did not receive replicated post")
	}

	var contentCID string
	if err = nodeB.db.QueryRow(`SELECT content_cid FROM messages WHERE id = ?;`, postID).Scan(&contentCID); err != nil {
		t.Fatalf("query content_cid failed: %v", err)
	}
	if contentCID == "" {
		t.Fatalf("expected content_cid on nodeB")
	}

	if _, err = nodeB.db.Exec(`DELETE FROM content_blobs WHERE content_cid = ?;`, contentCID); err != nil {
		t.Fatalf("delete local blob failed: %v", err)
	}

	blob, err := nodeB.GetPostBodyByID(postID)
	if err != nil {
		t.Fatalf("GetPostBodyByID should fetch from peer, got error: %v", err)
	}
	if blob.Body != body {
		t.Fatalf("expected fetched body %q, got %q", body, blob.Body)
	}
}

func TestR1ContentFetchNoPeers(t *testing.T) {
	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), "r1_no_peers.db"))
	if err := app.initDatabase(); err != nil {
		t.Fatalf("initDatabase failed: %v", err)
	}
	defer func() {
		_ = app.StopP2P()
		if app.db != nil {
			_ = app.db.Close()
		}
	}()

	if _, err := app.StartP2P(41201, nil); err != nil {
		t.Fatalf("StartP2P failed: %v", err)
	}

	_, err := app.GetPostBodyByCID("cidv1-missing")
	if err == nil {
		t.Fatalf("expected error when no peers are connected")
	}
	if !errors.Is(err, errContentFetchNoPeers) {
		t.Fatalf("expected errContentFetchNoPeers, got %v", err)
	}
}

func TestR1ContentFetchTimeout(t *testing.T) {
	nodeA := NewApp()
	nodeA.SetDatabasePath(filepath.Join(t.TempDir(), "r1_timeout_a.db"))
	if err := nodeA.initDatabase(); err != nil {
		t.Fatalf("nodeA initDatabase failed: %v", err)
	}
	defer func() {
		_ = nodeA.StopP2P()
		if nodeA.db != nil {
			_ = nodeA.db.Close()
		}
	}()

	nodeB := NewApp()
	nodeB.SetDatabasePath(filepath.Join(t.TempDir(), "r1_timeout_b.db"))
	if err := nodeB.initDatabase(); err != nil {
		t.Fatalf("nodeB initDatabase failed: %v", err)
	}
	defer func() {
		_ = nodeB.StopP2P()
		if nodeB.db != nil {
			_ = nodeB.db.Close()
		}
	}()

	statusA, err := nodeA.StartP2P(41211, nil)
	if err != nil {
		t.Fatalf("nodeA StartP2P failed: %v", err)
	}
	statusB, err := nodeB.StartP2P(41212, nil)
	if err != nil {
		t.Fatalf("nodeB StartP2P failed: %v", err)
	}

	addrA := pickLoopbackAddr(statusA.ListenAddrs)
	addrB := pickLoopbackAddr(statusB.ListenAddrs)
	if addrA == "" || addrB == "" {
		t.Fatalf("failed to resolve loopback addresses")
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

	if err = nodeA.fetchContentBlobFromNetwork("cidv1-not-found", 600*time.Millisecond); err == nil {
		t.Fatalf("expected timeout for non-existent cid")
	}
	if !errors.Is(err, errContentFetchTimeout) {
		t.Fatalf("expected errContentFetchTimeout, got %v", err)
	}
}
