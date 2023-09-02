// Package grpctest provides facilities to test gRPC related code.
package grpctest

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
)

// NewServer creates new gRPC server with provided description and handler.
func NewServer(sd *grpc.ServiceDesc, ss any) *Server {
	ts := NewUnstartedServer(sd, ss)
	ts.Start()

	return ts
}

// NewUnstartedServer returns new unstarted test server.
func NewUnstartedServer(sd *grpc.ServiceDesc, ss any) *Server {
	srv := grpc.NewServer()
	srv.RegisterService(sd, ss)

	return &Server{
		listener: newLocalListener(),
		srv:      srv,
	}
}

// Server is a test gRPC server.
type Server struct {
	Addr string

	listener net.Listener
	srv      *grpc.Server
	wg       sync.WaitGroup
}

// Start starts server.
func (s *Server) Start() {
	if s.Addr != "" {
		panic("Server already started")
	}

	s.Addr = s.listener.Addr().String()

	s.goServe()
}

// Close waits for all connections to finish and closes server.
func (s *Server) Close() {
	s.srv.GracefulStop()

	if err := s.listener.Close(); err != nil {
		if !errors.Is(err, net.ErrClosed) {
			log.Printf("could not close listener: %v", err)
		}
	}

	s.wg.Wait()
}

func (s *Server) goServe() {
	s.wg.Add(1)
	go func() {
		s.wg.Done()

		s.srv.Serve(s.listener)
	}()
}

func newLocalListener() net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			panic(fmt.Sprintf("httptest: failed to listen on a port: %v", err))
		}
	}

	return l
}
