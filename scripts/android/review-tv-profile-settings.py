#!/usr/bin/env python3
"""Check TV profile settings using remote keys and discard drafts without changing live preferences."""
import argparse
from pathlib import Path
import subprocess
import time
import xml.etree.ElementTree as ET


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--serial', required=True)
    parser.add_argument('--sdk', type=Path, required=True)
    args = parser.parse_args()
    adb = [str(args.sdk / 'platform-tools/adb'), '-s', args.serial]
    package = 'io.github.lynchbros.alreadyon.tv.debug'

    def run(*words):
        return subprocess.check_output(adb + list(words), text=True, timeout=30)

    def key(code):
        run('shell', 'input', 'keyevent', str(code))
        time.sleep(0.1) # Match deliberate remote presses without awaiting UI-tree extraction.

    def snapshot():
        path = '/data/local/tmp/alreadyon-review.xml'
        for attempt in range(3):
            run('shell', 'rm', '-f', path)
            output = run('shell', 'uiautomator', 'dump', path)
            if 'dumped to' in output:
                return ET.fromstring(run('exec-out', 'cat', path))
            time.sleep(0.2)
        raise AssertionError('TV did not provide a fresh UI tree: ' + output.strip())

    def texts(tree):
        return {n.get('text') for n in tree.iter('node') if n.get('text')}

    def focused(tree):
        for node in tree.iter('node'):
            if node.get('focused') == 'true':
                return next((n.get('content-desc') for n in node.iter('node') if n.get('content-desc')), '') or ', '.join(n.get('text', '') for n in node.iter('node')).strip(', ')
        return ''

    run('shell', 'am', 'force-stop', package)
    run('shell', 'am', 'start', '-W', '-n', f'{package}/io.github.lynchbros.alreadyon.MainActivity')
    time.sleep(3)
    key(20)
    key(4)
    key(4)
    assert focused(snapshot()) in ('Pause', 'Play')
    key(19)
    assert focused(snapshot()) == 'Switch Profile'
    key(20)
    assert focused(snapshot()) == 'Profile Settings'
    key(23)
    tree = snapshot()
    assert 'CHANNEL' not in texts(tree) and 'Profile Settings' in texts(tree)
    assert focused(tree) == 'Display Name'
    for category, expected in [('Icon', 'Upload Custom Icon'), ('PIN', 'Status'), ('Guide Display', 'Time Format'), ('Audio', 'Preferred Audio Language'), ('Subtitles', 'Subtitles By Default')]:
        key(20)
        key(23)
        tree = snapshot()
        assert any(expected in value for value in texts(tree)), (category, texts(tree))
        assert focused(tree) == category
        print('PASS: settings category', category, flush=True)
    key(22)
    key(23)
    tree = snapshot()
    assert focused(tree).startswith('✓ '), 'Language menu must focus current value'
    original = focused(tree)
    key(20)
    key(23)
    tree = snapshot()
    assert 'Save Changes' in texts(tree)
    assert not any(original.removeprefix('✓ ') + ' ▾' == value for value in texts(tree)), 'Language choice must update the draft'
    print('PASS: language current focus and editable draft', flush=True)
    key(4)
    tree = snapshot()
    assert 'CHANNEL' in texts(tree), 'Back must restore the guide'
    print('PASS: profile settings menu, fullscreen categories, remote focus, Back to guide; no live settings changed')


if __name__ == '__main__':
    main()
