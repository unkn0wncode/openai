// Package inresponses / regions_test.go tests regional endpoint recognition for cost estimates.
package inresponses

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessingRegion(t *testing.T) {
	for _, tt := range []struct{ endpoint, want string }{
		{"https://api.openai.com/", "global"},
		{"https://API.OPENAI.COM/", "global"},
		{"https://us.api.openai.com/", "us"},
		{"https://eu.api.openai.com/", "eu"},
		{"https://au.api.openai.com/", "au"},
		{"https://ca.api.openai.com/", "ca"},
		{"https://jp.api.openai.com/", "jp"},
		{"https://in.api.openai.com/", "in"},
		{"https://sg.api.openai.com/", "sg"},
		{"https://kr.api.openai.com/", "kr"},
		{"https://gb.api.openai.com/", "gb"},
		{"https://ae.api.openai.com/", "ae"},
		{"https://proxy.example/v1/", ""},
		{"http://127.0.0.1:8080/", ""},
		{"https://unknown.api.openai.com/", ""},
		{"https://us.api.openai.com.example/", ""},
	} {
		t.Run(tt.endpoint, func(t *testing.T) {
			endpoint, err := url.Parse(tt.endpoint)
			require.NoError(t, err)
			require.Equal(t, tt.want, processingRegion(endpoint))
		})
	}
}
