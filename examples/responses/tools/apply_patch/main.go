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

	if err := client.Tools().RegisterTool(tools.Tool{Type: "apply_patch"}); err != nil {
		panic(err)
	}

	req := responses.Request{
		Model:      models.Default,
		Input:      "Use the apply_patch tool to create docs/RELEASE_NOTES.md containing exactly:\n# Release Notes\n\n- Introduced responses apply_patch example.",
		Tools:      []string{"apply_patch"},
		ToolChoice: responses.ForceToolChoice("apply_patch", ""),
		Reasoning: &responses.ReasoningConfig{
			Effort: "none",
		},
		User: "responses-apply-patch-example",
	}

	resp, err := client.Responses.Send(&req)
	if err != nil {
		panic(err)
	}

	calls := resp.ApplyPatchCalls()
	if len(calls) != 1 {
		panic(fmt.Sprintf("expected one apply_patch call, got %d", len(calls)))
	}

	call := calls[0]
	if call.Operation.Type != "create_file" {
		panic(fmt.Sprintf("unexpected operation type: %s", call.Operation.Type))
	}
	if call.Operation.Path != "docs/RELEASE_NOTES.md" {
		panic(fmt.Sprintf("unexpected operation path: %s", call.Operation.Path))
	}
	if !strings.Contains(call.Operation.Diff, "- Introduced responses apply_patch example.") {
		panic(fmt.Sprintf("diff does not contain expected content: %q", call.Operation.Diff))
	}

	fmt.Printf("Model requested patch:\n%s\n", call.Operation.Diff)

	toolOutput := output.ApplyPatchCallOutput{
		CallID: call.CallID,
		Status: "completed",
		Output: "File created successfully.",
	}

	followResp, err := client.Responses.Send(&responses.Request{
		Model:              models.Default,
		PreviousResponseID: resp.ID,
		Input:              []output.ApplyPatchCallOutput{toolOutput},
		Reasoning: &responses.ReasoningConfig{
			Effort: "none",
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Assistant response:", followResp.JoinedTexts())
}
