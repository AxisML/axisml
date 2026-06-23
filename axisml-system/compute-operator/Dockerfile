# syntax=docker/dockerfile:1.7
#
# axisml-compute-operator builds from the repo root.
#   docker build -f axisml-system/compute-operator/Dockerfile <REPO_ROOT>

FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace

COPY axisml-system/compute-operator/go.mod axisml-system/compute-operator/go.sum axisml-system/compute-operator/
RUN --mount=type=cache,target=/go/pkg/mod \
    cd axisml-system/compute-operator && go mod download

COPY axisml-system/compute-operator/ axisml-system/compute-operator/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd axisml-system/compute-operator && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/axisml-compute-operator ./cmd

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/axisml-compute-operator /usr/local/bin/axisml-compute-operator
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/axisml-compute-operator"]
