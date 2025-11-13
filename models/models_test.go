package models

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	. "github.com/unkn0wncode/openai/internal"
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

// modelData is the payload returned by the API for each model in model list.
type modelData struct {
	Object  string `json:"object"` // always "model"
	ID      string `json:"id"`
	Created int    `json:"created"`  // Unix timestamp
	OwnedBy string `json:"owned_by"` // "system" or "openai" for OpenAI-owned models
}

// TestModelsList fetches all models from https://api.openai.com/v1/models and checks if
// the list is same as the hard-coded list in this package.
func TestModelsList(t *testing.T) {
	// fetch list of models from API

	client := NewHTTPClient()
	req, err := http.NewRequest(http.MethodGet, "https://api.openai.com/v1/models", nil)
	require.NoError(t, err)
	req.Header.Add("Authorization", "Bearer "+testToken)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var respData struct {
		Object string      `json:"object"` // always "list"
		Models []modelData `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &respData))
	t.Logf("Got %d models", len(respData.Models))

	// inventorize models from our package and from API to match them later

	var apiModels []string
	var packageModels []string
	for _, model := range respData.Models {
		if model.OwnedBy != "openai" && model.OwnedBy != "system" {
			// skip non-OpenAI models, such as organization-owned fine-tuned models
			continue
		}

		// add all models to the API models list
		apiModels = append(apiModels, model.ID)
	}

	// go through all implemented models and add them to maps
	for model := range Data {
		packageModels = append(packageModels, model)
	}
	for model := range DataEmbedding {
		packageModels = append(packageModels, model)
	}
	for model := range PricePerImageData {
		packageModels = append(packageModels, model)
	}

	// find mismatches:
	// 1. models in the package but not in the API are "deleted"
	// 2. models in the API but not in the package are "unimplemented"

	var deletedModels []string
	var unimplementedModels []string
	for _, model := range packageModels {
		if model == "" {
			// skip default model placeholder
			continue
		}
		if !slices.Contains(apiModels, model) {
			deletedModels = append(deletedModels, model)
		}
	}
	for _, model := range apiModels {
		if !slices.Contains(packageModels, model) {
			unimplementedModels = append(unimplementedModels, model)
		}
	}

	require.Empty(
		t,
		unimplementedModels,
		"These unimplemented models need to be added to the package:\n%s",
		strings.Join(unimplementedModels, "\n"),
	)
	require.Empty(
		t,
		deletedModels,
		"These deleted models need to be removed from the package:\n%s",
		strings.Join(deletedModels, "\n"),
	)
}
