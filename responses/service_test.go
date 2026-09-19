// Package responses_test / service_test.go tests the separation of root answers and agent work.
package responses_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkn0wncode/openai/content/output"
	"github.com/unkn0wncode/openai/responses"
)

func TestResponseSeparatesRootAnswerFromAgentWork(t *testing.T) {
	t.Parallel()

	const work = `[
		{"type":"message","role":"assistant","phase":"commentary","agent":{"agent_name":"/root"},"content":[{"type":"output_text","text":"Checking the reports","annotations":[]}]},
		{"type":"message","role":"assistant","phase":"final_answer","agent":{"agent_name":"/root/reviewer"},"content":[{"type":"output_text","text":"Subagent report","annotations":[]}]},
		{"type":"message","role":"assistant","phase":"final_answer","agent":{"agent_name":"/root/reviewer"},"content":[{"type":"refusal","refusal":"Subagent refusal"}]},
		{"type":"reasoning","id":"rs_root","agent":{"agent_name":"/root"},"summary":[{"type":"summary_text","text":"Root summary"}]},
		{"type":"reasoning","id":"rs_child","agent":{"agent_name":"/root/reviewer"},"summary":[{"type":"summary_text","text":"Subagent summary"}]},
		{"type":"function_call","call_id":"call_root","name":"lookup","arguments":"{}","agent":{"agent_name":"/root"}},
		{"type":"function_call","call_id":"call_child","name":"lookup","arguments":"{}","agent":{"agent_name":"/root/reviewer"}},
		{"type":"custom_tool_call","call_id":"custom_child","name":"lookup","input":"query","agent":{"agent_name":"/root/reviewer"}},
		{"type":"shell_call","call_id":"shell_child","action":{"commands":["pwd"]},"agent":{"agent_name":"/root/reviewer"}},
		{"type":"apply_patch_call","call_id":"patch_child","operation":{"type":"create_file","path":"report.txt","diff":"+Report"},"agent":{"agent_name":"/root/reviewer"}},
		{"type":"mcp_approval_request","id":"approval_child","server_label":"search","name":"lookup","arguments":"{}","agent":{"agent_name":"/root/reviewer"}}
	]`
	for _, tc := range []struct {
		name    string
		answer  string
		texts   []string
		refusal []string
	}{
		{
			name:   "root answer",
			answer: `{"type":"message","role":"assistant","phase":"final_answer","agent":{"agent_name":"/root"},"content":[{"type":"output_text","text":"Combined answer","annotations":[]},{"type":"output_text","text":"Conclusion","annotations":[]}]}`,
			texts:  []string{"Combined answer", "Conclusion"},
		},
		{
			name:   "answer without agent or phase metadata",
			answer: `{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Combined answer","annotations":[]},{"type":"output_text","text":"Conclusion","annotations":[]}]}`,
			texts:  []string{"Combined answer", "Conclusion"},
		},
		{
			name:    "root refusal",
			answer:  `{"type":"message","role":"assistant","phase":"final_answer","agent":{"agent_name":"/root"},"content":[{"type":"refusal","refusal":"Root refusal"}]}`,
			refusal: []string{"Root refusal"},
		},
		{name: "no root answer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var resp responses.Response
			require.NoError(t, json.Unmarshal([]byte(work), &resp.Outputs))
			if tc.answer != "" {
				var answer output.Any
				require.NoError(t, json.Unmarshal([]byte(tc.answer), &answer))
				resp.Outputs = append(resp.Outputs, answer)
			}
			original, err := json.Marshal(resp.Outputs)
			require.NoError(t, err)
			require.NoError(t, resp.Parse())

			require.Equal(t, tc.texts, resp.Texts())
			if len(tc.texts) > 0 {
				require.Equal(t, "Combined answer\nConclusion", resp.JoinedTexts())
				require.Equal(t, "Combined answer", resp.FirstText())
				require.Equal(t, "Conclusion", resp.LastText())
			} else {
				require.Empty(t, resp.JoinedTexts())
				require.Empty(t, resp.FirstText())
				require.Empty(t, resp.LastText())
			}
			require.Equal(t, tc.refusal, resp.Refusals())
			require.Equal(t, []string{"Root summary"}, resp.ReasoningSummaries())
			require.Equal(t, "Root summary", resp.JoinedReasoningSummaries())

			// Structured helpers retain subagent work and the attribution needed to handle it.
			require.Len(t, resp.Reasonings(), 2)
			require.Equal(t, "/root/reviewer", resp.Reasonings()[1].Agent.AgentName)
			require.Len(t, resp.FunctionCalls(), 2)
			require.Equal(t, "/root/reviewer", resp.FunctionCalls()[1].Agent.AgentName)
			require.Len(t, resp.CustomToolCalls(), 1)
			require.Equal(t, "/root/reviewer", resp.CustomToolCalls()[0].Agent.AgentName)
			require.Len(t, resp.ShellCalls(), 1)
			require.Equal(t, "/root/reviewer", resp.ShellCalls()[0].Agent.AgentName)
			require.Len(t, resp.ApplyPatchCalls(), 1)
			require.Equal(t, "/root/reviewer", resp.ApplyPatchCalls()[0].Agent.AgentName)
			require.Len(t, resp.MCPApprovalRequests(), 1)
			require.Equal(t, "/root/reviewer", resp.MCPApprovalRequests()[0].Agent.AgentName)
			after, err := json.Marshal(resp.Outputs)
			require.NoError(t, err)
			require.JSONEq(t, string(original), string(after))
			require.Len(t, resp.ParsedOutputs, len(resp.Outputs))
		})
	}
}
