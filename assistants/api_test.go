//go:build experiment
package assistants

import (
	"context"
	"fmt"
	"log/slog"
	"macbot/framework"
	"testing"
	"time"
)

var (
	testAstID    = "asst_unPsciAmjPUfeMgYtn3fwFwG"
	testThreadID = "thread_m636e1iuyO6N5y1n5K4rcsod"
)

func init() {
	framework.LoadConfig("../../config.json")
}

func TestAssistant_RunAndFetch(t *testing.T) {
	// load an assistant
	ast, err := Load(testAstID)
	if err != nil {
		slog.Error(fmt.Sprintf("Error loading assistant: %s", err))
		t.FailNow()
	}
	slog.Info(fmt.Sprintf("Loaded assistant: %+v", ast))

	// load a thread
	tr, err := LoadThread(testThreadID)
	if err != nil {
		slog.Error(fmt.Sprintf("Error loading thread: %s", err))
		t.FailNow()
	}
	slog.Info(fmt.Sprintf("Loaded thread: %+v", tr))

	// run the assistant
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// run, msg, err := tr.RunAndFetch(ctx, RunOptions{AssistantID: ast.ID}, InputMessage{Content: "test"})
	run, msg, err := tr.RunAndFetch(ctx, RunOptions{AssistantID: ast.ID})
	if err != nil {
		slog.Error(fmt.Sprintf("Error running assistant: %s", err))
		t.FailNow()
	}
	slog.Info(fmt.Sprintf("Assistant replied in %ds: %s", run.CompletedAt-run.CreatedAt, msg))

	// fail to see output
	t.FailNow()
}
