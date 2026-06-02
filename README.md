# unifi-local-backup

A lightweight Go container that logs into a UniFi Network Application and downloads a backup file to a local directory.

## Usage

```bash
docker run --rm \
  -e UNIFI_HOST=https://your-controller \
  -e UNIFI_USERNAME=admin \
  -e UNIFI_PASSWORD=secret \
  -e BACKUP_DIR=/backups \
  -v /path/to/local/backups:/backups \
  ghcr.io/ben-freke/unifi-local-backup
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `UNIFI_HOST` | Base URL of the UniFi controller (e.g. `https://nas.example.com`) |
| `UNIFI_USERNAME` | UniFi username |
| `UNIFI_PASSWORD` | UniFi password |
| `BACKUP_DIR` | Directory inside the container to write the backup file |

Backups are saved as `unifi-backup-<timestamp>.unf`.

## Building

```bash
docker build -t unifi-local-backup .
```
