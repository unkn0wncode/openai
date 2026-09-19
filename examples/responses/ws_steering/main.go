// Package main demonstrates adding instructions while a WebSocket response is running.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	ws, err := client.Responses.WebSocket(ctx)
	if err != nil {
		panic(err)
	}
	defer ws.Close()

	stream, err := ws.Send(ctx, &responses.Request{
		Model:           models.GPT6Astra, // mid-turn steering requires GPT-6 Astra
		Input:           "Draft a short project plan for building a task-tracking application.",
		Reasoning:       &responses.ReasoningConfig{Effort: "low"},
		MaxOutputTokens: 1024,
	})
	if err != nil {
		panic(err)
	}
	var initialResponseID string
	var successorCompleted bool
	for stream.Next() {
		switch event := stream.Event().(type) {
		case streaming.ResponseCreated:
			if initialResponseID == "" {
				initialResponseID = event.Response.ID
				if err := ws.Steer(ctx, initialResponseID, "Limit the plan to one developer, two weeks, and three bullet points."); err != nil {
					panic(err)
				}
			} else {
				fmt.Println("\nUpdated response:")
			}
		case streaming.ResponseOutputTextDelta:
			fmt.Print(event.Delta)
		case streaming.ResponseSteerAccepted:
			// Acceptance queues input; keep reading for the automatic continuation.
		case streaming.ResponseSteerFailed:
			panic(event.Error.Message)
		case streaming.ResponseIncomplete:
			if event.Response.ID != initialResponseID || event.Response.IncompleteDetails == nil || event.Response.IncompleteDetails.Reason != "steered" {
				panic("response ended before completion")
			}
		case streaming.ResponseFailed:
			panic(fmt.Sprintf("response failed: %+v", event.Response.Error))
		case streaming.ResponseCompleted:
			successorCompleted = event.Response.ID != initialResponseID
		}
	}
	if err := stream.Err(); err != nil {
		panic(err)
	}
	if !successorCompleted {
		panic("connection ended before the steering continuation completed")
	}
	fmt.Println()
}
