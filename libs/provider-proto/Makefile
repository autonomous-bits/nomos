.PHONY: help generate generate-protoc lint test clean

help:
	@echo "Available targets:"
	@echo "  generate        - Generate Go code from protobuf definitions using buf"
	@echo "  generate-protoc - Generate Go code using protoc directly (fallback)"
	@echo "  lint            - Run buf linting"
	@echo "  test            - Run tests"
	@echo "  clean           - Remove generated files"

generate:
	@echo "Generating Go code from protobuf..."
	buf generate

generate-protoc:
	@echo "Generating Go code from protobuf using protoc..."
	@mkdir -p gen/go/nomos/provider/v1
	protoc \
		--proto_path=proto \
		--go_out=gen/go \
		--go_opt=paths=source_relative \
		--go-grpc_out=gen/go \
		--go-grpc_opt=paths=source_relative \
		proto/nomos/provider/v1/provider.proto

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
