#!/usr/bin/env bash
# Report installed tool versions and flag missing prerequisites for
# AlreadyOn local development. See docs/PROJECT_BRIEF.md section 10.
set -u

ok=0
missing=0

check() {
  local name="$1"
  local cmd="$2"
  if command -v "$cmd" >/dev/null 2>&1; then
    local version
    version="$("$cmd" "${3:---version}" 2>&1 | head -n1)"
    printf '[ok]      %-10s %s\n' "$name" "$version"
    ok=$((ok + 1))
  else
    printf '[missing] %-10s not found on PATH\n' "$name"
    missing=$((missing + 1))
  fi
}

check "go" go version
check "node" node
check "npm" npm
check "ffmpeg" ffmpeg
check "ffprobe" ffprobe
check "git" git
check "gh" gh

check "make" make
check "python3" python3
check "sqlite3" sqlite3 -version
check "pgrep" pgrep

if command -v python3 >/dev/null; then
  python3 "$(dirname "$0")/check-toolchain.py" --installed || missing=$((missing + 1))
fi
if command -v ffmpeg >/dev/null; then
  for encoder in libopenh264 aac; do
    if ! ffmpeg -hide_banner -encoders 2>/dev/null | grep -qw "$encoder"; then
      echo "[missing] FFmpeg encoder: $encoder"; missing=$((missing + 1))
    fi
  done
fi
if [[ -d "$(dirname "$0")/../web/node_modules/playwright" ]]; then
  node -e 'const {chromium}=require(process.argv[1]); chromium.launch({headless:true}).then(b=>b.close()).catch(e=>{console.error(e.message);process.exit(1)})' "$(cd "$(dirname "$0")/../web" && pwd)/node_modules/playwright" || missing=$((missing + 1))
else
  echo "[missing] Playwright: run npm ci and npx playwright install chromium in web/"
  missing=$((missing + 1))
fi

echo
echo "$ok found, $missing missing."
if [ "$missing" -gt 0 ]; then
  echo "Install the missing tools above before running 'make dev'/'make build'."
  exit 1
fi
