package main

import (
	"fmt"
	"path/filepath"
	"testing"
)

func newSoakApp(t *testing.T, name string) *App {
	t.Helper()
	app := NewApp()
	app.SetDatabasePath(filepath.Join(t.TempDir(), fmt.Sprintf("%s.db", name)))
	if err := app.initDatabase(); err != nil {
		t.Fatalf("init database %s: %v", name, err)
	}
	t.Cleanup(func() {
		if app.db != nil {
			_ = app.db.Close()
		}
	})
	return app
}

func syncPublicState(t *testing.T, source *App, target *App) {
	t.Helper()

	postDigests, err := source.listPublicPostDigestsSince(0, 5000)
	if err != nil {
		t.Fatalf("list post digests: %v", err)
	}
	for _, digest := range postDigests {
		if _, err = target.upsertPublicPostIndexFromDigest(digest); err != nil {
			t.Fatalf("apply post digest %s: %v", digest.ID, err)
		}
	}

	commentDigests, err := source.listPublicCommentDigestsSince(0, 5000)
	if err != nil {
		t.Fatalf("list comment digests: %v", err)
	}
	for _, digest := range commentDigests {
		if digest.Deleted || normalizeOperationType(digest.OpType, postOpTypeCreate) == postOpTypeDelete {
			deleteLamport := digest.Lamport
			if digest.DeletedAtLamport > deleteLamport {
				deleteLamport = digest.DeletedAtLamport
			}
			if err = target.upsertCommentTombstone(digest.ID, digest.PostID, digest.Pubkey, digest.Timestamp, deleteLamport, digest.OpID); err != nil {
				t.Fatalf("apply comment tombstone %s: %v", digest.ID, err)
			}
			continue
		}

		if _, err = target.insertComment(Comment{
			ID:          digest.ID,
			PostID:      digest.PostID,
			ParentID:    digest.ParentID,
			Pubkey:      digest.Pubkey,
			OpID:        digest.OpID,
			Body:        digest.Body,
			Attachments: digest.Attachments,
			Score:       digest.Score,
			Timestamp:   digest.Timestamp,
			Lamport:     digest.Lamport,
		}); err != nil {
			t.Fatalf("apply comment digest %s: %v", digest.ID, err)
		}
	}
}

func snapshotPublicPosts(t *testing.T, app *App) []string {
	t.Helper()
	rows, err := app.db.Query(`
		SELECT id, visibility, lamport, current_op_id, deleted_at_lamport
		FROM messages
		WHERE zone = 'public'
		ORDER BY id ASC;
	`)
	if err != nil {
		t.Fatalf("query post snapshot: %v", err)
	}
	defer rows.Close()

	out := make([]string, 0)
	for rows.Next() {
		var id, visibility, opID string
		var lamport, deletedAtLamport int64
		if err = rows.Scan(&id, &visibility, &lamport, &opID, &deletedAtLamport); err != nil {
			t.Fatalf("scan post snapshot: %v", err)
		}
		out = append(out, fmt.Sprintf("%s|%s|%d|%s|%d", id, visibility, lamport, opID, deletedAtLamport))
	}
	if err = rows.Err(); err != nil {
		t.Fatalf("iterate post snapshot: %v", err)
	}
	return out
}

func snapshotComments(t *testing.T, app *App) []string {
	t.Helper()
	rows, err := app.db.Query(`
		SELECT id, post_id, deleted_at, lamport, current_op_id, deleted_at_lamport
		FROM comments
		ORDER BY id ASC;
	`)
	if err != nil {
		t.Fatalf("query comment snapshot: %v", err)
	}
	defer rows.Close()

	out := make([]string, 0)
	for rows.Next() {
		var id, postID, opID string
		var deletedAt, lamport, deletedAtLamport int64
		if err = rows.Scan(&id, &postID, &deletedAt, &lamport, &opID, &deletedAtLamport); err != nil {
			t.Fatalf("scan comment snapshot: %v", err)
		}
		out = append(out, fmt.Sprintf("%s|%s|%d|%d|%s|%d", id, postID, deletedAt, lamport, opID, deletedAtLamport))
	}
	if err = rows.Err(); err != nil {
		t.Fatalf("iterate comment snapshot: %v", err)
	}
	return out
}

func assertConvergedSnapshots(t *testing.T, apps ...*App) {
	t.Helper()
	basePosts := snapshotPublicPosts(t, apps[0])
	baseComments := snapshotComments(t, apps[0])
	for i := 1; i < len(apps); i++ {
		posts := snapshotPublicPosts(t, apps[i])
		if fmt.Sprint(posts) != fmt.Sprint(basePosts) {
			t.Fatalf("post snapshot divergence node=%d base=%v got=%v", i, basePosts, posts)
		}
		comments := snapshotComments(t, apps[i])
		if fmt.Sprint(comments) != fmt.Sprint(baseComments) {
			t.Fatalf("comment snapshot divergence node=%d base=%v got=%v", i, baseComments, comments)
		}
	}
}

func TestLamportThreeNodeSoakConvergence(t *testing.T) {
	const rounds = 24

	for round := 1; round <= rounds; round++ {
		t.Run(fmt.Sprintf("round_%02d", round), func(t *testing.T) {
			nodeA := newSoakApp(t, fmt.Sprintf("nodeA_%d", round))
			nodeB := newSoakApp(t, fmt.Sprintf("nodeB_%d", round))
			nodeC := newSoakApp(t, fmt.Sprintf("nodeC_%d", round))
			author := "author-pubkey"

			postID := fmt.Sprintf("post-soak-%d", round)
			commentID := fmt.Sprintf("comment-soak-%d", round)

			_, err := nodeA.insertMessage(ForumMessage{
				ID:        postID,
				Pubkey:    author,
				OpID:      fmt.Sprintf("%s:%s:%d:%s", postID, author, 10, "create"),
				Title:     "seed",
				Body:      "seed-body",
				Timestamp: 10,
				Lamport:   10,
				Zone:      "public",
				SubID:     defaultSubID,
			})
			if err != nil {
				t.Fatalf("seed post: %v", err)
			}
			_, err = nodeA.insertComment(Comment{
				ID:        commentID,
				PostID:    postID,
				Pubkey:    author,
				OpID:      fmt.Sprintf("%s:%s:%d:%s", commentID, author, 11, "create"),
				Body:      "seed-comment",
				Timestamp: 11,
				Lamport:   11,
			})
			if err != nil {
				t.Fatalf("seed comment: %v", err)
			}

			syncPublicState(t, nodeA, nodeB)

			_, err = nodeC.insertMessage(ForumMessage{
				ID:        postID,
				Pubkey:    author,
				OpID:      fmt.Sprintf("%s:%s:%d:%s", postID, author, 15, "stale"),
				Title:     "stale",
				Body:      "stale-body",
				Timestamp: 15,
				Lamport:   15,
				Zone:      "public",
				SubID:     defaultSubID,
			})
			if err != nil {
				t.Fatalf("offline stale create: %v", err)
			}

			deleteOpID := fmt.Sprintf("%s:%s:%d:%s", postID, author, 20, "zz")
			if err = nodeB.deleteLocalPostAsAuthor(author, postID, 20, 20, deleteOpID); err != nil {
				t.Fatalf("delete on nodeB: %v", err)
			}
			if err = nodeB.upsertCommentTombstone(commentID, postID, author, 21, 21, fmt.Sprintf("%s:%s:%d:%s", commentID, author, 21, "zz")); err != nil {
				t.Fatalf("delete comment on nodeB: %v", err)
			}

			syncPublicState(t, nodeB, nodeA)
			syncPublicState(t, nodeB, nodeC)
			syncPublicState(t, nodeC, nodeA)
			syncPublicState(t, nodeA, nodeB)
			syncPublicState(t, nodeA, nodeC)

			_, err = nodeC.insertMessage(ForumMessage{
				ID:        postID,
				Pubkey:    author,
				OpID:      fmt.Sprintf("%s:%s:%d:%s", postID, author, 18, "late-stale"),
				Title:     "late-stale",
				Body:      "late-stale-body",
				Timestamp: 18,
				Lamport:   18,
				Zone:      "public",
				SubID:     defaultSubID,
			})
			if err != nil {
				t.Fatalf("late stale replay: %v", err)
			}

			syncPublicState(t, nodeA, nodeB)
			syncPublicState(t, nodeB, nodeC)
			syncPublicState(t, nodeC, nodeA)

			_, err = nodeA.insertMessage(ForumMessage{
				ID:        postID,
				Pubkey:    author,
				OpID:      fmt.Sprintf("%s:%s:%d:%s", postID, author, 30, "a"),
				Title:     "concurrent-update",
				Body:      "concurrent-body",
				Timestamp: 30,
				Lamport:   30,
				Zone:      "public",
				SubID:     defaultSubID,
			})
			if err != nil {
				t.Fatalf("concurrent update on A: %v", err)
			}
			if err = nodeB.deleteLocalPostAsAuthor(author, postID, 30, 30, fmt.Sprintf("%s:%s:%d:%s", postID, author, 30, "z")); err != nil {
				t.Fatalf("concurrent delete on B: %v", err)
			}

			syncPublicState(t, nodeA, nodeB)
			syncPublicState(t, nodeB, nodeC)
			syncPublicState(t, nodeC, nodeA)
			syncPublicState(t, nodeB, nodeA)
			syncPublicState(t, nodeA, nodeC)

			assertConvergedSnapshots(t, nodeA, nodeB, nodeC)

			var visibility string
			var lamport int64
			var currentOpID string
			if err = nodeA.db.QueryRow(`SELECT visibility, lamport, current_op_id FROM messages WHERE id = ?;`, postID).Scan(&visibility, &lamport, &currentOpID); err != nil {
				t.Fatalf("read final post state: %v", err)
			}
			if visibility != "deleted" {
				t.Fatalf("expected final post deleted, got visibility=%s", visibility)
			}
			if lamport != 30 {
				t.Fatalf("expected final lamport 30, got %d", lamport)
			}
			if currentOpID != fmt.Sprintf("%s:%s:%d:%s", postID, author, 30, "z") {
				t.Fatalf("expected delete op to win at equal lamport, got op=%s", currentOpID)
			}

			var opCount int
			if err = nodeA.db.QueryRow(`SELECT COUNT(1) FROM entity_ops WHERE entity_type = ? AND entity_id = ?;`, entityTypePost, postID).Scan(&opCount); err != nil {
				t.Fatalf("count entity ops: %v", err)
			}
			if opCount == 0 {
				t.Fatalf("expected entity_ops records for post %s", postID)
			}
		})
	}
}
