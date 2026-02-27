import re

file_path = 'aegis-app/app.go'
with open(file_path, 'r') as f:
    content = f.read()

# The incorrect line is: token = strings.ReplaceAll(token, """, """")
# It should be: token = strings.ReplaceAll(token, "\"", "\"\"")
# I need to escape the quotes properly for Go.

bad_line = 'token = strings.ReplaceAll(token, """, """")'
good_line = 'token = strings.ReplaceAll(token, "\\"", "\\"\\"")'

content = content.replace(bad_line, good_line)

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
