# syntax=docker/dockerfile:1.6

FROM golang:1.26.3-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
COPY --from=semstreams . /semstreams
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/semgcs-demo ./cmd/semgcs-demo

FROM alpine:3.22
LABEL version="0.1.0" \
      permissions='{"ExposedPorts":{"80/tcp":{}},"HostConfig":{"ExtraHosts":["host.docker.internal:host-gateway"],"PortBindings":{"80/tcp":[{"HostPort":""}]},"Binds":["/usr/blueos/extensions/semlink:/data"],"Memory":268435456}}' \
      authors='[{"name":"C360 Studio","email":"support@c360studio.com"}]' \
      company='{"about":"SemLink MAVLink companion and mesh node","name":"C360 Studio","email":"support@c360studio.com"}' \
      readme="https://raw.githubusercontent.com/C360Studio/semlink/{tag}/docs/blueos-extension.md" \
      links='{"website":"https://github.com/C360Studio/semlink","documentation":"https://github.com/C360Studio/semlink/blob/main/docs/blueos-extension.md","support":"https://github.com/C360Studio/semlink/issues"}' \
      type="tool" \
      tags='["mavlink","companion","mesh","navigation"]'

RUN apk add --no-cache ca-certificates wget \
    && addgroup -S semlink \
    && adduser -S -G semlink -h /app semlink
WORKDIR /app
COPY --from=builder /out/semgcs-demo /app/semgcs-demo
COPY docker/blueos-extension/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh \
    && mkdir -p /data \
    && chown -R semlink:semlink /app /data
USER semlink
EXPOSE 80
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=6 \
    CMD wget -q -O - http://127.0.0.1/api/health || exit 1
ENTRYPOINT ["/app/entrypoint.sh"]
