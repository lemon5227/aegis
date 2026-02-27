import re

file_path = 'aegis-app/p2p.go'
with open(file_path, 'r') as f:
    lines = f.readlines()

new_lines = []
skip = False
for line in lines:
    if 'func (a *App) BroadcastReport' in line:
        skip = True

    if not skip:
        new_lines.append(line)

    if skip and line.strip() == '}':
        skip = False

with open(file_path, 'w') as f:
    f.writelines(new_lines)

print("Done")
