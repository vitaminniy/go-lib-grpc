// Package golibgrpc provides tools for gRPC-related code.
package golibgrpc

import (
	"context"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// RequestStartInstant is a request metadata header key for request's start
	// timestamp.
	RequestStartInstant = "Request-Start-Instant"
	// RequestExpiryInstant is a request metadata header key of a request
	// expiration time.
	RequestExpiryInstant = "Request-Expiry-Instant"
)

// MetaConfig controls meta options that would be applied to outgoing gRPC
// request.
type MetaConfig struct {
	NowFunc func() time.Time
	Expiry  time.Duration
}

func (cfg *MetaConfig) pairs() []string {
	const (
		headers       = 2
		valuesPerPair = 2
	)

	now := cfg.now()

	pairs := make([]string, 0, headers*valuesPerPair)
	pairs = append(pairs, RequestStartInstant, formatint(now.UnixMilli()))

	if cfg.Expiry > 0 {
		expiry := now.Add(cfg.Expiry).UnixMilli()
		pairs = append(pairs, RequestExpiryInstant, formatint(expiry))
	}

	return pairs
}

func formatint(value int64) string {
	const base = 10

	return strconv.FormatInt(value, base)
}

func (cfg *MetaConfig) now() time.Time {
	if cfg.NowFunc != nil {
		return cfg.NowFunc()
	}

	return time.Now()
}

// MetaCallOption is a grpc.CallOption that updates request meta to add meta
// headers.
type MetaCallOption struct {
	grpc.EmptyCallOption

	apply func(*MetaConfig)
}

// WithMetaNowFunc sets function to get now time.
func WithMetaNowFunc(nowfn func() time.Time) MetaCallOption {
	return MetaCallOption{apply: func(cfg *MetaConfig) {
		cfg.NowFunc = nowfn
	}}
}

// WithMetaRequestExpiry sets request expiry header.
func WithMetaRequestExpiry(expiry time.Duration) MetaCallOption {
	return MetaCallOption{apply: func(cfg *MetaConfig) {
		cfg.Expiry = expiry
	}}
}

// MetaUnaryClientInterceptor returns grpc.UnaryClientInterceptor that can
// extend request metadata.
func MetaUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		grpcopts, metaopts := filterMetaCallOptions(opts)

		var cfg MetaConfig
		for _, opt := range metaopts {
			opt.apply(&cfg)
		}

		pairs := cfg.pairs()
		ctx = metadata.AppendToOutgoingContext(ctx, pairs...)

		return invoker(ctx, method, req, reply, cc, grpcopts...)
	}
}

func filterMetaCallOptions(options []grpc.CallOption) (
	grpcopts []grpc.CallOption,
	metaopts []MetaCallOption,
) {
	for _, opt := range options {
		if co, ok := opt.(MetaCallOption); ok {
			metaopts = append(metaopts, co)
		} else {
			grpcopts = append(grpcopts, opt)
		}
	}

	return
}
