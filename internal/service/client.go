package service

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/goppydae/gollm/internal/gen/gollm/v1"
)

// NewInProcessClient creates a pb.AgentServiceClient that talks to an in-memory Service instance.
// It returns the client and a cleanup function to stop the server and close the connection.
func NewInProcessClient(srv *Service) (pb.AgentServiceClient, func(), error) {
	const bufSize = 1024 * 1024
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterAgentServiceServer(s, srv)

	go func() {
		_ = s.Serve(lis)
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}

	client := pb.NewAgentServiceClient(conn)
	cleanup := func() {
		_ = conn.Close()
		s.Stop()
	}

	return client, cleanup, nil
}
