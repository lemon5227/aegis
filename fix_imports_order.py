file_path = 'aegis-app/p2p.go'
with open(file_path, 'r') as f:
    lines = f.readlines()

# Remove the incorrectly placed const
new_lines = []
const_decl = 'const messageTypeReport = "REPORT"\n'
for line in lines:
    if line == const_decl:
        continue
    new_lines.append(line)

# Add it back AFTER imports
import_end = -1
for i, line in enumerate(new_lines):
    if line.strip() == ')':
        import_end = i

if import_end != -1:
    new_lines.insert(import_end + 1, '\n' + const_decl)

with open(file_path, 'w') as f:
    f.writelines(new_lines)

print("Done")
