import re

file_path = 'aegis-app/p2p.go'
with open(file_path, 'r') as f:
    content = f.read()

# The error `syntax error: unexpected literal "REPORT", expected name`
# usually happens in a switch statement if the case label is malformed or if `case` keyword is repeated?
# `case "REPORT":` is valid Go if the switch is on a string.
# But looking at line 1956 in grep output:
# `			case "REPORT":`
# It seems fine.
# Wait, line 3499?
# The grep output showed line 1956. Where is 3499 coming from?
# Ah, the file is large. The grep output might be deceptive if there are multiple occurrences.
# Let's check line 3499.

# If I replaced ALL `messageTypeReport` with `"REPORT"`, maybe I replaced a const definition?
# `const "REPORT" = "REPORT"` -> Syntax error.

content = content.replace('const "REPORT" = "REPORT"', 'const messageTypeReport = "REPORT"')
# Restore usages to use the const
content = content.replace('case "REPORT":', 'case messageTypeReport:')

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
