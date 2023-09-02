// Package golibgrpc provides tools for gRPC-related code.
package golibgrpc

import (
	"context"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding/gzip"
)

// UnaryClientInterceptorConfig controls unary interceptor creation.
type UnaryClientInterceptorConfig struct {
	Allower Allower
}

// UnaryClientInterceptorOption overrides interceptor config.
type UnaryClientInterceptorOption func(*UnaryClientInterceptorConfig)

// WithCircuitBreakerInterceptor enables circuit breaker interceptor.
func WithCircuitBreakerInterceptor(allower Allower) UnaryClientInterceptorOption {
	return func(cfg *UnaryClientInterceptorConfig) {
		cfg.Allower = allower
	}
}

// UnaryClientInterceptor returns gRPC unary client interceptor.
func UnaryClientInterceptor(opts ...UnaryClientInterceptorOption) grpc.UnaryClientInterceptor {
	var cfg UnaryClientInterceptorConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	interceptors := []grpc.UnaryClientInterceptor{
		grpc_retry.UnaryClientInterceptor(),
		CircuitBreakerClientInterceptor(cfg.Allower),
		MetaUnaryClientInterceptor(),
	}

	return grpc_middleware.ChainUnaryClient(interceptors...)
}

// QOS is a client's quality-of-service config.
type QOS struct {
	// Timeout controls timeout for single attempt.
	//
	// 0 means no timeout.
	Timeout time.Duration
	// Attempts controls the number of request attempts.
	//
	// 0 and 1 are equal and mean that no extra attempts would be done.
	RetryAttempts uint
	// CircuitBreakerEnabled enables circuit-breaker.
	CircuitBreakerEnabled bool
	// GZIPEnabled enables gzip compression on call.
	GZIPEnabled bool
}

// CallOptions returns
func (q *QOS) CallOptions() []grpc.CallOption {
	opts := []grpc.CallOption{
		grpc_retry.WithPerRetryTimeout(q.Timeout),
		grpc_retry.WithMax(q.RetryAttempts),
		grpc_retry.WithCodes(
			codes.ResourceExhausted,
			codes.Unavailable,
			codes.Aborted,
			codes.DeadlineExceeded,
		),
		WithMetaRequestExpiry(q.Timeout),
	}

	if q.CircuitBreakerEnabled {
		opts = append(opts, WithCircuitBreakerEnabled())
	}

	if q.GZIPEnabled {
		opts = append(opts, grpc.UseCompressor(gzip.Name))
	}

	return opts
}

// Context returns per-call context.
func (q *QOS) Context(ctx context.Context) (context.Context, context.CancelFunc) {
	if q.RetryAttempts == 0 && q.Timeout != 0 {
		return context.WithTimeout(ctx, q.Timeout)
	}

	return ctx, func() {}
}
