FROM alpine:3.21 AS builder

RUN apk add --no-cache build-base git go

WORKDIR /build

# Build ByeDPI from pinned release v0.17.3
RUN git clone --depth 1 --branch v0.17.3 https://github.com/hufrea/byedpi.git byedpi \
    && cd byedpi \
    && make \
    && cp ciadpi /build/ciadpi

# Build dpi-bridge
COPY bridge/ /build/bridge/
RUN cd /build/bridge \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /build/dpi-bridge .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tini wget \
    && addgroup -S -g 10001 dpi \
    && adduser -S -u 10001 -G dpi dpi

COPY --from=builder /build/ciadpi /usr/local/bin/ciadpi
COPY --from=builder /build/dpi-bridge /usr/local/bin/dpi-bridge

RUN chmod 755 /usr/local/bin/ciadpi /usr/local/bin/dpi-bridge

USER dpi:dpi

EXPOSE 8080 1080

ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/dpi-bridge"]
