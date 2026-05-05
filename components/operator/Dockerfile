# syntax=docker/dockerfile:1.7
#
# axisml-operator builds from the repo root so the test/envtest sibling
# module's `replace` paths still resolve in CI builds. The top-level Makefile
# target invokes:
#   docker build -f components/operator/Dockerfile <REPO_ROOT>

# Run the build stage on the host architecture and cross-compile to TARGETARCH.
# Pure Go (CGO_ENABLED=0) cross-compiles cleanly, so this avoids QEMU emulation
# when buildx targets a foreign platform.
FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace

COPY components/operator/go.mod components/operator/go.sum components/operator/
RUN --mount=type=cache,target=/go/pkg/mod \
    cd components/operator && go mod download

COPY components/operator/ components/operator/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd components/operator && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/axisml-operator ./cmd

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/axisml-operator /usr/local/bin/axisml-operator
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/axisml-operator"]
