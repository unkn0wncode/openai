package inembedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/unkn0wncode/openai/embedding"
	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/models"
)

// Client is the client for the Embeddings API.
type Client struct {
	*openai.Config
	dimensions int
}

// defaultDimensions is the number of dimensions that will be used when not specified.
const defaultDimensions = 256

// NewClient creates a new EmbeddingsClient.
func NewClient(config *openai.Config) *Client {
	return &Client{Config: config, dimensions: defaultDimensions}
}

// interface compliance checks
var _ embedding.Service = (*Client)(nil)

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

// result contains one embedding in response.
type result struct {
	Object    string           `json:"object"`
	Index     int              `json:"index"`
	Embedding embedding.Vector `json:"embedding"`
}

func (c *Client) executeRequest(data Request) ([]embedding.Vector, error) {
	if len(data.Inputs) == 0 {
		return nil, fmt.Errorf("no inputs provided")
	}

	if data.Model == "" {
		data.Model = models.DefaultEmbedding
	}

	if data.Dimensions == 0 && data.Model != models.Ada2 {
		data.Dimensions = defaultDimensions
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseAPI+"v1/embeddings", bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.AddHeaders(req)

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

	vecs := make([]embedding.Vector, len(data.Inputs))
	for i, r := range res.Data {
		vecs[i] = r.Embedding
	}

	return vecs, nil
}

// One sends a request to the Embeddings API with one input, returns one calculated embedding.
func (c *Client) One(input string) (embedding.Vector, error) {
	vecs, err := c.executeRequest(Request{
		Inputs: []string{input},
	})
	if err != nil {
		return nil, err
	}

	return vecs[0], nil
}

// Array calculates embeddings for multiple inputs.
func (c *Client) Array(inputs ...string) ([]embedding.Vector, error) {
	return c.executeRequest(Request{
		Inputs: inputs,
	})
}

// Dimensions returns the number of dimensions used for embeddings.
func (c *Client) Dimensions() int {
	return c.dimensions
}

// SetDimensions sets the number of dimensions used for embeddings.
func (c *Client) SetDimensions(dimensions int) {
	c.dimensions = dimensions
}
