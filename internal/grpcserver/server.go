// Package grpcserver implements the gollm gRPC service.
package grpcserver

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/goppydae/gollm/internal/agent"
	pb "github.com/goppydae/gollm/internal/gen/gollm/v1"
	"github.com/goppydae/gollm/internal/llm"
	"github.com/goppydae/gollm/internal/session"
	"github.com/goppydae/gollm/internal/tools"
	"github.com/goppydae/gollm/internal/types"
)

// isTerminalEvent reports whether ev signals end-of-turn.
// EventAgentEnd, EventError, and EventAbort are the three ways a turn ends.
func isTerminalEvent(t agent.EventType) bool {
	return t == agent.EventAgentEnd || t == agent.EventError || t == agent.EventAbort
}

const heartbeatInterval = 5 * time.Second

type sessionEntry struct {
	ag       *agent.Agent
	cancelHB context.CancelFunc
}

// Server implements pb.AgentServiceServer.
// Sessions are auto-created on first use; rootCtx cancellation stops all heartbeats.
type Server struct {
	pb.UnimplementedAgentServiceServer

	provider   llm.Provider
	registry   *tools.ToolRegistry
	extensions []agent.Extension
	rootCtx    context.Context  // propagated to all heartbeat goroutines
	manager    *session.Manager // nil = no persistence

	mu       sync.RWMutex
	sessions map[string]*sessionEntry
}

// New creates a new Server. mgr may be nil to disable session persistence.
func New(ctx context.Context, provider llm.Provider, registry *tools.ToolRegistry, mgr *session.Manager, exts []agent.Extension) *Server {
	return &Server{
		provider:   provider,
		registry:   registry,
		extensions: exts,
		rootCtx:    ctx,
		manager:    mgr,
		sessions:   make(map[string]*sessionEntry),
	}
}

// getOrCreate returns the sessionEntry for id, creating a new agent if needed.
// On first creation it attempts to restore the session from disk.
func (s *Server) getOrCreate(id string) *sessionEntry {
	if id == "" {
		id = uuid.New().String()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.sessions[id]; ok {
		return e
	}
	ag := agent.New(s.provider, s.registry)
	ag.SetExtensions(s.extensions)

	if s.manager != nil {
		if saved, err := s.manager.Load(id); err == nil {
			ag.LoadSession(saved.ToTypes())
		}
	}

	hbCtx, hbCancel := context.WithCancel(s.rootCtx)
	e := &sessionEntry{ag: ag, cancelHB: hbCancel}
	s.sessions[id] = e
	go s.runHeartbeat(hbCtx, ag)
	return e
}

func (s *Server) lookup(id string) (*sessionEntry, bool) {
	s.mu.RLock()
	e, ok := s.sessions[id]
	s.mu.RUnlock()
	return e, ok
}

// runHeartbeat publishes a heartbeat event every 5 s while the agent is running.
func (s *Server) runHeartbeat(ctx context.Context, ag *agent.Agent) {
	t := time.NewTicker(heartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if ag.IsRunning() {
				ag.EventBus().Publish(agent.Event{
					Type:    agent.EventHeartbeat,
					Value:   time.Now().UnixMilli(),
					Content: ag.LifecycleState(),
				})
			}
		}
	}
}

// saveSession persists the named session to disk. No-op when manager is nil.
func (s *Server) saveSession(id string) {
	if s.manager == nil {
		return
	}
	s.mu.RLock()
	e, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		return
	}
	st := e.ag.State()
	sess := agentStateToSession(st)
	sess.ID = id // use the gRPC session key as the persistence key
	_ = s.manager.Save(sess)
}

func agentStateToSession(st *agent.AgentState) *session.Session {
	return &session.Session{
		ID:           st.Session.ID,
		ParentID:     st.Session.ParentID,
		Name:         st.Session.Name,
		CreatedAt:    st.Session.CreatedAt,
		UpdatedAt:    time.Now(),
		Model:        st.Model,
		Provider:     st.Provider,
		Thinking:     string(st.Thinking),
		SystemPrompt: st.SystemPrompt,
		Messages:     st.Messages,
	}
}

// SaveAllSessions flushes all in-memory sessions to disk.
func (s *Server) SaveAllSessions() {
	s.mu.RLock()
	ids := make([]string, 0, len(s.sessions))
	for id := range s.sessions {
		ids = append(ids, id)
	}
	s.mu.RUnlock()
	for _, id := range ids {
		s.saveSession(id)
	}
}

// StopAllSessions aborts any in-flight turns so GracefulStop can proceed.
func (s *Server) StopAllSessions() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.sessions {
		if e.ag.IsRunning() {
			e.ag.Abort()
		}
	}
}

// ── Core interaction ───────────────────────────────────────────────────────────

func (s *Server) Prompt(req *pb.PromptRequest, stream pb.AgentService_PromptServer) error {
	if req.SessionId == "" {
		return status.Error(codes.InvalidArgument, "session_id required")
	}
	e := s.getOrCreate(req.SessionId)
	ag := e.ag

	// streamDone is closed inside the subscriber goroutine after the terminal
	// event (agent_end / error / abort) has been sent through the stream.
	// This guarantees all events are delivered before the RPC returns.
	streamDone := make(chan struct{})
	var streamDoneOnce sync.Once
	closeStreamDone := func() { streamDoneOnce.Do(func() { close(streamDone) }) }

	sendErr := make(chan error, 1)

	unsub := ag.Subscribe(func(ev agent.Event) {
		msg := eventToProto(req.SessionId, ev)
		if msg != nil {
			if err := stream.Send(msg); err != nil {
				select {
				case sendErr <- err:
				default:
				}
				closeStreamDone()
				return
			}
		}
		if isTerminalEvent(ev.Type) {
			closeStreamDone()
		}
	})
	defer unsub()

	var imgs []agent.Image
	for _, img := range req.Images {
		imgs = append(imgs, agent.Image{MIMEType: img.MimeType, Data: img.Data})
	}

	if err := ag.Prompt(stream.Context(), req.Message, imgs...); err != nil {
		return status.Errorf(codes.FailedPrecondition, "prompt: %v", err)
	}

	select {
	case <-streamDone:
		s.saveSession(req.SessionId)
		return nil
	case err := <-sendErr:
		return err
	case <-stream.Context().Done():
		return nil
	}
}

func (s *Server) Steer(_ context.Context, req *pb.SteerRequest) (*pb.SteerResponse, error) {
	e, ok := s.lookup(req.SessionId)
	if !ok {
		return &pb.SteerResponse{Ok: false}, nil
	}
	e.ag.Steer(req.Message)
	return &pb.SteerResponse{Ok: true}, nil
}

func (s *Server) Abort(_ context.Context, req *pb.AbortRequest) (*pb.AbortResponse, error) {
	e, ok := s.lookup(req.SessionId)
	if !ok {
		return &pb.AbortResponse{Ok: false}, nil
	}
	e.ag.Abort()
	return &pb.AbortResponse{Ok: true}, nil
}

// ── Session management ─────────────────────────────────────────────────────────

func (s *Server) NewSession(_ context.Context, req *pb.NewSessionRequest) (*pb.NewSessionResponse, error) {
	id := req.SessionId
	if id == "" {
		id = uuid.New().String()
	}
	s.mu.Lock()
	if e, ok := s.sessions[id]; ok {
		if e.ag.IsRunning() {
			s.mu.Unlock()
			return nil, status.Error(codes.FailedPrecondition, "agent is running")
		}
		e.ag.Reset()
		if req.Name != "" {
			e.ag.SetSessionName(req.Name)
		}
		s.mu.Unlock()
		return &pb.NewSessionResponse{SessionId: id}, nil
	}
	s.mu.Unlock()
	s.getOrCreate(id)
	return &pb.NewSessionResponse{SessionId: id}, nil
}

func (s *Server) DeleteSession(_ context.Context, req *pb.DeleteSessionRequest) (*pb.DeleteSessionResponse, error) {
	s.mu.Lock()
	e, ok := s.sessions[req.SessionId]
	if ok {
		e.cancelHB()
		delete(s.sessions, req.SessionId)
	}
	s.mu.Unlock()
	if ok && s.manager != nil {
		_ = s.manager.Delete(req.SessionId)
	}
	return &pb.DeleteSessionResponse{Ok: ok}, nil
}

func (s *Server) ListSessions(_ context.Context, _ *pb.ListSessionsRequest) (*pb.ListSessionsResponse, error) {
	s.mu.RLock()
	summaries := make([]*pb.SessionSummary, 0, len(s.sessions))
	inMemory := make(map[string]struct{}, len(s.sessions))
	for id, e := range s.sessions {
		inMemory[id] = struct{}{}
		st := e.ag.State()
		summaries = append(summaries, &pb.SessionSummary{
			SessionId: id,
			Name:      st.Session.Name,
			Lifecycle: e.ag.LifecycleState(),
			CreatedAt: st.Session.CreatedAt.Unix(),
			UpdatedAt: st.Session.UpdatedAt.Unix(),
		})
	}
	s.mu.RUnlock()

	if s.manager != nil {
		if diskSummaries, err := s.manager.ListSummaries(); err == nil {
			for _, ds := range diskSummaries {
				if _, loaded := inMemory[ds.ID]; loaded {
					continue
				}
				summaries = append(summaries, &pb.SessionSummary{
					SessionId: ds.ID,
					Name:      ds.Name,
					Lifecycle: "idle",
					CreatedAt: ds.CreatedAt.Unix(),
					UpdatedAt: ds.UpdatedAt.Unix(),
				})
			}
		}
	}

	return &pb.ListSessionsResponse{Sessions: summaries}, nil
}

// ── State queries ─────────────────────────────────────────────────────────────

func (s *Server) GetState(_ context.Context, req *pb.GetStateRequest) (*pb.GetStateResponse, error) {
	e, ok := s.lookup(req.SessionId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "session %q not found", req.SessionId)
	}
	st := e.ag.State()
	return &pb.GetStateResponse{
		SessionId:     req.SessionId,
		Model:         st.Model,
		Provider:      st.Provider,
		ThinkingLevel: string(st.Thinking),
		Lifecycle:     e.ag.LifecycleState(),
		MessageCount:  int32(len(st.Messages)),
	}, nil
}

func (s *Server) GetMessages(_ context.Context, req *pb.GetMessagesRequest) (*pb.GetMessagesResponse, error) {
	e, ok := s.lookup(req.SessionId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "session %q not found", req.SessionId)
	}
	msgs := e.ag.Messages()
	out := make([]*pb.ConversationMessage, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, &pb.ConversationMessage{
			Role:      m.Role,
			Content:   m.Content,
			Timestamp: m.Timestamp.UnixMilli(),
		})
	}
	return &pb.GetMessagesResponse{Messages: out}, nil
}

// ── Configuration ─────────────────────────────────────────────────────────────

func (s *Server) SetModel(_ context.Context, req *pb.SetModelRequest) (*pb.SetModelResponse, error) {
	e, ok := s.lookup(req.SessionId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "session %q not found", req.SessionId)
	}
	e.ag.SetModel(req.Model)
	return &pb.SetModelResponse{Ok: true}, nil
}

func (s *Server) SetThinkingLevel(_ context.Context, req *pb.SetThinkingLevelRequest) (*pb.SetThinkingLevelResponse, error) {
	e, ok := s.lookup(req.SessionId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "session %q not found", req.SessionId)
	}
	e.ag.SetThinkingLevel(types.ThinkingLevel(req.ThinkingLevel))
	return &pb.SetThinkingLevelResponse{Ok: true}, nil
}

func (s *Server) SetSessionName(_ context.Context, req *pb.SetSessionNameRequest) (*pb.SetSessionNameResponse, error) {
	e, ok := s.lookup(req.SessionId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "session %q not found", req.SessionId)
	}
	e.ag.SetSessionName(req.Name)
	return &pb.SetSessionNameResponse{Ok: true}, nil
}

func (s *Server) Compact(ctx context.Context, req *pb.CompactRequest) (*pb.CompactResponse, error) {
	e, ok := s.lookup(req.SessionId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "session %q not found", req.SessionId)
	}
	e.ag.Compact(ctx, 0)
	return &pb.CompactResponse{Ok: true}, nil
}

// ── Event mapping ─────────────────────────────────────────────────────────────

func eventToProto(sessionID string, ev agent.Event) *pb.AgentEvent {
	out := &pb.AgentEvent{SessionId: sessionID}
	switch ev.Type {
	case agent.EventAgentStart:
		out.Payload = &pb.AgentEvent_AgentStart{AgentStart: &pb.AgentStartEvent{}}
	case agent.EventTurnStart:
		out.Payload = &pb.AgentEvent_TurnStart{TurnStart: &pb.TurnStartEvent{}}
	case agent.EventMessageStart:
		out.Payload = &pb.AgentEvent_MessageStart{MessageStart: &pb.MessageStartEvent{}}
	case agent.EventTextDelta:
		out.Payload = &pb.AgentEvent_TextDelta{TextDelta: &pb.TextDeltaEvent{Content: ev.Content}}
	case agent.EventThinkingDelta:
		out.Payload = &pb.AgentEvent_ThinkingDelta{ThinkingDelta: &pb.ThinkingDeltaEvent{Content: ev.Content}}
	case agent.EventToolCall:
		if ev.ToolCall == nil {
			return nil
		}
		out.Payload = &pb.AgentEvent_ToolCall{ToolCall: &pb.ToolCallEvent{
			Id:       ev.ToolCall.ID,
			Name:     ev.ToolCall.Name,
			ArgsJson: string(ev.ToolCall.Args),
			Position: int32(ev.ToolCall.Position),
		}}
	case agent.EventToolDelta:
		toolID := ""
		if ev.ToolCall != nil {
			toolID = ev.ToolCall.ID
		}
		out.Payload = &pb.AgentEvent_ToolDelta{ToolDelta: &pb.ToolDeltaEvent{
			Content:    ev.Content,
			ToolCallId: toolID,
		}}
	case agent.EventToolOutput:
		if ev.ToolOutput == nil {
			return nil
		}
		out.Payload = &pb.AgentEvent_ToolOutput{ToolOutput: &pb.ToolOutputEvent{
			ToolCallId: ev.ToolOutput.ToolCallID,
			ToolName:   ev.ToolOutput.ToolName,
			Content:    ev.ToolOutput.Content,
			IsError:    ev.ToolOutput.IsError,
		}}
	case agent.EventMessageEnd:
		msg := &pb.MessageEndEvent{}
		if ev.Usage != nil {
			msg.PromptTokens = int32(ev.Usage.PromptTokens)
			msg.CompletionTokens = int32(ev.Usage.CompletionTokens)
			msg.TotalTokens = int32(ev.Usage.TotalTokens)
		}
		out.Payload = &pb.AgentEvent_MessageEnd{MessageEnd: msg}
	case agent.EventAgentEnd:
		out.Payload = &pb.AgentEvent_AgentEnd{AgentEnd: &pb.AgentEndEvent{}}
	case agent.EventError:
		msg := ""
		if ev.Error != nil {
			msg = ev.Error.Error()
		}
		out.Payload = &pb.AgentEvent_Error{Error: &pb.ErrorEvent{Message: msg}}
	case agent.EventAbort:
		out.Payload = &pb.AgentEvent_Abort{Abort: &pb.AbortEvent{}}
	case agent.EventStateChange:
		if ev.StateChange == nil {
			return nil
		}
		out.Payload = &pb.AgentEvent_StateChange{StateChange: &pb.StateChangeEvent{
			From: string(ev.StateChange.From),
			To:   string(ev.StateChange.To),
		}}
	case agent.EventTokens:
		out.Payload = &pb.AgentEvent_Tokens{Tokens: &pb.TokensEvent{Value: ev.Value}}
	case agent.EventHeartbeat:
		out.Payload = &pb.AgentEvent_Heartbeat{Heartbeat: &pb.HeartbeatEvent{
			Timestamp: ev.Value,
			Lifecycle: ev.Content,
		}}
	case agent.EventCompactStart:
		out.Payload = &pb.AgentEvent_CompactStart{CompactStart: &pb.CompactStartEvent{Message: ev.Content}}
	case agent.EventCompactEnd:
		out.Payload = &pb.AgentEvent_CompactEnd{CompactEnd: &pb.CompactEndEvent{}}
	case agent.EventQueueUpdate:
		out.Payload = &pb.AgentEvent_QueueUpdate{QueueUpdate: &pb.QueueUpdateEvent{}}
	default:
		return nil
	}
	return out
}
