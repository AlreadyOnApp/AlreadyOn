#!/usr/bin/env bash
# Keep a repository-scoped laptop preview alive after an assistant session exits.
# Transient user units: restart the shared preview after reboot.
set -euo pipefail
cd "$(dirname "$0")/.."
repo_root="$PWD"
suffix=$(printf '%s' "$repo_root" | sha256sum | cut -c1-12)
api_unit="alreadyon-preview-$suffix-api"
web_unit="alreadyon-preview-$suffix-web"
# Both assistants use the primary shared preview; contributor overrides remain supported.
web_port=${PREVIEW_PORT:-5173}
api_port=${PREVIEW_API_PORT:-15900}
for port in "$web_port" "$api_port"; do
  [[ $port =~ ^[0-9]+$ ]] && (( port > 0 && port < 65536 )) || { echo 'Invalid preview port' >&2; exit 2; }
done
if ! systemctl --user show-environment >/dev/null 2>&1; then
  echo 'Managed preview requires a Linux user systemd session; use make dev in a persistent terminal when unavailable.' >&2
  exit 1
fi
case "${1:-start}" in
  start)
    # Build before restarting: a failed build leaves the existing preview available.
    # Never mistake another service for this preview or silently reuse old ports.
    for unit in "$api_unit" "$web_unit"; do
      if systemctl --user is-active --quiet "$unit"; then
        settings=$(systemctl --user show "$unit" --property=Environment --value)
        if [[ $unit == "$api_unit" ]]; then
          [[ " $settings " == *" ALREADYON_WEB_PORT=$api_port "* ]] || {
            echo 'Stop the existing preview before changing ports.' >&2; exit 1;
          }
        else
          command_line=$(systemctl --user show "$unit" --property=ExecStart --value)
          [[ " $settings " == *" PREVIEW_API_PORT=$api_port "* && $command_line == *"--port $web_port "* ]] || {
            echo 'Stop the existing preview before changing ports.' >&2; exit 1;
          }
        fi
      fi
    done
    if ! systemctl --user is-active --quiet "$api_unit"; then
      python3 - "$api_port" <<'PYTHON'
import socket, sys
# Probe without taking ownership of an unrelated listener.
for port in sys.argv[1:]:
    with socket.socket() as listener:
        listener.bind(('127.0.0.1', int(port)))
PYTHON
    fi
    if ! systemctl --user is-active --quiet "$web_unit"; then
      python3 - "$web_port" <<'PYTHON'
import socket, sys
with socket.socket() as listener:
    listener.bind(('127.0.0.1', int(sys.argv[1])))
PYTHON
    fi
    make preview-build
    (cd server && PREVIEW_PORT="$web_port" go run ./cmd/preview-account)
    (cd server && go run ./cmd/tls prepare)
    if systemctl --user is-active --quiet "$api_unit"; then
      systemctl --user restart "$api_unit"
    else
      systemd-run --user --collect --unit="$api_unit" \
        --property=Restart=on-failure --property=RestartSec=3 \
        --working-directory="$repo_root/server" \
        --setenv="PATH=$PATH" --setenv=TZ=America/New_York \
        --setenv="ALREADYON_WEB_PORT=$api_port" \
        "$repo_root/server/alreadyon-server"
    fi
    if ! systemctl --user is-active --quiet "$web_unit"; then
      systemd-run --user --collect --unit="$web_unit" \
        --property=Restart=on-failure --property=RestartSec=3 \
        --working-directory="$repo_root/web" \
        --setenv="PATH=$PATH" --setenv="PREVIEW_API_PORT=$api_port" \
        "$(command -v npm)" run dev -- --host 127.0.0.1 --port "$web_port" --strictPort
    fi
    # Reload Vite's certificate after SAN renewal, touching only this preview.
    systemctl --user restart "$web_unit"
    for attempt in {1..60}; do
      if systemctl --user is-active --quiet "$api_unit" "$web_unit" && \
         (cd server && ALREADYON_WEB_PORT="$api_port" go run ./cmd/tls health "$web_port") >/dev/null 2>&1; then
        echo "Preview: https://localhost:$web_port (UI/media $web_port; backend $api_port)"
        echo "Trust CA: $repo_root/server/data/tls/ca.pem (verify fingerprint via: cd server && go run ./cmd/tls fingerprint)"
        echo "Logs: journalctl --user -u $api_unit -u $web_unit -f"
        exit 0
      fi
      sleep 1
    done
    echo 'Preview did not become healthy; inspect make preview-status and its journal.' >&2
    exit 1
    ;;
  status)
    systemctl --user show "$api_unit" "$web_unit" --property=Id,ActiveState,SubState,MainPID
    systemctl --user is-active --quiet "$api_unit" "$web_unit"
    (cd server && ALREADYON_WEB_PORT="$api_port" go run ./cmd/tls health "$web_port")
    ;;
  stop)
    systemctl --user stop "$web_unit" "$api_unit"
    echo 'Preview stopped; server/data is preserved.'
    ;;
  *) echo 'Usage: scripts/preview.sh start|status|stop' >&2; exit 2 ;;
esac
