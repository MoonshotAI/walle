gen:
	@find . -type f -name "mock_*.go" -delete
	@go generate ./...

lint:
	@golangci-lint run

test:
	@go test -v ./...

build:
	@go build -o bin/walle ./cmd/walle

clean:
	@rm -rf bin/

.PHONY: gen lint test build clean

