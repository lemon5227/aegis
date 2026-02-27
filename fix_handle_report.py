import re

file_path = 'aegis-app/p2p.go'
with open(file_path, 'r') as f:
    content = f.read()

# The error is in handleReport signature!
# func (a *App) handleReport(msg P2PMessage) error {
# It should receive `IncomingMessage` or whatever `handleStream` provides.
# In `handleStream`, we likely use `IncomingMessage`.
# grep output showed: `if err := a.handleReport(incoming);`
# So `incoming` is likely `IncomingMessage`.

content = content.replace('func (a *App) handleReport(msg P2PMessage) error {', 'func (a *App) handleReport(msg IncomingMessage) error {')

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
