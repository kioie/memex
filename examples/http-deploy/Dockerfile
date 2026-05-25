# Build from repository root:
#   docker build -f examples/http-deploy/Dockerfile -t memex-http .
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY memex/ memex/
COPY examples/http-deploy/ examples/http-deploy/

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/server \
    ./examples/http-deploy

FROM alpine:3.21

RUN apk add --no-cache ca-certificates && \
    mkdir -p /data && chown nobody:nobody /data

COPY --from=builder /out/server /usr/local/bin/memex-http

ENV MEMEX_DIR=/data
ENV MEMEX_VERBOSE=

USER nobody
EXPOSE 8080
VOLUME /data

CMD ["/usr/local/bin/memex-http"]
