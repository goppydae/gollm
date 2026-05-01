package grpcserver

import (
	"context"

	"github.com/goppydae/sharur/internal/agent"
	"github.com/goppydae/sharur/internal/llm"
	"github.com/goppydae/sharur/internal/service"
	"github.com/goppydae/sharur/internal/session"
	"github.com/goppydae/sharur/internal/tools"
)

// Server is a thin wrapper (type alias) around service.Service for backward compatibility.
type Server = service.Service

// New creates a new Server instance.
func New(ctx context.Context, provider llm.Provider, registry *tools.ToolRegistry, mgr *session.Manager, exts []agent.Extension) *Server {
	return service.New(ctx, provider, registry, mgr, exts)
}
