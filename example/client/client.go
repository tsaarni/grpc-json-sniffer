package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	sniffer "github.com/tsaarni/grpc-json-sniffer"
	"github.com/tsaarni/grpc-json-sniffer/example/demo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	grpcServerAddress = "localhost:50051"
	httpViewerAddress = "localhost:8080"
)

func main() {
	if len(os.Args) < 3 {
		slog.Error("Invalid arguments")
		fmt.Println("Usage: client <action> <param>")
		fmt.Println("Actions:")
		fmt.Println("  greetings <name>")
		fmt.Println("  countdown <start>")
		os.Exit(1)
	}

	action := os.Args[1]
	param := os.Args[2]

	interceptor, err := sniffer.NewGrpcJsonInterceptor(
		sniffer.WithFilename("grpc_client_capture.json"), sniffer.WithAddr("localhost:8081"))
	if err != nil {
		slog.Error("failed to create capture interceptor", "error", err)
		return
	}

	conn, err := grpc.NewClient(grpcServerAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(interceptor.StreamClientInterceptor()),
	)
	if err != nil {
		slog.Error("Failed to connect", "error", err)
		return
	}
	defer conn.Close()

	client := demo.NewDemoClient(conn)

	switch action {
	case "greetings":
		slog.Info("Sending Hello request", "name", param)
		resp, err := client.Hello(context.Background(), &demo.HelloRequest{Name: param})
		if err != nil {
			slog.Error("Failed to greet", "error", err)
			return
		}
		slog.Info("Received response", "message", resp.GetMessage())
	case "countdown":
		start, err := strconv.Atoi(param)
		if err != nil {
			slog.Error("Invalid start value", "error", err)
			return
		}
		slog.Info("Starting Countdown", "start", start)
		stream, err := client.Countdown(context.Background(), &demo.CountdownRequest{Start: int32(start)})
		if err != nil {
			slog.Error("Failed to start countdown", "error", err)
			return
		}
		for {
			resp, err := stream.Recv()
			if err != nil {
				break
			}
			slog.Info("Countdown", "count", resp.GetCount())
		}
	default:
		flag.Usage()
	}
}
