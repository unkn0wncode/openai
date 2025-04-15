// Package main / api.go contains types and functions for interacting with models.dev API.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const modelsAPI = "https://models.dev/api.json"

type payload map[providerName]providerData

type (
	providerName string
	modelID      string
)

type providerData struct {
	ID     providerName          `json:"id"`
	Env    []string              `json:"env"`
	NPM    string                `json:"npm"`
	Doc    string                `json:"doc"`
	Models map[modelID]modelData `json:"models"`
}

type modelData struct {
	ID          modelID `json:"id"`
	Name        string  `json:"name"`
	Attachment  bool    `json:"attachment"`
	Reasoning   bool    `json:"reasoning"`
	Temperature bool    `json:"temperature"`
	ToolCalls   bool    `json:"tool_calls"`
	Knowledge   string  `json:"knowledge"`    // YYYY-MM
	ReleaseDate string  `json:"release_date"` // YYYY-MM-DD
	LastUpdated string  `json:"last_updated"` // YYYY-MM-DD
	OpenWeights bool    `json:"open_weights"`
	Modalities  struct {
		Input  []string `json:"input"`
		Output []string `json:"output"`
	} `json:"modalities"`
	Cost struct {
		Input     float64 `json:"input"`
		Output    float64 `json:"output"`
		CacheRead float64 `json:"cache_read"`
	} `json:"cost"`
	Limit struct {
		Context int `json:"context"`
		Output  int `json:"output"`
	} `json:"limit"`
}

// fetchOpenAIData retrieves the model metadata for the `openai` provider.
func fetchOpenAIData() (*providerData, error) {
	logger.Info("Getting models from models.dev")
	resp, err := http.Get(modelsAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to get models.dev API data: %w", err)
	}
	defer resp.Body.Close()
	logger.Info("Response received", "status", resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read models.dev response: %w", err)
	}

	var apiData payload
	err = json.Unmarshal(body, &apiData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal models.dev response: %w", err)
	}
	logger.Info(fmt.Sprintf("Got providers: %d", len(apiData)))

	openAIData := apiData[providerOpenAI]
	logger.Info(fmt.Sprintf("Got %d models for %s", len(openAIData.Models), providerOpenAI))
	if len(openAIData.Models) == 0 {
		panic("no models returned for openai")
	}

	return &openAIData, nil
}
