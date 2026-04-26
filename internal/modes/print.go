// Package modes provides the four mode implementations.
package modes

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/goppydae/gollm/internal/config"
	pb "github.com/goppydae/gollm/internal/gen/gollm/v1"
	"github.com/goppydae/gollm/internal/modes/interactive"
)

// Handler is the interface for mode implementations.
type Handler interface {
	Run(args []string) error
}

// PrintOptions holds optional startup options for PrintHandler.
type PrintOptions struct {
	NoSession      bool
	PreloadSession string // "continue", "resume", "fork:<path>"
	SystemPrompt   string
	ThinkingLevel  string
	JSON           bool
	DryRun         bool
}

// NewPrintHandler creates a print mode handler.
func NewPrintHandler(client pb.AgentServiceClient, sessionID string, opts PrintOptions) Handler {
	return &PrintHandler{
		Client:    client,
		SessionID: sessionID,
		Options:   opts,
	}
}

// NewInteractiveHandler creates an interactive mode handler.
func NewInteractiveHandler(client pb.AgentServiceClient, sessionID string, cfg *config.Config, theme string, opts interactive.Options) Handler {
	return &InteractiveHandler{
		Client:    client,
		SessionID: sessionID,
		Config:    cfg,
		Theme:     theme,
		Options:   opts,
	}
}

// PrintHandler handles print mode (single-shot output).
type PrintHandler struct {
	Client    pb.AgentServiceClient
	SessionID string
	Options   PrintOptions
}

// InteractiveHandler handles interactive mode (bubbletea TUI).
type InteractiveHandler struct {
	Client    pb.AgentServiceClient
	SessionID string
	Config    *config.Config
	Theme     string
	Options   interactive.Options
}

func (h *InteractiveHandler) Run(args []string) error {
	theme := h.Theme
	if theme == "" {
		theme = "dark"
	}
	return interactive.Run(h.Client, h.SessionID, h.Config, theme, h.Options, args)
}

func (h *PrintHandler) Run(args []string) error {
	ctx := context.Background()
	prompt, fileData, err := buildPrompt(args)
	if err != nil {
		return err
	}
	if prompt == "" && fileData == "" {
		return fmt.Errorf("no prompt provided — pass text or pipe stdin")
	}

	full := mergePrompt(prompt, fileData)

	// Configure session
	_, _ = h.Client.ConfigureSession(ctx, &pb.ConfigureSessionRequest{
		SessionId:     h.SessionID,
		SystemPrompt:  ptr(h.Options.SystemPrompt),
		ThinkingLevel: ptr(h.Options.ThinkingLevel),
		DryRun:        ptr(h.Options.DryRun),
	})

	stream, err := h.Client.Prompt(ctx, &pb.PromptRequest{
		SessionId: h.SessionID,
		Message:   full,
	})
	if err != nil {
		return err
	}

	for {
		ev, err := stream.Recv()
		if err != nil {
			break
		}

		if h.Options.JSON {
			if b, err := protojson.Marshal(ev); err == nil {
				fmt.Println(string(b))
			}
			continue
		}

		switch p := ev.Payload.(type) {
		case *pb.AgentEvent_TextDelta:
			fmt.Print(p.TextDelta.Content)
		case *pb.AgentEvent_ThinkingDelta:
			// suppress thinking in plain-text mode
			_ = p
		case *pb.AgentEvent_ToolCall:
			fmt.Fprintf(os.Stderr, "\n[tool: %s]\n", p.ToolCall.Name)
		case *pb.AgentEvent_ToolOutput:
			if p.ToolOutput.IsError {
				fmt.Fprintf(os.Stderr, "[tool error: %s]\n", p.ToolOutput.Content)
			}
		case *pb.AgentEvent_CompactStart:
			fmt.Fprintf(os.Stderr, "\n[compacting context…]\n")
		case *pb.AgentEvent_Error:
			fmt.Fprintf(os.Stderr, "\nError: %s\n", p.Error.Message)
		}
	}
	fmt.Println()

	return nil
}

func ptr[T any](v T) *T { return &v }

// buildPrompt resolves args into a prompt string and any @file contents.
func buildPrompt(args []string) (prompt, fileData string, err error) {
	var promptParts []string
	var fileParts []string

	if !isTerminal(os.Stdin) {
		scanner := bufio.NewScanner(os.Stdin)
		var sb strings.Builder
		for scanner.Scan() {
			sb.WriteString(scanner.Text())
			sb.WriteByte('\n')
		}
		if scanner.Err() != nil {
			return "", "", fmt.Errorf("read stdin: %w", scanner.Err())
		}
		if text := strings.TrimSpace(sb.String()); text != "" {
			fileParts = append(fileParts, text)
		}
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "@") {
			path := arg[1:]
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return "", "", fmt.Errorf("read %s: %w", path, readErr)
			}
			fileParts = append(fileParts, fmt.Sprintf("--- %s ---\n%s", path, string(data)))
		} else {
			promptParts = append(promptParts, arg)
		}
	}

	return strings.Join(promptParts, " "), strings.Join(fileParts, "\n\n"), nil
}

func mergePrompt(prompt, fileData string) string {
	switch {
	case fileData == "":
		return prompt
	case prompt == "":
		return fileData
	default:
		return fileData + "\n\n" + prompt
	}
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
