# unifi-local-backup

A lightweight Go container that logs into a UniFi Network Application and downloads a backup file to a local directory.

Supports one-shot and scheduled (cron) modes, multiple controllers, and Kubernetes-native deployment.

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
  -e UNIFI_USERNAMES=admin,admin \
  -e UNIFI_PASSWORDS=secret1,secret2 \
  -e BACKUP_DIR=/backups \
  -e BACKUP_INTERVAL=6h \
  -p 8080:8080 \
  -v /path/to/local/backups:/backups \
  ghcr.io/ben-freke/unifi-local-backup
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `UNIFI_HOST` | Base URL of a single UniFi controller (e.g. `https://nas.example.com`) |
| `UNIFI_USERNAME` | Username for single-controller mode |
| `UNIFI_PASSWORD` | Password for single-controller mode |
| `UNIFI_HOSTS` | Comma-separated list of controller URLs for multi-controller mode |
| `UNIFI_USERNAMES` | Comma-separated usernames (must match length of `UNIFI_HOSTS`) |
| `UNIFI_PASSWORDS` | Comma-separated passwords (must match length of `UNIFI_HOSTS`) |
| `BACKUP_DIR` | Directory inside the container to write backup files |
| `BACKUP_INTERVAL` | Go duration string (e.g. `6h`, `30m`). If set, runs on a repeating schedule instead of one-shot |
| `HEALTH_PORT` | Port for the `/healthz` endpoint when running in scheduled mode (default: `8080`) |
| `PUID` | User ID to run as — Docker/Compose only (default: internal `backup` user) |
| `PGID` | Group ID to run as — Docker/Compose only (default: internal `backup` group) |

Backups are saved as `unifi-backup-<host>-<timestamp>.unf`.

## Health check

When running in scheduled mode (`BACKUP_INTERVAL` set), a `/healthz` endpoint is served on `HEALTH_PORT`:

- `503` — no backup run has completed yet (starting up)
- `200 {"status":"ok"}` — last run completed without errors
- `200 {"status":"degraded", "errors":[...]}` — last run completed with one or more failures

### Kubernetes example

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30
readinessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 10
```

## Building

```bash
docker build -t unifi-local-backup .
```
