# syntax=docker/dockerfile:1.7
#
# Compute imports operator API packages via local replace directives, so the
# build context must be the repo root. The top-level Makefile target uses
#   docker build -f components/compute/Dockerfile <REPO_ROOT>
FROM golang:1.26 AS builder
WORKDIR /workspace

# Copy the merged operator module first so the replace target resolves.
COPY components/operator/ components/operator/
COPY components/compute/ components/compute/

WORKDIR /workspace/components/compute
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' \
        -o /out/compute ./cmd/compute

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/compute /usr/local/bin/compute
USER 65532:65532
EXPOSE 8080 8081 8082
ENTRYPOINT ["/usr/local/bin/compute"]
CMD ["serve"]
