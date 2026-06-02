#!/bin/sh
set -e

PUID=${PUID:-$(id -u backup)}
PGID=${PGID:-$(id -g backup)}

groupmod -o -g "$PGID" backup
usermod -o -u "$PUID" backup
chown -R backup:backup /backups

exec su-exec backup /usr/local/bin/unifi-backup
