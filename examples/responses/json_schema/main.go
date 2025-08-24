package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/unkn0wncode/openai"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
)

func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(token)

	schema := json.RawMessage(`{
        "type": "object",
        "properties": {
            "ok": {"type": "boolean"}
        },
        "required": ["ok"],
        "additionalProperties": false
    }`)

	req := responses.Request{
		Model: models.Default,
		Text: &responses.TextOptions{
			Format: responses.TextFormatType{
				Type:   "json_schema",
				Schema: schema,
				Strict: true,
				Name:   "response",
			},
		},
		Input: "send true if you see this correctly",
	}

	resp, err := client.Responses.Send(&req)
	if err != nil {
		panic(err)
	}

	// The assistant should respond with JSON matching the schema.
	text := resp.FirstText()
	fmt.Println("Raw JSON:", text)

	var parsed struct {
		Ok bool `json:"ok"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		panic(fmt.Sprintf("failed to parse JSON: %v", err))
	}

	fmt.Printf("Parsed value: ok=%v\n", parsed.Ok)
}
