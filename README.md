# unifi-local-backup

A lightweight Go container that logs into a UniFi Network Application and downloads a backup file to a local directory.

Supports one-shot and scheduled (cron) modes, and multiple controllers sharing the same credentials.

## Usage

### One-shot (single controller)

```bash
docker run --rm \
  -e UNIFI_HOST=https://your-controller \
  -e UNIFI_USERNAME=admin \
  -e UNIFI_PASSWORD=secret \
  -e BACKUP_DIR=/backups \
  -v /path/to/local/backups:/backups \
  ghcr.io/ben-freke/unifi-local-backup
```

### Scheduled (multiple controllers)

```bash
docker run \
  -e UNIFI_HOSTS=https://controller1,https://controller2 \
  -e UNIFI_USERNAME=admin \
  -e UNIFI_PASSWORD=secret \
  -e BACKUP_DIR=/backups \
  -e BACKUP_INTERVAL=6h \
  -v /path/to/local/backups:/backups \
  ghcr.io/ben-freke/unifi-local-backup
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `UNIFI_HOST` | Base URL of a single UniFi controller (e.g. `https://nas.example.com`) |
| `UNIFI_HOSTS` | Comma-separated list of controller URLs (alternative to `UNIFI_HOST`) |
| `UNIFI_USERNAME` | UniFi username (shared across all controllers) |
| `UNIFI_PASSWORD` | UniFi password (shared across all controllers) |
| `BACKUP_DIR` | Directory inside the container to write backup files |
| `BACKUP_INTERVAL` | Go duration string (e.g. `6h`, `30m`). If set, runs on a repeating schedule instead of one-shot |
| `PUID` | User ID to run as — Docker/Compose only (default: internal `backup` user) |
| `PGID` | Group ID to run as — Docker/Compose only (default: internal `backup` group) |

Backups are saved as `unifi-backup-<host>-<timestamp>.unf`.

## Building

```bash
docker build -t unifi-local-backup .
```
