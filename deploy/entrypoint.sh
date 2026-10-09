#!/bin/sh
# Container entrypoint: makes /config writable for PUID:PGID, then runs the
# server as that user. PUID and PGID default to 1000 and may be any number,
# including 0 (run as root).
set -eu

PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

case "$PUID" in '' | *[!0-9]*)
	echo "PUID must be a number, got '$PUID'" >&2
	exit 1
	;;
esac
case "$PGID" in '' | *[!0-9]*)
	echo "PGID must be a number, got '$PGID'" >&2
	exit 1
	;;
esac

SERVER=/usr/local/bin/alreadyon-server

# Started as a non-root user already (for example `user:` in compose or
# rootless podman): nothing to change or drop.
if [ "$(id -u)" != "0" ]; then
	exec "$SERVER"
fi

mkdir -p /config
# Only re-own /config when its owner differs, so restarts stay fast.
if [ "$(stat -c '%u:%g' /config)" != "$PUID:$PGID" ]; then
	chown -R "$PUID:$PGID" /config
fi

if [ "$PUID" = "0" ]; then
	exec "$SERVER"
fi

export HOME=/config

# Hardware transcoding needs the GPU's render node, which is normally
# root:render with mode 0660. Give the server the group of whichever /dev/dri
# render node passed in, so PUID/PGID can stay non-root without a group_add. Any
# other supplementary group (root's, for one) is still dropped.
DEVICE_GROUPS=""
for node in /dev/dri/renderD* /dev/nvidia* /dev/nvidia-caps/*; do
        [ -c "$node" ] || continue
        gid="$(stat -c '%g' "$node")"
        [ "$gid" = "0" ] && continue
        case ",$DEVICE_GROUPS," in
        *",$gid,"*) ;;
        *) DEVICE_GROUPS="${DEVICE_GROUPS:+$DEVICE_GROUPS,}$gid" ;;
        esac
done
if [ -n "$DEVICE_GROUPS" ]; then
        exec setpriv --reuid="$PUID" --regid="$PGID" --groups="$DEVICE_GROUPS" "$SERVER"
fi
exec setpriv --reuid="$PUID" --regid="$PGID" --clear-groups "$SERVER"
