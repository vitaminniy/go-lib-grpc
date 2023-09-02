package golibgrpc

import (
	"context"
	"testing"
	"time"

	"github.com/vitaminniy/go-lib-grpc/grpctest"
	"github.com/vitaminniy/go-lib-grpc/internal/mockserver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func TestMetaUnaryClientInterceptorSimple(t *testing.T) {
	t.Parallel()

	var (
		now    = time.Date(2022, time.November, 16, 15, 0, 0, 0, time.UTC)
		expiry = time.Second * 10
		nowfn  = func() time.Time { return now }
	)

	mock := metaUnaryClientInterceptorHandler(t, now, now.Add(expiry))

	srv := grpctest.NewServer(&mockserver.Sample_ServiceDesc, &mock)
	defer srv.Close()

	opts := []grpc.DialOption{
		grpc.WithUnaryInterceptor(MetaUnaryClientInterceptor()),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}

	conn, err := grpc.Dial(srv.Addr, opts...)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	cli := mockserver.NewSampleClient(conn)

	callopts := []grpc.CallOption{
		WithMetaNowFunc(nowfn),
		WithMetaRequestExpiry(expiry),
	}

	cli.GetResponse(context.Background(), &mockserver.Request{}, callopts...)
}

func metaUnaryClientInterceptorHandler(
	t *testing.T,
	start time.Time,
	expiry time.Time,
) mockserver.MockedSampleServer {
	t.Helper()

	type (
		Request  = mockserver.Request
		Response = mockserver.Response
	)

	handler := func(ctx context.Context, req *Request) (*Response, error) {
		md, ok := metadata.FromIncomingContext(ctx) //nolint:varnamelen
		if !ok {
			t.Fatal("expected to get request metdata")
		}

		wantstart := formatint(start.UnixMilli())
		gotstart := findKey(t, md, RequestStartInstant)

		if wantstart != gotstart {
			t.Fatalf("%q key mismatch: want %q; got %q",
				RequestStartInstant, wantstart, gotstart)
		}

		wantexpiry := formatint(expiry.UnixMilli())
		gotexpiry := findKey(t, md, RequestExpiryInstant)

		if wantexpiry != gotexpiry {
			t.Fatalf("%q key mismatch: want %q; got %q",
				RequestExpiryInstant, wantexpiry, gotexpiry)
		}

		return &Response{Id: req.Id}, nil
	}

	return mockserver.MockedSampleServer{Handler: handler}
}

func findKey(t *testing.T, md metadata.MD, key string) string {
	t.Helper()

	values := md.Get(key)
	if len(values) == 0 {
		t.Fatalf("values for key %q not found", key)
	}

	return values[0]
}
