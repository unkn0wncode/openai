package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/unkn0wncode/openai"
	"github.com/unkn0wncode/openai/content/output"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/tools"
)

// This example shows how to handle function calls manually: we set ReturnToolCalls=true
// so the SDK returns the calls instead of executing them automatically.
func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(token)

	// Register a function WITHOUT implementation (F is nil)
	echoFunc := tools.FunctionCall{
		Name:         "echo",
		Description:  "Echo the provided text back",
		ParamsSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
		// F is nil: we will handle this manually.
	}
	if err := client.Tools().CreateFunction(echoFunc); err != nil {
		panic(err)
	}

	// Ask the model to use the function
	req := responses.Request{
		Model:           models.DefaultNano,
		Input:           "Please echo back the word 'hello' using the echo function.",
		Tools:           []string{"echo"},
		ReturnToolCalls: true, // important: do not auto-execute
		User:            "manual-example",
	}

	resp, err := client.Responses.Send(&req)
	if err != nil {
		panic(err)
	}

	// Expect a function call in the outputs
	calls := resp.FunctionCalls()
	if len(calls) == 0 {
		panic("expected at least one function call")
	}
	call := calls[0]

	// Parse the arguments provided by the model
	var args struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		panic(fmt.Sprintf("failed to parse function args: %v", err))
	}
	fmt.Printf("Received function call: %s(%q)\n", call.Name, args.Text)

	// Provide the tool output using the parsed argument
	toolOutput := output.FunctionCallOutput{
		Type:   "function_call_output",
		CallID: call.CallID,
		Output: fmt.Sprintf(`{"text":%q}`, args.Text),
	}

	followResp, err := client.Responses.Send(&responses.Request{
		Input:              []output.FunctionCallOutput{toolOutput},
		PreviousResponseID: resp.ID,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println("Assistant said:", followResp.JoinedTexts())
}
