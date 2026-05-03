CONTROLLER_GEN_VERSION ?= v0.15.0
CONTROLLER_GEN ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION)

.PHONY: all
all: fmt vet build

.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile=hack/boilerplate.go.txt paths=./api/...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test: ## Run unit tests (no cluster required)
	go test ./... -count=1

.PHONY: build
build:
	go build -o bin/manager ./cmd/manager

.PHONY: tidy
tidy:
	go mod tidy
