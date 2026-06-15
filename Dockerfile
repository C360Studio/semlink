# syntax=docker/dockerfile:1.6

FROM node:24-alpine AS ui
WORKDIR /src/ui
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ ./
RUN npm run build

FROM golang:1.26.3-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
COPY --from=semstreams . /semstreams
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=ui /src/ui/dist ./ui/dist
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/semgcs-demo ./cmd/semgcs-demo

FROM alpine:3.22
RUN addgroup -S semlink && adduser -S -G semlink -h /app semlink
WORKDIR /app
COPY --from=builder /out/semgcs-demo /app/semgcs-demo
COPY --from=ui /src/ui/dist /app/ui/dist
USER semlink
EXPOSE 8080
ENTRYPOINT ["/app/semgcs-demo"]
CMD ["-embedded-nats=false", "-nats-url=nats://semlink-nats:4222", "-static=/app/ui/dist", "-listen=:8080"]
