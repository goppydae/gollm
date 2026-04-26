package interactive

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/goppydae/gollm/internal/session"
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/table"
	lipgloss "charm.land/lipgloss/v2"
)

// modalKind identifies the type of modal overlay.
type modalKind int

const (
	modalNone modalKind = iota
	modalStats
	modalConfig
	modalTree
	modalModels
	modalHelp
)


// modalState holds the state of a modal overlay.
type modalState struct {
	kind    modalKind
	title   string
	visible bool
	table   table.Model
	list    list.Model
}

// treeItem implements list.Item for the session tree.
type treeItem struct {
	node session.FlatNode
}

func (i treeItem) Title() string       { return i.node.Node.ID }
func (i treeItem) Description() string { return i.node.Node.FirstMessage }
func (i treeItem) FilterValue() string { return i.node.Node.ID + " " + i.node.Node.Name }

// modelItem implements list.Item for the model selection.
type modelItem struct {
	name     string
	provider string
}

func (i modelItem) Title() string       { return i.name }
func (i modelItem) Description() string { return i.provider }
func (i modelItem) FilterValue() string { return i.name + " " + i.provider }

// newModal creates an idle modal state.
func newModal() modalState {
	return modalState{
		kind:    modalNone,
		visible: false,
	}
}

func (m *modalState) openStatsModal(stats agentStats, style Style) {
	m.kind = modalStats
	m.title = "Session Stats"
	m.visible = true

	columns := []table.Column{
		{Title: "Property", Width: 20},
		{Title: "Value", Width: 40},
	}

	rows := []table.Row{
		{"ID", stats.SessionID},
		{"File", filepath.Base(stats.SessionFile)},
		{"Created", stats.CreatedAt.Format("Jan 02 15:04:05")},
		{"Updated", stats.UpdatedAt.Format("Jan 02 15:04:05")},
		{"Model", stats.Model},
		{"Provider", stats.Provider},
		{"Thinking", stats.Thinking},
		{"---", "---"},
		{"User Msg", strconv.Itoa(stats.UserMessages)},
		{"Assistant Msg", strconv.Itoa(stats.AssistantMsgs)},
		{"Total Msg", strconv.Itoa(stats.TotalMessages)},
		{"---", "---"},
		{"Input Tokens", strconv.Itoa(stats.InputTokens)},
		{"Output Tokens", strconv.Itoa(stats.OutputTokens)},
		{"Total Tokens", strconv.Itoa(stats.TotalTokens)},
		{"Context", fmt.Sprintf("%d / %d", stats.ContextTokens, stats.ContextWindow)},
	}

	if stats.Cost > 0 {
		rows = append(rows, table.Row{"Cost", fmt.Sprintf("$%.4f", stats.Cost)})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Foreground(style.AccentColor()).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(style.AccentColor()).
		Bold(false)
	t.SetStyles(s)

	m.table = t
}

func (m *modalState) openConfigModal(model, provider, thinking, theme, mode string,
	ollamaURL, openaiURL, anthropicKeySet, llamacppURL string,
	compactionEnabled bool, reserveTokens, keepRecentTokens int, style Style) {

	m.kind = modalConfig
	m.title = "Configuration"
	m.visible = true

	columns := []table.Column{
		{Title: "Setting", Width: 20},
		{Title: "Value", Width: 40},
	}

	rows := []table.Row{
		{"Model", model},
		{"Provider", provider},
		{"Thinking", thinking},
		{"Theme", theme},
		{"Mode", mode},
		{"---", "---"},
		{"Ollama URL", ollamaURL},
		{"OpenAI URL", openaiURL},
		{"Anthropic Key", anthropicKeySet},
		{"llama.cpp URL", llamacppURL},
		{"---", "---"},
		{"Compaction", strconv.FormatBool(compactionEnabled)},
		{"Reserve Tokens", strconv.Itoa(reserveTokens)},
		{"Keep Recent", strconv.Itoa(keepRecentTokens)},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Foreground(style.AccentColor()).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(style.AccentColor()).
		Bold(false)
	t.SetStyles(s)

	m.table = t
}

// treeDelegate handles rendering for session tree items.
type treeDelegate struct {
	style     Style
	currentID string
}

func (d treeDelegate) Height() int                               { return 1 } //nolint:unused
func (d treeDelegate) Spacing() int                              { return 0 } //nolint:unused
func (d treeDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil } //nolint:unused
func (d treeDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) { //nolint:unused
	i, ok := listItem.(treeItem)
	if !ok {
		return
	}

	n := i.node
	bg := d.style.PanelBgColor()
	
	// Selection cursor
	cursor := "  "
	cursorStyle := lipgloss.NewStyle().Background(bg).Foreground(d.style.MutedTextColor())
	if index == m.Index() {
		cursor = "› "
		cursorStyle = cursorStyle.Foreground(d.style.AccentColor())
	}

	// Indent and tree structure
	var prefix strings.Builder
	gutterMap := make(map[int]bool)
	for _, g := range n.Gutters {
		if g.Show {
			gutterMap[g.Position] = true
		}
	}
	connectorPos := -1
	if n.ShowConnector {
		connectorPos = n.Indent - 1
	}
	for l := 0; l < n.Indent; l++ {
		if l == connectorPos {
			if n.IsLast {
				prefix.WriteString("└─")
			} else {
				prefix.WriteString("├─")
			}
		} else if gutterMap[l] {
			prefix.WriteString("│ ")
		} else {
			prefix.WriteString("  ")
		}
	}

	activeMarker := " "
	activeStyle := lipgloss.NewStyle().Background(bg)
	if n.Node.ID == d.currentID {
		activeMarker = "•"
		activeStyle = activeStyle.Foreground(d.style.AccentColor())
	}

	labelStyle := lipgloss.NewStyle().Background(bg).Foreground(d.style.MutedTextColor())
	if index == m.Index() {
		labelStyle = labelStyle.Foreground(d.style.AccentColor()).Bold(true)
	}

	// Session Info Columns
	firstMsg := n.Node.FirstMessage
	if len(firstMsg) > 15 {
		firstMsg = firstMsg[:12] + "..."
	}
	firstMsg = strings.ReplaceAll(firstMsg, "\n", " ")

	info := fmt.Sprintf("%-36s │ %-15s │ %s │ %s", n.Node.ID, firstMsg, n.Node.CreatedAt.Format("Jan 02 15:04"), n.Node.UpdatedAt.Format("Jan 02 15:04"))

	cStr := cursorStyle.Render(cursor)
	// Use a fixed width for the tree prefix area to keep columns aligned
	pStr := lipgloss.NewStyle().Background(bg).Foreground(d.style.MutedTextColor()).Width(24).Render(prefix.String())
	aStr := activeStyle.Render(activeMarker)
	lStr := labelStyle.Render(info)

	_, _ = fmt.Fprint(w, aStr+cStr+pStr+" "+lStr)
}

// close hides the modal.
func (m *modalState) close() {
	m.kind = modalNone
	m.visible = false
}

// render draws the modal as a centered box.
func (m *modalState) render(width, height int, style Style, h help.Model, k KeyMap) string {
	if !m.visible {
		return ""
	}

	modalW := width * 9 / 10
	if modalW > 140 { modalW = 140 }
	modalH := height * 8 / 10

	var content string
	switch m.kind {
	case modalStats, modalConfig:
		m.table.SetWidth(modalW - 6)
		content = m.table.View()
	case modalTree, modalModels:
		m.list.SetSize(modalW-6, modalH-6)
		content = m.list.View()
	case modalHelp:
		// help uses the helper bubble
		h.ShowAll = true // Always show full help in modal
		content = h.View(k)
	}

	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(style.AccentColor()).
		Background(style.PanelBgColor()).
		Padding(1, 2)

	innerW := modalW - 6
	titleStyle := lipgloss.NewStyle().
		Foreground(style.PanelBgColor()).
		Background(style.AccentColor()).
		Bold(true).
		Padding(0, 1).
		Width(innerW)

	titleBlock := titleStyle.Render(m.title)
	inner := lipgloss.JoinVertical(lipgloss.Left, titleBlock, "", content)

	return borderStyle.Width(modalW).Render(inner)
}

// openTreeModal builds the session tree overlay content.
func (m *modalState) openTreeModal(nodes []session.FlatNode, currentID string, style Style) {
	m.kind = modalTree
	m.title = "Session Tree"
	m.visible = true

	items := make([]list.Item, len(nodes))
	startIndex := 0
	for i, n := range nodes {
		items[i] = treeItem{node: n}
		if n.Node.ID == currentID {
			startIndex = i
		}
	}

	l := list.New(items, treeDelegate{style: style, currentID: currentID}, 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Select(startIndex)

	m.list = l
}

// modelDelegate handles rendering for model selection items.
type modelDelegate struct {
	style Style
}

func (d modelDelegate) Height() int                               { return 1 }
func (d modelDelegate) Spacing() int                              { return 0 }
func (d modelDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d modelDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(modelItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("  %s (%s)", i.name, i.provider)
	fn := d.style.Muted().Render
	if index == m.Index() {
		fn = d.style.StatusWorking().Bold(true).Render
		str = "> " + str[2:]
	}

	_, _ = fmt.Fprint(w, fn(str))
}

func (m *modalState) openModelsModal(availableModels []string, currentModel string, style Style) {
	m.kind = modalModels
	m.title = "Select Model"
	m.visible = true

	items := make([]list.Item, len(availableModels))
	startIndex := 0
	for i, mstr := range availableModels {
		name := mstr
		provider := "default"
		if idx := strings.IndexByte(mstr, '/'); idx >= 0 {
			provider = mstr[:idx]
			name = mstr[idx+1:]
		}
		items[i] = modelItem{name: name, provider: provider}
		if name == currentModel {
			startIndex = i
		}
	}

	l := list.New(items, modelDelegate{style: style}, 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Select(startIndex)

	m.list = l
}
