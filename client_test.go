package openai

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/unkn0wncode/openai/assistants"
	"github.com/unkn0wncode/openai/chat"
	"github.com/unkn0wncode/openai/completion"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
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
		Model: models.Default,
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
	require.NoError(t, c.Chat.Config.Tools.CreateFunction(testFunc))

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

	resp, err := c.Completion.Completion(req)
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
		Model: models.DefaultNano,
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
	c.Config().Tools.CreateFunction(testFunction)
	toolReg := c.Tools()
	require.NotNil(t, toolReg)
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
