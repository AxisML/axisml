# syntax=docker/dockerfile:1.6

FROM golang:1 AS builder
WORKDIR /workspace

COPY components/operators/tenant-operator/go.mod components/operators/tenant-operator/go.sum components/operators/tenant-operator/
COPY components/operators/mljob-operator/go.mod components/operators/mljob-operator/go.sum components/operators/mljob-operator/
COPY components/operators/mlservice-operator/go.mod components/operators/mlservice-operator/go.sum components/operators/mlservice-operator/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd components/operators/tenant-operator && go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd components/operators/mljob-operator && go mod download
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd components/operators/mlservice-operator && go mod download

COPY components/operators/tenant-operator/ components/operators/tenant-operator/
COPY components/operators/mljob-operator/ components/operators/mljob-operator/
COPY components/operators/mlservice-operator/ components/operators/mlservice-operator/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd components/operators/tenant-operator && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/tenant-operator ./cmd
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd components/operators/mljob-operator && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mljob-operator ./cmd/manager
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd components/operators/mlservice-operator && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mlservice-operator ./cmd/manager

FROM alpine:3.20
WORKDIR /
COPY --from=builder /out/tenant-operator /usr/local/bin/tenant-operator
COPY --from=builder /out/mljob-operator /usr/local/bin/mljob-operator
COPY --from=builder /out/mlservice-operator /usr/local/bin/mlservice-operator
COPY components/operators/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod 0555 /usr/local/bin/entrypoint.sh
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
