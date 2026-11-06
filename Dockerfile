# syntax=docker/dockerfile:1

# The builder runs on the build machine's platform and cross-compiles for
# the target platform, so multi-platform builds need no emulation.
FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/server ./cmd/server

FROM alpine:3.24

WORKDIR /app
COPY --from=builder /out/server /app/server

# A numeric unprivileged user, so runtimes can verify it isn't root (such as
# Kubernetes runAsNonRoot, which rejects a user given by name).
USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/app/server"]
