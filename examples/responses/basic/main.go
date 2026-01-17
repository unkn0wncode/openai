package main

import (
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

	resp, err := client.Responses.Send(&responses.Request{
		Model: models.Default,
		Input: "hi",
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.JoinedTexts())
}
