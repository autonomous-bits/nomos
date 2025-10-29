.PHONY: help generate lint test clean

help:
	@echo "Available targets:"
	@echo "  generate - Generate Go code from protobuf definitions"
	@echo "  lint     - Run buf linting"
	@echo "  test     - Run tests"
	@echo "  clean    - Remove generated files"

generate:
	@echo "Generating Go code from protobuf..."
	buf generate

lint:
	@echo "Linting protobuf definitions..."
	buf lint

test:
	@echo "Running tests..."
	go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	@echo "Cleaning generated files..."
	rm -rf gen/
	rm -f coverage.out coverage.html
