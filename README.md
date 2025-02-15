# gRPC JSON Sniffer

gRPC JSON Sniffer is a tool for capturing and visualizing gRPC messages in real-time.
It intercepts gRPC calls using `grpc.UnaryServerInterceptor` and `grpc.UnaryClientInterceptor`, logs the calls in JSON format file, and provides a web-based interface to view and analyze the captured messages.

## Usage

To use the gRPC JSON Sniffer, you need to integrate it into your gRPC server.
Below is an example of how to set it up.

```go
import sniffer "github.com/tsaarni/grpc-json-sniffer"

// Create a new JSON interceptor
func setupGrpcServer() {
    ...
	interceptor, err := sniffer.NewGrpcJsonInterceptor(	)

	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(interceptor.StreamServerInterceptor()),
		grpc.UnaryInterceptor(interceptor.UnaryServerInterceptor()),
	}

	s := grpc.NewServer(opts...)
    ...
}
```

See [`example/server/server.go`](example/server/server.go) for full example.

By default the interceptor does not capture any messages.
Its functionality is enabled by following variables:

- `GRPC_JSON_SNIFFER_FILE` - Path to the JSON file where the intercepted messages will be logged.
For example `/tmp/grpc_capture.json`.
By default, the interceptor does not log any messages. Setting this variable enables the interceptor.
- `GRPC_JSON_SNIFFER_ADDR` - Address for the web server that serves the captured messages.
For example `localhost:8080`.
By default, the web server is not started.

The interceptor can be configured programmatically using options:

```go
interceptor, err := sniffer.NewGrpcJsonInterceptor(
    sniffer.WithFilename("/tmp/grpc_capture.json"),
    sniffer.WithAddr("localhost:8080"),
)
```

## Developing

The repository includes example gRPC server and client implementations located in the [`example`](example) directory.
These can be used to test the gRPC JSON Sniffer.
To start the gRPC server with the JSON interceptor, run:

```bash
go run -tags live_public github.com/tsaarni/grpc-json-sniffer/example/server
```

The `live_public` tag is used to enable the web server that serves static files from the `public` directory, for development purposes.
Otherwise, the files embedded during the build process are used.

Access the captured messages by visiting [http://localhost:8080](http://localhost:8080).

Send a greeting request:

```bash
go run github.com/tsaarni/grpc-json-sniffer/example/client -action greetings -param Joe
```

Send a countdown request:

```bash
go run github.com/tsaarni/grpc-json-sniffer/example/client -action countdown -param 10
```
