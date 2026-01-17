package main

import (
	"fmt"
	"os"

	"github.com/unkn0wncode/openai"
)

func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(token)

	bld := client.Moderation.NewModerationBuilder()
	bld.SetMinConfidence(50).AddText("Hello, world!")

	res, err := bld.Execute()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Flagged: %v\n", res[0].Flagged)
}
