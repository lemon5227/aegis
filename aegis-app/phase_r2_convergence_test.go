package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestR2OfflineRejoinConvergesWithinWindow(t *testing.T) {
	t.Setenv("AEGIS_ANTI_ENTROPY_INTERVAL_SEC", "1")
	t.Setenv("AEGIS_ANTI_ENTROPY_WINDOW_SEC", "3600")
	t.Setenv("AEGIS_ANTI_ENTROPY_BATCH_SIZE", "64")
	t.Setenv("AEGIS_ANTI_ENTROPY_INDEX_BUDGET", "3")
	t.Setenv("AEGIS_ANTI_ENTROPY_BODY_BUDGET", "2")

	tempDir := t.TempDir()
	nodeA := NewApp()
	nodeB := NewApp()
	nodeA.SetDatabasePath(filepath.Join(tempDir, "r2_rejoin_a.db"))
	nodeB.SetDatabasePath(filepath.Join(tempDir, "r2_rejoin_b.db"))

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

	statusA, err := nodeA.StartP2P(41301, nil)
	if err != nil {
		t.Fatalf("nodeA StartP2P failed: %v", err)
	}
	statusB, err := nodeB.StartP2P(41302, nil)
	if err != nil {
		t.Fatalf("nodeB StartP2P failed: %v", err)
	}

	addrA := pickLoopbackAddr(statusA.ListenAddrs)
	addrB := pickLoopbackAddr(statusB.ListenAddrs)
	if addrA == "" || addrB == "" {
		t.Fatalf("missing listen addresses: addrA=%q addrB=%q", addrA, addrB)
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

	if err = nodeB.StopP2P(); err != nil && !strings.Contains(strings.ToLower(err.Error()), "context canceled") {
		t.Fatalf("nodeB StopP2P failed: %v", err)
	}

	type expectedPost struct {
		title string
		body  string
	}
	expected := []expectedPost{
		{title: "r2-rejoin-1", body: "body-rejoin-1"},
		{title: "r2-rejoin-2", body: "body-rejoin-2"},
		{title: "r2-rejoin-3", body: "body-rejoin-3"},
		{title: "r2-rejoin-4", body: "body-rejoin-4"},
	}

	for _, post := range expected {
		if err = nodeA.PublishPostStructuredToSub("node-a", post.title, post.body, "general"); err != nil {
			t.Fatalf("nodeA PublishPostStructuredToSub %s failed: %v", post.title, err)
		}
	}

	statusB, err = nodeB.StartP2P(41302, nil)
	if err != nil {
		t.Fatalf("nodeB restart StartP2P failed: %v", err)
	}
	addrB = pickLoopbackAddr(statusB.ListenAddrs)
	if addrB == "" {
		t.Fatalf("nodeB restart listen address missing")
	}
	if err = nodeB.ConnectPeer(addrA); err != nil {
		t.Fatalf("nodeB reconnect failed: %v", err)
	}
	if err = nodeA.ConnectPeer(addrB); err != nil {
		t.Fatalf("nodeA reconnect failed: %v", err)
	}
	if err = waitForConnectedPeers(nodeB, 6*time.Second); err != nil {
		t.Fatalf("nodeB did not reconnect to peers: %v", err)
	}

	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		feed, feedErr := nodeB.GetFeedIndexBySubSorted("general", "new")
		if feedErr == nil {
			allRecovered := true
			for _, expectedPost := range expected {
				matchedID := ""
				for _, row := range feed {
					if strings.TrimSpace(row.Title) == expectedPost.title {
						matchedID = row.ID
						break
					}
				}
				if matchedID == "" {
					allRecovered = false
					break
				}

				blob, blobErr := nodeB.GetPostBodyByID(matchedID)
				if blobErr != nil || strings.TrimSpace(blob.Body) != expectedPost.body {
					allRecovered = false
					break
				}
			}

			if allRecovered {
				stats := nodeB.GetAntiEntropyStats()
				if stats.IndexInsertions == 0 {
					t.Fatalf("expected anti-entropy index insertions > 0")
				}
				if stats.BlobFetchAttempts == 0 {
					t.Fatalf("expected anti-entropy blob fetch attempts > 0")
				}
				return
			}
		}

		time.Sleep(300 * time.Millisecond)
	}

	stats := nodeB.GetAntiEntropyStats()
	t.Fatalf("nodeB did not converge after rejoin in time: stats=%+v", stats)
}
