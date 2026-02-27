import re

file_path = 'aegis-app/db.go'
with open(file_path, 'r') as f:
    content = f.read()

# I see double backticks: ``CREATE TRIGGER ...
# This is a syntax error.
content = content.replace('``CREATE TRIGGER', '`CREATE TRIGGER')

with open(file_path, 'w') as f:
    f.write(content)

print("Done")
