package modes

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/goppydae/gollm/internal/agent"
	pb "github.com/goppydae/gollm/internal/gen/gollm/v1"
	"github.com/goppydae/gollm/internal/grpcserver"
	"github.com/goppydae/gollm/internal/llm"
	"github.com/goppydae/gollm/internal/session"
	"github.com/goppydae/gollm/internal/tools"
)

// GRPCHandler starts a gRPC server exposing the AgentService.
type GRPCHandler struct {
	Provider   llm.Provider
	Registry   *tools.ToolRegistry
	Manager    *session.Manager
	Extensions []agent.Extension
	Addr       string
}

// NewGRPCHandler creates a GRPCHandler.
func NewGRPCHandler(provider llm.Provider, registry *tools.ToolRegistry, mgr *session.Manager, exts []agent.Extension, addr string) Handler {
	return &GRPCHandler{
		Provider:   provider,
		Registry:   registry,
		Manager:    mgr,
		Extensions: exts,
		Addr:       addr,
	}
}

func (h *GRPCHandler) Run(_ []string) error {
	lis, err := net.Listen("tcp", h.Addr)
	if err != nil {
		return fmt.Errorf("grpc listen %s: %w", h.Addr, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := grpcserver.New(ctx, h.Provider, h.Registry, h.Manager, h.Extensions)
	gs := grpc.NewServer()
	pb.RegisterAgentServiceServer(gs, srv)

	serveErr := make(chan error, 1)
	go func() { serveErr <- gs.Serve(lis) }()
	fmt.Printf("gollm gRPC server listening on %s\n", h.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	select {
	case err := <-serveErr:
		return err
	case <-quit:
	}

	fmt.Println("shutting down gRPC server…")
	cancel()
	srv.SaveAllSessions()
	srv.StopAllSessions()

	done := make(chan struct{})
	go func() { gs.GracefulStop(); close(done) }()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		gs.Stop()
	}
	return nil
}
