#!/usr/bin/env python3
"""Serialize dev PR pipelines: at most two at once, the second waits, then each self-updates from dev.

A PR run waits while one earlier dev PR run is active and fails fast when two are.
When its turn comes it merges the current dev head into the PR branch and pushes
that commit, so every later job tests and merges the combined tree. Pushes made
with the workflow token start no new run, so this run continues on the new commit.
"""
import base64
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import subprocess
import sys
import time
import urllib.request

POLL_SECONDS = 300
MAX_WAIT_SECONDS = 4 * 3600
QUEUE_LIMIT = 2


def api(path, body=None):
    """Call the REST API with the job token; body switches the request to POST."""
    request = urllib.request.Request(
        f"https://api.github.com/repos/{os.environ['GITHUB_REPOSITORY']}/{path}",
        data=None if body is None else json.dumps(body).encode(),
        headers={'Authorization': 'Bearer ' + os.environ['GH_TOKEN'], 'Accept': 'application/vnd.github+json'})
    with urllib.request.urlopen(request, timeout=30) as response:
        return json.load(response)


def order(run):
    """Re-runs restart run_started_at, so a re-queued run joins the back of the line."""
    return (run.get('run_started_at') or run['created_at'], run['id'])


def ahead_of(runs, me, number):
    """Active dev PR runs of other PRs that started before this run, oldest first."""
    queued = []
    for run in runs:
        prs = run.get('pull_requests') or []
        if run['status'] == 'completed' or run['id'] == me['id'] or not prs:
            continue
        if prs[0]['base']['ref'] != 'dev' or prs[0]['number'] == number:
            continue
        if order(run) < order(me):
            queued.append(run)
    return sorted(queued, key=order)


def wait_for_turn(number):
    """Return once no earlier dev PR run is active; exit when the queue is full or stuck."""
    me = api(f"actions/runs/{os.environ['GITHUB_RUN_ID']}")
    deadline = time.monotonic() + MAX_WAIT_SECONDS
    while True:
        runs = api('actions/workflows/pull-request.yml/runs?event=pull_request&per_page=100')['workflow_runs']
        ahead = ahead_of(runs, me, number)
        if len(ahead) >= QUEUE_LIMIT:
            sys.exit(f'::error::Queue full: {len(ahead)} dev PRs are already in progress. '
                     'Re-run this workflow or push again once one finishes.')
        if not ahead:
            return
        if time.monotonic() > deadline:
            sys.exit('::error::Queue timeout: the earlier PR did not finish within 4 hours. Re-run this workflow.')
        print(f"{datetime.now(timezone.utc):%H:%M} UTC: waiting for PR #{ahead[0]['pull_requests'][0]['number']} "
              f"({ahead[0]['html_url']}); next check in {POLL_SECONDS // 60} minutes", flush=True)
        time.sleep(POLL_SECONDS)


def git(*args, auth=False):
    """Run git with the job token supplied per command; checkout credentials are not persisted."""
    env = dict(os.environ)
    if auth:
        token = base64.b64encode(('x-access-token:' + os.environ['GH_TOKEN']).encode()).decode()
        env.update(GIT_CONFIG_COUNT='1', GIT_CONFIG_KEY_0='http.https://github.com/.extraheader',
                   GIT_CONFIG_VALUE_0='AUTHORIZATION: basic ' + token)
    return subprocess.run(['git', *args], env=env, check=False, text=True, capture_output=True)


def update_from_dev(pr):
    """Merge the current dev head into the PR branch and return the commit to test."""
    fetched = git('fetch', '--quiet', 'origin', 'dev', auth=True)
    if fetched.returncode:
        sys.exit(fetched.stderr)
    if git('merge-base', '--is-ancestor', 'FETCH_HEAD', 'HEAD').returncode == 0:
        print('Branch already contains the current dev head')
        return git('rev-parse', 'HEAD').stdout.strip()
    git('config', 'user.name', 'github-actions[bot]')
    git('config', 'user.email', '41898282+github-actions[bot]@users.noreply.github.com')
    merged = git('merge', '--no-edit', '-m', 'Merge current dev into ' + pr['head']['ref'], 'FETCH_HEAD')
    if merged.returncode:
        git('merge', '--abort')
        api(f"issues/{pr['number']}/comments", {'body': (
            'The PR queue could not merge the current `dev` head into this branch because of a conflict. '
            'Merge `origin/dev` locally, resolve it and push to restart the pipeline.')})
        sys.exit('::error::Merge conflict with the current dev head')
    pushed = git('push', '--quiet', 'origin', 'HEAD:refs/heads/' + pr['head']['ref'], auth=True)
    if pushed.returncode:
        # The workflow token cannot push commits that change workflow files, and a
        # concurrent push to the branch also rejects this one.
        sys.exit('::error::Could not push the dev update to the PR branch. Merge origin/dev locally and push. '
                 + pushed.stderr.strip())
    sha = git('rev-parse', 'HEAD').stdout.strip()
    print(f'Merged the current dev head into the PR branch as {sha}; this run continues on it')
    return sha


def main():
    pr = json.loads(Path(os.environ['GITHUB_EVENT_PATH']).read_text())['pull_request']
    if pr['head']['repo']['full_name'] != os.environ['GITHUB_REPOSITORY']:
        sha = pr['head']['sha']
    else:
        wait_for_turn(pr['number'])
        sha = update_from_dev(pr)
    with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
        output.write(f'sha={sha}\n')


if __name__ == '__main__':
    main()
