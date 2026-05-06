# syntax=docker/dockerfile:1.7
#
# axisml-artifacts builds from the repo root for consistency with the other
# AxisML components. The top-level Makefile target invokes:
#   docker build -f components/artifacts/Dockerfile <REPO_ROOT>
#
# Run the build stage on the host architecture and cross-compile to TARGETARCH.
# Pure Go (CGO_ENABLED=0) cross-compiles cleanly, so this avoids QEMU emulation
# when buildx targets a foreign platform.
FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace

COPY components/artifacts/go.mod components/artifacts/go.sum components/artifacts/
RUN --mount=type=cache,target=/go/pkg/mod \
    cd components/artifacts && go mod download

COPY components/artifacts/ components/artifacts/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd components/artifacts && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/artifacts ./cmd/artifacts

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/artifacts /usr/local/bin/artifacts
USER 65532:65532
EXPOSE 8080 8082 8083
ENTRYPOINT ["/usr/local/bin/artifacts"]
CMD ["serve"]
