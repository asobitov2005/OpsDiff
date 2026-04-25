APP := opsdiff
BUILD_DIR := bin

.PHONY: build test fmt

build:
	mkdir -p $(BUILD_DIR)
	go build -ldflags "-X github.com/opsdiff/opsdiff/internal/app.Version=dev" -o $(BUILD_DIR)/$(APP) ./cmd/opsdiff

test:
	go test ./...

fmt:
	go fmt ./...

