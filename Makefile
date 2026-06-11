.PHONY: test lint build run cov

test:
	go test ./... -race -coverprofile=coverage.out

lint:
	golangci-lint run

build:
	go build -o waveshell ./cmd/waveshell

run: build
	@echo "waveshell built. Run ./waveshell to start."

cov:
	go tool cover -html=coverage.out
