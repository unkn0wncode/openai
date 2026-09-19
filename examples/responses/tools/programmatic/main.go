// Package main demonstrates automatic programmatic tool calling without stored responses.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/unkn0wncode/openai"
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
	if err := client.Tools().RegisterTool(tools.Tool{Type: "programmatic_tool_calling"}); err != nil {
		panic(err)
	}

	// This example uses local inventory data; the model can access it through the tool.
	inventory := map[string]int{"notebook": 12, "pencil": 30}
	if err := client.Tools().CreateFunction(tools.FunctionCall{
		Name:           "get_stock",
		Description:    "Return available units of one product from the example inventory.",
		ParamsSchema:   json.RawMessage(`{"type":"object","properties":{"product":{"type":"string","enum":["notebook","pencil"]}},"required":["product"],"additionalProperties":false}`),
		OutputSchema:   json.RawMessage(`{"type":"object","properties":{"available":{"type":"integer"}},"required":["available"],"additionalProperties":false}`),
		AllowedCallers: []string{"programmatic"},
		Strict:         true,
		F: func(params json.RawMessage) (string, error) {
			var args struct {
				Product string `json:"product"`
			}
			if err := json.Unmarshal(params, &args); err != nil {
				return "", err
			}
			available, ok := inventory[args.Product]
			if !ok {
				return "", fmt.Errorf("unknown product %q", args.Product)
			}
			return fmt.Sprintf(`{"available":%d}`, available), nil
		},
	}); err != nil {
		panic(err)
	}

	store := false
	resp, err := client.Responses.Send(&responses.Request{
		Model:           models.Default,
		Input:           "Use one program to fetch stock for notebook and pencil in parallel, add the available counts, and report the total in one sentence.",
		Tools:           []string{"programmatic_tool_calling", "get_stock"},
		Store:           &store,
		Reasoning:       &responses.ReasoningConfig{Effort: "low"},
		MaxOutputTokens: 2048,
	})
	if err != nil {
		panic(err)
	}
	// Send executes the registered function and replays program state for continuation.
	fmt.Println(resp.JoinedTexts())
}
