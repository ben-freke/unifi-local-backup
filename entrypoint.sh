#!/bin/sh
set -e

# If already running as non-root (e.g. Kubernetes securityContext), skip user remapping
if [ "$(id -u)" != "0" ]; then
  exec /usr/local/bin/unifi-backup
fi

PUID=${PUID:-$(id -u backup)}
PGID=${PGID:-$(id -g backup)}

groupmod -o -g "$PGID" backup
usermod -o -u "$PUID" backup
chown -R backup:backup /backups

exec su-exec backup /usr/local/bin/unifi-backup
