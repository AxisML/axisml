# syntax=docker/dockerfile:1.7
#
# Builds the single axisml-core image (Cluster Manager + Compute Service +
# Artifact Hub modules + the in-process Standalone runtime). The build
# context is the repo ROOT so the go.mod replace directives (../../axisml-system/*,
# ../../pkg/axismlconfig) resolve. Build from the repo root:
#   docker build -f axisml-lite/axisml-core/Dockerfile -t axisml-core:dev .
#
# The build is cgo-enabled: the Standalone runtime uses github.com/NVIDIA/go-nvml
# (AXISML_GPU_DEVICES=all) which links against cgo. libnvidia-ml is NOT needed at
# build time (go-nvml dlopen's it at runtime); only a C toolchain is required.

FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG BUILDARCH
WORKDIR /workspace

# cgo needs a C toolchain matching the target arch. Native builds use the image's
# gcc; a cross build installs the matching cross-gcc and points CC at it.
RUN set -eux; \
    if [ "$TARGETARCH" != "$BUILDARCH" ]; then \
      apt-get update; \
      case "$TARGETARCH" in \
        arm64) apt-get install -y --no-install-recommends gcc-aarch64-linux-gnu; echo aarch64-linux-gnu-gcc > /cc ;; \
        amd64) apt-get install -y --no-install-recommends gcc-x86-64-linux-gnu; echo x86_64-linux-gnu-gcc > /cc ;; \
        *) echo "unsupported cross TARGETARCH=$TARGETARCH" >&2; exit 1 ;; \
      esac; \
      rm -rf /var/lib/apt/lists/*; \
    else echo gcc > /cc; fi

# Copy the modules the axisml-core replace directives point at, plus axisml-lite.
COPY pkg/axismlconfig/ pkg/axismlconfig/
COPY axisml-system/ axisml-system/
COPY axisml-lite/ axisml-lite/

WORKDIR /workspace/axisml-lite/axisml-core
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 CC="$(cat /cc)" GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/axisml-core ./cmd/axisml-core

# axisml-core needs root to reach the Docker socket (design §7.2: it is the only
# process with socket access). base-debian12 (root) ships glibc for the
# cgo-linked binary plus CA certs for OCI/S3.
FROM gcr.io/distroless/base-debian12:latest
COPY --from=builder /out/axisml-core /usr/local/bin/axisml-core
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/axisml-core"]
CMD ["serve"]
