package interactive

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/goppydae/gollm/internal/agent"
)

func (m *model) handleAgentEvent(ev agent.Event) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch ev.Type {
	case agent.EventAgentStart:
		m.isRunning = true
		m.startTime = time.Now()
		cmds = append(cmds, m.spinner.Tick)
		cmds = append(cmds, m.stopwatch.Reset())
		cmds = append(cmds, m.stopwatch.Start())
		cmds = append(cmds, m.progressBar.SetPercent(0))

	case agent.EventTextDelta:
		entry := m.ensureAssistantEntry()
		if len(entry.items) > 0 && entry.items[len(entry.items)-1].kind == contentItemText {
			entry.items[len(entry.items)-1].text += ev.Content
		} else {
			entry.items = append(entry.items, contentItem{kind: contentItemText, text: ev.Content})
		}
		// Live estimate: 4 chars ≈ 1 token
		m.tokens += (len(ev.Content) + 3) / 4

	case agent.EventToolCall:
		if ev.ToolCall != nil {
			// Deduplicate: don't add the same tool call ID twice in the current conversation turn.
			// ONLY deduplicate if the ID is not empty. Empty IDs (common with some models)
			// should always be treated as unique tool calls.
			duplicate := false
			if ev.ToolCall.ID != "" {
				for hIdx := len(m.history) - 1; hIdx >= 0; hIdx-- {
					if m.history[hIdx].role != "assistant" {
						break
					}
					for _, item := range m.history[hIdx].items {
						if item.kind == contentItemToolCall && item.tc.id == ev.ToolCall.ID {
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
				arg := extractFirstArgument(string(ev.ToolCall.Args))
				entry.items = append(entry.items, contentItem{
					kind: contentItemToolCall,
					tc: toolCallEntry{
						id:     ev.ToolCall.ID,
						name:   ev.ToolCall.Name,
						arg:    arg,
						status: toolCallRunning,
					},
				})
			}
		}

	case agent.EventToolDelta:
		if ev.ToolCall != nil && ev.Content != "" {
			for hIdx := len(m.history) - 1; hIdx >= 0; hIdx-- {
				if m.history[hIdx].role != "assistant" {
					continue
				}
				entry := &m.history[hIdx]
				for i := range entry.items {
					if entry.items[i].kind == contentItemToolCall && entry.items[i].tc.id == ev.ToolCall.ID {
						// Don't update if already finished
						if entry.items[i].tc.status == toolCallRunning {
							entry.items[i].tc.streamingOutput += ev.Content
						}
						break
					}
				}
			}
		}

	case agent.EventThinkingDelta:
		if ev.Content != "" {
			entry := m.ensureAssistantEntry()
			// Update last thinking item in-place
			if len(entry.items) > 0 && entry.items[len(entry.items)-1].kind == contentItemThinking {
				entry.items[len(entry.items)-1].text += ev.Content
			} else {
				entry.items = append(entry.items, contentItem{kind: contentItemThinking, text: ev.Content})
			}
			// Live estimate: 4 chars ≈ 1 token
			m.tokens += (len(ev.Content) + 3) / 4
		}

	case agent.EventToolOutput:
		if ev.ToolOutput != nil {
			// Find the assistant entry that contains this tool call.
			// We search backwards from the end of history.
			var entry *historyEntry
			found := false
			for hIdx := len(m.history) - 1; hIdx >= 0; hIdx-- {
				if m.history[hIdx].role != "assistant" {
					continue
				}
				entry = &m.history[hIdx]
				for i := range entry.items {
					if entry.items[i].kind == contentItemToolCall && entry.items[i].tc.id == ev.ToolOutput.ToolCallID {
						// If ID is empty, only match if it's still running (to handle multiple empty-ID calls in order)
						if ev.ToolOutput.ToolCallID == "" && entry.items[i].tc.status != toolCallRunning {
							continue
						}

						entry.items[i].tc.status = toolCallSuccess
						if ev.ToolOutput.IsError || strings.HasPrefix(ev.ToolOutput.Content, "Error:") || strings.HasPrefix(ev.ToolOutput.Content, "tool error:") {
							entry.items[i].tc.status = toolCallFailure
						}
						// Insert output right after the tool call.
						outItem := contentItem{
							kind: contentItemToolOutput,
							out: toolOutputEntry{
								toolCallID: ev.ToolOutput.ToolCallID,
								toolName:   ev.ToolOutput.ToolName,
								content:    ev.ToolOutput.Content,
								isError:    ev.ToolOutput.IsError,
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
		}

	case agent.EventMessageEnd:
		m.newAssistantEntry = true

	case agent.EventStateChange:
		if ev.StateChange != nil {
			m.isRunning = (ev.StateChange.To == agent.StateThinking ||
			               ev.StateChange.To == agent.StateExecuting ||
			               ev.StateChange.To == agent.StateCompacting)
			if m.isRunning {
				cmds = append(cmds, m.spinner.Tick)
				cmds = append(cmds, m.stopwatch.Start())
			}
		}

	case agent.EventQueueUpdate:
		stats := m.ag.GetStats()
		m.queuedSteer = stats.QueuedSteer
		m.queuedFollowUp = stats.QueuedFollowUp

	case agent.EventAgentEnd, agent.EventAbort:
		m.isRunning = false
		cmds = append(cmds, m.stopwatch.Stop())
		cmds = append(cmds, m.stopwatch.Reset())
		if err := m.saveSession(); err != nil {
			m.history = append(m.history, historyEntry{role: "error", items: []contentItem{{kind: contentItemText, text: "Failed to save session: " + err.Error()}}})
		}

	case agent.EventError:
		if ev.Error != nil {
			m.isRunning = false
			cmds = append(cmds, m.stopwatch.Stop())
			cmds = append(cmds, m.stopwatch.Reset())
			m.history = append(m.history, historyEntry{role: "error", items: []contentItem{{kind: contentItemText, text: ev.Error.Error()}}})
		}

	case agent.EventTokens:
		m.tokens = int(ev.Value)

	case agent.EventCompactStart:
		m.isCompacting.Store(true)
		cmds = append(cmds, m.spinner.Tick)
		cmds = append(cmds, m.stopwatch.Reset())
		cmds = append(cmds, m.stopwatch.Start())
		if ev.Content != "" {
			m.history = append(m.history, historyEntry{
				role:  "info",
				items: []contentItem{{kind: contentItemText, text: ev.Content}},
			})
		}

	case agent.EventCompactEnd:
		m.isCompacting.Store(false)
		if !m.isRunning {
			cmds = append(cmds, m.stopwatch.Stop())
		}
		m.syncHistoryFromAgent()
		if ev.Content != "" {
			m.history = append(m.history, historyEntry{
				role:  "success",
				items: []contentItem{{kind: contentItemText, text: ev.Content}},
			})
		}
	}

	m.chatContent = m.buildChatContent()
	m.vp.SetContent(m.chatContent)
	if !m.userScrolled {
		m.vp.GotoBottom()
	}

	cmds = append(cmds, listenForEvent(m.eventCh))
	return m, tea.Batch(cmds...)
}

func (m *model) syncHistoryFromAgent() {
	// Identify transient or "live" messages that shouldn't be lost.
	// 1. A live assistant response currently being streamed (m.isRunning && role == assistant)
	// 2. UI-only notices like errors or working indicators that aren't in the agent's persisted messages.
	var trailingMeta []historyEntry
	for i := len(m.history) - 1; i >= 0; i-- {
		entry := m.history[i]

		// If we find an assistant message while running, it's our live buffer.
		if m.isRunning && entry.role == "assistant" {
			trailingMeta = append([]historyEntry{entry}, trailingMeta...)
			continue
		}

		// Identify if it's a transient UI notice.
		// We avoid string-matching heuristics and instead trust the role and state.
		isNotice := entry.role == "error" || entry.role == "info" || entry.role == "warning" || entry.role == "success"
		
		// If the notice is related to compaction, we skip it because syncHistoryFromAgent
		// will rebuild the compaction summary from the agent's message list.
		content := ""
		if len(entry.items) > 0 { content = entry.items[0].text }
		isCompactionNotice := strings.Contains(content, "Compacting") || strings.Contains(content, "Context compacted")

		if isNotice && !isCompactionNotice {
			trailingMeta = append([]historyEntry{entry}, trailingMeta...)
		} else {
			// Stop at the first real message that belongs in the agent's history.
			break
		}
	}

	msgs := m.ag.Messages()
	m.history = make([]historyEntry, 0, len(msgs)+len(trailingMeta))

	for _, msg := range msgs {
		if msg.Role == "tool" {
			// Find corresponding tool call and update its status/output
			found := false
			for hIdx := len(m.history) - 1; hIdx >= 0; hIdx-- {
				if m.history[hIdx].role != "assistant" {
					continue
				}
				entry := &m.history[hIdx]
				for i := range entry.items {
					if entry.items[i].kind == contentItemToolCall && entry.items[i].tc.id == msg.ToolCallID {
						entry.items[i].tc.status = toolCallSuccess
						if strings.HasPrefix(msg.Content, "Error:") || strings.HasPrefix(msg.Content, "tool error:") {
							entry.items[i].tc.status = toolCallFailure
						}
						// Add output item
						outItem := contentItem{
							kind: contentItemToolOutput,
							out: toolOutputEntry{
								toolCallID: msg.ToolCallID,
								content:    msg.Content,
							},
						}
						// Insert after the tool call
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

		entry := historyEntry{
			role: msg.Role,
		}

		if msg.Role == "assistant" {
			if msg.Thinking != "" {
				entry.items = append(entry.items, contentItem{kind: contentItemThinking, text: msg.Thinking})
			}
			if msg.Content != "" {
				entry.items = append(entry.items, contentItem{kind: contentItemText, text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				arg := extractFirstArgument(string(tc.Args))
				entry.items = append(entry.items, contentItem{
					kind: contentItemToolCall,
					tc: toolCallEntry{
						id:     tc.ID,
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
	m.tokens = m.ag.EstimateContextTokens()
	m.chatContent = m.buildChatContent()
	m.vp.SetContent(m.chatContent)
	if !m.userScrolled {
		m.vp.GotoBottom()
	}
	m.syncPromptHistory()
}

func (m *model) syncPromptHistory() {
	msgs := m.ag.Messages()
	m.promptHistory = make([]string, 0)
	seen := make(map[string]bool)
	for _, msg := range msgs {
		if msg.Role == "user" && msg.Content != "" && msg.Content != "Continue" {
			if !seen[msg.Content] {
				m.promptHistory = append(m.promptHistory, msg.Content)
				seen[msg.Content] = true
			}
		}
	}
	m.historyIndex = -1
}

// ensureAssistantEntry returns the latest historyEntry if it is of role assistant,
// or creates and appends a new one if necessary.
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

// listenForEvent returns a tea.Cmd that waits for the next agent event.
func listenForEvent(eventCh <-chan agent.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-eventCh
		if !ok {
			return nil
		}
		return agentEventMsg{ev}
	}
}
