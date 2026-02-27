import re

file_path = 'aegis-app/p2p.go'
with open(file_path, 'r') as f:
    content = f.read()

# Replace any occurrence of messageTypeReport with literal string
content = content.replace('messageTypeReport', '"REPORT"')

# Remove any function that looks like BroadcastReport
content = re.sub(r'func \(a \*App\) BroadcastReport.*?\n}', '', content, flags=re.DOTALL)

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
