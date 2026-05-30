package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"picoclaw/pkg/cron"
	"picoclaw/pkg/llm"
)

// cronTool lets the agent schedule prompts to run later via the shared
// scheduler. It is only registered when a scheduler is available.
type cronTool struct{ sched *cron.Scheduler }

func (t *cronTool) Schema() llm.ToolSchema {
	return llm.ToolSchema{
		Name: "cron",
		Description: "Schedule, list, or remove timed prompts. " +
			"action=add needs schedule (\"in 10m\", \"every 1h\", or RFC3339) and prompt; " +
			"action=list returns jobs; action=remove needs id.",
		Parameters: objectSchema(map[string]any{
			"action":   strProp("add | list | remove"),
			"schedule": strProp(`For add: "in <dur>", "every <dur>", or RFC3339.`),
			"prompt":   strProp("For add: the prompt to run when it fires."),
			"id":       strProp("For remove: the job id."),
		}, "action"),
	}
}

func (t *cronTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var a struct {
		Action   string `json:"action"`
		Schedule string `json:"schedule"`
		Prompt   string `json:"prompt"`
		ID       string `json:"id"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return "", err
	}
	switch a.Action {
	case "add":
		if a.Schedule == "" || a.Prompt == "" {
			return "", fmt.Errorf("add requires schedule and prompt")
		}
		j, err := t.sched.Add(a.Schedule, a.Prompt)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("scheduled %s (%s) next at %s", j.ID, j.Kind, j.NextRun.Format("2006-01-02 15:04:05")), nil
	case "list":
		jobs := t.sched.List()
		if len(jobs) == 0 {
			return "no scheduled jobs", nil
		}
		var b strings.Builder
		for _, j := range jobs {
			fmt.Fprintf(&b, "%s [%s] %q next=%s enabled=%v\n", j.ID, j.Kind, j.Prompt, j.NextRun.Format("15:04:05"), j.Enabled)
		}
		return b.String(), nil
	case "remove":
		if a.ID == "" {
			return "", fmt.Errorf("remove requires id")
		}
		if !t.sched.Remove(a.ID) {
			return "", fmt.Errorf("no such job %q", a.ID)
		}
		return "removed " + a.ID, nil
	default:
		return "", fmt.Errorf("unknown action %q", a.Action)
	}
}
