// Package mockserver contains mockserver for testing purposes.
//
//go:generate protoc --go_out=. --go_opt=paths=source_relative server.proto
//go:generate protoc --go-grpc_out=. --go-grpc_opt=paths=source_relative server.proto
package mockserver

import "context"

// MockedSampleServer is a mocked handler for sample gRPC server.
type MockedSampleServer struct {
	UnimplementedSampleServer

	Handler func(ctx context.Context, req *Request) (*Response, error)
}

// GetResponse returns sample response.
func (srv *MockedSampleServer) GetResponse(ctx context.Context, req *Request) (*Response, error) {
	if srv.Handler == nil {
		return srv.UnimplementedSampleServer.GetResponse(ctx, req)
	}

	return srv.Handler(ctx, req)
}
