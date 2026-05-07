# syntax=docker/dockerfile:1.7
#
# axisml-compute-operator builds from the repo root.
#   docker build -f components/compute-operator/Dockerfile <REPO_ROOT>

FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace

COPY components/compute-operator/go.mod components/compute-operator/go.sum components/compute-operator/
RUN --mount=type=cache,target=/go/pkg/mod \
    cd components/compute-operator && go mod download

COPY components/compute-operator/ components/compute-operator/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd components/compute-operator && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/axisml-compute-operator ./cmd

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/axisml-compute-operator /usr/local/bin/axisml-compute-operator
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/axisml-compute-operator"]
