// Package chat / prompts.go provides exported functions for sending requests to the Chat API with different level of control.
package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"openai/chat"
	openai "openai/internal"
	"openai/models"
	"openai/roles"
	"openai/tools"
)

// SinglePrompt sends a request to the Chat API with a single user prompt and no additional context or settings.
// Returns the AI reply.
func (c *Client) SinglePrompt(prompt, userID string) (string, error) {
	req := chat.Request{
		Model: models.Default,
		Messages: []chat.Message{
			{Role: roles.User, Content: prompt},
		},
		User: userID,
	}

	return c.CustomPrompt(req)
}

// PrimedPrompt sends a request to the Chat API with a single user prompt primed by a given "system" message.
// Returns the AI reply.
func (c *Client) PrimedPrompt(systemMessage, prompt, userID string) (string, error) {
	req := chat.Request{
		Model: models.Default,
		Messages: []chat.Message{
			{Role: roles.System, Content: systemMessage},
			{Role: roles.User, Content: prompt},
		},
		User: userID,
	}

	return c.CustomPrompt(req)
}

// MessagesPrompt sends a request to the Chat API with a given sequence of messages.
// Returns the AI reply.
func (c *Client) MessagesPrompt(messages []chat.Message, userID string) (string, error) {
	req := chat.Request{
		Model:    models.Default,
		Messages: messages,
		User:     userID,
	}

	return c.CustomPrompt(req)
}

// CustomPrompt sends a request to the Chat API with custom data.
func (c *Client) CustomPrompt(req chat.Request) (string, error) {
	respData, err := c.execute(req)
	if err != nil {
		return "", err
	}

	// if response contains tool/function calls, it needs to be handled specially
	aiMessage := respData.Choices[0].Message
	if len(aiMessage.ToolCalls) > 0 {
		_, err := c.checkFirst(respData)
		if err != nil {
			return "", err
		}

		var callsToReturn []openai.FunctionCallData
		var callErrors []error
		for i, tc := range aiMessage.ToolCalls {
			f, ok := c.Config.Tools.GetFunction(tc.Function.Name)
			if !ok {
				return "", fmt.Errorf("function '%s' is not registered", tc.Function.Name)
			}

			// if function calls are returned or there's no function to run
			if req.ReturnFunctionCalls || f.F == nil {
				callsToReturn = append(callsToReturn, aiMessage.ToolCalls[i].Function)
				continue
			}

			// by default function calls are executed.
			// execute the function and add the result to the response,
			// so request can be sent again with results

			// add the response we got to the request messages
			// so AI would know what functions it used
			if i == 0 {
				req.Messages = append(req.Messages, aiMessage)
			}

			fResult, err := f.F([]byte(tc.Function.Arguments))
			switch {
			case err == nil:
			case errors.Is(err, tools.ErrDoNotRespond):
				callErrors = append(callErrors, err)
				if fResult == "" {
					fResult = tools.TextDoNotRespond
				}
			default:
				// return "", fmt.Errorf("failed to execute function '%s': %w", tc.Function.Name, err)
				callErrors = append(callErrors, fmt.Errorf(
					"failed to execute function '%s': %w",
					tc.Function.Name, err,
				))
				c.Config.Log.Error("Function call error: %s", err)
			}
			c.Config.Log.Debug("Function '%s' returned: %s", tc.Function.Name, fResult)

			req.Messages = append(req.Messages, chat.Message{
				Role:       roles.Tool,
				ToolCallID: tc.ID,
				Content:    fResult,
			})

			// if function was used CallLimit times, force non-function response
			if f.CallLimit > 0 {
				uses := 0
				for _, msg := range req.Messages {
					if msg.Role == roles.Function && msg.Name == tc.Function.Name {
						uses++
					}
				}
				if uses >= f.CallLimit {
					c.Config.Log.Warn(
						"Function '%s' has reached its CallLimit (%d) times, forcing non-function response",
						tc.Function.Name,
						uses,
					)
					req.ToolChoice = "none"
				}
			}
		}

		// if any function calls gave error, return an error
		if err := errors.Join(callErrors...); err != nil {
			allDoNotRespond := errors.Is(err, tools.ErrDoNotRespond)
			for j := 0; allDoNotRespond && j < len(callErrors); j++ {
				if !errors.Is(callErrors[j], tools.ErrDoNotRespond) {
					allDoNotRespond = false
				}
			}
			if allDoNotRespond {
				err = tools.ErrDoNotRespond
			}

			// try to return what we normally would plus all errors
			if len(callsToReturn) > 0 {
				var marshalErr error
				b, marshalErr := json.Marshal(callsToReturn)
				return string(b), errors.Join(err, marshalErr)
			}

			respText, respCheckErr := c.checkFirst(respData)
			return respText, errors.Join(err, respCheckErr)
		}

		// if we found any function calls that need to be returned, encode and return them
		// warning: we ignore text content of the response in this case, unless it's
		// extracted from modified request messages after returning function calls
		if len(callsToReturn) > 0 {
			b, err := json.Marshal(callsToReturn)
			if err != nil {
				return "", fmt.Errorf("failed to marshal function calls: %w", err)
			}

			return string(b), nil
		}

		// if the previous request forced use of a function, we must reset it to avoid a loop
		if req.ToolChoice != "auto" && req.ToolChoice != "none" {
			req.ToolChoice = ""
		}

		// Default supports now 16k
		// if model is default but a lot of tokens are already used, switch to a bigger model
		//if req.contextTokenLimit() < 16000 && req.countTokens() > int(0.9*float64(req.contextTokenLimit())) {
		//	openai.LogOpenAI.Println("Switching to larger model due to high token count")
		//	req.Model = ModelChatGPT16k
		//}

		return c.CustomPrompt(req)
	}

	return c.checkFirst(respData)
}
