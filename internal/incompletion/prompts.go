// Package incompletion / prompts.go provides exported functions for sending requests to the Completion API with different level of control.
package incompletion

import "github.com/unkn0wncode/openai/completion"

// Completion sends a request to the Completion API with custom data.
func (c *Client) Completion(req completion.Request) (string, error) {
	respData, err := c.execute(req)
	if err != nil {
		return "", err
	}

	return c.checkFirst(respData)
}
