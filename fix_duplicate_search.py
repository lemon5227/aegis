import re

file_path = 'aegis-app/db.go'
with open(file_path, 'r') as f:
    content = f.read()

# Remove SearchPosts from db.go since I moved/reimplemented it in app.go
# Or better, remove the old one in db.go.
# The new one in app.go is using FTS.
# The old one in db.go (around line 3573) is using LIKE.

# Regex to remove the function
content = re.sub(r'func \(a \*App\) SearchPosts\(keyword string, subID string, limit int\) \(\[\]ForumMessage, error\) \{.*?^}', '', content, flags=re.MULTILINE|re.DOTALL)

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
