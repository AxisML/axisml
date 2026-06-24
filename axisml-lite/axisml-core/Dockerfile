# syntax=docker/dockerfile:1.7
#
# Builds the single axisml-core image (Cluster Manager + Compute Service +
# Artifact Hub modules + the in-process Standalone Docker runtime). The build
# context is the repo ROOT so the go.mod replace directives (../../axisml-system/*,
# ../../pkg/openapigen) resolve. Build from the repo root:
#   docker build -f axisml-lite/axisml-core/Dockerfile -t axisml-core:dev .

FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace

# Copy the modules the axisml-lite replace directives point at, plus axisml-lite.
COPY pkg/openapigen/ pkg/openapigen/
COPY axisml-system/ axisml-system/
COPY axisml-lite/ axisml-lite/

WORKDIR /workspace/axisml-lite/axisml-core
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/axisml-core ./cmd/axisml-core

# axisml-core needs root to reach the Docker socket (design §7.2: it is the only
# process with socket access). static-debian12 (root) ships CA certs for OCI/S3.
FROM gcr.io/distroless/static-debian12:latest
COPY --from=builder /out/axisml-core /usr/local/bin/axisml-core
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/axisml-core"]
CMD ["serve"]
