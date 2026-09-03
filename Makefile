.PHONY: lint-go lint-js generate update-js update-modules

all: build lint

build:
	go build -o grpc-json-sniffer-viewer cmd/grpc-json-sniffer-viewer/viewer.go
	go build -o server example/server/server.go
	go build -o client example/client/client.go

clean:
	rm -f grpc-json-sniffer-viewer server client

lint: lint-go lint-js

lint-go:
	go tool -modfile=tools/go.mod golangci-lint run

lint-js:
	npm install
	npm run lint

update-js:
	npm install
	npm update
	npm run prepare:cel

# Regenerate the proto files.
generate:
	go tool -modfile=tools/go.mod buf generate

update-modules:
	go get -u -t ./... && go mod tidy
