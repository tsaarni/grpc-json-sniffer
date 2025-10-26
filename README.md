# gRPC JSON Sniffer for Go

gRPC JSON Sniffer is a Go module designed to capture and visualize gRPC messages in real-time.
It intercepts gRPC calls using `grpc.StreamServerInterceptor` and `grpc.UnaryServerInterceptor` or `grpc.StreamClientInterceptor` and `grpc.UnaryClientInterceptor`, logs the calls to a JSON file, and provides a web-based interface for viewing and analyzing the captured messages.
Captured messages can be filtered using CEL (Common Expression Language) queries.

For more information about interceptors, see [grpc-go documentation](https://github.com/grpc/grpc-go/blob/master/examples/features/interceptor/README.md).


![gRPC JSON Sniffer Web UI](example/webui-screenshot.png)

## Usage

### Integration into a gRPC Server

To integrate the JSON Sniffer into your gRPC server, import the package and create a new JSON interceptor.
Then add the interceptor to your server options:

```go
import grpc_json_sniffer "github.com/tsaarni/grpc-json-sniffer"

func setupGrpcServer() {
    // Create the interceptor. By default, logging is disabled.
    interceptor, err := grpc_json_sniffer.NewGrpcJsonInterceptor()
    if err != nil {
        // Handle error.
    }

    // Add interceptors to the gRPC server options.
    opts := []grpc.ServerOption{
        grpc.StreamInterceptor(interceptor.StreamServerInterceptor()),
        grpc.UnaryInterceptor(interceptor.UnaryServerInterceptor()),

        // Or use github.com/grpc-ecosystem/go-grpc-middleware to chain existing interceptors:
        // grpc.ChainStreamInterceptor(interceptor.StreamServerInterceptor(), <other>)
        // grpc.ChainUnaryInterceptor(interceptor.UnaryServerInterceptor(), <other>)
    }

    // Create new gRPC server with the options.
    s := grpc.NewServer(opts...)
    // ... further setup.
}
```

See [`example/server/server.go`](example/server/server.go) for full example.


### Integration into a gRPC Client

Similar to the server, the JSON Sniffer can be integrated into a gRPC client.

```go
import grpc_json_sniffer "github.com/tsaarni/grpc-json-sniffer"

func setupGrpcClient() {
	interceptor, err := grpc_json_sniffer.NewGrpcJsonInterceptor()
	// ...
	conn, err := grpc.NewClient(grpcServerAddress,
		// ...
		grpc.WithUnaryInterceptor(interceptor.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(interceptor.StreamClientInterceptor()),
	)
	// ...
}
```

See [`example/client/client.go`](example/client/client.go) for full example.

### Configuration

By default the interceptor does not capture any messages.
Its functionality is enabled by following environment variables:

- `GRPC_JSON_SNIFFER_FILE` - Setting this variable enables the interceptor to log messages to a JSON file, for example `/tmp/grpc_capture.json`.
- `GRPC_JSON_SNIFFER_ADDR` - Setting this variable enables the web server to serve the web viewer and captured messages, for example `localhost:8080`.

Alternatively, the interceptor can be configured programmatically using options:

```go
interceptor, err := grpc_json_sniffer.NewGrpcJsonInterceptor(
    grpc_json_sniffer.WithFilename("/tmp/grpc_capture.json"),
    grpc_json_sniffer.WithAddr("localhost:8080"),
)
```

## Standalone Viewer

The JSON Sniffer can be used to view previously captured messages.
To install the viewer, run:

```bash
go install github.com/tsaarni/grpc-json-sniffer/cmd/grpc-json-sniffer-viewer
```

Then start the viewer with the path to the JSON file:

```console
$ grpc-json-sniffer-viewer grpc_server_capture.json
Starting gRPC JSON sniffer viewer on localhost:8080
```

The server bind `localhost:8080` by default.
The address can be changed using the `-addr` flag:

```console
$ grpc-json-sniffer-viewer -addr <address> <filename>
```

## Contributing

Please refer to [CONTRIBUTING.md](CONTRIBUTING.md).

## Credits

* This project uses [cel-js](https://github.com/marcbachmann/cel-js) by Marc Bachmann for CEL (Common Expression Language) parsing and evaluation, licensed under MIT.
