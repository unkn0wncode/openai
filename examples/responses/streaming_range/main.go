// Package main demonstrates Responses API streaming range usage.
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
		Model: models.Default,
		Input: "Write a 1000-character long Lorem Ipsum text.",
		Reasoning: &responses.ReasoningConfig{
			Effort: "low",
		},
		Stream: true,
	}

	stream, err := client.Responses.Stream(context.Background(), req)
	if err != nil {
		panic(err)
	}

	for event, err := range stream.Seq() {
		if err != nil {
			panic(err)
		}
		if delta, ok := event.(streaming.ResponseOutputTextDelta); ok {
			fmt.Print(delta.Delta)
		}
	}
}
