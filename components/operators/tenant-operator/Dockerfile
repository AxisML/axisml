# syntax=docker/dockerfile:1.7

FROM golang:1.26 AS builder
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY api/ api/
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /tenant-operator ./cmd

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /tenant-operator /usr/local/bin/tenant-operator
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/tenant-operator"]
