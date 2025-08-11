package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/unkn0wncode/openai/assistants"
	"github.com/unkn0wncode/openai/chat"
	"github.com/unkn0wncode/openai/completion"
	"github.com/unkn0wncode/openai/content/output"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/responses/streaming"
	"github.com/unkn0wncode/openai/roles"
	"github.com/unkn0wncode/openai/tools"

	"github.com/stretchr/testify/require"
)

var testToken string

// TestMain prepares the test environment by reading the API token from the .env file.
func TestMain(m *testing.M) {
	if data, err := os.ReadFile(".env"); err == nil {
		for line := range strings.SplitSeq(string(data), "\n") {
			if kv := strings.SplitN(line, "=", 2); len(kv) == 2 {
				os.Setenv(kv[0], kv[1])
			}
		}
	}
	if testToken = os.Getenv("OPENAI_API_KEY"); testToken == "" {
		fmt.Fprintln(os.Stderr, "OPENAI_API_KEY not set, skipping integration tests")
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// TestClient_Chat_hi checks the basic chat functionality by sending a "hi" message
// and checking the response.
func TestClient_Chat_hi(t *testing.T) {
	c := NewClient(testToken)

	req := chat.Request{
		Messages: []chat.Message{
			{Role: roles.User, Content: "hi"},
		},
	}

	resp, err := c.Chat.Send(req)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	t.Logf("resp: %s", resp)
}

// TestClient_Chat_Function checks the function calling functionality in chat API.
func TestClient_Chat_Function(t *testing.T) {
	c := NewClient(testToken)

	// Register test function
	var called bool
	testFunc := tools.FunctionCall{
		Name:         "test_function",
		Description:  "Test function",
		ParamsSchema: tools.EmptyParamsSchema,
		F: func(params json.RawMessage) (string, error) {
			called = true
			return `{"result":true}`, nil
		},
	}
	require.NoError(t, c.Tools().CreateFunction(testFunc))

	// Prepare request with forced function call
	req := chat.Request{
		Model:      models.Default,
		Messages:   []chat.Message{{Role: roles.User, Content: "call test function"}},
		Functions:  []string{"test_function"},
		ToolChoice: tools.ToolChoiceOption("test_function"),
	}

	resp, err := c.Chat.Send(req)
	require.NoError(t, err)
	require.True(t, called, "expected function to be called")
	require.NotEmpty(t, resp)

	t.Logf("resp: %s", resp)
}

// TestClient_Moderation checks the moderation functionality in moderation API.
func TestClient_Moderation(t *testing.T) {
	c := NewClient(testToken)
	bld := c.Moderation.NewModerationBuilder()
	bld.SetMinConfidence(50)

	t.Run("safe", func(t *testing.T) {
		bld.AddText("hi")
		res, err := bld.Execute()
		require.NoError(t, err)
		require.NotEmpty(t, res)
		require.False(t, res[0].Flagged)
		t.Logf("res: %v", res[0].CategoryScores)
	})

	t.Run("harmful", func(t *testing.T) {
		bld.AddText("fuck you")
		res, err := bld.Execute()
		require.NoError(t, err)
		require.NotEmpty(t, res)
		require.True(t, res[0].Flagged)
		t.Logf("res: %v", res[0].CategoryScores)
	})
}

// TestClient_Completion checks the completion functionality in completion API.
func TestClient_Completion(t *testing.T) {
	c := NewClient(testToken)

	req := completion.Request{
		Model:     models.GPTInstruct,
		Prompt:    "hi, how are you?",
		MaxTokens: 1024,
	}

	resp, err := c.Completion.Send(req)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	t.Logf("resp: %s", resp)
}

// TestClient_Assistants checks the assistants functionality in assistants API.
func TestClient_Assistants(t *testing.T) {
	c := NewClient(testToken)

	// Create a new assistant
	assistant, err := c.Assistants.CreateAssistant(assistants.CreateParams{
		Name:  "Test Assistant",
		Model: models.GPT4QuasarNano, // GPT-5 does not support assistants
	})
	require.NoError(t, err)
	require.NotNil(t, assistant)

	// List assistantsList
	assistantsList, err := c.Assistants.ListAssistant()
	require.NoError(t, err)
	require.NotEmpty(t, assistantsList)

	// Create a new thread
	thread, err := assistant.NewThread(nil)
	require.NoError(t, err)
	require.NotNil(t, thread)

	// Add a message to the thread
	addedMsg, err := thread.AddMessage(assistants.InputMessage{
		Role:    roles.User,
		Content: "Hello, how are you?",
	})
	require.NoError(t, err)
	require.NotNil(t, addedMsg)

	// Run the thread
	run, msg, err := thread.RunAndFetch(t.Context(), nil)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.NotNil(t, msg)

	// Delete the assistant
	err = c.Assistants.DeleteAssistant(assistant.ID())
	require.NoError(t, err)

	t.Logf("response: %s", msg.Content)
}

// TestClient_Responses_hi checks the responses functionality in responses API.
func TestClient_Responses_hi(t *testing.T) {
	c := NewClient(testToken)

	req := &responses.Request{
		Model: models.Default,
		Input: "hi",
	}

	resp, err := c.Responses.Send(req)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	require.NotEmpty(t, resp.ID)

	// t.Logf("outputs count: %d, parsed: %d", len(resp.Outputs), len(resp.ParsedOutputs))
	// for i := range resp.Outputs {
	// 	if m, ok := resp.ParsedOutputs[i].(output.Message); ok {
	// 		t.Logf("output %d is a message with content count: %d, parsed: %d", i, len(m.Content), len(m.ParsedContent))
	// 		continue
	// 	}
	// 	t.Logf("output %d: %T, %#v", i, resp.Outputs[i], resp.ParsedOutputs[i])
	// }

	t.Logf("resp: %v", resp.Texts())
}

// TestClient_Responses_dialogue checks the responses functionality with mixed input types.
func TestClient_Responses_dialogue(t *testing.T) {
	c := NewClient(testToken)

	req := &responses.Request{
		Input: []any{
			output.Message{Role: roles.User, Content: "hi"},
			output.Message{Role: roles.AI, Content: "hello, how are you?"},
			output.Message{Role: roles.User, Content: "i'm fine, and you?"},
		},
	}

	resp, err := c.Responses.Send(req)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	require.NotEmpty(t, resp.ID)

	t.Logf("resp: %v", resp.Texts())
}

func TestClient_Responses_Function(t *testing.T) {
	c := NewClient(testToken)

	// Register a test function
	testFunctionCalled := false
	testFunctionArgs := ""

	testFunction := tools.FunctionCall{
		Name:         "get_current_weather",
		Description:  "Get the current weather in a given location",
		ParamsSchema: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string","description":"The city and state, e.g. San Francisco, CA"},"unit":{"type":"string","enum":["celsius","fahrenheit"]}},"required":["location"]}`),
		F: func(params json.RawMessage) (string, error) {
			testFunctionCalled = true
			testFunctionArgs = string(params)
			return `{"temperature": 22, "unit": "celsius", "description": "Sunny"}`, nil
		},
	}

	// Register the function
	toolReg := c.Tools()
	require.NoError(t, toolReg.CreateFunction(testFunction))
	require.Len(t, toolReg.FunctionCalls, 1)
	gotFunc, ok := toolReg.GetFunction("get_current_weather")
	require.True(t, ok)
	require.Equal(t, testFunction.Name, gotFunc.Name)
	require.Equal(t, testFunction.Description, gotFunc.Description)
	require.Equal(t, testFunction.ParamsSchema, gotFunc.ParamsSchema)

	// Create a request with tools
	req := responses.Request{
		Model: models.Default,
		Input: "What's the weather like in San Francisco?",
		Tools: []string{"get_current_weather"},
		User:  "test-user",
	}

	response, err := c.Responses.Send(&req)
	require.NoError(t, err)
	require.True(t, testFunctionCalled)
	require.NotEmpty(t, response.ID)

	// The API might not include the location in the arguments, so we don't check for it
	t.Logf("Function args: %s", testFunctionArgs)
	require.NotEqual(t, testFunctionArgs, "{}", "Expected function arguments to be non-empty")
	require.NotEmpty(t, response)
	t.Logf("Function calling response: %v", response.Texts())
}

func TestClient_Responses_jsonSchema(t *testing.T) {
	c := NewClient(testToken)

	req := responses.Request{
		Model: models.Default,
		Text: &responses.TextOptions{
			Format: responses.TextFormatType{
				Type: responses.TextFormatTypeJSONSchema,
				Schema: json.RawMessage(`{
					"type": "object",
					"properties": {
						"test_ok": {"type": "boolean"}
					},
					"required": ["test_ok"],
					"additionalProperties": false
				}`),
				Strict:      true,
				Name:        "test",
				Description: "send true if you see this correctly",
			},
		},
		Input: "send true",
	}

	resp, err := c.Responses.Send(&req)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	require.NotEmpty(t, resp.ID)
	outputs := resp.Texts()
	require.Len(t, outputs, 1)

	var respData struct {
		TestOk bool `json:"test_ok"`
	}
	require.NoError(t, json.Unmarshal([]byte(outputs[0]), &respData))
	require.True(t, respData.TestOk)
}

func TestClient_Embedding(t *testing.T) {
	c := NewClient(testToken)

	vec, err := c.Embedding.One("Hello, world!")
	require.NoError(t, err)
	require.NotEmpty(t, vec)
}

// TestClient_Responses_BackgroundPolling verifies background mode and Polling.
func TestClient_Responses_BackgroundPolling(t *testing.T) {
	c := NewClient(testToken)

	// Send with background mode
	resp, err := c.Responses.Send(&responses.Request{
		Model:      models.Default,
		Input:      "Tell me a short joke.",
		Background: true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.ID)
	require.Empty(t, resp.Outputs)

	// Poll until completed
	ctx, cancel := context.WithTimeout(t.Context(), 6*time.Second)
	defer cancel()
	final, err := c.Responses.Poll(ctx, resp.ID, 2*time.Second)
	require.NoError(t, err)
	require.NotEmpty(t, final.Texts())
	t.Logf("Background poll texts: %v", final.JoinedTexts())
}

// TestClient_Responses_WebSearch checks the web_search tool usage in responses API.
func TestClient_Responses_WebSearch(t *testing.T) {
	c := NewClient(testToken)

	c.Tools().RegisterTool(tools.Tool{
		Type: "web_search",
	})

	// Prepare request forcing the use of web_search tool
	req := responses.Request{
		Model: models.DefaultMini,
		Input: "What's the newest version of Golang? Use web_search tool to check.",
		Tools: []string{"web_search"}, // GPT-5 cannot force tool choice for web_search
		User:  "test-user",
	}

	resp, err := c.Responses.Send(&req)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	require.NotEmpty(t, resp.ID)

	// Ensure that the response includes a web_search_call output
	require.NoError(t, resp.Parse())
	var found bool
	for _, o := range resp.ParsedOutputs {
		if _, ok := o.(output.WebSearchCall); ok {
			found = true
			break
		}
	}
	t.Logf("resp: %v", resp.Texts())
	require.True(t, found, "expected web_search_call in response outputs")
}

func TestClient_Responses_Stream(t *testing.T) {
	c := NewClient(testToken)

	req := &responses.Request{
		Model:  models.DefaultNano,
		Input:  "Write a haiku about AI agents.",
		Stream: true,
	}

	stream, err := c.Responses.Stream(t.Context(), req)
	require.NoError(t, err)
	require.NotNil(t, stream)

	outputText := ""
	eventCount := 0
	for stream.Next() {
		eventCount++
		event := stream.Event()

		switch e := event.(type) {
		case streaming.ResponseOutputTextDelta:
			outputText += e.Delta
			t.Logf("text delta: %s", e.Delta)
		case streaming.ResponseOutputTextDone:
			t.Logf("streamed text: %s", e.Text)
		}
	}

	require.NoError(t, stream.Err())
	require.NotZero(t, eventCount)
	require.NotEmpty(t, outputText)
}

func TestClient_Responses_Stream_ContextCancellation(t *testing.T) {
	c := NewClient(testToken)

	req := &responses.Request{
		Model:  models.DefaultNano,
		Input:  "Write a very long detailed essay about artificial intelligence, machine learning, and the future of technology. Make it at least 2000 words.",
		Stream: true,
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	stream, err := c.Responses.Stream(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, stream)

	eventCount := 0
	for stream.Next() {
		eventCount++

		if eventCount == 5 {
			cancel()
		}

		event := stream.Event()
		if delta, ok := event.(streaming.ResponseOutputTextDelta); ok {
			t.Logf("Received delta: %s", delta.Delta)
		}
	}

	require.Error(t, stream.Err())
	require.Equal(t, context.Canceled, stream.Err())
	require.True(t, eventCount >= 5, "Should have received at least 5 events before cancellation")
	t.Logf("Received %d events before cancellation", eventCount)
}

func TestClient_Responses_Stream_ContextCancellation_Range(t *testing.T) {
	c := NewClient(testToken)

	req := &responses.Request{
		Model:  models.DefaultNano,
		Input:  "Write a very long detailed essay about artificial intelligence, machine learning, and the future of technology. Make it at least 2000 words.",
		Stream: true,
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	stream, err := c.Responses.Stream(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, stream)

	eventCount := 0
	for event := range stream.Chan() {
		eventCount++

		if eventCount == 5 {
			cancel()
		}

		switch e := event.(type) {
		case streaming.ResponseOutputTextDelta:
			t.Logf("Received delta: %s", e.Delta)
		case error:
			require.Equal(t, context.Canceled, e)
			t.Logf("Received expected cancellation error through channel after %d events", eventCount)
		}
	}

	if stream.Err() != nil {
		require.Equal(t, context.Canceled, stream.Err())
		t.Logf("Cancellation error also available via Err()")
	}
}

func TestClient_Responses_Stream_CollectText(t *testing.T) {
	c := NewClient(testToken)

	req := &responses.Request{
		Model:  models.DefaultNano,
		Input:  "Write a haiku about AI agents.",
		Stream: true,
	}

	stream, err := c.Responses.Stream(t.Context(), req)
	require.NoError(t, err)
	require.NotNil(t, stream)

	var fullText string
	for stream.Next() {
		event := stream.Event()
		if delta, ok := event.(streaming.ResponseOutputTextDelta); ok {
			fullText += delta.Delta
		}
	}
	require.NotEmpty(t, fullText)

	t.Logf("collected text: %s", fullText)
}

func TestClient_Responses_Stream_Range(t *testing.T) {
	c := NewClient(testToken)

	req := &responses.Request{
		Model:  models.DefaultNano,
		Input:  "Write a haiku about AI agents.",
		Stream: true,
	}

	stream, err := c.Responses.Stream(t.Context(), req)
	require.NoError(t, err)
	require.NotNil(t, stream)

	outputText := ""
	eventCount := 0
	for event := range stream.Chan() {
		eventCount++

		switch e := event.(type) {
		case streaming.ResponseOutputTextDelta:
			outputText += e.Delta
			t.Logf("text delta: %s", e.Delta)
		case streaming.ResponseOutputTextDone:
			t.Logf("streamed text: %s", e.Text)
		case error:
			require.NoError(t, e)
		}
	}

	require.NoError(t, stream.Err())
	require.NotZero(t, eventCount)
	require.NotEmpty(t, outputText)
}

func TestClient_Responses_Stream_All(t *testing.T) {
	c := NewClient(testToken)

	req := &responses.Request{
		Model:  models.DefaultNano,
		Input:  "Write a haiku about AI agents.",
		Stream: true,
	}

	stream, err := c.Responses.Stream(t.Context(), req)
	require.NoError(t, err)
	require.NotNil(t, stream)

	events := stream.All()
	require.NotEmpty(t, events)

	var outputText string
	textDeltaCount := 0
	for _, event := range events {
		if delta, ok := event.(streaming.ResponseOutputTextDelta); ok {
			outputText += delta.Delta
			textDeltaCount++
		}
	}

	require.NoError(t, stream.Err())
	require.NotZero(t, textDeltaCount)
	require.NotEmpty(t, outputText)
	t.Logf("Collected %d total events, %d text deltas", len(events), textDeltaCount)
	t.Logf("Final text: %s", outputText)
}

func TestClient_Responses_Stream_MultipleChan(t *testing.T) {
	c := NewClient(testToken)

	req := &responses.Request{
		Model:  models.DefaultNano,
		Input:  "Write a haiku about AI agents.",
		Stream: true,
	}

	stream, err := c.Responses.Stream(t.Context(), req)
	require.NoError(t, err)
	require.NotNil(t, stream)

	ch1 := stream.Chan()
	ch2 := stream.Chan()
	ch3 := stream.Chan()

	require.Equal(t, ch1, ch2, "Chan() should return same channel on multiple calls")
	require.Equal(t, ch1, ch3, "Chan() should return same channel on multiple calls")

	eventCount := 0
	var outputText string
	for event := range ch1 {
		eventCount++
		if delta, ok := event.(streaming.ResponseOutputTextDelta); ok {
			outputText += delta.Delta
		}
	}

	require.NoError(t, stream.Err())
	require.NotZero(t, eventCount)
	require.NotEmpty(t, outputText)
	t.Logf("Multiple Chan() calls work correctly, got %d events", eventCount)
}
