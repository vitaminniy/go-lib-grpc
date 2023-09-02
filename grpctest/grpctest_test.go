package grpctest

import (
	"context"
	"fmt"
	"log"

	"github.com/vitaminniy/go-lib-grpc/internal/mockserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func ExampleNewServer() {
	const id = "test"

	handler := func(context.Context, *mockserver.Request) (*mockserver.Response, error) {
		return &mockserver.Response{Id: id}, nil
	}

	mock := mockserver.MockedSampleServer{}
	mock.Handler = handler

	srv := NewServer(&mockserver.Sample_ServiceDesc, &mock)
	defer srv.Close()

	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.Dial(srv.Addr, opts...)
	if err != nil {
		log.Fatalf("could not dial server: %v", err)
	}
	defer conn.Close()

	cli := mockserver.NewSampleClient(conn)

	resp, err := cli.GetResponse(context.Background(), &mockserver.Request{})
	if err != nil {
		log.Fatalf("could not get response: %v", err)
	}

	fmt.Println(resp.Id)

	// Output: test
}
