# syntax=docker/dockerfile:1.7
#
# axisml-artifact-hub builds from the repo root for consistency with the other
# AxisML components. The top-level Makefile target invokes:
#   docker build -f axisml-system/artifact-hub/Dockerfile <REPO_ROOT>
#
# Run the build stage on the host architecture and cross-compile to TARGETARCH.
# Pure Go (CGO_ENABLED=0) cross-compiles cleanly, so this avoids QEMU emulation
# when buildx targets a foreign platform.
FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace

# Copy the shared pkg modules first so the replace targets resolve.
COPY pkg/openapigen/ pkg/openapigen/
COPY pkg/axismlconfig/ pkg/axismlconfig/
COPY pkg/configdoc/ pkg/configdoc/
COPY axisml-system/artifact-hub/go.mod axisml-system/artifact-hub/go.sum axisml-system/artifact-hub/
RUN --mount=type=cache,target=/go/pkg/mod \
    cd axisml-system/artifact-hub && go mod download

COPY axisml-system/artifact-hub/ axisml-system/artifact-hub/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd axisml-system/artifact-hub && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/artifact-hub ./cmd/artifact-hub

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/artifact-hub /usr/local/bin/artifact-hub
USER 65532:65532
EXPOSE 8080 8082 8083
ENTRYPOINT ["/usr/local/bin/artifact-hub"]
CMD ["serve"]
