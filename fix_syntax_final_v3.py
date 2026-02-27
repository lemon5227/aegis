import re

file_path = 'aegis-app/p2p.go'
with open(file_path, 'r') as f:
    lines = f.readlines()

# Identify lines causing "non-declaration statement outside function body"
# This usually happens if braces are mismatched.
# I might have deleted a closing brace or inserted code outside a function.

# Let's try to count braces to find where it breaks.
balance = 0
for i, line in enumerate(lines):
    balance += line.count('{')
    balance -= line.count('}')
    if balance < 0:
        print(f"Brace mismatch at line {i+1}: {line.strip()}")
        # Reset balance to avoid cascading errors?
        balance = 0

# I suspect `handleStream` switch block got corrupted.
# Let's inspect around line 1870 and 1909.
