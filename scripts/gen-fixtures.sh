#!/usr/bin/env bash
# Generate short, entirely synthetic media fixtures for tests (no
# copyrighted content). See docs/PROJECT_BRIEF.md engineering
# expectations ("use synthetic or appropriately licensed test media").
#
# Usage: scripts/gen-fixtures.sh [output-dir]
set -euo pipefail

out="${1:-tests/fixtures/generated}"
mkdir -p "$out"

# A 2-second synthetic H.264/AAC clip: color bars + a tone, valid enough
# for ffprobe-inspection and playback-pipeline tests.
ffmpeg -y -loglevel error \
  -f lavfi -i "testsrc=size=320x240:rate=30:duration=2" \
  -f lavfi -i "sine=frequency=440:duration=2" \
  -c:v libopenh264 -pix_fmt yuv420p -c:a aac \
  "$out/sample_program.mp4"

# A shorter clip to stand in for a commercial break.
ffmpeg -y -loglevel error \
  -f lavfi -i "testsrc=size=320x240:rate=30:duration=1:decimals=2" \
  -f lavfi -i "sine=frequency=880:duration=1" \
  -c:v libopenh264 -pix_fmt yuv420p -c:a aac \
  "$out/sample_commercial.mp4"

echo "Generated fixtures in $out:"
ls -la "$out"
