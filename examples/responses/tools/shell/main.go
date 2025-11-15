package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/unkn0wncode/openai"
	"github.com/unkn0wncode/openai/content/output"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/tools"
)

func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(token)

	if err := client.Tools().RegisterTool(tools.Tool{Type: "shell"}); err != nil {
		panic(err)
	}

	req := responses.Request{
		Model:      models.Default,
		Input:      "Use the shell tool to list Go example directories under `examples/responses/tools`.",
		Tools:      []string{"shell"},
		ToolChoice: responses.ForceToolChoice("shell", ""),
		Reasoning: &responses.ReasoningConfig{
			Effort: "none",
		},
		User: "responses-shell-example",
	}

	resp, err := client.Responses.Send(&req)
	if err != nil {
		panic(err)
	}

	calls := resp.ShellCalls()
	if len(calls) != 1 {
		panic(fmt.Sprintf("expected one shell call, got %d", len(calls)))
	}

	call := calls[0]
	if len(call.Action.Commands) == 0 {
		panic("shell call has no commands")
	}
	if !strings.Contains(call.Action.Commands[0], "ls") {
		panic(fmt.Sprintf("unexpected shell command: %s", call.Action.Commands[0]))
	}

	fmt.Printf("Model requested command: %s\n", call.Action.Commands[0])

	exitCode := 0
	toolOutput := output.ShellCallOutput{
		CallID: call.CallID,
		Output: []output.ShellCommandResult{
			{
				Stdout:  "apply_patch\nfunction_call_manual\nshell\n",
				Stderr:  "",
				Outcome: output.ShellCallOutcome{Type: "exit", ExitCode: &exitCode},
			},
		},
	}

	followResp, err := client.Responses.Send(&responses.Request{
		Model:              models.Default,
		PreviousResponseID: resp.ID,
		Input:              []output.ShellCallOutput{toolOutput},
		Reasoning: &responses.ReasoningConfig{
			Effort: "none",
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Assistant response:", followResp.JoinedTexts())
}
