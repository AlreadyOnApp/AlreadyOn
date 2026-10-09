#!/usr/bin/env python3
"""Select the shared build branch without discarding local work."""
import argparse
import subprocess
import sys


def main():
    """Keep both assistants on the same persistent development branch."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('command', choices=['start'])
    parser.add_argument('name', nargs='?', default='build', choices=['build'])
    parser.parse_args()
    state = subprocess.check_output(['git', 'status', '--porcelain'], text=True)
    if state:
        sys.exit('Commit existing changes before switching the shared checkout')
    subprocess.run(['git', 'fetch', '--prune', 'origin'], check=True)
    exists = subprocess.run(['git', 'show-ref', '--verify', '--quiet', 'refs/heads/build']).returncode == 0
    command = ['git', 'switch', 'build'] if exists else ['git', 'switch', '--no-track', '-c', 'build', 'origin/dev']
    subprocess.run(command, check=True)
    print('Shared build branch selected; coordinate changes with the other assistant')


if __name__ == '__main__':
    main()
