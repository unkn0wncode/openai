package main

import (
	"fmt"
	"os"

	"github.com/unkn0wncode/openai"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
)

// Demonstrates conversation continuation via PreviousResponseID.
func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(token)

	// First message – ask the assistant to remember something.
	firstReq := responses.Request{
		Model: models.Default,
		Input: "Remember the number 12345.",
		User:  "chaining-example",
	}

	firstResp, err := client.Responses.Send(&firstReq)
	if err != nil {
		panic(err)
	}
	fmt.Println("Assistant replied:", firstResp.JoinedTexts())

	// Second message – rely on automatic context using PreviousResponseID.
	secondReq := responses.Request{
		Input:              "What number did I ask you to remember?",
		PreviousResponseID: firstResp.ID,
		User:               "chaining-example",
	}

	secondResp, err := client.Responses.Send(&secondReq)
	if err != nil {
		panic(err)
	}

	fmt.Println("Assistant recalled:", secondResp.JoinedTexts())
}
