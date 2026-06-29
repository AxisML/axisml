.PHONY: build run tidy fmt

OUTPUT ?= bin/axisml-core-embed

build:
	go build -o $(OUTPUT) .

run:
	go run .

tidy:
	go mod tidy

fmt:
	gofmt -w .
	go run golang.org/x/tools/cmd/goimports@latest -w . 2>/dev/null || true
