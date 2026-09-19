// Package main demonstrates hosted multi-agent collaboration over HTTP.
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
		Input: "Compare two backup plans: A copies files nightly to another disk; B sends encrypted snapshots hourly to remote storage. " +
			"Ask one subagent to assess recovery time and another to assess failure risks. Each should give one sentence. Then give a two-sentence recommendation.",
		MultiAgent:      &responses.MultiAgentConfig{Enabled: true, MaxConcurrentSubagents: 2},
		Reasoning:       &responses.ReasoningConfig{Effort: "low"},
		MaxOutputTokens: 4096,
	})
	if err != nil {
		panic(err)
	}

	// JoinedTexts selects the root assistant's final answer. Commentary and subagent
	// messages remain in resp.ParsedOutputs with agent metadata for tracing.
	fmt.Println(resp.JoinedTexts())
}
