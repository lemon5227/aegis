package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestR2AntiEntropyManualSyncRecoversMissedPost(t *testing.T) {
	tempDir := t.TempDir()

	nodeA := NewApp()
	nodeB := NewApp()
	nodeA.SetDatabasePath(filepath.Join(tempDir, "r2_manual_a.db"))
	nodeB.SetDatabasePath(filepath.Join(tempDir, "r2_manual_b.db"))

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

	statusA, err := nodeA.StartP2P(41201, nil)
	if err != nil {
		t.Fatalf("nodeA StartP2P failed: %v", err)
	}

	title := "r2-manual-title"
	body := "r2-manual-body-from-node-a"
	if err = nodeA.PublishPostStructuredToSub("node-a", title, body, "general"); err != nil {
		t.Fatalf("nodeA PublishPostStructuredToSub failed: %v", err)
	}

	indexA, err := nodeA.GetFeedIndexBySubSorted("general", "new")
	if err != nil || len(indexA) == 0 {
		t.Fatalf("nodeA GetFeedIndexBySubSorted failed: %v, len=%d", err, len(indexA))
	}

	targetID := indexA[0].ID
	targetCID := indexA[0].ContentCID
	if targetID == "" || targetCID == "" {
		t.Fatalf("nodeA target index invalid: id=%q cid=%q", targetID, targetCID)
	}

	statusB, err := nodeB.StartP2P(41202, nil)
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

	if _, err = nodeB.GetPostBodyByID(targetID); err == nil {
		t.Fatalf("expected nodeB to miss post body before anti-entropy sync")
	}

	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		_ = nodeB.TriggerAntiEntropySyncNow()

		rows, fetchErr := nodeB.GetFeedIndexBySubSorted("general", "new")
		if fetchErr == nil {
			for _, item := range rows {
				if item.ID == targetID {
					blob, blobErr := nodeB.getContentBlobLocal(targetCID)
					if blobErr == nil && blob.Body == body {
						return
					}
				}
			}
		}

		time.Sleep(400 * time.Millisecond)
	}

	rows, _ := nodeB.GetFeedIndexBySubSorted("general", "new")
	t.Fatalf("nodeB anti-entropy sync did not recover index+body, feed_len=%d", len(rows))
}

func TestR2AntiEntropyPeriodicSyncRecoversMissedPost(t *testing.T) {
	t.Setenv("AEGIS_ANTI_ENTROPY_INTERVAL_SEC", "1")
	tempDir := t.TempDir()

	nodeA := NewApp()
	nodeB := NewApp()
	nodeA.SetDatabasePath(filepath.Join(tempDir, "r2_periodic_a.db"))
	nodeB.SetDatabasePath(filepath.Join(tempDir, "r2_periodic_b.db"))

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

	statusA, err := nodeA.StartP2P(41211, nil)
	if err != nil {
		t.Fatalf("nodeA StartP2P failed: %v", err)
	}

	title := "r2-periodic-title"
	body := "r2-periodic-body-from-node-a"
	if err = nodeA.PublishPostStructuredToSub("node-a", title, body, "general"); err != nil {
		t.Fatalf("nodeA PublishPostStructuredToSub failed: %v", err)
	}

	indexA, err := nodeA.GetFeedIndexBySubSorted("general", "new")
	if err != nil || len(indexA) == 0 {
		t.Fatalf("nodeA GetFeedIndexBySubSorted failed: %v, len=%d", err, len(indexA))
	}
	targetID := indexA[0].ID
	targetCID := indexA[0].ContentCID

	statusB, err := nodeB.StartP2P(41212, nil)
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

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		rows, fetchErr := nodeB.GetFeedIndexBySubSorted("general", "new")
		if fetchErr == nil {
			for _, item := range rows {
				if item.ID == targetID {
					blob, blobErr := nodeB.getContentBlobLocal(targetCID)
					if blobErr == nil && blob.Body == body {
						return
					}
				}
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

	rows, _ := nodeB.GetFeedIndexBySubSorted("general", "new")
	t.Fatalf("nodeB periodic anti-entropy sync did not recover index+body, feed_len=%d", len(rows))
}
