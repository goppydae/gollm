package modes

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	pb "github.com/goppydae/sharur/internal/gen/sharur/v1"
	"github.com/goppydae/sharur/internal/service"
)

// GRPCHandler starts a gRPC server exposing the AgentService.
type GRPCHandler struct {
	Service *service.Service
	Addr    string
}

// NewGRPCHandler creates a GRPCHandler.
func NewGRPCHandler(svc *service.Service, addr string) Handler {
	return &GRPCHandler{
		Service: svc,
		Addr:    addr,
	}
}

func (h *GRPCHandler) Run(_ []string) error {
	lis, err := net.Listen("tcp", h.Addr)
	if err != nil {
		return fmt.Errorf("grpc listen %s: %w", h.Addr, err)
	}

	gs := grpc.NewServer()
	pb.RegisterAgentServiceServer(gs, h.Service)

	serveErr := make(chan error, 1)
	go func() { serveErr <- gs.Serve(lis) }()
	fmt.Printf("sharur gRPC server listening on %s\n", h.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-serveErr:
		return err
	case <-quit:
	}

	fmt.Println("shutting down gRPC server…")
	h.Service.SaveAllSessions()
	h.Service.StopAllSessions()

	done := make(chan struct{})
	go func() { gs.GracefulStop(); close(done) }()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		gs.Stop()
	}
	return nil
}
