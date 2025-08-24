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

	regexFormat := &tools.CustomToolFormat{
		Type:       "grammar",
		Syntax:     "regex",
		Definition: "(?s)^SELECT\\s+.+\\s+FROM\\s+\\w+(\\s+WHERE\\s+.+)?;?$",
	}

	if err := client.Config().Tools.RegisterTool(tools.Tool{
		Type:        "custom",
		Name:        "sql_runner",
		Description: "Executes a constrained SQL SELECT statement.",
		Format:      regexFormat,
		Custom: func(input string) (string, error) {
			fmt.Println("Called tool with input:", input)
			return fmt.Sprintf("ran: %s", input), nil
		},
	}); err != nil {
		panic(err)
	}

	req := responses.Request{
		Model: models.GPT5Mini,
		Input: "Use sql_runner to select user_name from users where id=1;",
		Tools: []string{"sql_runner"},
	}

	resp, err := client.Responses.Send(&req)
	if err != nil {
		panic(err)
	}
	fmt.Println("Response ID:", resp.ID)
	fmt.Println(resp.JoinedTexts())
}
