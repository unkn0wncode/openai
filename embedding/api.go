// Package embedding provides functions to use OpenAI's Embeddings API.
package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	openai "macbot/openai/internal"
	"math"
	"net/http"
)

const (
	apiURL = openai.BaseAPI + "v1/embeddings"
)

// DefaultDimensions is the number of dimensions that will be used when not specified.
// Can be altered externally.
var DefaultDimensions = 256

// Request is the request body for the Embeddings API.
type Request struct {
	// One or multiple strings to calculate embeddings for.
	Inputs []string `json:"input"`

	// The model to use for the embeddings.
	Model string `json:"model"`

	// Optional

	// The format to return the embeddings in.
	// Can be either float or base64.
	// Defaults to float.
	Format string `json:"encoding_format,omitempty"`

	// The number of dimensions the resulting output embeddings should have.
	// Only supported in text-embedding-3 and later models.
	Dimensions int `json:"dimensions,omitempty"`

	// A unique identifier representing your end-user, which can help OpenAI to monitor and detect abuse.
	User string `json:"user,omitempty"`
}

// response is the response body for the Embeddings API.
type response struct {
	Object string   `json:"object"`
	Data   []result `json:"data"`
	Model  string   `json:"model"`
	Usage  struct {
		Prompt int `json:"prompt_tokens"`
		Total  int `json:"total_tokens"`
	} `json:"usage"`
}

// Vector is a calculated embedding.
type Vector []float64

// result contains one embedding in response.
type result struct {
	Object    string  `json:"object"`
	Index     int     `json:"index"`
	Embedding Vector `json:"embedding"`
}

func (data Request) execute() ([]Vector, error) {
	if len(data.Inputs) == 0 {
		return nil, fmt.Errorf("no inputs provided")
	}

	if data.Model == "" {
		data.Model = DefaultModel
	}

	if data.Dimensions == 0 && data.Model != ModelAda2 {
		data.Dimensions = DefaultDimensions
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	openai.AddHeaders(req)

	resp, err := openai.Cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"request (model %s) failed with status: %s, response body: %s",
			data.Model, resp.Status, string(body),
		)
	}

	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var res response
	if err := json.Unmarshal(rb, &res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	vecs := make([]Vector, len(data.Inputs))
	for i, r := range res.Data {
		vecs[i] = r.Embedding
	}

	return vecs, nil
}

// CustomRequest sends a request to the Embeddings API, returns calculated embeddings.
func CustomRequest(req Request) ([]Vector, error) {
	return req.execute()
}

// One sends a request to the Embeddings API with one input, returns one calculated embedding.
func One(input string) (Vector, error) {
	vecs, err := CustomRequest(Request{
		Inputs: []string{input},
	})
	if err != nil {
		return nil, err
	}

	return vecs[0], nil
}

// Array calculates embeddings for multiple inputs.
func Array(inputs ...string) ([]Vector, error) {
	return CustomRequest(Request{
		Inputs: inputs,
	})
}

// AngleDiff calculates the cosine similarity between two vectors.
// 1 means that vectors are parallel, 0 - orthogonal, -1 - antiparallel.
func (v Vector) AngleDiff(v2 Vector) float64 {
	if len(v) != len(v2) {
		return 0
	}

	var dot, mag1, mag2 float64
	for i := range v {
		dot += (v)[i] * (v2)[i]
		mag1 += (v)[i] * (v)[i]
		mag2 += (v2)[i] * (v2)[i]
	}

	mag1 = math.Sqrt(mag1)
	mag2 = math.Sqrt(mag2)

	return dot / (mag1 * mag2)
}

// Distance calculates the Euclidean distance between two vectors.
func (v Vector) Distance(v2 Vector) float64 {
	if len(v) != len(v2) {
		return 0
	}

	var sum float64
	for i := range v {
		sum += ((v)[i] - (v2)[i]) * ((v)[i] - (v2)[i])
	}

	return math.Sqrt(sum)
}
