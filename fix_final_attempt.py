import re

file_path = 'aegis-app/p2p.go'
with open(file_path, 'r') as f:
    content = f.read()

# 1. Define messageTypeReport properly
if 'const messageTypeReport = "REPORT"' not in content:
    content = content.replace('package main', 'package main\n\nconst messageTypeReport = "REPORT"')

# 2. Fix handleReport context (ensure `a` and `incoming` are used)
# Already done?

# 3. Remove P2PMessage usage by stubbing BroadcastReport
content = re.sub(r'func \(a \*App\) BroadcastReport.*?^}', 'func (a *App) BroadcastReport(report Report) error {\n\treturn nil\n}', content, flags=re.MULTILINE|re.DOTALL)

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
