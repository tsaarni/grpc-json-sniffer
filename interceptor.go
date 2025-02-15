package grpc_json_sniffer

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// GrpcJsonInterceptor intercepts gRPC calls and logs the request and response messages as JSON to a file and serves a web viewer.
type GrpcJsonInterceptor struct {
	output    *os.File
	mutex     sync.Mutex
	messageId int
	marshaler protojson.MarshalOptions
	viewer    *GrpcWebViewer
}

type serverStreamWrapper struct {
	grpc.ServerStream
	info        *grpc.StreamServerInfo
	interceptor *GrpcJsonInterceptor
}

type grpcJsonInterceptorOptions struct {
	Filename string
	Addr     string
}

type capturedMessage struct {
	Id         int             `json:"id"`
	Direction  direction       `json:"direction"`
	Time       string          `json:"time"`
	FullMethod string          `json:"method"`
	Type       string          `json:"type"`
	PeerAddr   string          `json:"peer_address"`
	Message    json.RawMessage `json:"message"`
}

type direction string

const (
	directionSend    direction = "send"
	directionReceive direction = "recv"
)

// NewGrpcJsonInterceptor creates a new GrpcJsonInterceptor instance.
// It can be configured using the environment variables GRPC_JSON_SNIFFER_FILE and GRPC_JSON_SNIFFER_ADDR,
// or through options:
// - WithFilename: enables JSON logging to a specified file.
// - WithAddr: enables serving the web viewer at a specified address.
func NewGrpcJsonInterceptor(options ...func(*grpcJsonInterceptorOptions)) (*GrpcJsonInterceptor, error) {
	opts := grpcJsonInterceptorOptions{
		Filename: os.Getenv("GRPC_JSON_SNIFFER_FILE"),
		Addr:     os.Getenv("GRPC_JSON_SNIFFER_ADDR"),
	}

	for _, option := range options {
		option(&opts)
	}

	// If no filename is provided, return an interceptor that does nothing.
	if opts.Filename == "" {
		return &GrpcJsonInterceptor{}, nil
	}

	f, err := os.OpenFile(opts.Filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}

	var viewer *GrpcWebViewer
	if opts.Addr != "" {
		viewer = NewGrpcWebViewer(opts.Addr, opts.Filename)
		go viewer.Serve()
	}

	return &GrpcJsonInterceptor{
		output:    f,
		messageId: 1,
		marshaler: protojson.MarshalOptions{
			EmitUnpopulated: true,
		},
		viewer: viewer,
	}, nil
}

// WithFilename sets the filename for the GrpcJsonInterceptor.
func WithFilename(filename string) func(*grpcJsonInterceptorOptions) {
	return func(o *grpcJsonInterceptorOptions) {
		o.Filename = filename
	}
}

// WithAddr sets the address for the GrpcJsonInterceptor.
func WithAddr(addr string) func(*grpcJsonInterceptorOptions) {
	return func(o *grpcJsonInterceptorOptions) {
		o.Addr = addr
	}
}

func (i *GrpcJsonInterceptor) writeMessage(ctx context.Context, direction direction, fullMethod string, msg proto.Message) error {
	b, err := i.marshaler.Marshal(msg)
	if err != nil {
		return err
	}
	var peerAddr string
	p, ok := peer.FromContext(ctx)
	if ok {
		if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
			peerAddr = tcpAddr.String()
		} else {
			peerAddr = p.Addr.String()
		}
	} else {
		peerAddr = "unknown"
	}

	i.mutex.Lock()
	defer i.mutex.Unlock()

	m := capturedMessage{
		Id:         i.messageId,
		Direction:  direction,
		Time:       time.Now().Format(time.RFC3339Nano),
		FullMethod: fullMethod,
		Type:       string(msg.ProtoReflect().Descriptor().FullName()),
		PeerAddr:   peerAddr,
		Message:    json.RawMessage(b),
	}

	var data []byte
	if data, err = json.Marshal(m); err != nil {
		return err
	}

	if _, err := i.output.Write(data); err != nil {
		return err
	}
	_, _ = i.output.WriteString("\n")

	i.messageId++

	return nil
}

// UnaryServerInterceptor returns a gRPC unary server interceptor that logs the request and response messages as JSON.
func (i *GrpcJsonInterceptor) UnaryServerInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// If no output file is provided, return an interceptor that does nothing.
	if i.output == nil {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		msg, ok := req.(proto.Message)
		if !ok {
			return nil, fmt.Errorf("request does not implement proto.Message")
		}
		if err := i.writeMessage(ctx, directionReceive, info.FullMethod, msg); err != nil {
			return nil, err
		}

		resp, err := handler(ctx, req)
		if err != nil {
			return nil, err
		}

		msg, ok = resp.(proto.Message)
		if !ok {
			return nil, fmt.Errorf("response does not implement proto.Message")
		}
		if err := i.writeMessage(ctx, directionSend, info.FullMethod, msg); err != nil {
			return nil, err
		}

		return resp, nil
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor that logs the request and response messages as JSON.
func (i *GrpcJsonInterceptor) StreamServerInterceptor() func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// If no output file is provided, return an interceptor that does nothing.
	if i.output == nil {
		return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, stream)
		}
	}

	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &serverStreamWrapper{
			ServerStream: stream,
			info:         info,
			interceptor:  i,
		})
	}
}

func (ssw *serverStreamWrapper) RecvMsg(m interface{}) error {
	msg, ok := m.(proto.Message)
	if !ok {
		return fmt.Errorf("message does not implement proto.Message")
	}
	if err := ssw.interceptor.writeMessage(ssw.Context(), directionReceive, ssw.info.FullMethod, msg); err != nil {
		return err
	}

	return ssw.ServerStream.RecvMsg(m)
}

func (ssw *serverStreamWrapper) SendMsg(m interface{}) error {
	msg, ok := m.(proto.Message)
	if !ok {
		return fmt.Errorf("message does not implement proto.Message")
	}
	if err := ssw.interceptor.writeMessage(ssw.Context(), directionSend, ssw.info.FullMethod, msg); err != nil {
		return err
	}
	return ssw.ServerStream.SendMsg(m)
}
