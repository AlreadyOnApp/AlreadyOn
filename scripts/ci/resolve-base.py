#!/usr/bin/env python3
"""Resolve moving base tags by recipe label, pinning every consumer by digest."""
import argparse
import importlib.util
import json
import os
import pathlib
import subprocess


def matches(labels, key, runtime):
    """CI must inherit the selected runtime, including weekly security refreshes."""
    return labels.get('io.alreadyon.base.recipe') == key and (
        not runtime or labels.get('io.alreadyon.base.runtime') == runtime)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('image')
    parser.add_argument('kind', choices=['ci', 'runtime'])
    parser.add_argument('--runtime', default='')
    parser.add_argument('--refresh', action='store_true')
    args = parser.parse_args()
    root = pathlib.Path(__file__).resolve().parents[2]
    spec = importlib.util.spec_from_file_location('base_key', root / 'scripts/ci/base-image-key.py')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    key = module.recipe_key(root, args.kind)
    tag = f'{args.image}:base-{args.kind}'
    result = subprocess.run(['docker', 'buildx', 'imagetools', 'inspect', tag,
                             '--format', '{"Manifest":{{json .Manifest}},"Image":{{json .Image}}}'],
                            text=True, capture_output=True)
    digest = ''
    valid = False
    if result.returncode == 0:
        data = json.loads(result.stdout)
        digest = data['Manifest']['digest']
        labels = data['Image']['config'].get('Labels') or {}
        valid = matches(labels, key, args.runtime)
    elif not any(reason in result.stderr.lower() for reason in
                 ('manifest unknown', 'not found', 'name unknown')):
        raise RuntimeError(f'Cannot safely resolve {tag}: {result.stderr}')
    with open(os.environ['GITHUB_OUTPUT'], 'a', encoding='utf-8') as output:
        output.write(f'key={key}\nneeded={str(args.refresh or not valid).lower()}\n')
        output.write(f'reference={tag}@{digest}\n')


if __name__ == '__main__':
    main()
