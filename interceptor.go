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
	messageId int // Unique identifier for each message.
	streamId  int // Unique identifier for each stream.
	marshaler protojson.MarshalOptions
	viewer    *GrpcWebViewer
}

type serverStreamWrapper struct {
	grpc.ServerStream
	info        *grpc.StreamServerInfo
	interceptor *GrpcJsonInterceptor
	streamId    int
}

type grpcJsonInterceptorOptions struct {
	Filename string
	Addr     string
}

type capturedMessage struct {
	MessageId  int             `json:"message_id"`
	StreamId   *int            `json:"stream_id,omitempty"`
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
		streamId:  1,
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

func (i *GrpcJsonInterceptor) writeMessage(ctx context.Context, direction direction, fullMethod string, payload any, handlerError error, streamId *int) {
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

	i.mutex.Lock()
	defer i.mutex.Unlock()

	m := capturedMessage{
		MessageId:  i.messageId,
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

	i.messageId++

	return
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
		i.mutex.Lock()
		wrapper := &serverStreamWrapper{
			ServerStream: stream,
			info:         info,
			interceptor:  i,
			streamId:     i.streamId,
		}
		i.streamId++
		i.mutex.Unlock()

		return handler(srv, wrapper)
	}
}

func (ssw *serverStreamWrapper) RecvMsg(m interface{}) error {
	err := ssw.ServerStream.RecvMsg(m)
	ssw.interceptor.writeMessage(ssw.Context(), directionReceive, ssw.info.FullMethod, m, err, &ssw.streamId)
	return err
}

func (ssw *serverStreamWrapper) SendMsg(m interface{}) error {
	err := ssw.ServerStream.SendMsg(m)
	ssw.interceptor.writeMessage(ssw.Context(), directionSend, ssw.info.FullMethod, m, err, &ssw.streamId)
	return err
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
		i.mutex.Lock()
		streamId := i.streamId
		i.streamId++
		i.mutex.Unlock()

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

type clientStreamWrapper struct {
	grpc.ClientStream
	interceptor *GrpcJsonInterceptor
	method      string
	streamId    int
}

func (csw *clientStreamWrapper) SendMsg(m interface{}) error {
	err := csw.ClientStream.SendMsg(m)
	csw.interceptor.writeMessage(csw.Context(), directionSend, csw.method, m, err, &csw.streamId)
	return err
}

func (csw *clientStreamWrapper) RecvMsg(m interface{}) error {
	err := csw.ClientStream.RecvMsg(m)
	csw.interceptor.writeMessage(csw.Context(), directionReceive, csw.method, m, err, &csw.streamId)
	return err
}
