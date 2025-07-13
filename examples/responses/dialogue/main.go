package main

import (
    "fmt"
    "os"

    "github.com/unkn0wncode/openai"
    "github.com/unkn0wncode/openai/content/output"
    "github.com/unkn0wncode/openai/responses"
    "github.com/unkn0wncode/openai/roles"
)

func main() {
    token := os.Getenv("OPENAI_API_KEY")
    if token == "" {
        panic("OPENAI_API_KEY not set")
    }

    client := openai.NewClient(token)

    req := &responses.Request{
        Input: []any{
            output.Message{Role: roles.User, Content: "hi"},
            output.Message{Role: roles.AI, Content: "hello, how are you?"},
            output.Message{Role: roles.User, Content: "I'm fine, and you?"},
        },
    }

    resp, err := client.Responses.Send(req)
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.JoinedTexts())
} 