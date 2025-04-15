// Package chat / prompts.go provides exported functions for sending requests to the Chat API with different level of control.
package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	openai "openai/internal"
)

// SinglePrompt sends a request to the Chat API with a single user prompt and no additional context or settings.
// Returns the AI reply.
func SinglePrompt(prompt, userID string) (string, error) {
	req := Request{
		Model: DefaultModel,
		Messages: []Message{
			{Role: RoleUser, Content: prompt},
		},
		User: userID,
	}

	return CustomPrompt(req)
}

// PrimedPrompt sends a request to the Chat API with a single user prompt primed by a given "system" message.
// Returns the AI reply.
func PrimedPrompt(systemMessage, prompt, userID string) (string, error) {
	req := Request{
		Model: DefaultModel,
		Messages: []Message{
			{Role: RoleSystem, Content: systemMessage},
			{Role: RoleUser, Content: prompt},
		},
		User: userID,
	}

	return CustomPrompt(req)
}

// MessagesPrompt sends a request to the Chat API with a given sequence of messages.
// Returns the AI reply.
func MessagesPrompt(messages []Message, userID string) (string, error) {
	req := Request{
		Model:    DefaultModel,
		Messages: messages,
		User:     userID,
	}

	return CustomPrompt(req)
}

// CustomPrompt sends a request to the Chat API with custom data.
func CustomPrompt(req Request) (string, error) {
	respData, err := req.execute()
	if err != nil {
		openai.Mon.Fail(err)
		return "", err
	}
	openai.Mon.Up()

	// if response contains tool/function calls, it needs to be handled specially
	aiMessage := respData.Choices[0].Message
	if len(aiMessage.ToolCalls) > 0 {
		_, err := respData.checkFirst()
		if err != nil {
			return "", err
		}

		var callsToReturn []FunctionCallData
		var callErrors []error
		for i, tc := range aiMessage.ToolCalls {
			f, ok := funcCalls[tc.Function.Name]
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
			case errors.Is(err, ErrDoNotRespond):
				callErrors = append(callErrors, err)
				if fResult == "" {
					fResult = TextDoNotRespond
				}
			default:
				// return "", fmt.Errorf("failed to execute function '%s': %w", tc.Function.Name, err)
				callErrors = append(callErrors, fmt.Errorf(
					"failed to execute function '%s': %w",
					tc.Function.Name, err,
				))
				openai.LogStd.Printf("Function call error: %s", err)
			}
			openai.Log.Printf("Function '%s' returned: %s", tc.Function.Name, fResult)

			req.Messages = append(req.Messages, Message{
				Role:       RoleTool,
				ToolCallID: tc.ID,
				Content:    fResult,
			})

			// if function was used CallLimit times, force non-function response
			if f.CallLimit > 0 {
				uses := 0
				for _, msg := range req.Messages {
					if msg.Role == RoleFunction && msg.Name == tc.Function.Name {
						uses++
					}
				}
				if uses >= f.CallLimit {
					openai.Log.Printf(
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
			allDoNotRespond := errors.Is(err, ErrDoNotRespond)
			for j := 0; allDoNotRespond && j < len(callErrors); j++ {
				if !errors.Is(callErrors[j], ErrDoNotRespond) {
					allDoNotRespond = false
				}
			}
			if allDoNotRespond {
				err = ErrDoNotRespond
			}

			// try to return what we normally would plus all errors
			if len(callsToReturn) > 0 {
				var marshalErr error
				b, marshalErr := json.Marshal(callsToReturn)
				return string(b), errors.Join(err, marshalErr)
			}

			respText, respCheckErr := respData.checkFirst()
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

		return CustomPrompt(req)
	}

	return respData.checkFirst()
}
