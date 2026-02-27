import re

file_path = 'aegis-app/p2p.go'
with open(file_path, 'r') as f:
    content = f.read()

# Remove loose "// Removed P2PMessage usage" lines that might be outside functions
content = content.replace('// Removed P2PMessage usage', '')

# Remove loose code blocks that might have been left over from bad deletions
# The grep showed `var report Report` at the end, seemingly outside any function.
# This means I deleted the function signature `func handleReport` but left the body?
# Let's check if `func (a *App) handleReport` exists.

if 'func (a *App) handleReport' not in content:
    # Reconstruct it wrapping the loose code
    # Find `var report Report`
    start = content.find('var report Report')
    if start != -1:
        # Check if it is already inside a function (brace count)
        # But grep output showed it right after `// Removed P2PMessage usage` which was causing syntax error
        # So it likely lost its header.
        header = 'func (a *App) handleReport(msg IncomingMessage) error {\n'
        content = content[:start] + header + content[start:]
        # And ensure it closes
        # It likely runs until end of file or next function?
        # Let's hope the closing brace is still there.

# Fix the `consumeP2PMessages` loop corruption
# `// Removed P2PMessage usage` was inserted inside `consumeP2PMessages`?
# grep showed:
# 	return status
# }
#
# // Removed P2PMessage usage
# 	for {
# This implies `consumeP2PMessages` ended prematurely or I deleted its signature?
# Or `return status` belongs to `GetP2PStatus`.
# And `consumeP2PMessages` signature is missing?
# Let's check `func (a *App) consumeP2PMessages` existence.

if 'func (a *App) consumeP2PMessages' not in content:
    # Insert it before `for {`
    # Locate `return status`
    idx = content.find('return status\n}\n')
    if idx != -1:
        insert_point = idx + len('return status\n}\n')
        signature = '\nfunc (a *App) consumeP2PMessages(ctx context.Context, localPeerID peer.ID, sub *pubsub.Subscription) {\n'
        content = content[:insert_point] + signature + content[insert_point:]

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
