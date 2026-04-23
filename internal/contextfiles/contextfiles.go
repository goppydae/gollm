// Package contextfiles discovers and loads AGENTS.md / CLAUDE.md context files
// from the current directory and its parents, stopping at the git root or home dir.
package contextfiles

import (
	"os"
	"path/filepath"
	"strings"
)

// contextFileNames lists the known context filenames to search for.
var contextFileNames = []string{
	"AGENTS.md",
	"CLAUDE.md",
	"GEMINI.md",
	".context.md",
}

// Discover walks from the given root directory upward, collecting any
// context files it finds. Results are ordered from outermost to innermost
// so that more specific (closer) rules win.
func Discover(root string) []string {
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil
	}

	home, _ := os.UserHomeDir()
	var dirs []string

	dir := abs
	for {
		dirs = append([]string{dir}, dirs...) // prepend: outermost first
		parent := filepath.Dir(dir)
		if parent == dir || dir == home {
			break
		}
		// Stop at git root
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			break
		}
		dir = parent
	}

	var found []string
	seen := map[string]bool{}
	for _, d := range dirs {
		for _, name := range contextFileNames {
			p := filepath.Join(d, name)
			if seen[p] {
				continue
			}
			if _, err := os.Stat(p); err == nil {
				found = append(found, p)
				seen[p] = true
			}
		}
	}
	return found
}

// Load reads all discovered context files and concatenates their contents,
// separated by a header comment indicating the source file.
func Load(root string) string {
	files := Discover(root)
	if len(files) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		sb.WriteString("\n\n<!-- context: ")
		sb.WriteString(f)
		sb.WriteString(" -->\n")
		sb.Write(data)
	}
	return strings.TrimSpace(sb.String())
}
