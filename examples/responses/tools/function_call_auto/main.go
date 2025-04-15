package main

import (
    "encoding/json"
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
        panic("OPENAI_API_KEY not set")
    }

    client := openai.NewClient(token)

    fn := tools.FunctionCall{
        Name:         "get_current_weather",
        Description:  "Get the current weather in a given location",
        ParamsSchema: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"},"unit":{"type":"string","enum":["celsius","fahrenheit"]}},"required":["location"]}`),
        F: func(params json.RawMessage) (string, error) {
            return `{"temperature":22,"unit":"celsius","description":"Sunny"}`, nil
        },
    }
    if err := client.Tools().CreateFunction(fn); err != nil {
        panic(err)
    }

    req := responses.Request{
        Model: models.Default,
        Input: "What's the weather like in San Francisco?",
        Tools: []string{"get_current_weather"},
        User:  "example-user",
    }

    resp, err := client.Responses.Send(&req)
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.JoinedTexts())
} 