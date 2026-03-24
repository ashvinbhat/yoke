.PHONY: build install test clean run

# Build the binary
build:
	go build -o yoke ./cmd/yoke

# Install globally
install:
	go install ./cmd/yoke

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f yoke
	rm -rf dist/

# Run directly
run:
	go run ./cmd/yoke $(ARGS)

# Format code
fmt:
	go fmt ./...

# Lint
lint:
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy
