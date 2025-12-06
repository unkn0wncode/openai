package main

import (
	"fmt"
	"os"

	"github.com/unkn0wncode/openai"
	"github.com/unkn0wncode/openai/content/input"
	"github.com/unkn0wncode/openai/content/output"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
)

func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(token)
	conv, err := client.Responses.CreateConversation(
		map[string]string{"topic": "examples"},
		output.Message{
			Role: "user",
			Content: []any{
				input.InputText{Text: "Remember that my favorite language is Go."},
			},
		},
	)
	if err != nil {
		panic(err)
	}

	appendResult, err := conv.AppendItems(&responses.ConversationItemsInclude{
		MessageOutputTextLogprobs: true,
	},
		output.Message{Role: "user", Content: "What did I ask you to remember?"},
	)
	if err != nil {
		panic(err)
	}
	if len(appendResult.ParsedData) == 0 {
		if err := appendResult.Parse(); err != nil {
			panic(err)
		}
	}

	resp, err := client.Responses.Send(&responses.Request{
		Model:        models.Default,
		Input:        "Greet me and mention my favorite language.",
		Conversation: conv.ID,
		Reasoning: &responses.ReasoningConfig{
			Effort: "none",
		},
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.JoinedTexts())

	if err := conv.Delete(); err != nil {
		panic(err)
	}
}
