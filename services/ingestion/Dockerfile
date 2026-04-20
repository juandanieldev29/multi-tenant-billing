# syntax=docker/dockerfile:1.7
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Cache module downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags=-static" \
    -trimpath \
    -o /ingestion ./cmd/server

# ---- distroless runtime — no shell, no package manager ----
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /ingestion /ingestion

EXPOSE 8080

ENTRYPOINT ["/ingestion"]
