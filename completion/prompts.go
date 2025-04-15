// Package completion / prompts.go provides exported functions for sending requests to the Completion API with different level of control.
package completion

// CustomPrompt sends a request to the Completion API with custom data.
func CustomPrompt(req Request) (string, error) {
	respData, err := req.execute()
	if err != nil {
		return "", err
	}

	return respData.checkFirst()
}
