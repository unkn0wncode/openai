// Package inmoderation provides a wrapper for the OpenAI Moderation API.
package inmoderation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/moderation"
	"github.com/unkn0wncode/openai/util"
)

// ModerationClient is a client for the OpenAI Moderation API.
type ModerationClient struct {
	*openai.Config

	// MinConfidencePercent is the minimum confidence percentage for a flag to be reported.
	// Zero (default) means everything that API returns is reported.
	// Can be set externally.
	MinConfidencePercent int
}

// interface conformity checks
var (
	_ moderation.Service = (*ModerationClient)(nil)
	_ moderation.Builder = (*Builder)(nil)
)

// request is the request body for the Moderation API.
type request struct {
	// required
	Input any `json:"input"` // can be: string, []string, []interface{Image, Text}

	// optional
	Model string `json:"model,omitempty"` // default "text-moderation-latest"
}

// response is the response body for the Moderation API.
type response struct {
	ID      string               `json:"id"`
	Model   string               `json:"model"`
	Results []*moderation.Result `json:"results"`
}

// Image is an image URL or a base64-encoded image to be moderated.
// It is marshaled as an object containing its type and value.
type Image string

// MarshalJSON marshals the image as an object containing its type and value.
func (i Image) MarshalJSON() ([]byte, error) {
	if i == "" {
		return []byte(`null`), nil
	}

	return json.Marshal(struct {
		Type  string `json:"type"`
		Image struct {
			URL string `json:"url"`
		} `json:"image_url"`
	}{
		Type: "image_url",
		Image: struct {
			URL string `json:"url"`
		}{
			URL: string(i),
		},
	})
}

// Text is a string of text to be moderated.
// It is marshaled as an object containing its type and value.
type Text string

// MarshalJSON marshals the text as an object containing its type and value.
func (t Text) MarshalJSON() ([]byte, error) {
	if t == "" {
		return []byte(`null`), nil
	}

	return json.Marshal(struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{
		Type: "text",
		Text: string(t),
	})
}

// send executes a moderation request using the client's HTTPClient and logger.
func (c *ModerationClient) send(r *request) (*response, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseAPI+"v1/moderations", bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.AddHeaders(req)

	var resp *http.Response
	err = util.Retry(func() error {
		var err error
		resp, err = c.HTTPClient.Do(req)
		return err
	}, 3, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %s, body: %s", resp.Status, string(body))
	}

	var res response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// Builder allows to add inputs and execute a moderation request on a multimodal "omni" model.
// Ready for use at zero value.
// Can be reused for multiple requests but not asynchronously.
type Builder struct {
	client               *ModerationClient
	texts                []Text
	images               []Image
	MinConfidencePercent int
}

// NewModerationBuilder returns a new Builder with client settings applied.
func (c *ModerationClient) NewModerationBuilder() moderation.Builder {
	return &Builder{
		client:               c,
		texts:                []Text{},
		images:               []Image{},
		MinConfidencePercent: c.MinConfidencePercent,
	}
}

// AddText adds a text input to the builder.
func (b *Builder) AddText(text string) moderation.Builder {
	if text == "" {
		return b
	}
	b.texts = append(b.texts, Text(text))
	return b
}

// AddImage adds an image input to the builder.
// Ignores unsupported image types.
// Can be a URL or a base64-encoded image data-uri, like:
//
//	AddImage("data:image/jpeg;base64,abcdefg...")
func (b *Builder) AddImage(url string) moderation.Builder {
	if url == "" {
		return b
	}

	supported := false
	for _, ext := range openai.SupportedImageTypes {
		if strings.Contains(url, ext) {
			supported = true
			break
		}
	}

	if !supported {
		b.client.Log.Warn(fmt.Sprintf(
			"OpenAI moderation got unsupported image type: %s",
			url,
		))
		return b
	}

	b.images = append(b.images, Image(url))
	return b
}

// SetMinConfidence sets the minimum confidence percentage for a flag to be reported.
func (b *Builder) SetMinConfidence(minPercent int) moderation.Builder {
	b.MinConfidencePercent = minPercent
	return b
}

// Clear removes all inputs and settings from the builder.
func (b *Builder) Clear() moderation.Builder {
	b.texts = nil
	b.images = nil
	b.MinConfidencePercent = b.client.MinConfidencePercent
	return b
}

// Execute sends the request to the OpenAI Moderation API and returns
// a common result for all text inputs combined with each image.
// If only some of requests failed, can return partial results.
// The inputs are not reset automatically, Clear() must be called for that.
func (bld *Builder) Execute() ([]*moderation.Result, error) {
	var inputSets [][]any
	switch {
	case len(bld.texts) == 0 && len(bld.images) == 0:
		// no inputs
		return nil, fmt.Errorf("no inputs provided")
	case len(bld.images) <= 1:
		// with at most 1 image we can send everything together
		var inputSet []any
		for _, text := range bld.texts {
			inputSet = append(inputSet, text)
		}
		for _, image := range bld.images {
			inputSet = append(inputSet, image)
		}
		inputSets = append(inputSets, inputSet)
	default:
		// with more than 1 image we need to send all texts with each image
		for _, image := range bld.images {
			var inputSet []any
			for _, text := range bld.texts {
				inputSet = append(inputSet, text)
			}
			inputSet = append(inputSet, image)
			inputSets = append(inputSets, inputSet)
		}
	}

	var results []*moderation.Result
	var errs []error
	for _, input := range inputSets {
		req := &request{
			Input: input,
			Model: models.DefaultModeration,
		}
		res, err := bld.client.send(req)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"failed to execute moderation request: %w",
				err,
			))
			results = append(results, nil)
			continue
		}

		if len(res.Results) != 1 {
			errs = append(errs, fmt.Errorf(
				"expected 1 result, got %d: %+v",
				len(res.Results), res,
			))
			results = append(results, res.Results[0]) // include first result
			continue
		}

		result := res.Results[0]
		result.WithConfidence(bld.MinConfidencePercent)
		result.Input = fmt.Sprint(input)
		if result.Flagged {
			bld.client.Log.Info(fmt.Sprintf(
				"OpenAI moderation flagged input: %s",
				result.Input,
			))
		}

		results = append(results, result)
	}

	return results, errors.Join(errs...)
}
