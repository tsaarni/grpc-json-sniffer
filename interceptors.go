package grpc_json_sniffer

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// GrpcJsonInterceptor intercepts gRPC calls and logs the request and response messages as JSON to a file.
// It also serves a web viewer for the logged messages.
type GrpcJsonInterceptor struct {
	output    *os.File
	messageId int64 // Unique identifier for each message.
	streamId  int64 // Unique identifier for each stream.
	marshaler protojson.MarshalOptions
	viewer    *GrpcWebViewer
}

type grpcJsonInterceptorOptions struct {
	Filename string
	Addr     string
}

type capturedMessage struct {
	MessageId  int64           `json:"message_id"`
	StreamId   *int64          `json:"stream_id,omitempty"`
	Direction  direction       `json:"direction"`
	Time       string          `json:"time"`
	FullMethod string          `json:"method"`
	Message    string          `json:"message"`
	PeerAddr   string          `json:"peer_address"`
	Error      string          `json:"error,omitempty"`
	Content    json.RawMessage `json:"content"`
}

type direction string

const (
	directionSend    direction = "send"
	directionReceive direction = "recv"
)

// NewGrpcJsonInterceptor creates a new GrpcJsonInterceptor instance.
//
// It can be configured using the environment variables:
// - GRPC_JSON_SNIFFER_FILE: enables JSON logging to a specified file.
// - GRPC_JSON_SNIFFER_ADDR: enables serving the web viewer at a specified address.
//
// Alternatively, it can be configured through options:
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
		output: f,
		marshaler: protojson.MarshalOptions{
			EmitUnpopulated: true,
		},
		viewer: viewer,
	}, nil
}

// WithFilename sets the filename for the GrpcJsonInterceptor.
//
// Example:
//
//	interceptor, err := NewGrpcJsonInterceptor(WithFilename("grpc_messages.json"))
func WithFilename(filename string) func(*grpcJsonInterceptorOptions) {
	return func(o *grpcJsonInterceptorOptions) {
		o.Filename = filename
	}
}

// WithAddr sets the address for the GrpcJsonInterceptor.
//
// Example:
//
//	interceptor, err := NewGrpcJsonInterceptor(WithAddr("localhost:8080"))
func WithAddr(addr string) func(*grpcJsonInterceptorOptions) {
	return func(o *grpcJsonInterceptorOptions) {
		o.Addr = addr
	}
}

func (i *GrpcJsonInterceptor) writeMessage(ctx context.Context, direction direction, fullMethod string, payload any, handlerError error, streamId *int64) {
	msg, ok := payload.(proto.Message)
	if !ok {
		return
	}

	handlerErrorMessage := ""
	if handlerError != nil {
		handlerErrorMessage = fmt.Sprintf("%v", handlerError)
	}

	b, err := i.marshaler.Marshal(msg)
	if err != nil {
		return
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

	messageId := atomic.AddInt64(&i.messageId, 1)

	m := capturedMessage{
		MessageId:  messageId,
		Direction:  direction,
		Time:       time.Now().Format(time.RFC3339Nano),
		FullMethod: fullMethod,
		Message:    string(msg.ProtoReflect().Descriptor().FullName()),
		StreamId:   streamId,
		PeerAddr:   peerAddr,
		Error:      handlerErrorMessage,
		Content:    json.RawMessage(b),
	}

	var data []byte
	if data, handlerError = json.Marshal(m); handlerError != nil {
		return
	}

	if _, err := i.output.Write(data); err != nil {
		return
	}
	_, _ = i.output.WriteString("\n")
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
		i.writeMessage(ctx, directionReceive, info.FullMethod, req, nil, nil)
		resp, err := handler(ctx, req)
		i.writeMessage(ctx, directionSend, info.FullMethod, req, err, nil)
		return resp, err
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
		streamId := atomic.AddInt64(&i.streamId, 1)

		wrapper := &serverStreamWrapper{
			ServerStream: stream,
			info:         info,
			interceptor:  i,
			streamId:     streamId,
		}

		return handler(srv, wrapper)
	}
}

// UnaryClientInterceptor returns a gRPC unary client interceptor that logs the request and response messages as JSON.
func (i *GrpcJsonInterceptor) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	// If no output file is provided, return an interceptor that does nothing.
	if i.output == nil {
		return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
	}

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		i.writeMessage(ctx, directionReceive, method, req, nil, nil)
		err := invoker(ctx, method, req, reply, cc, opts...)
		i.writeMessage(ctx, directionSend, method, reply, err, nil)
		return err
	}
}

// StreamClientInterceptor returns a gRPC stream client interceptor that logs the request and response messages as JSON.
func (i *GrpcJsonInterceptor) StreamClientInterceptor() grpc.StreamClientInterceptor {
	// If no output file is provided, return an interceptor that does nothing.
	if i.output == nil {
		return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			return streamer(ctx, desc, cc, method, opts...)
		}
	}

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		streamId := atomic.AddInt64(&i.streamId, 1)

		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}

		wrappedStream := &clientStreamWrapper{
			ClientStream: clientStream,
			interceptor:  i,
			method:       method,
			streamId:     streamId,
		}

		return wrappedStream, nil
	}
}
