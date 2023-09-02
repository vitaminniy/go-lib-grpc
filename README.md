# go-lib-grpc

A collection of tools for gRPC related code.

## Contributing

Call `make` or `make help` for available commands.

## Usage

```go
type Config struct {
    ServerAddr string
}

func main() {
    cfg := LoadConfig()

    circuitbreaker := NewCircuitBreaker(cfg)

    interceptoropts := []golibgrpc.UnaryClientInterceptorOption{
        golibgrpc.WithCircuitBreakerInterceptor(circuitbreaker),
    }
    interceptor := golibgrpc.UnaryClientInterceptor(interceptoropts...)

    conn, err := grpc.Dial(
        cfg.ServerAddr,
        grpc.WithUnaryClientInterceptor(interceptor),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    cli := NewClient(conn)
    qos := golibgrpc.QOS{
   	    Timeout:               time.Millisecond * 300,
	    RetryAttempts:         2,
	    CircuitBreakerEnabled: true,
	    GZIPEnabled:           true,
    }

    ctx, cancel := qos.Context(context.Background())
    defer cancel()

    cli.GetResponse(ctx, &Request{}, qos.CallOptions()...)
}

```
