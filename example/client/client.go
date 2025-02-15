package main

import (
	"context"
	"flag"
	"log/slog"
	"strconv"

	"github.com/tsaarni/grpc-json-sniffer/example/demo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	action := flag.String("action", "", "Action to perform: greetings or countdown")
	param := flag.String("param", "", "Name for greetings or start value for countdown")
	address := flag.String("address", "localhost:50051", "The address to connect to")
	flag.Parse()

	if *action == "" || *param == "" {
		slog.Error("Invalid arguments")
		flag.Usage()
		return
	}

	conn, err := grpc.NewClient(*address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("Failed to connect", "error", err)
		return
	}
	defer conn.Close()

	client := demo.NewDemoClient(conn)

	switch *action {
	case "greetings":
		slog.Info("Sending Hello request", "name", *param)
		resp, err := client.Hello(context.Background(), &demo.HelloRequest{Name: *param})
		if err != nil {
			slog.Error("Failed to greet", "error", err)
			return
		}
		slog.Info("Received response", "message", resp.GetMessage())
	case "countdown":
		start, err := strconv.Atoi(*param)
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
