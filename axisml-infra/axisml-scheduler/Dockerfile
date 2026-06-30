# syntax=docker/dockerfile:1.7
#
# axisml-scheduler builds both binaries (scheduler + controller) into one image;
# the Helm Deployments pick the entrypoint per workload. Built from the repo
# root so the build context matches the other layers' Makefile targets:
#   docker build -f axisml-infra/axisml-scheduler/Dockerfile <REPO_ROOT>

FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace

COPY axisml-infra/axisml-scheduler/go.mod axisml-infra/axisml-scheduler/go.sum axisml-infra/axisml-scheduler/
RUN --mount=type=cache,target=/go/pkg/mod \
    cd axisml-infra/axisml-scheduler && go mod download

COPY axisml-infra/axisml-scheduler/ axisml-infra/axisml-scheduler/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd axisml-infra/axisml-scheduler && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/axisml-scheduler ./cmd/scheduler && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/axisml-scheduler-controller ./cmd/controller

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/axisml-scheduler /usr/local/bin/axisml-scheduler
COPY --from=builder /out/axisml-scheduler-controller /usr/local/bin/axisml-scheduler-controller
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/axisml-scheduler"]
