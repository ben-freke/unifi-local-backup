FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN go build -o unifi-backup .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates shadow su-exec && \
    addgroup -S backup && adduser -S backup -G backup && \
    mkdir -p /backups && chown backup:backup /backups
COPY --from=builder /app/unifi-backup /usr/local/bin/unifi-backup
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
