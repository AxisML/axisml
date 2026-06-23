# syntax=docker/dockerfile:1.7
#
# axisml-platform-backend builds from the repo root so the pkg/openapigen
# `replace` path resolves. The top-level Makefile target uses
#   docker build -f axisml-platform/backend/Dockerfile <REPO_ROOT>

FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace

# Copy the shared pkg module first so the replace target resolves.
COPY pkg/openapigen/ pkg/openapigen/
COPY axisml-platform/backend/ axisml-platform/backend/

WORKDIR /workspace/axisml-platform/backend
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/axisml-platform-backend ./cmd/platform-backend

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/axisml-platform-backend /usr/local/bin/axisml-platform-backend
USER 65532:65532
EXPOSE 8080 8082
ENTRYPOINT ["/usr/local/bin/axisml-platform-backend"]
