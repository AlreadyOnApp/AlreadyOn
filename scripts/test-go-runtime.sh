#!/usr/bin/env bash
# Run source regressions with the production FFmpeg patch and the selected Go toolchain.
set -euo pipefail
cd "$(dirname "$0")/.."
engine=${CONTAINER_ENGINE:-$(command -v podman || command -v docker)}
image=${1:-localhost/alreadyon-base:base-runtime}
go_root=$(go env GOROOT)
module_cache=$(go env GOMODCACHE)
mkdir -p "$module_cache"
# Exclude ignored databases, credentials and media from the test container.
source_dir=$(mktemp -d /tmp/alreadyon-go-source.XXXXXXXX)
trap 'rm -rf "$source_dir"' EXIT
python3 - "$source_dir" <<'COPY'
import pathlib, shutil, subprocess, sys
root = pathlib.Path(sys.argv[1])
for name in subprocess.check_output(['git', 'ls-files', '--cached', '--others', '--exclude-standard', '-z']).decode().split('\0'):
    if name and (pathlib.Path(name).is_file() or pathlib.Path(name).is_symlink()):
        dest = root / name
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(name, dest, follow_symlinks=False)
COPY
run_options=()
if [[ $(basename "$engine") == podman ]]; then run_options+=(--userns=keep-id); fi
"$engine" run --rm "${run_options[@]}" --security-opt label=disable --user "$(id -u):$(id -g)" \
  -e HOME=/tmp -e GOROOT=/opt/go -e GOPATH=/go -e GOTOOLCHAIN=local \
  -v "$source_dir:/workspaces/alreadyon:ro" -v "$go_root:/opt/go:ro" \
  -v "$module_cache:/go/pkg/mod" -w /workspaces/alreadyon/server \
  "$image" /opt/go/bin/go test -p=1 -count=1 ./...
