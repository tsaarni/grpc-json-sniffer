package grpc_json_sniffer

import (
	"google.golang.org/grpc"
)

type serverStreamWrapper struct {
	grpc.ServerStream
	info        *grpc.StreamServerInfo
	interceptor *GrpcJsonInterceptor
	streamId    int64
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

type clientStreamWrapper struct {
	grpc.ClientStream
	interceptor *GrpcJsonInterceptor
	method      string
	streamId    int64
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
