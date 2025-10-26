# Contributing

This guide is for those who wish to contribute to the project.

## Developing

To quickly test that nothing major broke, build and lint the project:

```bash
make
make lint
```

## Testing with Example Server and Client

The repository includes example gRPC server and client implementations located in the [`example`](example) directory.
These can be used to test the gRPC JSON Sniffer.
To start the gRPC server with the JSON interceptor, run:

```bash
go run -tags live_public github.com/tsaarni/grpc-json-sniffer/example/server
```

The messages are written to the `grpc_server_capture.json` file in the current directory.
Access the captured messages by visiting [http://localhost:8080](http://localhost:8080).

The `live_public` tag is used to enable the web server that serves static files from the `public` directory, for development purposes.
Otherwise, the files embedded during the build process are used.


Send unary greeting request:

```bash
go run github.com/tsaarni/grpc-json-sniffer/example/client greetings Joe
```

Send streaming countdown request:

```bash
go run -tags live_public github.com/tsaarni/grpc-json-sniffer/example/client countdown 6000
```

The client writes messages to the `grpc_client_capture.json` file in the current directory.
While the client is running the client message viewer can be accessed at [http://localhost:8081](http://localhost:8081).
To view messages after the client has finished, use the standalone viewer:

```bash
go run github.com/tsaarni/grpc-json-sniffer/cmd/grpc-json-sniffer-viewer -addr localhost:8081 grpc_client_capture.json
```

To run the offline viewer

```bash
go run -tags live_public github.com/tsaarni/grpc-json-sniffer/cmd/grpc-json-sniffer-viewer grpc_client_capture.json
```
