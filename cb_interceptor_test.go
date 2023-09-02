package golibgrpc

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/vitaminniy/go-lib-grpc/grpctest"
	"github.com/vitaminniy/go-lib-grpc/internal/mockserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type closeAfterFailAllower struct {
	failed atomic.Bool
}

func (a *closeAfterFailAllower) Allow() (done func(bool), err error) {
	failed := a.failed.Load()
	if failed {
		return nil, errors.New("closed indefinetely")
	}

	done = func(success bool) {
		if success {
			return
		}

		a.failed.Store(true)
	}

	return
}

type cbInterceptorTestCase struct {
	name     string
	allower  Allower
	opts     []grpc.CallOption
	attempts int

	want int
}

func TestCircuitBreakerClientInterceptor(t *testing.T) {
	t.Parallel()

	cases := []cbInterceptorTestCase{
		{
			name:     "nil allower",
			allower:  nil,
			attempts: 3,
			want:     3,
		},
		{
			name:     "disabled",
			allower:  &closeAfterFailAllower{},
			attempts: 4,
			want:     4,
		},
		{
			name:     "enabled",
			allower:  &closeAfterFailAllower{},
			opts:     []grpc.CallOption{WithCircuitBreakerEnabled()},
			attempts: 100,
			want:     1,
		},
	}

	for _, testcase := range cases {
		testcase := testcase

		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			testCBInterceptor(t, testcase)
		})
	}
}

func testCBInterceptor(t *testing.T, testcase cbInterceptorTestCase) {
	t.Helper()

	type (
		Request  = mockserver.Request
		Response = mockserver.Response
	)

	var calls atomic.Int32

	mock := mockserver.MockedSampleServer{}
	mock.Handler = func(context.Context, *Request) (*Response, error) {
		calls.Add(1)

		return nil, errors.New("fail on purpose")
	}

	srv := grpctest.NewServer(&mockserver.Sample_ServiceDesc, &mock)
	defer srv.Close()

	dialopts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(CircuitBreakerClientInterceptor(testcase.allower)),
	}

	conn, err := grpc.Dial(srv.Addr, dialopts...)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	cli := mockserver.NewSampleClient(conn)

	for i := 0; i < testcase.attempts; i++ {
		cli.GetResponse(context.Background(), &Request{}, testcase.opts...)
	}

	got := int(calls.Load())

	if testcase.want != got {
		t.Fatalf("calls mismatch: want %d; got %d", testcase.want, got)
	}
}
