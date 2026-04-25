package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// record is a single JSONL line in a session file.
// Each message is written as its own line; the header line (index 0) stores session metadata.
type record struct {
	Kind      string          `json:"kind"`                // "header" | "message"
	ID        string          `json:"id,omitempty"`
	ParentID  *string         `json:"parentId,omitempty"`
	Name      string          `json:"name,omitempty"`
	Model     string          `json:"model,omitempty"`
	Provider  string          `json:"provider,omitempty"`
	Thinking  string          `json:"thinkingLevel,omitempty"`
	System    string          `json:"systemPrompt,omitempty"`
	CreatedAt int64           `json:"createdAt,omitempty"` // unix ms
	UpdatedAt int64           `json:"updatedAt,omitempty"` // unix ms
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	Raw       json.RawMessage `json:"raw,omitempty"` // full message payload
}

// store handles low-level JSONL file I/O for a session directory.
type store struct {
	dir string
}

func newStore(dir string) *store {
	return &store{dir: dir}
}

// path returns the file path for a given session ID.
// It searches the directory for a file ending in _ID.jsonl to handle timestamped filenames.
func (s *store) path(id string) string {
	if filepath.IsAbs(id) {
		return id
	}

	// 1. Check for exact match (backward compatibility)
	p := filepath.Join(s.dir, id+".jsonl")
	if _, err := os.Stat(p); err == nil {
		return p
	}

	// 2. Search for timestamped variant: {Timestamp}_{ID}.jsonl
	entries, err := os.ReadDir(s.dir)
	if err == nil {
		suffix := "_" + id + ".jsonl"
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), suffix) {
				return filepath.Join(s.dir, e.Name())
			}
		}
	}

	return p // fallback to default
}

// writePath serialises a Session to JSONL at the given path.
func (s *store) writePath(sess *Session, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)

	// Header line
	if err := enc.Encode(record{
		Kind:      "header",
		ID:        sess.ID,
		ParentID:  sess.ParentID,
		Name:      sess.Name,
		Model:     sess.Model,
		Provider:  sess.Provider,
		Thinking:  sess.Thinking,
		System:    sess.SystemPrompt,
		CreatedAt: sess.CreatedAt.UnixMilli(),
		UpdatedAt: sess.UpdatedAt.UnixMilli(),
	}); err != nil {
		return err
	}

	// One line per message
	for _, m := range sess.Messages {
		raw, _ := json.Marshal(m)
		if err := enc.Encode(record{
			Kind:    "message",
			Role:    m.Role,
			Content: m.Content,
			Raw:     raw,
		}); err != nil {
			return err
		}
	}
	return nil
}

// write serialises a Session to JSONL using a timestamped filename in s.dir.
func (s *store) write(sess *Session) error {
	path := s.path(sess.ID)

	// If the resolved path doesn't exist yet, it's a new session.
	// Create a new timestamped filename.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Format: 2006-01-02T15-04-05-000Z_UUID.jsonl
		ts := sess.CreatedAt.UTC().Format("2006-01-02T15-04-05-000Z")
		ts = strings.ReplaceAll(ts, ".", "-")
		path = filepath.Join(s.dir, ts+"_"+sess.ID+".jsonl")
	}

	return s.writePath(sess, path)
}

// readPath deserialises a JSONL session file from the given path back into a Session.
func (s *store) readPath(path string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	sess := &Session{}
	scanner := bufio.NewScanner(f)
	first := true
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r record
		if err := json.Unmarshal(line, &r); err != nil {
			if first {
				return nil, fmt.Errorf("corrupt session header: %w", err)
			}
			continue
		}
		if first {
			first = false
			if r.Kind != "header" || r.ID == "" {
				return nil, fmt.Errorf("corrupt session header: unexpected kind %q or empty ID", r.Kind)
			}
			sess.ID = r.ID
			sess.ParentID = r.ParentID
			sess.Name = r.Name
			sess.Model = r.Model
			sess.Provider = r.Provider
			sess.Thinking = r.Thinking
			sess.SystemPrompt = r.System
			sess.CreatedAt = time.UnixMilli(r.CreatedAt)
			sess.UpdatedAt = time.UnixMilli(r.UpdatedAt)
			continue
		}
		if r.Kind == "message" && len(r.Raw) > 0 {
			var msg message
			if err := json.Unmarshal(r.Raw, &msg); err == nil {
				sess.Messages = append(sess.Messages, msg)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return sess, nil
}

// read deserialises a JSONL session file back into a Session by ID.
func (s *store) read(id string) (*Session, error) {
	return s.readPath(s.path(id))
}

// readSummary extracts just the metadata and first message from a session file.
func (s *store) readSummary(id string) (*SessionSummary, error) {
	path := s.path(id)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	sum := &SessionSummary{ID: id}
	scanner := bufio.NewScanner(f)
	
	// Line 1: Header
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty session file")
	}
	var header record
	if err := json.Unmarshal(scanner.Bytes(), &header); err == nil && header.Kind == "header" {
		sum.ID = header.ID
		sum.ParentID = header.ParentID
		sum.Name = header.Name
		if header.CreatedAt > 0 {
			sum.CreatedAt = time.UnixMilli(header.CreatedAt)
		}
		if header.UpdatedAt > 0 {
			sum.UpdatedAt = time.UnixMilli(header.UpdatedAt)
		}
	}

	// Line 2: First Message (optional)
	if scanner.Scan() {
		var msg record
		if err := json.Unmarshal(scanner.Bytes(), &msg); err == nil && msg.Kind == "message" {
			sum.FirstMessage = msg.Content
		}
	}

	return sum, nil
}

// list returns all session IDs in the directory.
// It correctly handles both flat UUID filenames and timestamped {TS}_{UUID}.jsonl filenames.
func (s *store) list() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	type entryInfo struct {
		id    string
		mtime time.Time
	}
	var infos []entryInfo

	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".jsonl" {
			info, err := e.Info()
			if err != nil {
				continue
			}

			// Use the full filename (minus .jsonl) as the ID for internal tracking.
			// This ensures path() can find it via a direct file check.
			name := strings.TrimSuffix(e.Name(), ".jsonl")
			infos = append(infos, entryInfo{
				id:    name,
				mtime: info.ModTime(),
			})
		}
	}

	// Sort by modification time ascending so latest is last
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].mtime.Before(infos[j].mtime)
	})

	var ids []string
	for _, info := range infos {
		ids = append(ids, info.id)
	}
	return ids, nil
}
