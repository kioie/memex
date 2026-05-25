# Stdio MCP image for registry (server.json) and Smithery MCPB.
# Build from repository root:
#   docker build -t ghcr.io/kioie/memex:0.6.0 .
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY memex/ memex/
COPY cmd/memex/ cmd/memex/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/memex \
    ./cmd/memex

FROM alpine:3.21

RUN apk add --no-cache ca-certificates && \
    mkdir -p /data && chown nobody:nobody /data

COPY --from=builder /out/memex /usr/local/bin/memex

ENV MEMEX_DIR=/data

USER nobody
VOLUME /data

ENTRYPOINT ["memex"]
CMD ["serve"]
