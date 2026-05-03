.PHONY: build test lint

BINARY_NAME := call2action
CMD_PATH    := ./cmd/call2action

build:
	go build -o $(BINARY_NAME) $(CMD_PATH)

test:
	go test ./...

lint:
	golangci-lint run ./...
