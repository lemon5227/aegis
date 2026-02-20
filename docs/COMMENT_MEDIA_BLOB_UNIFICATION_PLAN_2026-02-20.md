# Comment Media Blob Unification Plan (B1-B4)

## Objective

Unify comment image handling with the existing post media blob architecture so that:

- comment text stays clean (no giant base64 blobs in editor text),
- local and remote nodes exchange compact references (CID/URL),
- media fetch path is reused (`MEDIA_FETCH_REQUEST/RESPONSE`),
- A3/A4 governance and serve-gating semantics continue to apply,
- UX supports thumbnail preview, removal, and click-to-zoom.

## Scope

- Backend schema/model/protocol updates for structured comment attachments.
- Frontend composer and renderer updates for attachment chips + gallery.
- Backward compatibility for legacy markdown image comments.

## Design

1. **Structured attachments in comments**
   - Introduce `CommentAttachment` model.
   - `Comment` and comment sync payloads carry `attachments` list.
   - Supported kinds:
     - `media_cid` for local-uploaded images stored in `media_blobs`.
     - `external_url` for lightweight offloaded image links.

2. **Persistence model**
   - Add `comments.attachments_json` (`TEXT`, default `[]`).
   - Add `comment_media_refs(comment_id, content_cid)` for efficient reverse lookup.
   - Keep comment body as plain text only.

3. **Publish path**
   - Add `PublishCommentWithAttachments(...)` backend API.
   - Local file images are compressed and stored as media blobs, then represented as `media_cid` attachments.
   - External image URLs become `external_url` attachments.
   - Existing `PublishComment(...)` remains for compatibility and delegates with no attachments.

4. **Sync/anti-entropy**
   - `COMMENT` and `COMMENT_SYNC_RESPONSE` include structured attachments.
   - Receiver stores attachment metadata and media references.

5. **Serving and governance**
   - Extend media serve policy checks so comment-linked `media_cid` obeys shadow-ban A3/A4 semantics.
   - A blob is shareable if referenced by at least one allowed post/comment.

6. **Frontend UX**
   - Composer shows attached image thumbnails instead of inserting encoded text into textarea.
   - Comment submit sends body + structured attachments.
   - Comment list renders structured attachments (CID resolved via `GetMediaByCID`) and supports click-to-zoom.
   - Keep legacy markdown image rendering to avoid breaking historical comments.

## Execution Phases

- **B1**: Schema + backend models + persistence helpers.
- **B2**: Publish/process/sync wiring for comment attachments.
- **B3**: Frontend composer integration and renderer support.
- **B4**: Governance-aware media serving + verification.

## Verification

1. Local comment with uploaded image:
   - editor remains responsive,
   - comment sends successfully,
   - local and remote render thumbnail,
   - click opens zoom modal.
2. External URL attachment:
   - renders directly without local blob storage.
3. Legacy markdown image comments:
   - still render correctly.
4. Shadow-ban policy:
   - comment-linked media from disallowed content is not served to peers.
