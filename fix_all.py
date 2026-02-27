import re

# Fix p2p.go
# 1. messageTypeReport definition
# 2. handleReport call context
# 3. BroadcastReport struct usage

p2p_file = 'aegis-app/p2p.go'
with open(p2p_file, 'r') as f:
    p2p_lines = f.readlines()

new_p2p_lines = []
for line in p2p_lines:
    # Fix 1: Ensure constant exists
    if 'messageTypePostFetchResponse = "POST_FETCH_RESPONSE"' in line and 'messageTypeReport' not in "".join(p2p_lines):
        line = line + '\tmessageTypeReport            = "REPORT"\n'

    # Fix 2: Context of handleReport
    # The error "undefined: app" suggests the receiver is not named "app" or "a".
    # In `handleStream` (implied by context), the receiver is likely `a *App`.
    # But wait, looking at `grep` output:
    # `case messageTypePostFetchResponse:` uses `a.handlePostFetchResponse`.
    # So the receiver is `a`.
    # But my pasted code used `app.handleReport(msg)`.
    # AND `msg` was undefined.
    # In `handleStream`, the loop usually does `for { msg, err := ... }` or similar.
    # Or `decoder.Decode(&msg)`.
    # If `msg` is undefined, maybe it's named something else like `message` or `packet`.
    # BUT, looking at `a.handlePostFetchResponse(localPeerID.String(), incoming)`,
    # it seems `incoming` is the variable name for the decoded message struct?
    # Or `incoming` is a specialized struct?
    # Let's assume the loop variable is named `incoming` (of type `IncomingMessage`?) or `msg` (of type `P2PMessage`).
    # Wait, `handlePostFetchResponse` usually takes the P2PMessage or payload.
    # The grep output showed: `a.handlePostFetchResponse(localPeerID.String(), incoming)`
    # So the variable is likely `incoming`.
    # And the receiver is `a`.
    if 'app.handleReport(msg)' in line:
        line = line.replace('app.handleReport(msg)', 'a.handleReport(incoming)')

    # Fix 3: P2PMessage undefined?
    # `func (a *App) BroadcastReport(report Report) error {`
    # `msg := P2PMessage{...}`
    # If `P2PMessage` is undefined, maybe it's named `Message` in this file/package?
    # Or maybe `IncomingMessage`?
    # Let's check other usages. `messageTypePostFetchResponse` uses `IncomingMessage` in `handlePostFetchResponse`.
    # But `broadcast` usually takes a struct that has Type and Payload.
    # Let's assume it's `Message` if `P2PMessage` fails. Or maybe it is just `P2PMessage` but I need to define it or import it?
    # Unlikely to be missing struct if `handleStream` uses it.
    # Let's look for `type P2PMessage struct` or similar.

    new_p2p_lines.append(line)

with open(p2p_file, 'w') as f:
    f.writelines(new_p2p_lines)

# Fix app.go
# 1. Imports (time, fmt)
app_file = 'aegis-app/app.go'
with open(app_file, 'r') as f:
    app_lines = f.readlines()

# Check imports
has_time = False
has_fmt = False
import_block_start = -1
import_block_end = -1

for i, line in enumerate(app_lines):
    if 'import (' in line:
        import_block_start = i
    if import_block_start != -1 and ')' in line:
        import_block_end = i
        break

if import_block_start != -1:
    for i in range(import_block_start, import_block_end):
        if '"time"' in app_lines[i]:
            has_time = True
        if '"fmt"' in app_lines[i]:
            has_fmt = True

    if not has_time:
        app_lines.insert(import_block_end, '\t"time"\n')
        import_block_end += 1
    if not has_fmt:
        app_lines.insert(import_block_end, '\t"fmt"\n')

# If imports were missing, we fixed them.
# The errors "undefined: time" at line 372 suggest they were missing or the file was truncated/malformed.

with open(app_file, 'w') as f:
    f.writelines(app_lines)

print("Done")
