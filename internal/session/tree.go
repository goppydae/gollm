package session

import (
	"sort"
	"strings"
	"time"
)

// TreeNode represents a node in the session tree.
type TreeNode struct {
	ID        string
	ParentID  *string
	Name      string
	Model     string
	Provider  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	MsgCount     int
	FirstMessage string
	Children     []*TreeNode
}

// BuildTree loads all sessions and returns the roots of a session tree.
// Orphaned nodes (whose parentID is not found) are treated as roots.
func (m *Manager) BuildTree() ([]*TreeNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids, err := m.store.list()
	if err != nil {
		return nil, err
	}

	byID := make(map[string]*TreeNode, len(ids))
	for _, id := range ids {
		sum, err := m.store.readSummary(id)
		if err != nil {
			continue
		}
		// Normalize ID for mapping
		normID := strings.TrimSpace(strings.ToLower(id))
		byID[normID] = &TreeNode{
			ID:           sum.ID,
			ParentID:     sum.ParentID,
			Name:         sum.Name,
			FirstMessage: sum.FirstMessage,
			CreatedAt:    sum.CreatedAt,
			UpdatedAt:    sum.UpdatedAt,
		}
	}

	// Also build a short-ID map for flexible matching (e.g. if parentId is short but ID is long)
	byShortID := make(map[string]*TreeNode)
	for id, node := range byID {
		if len(id) >= 8 {
			byShortID[id[:8]] = node
		}
	}

	// Link children
	var roots []*TreeNode
	for _, id := range ids {
		normID := strings.TrimSpace(strings.ToLower(id))
		n, ok := byID[normID]
		if !ok {
			continue
		}
		if n.ParentID != nil {
			pid := strings.TrimSpace(strings.ToLower(*n.ParentID))
			parent, ok := byID[pid]
			if !ok && len(pid) >= 8 {
				parent, ok = byShortID[pid[:8]]
			}
			if ok {
				parent.Children = append(parent.Children, n)
				continue
			}
		} else if strings.HasPrefix(n.Name, "Fork of ") {
			// Fallback for older sessions where ParentID wasn't persisted but name contains it
			parts := strings.Fields(n.Name)
			if len(parts) >= 3 {
				pid := strings.TrimSpace(strings.ToLower(parts[2]))
				if parent, ok := byShortID[pid]; ok {
					parent.Children = append(parent.Children, n)
					continue
				}
			}
		}
		roots = append(roots, n)
	}

	// Sort roots and children by UpdatedAt descending
	sortNodes(roots)
	for _, n := range byID {
		sortNodes(n.Children)
	}

	return roots, nil
}

func sortNodes(nodes []*TreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].UpdatedAt.After(nodes[j].UpdatedAt)
	})
}

// GutterInfo tracks vertical branch lines for descendants.
type GutterInfo struct {
	Position int
	Show     bool
}

// FlatNode represents a node in the flattened tree list with layout metadata.
type FlatNode struct {
	Node               *TreeNode
	Indent             int
	ShowConnector      bool
	IsLast             bool
	Gutters []GutterInfo
}

// FlattenTree returns a depth-first flat list of all tree nodes with their depth and prefix.
func FlattenTree(roots []*TreeNode) []FlatNode {
	var result []FlatNode

	multipleRoots := len(roots) > 1

	var walk func(n *TreeNode, indent int, isLast bool, gutters []GutterInfo)
	walk = func(n *TreeNode, indent int, isLast bool, gutters []GutterInfo) {
		result = append(result, FlatNode{
			Node:          n,
			Indent:        indent,
			ShowConnector: indent > 0,
			IsLast:        isLast,
			Gutters:       gutters,
		})

		children := n.Children
		for i, child := range children {
			childIsLast := i == len(children)-1
			
			// Build child gutters: inherit parent gutters and add a vertical line
			// for this level if the child has siblings below it.
			var childGutters []GutterInfo
			if len(gutters) > 0 {
				childGutters = make([]GutterInfo, len(gutters))
				copy(childGutters, gutters)
			}
			
			if !childIsLast {
				// The vertical line should be at the same position as the child's connector.
				pos := indent
				childGutters = append(childGutters, GutterInfo{Position: pos, Show: true})
			}
			
			walk(child, indent+1, childIsLast, childGutters)
		}
	}

	for i, root := range roots {
		indent := 0
		var initialGutters []GutterInfo
		if multipleRoots {
			indent = 1
			// Add a gutter for the virtual root to connect sibling roots
			if i < len(roots)-1 {
				initialGutters = []GutterInfo{{Position: 0, Show: true}}
			}
		}
		walk(root, indent, i == len(roots)-1, initialGutters)
	}

	return result
}


// RenderTree returns a text tree using Unicode box-drawing characters.
func RenderTree(roots []*TreeNode, currentID string) string {
	var sb strings.Builder
	var render func(nodes []*TreeNode, prefix string, last bool)
	render = func(nodes []*TreeNode, prefix string, _ bool) {
		for i, n := range nodes {
			isLast := i == len(nodes)-1
			connector := "├── "
			childPrefix := prefix + "│   "
			if isLast {
				connector = "└── "
				childPrefix = prefix + "    "
			}
			marker := " "
			if n.ID == currentID {
				marker = "▶"
			}
			label := n.ID[:8]
			if n.Name != "" {
				label = n.Name
			}
			sb.WriteString(prefix)
			sb.WriteString(connector)
			sb.WriteString(marker)
			sb.WriteString(" ")
			sb.WriteString(label)
			sb.WriteString("  (")
			sb.WriteString(n.UpdatedAt.Format("Jan 02 15:04"))
			sb.WriteString(", ")
			sb.WriteString(formatMsgCount(n.MsgCount))
			sb.WriteString(")\n")
			if len(n.Children) > 0 {
				render(n.Children, childPrefix, isLast)
			}
		}
	}
	render(roots, "", true)
	return strings.TrimRight(sb.String(), "\n")
}

func formatMsgCount(n int) string {
	if n == 1 {
		return "1 msg"
	}
	s := ""
	for i := n; i > 0; i /= 10 {
		s = string(rune('0'+i%10)) + s
	}
	if n == 0 {
		return "0 msgs"
	}
	return s + " msgs"
}
