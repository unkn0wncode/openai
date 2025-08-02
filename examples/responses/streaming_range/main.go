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

	req := &responses.Request{
		Model:  models.DefaultNano,
		Input:  "Write a 1000-character long Lorem Ipsum text.",
		Stream: true,
	}

	stream, err := client.Responses.Stream(context.Background(), req)
	if err != nil {
		panic(err)
	}

	for event := range stream.Chan() {
		if delta, ok := event.(streaming.ResponseOutputTextDelta); ok {
			fmt.Print(delta.Delta)
		}
	}

	if err := stream.Err(); err != nil {
		panic(err)
	}
}
