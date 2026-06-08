# syntax=docker/dockerfile:1.7
#
# axisml-cluster-manager builds from the repo root so the tenant-operator
# `replace` path resolves. The top-level Makefile target uses
#   docker build -f components/cluster-manager/Dockerfile <REPO_ROOT>

FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace

# Copy the tenant-operator and shared pkg modules first so the replace targets resolve.
COPY pkg/openapigen/ pkg/openapigen/
COPY components/tenant-operator/ components/tenant-operator/
COPY components/cluster-manager/ components/cluster-manager/

WORKDIR /workspace/components/cluster-manager
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/axisml-cluster-manager ./cmd/cluster-manager

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/axisml-cluster-manager /usr/local/bin/axisml-cluster-manager
USER 65532:65532
EXPOSE 8080 8081 8082
ENTRYPOINT ["/usr/local/bin/axisml-cluster-manager"]
