.PHONY: build test clean run install release release-snapshot release-test

# Build the application
build:
	go build -o flink-analyzer

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o flink-analyzer-linux-amd64
	GOOS=linux GOARCH=arm64 go build -o flink-analyzer-linux-arm64
	GOOS=darwin GOARCH=amd64 go build -o flink-analyzer-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build -o flink-analyzer-darwin-arm64
	GOOS=windows GOARCH=amd64 go build -o flink-analyzer-windows-amd64.exe

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f flink-analyzer
	rm -f flink-analyzer-*
	rm -rf dist/
	go clean

# Run on examples
run: build
	./flink-analyzer examples

# Install dependencies
install:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Run with custom directory
analyze:
	@if [ -z "$(DIR)" ]; then \
		echo "Usage: make analyze DIR=/path/to/yamls"; \
		exit 1; \
	fi
	./flink-analyzer $(DIR)

# GoReleaser commands
# Build release binaries using GoReleaser
release:
	goreleaser release --clean

# Build snapshot (without releasing)
release-snapshot:
	goreleaser release --snapshot --clean

# Test the release process without creating artifacts
release-test:
	goreleaser check
	goreleaser build --snapshot --clean
