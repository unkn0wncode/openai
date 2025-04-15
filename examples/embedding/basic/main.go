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

	vec, err := client.Embedding.One("Hello, world!")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Vector length: %d\n", len(vec))
}
