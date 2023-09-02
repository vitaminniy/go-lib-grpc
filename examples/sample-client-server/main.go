package main

import (
	"context"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	golibgrpc "github.com/vitaminniy/go-lib-grpc"
	"github.com/vitaminniy/go-lib-grpc/internal/mockserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var qos = golibgrpc.QOS{
	Timeout:               time.Millisecond * 300,
	RetryAttempts:         2,
	CircuitBreakerEnabled: true,
	GZIPEnabled:           true,
}

func main() {
	log.SetPrefix("sample-client-server: ")
	log.SetFlags(0)

	rand.Seed(time.Now().Unix())

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("could not start listening for conns: %v", err)
	}
	defer l.Close()

	sample := mockserver.MockedSampleServer{}
	sample.Handler = handler

	server := grpc.NewServer()
	server.RegisterService(&mockserver.Sample_ServiceDesc, &sample)

	errs := make(chan error, 1)
	go func() {
		log.Printf("server started on %q", l.Addr().String())

		errs <- server.Serve(l)
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	dialopts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithUnaryInterceptor(golibgrpc.UnaryClientInterceptor()),
	}

	conn, err := grpc.DialContext(ctx, l.Addr().String(), dialopts...)
	if err != nil {
		log.Fatalf("could not dial server: %v", err)
	}
	defer conn.Close()

	cli := mockserver.NewSampleClient(conn)

	go dorequests(ctx, qos, cli)

	select {
	case <-ctx.Done():
		server.GracefulStop()

		log.Println("server stopped")
	case err := <-errs:
		log.Fatalf("could not serve gRPC: %v", err)
	}
}

func handler(_ context.Context, req *mockserver.Request) (*mockserver.Response, error) {
	timeout := rand.Int63n(qos.Timeout.Milliseconds() + 10)
	time.Sleep(time.Duration(timeout))

	return &mockserver.Response{Id: req.Id}, nil
}

func dorequests(ctx context.Context, qos golibgrpc.QOS, cli mockserver.SampleClient) {
	timer := time.NewTimer(0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			timer.Reset(time.Second)
		}

		dorequest(ctx, qos, cli)
	}
}

func dorequest(ctx context.Context, qos golibgrpc.QOS, cli mockserver.SampleClient) {
	ctx, cancel := qos.Context(ctx)
	defer cancel()

	opts := qos.CallOptions()

	cli.GetResponse(ctx, &mockserver.Request{}, opts...)
}
