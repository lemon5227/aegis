import re

file_path = 'aegis-app/p2p.go'
with open(file_path, 'r') as f:
    content = f.read()

# Just hardcode the string in the switch case
content = content.replace('case messageTypeReport:', 'case "REPORT":')

# And delete the function causing P2PMessage error
content = re.sub(r'func \(a \*App\) BroadcastReport.*?^}', '', content, flags=re.MULTILINE|re.DOTALL)

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
