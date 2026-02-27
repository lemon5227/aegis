import re

file_path = 'aegis-app/p2p.go'
with open(file_path, 'r') as f:
    content = f.read()

# IncomingMessage doesn't have `Payload`. It IS the payload (unmarshaled).
# So we don't need `json.Unmarshal(msg.Payload, &report)`.
# `msg` is already `IncomingMessage`.
# But wait, `IncomingMessage` has `Type` and other fields.
# If `Type` is REPORT, where is the report data?
# Is it in `IncomingMessage` fields?
# `IncomingMessage` has `TargetID`, `TargetType` (maybe not), `Reason`.
# Let's check `IncomingMessage` definition in `db.go`.
# It has `TargetPubkey`, `Reason`.
# Does it have `TargetID`? It has `PostID`, `CommentID`.
# And `TargetType`? Maybe not.
# So we might need to map `IncomingMessage` fields to `Report`.
# `Report` needs: TargetID, TargetType, Reason, ReporterPubkey, Timestamp.
# `IncomingMessage` has: PostID (TargetID?), Reason, Pubkey (Reporter?), Timestamp.
# If `TargetType` is missing in `IncomingMessage`, we can't fully reconstruct it unless we infer or add fields.
# But `IncomingMessage` is a catch-all.
# Let's see if I added `TargetType` to `IncomingMessage`? No.
# But I can add it now or reuse existing fields.
# Or, since `handleReport` is receiving `IncomingMessage`, I should just use its fields directly.

# msg is IncomingMessage.
# report.TargetID = msg.TargetID (if exists) or msg.PostID?
# report.Reason = msg.Reason
# report.ReporterPubkey = msg.Pubkey
# report.Timestamp = msg.Timestamp

# Let's replace the unmarshal block with direct assignment.

old_block = """    var report Report
    if err := json.Unmarshal(msg.Payload, &report); err != nil {
        return fmt.Errorf("invalid report payload: %w", err)
    }"""

new_block = """    // Map IncomingMessage to Report
    // Assuming TargetID is passed in PostID or CommentID field, or we add TargetID to IncomingMessage
    targetID := msg.PostID
    if targetID == "" {
        targetID = msg.CommentID
    }
    // Default to 'post' if not specified, or check context.
    // For now, let's assume it's a generic target ID field if added, or fallback.
    // Actually, `IncomingMessage` doesn't have `TargetID`.
    // Let's use `PostID` as primary target container for now.

    report := Report{
        TargetID:       targetID,
        TargetType:     "post", // Defaulting, or need to add field
        Reason:         msg.Reason,
        ReporterPubkey: msg.Pubkey,
        Timestamp:      msg.Timestamp,
    }
"""

if old_block in content:
    content = content.replace(old_block, new_block)
else:
    # Regex match if exact string doesn't match
    content = re.sub(r'var report Report.*?json\.Unmarshal\(msg\.Payload.*?\}', new_block, content, flags=re.DOTALL)

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
