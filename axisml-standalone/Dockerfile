# syntax=docker/dockerfile:1.7
#
# Builds the top-level AxisML Standalone distribution. The build context is the
# AxisML repository root so this module can consume the sibling System modules.

FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace
COPY axisml-system/apis/go.mod axisml-system/apis/go.sum axisml-system/apis/
COPY axisml-system/artifact-hub/go.mod axisml-system/artifact-hub/go.sum axisml-system/artifact-hub/
COPY axisml-system/cluster-manager/go.mod axisml-system/cluster-manager/go.sum axisml-system/cluster-manager/
COPY axisml-system/compute-service/go.mod axisml-system/compute-service/go.sum axisml-system/compute-service/
COPY axisml-standalone/go.mod axisml-standalone/go.sum axisml-standalone/go.work axisml-standalone/
RUN --mount=type=cache,target=/go/pkg/mod \
    cd axisml-standalone && go mod download
COPY axisml-system/ axisml-system/
COPY axisml-standalone/ axisml-standalone/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd axisml-standalone && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/axisml-standalone ./cmd/axisml-standalone

# The standalone process needs root to reach the Docker socket. It is the only
# process in the distribution with socket access.
FROM gcr.io/distroless/static-debian12:latest
COPY --from=builder /out/axisml-standalone /usr/local/bin/axisml-standalone
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/axisml-standalone"]
CMD ["serve"]
