# syntax=docker/dockerfile:1.7
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags=-static" \
    -trimpath \
    -o /billing-api ./cmd/server

# ---- distroless runtime ----
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /billing-api /billing-api

EXPOSE 8081

ENTRYPOINT ["/billing-api"]
