package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitMsg(t *testing.T) {
	tests := []struct {
		name        string
		lengthLimit int
		msg         string
		expected    []string
	}{
		{
			name:        "Empty message",
			lengthLimit: 10,
			msg:         "",
			expected:    []string{""},
		},
		{
			name:        "No limit",
			lengthLimit: 0,
			msg:         "no split needed",
			expected:    []string{"no split needed"},
		},
		{
			name:        "3-letter split",
			lengthLimit: 2,
			msg:         "abc",
			expected:    []string{"ab", "c"},
		},
		{
			name:        "Below limit",
			lengthLimit: 20,
			msg:         "short message",
			expected:    []string{"short message"},
		},
		{
			name:        "Split on space",
			lengthLimit: 10,
			msg:         "this is a longer message that needs to be split",
			expected:    []string{"this is a", "longer", "message", "that", "needs to", "be split"},
		},
		{
			name:        "Newlines",
			lengthLimit: 10,
			msg:         "a message\nwith\nn and s",
			expected:    []string{"a message", "with", "n and s"},
		},
		{
			name:        "Long word split",
			lengthLimit: 10,
			msg:         "message with a verylongwordthatneedstobesplit",
			expected:    []string{"message", "with a", "verylongwo", "rdthatneed", "stobesplit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitMsg(tt.msg, tt.lengthLimit)

			for _, part := range result {
				if tt.lengthLimit <= 0 {
					break
				}

				require.LessOrEqual(
					t, len(part), tt.lengthLimit,
					"part '%s' is longer than length limit (%d)",
					part, tt.lengthLimit,
				)
			}

			require.Equal(t, tt.expected, result)
		})
	}
}
