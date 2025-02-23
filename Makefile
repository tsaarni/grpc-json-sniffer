
.PHONY: lint-go generate tools

build:
	go build -o grpc-json-sniffer-viewer cmd/grpc-json-sniffer-viewer/viewer.go
	go build -o server example/server/server.go
	go build -o client example/client/client.go

clean:
	rm -f grpc-json-sniffer-viewer server client

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2 run

# Regenerate the proto files.
generate:
	protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative example/demo/demo.proto

# Install tools required for protobuf code generation.
tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.5
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
