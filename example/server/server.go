package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"time"

	sniffer "github.com/tsaarni/grpc-json-sniffer"
	"github.com/tsaarni/grpc-json-sniffer/example/demo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

type server struct {
	demo.UnimplementedDemoServer
}

func peerAddr(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}
	return "unknown"
}

func (s *server) Hello(ctx context.Context, req *demo.HelloRequest) (*demo.HelloReply, error) {
	slog.Info("Received", "name", req.GetName(), "peer", peerAddr(ctx))
	return &demo.HelloReply{Message: "Hello " + req.GetName()}, nil
}

func (s *server) Countdown(req *demo.CountdownRequest, stream demo.Demo_CountdownServer) error {
	for i := req.GetStart(); i >= 0; i-- {
		if err := stream.Send(&demo.CountdownReply{Count: i}); err != nil {
			return err
		}
		slog.Info("Countdown", "count", i, "peer", peerAddr(stream.Context()))
		time.Sleep(1 * time.Second)
	}
	return nil
}

func main() {
	address := flag.String("address", "localhost:50051", "The address to bind the server to")
	flag.Parse()

	lis, err := net.Listen("tcp", *address)
	if err != nil {
		slog.Error("failed to listen", "error", err)
		return
	}

	interceptor, err := sniffer.NewGrpcJsonInterceptor(
		sniffer.WithFilename("grpc_capture.json"), sniffer.WithAddr("localhost:8080"))
	if err != nil {
		slog.Error("failed to create capture interceptor", "error", err)
		return
	}

	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(interceptor.StreamServerInterceptor()),
		grpc.UnaryInterceptor(interceptor.UnaryServerInterceptor()),
	}

	s := grpc.NewServer(opts...)
	demo.RegisterDemoServer(s, &server{})

	fmt.Printf("gRPC server is running on %s\n", *address)
	fmt.Printf("HTTP server is running on http://localhost:8080\n")
	if err := s.Serve(lis); err != nil {
		slog.Error("failed to serve", "error", err)
	}
}
