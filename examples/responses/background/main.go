package main

import (
    "context"
    "fmt"
    "os"
    "time"

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
        Model:      models.Default,
        Input:      "Tell me a short joke.",
        Background: true,
    })
    if err != nil {
        panic(err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    final, err := client.Responses.Poll(ctx, resp.ID, 2*time.Second)
    if err != nil {
        panic(err)
    }

    fmt.Println(final.JoinedTexts())
} 