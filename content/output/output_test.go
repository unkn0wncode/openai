package output

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkn0wncode/openai/content/input"
)

func TestAny_Unmarshal(t *testing.T) {
	input := []byte(`{"type":"my_type","my_field":"my_value"}`)

	var a Any
	err := json.Unmarshal(input, &a)
	require.NoError(t, err)
	require.NotEmpty(t, a.Type)
	require.NotEmpty(t, a.raw)
	require.Equal(t, "my_type", a.Type)

	var MyStruct struct {
		Type    string `json:"type"`
		MyField string `json:"my_field"`
	}
	err = a.UnmarshalToTarget(&MyStruct)
	require.NoError(t, err)
	require.NotEmpty(t, MyStruct.Type)
	require.NotEmpty(t, MyStruct.MyField)
	require.Equal(t, "my_type", MyStruct.Type)
	require.Equal(t, "my_value", MyStruct.MyField)
}

func TestAny_UnmarshalInputTypes(t *testing.T) {
	data := []byte(`{"type":"input_text","text":"hello"}`)
	var a Any
	require.NoError(t, json.Unmarshal(data, &a))

	value, err := a.Unmarshal()
	require.NoError(t, err)

	text, ok := value.(input.InputText)
	require.True(t, ok)
	require.Equal(t, "hello", text.Text)
}
