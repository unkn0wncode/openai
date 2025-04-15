package main

import (
	"context"
	"fmt"
	"os"

	"github.com/unkn0wncode/openai"
	"github.com/unkn0wncode/openai/assistants"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/roles"
)

func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(token)

	assistant, err := client.Assistants.CreateAssistant(assistants.CreateParams{
		Name:  "Example Assistant",
		Model: models.DefaultNano,
	})
	if err != nil {
		panic(err)
	}
	defer client.Assistants.DeleteAssistant(assistant.ID())

	thread, err := assistant.NewThread(nil)
	if err != nil {
		panic(err)
	}

	_, err = thread.AddMessage(assistants.InputMessage{Role: roles.User, Content: "Hello, how are you?"})
	if err != nil {
		panic(err)
	}

	_, msg, err := thread.RunAndFetch(context.Background(), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(msg.Content)
}
