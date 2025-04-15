package responses

/*
// cleanupFunctions unregisters all functions
func cleanupFunctions() {
	// Reset the function registry
	funcCalls = map[string]FunctionCall{}
	toolRegistry = map[string]Tool{}
}

// TestBasicResponseRequest tests a basic request/response
func TestBasicResponseRequest(t *testing.T) {
	// Test SingleResponsePrompt
	response, id, err := SingleResponsePrompt("Hello, world! Please respond with a short greeting.", "test-user")
	require.NoError(t, err, "SingleResponsePrompt should not fail")
	require.NotEmpty(t, response, "Expected non-empty response")
	require.NotEmpty(t, id, "Expected non-empty response ID")
	t.Logf("SingleResponsePrompt response: %s", response)
	t.Logf("SingleResponsePrompt ID: %s", id)

	// Test PrimedResponsePrompt
	response, id, err = PrimedResponsePrompt(
		"You are a helpful assistant that responds with short, concise answers.",
		"Hello, world! Please respond with a short greeting.",
		"test-user",
	)
	require.NoError(t, err, "PrimedResponsePrompt should not fail")
	require.NotEmpty(t, response, "Expected non-empty response")
	require.NotEmpty(t, id, "Expected non-empty response ID")
	t.Logf("PrimedResponsePrompt response: %s", response)
	t.Logf("PrimedResponsePrompt ID: %s", id)
}

// TestContinuedConversation tests a continued conversation using previous_response_id
func TestContinuedConversation(t *testing.T) {
	// First request to get a response ID
	initialPrompt := "Hello, world! Please remember the number 12345."
	initialResponse, initialResponseID, err := SingleResponsePrompt(initialPrompt, "test-user")
	require.NoError(t, err, "Initial request should not fail")
	require.NotEmpty(t, initialResponse, "Expected non-empty initial response")
	require.NotEmpty(t, initialResponseID, "Expected non-empty initial response ID")
	t.Logf("Initial response: %s", initialResponse)
	t.Logf("Initial response ID: %s", initialResponseID)

	// Second request using previous_response_id
	continuedReq := ResponseRequest{
		Model:              DefaultModel,
		Input:              "What number did I ask you to remember?",
		PreviousResponseID: initialResponseID,
		User:               "test-user",
	}

	continuedResponse, continuedID, err := CustomResponsePrompt(&continuedReq)
	require.NoError(t, err, "Continued conversation request should not fail")
	require.NotEmpty(t, continuedResponse, "Expected non-empty response")
	require.NotEmpty(t, continuedID, "Expected non-empty response ID")
	t.Logf("Continued conversation response: %s", continuedResponse)
	t.Logf("Continued conversation ID: %s", continuedID)
}

// TestReturnToolCalls tests returning tool calls instead of executing them
func TestReturnToolCalls(t *testing.T) {
	// Clean up any registered functions
	cleanupFunctions()

	// Register a test function
	testFunction := FunctionCall{
		Name:         "get_current_weather",
		Description:  "Get the current weather in a given location",
		ParamsSchema: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string","description":"The city and state, e.g. San Francisco, CA"},"unit":{"type":"string","enum":["celsius","fahrenheit"]}},"required":["location"]}`),
	}

	// Register the function
	RegisterFunction(testFunction)
	RegisterTool(Tool{
		Type:     "function",
		Name:     "get_current_weather",
		Function: testFunction,
	})

	// Create a request with tools and ReturnToolCalls=true
	req := ResponseRequest{
		Model: DefaultModel,
		Input: "What's the weather like in San Francisco?",
		Tools: []Tool{
			{
				Type:     "function",
				Name:     "get_current_weather",
				Function: testFunction,
			},
		},
		User:            "test-user",
		ReturnToolCalls: true,
	}

	response, id, err := CustomResponsePrompt(&req)
	require.NoError(t, err, "CustomResponsePrompt with ReturnToolCalls should not fail")
	require.NotEmpty(t, id, "Expected non-empty response ID")

	// Parse the response as JSON
	var toolCalls []ToolCall
	err = json.Unmarshal([]byte(response), &toolCalls)
	require.NoError(t, err, "Failed to parse tool calls")
	require.NotEmpty(t, toolCalls, "Expected at least one tool call")

	foundWeatherFunction := false
	for _, tc := range toolCalls {
		if tc.Function.Name == "get_current_weather" {
			foundWeatherFunction = true
			// The API might not include the location in the arguments, so we don't check for it
			t.Logf("Function arguments: %s", tc.Function.Arguments)
		}
	}

	require.True(t, foundWeatherFunction, "Expected to find get_current_weather function call")
	t.Logf("Tool calls: %s", response)
	t.Logf("Tool calls ID: %s", id)
}
*/
