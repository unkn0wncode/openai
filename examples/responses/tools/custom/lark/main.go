package main

import (
	"fmt"
	"os"

	"github.com/unkn0wncode/openai"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/tools"
)

func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("set OPENAI_API_KEY env var")
	}

	client := openai.NewClient(token)

	larkFormat := &tools.CustomToolFormat{
		Type:       "grammar",
		Syntax:     "lark",
		Definition: "start: expr\n?expr: term ((\"+\"|\"-\") term)*\n?term: factor ((\"*\"|\"/\") factor)*\n?factor: NUMBER | \"(\" expr \")\"\n%import common.NUMBER\n%import common.WS\n%ignore WS",
	}

	if err := client.Config().Tools.RegisterTool(tools.Tool{
		Type:        "custom",
		Name:        "calc",
		Description: "Evaluates arithmetic expressions.",
		Format:      larkFormat,
		Custom: func(input string) (string, error) {
			fmt.Println("Tool called with input:", input)
			return fmt.Sprintf("evaluated: %s", input), nil
		},
	}); err != nil {
		panic(err)
	}

	req := responses.Request{
		Model: models.GPT5Mini,
		Input: "Use calc to compute (2+3)*4.",
		Tools: []string{"calc"},
	}

	resp, err := client.Responses.Send(&req)
	if err != nil {
		panic(err)
	}
	fmt.Println("Response ID:", resp.ID)
	fmt.Println(resp.JoinedTexts())
}
