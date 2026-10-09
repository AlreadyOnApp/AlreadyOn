#!/usr/bin/env python3
"""Open a promotion PR for an exact dev commit or the current test head."""
import argparse
import json
import subprocess


def run(*args):
    """Use argument arrays so branch names and user input never become shell code."""
    return subprocess.check_output(args, text=True).strip()


def main():
    """Create an immutable source ref; branch policy verifies the merged tree again."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('target', choices=['test', 'main'])
    parser.add_argument('commit', nargs='?', help='Required dev commit for test; omitted for main')
    args = parser.parse_args()
    if (args.target == 'test') != bool(args.commit):
        parser.error('test requires a selected dev commit; main always selects the current test head')
    run('git', 'fetch', 'origin', 'dev', 'test', 'main')
    upstream = 'origin/dev' if args.target == 'test' else 'origin/test'
    selected = run('git', 'rev-parse', '--verify', (args.commit or upstream) + '^{commit}')
    subprocess.run(['git', 'merge-base', '--is-ancestor', selected, upstream], check=True)
    repo = run('gh', 'repo', 'view', '--json', 'nameWithOwner', '--jq', '.nameWithOwner')
    branch = 'promotion/' + args.target + '-' + selected[:12]
    payload = json.dumps({'ref': 'refs/heads/' + branch, 'sha': selected})
    subprocess.run(['gh', 'api', '--method', 'POST', f'repos/{repo}/git/refs', '--input', '-'],
                   input=payload, text=True, check=True, stdout=subprocess.DEVNULL)
    body = (f'Promote exact source commit `{selected}` from `{upstream}` to `{args.target}`.\n\n'
            'Merge with **Create a merge commit**. Do not squash/rebase or add changes to this branch. '
            'The merged tree must equal the selected source. Merging retags the already tested image; nothing is rebuilt or retested.\n')
    subprocess.run(['gh', 'pr', 'create', '--base', args.target, '--head', branch,
                    '--title', f'Promote {selected[:12]} to {args.target}', '--body', body], check=True)


if __name__ == '__main__':
    main()
