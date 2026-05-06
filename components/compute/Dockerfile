# syntax=docker/dockerfile:1.7
#
# Compute imports tenant-operator + compute-operator API packages via local
# replace directives, so the build context must be the repo root. The top-
# level Makefile target uses
#   docker build -f components/compute/Dockerfile <REPO_ROOT>
# Run the build stage on the host architecture and cross-compile to TARGETARCH.
# Pure Go (CGO_ENABLED=0) cross-compiles cleanly, so this avoids QEMU emulation
# when buildx targets a foreign platform.
FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace

# Copy the operator modules first so the replace targets resolve.
COPY components/tenant-operator/ components/tenant-operator/
COPY components/compute-operator/ components/compute-operator/
COPY components/compute/ components/compute/

WORKDIR /workspace/components/compute
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/compute ./cmd/compute

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/compute /usr/local/bin/compute
USER 65532:65532
EXPOSE 8080 8081 8082
ENTRYPOINT ["/usr/local/bin/compute"]
CMD ["serve"]
