package main

import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/unkn0wncode/openai"
    "github.com/unkn0wncode/openai/chat"
    "github.com/unkn0wncode/openai/models"
    "github.com/unkn0wncode/openai/roles"
    "github.com/unkn0wncode/openai/tools"
)

func main() {
    token := os.Getenv("OPENAI_API_KEY")
    if token == "" {
        panic("OPENAI_API_KEY not set")
    }

    client := openai.NewClient(token)

    fn := tools.FunctionCall{
        Name:         "test_function",
        Description:  "Returns true",
        ParamsSchema: tools.EmptyParamsSchema,
        F: func(params json.RawMessage) (string, error) {
            return `{"result":true}`, nil
        },
    }
    if err := client.Tools().CreateFunction(fn); err != nil {
        panic(err)
    }

    req := chat.Request{
        Model:     models.Default,
        Messages:  []chat.Message{{Role: roles.User, Content: "call test function"}},
        Functions: []string{"test_function"},
        ToolChoice: tools.ToolChoiceOption("test_function"),
    }

    resp, err := client.Chat.Send(req)
    if err != nil {
        panic(err)
    }

    fmt.Println(resp)
} 