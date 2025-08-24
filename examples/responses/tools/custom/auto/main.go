package main

import (
	"fmt"
	"maps"
	"os"
	"strings"

	"github.com/unkn0wncode/openai"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/tools"
)

// Example: automatic custom tool execution using Responses API.
func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("set OPENAI_API_KEY env var")
	}

	client := openai.NewClient(token)

	notes := map[string]string{
		"golang":    "Prefer clear names. Avoid deep nesting. Handle errors early.",
		"responses": "Custom tools can execute actions and return outputs back to the model.",
		"release":   "Run tests, check lints, update README, tag release.",
	}

	if err := client.Config().Tools.RegisterTool(tools.Tool{
		Type:        "custom",
		Name:        "notes_search",
		Description: "Searches project notes by keyword.",
		Custom: func(input string) (string, error) {
			query := strings.ToLower(strings.TrimSpace(input))
			if content, found := notes[query]; found {
				return fmt.Sprintf("Found note '%s': %s", query, content), nil
			}
			return fmt.Sprintf("No note found for '%s'. Available: %v", query, maps.Keys(notes)), nil
		},
	}); err != nil {
		panic(err)
	}

	req := responses.Request{
		Model: models.GPT5Mini,
		Input: "Search for golang tips using the notes_search tool and summarize what you find.",
		Tools: []string{"notes_search"},
	}

	resp, err := client.Responses.Send(&req)
	if err != nil {
		panic(err)
	}

	fmt.Println("Response:", resp.JoinedTexts())
}
