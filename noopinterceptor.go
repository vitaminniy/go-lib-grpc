// Package golibgrpc provides tools for gRPC-related code.
package golibgrpc

import (
	"context"

	"google.golang.org/grpc"
)

func noopinterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
