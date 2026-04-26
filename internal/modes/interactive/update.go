package interactive

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/stopwatch"
	tea "charm.land/bubbletea/v2"

	pb "github.com/goppydae/gollm/internal/gen/gollm/v1"
	"github.com/goppydae/gollm/internal/prompts"
	"github.com/goppydae/gollm/internal/skills"
)

// Update implements tea.Model.Update.
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(msg.Width - borderOffset)
		m.vp.SetWidth(msg.Width - borderOffset - chatMargin*2)
		m.input.SetHeight(m.currentInputHeight())
		m.vp.SetHeight(m.vpHeight())
		m.refreshViewport()

	case tea.KeyPressMsg:
		var nm tea.Model
		nm, cmd = m.handleKey(msg)
		m = nm.(*model)

	case tea.PasteMsg:
		m.input, cmd = m.input.Update(msg)
		m.input.SetHeight(m.currentInputHeight())

	case agentEventMsg:
		return m.handleAgentEvent(msg.ev)

	case tea.MouseWheelMsg:
		if !m.modal.visible {
			m.vp, cmd = m.vp.Update(msg)
			m.userScrolled = !m.vp.AtBottom()
		}

	case spinner.TickMsg:
		if m.isRunning || m.isCompacting.Load() {
			m.spinner, cmd = m.spinner.Update(msg)
			m.chatContent = m.buildChatContent()
			m.vp.SetContent(m.chatContent)
		}

	case stopwatch.TickMsg:
		if m.isRunning || m.isCompacting.Load() {
			m.stopwatch, cmd = m.stopwatch.Update(msg)
			m.chatContent = m.buildChatContent()
			m.vp.SetContent(m.chatContent)
		}

	case stopwatch.ResetMsg:
		m.stopwatch, cmd = m.stopwatch.Update(msg)

	case progress.FrameMsg:
		if m.isRunning || m.isCompacting.Load() {
			m.progressBar, cmd = m.progressBar.Update(msg)
			m.chatContent = m.buildChatContent()
			m.vp.SetContent(m.chatContent)
		}

	case initialPromptMsg:
		prompt := m.initialPrompt
		m.initialPrompt = ""
		entry := historyEntry{role: "user", items: []contentItem{{kind: contentItemText, text: prompt}}}
		m.history = append(m.history, entry)
		m.newContext()
		if err := m.promptGRPC(m.ctx, prompt); err != nil {
			m.history = append(m.history, historyEntry{role: "error", items: []contentItem{{kind: contentItemText, text: err.Error()}}})
		}
		cmd = listenForEvent(m.eventCh)

	case stopwatch.StartStopMsg:
		m.stopwatch, cmd = m.stopwatch.Update(msg)
	}

	if m.isRunning || m.isCompacting.Load() {
		cmd = tea.Batch(cmd, listenForEvent(m.eventCh))
	}

	return m, cmd
}

// promptGRPC starts a Prompt RPC and drains events into eventCh in a goroutine.
func (m *model) promptGRPC(ctx context.Context, text string, imgAttachments ...*pb.ImageAttachment) error {
	stream, err := m.client.Prompt(ctx, &pb.PromptRequest{
		SessionId: m.sessionID,
		Message:   text,
		Images:    imgAttachments,
	})
	if err != nil {
		return err
	}
	go func() {
		for {
			ev, recvErr := stream.Recv()
			if recvErr != nil {
				return
			}
			select {
			case m.eventCh <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

// onResize simulates a WindowSizeMsg — used by tests.
func (m *model) onResize(width, height int) {
	m.width = width
	m.height = height
	m.input.SetWidth(width - borderOffset)
	m.vp.SetWidth(width - borderOffset - chatMargin*2)
	m.input.SetHeight(m.currentInputHeight())
	m.vp.SetHeight(m.vpHeight())
}

func (m *model) currentInputHeight() int {
	lines := strings.Count(m.input.Value(), "\n") + 1
	if lines < inputHeight {
		return inputHeight
	}
	maxH := m.height / 3
	if maxH < inputHeight {
		maxH = inputHeight
	}
	if lines > maxH {
		return maxH
	}
	return lines
}

func (m *model) vpHeight() int {
	pickerH := 0
	if m.picker.Open {
		if len(m.picker.Matches) == 0 {
			pickerH = 1
		} else {
			pickerH = pickerPageSize
		}
	}
	return m.height - headerHeight - m.currentInputHeight() - footerHeight - separatorHeight - pickerH
}

func (m *model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.Key()

	if key.Mod == tea.ModCtrl && key.Code == 'c' {
		_, _ = m.client.Abort(context.Background(), &pb.AbortRequest{SessionId: m.sessionID})
		m.cancel()
		m.input.SetValue("")
		m.input.SetHeight(inputHeight)
		m.historyIndex = -1
		m.draftInput = ""
		m.vp.SetHeight(m.vpHeight())
		return m, listenForEvent(m.eventCh)
	}

	if m.modal.visible {
		return m.handleModalKey(msg)
	}

	if m.picker.Open {
		return m.handlePickerKey(msg)
	}

	if key.Code == tea.KeyEscape {
		if m.modal.visible {
			m.modal.close()
			return m, listenForEvent(m.eventCh)
		}
		if m.picker.Open {
			m.picker.Close()
			m.vp.SetHeight(m.vpHeight())
			return m, listenForEvent(m.eventCh)
		}
		if m.isRunning || m.isCompacting.Load() {
			_, _ = m.client.Abort(context.Background(), &pb.AbortRequest{SessionId: m.sessionID})
			m.cancel()
			return m, listenForEvent(m.eventCh)
		}
		return m, nil
	}

	if m.isCompacting.Load() {
		return m, listenForEvent(m.eventCh)
	}

	if Matches(msg, K.Up()) {
		if m.input.Line() > 0 {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		if len(m.promptHistory) == 0 {
			return m, nil
		}
		if m.historyIndex == -1 {
			m.draftInput = m.input.Value()
			m.historyIndex = len(m.promptHistory) - 1
		} else if m.historyIndex > 0 {
			m.historyIndex--
		} else {
			return m, nil
		}
		m.input.SetValue(m.promptHistory[m.historyIndex])
		m.input.SetHeight(m.currentInputHeight())
		m.vp.SetHeight(m.vpHeight())
		return m, nil
	}

	if Matches(msg, K.Down()) {
		if m.input.Line() < m.input.LineCount()-1 {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		if m.historyIndex == -1 {
			return m, nil
		}
		if m.historyIndex < len(m.promptHistory)-1 {
			m.historyIndex++
			m.input.SetValue(m.promptHistory[m.historyIndex])
		} else {
			m.historyIndex = -1
			m.input.SetValue(m.draftInput)
		}
		m.input.SetHeight(m.currentInputHeight())
		m.vp.SetHeight(m.vpHeight())
		return m, nil
	}

	if Matches(msg, K.Shift("enter")) {
		m.input.InsertString("\n")
		m.input.SetHeight(m.currentInputHeight())
		m.vp.SetHeight(m.vpHeight())
		return m, nil
	}

	if Matches(msg, K.Ctrl("enter")) {
		if m.input.Value() == "" {
			return m, nil
		}
		raw := m.input.Value()

		if raw != "" && (len(m.promptHistory) == 0 || m.promptHistory[len(m.promptHistory)-1] != raw) {
			m.promptHistory = append(m.promptHistory, raw)
		}
		m.historyIndex = -1
		m.draftInput = ""
		m.input.SetValue("")
		m.input.SetHeight(inputHeight)
		m.userScrolled = false
		m.vp.GotoBottom()
		m.vp.SetHeight(m.vpHeight())

		entry := historyEntry{role: "user", items: []contentItem{{kind: contentItemText, text: raw}}}
		if m.isRunning && len(m.history) > 0 && m.history[len(m.history)-1].role == "assistant" {
			idx := len(m.history) - 1
			m.history = append(m.history[:idx+1], m.history[idx])
			m.history[idx] = entry
		} else {
			m.history = append(m.history, entry)
		}
		if m.isRunning {
			_, _ = m.client.FollowUp(context.Background(), &pb.FollowUpRequest{
				SessionId: m.sessionID,
				Message:   raw,
			})
		} else {
			m.newContext()
			if err := m.promptGRPC(m.ctx, raw); err != nil {
				m.history = append(m.history, historyEntry{role: "error", items: []contentItem{{kind: contentItemText, text: err.Error()}}})
			}
		}
		return m, listenForEvent(m.eventCh)
	}

	if key.Code == tea.KeyEnter && key.Mod == 0 {
		if m.input.Value() == "" {
			return m, nil
		}
		raw := m.input.Value()

		if raw != "" && (len(m.promptHistory) == 0 || m.promptHistory[len(m.promptHistory)-1] != raw) {
			m.promptHistory = append(m.promptHistory, raw)
		}
		m.historyIndex = -1
		m.draftInput = ""
		m.input.SetValue("")
		m.input.SetHeight(inputHeight)
		m.userScrolled = false
		m.vp.GotoBottom()
		m.vp.SetHeight(m.vpHeight())

		if cmd := parseSlashCommand(raw); cmd != nil && knownCommand(cmd.name) {
			isBusy := m.isRunning || m.isCompacting.Load()
			if isBusy && (cmd.name == "new" || cmd.name == "resume" || cmd.name == "import" || cmd.name == "tree" || cmd.name == "fork" || cmd.name == "clone" || cmd.name == "model" || cmd.name == "compact") {
				m.history = append(m.history, historyEntry{role: "warning", items: []contentItem{{kind: contentItemText, text: fmt.Sprintf("Cannot use /%s while agent is busy. Abort first with Esc.", cmd.name)}}})
				return m.refreshViewport(), listenForEvent(m.eventCh)
			}
			result, err := handleSlashCommand(cmd, m.client, &m.sessionID, m.sessionMgr, m.config)
			if err != nil {
				m.history = append(m.history, historyEntry{role: "error", items: []contentItem{{kind: contentItemText, text: err.Error()}}})
			} else if result != nil {
				if result.newSessionID != "" {
					m.sessionID = result.newSessionID
					m.newContext()
				}
				if result.syncHistory {
					m.syncHistoryFromService()
				}
				if len(result.historyEntry.items) > 0 {
					m.history = append(m.history, result.historyEntry)
				}
				if result.modalKind != modalNone {
					switch result.modalKind {
					case modalStats:
						stats := m.getAgentStats()
						m.modal.openStatsModal(stats, m.style)
					case modalConfig:
						anthropicKeyStr := "(no key)"
						if m.config.AnthropicAPIKey != "" {
							anthropicKeyStr = "set"
						}
						m.modal.openConfigModal(m.modelName, m.provider, m.thinking, m.config.Theme, "interactive",
							m.config.OllamaBaseURL, m.config.OpenAIBaseURL, anthropicKeyStr, m.config.LlamaCppBaseURL,
							m.config.Compaction.Enabled, m.config.Compaction.ReserveTokens, m.config.Compaction.KeepRecentTokens, m.style)
					case modalTree:
						if len(result.modalNodes) > 0 {
							m.modal.openTreeModal(result.modalNodes, m.sessionID, m.style)
						} else {
							m.openModal(modalTree)
						}
					default:
						m.openModal(result.modalKind)
					}
				}
				if result.compact {
					m.isCompacting.Store(true)
					go func() {
						_, _ = m.client.Compact(context.Background(), &pb.CompactRequest{SessionId: m.sessionID})
						m.isCompacting.Store(false)
						m.syncHistoryFromService()
					}()
					return m, tea.Batch(m.spinner.Tick, listenForEvent(m.eventCh))
				}
				if result.quit {
					return m, tea.Quit
				}
				if result.expandInput != "" {
					m.input.SetValue(result.expandInput)
					m.input.SetHeight(m.currentInputHeight())
				}
				if result.sendDirectly != "" {
					m.userScrolled = false
					m.vp.GotoBottom()
					m.vp.SetHeight(m.vpHeight())

					entry := historyEntry{role: "user", items: []contentItem{{kind: contentItemText, text: result.sendDirectly}}}
					m.history = append(m.history, entry)

					if m.isRunning {
						_, _ = m.client.Steer(context.Background(), &pb.SteerRequest{
							SessionId: m.sessionID,
							Message:   result.sendDirectly,
						})
					} else {
						m.newContext()
						if promErr := m.promptGRPC(m.ctx, result.sendDirectly); promErr != nil {
							m.history = append(m.history, historyEntry{role: "error", items: []contentItem{{kind: contentItemText, text: promErr.Error()}}})
						}
					}
				}
				// invokeTool: send the tool args as a prompt (best-effort)
				if result.invokeTool != "" {
					m.userScrolled = false
					m.vp.GotoBottom()
					m.vp.SetHeight(m.vpHeight())
					m.newContext()
					if promErr := m.promptGRPC(m.ctx, result.invokeToolArgs); promErr != nil {
						m.history = append(m.history, historyEntry{role: "error", items: []contentItem{{kind: contentItemText, text: fmt.Sprintf("Tool invocation failed: %v", promErr)}}})
					}
					return m, listenForEvent(m.eventCh)
				}
			}
			return m.refreshViewport(), listenForEvent(m.eventCh)
		}

		if strings.HasPrefix(raw, "!") {
			bangResult, sendDirectly := HandleBangCommand(raw)
			if sendDirectly {
				entry := historyEntry{role: "user", items: []contentItem{{kind: contentItemText, text: raw}}}
				m.history = append(m.history, entry)
				m.newContext()
				if err := m.promptGRPC(m.ctx, bangResult.Output); err != nil {
					m.history = append(m.history, historyEntry{role: "error", items: []contentItem{{kind: contentItemText, text: err.Error()}}})
				}
			} else {
				m.input.SetValue(bangResult.Output)
				m.input.SetHeight(m.currentInputHeight())
			}
			return m.refreshViewport(), listenForEvent(m.eventCh)
		}

		entry := historyEntry{role: "user", items: []contentItem{{kind: contentItemText, text: raw}}}
		if m.isRunning && len(m.history) > 0 && m.history[len(m.history)-1].role == "assistant" {
			idx := len(m.history) - 1
			m.history = append(m.history[:idx+1], m.history[idx])
			m.history[idx] = entry
		} else {
			m.history = append(m.history, entry)
		}
		if m.isRunning {
			_, _ = m.client.Steer(context.Background(), &pb.SteerRequest{
				SessionId: m.sessionID,
				Message:   raw,
			})
		} else {
			m.newContext()
			if err := m.promptGRPC(m.ctx, raw); err != nil {
				m.history = append(m.history, historyEntry{role: "error", items: []contentItem{{kind: contentItemText, text: err.Error()}}})
				m.isRunning = false
			}
		}
		return m.refreshViewport(), listenForEvent(m.eventCh)
	}

	if Matches(msg, K.Ctrl("o")) {
		m.toolCallsExpanded = !m.toolCallsExpanded
		m.chatContent = m.buildChatContent()
		m.vp.SetContent(m.chatContent)
		return m, listenForEvent(m.eventCh)
	}

	if Matches(msg, K.Ctrl("p")) && len(m.models) > 0 {
		if m.isRunning {
			m.history = append(m.history, historyEntry{role: "warning", items: []contentItem{{kind: contentItemText, text: "Cannot switch models while agent is running. Abort first with Esc."}}})
			return m.refreshViewport(), listenForEvent(m.eventCh)
		}
		m.modelIndex = (m.modelIndex + 1) % len(m.models)
		next := strings.TrimSpace(m.models[m.modelIndex])
		providerName := m.config.Provider
		modelName := next
		if idx := strings.IndexByte(next, '/'); idx >= 0 {
			providerName = next[:idx]
			modelName = next[idx+1:]
		}
		req := &pb.ConfigureSessionRequest{
			SessionId: m.sessionID,
			Model:     ptr(modelName),
		}
		if providerName != m.config.Provider {
			req.Provider = ptr(providerName)
		}
		if _, err := m.client.ConfigureSession(context.Background(), req); err != nil {
			m.history = append(m.history, historyEntry{role: "error", items: []contentItem{{kind: contentItemText, text: err.Error()}}})
			return m.refreshViewport(), listenForEvent(m.eventCh)
		}
		m.modelName = modelName
		m.provider = providerName
		m.config.Provider = providerName
		m.config.Model = modelName
		notice := fmt.Sprintf("Switched model → %s", next)
		m.history = append(m.history, historyEntry{role: "info", items: []contentItem{{kind: contentItemText, text: notice}}})
		return m.refreshViewport(), listenForEvent(m.eventCh)
	}

	if key.Code == tea.KeyUp || key.Code == tea.KeyDown ||
		key.Code == tea.KeyPgUp || key.Code == tea.KeyPgDown {
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		m.userScrolled = !m.vp.AtBottom()
		return m, cmd
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.input.SetHeight(m.currentInputHeight())
	m = m.updatePicker()
	m.vp.SetHeight(m.vpHeight())
	return m, cmd
}

func (m *model) handlePickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.picker.Update(msg) {
		return m, nil
	}

	key := msg.Key()
	switch key.Code {
	case tea.KeyEscape:
		m.picker.Close()
		m.vp.SetHeight(m.vpHeight())

	case tea.KeyEnter, tea.KeyTab:
		selected, ok := m.picker.Selected()
		if ok {
			switch m.picker.Kind {
			case pickerTypeSlash:
				m.input.SetValue("/" + selected + " ")
				m.picker.Close()
				m.vp.SetHeight(m.vpHeight())
				m = m.updatePicker()
				m.vp.SetHeight(m.vpHeight())
				return m, nil
			case pickerTypeSession:
				parts := strings.Split(selected, " | ")
				id := parts[0]
				m.input.SetValue("/resume " + id)
			case pickerTypeSkill:
				prefix := "skill:"
				if strings.Contains(m.input.Value(), "/skill ") {
					prefix = "skill "
				}
				m.input.SetValue("/" + prefix + selected + " ")
			case pickerTypePrompt:
				prefix := "prompt:"
				if strings.Contains(m.input.Value(), "/prompt ") {
					prefix = "prompt "
				}
				m.input.SetValue("/" + prefix + selected + " ")
			default:
				if _, atIdx, ok := atFragment(m.input.Value()); ok {
					m.input.SetValue(replaceAtFragment(m.input.Value(), atIdx, selected+" "))
				}
			}
		}
		m.picker.Close()
		m.vp.SetHeight(m.vpHeight())

	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m = m.updatePicker()
		m.vp.SetHeight(m.vpHeight())
		return m, cmd
	}
	return m, nil
}

func (m *model) handleModalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.Key()
	switch key.Code {
	case tea.KeyEscape:
		m.modal.close()
		return m, listenForEvent(m.eventCh)

	case tea.KeyUp:
		if m.modal.kind == modalTree && m.modal.cursor > 0 {
			m.modal.cursor--
			if m.modal.cursor < m.modal.offset {
				m.modal.offset = m.modal.cursor
			}
			m.modal.refreshTreeContent(m.sessionID, m.style)
		}
		return m, nil

	case tea.KeyDown:
		if m.modal.kind == modalTree && m.modal.cursor < len(m.modal.nodes)-1 {
			m.modal.cursor++
			if m.modal.cursor >= m.modal.offset+treePageSize {
				m.modal.offset = m.modal.cursor - treePageSize + 1
			}
			m.modal.refreshTreeContent(m.sessionID, m.style)
		}
		return m, nil

	case tea.KeyEnter:
		if m.modal.kind == modalTree && len(m.modal.nodes) > 0 {
			selected := m.modal.nodes[m.modal.cursor].Node.ID
			m.modal.close()
			m.sessionID = selected
			m.newContext()
			m.syncHistoryFromService()
			m.history = append(m.history, historyEntry{role: "info", items: []contentItem{{kind: contentItemText, text: fmt.Sprintf("Switched to session: %s", selected)}}})
			return m.refreshViewport(), listenForEvent(m.eventCh)
		}

	default:
		key := msg.Key()
		if m.modal.kind == modalTree && key.Code == 'b' && len(m.modal.nodes) > 0 {
			selected := m.modal.nodes[m.modal.cursor].Node.ID
			m.modal.close()

			resp, err := m.client.ForkSession(context.Background(), &pb.ForkSessionRequest{SessionId: selected})
			if err != nil {
				m.history = append(m.history, historyEntry{role: "error", items: []contentItem{{kind: contentItemText, text: fmt.Sprintf("Failed to fork session: %v", err)}}})
				return m.refreshViewport(), listenForEvent(m.eventCh)
			}

			m.sessionID = resp.SessionId
			m.newContext()
			m.syncHistoryFromService()
			m.history = append(m.history, historyEntry{role: "info", items: []contentItem{{kind: contentItemText, text: fmt.Sprintf("Branched from session: %s", selected)}}})
			m.history = append(m.history, historyEntry{role: "info", items: []contentItem{{kind: contentItemText, text: fmt.Sprintf("New session created: %s", resp.SessionId)}}})
			return m.refreshViewport(), listenForEvent(m.eventCh)
		}
	}
	return m, nil
}

func (m *model) updatePicker() *model {
	val := m.input.Value()

	if strings.HasPrefix(val, "/resume ") {
		query := val[len("/resume "):]
		summaries, _ := m.sessionMgr.ListSummaries()
		var items []string
		for _, s := range summaries {
			firstMsg := s.FirstMessage
			if len(firstMsg) > 40 {
				firstMsg = firstMsg[:37] + "..."
			}
			firstMsg = strings.ReplaceAll(firstMsg, "\n", " ")
			items = append(items, fmt.Sprintf("%s | %-40s | C: %s | U: %s",
				s.ID, firstMsg, s.CreatedAt.Format("Jan 02 15:04"), s.UpdatedAt.Format("Jan 02 15:04")))
		}
		m.picker.Reset(pickerTypeSession, query, items)
		return m
	}

	if strings.HasPrefix(val, "/skill:") || strings.HasPrefix(val, "/skill ") {
		prefix := "/skill:"
		if strings.HasPrefix(val, "/skill ") {
			prefix = "/skill "
		}
		query := val[len(prefix):]
		found, _ := skills.Discover(m.config.SkillPaths...)
		var names []string
		for _, s := range found {
			names = append(names, s.Name)
		}
		m.picker.Reset(pickerTypeSkill, query, names)
		return m
	}

	if strings.HasPrefix(val, "/prompt:") || strings.HasPrefix(val, "/prompt ") {
		prefix := "/prompt:"
		if strings.HasPrefix(val, "/prompt ") {
			prefix = "/prompt "
		}
		query := val[len(prefix):]
		found, _ := prompts.Discover(m.config.PromptTemplatePaths...)
		var names []string
		for _, p := range found {
			names = append(names, strings.TrimSuffix(filepath.Base(p.Path), ".md"))
		}
		m.picker.Reset(pickerTypePrompt, query, names)
		return m
	}

	if strings.HasPrefix(val, "/") && !strings.ContainsRune(val, ' ') {
		query := val[1:]
		var cmds []string
		cmds = append(cmds, BaseSlashCommands...)

		skillDirs := append(skills.DefaultDirs(), m.config.SkillPaths...)
		foundSkills, _ := skills.Discover(skillDirs...)
		for _, s := range foundSkills {
			cmds = append(cmds, "skill:"+s.Name)
		}

		promptDirs := append(prompts.DefaultDirs(), m.config.PromptTemplatePaths...)
		foundPrompts, _ := prompts.Discover(promptDirs...)
		for _, p := range foundPrompts {
			name := strings.TrimSuffix(filepath.Base(p.Path), ".md")
			cmds = append(cmds, "prompt:"+name)
		}

		sort.Strings(cmds)
		m.picker.Reset(pickerTypeSlash, query, cmds)
		return m
	}

	query, _, ok := atFragment(val)
	if !ok {
		m.picker.Close()
		return m
	}

	if m.fileCache == nil {
		m.fileCache = discoverFiles(".")
	}

	m.picker.Reset(pickerTypeFile, query, m.fileCache)
	return m
}

func (m *model) openModal(kind modalKind) {
	m.modal.kind = kind
	m.modal.visible = true
}

