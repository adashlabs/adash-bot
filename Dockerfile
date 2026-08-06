FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
ENV CGO_ENABLED=1 GOOS=linux
RUN go test ./... && go build -trimpath -ldflags="-s -w" -o /out/adash ./cmd/adash

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates gosu tini \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --create-home adash \
    && mkdir -p /app/data && chown -R adash:adash /app
WORKDIR /app
COPY --from=builder --chown=adash:adash /out/adash /app/adash
COPY docker-entrypoint.sh /usr/local/bin/adash-entrypoint
RUN chmod 755 /usr/local/bin/adash-entrypoint
ENV ADASH_DB_PATH=/app/data/adash.db GOMEMLIMIT=64MiB GOGC=50
ENTRYPOINT ["/usr/bin/tini", "-s", "--", "/usr/local/bin/adash-entrypoint"]
CMD ["/app/adash"]
