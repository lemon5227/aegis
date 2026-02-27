import re

file_path = 'aegis-app/db.go'

with open(file_path, 'r') as f:
    content = f.read()

# Fix GetPostIndexByID
start_pattern_1 = r'func \(a \*App\) GetPostIndexByID\(postID string\) \(PostIndex, error\) \{'
end_pattern_1 = r'func \(a \*App\) GetMyPosts\(limit int, cursor string\) \(PostIndexPage, error\) \{'

replacement_1 = """func (a *App) GetPostIndexByID(postID string) (PostIndex, error) {
	if a.db == nil {
		return PostIndex{}, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return PostIndex{}, errors.New("post id is required")
	}

	viewerPubkey := ""
	if identity, err := a.getLocalIdentity(); err == nil {
		viewerPubkey = strings.TrimSpace(identity.PublicKey)
	}

	var item PostIndex
	err := a.db.QueryRow(`
		SELECT id, pubkey, title, SUBSTR(body, 1, 140) AS body_preview, content_cid, image_cid, thumb_cid, image_mime, image_size, image_width, image_height, score, timestamp, zone, sub_id, visibility
		FROM messages
		WHERE id = ?
		  AND (
			(zone = 'public' AND (visibility = 'normal' OR (pubkey = ? AND visibility != 'deleted')))
			OR (zone = 'private' AND pubkey = ?)
		  )
		LIMIT 1;
	`, postID, viewerPubkey, viewerPubkey).Scan(
		&item.ID,
		&item.Pubkey,
		&item.Title,
		&item.BodyPreview,
		&item.ContentCID,
		&item.ImageCID,
		&item.ThumbCID,
		&item.ImageMIME,
		&item.ImageSize,
		&item.ImageWidth,
		&item.ImageHeight,
		&item.Score,
		&item.Timestamp,
		&item.Zone,
		&item.SubID,
		&item.Visibility,
	)

	if err == nil {
		return item, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return PostIndex{}, err
	}

	// Try fetching from network
	status := a.GetP2PStatus()
	if !status.Started {
		return PostIndex{}, errors.New("post not found")
	}

	fetchErr := a.fetchPostFromNetwork(postID, 5*time.Second)
	if fetchErr != nil {
		return PostIndex{}, fetchErr
	}

	// Retry local fetch after successful network fetch
	err = a.db.QueryRow(`
		SELECT id, pubkey, title, SUBSTR(body, 1, 140) AS body_preview, content_cid, image_cid, thumb_cid, image_mime, image_size, image_width, image_height, score, timestamp, zone, sub_id, visibility
		FROM messages
		WHERE id = ?
		  AND (
			(zone = 'public' AND (visibility = 'normal' OR (pubkey = ? AND visibility != 'deleted')))
			OR (zone = 'private' AND pubkey = ?)
		  )
		LIMIT 1;
	`, postID, viewerPubkey, viewerPubkey).Scan(
		&item.ID,
		&item.Pubkey,
		&item.Title,
		&item.BodyPreview,
		&item.ContentCID,
		&item.ImageCID,
		&item.ThumbCID,
		&item.ImageMIME,
		&item.ImageSize,
		&item.ImageWidth,
		&item.ImageHeight,
		&item.Score,
		&item.Timestamp,
		&item.Zone,
		&item.SubID,
		&item.Visibility,
	)
	if err != nil {
		return PostIndex{}, err
	}
	return item, nil
}

"""

# Use regex to find the block
match_1 = re.search(f'({start_pattern_1}.*?){end_pattern_1}', content, re.DOTALL)
if match_1:
    content = content.replace(match_1.group(1), replacement_1)
else:
    print("Could not find GetPostIndexByID block")

# Fix duplicate getPublicPostDigest
# Finding the block between listPublicCommentDigestsByPostSince and listFavoriteOpsSince
start_pattern_2 = r'func \(a \*App\) listPublicCommentDigestsByPostSince\(postID string, sinceTimestamp int64, limit int\) \(\[\]SyncCommentDigest, error\) \{'
end_pattern_2 = r'func \(a \*App\) listFavoriteOpsSince\(pubkey string, sinceTimestamp int64, limit int\) \(\[\]FavoriteOpRecord, error\) \{'

# Need to be careful to match the end of listPublicCommentDigestsByPostSince
# We can search for the start of listPublicCommentDigestsByPostSince, then find the corresponding closing brace, but simpler is to search for the next function start.

match_2_start = re.search(start_pattern_2, content)
match_2_end = re.search(end_pattern_2, content)

if match_2_start and match_2_end:
    start_index = match_2_start.end()
    end_index = match_2_end.start()

    # Extract the chunk
    chunk = content[start_index:end_index]

    # We want to keep listPublicCommentDigestsByPostSince body intact.
    # It ends with a return and a brace.
    # So we search for the last '}' in the chunk? No, that's dangerous.

    # Better approach:
    # 1. Split content into lines.
    # 2. Find line number for listPublicCommentDigestsByPostSince definition.
    # 3. Walk down until we find the closing brace of that function (counting braces).
    # 4. Find line number for listFavoriteOpsSince definition.
    # 5. Replace everything between closing brace of (3) and start of (4).

    lines = content.splitlines(keepends=True)

    start_line_idx = -1
    for i, line in enumerate(lines):
        if 'func (a *App) listPublicCommentDigestsByPostSince' in line:
            start_line_idx = i
            break

    if start_line_idx != -1:
        brace_count = 0
        found_start = False
        end_function_idx = -1

        for i in range(start_line_idx, len(lines)):
            line = lines[i]
            brace_count += line.count('{')
            if brace_count > 0:
                found_start = True
            brace_count -= line.count('}')

            if found_start and brace_count == 0:
                end_function_idx = i
                break

        target_end_line_idx = -1
        for i in range(end_function_idx + 1, len(lines)):
            if 'func (a *App) listFavoriteOpsSince' in line: # Oops 'line' var from loop? No, use lines[i]
                pass
            if 'func (a *App) listFavoriteOpsSince' in lines[i]:
                target_end_line_idx = i
                break

        if end_function_idx != -1 and target_end_line_idx != -1:
            # We replace lines from end_function_idx + 1 to target_end_line_idx - 1

            new_code = """
func (a *App) getPublicPostDigest(postID string) (SyncPostDigest, error) {
	if a.db == nil {
		return SyncPostDigest{}, errors.New("database not initialized")
	}

	postID = strings.TrimSpace(postID)
	if postID == "" {
		return SyncPostDigest{}, errors.New("post id is required")
	}

	var digest SyncPostDigest
	var visibility string
	err := a.db.QueryRow(`
		SELECT id, pubkey, current_op_id, visibility, deleted_at_lamport, title, content_cid, image_cid, thumb_cid, image_mime, image_size, image_width, image_height, timestamp, lamport, sub_id
		FROM messages
		WHERE id = ? AND zone = 'public'
		LIMIT 1;
	`, postID).Scan(&digest.ID, &digest.Pubkey, &digest.OpID, &visibility, &digest.DeletedAtLamport, &digest.Title, &digest.ContentCID, &digest.ImageCID, &digest.ThumbCID, &digest.ImageMIME, &digest.ImageSize, &digest.ImageWidth, &digest.ImageHeight, &digest.Timestamp, &digest.Lamport, &digest.SubID)

	if err != nil {
		return SyncPostDigest{}, err
	}

	digest.Deleted = strings.EqualFold(strings.TrimSpace(visibility), "deleted")
	if digest.Deleted {
		digest.OpType = postOpTypeDelete
		digest.ContentCID = ""
	} else {
		digest.OpType = postOpTypeCreate
	}

	return digest, nil
}

func (a *App) getLatestFavoriteOpTimestamp(pubkey string) (int64, error) {
	if a.db == nil {
		return 0, errors.New("database not initialized")
	}

	var latest sql.NullInt64
	if err := a.db.QueryRow("SELECT MAX(created_at) FROM post_favorite_ops WHERE pubkey = ?", pubkey).Scan(&latest); err != nil {
		return 0, err
	}
	if !latest.Valid {
		return 0, nil
	}
	return latest.Int64, nil
}
"""
            # Construct new content
            new_lines = lines[:end_function_idx+1] + [new_code] + lines[target_end_line_idx:]
            content = "".join(new_lines)
        else:
            print("Could not identify ranges for part 2")
    else:
        print("Could not find listPublicCommentDigestsByPostSince")

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
