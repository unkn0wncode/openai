package main

import (
	"fmt"
	"os"

	"github.com/unkn0wncode/openai"
	"github.com/unkn0wncode/openai/chat"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/roles"
)

func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(token)

	req := chat.Request{
		Model:    models.GPT55, // lock model for chat tests since it's discontinued
		Messages: []chat.Message{{Role: roles.User, Content: "hi"}},
	}

	resp, err := client.Chat.Send(req)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)
}
