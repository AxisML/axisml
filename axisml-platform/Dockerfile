# syntax=docker/dockerfile:1.7
#
# Single Platform image: the frontend SPA is built and bundled into the backend
# binary, which serves both the API and the static bundle (same-origin, no
# separate frontend image or gateway route).
#
# Builds from the repo root so the pkg/openapigen `replace` path resolves. The
# Makefile `image` target uses:
#   docker build -f axisml-platform/Dockerfile <REPO_ROOT>

# --- frontend stage: build the SPA bundle ---
FROM node:22-alpine AS frontend
WORKDIR /app
RUN corepack enable
# Install deps first for layer caching (pnpm version is pinned via packageManager).
COPY axisml-platform/frontend/package.json axisml-platform/frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY axisml-platform/frontend/ ./
RUN pnpm run build

# --- backend stage: build the Go binary that serves the SPA ---
FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /workspace

# Copy the shared pkg modules first so the replace targets resolve.
COPY pkg/openapigen/ pkg/openapigen/
COPY pkg/axismlconfig/ pkg/axismlconfig/
COPY pkg/configdoc/ pkg/configdoc/
COPY axisml-platform/backend/ axisml-platform/backend/
# Stage the built SPA into the backend's static dir, replacing the placeholder.
COPY --from=frontend /app/dist/ axisml-platform/backend/internal/web/dist/

WORKDIR /workspace/axisml-platform/backend
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags='-s -w' \
        -o /out/axisml-platform ./cmd/platform-backend

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/axisml-platform /usr/local/bin/axisml-platform
USER 65532:65532
EXPOSE 8080 8081
ENTRYPOINT ["/usr/local/bin/axisml-platform"]
