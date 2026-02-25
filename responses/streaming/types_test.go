package streaming

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalErrorEvent_Flat(t *testing.T) {
	t.Parallel()

	event, err := Unmarshal([]byte(`{
		"type": "error",
		"sequence_number": 10,
		"message": "flat error",
		"code": "bad_request",
		"param": "input"
	}`))
	require.NoError(t, err)

	flatErr, ok := event.(Error)
	require.True(t, ok)
	require.Equal(t, "error", flatErr.Type)
	require.Equal(t, 10, flatErr.SequenceNumber)
	require.Equal(t, "flat error", flatErr.Message)
	require.Equal(t, "bad_request", flatErr.Code)
	require.NotNil(t, flatErr.Param)
	require.Equal(t, "input", *flatErr.Param)
}

func TestUnmarshalErrorEvent_NestedWebSocket(t *testing.T) {
	t.Parallel()

	event, err := Unmarshal([]byte(`{
		"type": "error",
		"status": 400,
		"error": {
			"type": "invalid_request_error",
			"code": "previous_response_not_found",
			"message": "Previous response with id 'resp_abc' not found.",
			"param": "previous_response_id"
		}
	}`))
	require.NoError(t, err)

	wsErr, ok := event.(WSError)
	require.True(t, ok)
	require.Equal(t, "error", wsErr.Type)
	require.Equal(t, 400, wsErr.Status)
	require.Equal(t, "invalid_request_error", wsErr.Error.Type)
	require.Equal(t, "previous_response_not_found", wsErr.Error.Code)
	require.Equal(t, "Previous response with id 'resp_abc' not found.", wsErr.Error.Message)
	require.NotNil(t, wsErr.Error.Param)
	require.Equal(t, "previous_response_id", *wsErr.Error.Param)
}
