import re

file_path = 'aegis-app/p2p.go'

with open(file_path, 'r') as f:
    content = f.read()

# The previous script failed to find the constant because it might have looked for specific context.
# Let's just find the const block or the last messageType definition.

if 'const (' in content:
    # Try to find a known message type
    if 'messageTypePostFetchResponse' in content:
         content = content.replace('messageTypePostFetchResponse = "POST_FETCH_RESPONSE"', 'messageTypePostFetchResponse = "POST_FETCH_RESPONSE"\n\tmessageTypeReport            = "REPORT"')
    elif 'messageTypeIdentity      = "IDENTITY"' in content:
         content = content.replace('messageTypeIdentity      = "IDENTITY"', 'messageTypeIdentity      = "IDENTITY"\n\tmessageTypeReport            = "REPORT"')
    else:
        print("Could not find suitable constant to hook into")
else:
    print("Could not find const block")

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
