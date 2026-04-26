package interactive

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	pb "github.com/goppydae/gollm/internal/gen/gollm/v1"
)

func (m *model) handleAgentEvent(ev *pb.AgentEvent) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch p := ev.Payload.(type) {
	case *pb.AgentEvent_AgentStart:
		_ = p
		m.isRunning = true
		m.startTime = time.Now()
		cmds = append(cmds, m.spinner.Tick)
		cmds = append(cmds, m.stopwatch.Reset())
		cmds = append(cmds, m.stopwatch.Start())
		cmds = append(cmds, m.progressBar.SetPercent(0))

	case *pb.AgentEvent_TextDelta:
		entry := m.ensureAssistantEntry()
		if len(entry.items) > 0 && entry.items[len(entry.items)-1].kind == contentItemText {
			entry.items[len(entry.items)-1].text += p.TextDelta.Content
		} else {
			entry.items = append(entry.items, contentItem{kind: contentItemText, text: p.TextDelta.Content})
		}
		m.tokens += (len(p.TextDelta.Content) + 3) / 4

	case *pb.AgentEvent_ToolCall:
		tc := p.ToolCall
		duplicate := false
		if tc.Id != "" {
			for hIdx := len(m.history) - 1; hIdx >= 0; hIdx-- {
				if m.history[hIdx].role != "assistant" {
					break
				}
				for _, item := range m.history[hIdx].items {
					if item.kind == contentItemToolCall && item.tc.id == tc.Id {
						duplicate = true
						break
					}
				}
				if duplicate {
					break
				}
			}
		}
		if !duplicate {
			entry := m.ensureAssistantEntry()
			arg := extractFirstArgument(tc.ArgsJson)
			entry.items = append(entry.items, contentItem{
				kind: contentItemToolCall,
				tc: toolCallEntry{
					id:     tc.Id,
					name:   tc.Name,
					arg:    arg,
					status: toolCallRunning,
				},
			})
		}

	case *pb.AgentEvent_ToolDelta:
		td := p.ToolDelta
		if td.Content != "" {
			for hIdx := len(m.history) - 1; hIdx >= 0; hIdx-- {
				if m.history[hIdx].role != "assistant" {
					continue
				}
				entry := &m.history[hIdx]
				for i := range entry.items {
					if entry.items[i].kind == contentItemToolCall && entry.items[i].tc.id == td.ToolCallId {
						if entry.items[i].tc.status == toolCallRunning {
							entry.items[i].tc.streamingOutput += td.Content
						}
						break
					}
				}
			}
		}

	case *pb.AgentEvent_ThinkingDelta:
		if p.ThinkingDelta.Content != "" {
			entry := m.ensureAssistantEntry()
			if len(entry.items) > 0 && entry.items[len(entry.items)-1].kind == contentItemThinking {
				entry.items[len(entry.items)-1].text += p.ThinkingDelta.Content
			} else {
				entry.items = append(entry.items, contentItem{kind: contentItemThinking, text: p.ThinkingDelta.Content})
			}
			m.tokens += (len(p.ThinkingDelta.Content) + 3) / 4
		}

	case *pb.AgentEvent_ToolOutput:
		to := p.ToolOutput
		var entry *historyEntry
		found := false
		for hIdx := len(m.history) - 1; hIdx >= 0; hIdx-- {
			if m.history[hIdx].role != "assistant" {
				continue
			}
			entry = &m.history[hIdx]
			for i := range entry.items {
				if entry.items[i].kind == contentItemToolCall && entry.items[i].tc.id == to.ToolCallId {
					if to.ToolCallId == "" && entry.items[i].tc.status != toolCallRunning {
						continue
					}
					entry.items[i].tc.status = toolCallSuccess
					if to.IsError || strings.HasPrefix(to.Content, "Error:") || strings.HasPrefix(to.Content, "tool error:") {
						entry.items[i].tc.status = toolCallFailure
					}
					outItem := contentItem{
						kind: contentItemToolOutput,
						out: toolOutputEntry{
							toolCallID: to.ToolCallId,
							toolName:   to.ToolName,
							content:    to.Content,
							isError:    to.IsError,
						},
					}
					entry.items = append(entry.items, contentItem{})
					copy(entry.items[i+2:], entry.items[i+1:])
					entry.items[i+1] = outItem
					found = true
					break
				}
			}
			if found {
				break
			}
		}

	case *pb.AgentEvent_MessageEnd:
		m.newAssistantEntry = true
		if p.MessageEnd.TotalTokens > 0 {
			m.tokens = int(p.MessageEnd.TotalTokens)
		}

	case *pb.AgentEvent_StateChange:
		sc := p.StateChange
		m.isRunning = (sc.To == "thinking" || sc.To == "executing" || sc.To == "compacting")
		if m.isRunning {
			cmds = append(cmds, m.spinner.Tick)
			cmds = append(cmds, m.stopwatch.Start())
		}

	case *pb.AgentEvent_QueueUpdate:
		_ = p
		// Fetch updated queue counts from the service.
		if state, err := m.client.GetState(context.Background(), &pb.GetStateRequest{SessionId: m.sessionID}); err == nil {
			_ = state // queue counts not yet in GetStateResponse; extend proto if needed
		}

	case *pb.AgentEvent_AgentEnd:
		_ = p
		m.isRunning = false
		cmds = append(cmds, m.stopwatch.Stop())
		cmds = append(cmds, m.stopwatch.Reset())

	case *pb.AgentEvent_Abort:
		_ = p
		m.isRunning = false
		cmds = append(cmds, m.stopwatch.Stop())
		cmds = append(cmds, m.stopwatch.Reset())

	case *pb.AgentEvent_Error:
		m.isRunning = false
		cmds = append(cmds, m.stopwatch.Stop())
		cmds = append(cmds, m.stopwatch.Reset())
		m.history = append(m.history, historyEntry{role: "error", items: []contentItem{{kind: contentItemText, text: p.Error.Message}}})

	case *pb.AgentEvent_Tokens:
		m.tokens = int(p.Tokens.Value)

	case *pb.AgentEvent_CompactStart:
		m.isCompacting.Store(true)
		cmds = append(cmds, m.spinner.Tick)
		cmds = append(cmds, m.stopwatch.Reset())
		cmds = append(cmds, m.stopwatch.Start())
		if p.CompactStart.Message != "" {
			m.history = append(m.history, historyEntry{
				role:  "info",
				items: []contentItem{{kind: contentItemText, text: p.CompactStart.Message}},
			})
		}

	case *pb.AgentEvent_CompactEnd:
		_ = p
		m.isCompacting.Store(false)
		if !m.isRunning {
			cmds = append(cmds, m.stopwatch.Stop())
		}
		m.syncHistoryFromService()
		m.history = append(m.history, historyEntry{
			role:  "success",
			items: []contentItem{{kind: contentItemText, text: "Context compacted."}},
		})
	}

	m.chatContent = m.buildChatContent()
	m.vp.SetContent(m.chatContent)
	if !m.userScrolled {
		m.vp.GotoBottom()
	}

	cmds = append(cmds, listenForEvent(m.eventCh))
	return m, tea.Batch(cmds...)
}

// syncHistoryFromService rebuilds the TUI history from the gRPC service.
func (m *model) syncHistoryFromService() {
	var trailingMeta []historyEntry
	for i := len(m.history) - 1; i >= 0; i-- {
		entry := m.history[i]
		if m.isRunning && entry.role == "assistant" {
			trailingMeta = append([]historyEntry{entry}, trailingMeta...)
			continue
		}
		isNotice := entry.role == "error" || entry.role == "info" || entry.role == "warning" || entry.role == "success"
		content := ""
		if len(entry.items) > 0 {
			content = entry.items[0].text
		}
		isCompactionNotice := strings.Contains(content, "Compacting") || strings.Contains(content, "Context compacted")
		if isNotice && !isCompactionNotice {
			trailingMeta = append([]historyEntry{entry}, trailingMeta...)
		} else {
			break
		}
	}

	resp, err := m.client.GetMessages(context.Background(), &pb.GetMessagesRequest{SessionId: m.sessionID})
	if err != nil {
		return
	}

	msgs := resp.Messages
	m.history = make([]historyEntry, 0, len(msgs)+len(trailingMeta))

	for _, msg := range msgs {
		if msg.Role == "tool" {
			found := false
			for hIdx := len(m.history) - 1; hIdx >= 0; hIdx-- {
				if m.history[hIdx].role != "assistant" {
					continue
				}
				entry := &m.history[hIdx]
				for i := range entry.items {
					if entry.items[i].kind == contentItemToolCall && entry.items[i].tc.id == msg.ToolCallId {
						entry.items[i].tc.status = toolCallSuccess
						if strings.HasPrefix(msg.Content, "Error:") || strings.HasPrefix(msg.Content, "tool error:") {
							entry.items[i].tc.status = toolCallFailure
						}
						outItem := contentItem{
							kind: contentItemToolOutput,
							out: toolOutputEntry{
								toolCallID: msg.ToolCallId,
								content:    msg.Content,
							},
						}
						entry.items = append(entry.items, contentItem{})
						copy(entry.items[i+2:], entry.items[i+1:])
						entry.items[i+1] = outItem
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			continue
		}

		entry := historyEntry{role: msg.Role}

		if msg.Role == "assistant" {
			if msg.Thinking != "" {
				entry.items = append(entry.items, contentItem{kind: contentItemThinking, text: msg.Thinking})
			}
			if msg.Content != "" {
				entry.items = append(entry.items, contentItem{kind: contentItemText, text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				arg := extractFirstArgument(tc.ArgsJson)
				entry.items = append(entry.items, contentItem{
					kind: contentItemToolCall,
					tc: toolCallEntry{
						id:     tc.Id,
						name:   tc.Name,
						arg:    arg,
						status: toolCallRunning,
					},
				})
			}
		} else {
			entry.items = append(entry.items, contentItem{kind: contentItemText, text: msg.Content})
		}

		m.history = append(m.history, entry)
	}
	m.history = append(m.history, trailingMeta...)
	m.newAssistantEntry = true

	// Refresh token count and model info from state.
	if state, err := m.client.GetState(context.Background(), &pb.GetStateRequest{SessionId: m.sessionID}); err == nil {
		m.modelName = state.Model
		m.provider = state.Provider
		m.thinking = state.ThinkingLevel
		if state.ProviderInfo != nil {
			m.contextWindow = int(state.ProviderInfo.ContextWindow)
		}
	}

	m.syncPromptHistory()
	m.chatContent = m.buildChatContent()
	m.vp.SetContent(m.chatContent)
	if !m.userScrolled {
		m.vp.GotoBottom()
	}
}

func (m *model) syncPromptHistory() {
	if m.client == nil {
		return
	}
	resp, err := m.client.GetMessages(context.Background(), &pb.GetMessagesRequest{SessionId: m.sessionID})
	if err != nil {
		return
	}
	m.promptHistory = make([]string, 0)
	seen := make(map[string]bool)
	for _, msg := range resp.Messages {
		if msg.Role == "user" && msg.Content != "" && msg.Content != "Continue" {
			if !seen[msg.Content] {
				m.promptHistory = append(m.promptHistory, msg.Content)
				seen[msg.Content] = true
			}
		}
	}
	m.historyIndex = -1
}

func (m *model) ensureAssistantEntry() *historyEntry {
	if len(m.history) > 0 && m.history[len(m.history)-1].role == "assistant" {
		last := &m.history[len(m.history)-1]
		if len(last.items) == 0 || !m.newAssistantEntry {
			m.newAssistantEntry = false
			return last
		}
	}

	if m.newAssistantEntry || len(m.history) == 0 || m.history[len(m.history)-1].role != "assistant" {
		m.history = append(m.history, historyEntry{role: "assistant"})
		m.newAssistantEntry = false
	}
	return &m.history[len(m.history)-1]
}

func listenForEvent(eventCh <-chan *pb.AgentEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-eventCh
		if !ok {
			return nil
		}
		return agentEventMsg{ev}
	}
}

// extractFirstArgument pulls the first string value from a JSON object or raw string.
func extractFirstArgument(argsJSON string) string {
	if argsJSON == "" {
		return ""
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(argsJSON), &m); err != nil {
		return argsJSON
	}
	for _, v := range m {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			return s
		}
	}
	return argsJSON
}

// getAgentStats retrieves stats from the service for display in modals.
func (m *model) getAgentStats() agentStats {
	state, err := m.client.GetState(context.Background(), &pb.GetStateRequest{SessionId: m.sessionID})
	if err != nil {
		return agentStats{SessionID: m.sessionID}
	}
	msgs, _ := m.client.GetMessages(context.Background(), &pb.GetMessagesRequest{SessionId: m.sessionID})
	var user, asst, toolCalls, toolResults int
	if msgs != nil {
		for _, msg := range msgs.Messages {
			switch msg.Role {
			case "user":
				user++
			case "assistant":
				asst++
				toolCalls += len(msg.ToolCalls)
			case "tool":
				toolResults++
			}
		}
	}
	total := user + asst + toolResults
	return agentStats{
		SessionID:    m.sessionID,
		Name:         state.GetModel(),
		Model:        state.Model,
		Provider:     state.Provider,
		Thinking:     state.ThinkingLevel,
		SessionFile:  m.sessionMgr.SessionPath(m.sessionID),
		UserMessages: user,
		AssistantMsgs: asst,
		ToolCalls:    toolCalls,
		ToolResults:  toolResults,
		TotalMessages: total,
		ContextTokens: m.tokens,
		ContextWindow: m.contextWindow,
	}
}

// agentStats mirrors the data previously provided by agent.AgentStats.
type agentStats struct {
	SessionID      string
	ParentID       string
	Name           string
	Model          string
	Provider       string
	Thinking       string
	SessionFile    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	UserMessages   int
	AssistantMsgs  int
	ToolCalls      int
	ToolResults    int
	TotalMessages  int
	InputTokens    int
	OutputTokens   int
	CacheRead      int
	CacheWrite     int
	TotalTokens    int
	ContextTokens  int
	ContextWindow  int
	Cost           float64
	QueuedSteer    int
	QueuedFollowUp int
}

