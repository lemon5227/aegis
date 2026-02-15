package main

import (
	"encoding/base64"
	"path/filepath"
	"testing"
	"time"
)

const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/w8AAgMBgVfQnNwAAAAASUVORK5CYII="

func TestR3ImageBlobDeduplicatedByCID(t *testing.T) {
	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), "r3_dedupe.db"))
	if err := app.initDatabase(); err != nil {
		t.Fatalf("initDatabase failed: %v", err)
	}
	defer func() {
		if app.db != nil {
			_ = app.db.Close()
		}
	}()

	post1, err := app.AddLocalPostWithImageToSub("pk-1", "img-1", "body-1", "public", "general", tinyPNGBase64, "image/png")
	if err != nil {
		t.Fatalf("AddLocalPostWithImageToSub post1 failed: %v", err)
	}
	post2, err := app.AddLocalPostWithImageToSub("pk-1", "img-2", "body-2", "public", "general", tinyPNGBase64, "image/png")
	if err != nil {
		t.Fatalf("AddLocalPostWithImageToSub post2 failed: %v", err)
	}

	if post1.ImageCID == "" || post2.ImageCID == "" {
		t.Fatalf("expected image cids to be set")
	}
	if post1.ImageCID != post2.ImageCID {
		t.Fatalf("expected same image cid for identical image payload")
	}

	var count int
	if err = app.db.QueryRow(`SELECT COUNT(1) FROM media_blobs WHERE content_cid = ?;`, post1.ImageCID).Scan(&count); err != nil {
		t.Fatalf("query media_blobs failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected deduplicated media blob count=1, got %d", count)
	}

	media, err := app.GetPostMediaByID(post1.ID)
	if err != nil {
		t.Fatalf("GetPostMediaByID failed: %v", err)
	}
	if media.ContentCID != post1.ImageCID {
		t.Fatalf("expected media cid %s got %s", post1.ImageCID, media.ContentCID)
	}
	if media.Mime != "image/png" {
		t.Fatalf("expected mime image/png got %s", media.Mime)
	}
}

func TestR3MediaFetchFromPeerOnLocalMiss(t *testing.T) {
	tempDir := t.TempDir()

	nodeA := NewApp()
	nodeB := NewApp()
	nodeA.SetDatabasePath(filepath.Join(tempDir, "r3_media_a.db"))
	nodeB.SetDatabasePath(filepath.Join(tempDir, "r3_media_b.db"))

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

	statusA, err := nodeA.StartP2P(41401, nil)
	if err != nil {
		t.Fatalf("nodeA StartP2P failed: %v", err)
	}
	statusB, err := nodeB.StartP2P(41402, nil)
	if err != nil {
		t.Fatalf("nodeB StartP2P failed: %v", err)
	}

	addrA := pickLoopbackAddr(statusA.ListenAddrs)
	addrB := pickLoopbackAddr(statusB.ListenAddrs)
	if addrA == "" || addrB == "" {
		t.Fatalf("missing loopback address addrA=%q addrB=%q", addrA, addrB)
	}
	if err = nodeB.ConnectPeer(addrA); err != nil {
		t.Fatalf("nodeB ConnectPeer failed: %v", err)
	}
	if err = nodeA.ConnectPeer(addrB); err != nil {
		t.Fatalf("nodeA ConnectPeer failed: %v", err)
	}

	if err = waitForConnectedPeers(nodeA, 6*time.Second); err != nil {
		t.Fatalf("nodeA waitForConnectedPeers failed: %v", err)
	}
	if err = waitForConnectedPeers(nodeB, 6*time.Second); err != nil {
		t.Fatalf("nodeB waitForConnectedPeers failed: %v", err)
	}

	title := "r3-image-post"
	body := "r3-image-body"
	if err = nodeA.PublishPostWithImageToSub("node-a", title, body, tinyPNGBase64, "image/png", "general"); err != nil {
		t.Fatalf("PublishPostWithImageToSub failed: %v", err)
	}

	var targetID string
	var targetCID string
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		rows, fetchErr := nodeB.GetFeedIndexBySubSorted("general", "new")
		if fetchErr == nil {
			for _, row := range rows {
				if row.Title == title {
					targetID = row.ID
					targetCID = row.ImageCID
					break
				}
			}
		}
		if targetID != "" && targetCID != "" {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if targetID == "" || targetCID == "" {
		t.Fatalf("nodeB did not receive image post metadata id=%q cid=%q", targetID, targetCID)
	}

	if _, err = nodeB.db.Exec(`DELETE FROM media_blobs WHERE content_cid = ?;`, targetCID); err != nil {
		t.Fatalf("cleanup nodeB media blob failed: %v", err)
	}

	var media MediaBlob
	deadline = time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		media, err = nodeB.GetPostMediaByID(targetID)
		if err == nil && media.ContentCID == targetCID && media.DataBase64 != "" {
			if _, decodeErr := base64.StdEncoding.DecodeString(media.DataBase64); decodeErr == nil {
				return
			}
		}
		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("GetPostMediaByID should fetch from peer within deadline, last err=%v", err)
}
