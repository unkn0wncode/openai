package main

import (
    "fmt"
    "os"

    "github.com/unkn0wncode/openai"
    "github.com/unkn0wncode/openai/models"
    "github.com/unkn0wncode/openai/responses"
    "github.com/unkn0wncode/openai/responses/streaming"
)

func main() {
    token := os.Getenv("OPENAI_API_KEY")
    if token == "" {
        panic("OPENAI_API_KEY not set")
    }

    client := openai.NewClient(token)

    req := &responses.Request{
        Model:  models.DefaultNano,
        Input:  "Write a 1000-character long Lorem Ipsum text.",
        Stream: true,
    }

    stream, err := client.Responses.Stream(req)
    if err != nil {
        panic(err)
    }

    for event := range stream {
        switch e := event.(type) {
        case streaming.ResponseOutputTextDelta:
            fmt.Print(e.Delta)
        case error:
            panic(e)
        }
    }
} 