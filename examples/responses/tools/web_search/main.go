package main

import (
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

	if err := client.Tools().RegisterTool(tools.Tool{Type: "web_search"}); err != nil {
		panic(err)
	}

	req := responses.Request{
		Model: models.DefaultMini,
		Input: "What's the newest version of Golang? Use web_search tool to check.",
		Tools: []string{"web_search"}, // GPT-5 cannot force tool choice for web_search
		User:  "example-user",
	}

	resp, err := client.Responses.Send(&req)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.JoinedTexts())
}
