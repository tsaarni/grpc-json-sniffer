
.PHONY: lint-go generate tools

lint-go:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2 run

# Regenerate the proto files.
generate:
	protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative example/demo/demo.proto

# Install tools required for protobuf code generation.
tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.5
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
