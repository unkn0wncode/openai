package openai

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"openai/chat"
	"openai/completion"
	"openai/models"
	"openai/roles"
	"openai/tools"

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
// and checking that the response.
func TestClient_Chat_hi(t *testing.T) {
	c := NewClient(testToken)

	req := chat.Request{
		Model: models.Default,
		Messages: []chat.Message{
			{Role: roles.User, Content: "hi"},
		},
	}

	resp, err := c.Chat(req)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	// t.Logf("resp: %s", resp)
	// t.FailNow()
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
	require.NoError(t, c.ChatClient.Config.Tools.CreateFunction(testFunc))

	// Prepare request with forced function call
	req := chat.Request{
		Model:      models.Default,
		Messages:   []chat.Message{{Role: roles.User, Content: "call test function"}},
		Functions:  []string{"test_function"},
		ToolChoice: tools.ToolChoiceOption("test_function"),
	}

	resp, err := c.Chat(req)
	require.NoError(t, err)
	require.True(t, called, "expected function to be called")
	require.NotEmpty(t, resp)

	// t.Logf("resp: %s", resp)
	// t.FailNow()
}

// TestClient_Moderation checks the moderation functionality in moderation API.
func TestClient_Moderation(t *testing.T) {
	c := NewClient(testToken)
	bld := c.NewModerationBuilder()

	t.Run("safe", func(t *testing.T) {
		bld.AddText("hi")
		res, err := bld.Execute()
		require.NoError(t, err)
		require.NotEmpty(t, res)
		require.False(t, res[0].Flagged)
	})

	t.Run("harmful", func(t *testing.T) {
		bld.AddText("fuck you")
		res, err := bld.Execute()
		require.NoError(t, err)
		require.NotEmpty(t, res)
		require.True(t, res[0].Flagged)
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

	resp, err := c.Completion(req)
	require.NoError(t, err)
	require.NotEmpty(t, resp)

	// t.Logf("resp: %s", resp)
	// t.FailNow()
}
