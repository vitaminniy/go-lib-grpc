// Package golibgrpc provides tools for gRPC-related code.
package golibgrpc

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Allower represents a circuit breaker that only checks whether a request can
// proceed and expects the caller to report the outcome in a separate step
// using a callback.
type Allower interface {
	// Allow checks that request can proceed and returns the callback to report
	// the outcome.
	Allow() (done func(success bool), err error)
}

// CircuitBreakerCallOption is a grpc.CallOption that controls circuit breaker
// usage per gRPC call.
type CircuitBreakerCallOption struct {
	grpc.EmptyCallOption

	apply func(*CircuitBreakerInterceptorConfig)
}

// CircuitBreakerInterceptorConfig controls circuit breaker interceptor.
type CircuitBreakerInterceptorConfig struct {
	// Enabled enables circuit breaker usage.
	Enabled bool
}

// WithCircuitBreakerEnabled enables circuit breaker for this particular call.
func WithCircuitBreakerEnabled() CircuitBreakerCallOption {
	return CircuitBreakerCallOption{
		apply: func(cfg *CircuitBreakerInterceptorConfig) {
			cfg.Enabled = true
		},
	}
}

// CircuitBreakerClientInterceptor return grpc.UnaryClientInterceptor that can
// prevents calls that are likely to fail.
func CircuitBreakerClientInterceptor(allower Allower) grpc.UnaryClientInterceptor {
	if allower == nil {
		return noopinterceptor()
	}

	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) (err error) {
		grpcopts, cbopts := filterCircuitBreakerCallOptions(opts)

		var cfg CircuitBreakerInterceptorConfig
		for _, opt := range cbopts {
			opt.apply(&cfg)
		}

		if !cfg.Enabled {
			return invoker(ctx, method, req, reply, cc, grpcopts...)
		}

		done, err := allower.Allow()
		if err != nil {
			return fmt.Errorf("request is not allowed: %w", err)
		}
		defer func() { done(isSuccessCall(err)) }()

		return invoker(ctx, method, req, reply, cc, grpcopts...)
	}
}

func filterCircuitBreakerCallOptions(options []grpc.CallOption) (
	grpcopts []grpc.CallOption,
	cbopts []CircuitBreakerCallOption,
) {
	for _, opt := range options {
		if co, ok := opt.(CircuitBreakerCallOption); ok {
			cbopts = append(cbopts, co)
		} else {
			grpcopts = append(grpcopts, opt)
		}
	}

	return
}

func isSuccessCall(err error) bool {
	if err == nil {
		return true
	}

	if errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled {
		return true
	}

	return false
}
