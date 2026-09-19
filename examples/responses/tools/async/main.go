// Package main demonstrates an application-owned async function job and its later result.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/unkn0wncode/openai"
	"github.com/unkn0wncode/openai/content/output"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/tools"
)

func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("OPENAI_API_KEY not set")
	}
	client := openai.NewClient(token)
	if err := client.Tools().CreateFunction(tools.FunctionCall{
		Name:         "count_words",
		Description:  "Count whitespace-separated words in a text.",
		ParamsSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"],"additionalProperties":false}`),
		Async:        true,
		Strict:       true,
	}); err != nil {
		panic(err)
	}

	resp, err := client.Responses.Send(&responses.Request{
		Model:           models.Default,
		Input:           "Call count_words for 'Clear writing uses concrete examples'. Meanwhile, give one short writing tip without guessing the count.",
		Tools:           []string{"count_words"},
		Reasoning:       &responses.ReasoningConfig{Effort: "low"},
		MaxOutputTokens: 1024,
	})
	if err != nil {
		panic(err)
	}
	calls := resp.FunctionCalls()
	if len(calls) != 1 || calls[0].Name != "count_words" || !calls[0].Async {
		panic("expected one async count_words call")
	}
	call := calls[0]
	var args struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
		panic(err)
	}

	// The application owns the job. Async tools do not create a server-side job queue.
	result := make(chan string)
	go func() {
		result <- fmt.Sprintf(`{"words":%d}`, len(strings.Fields(args.Text)))
	}()
	fmt.Println(resp.JoinedTexts())
	latestResponseID := resp.ID
	// Other conversation turns can happen here; retain their latest response ID.
	follow, err := client.Responses.Send(&responses.Request{
		Model:              models.Default,
		PreviousResponseID: latestResponseID,
		Input:              []output.FunctionCallOutput{{CallID: call.CallID, Output: <-result}},
		Reasoning:          &responses.ReasoningConfig{Effort: "low"},
		MaxOutputTokens:    1024,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(follow.JoinedTexts())
}
