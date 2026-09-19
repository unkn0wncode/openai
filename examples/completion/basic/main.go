// Package main demonstrates a request to the legacy Completions API.
package main

import (
	"fmt"
	"os"

	"github.com/unkn0wncode/openai"
	"github.com/unkn0wncode/openai/completion"
	"github.com/unkn0wncode/openai/models"
)

func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(token)

	req := completion.Request{
		Model:     models.GPT35TurboInstruct,
		Prompt:    "Q: What is 2 + 2? Answer with just the number.\nA:",
		MaxTokens: 128,
		Stop:      []string{"\n"},
	}

	resp, err := client.Completion.Send(req)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)
}
