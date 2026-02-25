package main

import (
	"context"
	"fmt"
	"os"

	"github.com/unkn0wncode/openai"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/responses/streaming"
)

func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(token)

	ctx := context.Background()

	ws, err := client.Responses.WebSocket(ctx)
	if err != nil {
		panic(err)
	}
	defer ws.Close()

	stream, err := ws.Send(ctx, &responses.Request{
		Model: models.Default,
		Input: "Write a 1000-character long Lorem Ipsum text.",
		Reasoning: &responses.ReasoningConfig{
			Effort: "none",
		},
	})
	if err != nil {
		panic(err)
	}

	for stream.Next() {
		if delta, ok := stream.Event().(streaming.ResponseOutputTextDelta); ok {
			fmt.Print(delta.Delta)
		}
	}

	if err := stream.Err(); err != nil {
		panic(err)
	}
}
