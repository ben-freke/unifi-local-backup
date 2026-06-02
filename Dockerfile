FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN go build -o unifi-backup .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/unifi-backup /usr/local/bin/unifi-backup
ENTRYPOINT ["unifi-backup"]
