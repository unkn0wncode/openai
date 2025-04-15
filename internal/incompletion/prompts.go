// Package incompletion / prompts.go provides exported functions for sending requests to the Completion API with different level of control.
package incompletion

import "github.com/unkn0wncode/openai/completion"

// Send sends a request to the Send API with custom data.
func (c *Client) Send(req completion.Request) (string, error) {
	respData, err := c.execute(req)
	if err != nil {
		return "", err
	}

	return c.checkFirst(respData)
}
