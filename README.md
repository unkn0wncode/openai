# OpenAI Package

[![Go Reference](https://pkg.go.dev/badge/github.com/unkn0wncode/openai.svg)](https://pkg.go.dev/github.com/unkn0wncode/openai)
[![Go Report Card](https://goreportcard.com/badge/github.com/unkn0wncode/openai)](https://goreportcard.com/report/github.com/unkn0wncode/openai)
[![CI](https://github.com/unkn0wncode/openai/actions/workflows/check-models.yml/badge.svg)](https://github.com/unkn0wncode/openai/actions/workflows/check-models.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Latest Release](https://img.shields.io/github/v/release/unkn0wncode/openai)](https://github.com/unkn0wncode/openai/releases/latest)

`github.com/unkn0wncode/openai` is a Golang package that wraps the functionality of multiple OpenAI APIs.

## The OpenAI Client

All APIs are accessed via the `Client` struct:

```go
package main

import (
	"os"
	"github.com/unkn0wncode/openai"
)

func main() {
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	// Use the client to access the APIs
}
```

A `Client` containing only a token is at its minimal working state and is ready for use.

You can create multiple clients with different configurations and use them independently. Clients are thread-safe.

### Client Config

The `Client` can be further configured by accessing `Client.Config()`:

```go
config := client.Config()
config.HTTPClient.Client.Timeout = 90 * time.Second
config.Log = slog.New(slog.NewJSONHandler(os.Stdout, nil))
```

The following settings are available in `Client.Config()`:
- `BaseAPI` is the base URL for the OpenAI API.
- `Token` is the API key to make requests with.
- `HTTPClient` is the HTTP client used to make API requests. It is a wrapper around `http.Client`.
- `Log` is the logger (based on `log/slog` package).

The `Client.Config().HTTPClient` contains a `LogTripper` that you can enable for debugging:

```go
client.Config().EnableLogTripper()
```

When enabled, the `LogTripper` will log HTTP request and response dumps with the `Debug` level.

`LogTripper` is a wrapper around a standard `http.RoundTripper` in `client.Config().HTTPClient.Client.Transport`. You can freely replace it with your own `http.RoundTripper` implementation losing this logging functionality, and then the `EnableLogTripper()` will return an error if you try to call it.

`HTTPClient` has additional fields with settings:
- `RequestAttempts` is the number of attempts to make the request, default is 3 (one initial attempt and two retries).
- `RetryInterval` is the interval to wait before each retry, default is 3 seconds.
- `AutoLogTripper` is a flag that can be set to true to enable automatic toggling of log tripper on errors/successes, default is false.

If any request fails or returns a non-200 status, it will be retried according to the `HTTPClient` settings.

### Client Tools

Tools, such as functions, can be managed per-client via `Client.Tools()`:

```go
toolRegistry := client.Tools()
toolRegistry.CreateFunction(tools.FunctionCall{...})
toolRegistry.RegisterTool(tools.Tool{...})
```

Functions/tools that you add to the client need to be added to the request by name via the `Functions`/`Tools` field when you want to supply them to the AI:

```go
req := client.Responses.NewRequest()
req.Tools = []string{"function_name"}
```

Mind that the same tool/function can be used across multiple APIs, as long as you use the same `Client` instance.

## Resources shared across APIs

- `openai/models` package contains data of all available models across all APIs. You can still just write any model as a literal string if it's not there. When you don't specify a model in a request, a default model appropriate for the API will be chosen. There's pricing and limits data there that can be used in logging.
- `openai/roles` package contains constants for roles that can be used in messages. Some models may be sensitive to the choice between the older "system" and the newer "developer" roles.
- `openai/tools` package contains types for tools/functions that can be used in requests in multiple APIs. You declare a tool/function, add it to the client, and then list its name in the `Functions`/`Tools` field of a request.
- `openai/content/input` and `openai/content/output` packages contain all types that can be sent to the API or received from it. Some types can be used for both input and output, such are placed in the output package. Note that there are types that are present in both packages and have the same name, but their implementations differ slightly.

# Use of APIs

The APIs are stored in the `Client` under interfaces defined as `<api_name>.Service` and accessible as `Client.<API_name>`. For example, `Client.Chat` or `Client.Moderation`.

Because they are interfaces, you can create wrappers around them, replace them with mocks, or even make your own implementations.

The `<api_name>` packages also contain API-specific types exposed for your use, most importantly the `Request` types. Where applicable, an `<api_name>.Content` interface is defined listing all types that can be used as content in the API.

Currently implemented APIs:
- Responses
- Chat (Legacy)
- Moderation
- Assistants (deprecated)
- Embeddings
- Completions (Legacy)

Not implemented:
- Audio
- Files
- Realtime
- Batch
- Vector Stores
- Fine-tuning
- Evals
- Administration

The most recently introduced API and most recommended for general use is `Client.Responses`.

## Responses API

Here's a minimal example of using the `Responses` API:

```go
resp, _ := client.Responses.Send(&responses.Request{Input: "Hello, world!"})
fmt.Println(resp.JoinedTexts())
```

The `client.Responses` exposes the following methods:
- `Send` sends a given request to the API and returns response data focusing on outputs. It may run a sequence of requests if the response contains tool calls that can be handled automatically (by using tools and sending tool outputs to API) and then will return all outputs at once, except for already handled tool calls.
- `Stream` sends a given request to the API and returns a stream of events. It can be used to read the response as it's being generated. See the Streaming section for details.
- `Poll` polls a background response by ID until completion, failure, or context cancellation.
- `NewRequest` creates a new empty request. It is only a shorthand to make the type `responses.Request` more easily discoverable. You can use the request type directly.
- `NewMessage` creates a new empty message. It is only a shorthand to make the type `output.Message` more easily discoverable. You can use the message type directly.
- `CreateConversation` creates a persistent conversation container where you can manage context items and reuse it in Responses requests.
- `Conversation` fetches a full conversation object by ID. It can be further used to manage the conversation.

Other exposed types/functions in the `responses` package:
- `Content` is an interface listing all types that can be used as content in the Responses API.
- `Request` is the request body. It has a few additional fields:
  - `IntermediateMessageHandler` is a function that can be set to handle `output.Message`s received alongside other outputs, like tool calls, that otherwise are returned in the response but can be handled sooner with this handler.
  - `ReturnToolCalls` is a flag that can be set to not execute tool calls automatically but return them as outputs instead.
- A few more types for request fields.
- `Response` wraps the API response and exposes the following:
  - `Response.ID` field contains the response ID that can be used to chain requests.
  - `Response.<ContentType>()` methods return a slice of outputs of a specific type extracted from the response. For example, `Texts()` returns string of all text outputs, usually just one.
  - `Response.Outputs` contains all received outputs as `[]output.Any`. `Any` contains parsed `type` field and raw data.
  - `Response.ParsedOutputs` contains all received outputs fully parsed in an `[]any` slice. The `.Parse()` method for populating it is called automatically before the response is returned so you don't need to call it.
- `ForceToolChoice` function is a helper that fills the `ToolChoice` field of the request, allowing you to enforce the use of a specific tool.
- `ForceFunction` is a further simplified function for enforcing the use of a specific function tool.

There are a few concepts in the Responses API that may need further explanation:
- Input/output types, such as in `responses.Request.Input` and `output.Message.Content` fields.
- `responses.Request.PreviousResponseID` field that can be filled with `responses.Response.ID` for chaining requests with automatically managed context.
- `responses.Request.Instructions` field that replaces system messages previously used in the Chat API.
- Prompt caching configuration via `responses.Request.PromptCacheKey` and `responses.Request.PromptCacheRetention` (for GPT-5.1+ you can set `"24h"` to enable extended caching).
- Our additional fields in the `responses.Request` type, such as `responses.Request.IntermediateMessageHandler`.

### Inputs

The Responses API can work with nearly everything in the `input` and `output` packages, and that's overwhelming so we'll show a few basic use cases with increasing complexity.

The most basic input is just a string:

```go
&responses.Request{Input: "Hello, world!"}
```

Otherwise, you need to provide a slice of some elements. It can be a slice of same-typed elements, like `[]output.Message`, which can in turn contain a string or a slice of elements in its `Content` field:

```go
&responses.Request{
  Input: []output.Message{
    {Role: roles.User, Content: "hi"},
    {Role: roles.User, Content: []input.InputImage{{ImageURL: "https://example.com/hi.gif"}}},
  },
}
```

You can also send mixed type input elements as an `[]any` slice:

```go
&responses.Request{
  Input: []any{
    output.Message{Role: roles.User, Content: "hi"},
    input.ItemReference{ID: "abc_123"},
  },
}
```

The `output.Message` is also what you receive from the API, so you can reuse it directly from response outputs.

According to the docs, the following types are allowed in `responses.Request.Input`:
- `string`
- a slice of elements of the following types (mixed):
  - `output.Message`, with its `Content` being one of the following:
    - `string`
    - a slice of elements of the following types (mixed):
      - `input.InputText`
      - `input.InputImage`
      - `input.InputFile`
      - `output.OutputText`
      - `output.Refusal`
  - `output.FileSearchCall`
  - `output.ComputerCall`
  - `output.ComputerCallOutput`
  - `output.WebSearchCall`
  - `output.FunctionCall`
  - `output.FunctionCallOutput`
  - `output.CustomToolCall`
  - `output.CustomToolCallOutput`
  - `output.Reasoning`
  - `output.MCPListTools`
  - `output.MCPApprovalRequest`
  - `output.MCPApprovalResponse`
  - `output.MCPCall`
  - `output.LocalShellCall`
  - `output.LocalShellCallOutput`
  - `output.ApplyPatchCall`
  - `output.ApplyPatchCallOutput`
  - `output.ShellCall`
  - `output.ShellCallOutput`
  - `output.CodeInterpreterCall`, with its `Results` being a slice of the following types (mixed):
    - `output.CodeInterpreterResultText`
    - `output.CodeInterpreterResultFile`
  - `input.ItemReference`

This may be overwhelming, but you can keep it simple by sending only strings and messages in most cases.

### Outputs

Possible output types in the `responses.Response.ParsedOutputs` slice are:
- `output.Message`, with its `Content` being a slice of the following types (mixed):
  - `output.OutputText`
  - `output.Refusal`
- `output.FileSearchCall`
- `output.FunctionCall`
- `output.CustomToolCall`
- `output.WebSearchCall`
- `output.ComputerCall`
- `output.Reasoning`
- `output.MCPListTools`
- `output.MCPApprovalRequest`
- `output.MCPApprovalResponse`
- `output.MCPCall`
- `output.LocalShellCall`
- `output.LocalShellCallOutput`
- `output.ApplyPatchCall`
- `output.ApplyPatchCallOutput`
- `output.ShellCall`
- `output.ShellCallOutput`
- `output.CodeInterpreterCall`, with its `Results` being a slice of the following types (mixed):
  - `output.CodeInterpreterResultText`
  - `output.CodeInterpreterResultFile`

You can simply iterate over the outputs and type-assert each, but also there are helper methods to extract outputs of specific types:
- `Response.Texts() []string` returns output texts from output messages.
- `Response.JoinedTexts() string` returns a single string joined from all text outputs with newlines. Since you usually get only one text output, this removes the extra work on a slice of strings you'd have to do with `Texts()`.
- `Response.FirstText() string` returns the first text output in the response, or an empty string if there are no text outputs.
- `Response.LastText() string` returns the last text output in the response, or an empty string if there are no text outputs.
- `Response.FunctionCalls() []output.FunctionCall` returns all function calls from the response's top level.
- `Response.CustomToolCalls() []output.CustomToolCall` returns all custom tool calls from the response's top level.
- `Response.Refusals() []string` returns all refusals texts from output messages.

### Chaining requests with PreviousResponseID

Unlike in the Chat API, where requests are stateless and must contain whole context, the Responses API saves the state of the conversation (unless you set `responses.Request.Store` to `false`) and returns a `responses.Response.ID` that you can use to send only new inputs in the next request. Trimming of older context is done automatically and IDs that you get are usable for 30 days.

It's still possible to use the API in a stateless way by sending all inputs every time and not using the `PreviousResponseID` field.

IDs also allow you to continue a conversation from any previous response:

```
User: is it normal for my pet to meow?
Assistant: Is your pet a cat? (ID_1)
PreviousResponseID=ID_1 User: yes it's a cat
Assistant: It's normal for cats to meow. (ID_2)

PreviousResponseID=ID_1 User: my pet is a dog
Assistant: It's not normal for dogs to meow. (ID_3)
```

### Persistent Conversations

Conversations let you store context items (messages, tool calls, references, etc.) and reuse them across requests instead of chaining with `PreviousResponseID`:

```go
conv, _ := client.Responses.CreateConversation(
  map[string]string{"topic": "cookbook"},
  output.Message{
    Role: "user",
    Content: []any{input.InputText{Text: "Remember that I like pineapples."}},
  },
)

conv.AppendItems(&responses.ConversationItemsInclude{
  MessageOutputTextLogprobs: true,
}, output.Message{Role: "user", Content: "I also like strawberries."})

resp, _ := client.Responses.Send(&responses.Request{
  Model:        models.Default,
  Input:        "Write me a salad recipe.",
  Conversation: conv.ID,
})
fmt.Println(resp.JoinedTexts())
```

A conversation object provides methods for managing the conversation:
- `AppendItems` appends new items to the conversation.
- `ListItems` lists items in the conversation.
- `Item` retrieves a single item from the conversation by ID.
- `DeleteItem` removes a single item from the conversation by ID.
- `Update` updates the metadata of the conversation.
- `Delete` removes the conversation from the API.

### Instructions

Because the conversation context is managed automatically, it is possible for "system" messages to be trimmed out. This is why prompting in Responses API is done via a separate field: `responses.Request.Instructions`. This field is supposed to be supplied with each request and can be easily changed between requests within the same conversation if you want the model to change its behavior.

### Streaming

You can set `responses.Request.Stream` to `true` and use `responses.Response.Stream(req)` to get a stream of `any` events.

In normal flow, you'll get a sequence of events with types from the `responses/streaming` package. If any error occurs during streaming, it will be sent to the same stream, and then the stream will be closed. Only streaming event types and errors can be sent in the stream. Successful termination of the stream is indicated by the stream closing with no error, `io.EOF` is ignored and not sent.

Some event types have fields than may contain multiple different types of data. Such fields are left as `json.RawMessage` and mostly can be parsed further using types from the `output` package, but this is not done automatically.

## Chat API (Legacy)

The Chat API service accessible through `Client.Chat` exposes the following methods:
- `Send` sends a given request to the API and returns response as a string. It can make a sequence of requests if the response contains tool calls that can be handled automatically (by using tools and sending tool outputs to API). Only the last response is returned.
- `NewRequest` creates a new empty request. It is only a shorthand to make the type `chat.Request` more easily discoverable. You can use the request type directly.
- `NewMessage` creates a new empty message. It is only a shorthand to make the type `chat.Message` more easily discoverable. You can use the message type directly.

Because the Chat API does not have such a multitude of types, the `Send` method is simplified down to returning just a string.

Other exposed types/functions in the `chat` package:
- `Request` is the request body. It has an additional field:
  - `ReturnFunctionCalls` is a flag that can be set to not execute tool calls automatically but return them instead. Returned function calls will be encoded in the response string as a JSON array.
- `Message` is the message body, which is used in the `Request.Messages` field.
- A few more types for request and message fields.

The use example:

```go
resp, _ := client.Chat.Send(&chat.Request{
  Messages: []chat.Message{
    {Role: roles.User, Content: "hi"},
  },
})
fmt.Println(resp)
```

## Moderation API

The Moderation API service accessible through `Client.Moderation` provides a method to create a builder for content checks:
- `NewModerationBuilder` creates a new moderation request builder.

The `Builder` provides methods to queue inputs, configure detection thresholds, and execute the request:
- `AddText` adds text for moderation.
- `AddImage` adds an image URL for moderation.
- `SetMinConfidence` sets a confidence threshold in percent.
- `Clear` clears all queued inputs.
- `Execute` sends the request and returns results. New inputs can be added after that to reuse the builder.

The `Result` type includes the following fields and methods:
- `Input` contains the original input content.
- `Flagged` indicates whether the content was flagged.
- `Categories` contains categories that were triggered.
- `CategoryScores` contains confidence scores per category.
- `CategoryAppliedInputTypes` contains input types that triggered each category.
- `WithConfidence(minPercent)` filters out low-confidence categories and adjusts `Flagged` accordingly.

Example:

```go
builder := client.Moderation.NewModerationBuilder()
builder.SetMinConfidence(50).
       AddText("Hello, world").
       AddImage("https://example.com/image.png")

results, _ := builder.Execute()

for _, res := range results {
    fmt.Printf("Input: %q | Flagged: %v | Scores: %v\n",
        res.Input, res.Flagged, res.CategoryScores)
}
```

## Assistants API (deprecated)

The Assistants API service accessible through `Client.Assistants` provides methods to manage assistants:
- `CreateAssistant` creates a new assistant.
- `ListAssistant` lists all assistants for the client.
- `LoadAssistant` retrieves an assistant by ID.
- `DeleteAssistant` removes an assistant.
- `AssistantsRunRefreshInterval` gets the polling interval for run status checks.
- `SetAssistantsRunRefreshInterval` configures the polling interval for run status checks.

Other exposed types in the `assistants` package:
- `Content` is an interface listing all types that can be used as content in the Assistants API.
- `Assistant` provides methods `ID`, `Model`, `NewThread`, and `LoadThread` to manage assistant metadata and start conversation threads.
- `Thread` provides methods `AddMessage`, `Messages`, `Run`, and `RunAndFetch` to add messages and obtain assistant responses.
- `Run` provides methods `Await`, `SubmitToolOutputs`, `IsPending`, and `IsExpectingToolOutputs` to handle execution lifecycle.
- Options for creating threads and runs.

A use example:

```go
// Create a new assistant
asst, _ := client.Assistants.CreateAssistant(assistants.CreateParams{
    Name:  "Demo Assistant",
    Model: models.DefaultMini,
})

// Start a new conversation thread
thread, _ := asst.NewThread(nil)

// Add a user message
thread.AddMessage(assistants.InputMessage{
    Role:    roles.User,
    Content: "hi how are you?",
})

// Run the assistant and fetch the reply
_, msg, _ := thread.RunAndFetch(context.Background(), nil)
fmt.Println(msg.Content)
```

## Embeddings API

The Embeddings API service accessible through `Client.Embedding` provides methods to generate vector representations:
- `One` computes a vector for a single input.
- `Array` computes vectors for multiple inputs.
- `Dimensions` returns the number of dimensions used (default is 256).
- `SetDimensions` configures the embedding dimensions.

The `Vector` type offers methods:
- `AngleDiff` computes cosine similarity between two vectors.
- `Distance` computes Euclidean distance between two vectors.

Example:

```go
vec, _ := client.Embedding.One("Hello, world!")
fmt.Printf("Vector length: %d\n", len(vec))
```

## Completions API (Legacy)

The Completions API service accessible through `Client.Completion` provides a legacy completion endpoint:
- `Send` executes the request and returns a completion string.
- `NewRequest` creates a new empty request. It is only a shorthand to make the type `completion.Request` more easily discoverable. You can use the request type directly.

Mind that the Completions API is legacy and it's recommended to use newer APIs.

Other exposed types in the `completion` package:
- `Request` is the request body for the Completions API.

Example:

```go
req := completion.Request{
    Model:     models.GPTInstruct,
    Prompt:    "Once upon a time",
    MaxTokens: 100,
}
text, _ := client.Completion.Send(req)
fmt.Println(text)
```
