# syntax=docker/dockerfile:1.7
#
# axisml-tenant-operator builds from the repo root so any sibling test
# module's `replace` paths resolve in CI builds. The top-level Makefile
# target invokes:
#   docker build -f axisml-system/tenant-operator/Dockerfile <REPO_ROOT>

FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace

COPY axisml-system/tenant-operator/go.mod axisml-system/tenant-operator/go.sum axisml-system/tenant-operator/
RUN --mount=type=cache,target=/go/pkg/mod \
    cd axisml-system/tenant-operator && go mod download

COPY axisml-system/tenant-operator/ axisml-system/tenant-operator/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd axisml-system/tenant-operator && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/axisml-tenant-operator ./cmd

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/axisml-tenant-operator /usr/local/bin/axisml-tenant-operator
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/axisml-tenant-operator"]
