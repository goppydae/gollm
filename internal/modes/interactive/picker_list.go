package interactive

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

// pickerType identifies the type of data being picked.
type pickerType int

const (
	pickerTypeFile pickerType = iota
	pickerTypeSlash
	pickerTypeSession
	pickerTypeSkill
	pickerTypePrompt
)

// pickerItem implements list.Item.
type pickerItem struct {
	kind        pickerType
	title       string
	description string
	value       string // actual value to insert
}

func (i pickerItem) Title() string       { return i.title }
func (i pickerItem) Description() string { return i.description }
func (i pickerItem) FilterValue() string { return i.value }

// pickerDelegate handles rendering for picker items.
type pickerDelegate struct { //nolint:unused
	style Style
}

// pickerPageSize is the number of items shown per page in the file picker.
const pickerPageSize = 10

func (d pickerDelegate) Height() int                               { return 1 }   //nolint:unused
func (d pickerDelegate) Spacing() int                              { return 0 }   //nolint:unused
func (d pickerDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil } //nolint:unused
func (d pickerDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) { //nolint:unused
	i, ok := listItem.(pickerItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("  %s", i.title)
	if i.description != "" {
		str += fmt.Sprintf(" %s", i.description)
	}

	fn := d.style.Muted().Render
	if index == m.Index() {
		fn = func(strs ...string) string {
			s := strings.Join(strs, " ")
			return d.style.StatusWorking().Render("> " + s[2:])
		}
	}

	_, _ = fmt.Fprint(w, fn(str))
}

// newPickerList creates a new list.Model configured for use as a picker.
func newPickerList(style Style) list.Model {
	items := []list.Item{}
	l := list.New(items, pickerDelegate{style: style}, 0, 0)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.KeyMap.Quit.Unbind() // Don't quit the whole app on 'q' in picker

	return l
}

// discoverFiles walks root and returns relative paths for both files and
// directories (directories have a trailing /), skipping hidden dirs and known
// large/binary directories.
func discoverFiles(root string) []string {
	skipDirs := map[string]bool{
		".git": true, "node_modules": true, "vendor": true,
		".cache": true, "dist": true, "build": true,
	}
	var entries []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path == root {
				return nil // skip the root entry itself
			}
			if strings.HasPrefix(d.Name(), ".") || skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			entries = append(entries, rel+"/")
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		entries = append(entries, rel)
		return nil
	})
	return entries
}

// atFragment finds an @-prefixed word at the end of the current line in val.
// Returns the fragment after @, the index of @ in val, and whether one was found.
func atFragment(val string) (query string, atIdx int, ok bool) {
	lastNL := strings.LastIndexByte(val, '\n')
	line := val[lastNL+1:]
	at := strings.LastIndexByte(line, '@')
	if at < 0 {
		return "", -1, false
	}
	fragment := line[at+1:]
	if strings.ContainsAny(fragment, " \t") {
		return "", -1, false
	}
	return fragment, lastNL + 1 + at, true
}

// replaceAtFragment inserts replacement after the @ at atIdx, keeping the @.
func replaceAtFragment(val string, atIdx int, replacement string) string {
	return val[:atIdx+1] + replacement
}
