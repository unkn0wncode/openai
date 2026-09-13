// Package main demonstrates cache diagnostics and reasoning updates between turns.
package main

import (
	"fmt"
	"os"
	"strings"

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

	// Use a long, stable document as the reusable prefix in a real application.
	policy := strings.Repeat("Support policy: unused items can be returned within 30 days with a receipt. Shipping fees are nonrefundable.\n", 80)
	req := responses.Request{
		Model: models.GPT6Astra, // configuration_update requires GPT-6 Astra
		Input: []output.Message{
			{Role: "developer", Content: []input.InputText{{Text: policy, PromptCacheBreakpoint: &input.PromptCacheBreakpoint{}}}},
			{Role: "user", Content: "Summarize the return policy in one sentence."},
		},
		PromptCacheOptions: &responses.PromptCacheOptions{Mode: "explicit", TTL: "30m"},
		Reasoning:          &responses.ReasoningConfig{Mode: "standard", Effort: "low"},
		MaxOutputTokens:    1024,
	}
	first, err := client.Responses.Send(&req)
	if err != nil {
		panic(err)
	}
	fmt.Println(first.JoinedTexts())

	// Keep request-level effort unchanged so the original prefix remains reusable.
	req.PreviousResponseID = first.ID
	req.PromptCacheOptions.ComparisonResponseID = first.ID
	req.Input = []any{
		input.ConfigurationUpdate{Reasoning: &input.ReasoningUpdate{Effort: "medium"}},
		output.Message{Role: "user", Content: "Does the policy allow refunding shipping on an unused item returned after 10 days? Answer briefly."},
	}
	second, err := client.Responses.Send(&req)
	if err != nil {
		panic(err)
	}
	fmt.Println(second.JoinedTexts())
	if diagnostics := second.PromptCacheDiagnostics; diagnostics != nil {
		fmt.Printf("Cache comparison: %s (%s)\n", diagnostics.Type, diagnostics.Reason)
	}
	// Diagnostics explain reuse; reported usage provides the billing counters.
	if second.Usage != nil {
		fmt.Printf("Cached tokens: %d; cache writes: %d\n", second.Usage.InputTokensDetails.CachedTokens, second.Usage.InputTokensDetails.CacheWriteTokens)
	}
}
