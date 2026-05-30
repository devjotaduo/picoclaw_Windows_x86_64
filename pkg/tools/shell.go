package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"picoclaw/pkg/llm"
)

// shell runs a command line in the workspace directory. Output is capped and a
// timeout is enforced. This is a minimal Phase-1 sandbox: full exec policies
// and process sessions arrive in a later phase.
type shell struct {
	workspace string
}

const (
	shellTimeout  = 60 * time.Second
	maxShellBytes = 64 * 1024
)

func (t *shell) Schema() llm.ToolSchema {
	return llm.ToolSchema{
		Name:        "shell",
		Description: "Run a shell command in the workspace and return combined stdout/stderr. Times out after 60s.",
		Parameters:  objectSchema(map[string]any{"command": strProp("Command line to execute.")}, "command"),
	}
}

func (t *shell) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var a struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return "", err
	}
	if a.Command == "" {
		return "", fmt.Errorf("empty command")
	}

	ctx, cancel := context.WithTimeout(ctx, shellTimeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", a.Command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", a.Command)
	}
	cmd.Dir = t.workspace

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()

	out := buf.Bytes()
	if len(out) > maxShellBytes {
		out = append(out[:maxShellBytes], []byte("\n... [truncated]")...)
	}
	result := string(out)
	if ctx.Err() == context.DeadlineExceeded {
		return result + "\n[timed out after 60s]", nil
	}
	if err != nil {
		return result + fmt.Sprintf("\n[exit: %v]", err), nil
	}
	return result, nil
}
