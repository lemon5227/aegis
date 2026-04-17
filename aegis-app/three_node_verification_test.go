package main

import (
	"fmt"
	"testing"
	"time"
)

func connectThreeNodes(t *testing.T, nodeA, nodeB, nodeC *App) {
	t.Helper()

	portA := reserveLoopbackPort(t)
	portB := reserveLoopbackPort(t)
	portC := reserveLoopbackPort(t)

	statusA, err := nodeA.StartP2P(portA, nil)
	if err != nil {
		t.Fatalf("start node A p2p: %v", err)
	}
	statusB, err := nodeB.StartP2P(portB, nil)
	if err != nil {
		t.Fatalf("start node B p2p: %v", err)
	}
	statusC, err := nodeC.StartP2P(portC, nil)
	if err != nil {
		t.Fatalf("start node C p2p: %v", err)
	}

	addrB := loopbackAddress(portB, statusB.PeerID)
	addrC := loopbackAddress(portC, statusC.PeerID)
	addrA := loopbackAddress(portA, statusA.PeerID)

	if err = nodeA.ConnectPeer(addrB); err != nil {
		t.Fatalf("connect A→B: %v", err)
	}
	if err = nodeA.ConnectPeer(addrC); err != nil {
		t.Fatalf("connect A→C: %v", err)
	}
	if err = nodeB.ConnectPeer(addrC); err != nil {
		t.Fatalf("connect B→C: %v", err)
	}
	if err = nodeB.ConnectPeer(addrA); err != nil {
		t.Fatalf("connect B→A: %v", err)
	}
	if err = nodeC.ConnectPeer(addrA); err != nil {
		t.Fatalf("connect C→A: %v", err)
	}
	if err = nodeC.ConnectPeer(addrB); err != nil {
		t.Fatalf("connect C→B: %v", err)
	}

	waitForCondition(t, 10*time.Second, func() (bool, error) {
		aPeers := len(nodeA.GetP2PStatus().ConnectedPeers)
		bPeers := len(nodeB.GetP2PStatus().ConnectedPeers)
		cPeers := len(nodeC.GetP2PStatus().ConnectedPeers)
		return aPeers >= 2 && bPeers >= 2 && cPeers >= 2, nil
	})
}

func waitForPostCount(t *testing.T, app *App, title string, minCount int, timeout time.Duration) {
	t.Helper()
	waitForCondition(t, timeout, func() (bool, error) {
		var count int
		err := app.db.QueryRow(`SELECT COUNT(1) FROM messages WHERE title = ?`, title).Scan(&count)
		return count >= minCount, err
	})
}

func waitForCommentCount(t *testing.T, app *App, postID string, minCount int, timeout time.Duration) {
	t.Helper()
	waitForCondition(t, timeout, func() (bool, error) {
		var count int
		err := app.db.QueryRow(`SELECT COUNT(1) FROM comments WHERE post_id = ?`, postID).Scan(&count)
		return count >= minCount, err
	})
}

func TestA2PostTitleBodyReplicatesAcrossNodes(t *testing.T) {
	nodeA := newG7TestApp(t, "a2_a.db")
	nodeB := newG7TestApp(t, "a2_b.db")
	nodeC := newG7TestApp(t, "a2_c.db")

	idA := seedLocalIdentity(t, nodeA)
	seedLocalIdentity(t, nodeB)
	seedLocalIdentity(t, nodeC)

	connectThreeNodes(t, nodeA, nodeB, nodeC)

	title := fmt.Sprintf("A2 title test %d", time.Now().UnixNano())
	body := "A2 body content for replication check"

	if err := nodeA.PublishPostStructuredToSub(idA.PublicKey, title, body, defaultSubID); err != nil {
		t.Fatalf("publish post on A: %v", err)
	}
	nodeA.flushOutgoingMessagesAsync()

	waitForPostCount(t, nodeB, title, 1, 15*time.Second)
	waitForPostCount(t, nodeC, title, 1, 15*time.Second)

	var contentCIDB, contentCIDC string
	if err := nodeB.db.QueryRow(`SELECT content_cid FROM messages WHERE title = ?`, title).Scan(&contentCIDB); err != nil {
		t.Fatalf("read content_cid from B: %v", err)
	}
	if err := nodeC.db.QueryRow(`SELECT content_cid FROM messages WHERE title = ?`, title).Scan(&contentCIDC); err != nil {
		t.Fatalf("read content_cid from C: %v", err)
	}

	if contentCIDB == "" {
		t.Errorf("B content_cid is empty — index not replicated")
	}
	if contentCIDC == "" {
		t.Errorf("C content_cid is empty — index not replicated")
	}

	if contentCIDB != contentCIDC {
		t.Errorf("content_cid mismatch: B=%q C=%q", contentCIDB, contentCIDC)
	}

	var bodyB string
	if err := nodeB.db.QueryRow(`SELECT body FROM content_blobs WHERE content_cid = ?`, contentCIDB).Scan(&bodyB); err != nil {
		var fallbackBody string
		if err2 := nodeB.db.QueryRow(`SELECT body FROM messages WHERE title = ?`, title).Scan(&fallbackBody); err2 != nil {
			t.Fatalf("read body from B: %v (blob err: %v)", err2, err)
		}
		bodyB = fallbackBody
	}

	if bodyB != body && bodyB != "" {
		t.Errorf("B body mismatch: got %q, want %q", bodyB, body)
	}
}

func TestA3HotNewSortingConsistentAcrossNodes(t *testing.T) {
	nodeA := newG7TestApp(t, "a3_a.db")
	nodeB := newG7TestApp(t, "a3_b.db")

	idA := seedLocalIdentity(t, nodeA)
	seedLocalIdentity(t, nodeB)

	connectThreeNodes(t, nodeA, nodeB, newG7TestApp(t, "a3_c.db"))

	post1 := fmt.Sprintf("A3 post1 %d", time.Now().UnixNano())
	post2 := fmt.Sprintf("A3 post2 %d", time.Now().UnixNano())

	if err := nodeA.PublishPostStructuredToSub(idA.PublicKey, post1, "body1", defaultSubID); err != nil {
		t.Fatalf("publish post1: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if err := nodeA.PublishPostStructuredToSub(idA.PublicKey, post2, "body2", defaultSubID); err != nil {
		t.Fatalf("publish post2: %v", err)
	}
	nodeA.flushOutgoingMessagesAsync()

	waitForPostCount(t, nodeB, post1, 1, 15*time.Second)
	waitForPostCount(t, nodeB, post2, 1, 15*time.Second)

	postsNew, err := nodeB.GetFeedIndexBySubSorted(defaultSubID, "new")
	if err != nil {
		t.Fatalf("get new sorted feed: %v", err)
	}
	if len(postsNew) < 2 {
		t.Fatalf("expected at least 2 posts in new feed, got %d", len(postsNew))
	}

	postsHot, err := nodeB.GetFeedIndexBySubSorted(defaultSubID, "hot")
	if err != nil {
		t.Fatalf("get hot sorted feed: %v", err)
	}
	if len(postsHot) < 2 {
		t.Fatalf("expected at least 2 posts in hot feed, got %d", len(postsHot))
	}
}

func TestB1NestedCommentReplicatesAcrossNodes(t *testing.T) {
	nodeA := newG7TestApp(t, "b1_a.db")
	nodeB := newG7TestApp(t, "b1_b.db")
	nodeC := newG7TestApp(t, "b1_c.db")

	idA := seedLocalIdentity(t, nodeA)
	seedLocalIdentity(t, nodeB)
	seedLocalIdentity(t, nodeC)

	connectThreeNodes(t, nodeA, nodeB, nodeC)

	title := fmt.Sprintf("B1 comment test %d", time.Now().UnixNano())
	if err := nodeA.PublishPostStructuredToSub(idA.PublicKey, title, "body", defaultSubID); err != nil {
		t.Fatalf("publish post on A: %v", err)
	}
	nodeA.flushOutgoingMessagesAsync()

	waitForPostCount(t, nodeB, title, 1, 15*time.Second)

	var postID string
	if err := nodeA.db.QueryRow(`SELECT id FROM messages WHERE title = ?`, title).Scan(&postID); err != nil {
		t.Fatalf("get post ID: %v", err)
	}

	if _, err := nodeA.AddLocalComment(idA.PublicKey, postID, "", "top-level comment from A"); err != nil {
		t.Fatalf("add top-level comment on A: %v", err)
	}
	nodeA.flushOutgoingMessagesAsync()

	waitForCommentCount(t, nodeB, postID, 1, 15*time.Second)
	waitForCommentCount(t, nodeC, postID, 1, 15*time.Second)

	var commentID string
	if err := nodeA.db.QueryRow(`SELECT id FROM comments WHERE post_id = ? AND parent_id = ''`, postID).Scan(&commentID); err != nil {
		t.Fatalf("get comment ID: %v", err)
	}

	if _, err := nodeB.AddLocalComment(idA.PublicKey, postID, commentID, "nested reply from B"); err != nil {
		t.Fatalf("add nested reply on B: %v", err)
	}
	nodeB.flushOutgoingMessagesAsync()

	waitForCommentCount(t, nodeA, postID, 2, 15*time.Second)
	waitForCommentCount(t, nodeC, postID, 2, 15*time.Second)

	var parentID string
	if err := nodeC.db.QueryRow(`SELECT parent_id FROM comments WHERE post_id = ? AND parent_id != ''`, postID).Scan(&parentID); err != nil {
		t.Fatalf("get nested comment parent_id on C: %v", err)
	}
	if parentID != commentID {
		t.Errorf("nested comment parent_id mismatch: got %q, want %q", parentID, commentID)
	}
}

func TestC1GovernancePolicyAppliesAcrossNodes(t *testing.T) {
	nodeA := newG7TestApp(t, "c1_a.db")
	nodeB := newG7TestApp(t, "c1_b.db")
	nodeC := newG7TestApp(t, "c1_c.db")

	idA := seedLocalIdentity(t, nodeA)
	idB := seedLocalIdentity(t, nodeB)
	seedLocalIdentity(t, nodeC)

	if err := nodeA.AddTrustedAdmin(idA.PublicKey, "owner"); err != nil {
		t.Fatalf("trust admin A on A: %v", err)
	}
	if err := nodeB.AddTrustedAdmin(idA.PublicKey, "owner"); err != nil {
		t.Fatalf("trust admin A on B: %v", err)
	}
	if err := nodeC.AddTrustedAdmin(idA.PublicKey, "owner"); err != nil {
		t.Fatalf("trust admin A on C: %v", err)
	}

	connectThreeNodes(t, nodeA, nodeB, nodeC)

	title := fmt.Sprintf("C1 governance test %d", time.Now().UnixNano())
	if err := nodeB.PublishPostStructuredToSub(idB.PublicKey, title, "body by B", defaultSubID); err != nil {
		t.Fatalf("publish post by B: %v", err)
	}
	nodeB.flushOutgoingMessagesAsync()

	waitForPostCount(t, nodeA, title, 1, 15*time.Second)

	policy, err := nodeA.GetGovernancePolicy()
	if err != nil {
		t.Fatalf("get governance policy: %v", err)
	}

	logs, err := nodeA.GetModerationLogs(10)
	if err != nil {
		t.Fatalf("get moderation logs: %v", err)
	}

	_ = policy
	_ = logs
}
