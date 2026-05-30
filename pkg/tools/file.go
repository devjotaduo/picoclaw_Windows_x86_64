package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"picoclaw/pkg/config"
	"picoclaw/pkg/llm"
)

// maxFileBytes caps how much a single read returns to the model.
const maxFileBytes = 200 * 1024

func objectSchema(props map[string]any, required ...string) map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": props,
		"required":   required,
	}
}

func strProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func intProp(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

// ---- read_file ----

type readFile struct{ sandbox config.Sandbox }

func (t *readFile) Schema() llm.ToolSchema {
	return llm.ToolSchema{
		Name:        "read_file",
		Description: "Read the entire contents of a text file.",
		Parameters:  objectSchema(map[string]any{"path": strProp("File path to read.")}, "path"),
	}
}

func (t *readFile) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return "", err
	}
	if !t.sandbox.AllowRead(a.Path) {
		return "", fmt.Errorf("path not allowed for reading: %s", a.Path)
	}
	data, err := os.ReadFile(a.Path)
	if err != nil {
		return "", err
	}
	if len(data) > maxFileBytes {
		return string(data[:maxFileBytes]) + "\n... [truncated]", nil
	}
	return string(data), nil
}

// ---- read_file_lines ----

type readFileLines struct{ sandbox config.Sandbox }

func (t *readFileLines) Schema() llm.ToolSchema {
	return llm.ToolSchema{
		Name:        "read_file_lines",
		Description: "Read a 1-based inclusive line range from a text file.",
		Parameters: objectSchema(map[string]any{
			"path":  strProp("File path to read."),
			"start": intProp("First line (1-based)."),
			"end":   intProp("Last line (inclusive)."),
		}, "path", "start", "end"),
	}
}

func (t *readFileLines) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var a struct {
		Path  string `json:"path"`
		Start int    `json:"start"`
		End   int    `json:"end"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return "", err
	}
	if !t.sandbox.AllowRead(a.Path) {
		return "", fmt.Errorf("path not allowed for reading: %s", a.Path)
	}
	data, err := os.ReadFile(a.Path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if a.Start < 1 {
		a.Start = 1
	}
	if a.End > len(lines) {
		a.End = len(lines)
	}
	if a.Start > a.End {
		return "", fmt.Errorf("start (%d) after end (%d)", a.Start, a.End)
	}
	var b strings.Builder
	for i := a.Start; i <= a.End; i++ {
		fmt.Fprintf(&b, "%d\t%s\n", i, lines[i-1])
	}
	return b.String(), nil
}

// ---- write_file ----

type writeFile struct{ sandbox config.Sandbox }

func (t *writeFile) Schema() llm.ToolSchema {
	return llm.ToolSchema{
		Name:        "write_file",
		Description: "Create or overwrite a file with the given content.",
		Parameters: objectSchema(map[string]any{
			"path":    strProp("File path to write."),
			"content": strProp("Full file content."),
		}, "path", "content"),
	}
}

func (t *writeFile) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var a struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return "", err
	}
	if !t.sandbox.AllowWrite(a.Path) {
		return "", fmt.Errorf("path not allowed for writing: %s", a.Path)
	}
	if dir := filepath.Dir(a.Path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(a.Path, []byte(a.Content), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(a.Content), a.Path), nil
}

// ---- append_file ----

type appendFile struct{ sandbox config.Sandbox }

func (t *appendFile) Schema() llm.ToolSchema {
	return llm.ToolSchema{
		Name:        "append_file",
		Description: "Append content to the end of a file, creating it if needed.",
		Parameters: objectSchema(map[string]any{
			"path":    strProp("File path to append to."),
			"content": strProp("Content to append."),
		}, "path", "content"),
	}
}

func (t *appendFile) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var a struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return "", err
	}
	if !t.sandbox.AllowWrite(a.Path) {
		return "", fmt.Errorf("path not allowed for writing: %s", a.Path)
	}
	f, err := os.OpenFile(a.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(a.Content); err != nil {
		return "", err
	}
	return fmt.Sprintf("appended %d bytes to %s", len(a.Content), a.Path), nil
}

// ---- list_dir ----

type listDir struct{ sandbox config.Sandbox }

func (t *listDir) Schema() llm.ToolSchema {
	return llm.ToolSchema{
		Name:        "list_dir",
		Description: "List the entries of a directory.",
		Parameters:  objectSchema(map[string]any{"path": strProp("Directory path to list.")}, "path"),
	}
}

func (t *listDir) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return "", err
	}
	if a.Path == "" {
		a.Path = "."
	}
	if !t.sandbox.AllowRead(a.Path) {
		return "", fmt.Errorf("path not allowed for reading: %s", a.Path)
	}
	entries, err := os.ReadDir(a.Path)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range entries {
		suffix := ""
		if e.IsDir() {
			suffix = "/"
		}
		fmt.Fprintf(&b, "%s%s\n", e.Name(), suffix)
	}
	if b.Len() == 0 {
		return "(empty)", nil
	}
	return b.String(), nil
}
