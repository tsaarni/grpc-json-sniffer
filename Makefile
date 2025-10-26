
.PHONY: lint-go generate tools

all: build lint

build:
	go build -o grpc-json-sniffer-viewer cmd/grpc-json-sniffer-viewer/viewer.go
	go build -o server example/server/server.go
	go build -o client example/client/client.go

clean:
	rm -f grpc-json-sniffer-viewer server client

lint: lint-go lint-js

lint-go:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0 run

lint-js:
	npm install
	npm run lint

update-js:
	npm install
	npm update
	npm run prepare:cel

# Regenerate the proto files.
generate:
	protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative example/demo/demo.proto

# Install tools required for protobuf code generation.
tools:
	# https://github.com/protocolbuffers/protobuf-go
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.7
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

update-modules:
	go get -u -t ./... && go mod tidy
