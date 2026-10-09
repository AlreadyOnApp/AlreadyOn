#!/usr/bin/env python3
"""Parse tracked shell, Python, JavaScript and JSON without executing source."""
import ast
import json
import pathlib
import subprocess

for name in subprocess.check_output(['git', 'ls-files', '-z'], text=True).split('\0'):
    path = pathlib.Path(name)
    if not path.is_file():
        continue
    if path.suffix == '.sh':
        subprocess.run(['bash', '-n', str(path)], check=True)
    elif path.suffix == '.py':
        ast.parse(path.read_text(), filename=name)
    elif path.suffix in {'.js', '.cjs', '.mjs'}:
        subprocess.run(['node', '--check', str(path)], check=True)
    elif path.suffix == '.json':
        # TypeScript validates its own JSONC configs during tsc -b.
        if path.parent == pathlib.Path('web') and path.name.startswith('tsconfig'):
            continue
        try:
            json.loads(path.read_text())
        except ValueError as error:
            raise ValueError(f'{name}: {error}') from error
print('Tracked shell, Python, JavaScript and JSON syntax OK')
