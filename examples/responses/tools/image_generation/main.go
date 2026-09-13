// Package main demonstrates the hosted image-generation tool in the Responses API.
package main

import (
	"fmt"
	"os"

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
	if err := client.Tools().RegisterTool(tools.Tool{
		Type:         "image_generation",
		Quality:      "low",
		Size:         "1024x1024",
		OutputFormat: "png",
	}); err != nil {
		panic(err)
	}
	resp, err := client.Responses.Send(&responses.Request{
		Model:           models.Default,
		Input:           "Generate a simple blue circle on a white background, with no text.",
		Tools:           []string{"image_generation"},
		ToolChoice:      responses.ForceToolChoice("image_generation", ""),
		Reasoning:       &responses.ReasoningConfig{Effort: "low"},
		MaxOutputTokens: 1024,
	})
	if err != nil {
		panic(err)
	}
	for _, item := range resp.ParsedOutputs {
		call, ok := item.(output.ImageGenerationCall)
		if !ok {
			continue
		}
		image, err := call.Data()
		if err != nil {
			panic(err)
		}
		// Pass these bytes to your application for display or storage.
		fmt.Printf("Generated %s image: %d bytes\n", call.OutputFormat, len(image))
		return
	}
	panic("response did not contain a generated image")
}
