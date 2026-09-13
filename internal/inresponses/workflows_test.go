// Package inresponses / workflows_test.go tests tool continuation and replay across API responses.
package inresponses

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/unkn0wncode/openai/content/output"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/tools"

	"github.com/stretchr/testify/require"
)

func TestSendProgramContinuesThroughToolPauseToFinalMessage(t *testing.T) {
	t.Parallel()

	for _, stored := range []bool{true, false} {
		name := "stored"
		if !stored {
			name = "stateless"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			reasoning := json.RawMessage(`{"type":"reasoning","id":"rs_program","summary":[],"encrypted_content":"opaque_reasoning","future_replay_data":{"nonce":"keep"}}`)
			program := json.RawMessage(`{"type":"program","id":"prog_lookup","call_id":"call_program","code":"text(await tools.lookup({}));","fingerprint":"opaque_program_state","future_replay_data":{"nonce":"also_keep"}}`)
			function := json.RawMessage(`{"type":"function_call","id":"fc_lookup","call_id":"call_lookup","name":"lookup","arguments":"{}","status":"completed","caller":{"type":"program","caller_id":"call_program"}}`)
			result := json.RawMessage(`{"type":"program_output","id":"po_lookup","call_id":"call_program","result":"found","status":"completed"}`)
			const functionOutput = `{"type":"function_call_output","call_id":"call_lookup","output":"found","caller":{"type":"program","caller_id":"call_program"}}`
			var requests, executions atomic.Int32
			client := newAccountingClient(t, func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					Store              bool            `json:"store"`
					PreviousResponseID string          `json:"previous_response_id"`
					Input              json.RawMessage `json:"input"`
					Tools              []tools.Tool    `json:"tools"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
				require.Equal(t, stored, req.Store)
				var programmaticTool bool
				for _, tool := range req.Tools {
					programmaticTool = programmaticTool || tool.Type == "programmatic_tool_calling"
				}
				require.True(t, programmaticTool, "the hosted runtime must be present in every request")
				call := requests.Add(1)
				var body map[string]any
				switch call {
				case 1:
					body = accountingResponse("resp_program", "completed", "default", 100, 10, reasoning, program, function)
				case 2, 3:
					var items []json.RawMessage
					require.NoError(t, json.Unmarshal(req.Input, &items))
					if stored {
						if call == 2 {
							require.Equal(t, "resp_program", req.PreviousResponseID)
							require.Len(t, items, 1)
							require.JSONEq(t, functionOutput, string(items[0]))
						} else {
							require.Equal(t, "resp_program_result", req.PreviousResponseID)
							require.Empty(t, items)
						}
					} else {
						require.Empty(t, req.PreviousResponseID)
						require.Len(t, items, int(call)+3)
						require.JSONEq(t, `{"type":"message","role":"user","content":"Look up the value"}`, string(items[0]))
						require.JSONEq(t, string(reasoning), string(items[1]))
						require.JSONEq(t, string(program), string(items[2]))
						require.JSONEq(t, string(function), string(items[3]))
						require.JSONEq(t, functionOutput, string(items[4]))
						if call == 3 {
							require.JSONEq(t, string(result), string(items[5]))
						}
					}
					if call == 2 {
						body = accountingResponse("resp_program_result", "completed", "default", 200, 20, result)
					} else {
						body = accountingResponse("resp_final", "completed", "default", 300, 30, accountingMessage("The value is found"))
					}
				default:
					t.Error("program continued after its final answer")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				body["model"] = models.Default
				require.NoError(t, json.NewEncoder(w).Encode(body))
			})
			require.NoError(t, client.Tools.CreateFunction(tools.FunctionCall{
				Name: "lookup", Description: "Look up a value", ParamsSchema: tools.EmptyParamsSchema,
				AllowedCallers: []string{"programmatic"},
				F: func(json.RawMessage) (string, error) {
					executions.Add(1)
					return "found", nil
				},
			}))
			require.NoError(t, client.Tools.RegisterTool(tools.Tool{Type: "programmatic_tool_calling"}))
			req := &responses.Request{
				Model: models.Default, Input: "Look up the value", Store: &stored,
				Tools: []string{"lookup", "programmatic_tool_calling"},
			}

			resp, err := client.Send(req)

			require.NoError(t, err)
			require.EqualValues(t, 3, requests.Load())
			require.EqualValues(t, 1, executions.Load())
			require.Equal(t, "resp_final", resp.ID)
			require.Equal(t, "The value is found", resp.FirstText())
			require.Len(t, resp.Calls, 3)
			require.Equal(t, "resp_program", resp.Calls[0].ID)
			require.Equal(t, "resp_program_result", resp.Calls[1].ID)
			require.Equal(t, "resp_final", resp.Calls[2].ID)
			require.JSONEq(t, string(program), resp.Calls[0].Outputs[1].String())
			require.JSONEq(t, string(result), resp.Calls[1].Outputs[0].String())
			var expectedTotal float64
			for i, call := range resp.Calls {
				require.NotNil(t, call.Usage)
				require.Equal(t, (i+1)*100, call.Usage.InputTokens)
				amount, costErr := models.Data[models.Default].Cost(call.Usage)
				require.NoError(t, costErr)
				require.InDelta(t, amount, call.EstimatedCost, 1e-12)
				expectedTotal += amount
				if call.CostError != nil {
					require.ErrorIs(t, resp.CostError, call.CostError)
				}
			}
			require.InDelta(t, expectedTotal, resp.EstimatedCost, 1e-12)
			require.False(t, resp.BillingIncomplete)
		})
	}
}

func TestSendAsyncToolsRemainPendingAcrossLaterConversationTurns(t *testing.T) {
	t.Parallel()

	var requests, executions atomic.Int32
	client := newAccountingClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PreviousResponseID string          `json:"previous_response_id"`
			Input              json.RawMessage `json:"input"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		var body map[string]any
		switch requests.Add(1) {
		case 1:
			body = accountingResponse("resp_started", "completed", "default", 100, 10,
				json.RawMessage(`{"type":"function_call","id":"fc_slow","call_id":"call_slow","name":"slow_lookup","arguments":"{}","async":true}`),
				json.RawMessage(`{"type":"custom_tool_call","id":"ct_slow","call_id":"call_custom","name":"slow_custom","input":"query","async":true}`),
				json.RawMessage(accountingFunctionCall), accountingMessage("The work has started"))
		case 2:
			require.Equal(t, "resp_started", req.PreviousResponseID)
			body = accountingResponse("resp_latest", "completed", "default", 120, 10, accountingMessage("Still working"))
		case 3:
			require.Equal(t, "resp_latest", req.PreviousResponseID)
			require.JSONEq(t, `[
				{"type":"function_call_output","call_id":"call_slow","output":"function result"},
				{"type":"custom_tool_call_output","call_id":"call_custom","output":"custom result"},
				{"type":"function_call_output","call_id":"call_lookup","output":"lookup result"}
			]`, string(req.Input))
			body = accountingResponse("resp_finished", "completed", "default", 150, 15, accountingMessage("All results received"))
		default:
			t.Error("async calls triggered an automatic continuation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body["model"] = models.Default
		require.NoError(t, json.NewEncoder(w).Encode(body))
	})
	execute := func(json.RawMessage) (string, error) { executions.Add(1); return "unexpected automatic result", nil }
	registerAccountingFunction(t, client, execute)
	require.NoError(t, client.Tools.CreateFunction(tools.FunctionCall{
		Name: "slow_lookup", Description: "Start a lookup", ParamsSchema: tools.EmptyParamsSchema, Async: true, F: execute,
	}))
	require.NoError(t, client.Tools.RegisterTool(tools.Tool{
		Type: "custom", Name: "slow_custom", Async: true,
		Custom: func(string) (string, error) { executions.Add(1); return "unexpected automatic result", nil },
	}))
	req := &responses.Request{Model: models.Default, Input: "Start the work", Tools: []string{"lookup", "slow_lookup", "slow_custom"}}

	started, err := client.Send(req)

	require.NoError(t, err)
	require.EqualValues(t, 1, requests.Load())
	require.Zero(t, executions.Load())
	require.Equal(t, "The work has started", started.FirstText())
	functions := started.FunctionCalls()
	require.Len(t, functions, 2)
	require.True(t, functions[0].Async)
	custom := started.CustomToolCalls()
	require.Len(t, custom, 1)
	require.True(t, custom[0].Async)

	req.PreviousResponseID, req.Input = started.ID, "What is the current status?"
	latest, err := client.Send(req)
	require.NoError(t, err)
	req.PreviousResponseID = latest.ID
	req.Input = []any{
		output.FunctionCallOutput{CallID: functions[0].CallID, Output: "function result"},
		output.CustomToolCallOutput{CallID: custom[0].CallID, Output: "custom result"},
		output.FunctionCallOutput{CallID: functions[1].CallID, Output: "lookup result"},
	}
	finished, err := client.Send(req)

	require.NoError(t, err)
	require.Equal(t, "All results received", finished.FirstText())
	require.EqualValues(t, 3, requests.Load())
	require.Zero(t, executions.Load())
	for _, resp := range []*responses.Response{started, latest, finished} {
		require.Len(t, resp.Calls, 1)
		require.Equal(t, resp.ID, resp.Calls[0].ID)
		require.NotNil(t, resp.Usage)
		require.NoError(t, resp.CostError)
	}
}

func TestSendMultiAgentKeepsHostedWorkAndFunctionResultsAttributed(t *testing.T) {
	t.Parallel()

	spawn := json.RawMessage(`{"type":"multi_agent_call","id":"mac_spawn","call_id":"call_spawn","action":"spawn_agent","arguments":"{}","agent":{"agent_name":"/root"}}`)
	spawnResult := json.RawMessage(`{"type":"multi_agent_call_output","id":"maco_spawn","call_id":"call_spawn","action":"spawn_agent","output":[{"type":"output_text","text":"spawned","annotations":[]}],"agent":{"agent_name":"/root"}}`)
	agentMessage := json.RawMessage(`{"type":"agent_message","id":"am_result","author":"/root/reviewer","recipient":"/root","content":[{"type":"encrypted_content","encrypted_content":"opaque_report"}],"agent":{"agent_name":"/root"}}`)
	var requests, executions atomic.Int32
	client := newAccountingClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, responses.MultiAgentBeta, r.Header.Get("OpenAI-Beta"))
		var req struct {
			PreviousResponseID string          `json:"previous_response_id"`
			Input              json.RawMessage `json:"input"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		var body map[string]any
		switch requests.Add(1) {
		case 1:
			body = accountingResponse("resp_agents", "completed", "default", 100, 10,
				spawn, spawnResult,
				json.RawMessage(`{"type":"function_call","id":"fc_root","call_id":"call_root","name":"lookup","arguments":"{}","agent":{"agent_name":"/root"}}`),
				json.RawMessage(`{"type":"function_call","id":"fc_reviewer","call_id":"call_reviewer","name":"lookup","arguments":"{}","agent":{"agent_name":"/root/reviewer"}}`))
		case 2:
			require.Equal(t, "resp_agents", req.PreviousResponseID)
			require.JSONEq(t, `[
				{"type":"function_call_output","call_id":"call_root","output":"found","agent":{"agent_name":"/root"}},
				{"type":"function_call_output","call_id":"call_reviewer","output":"found","agent":{"agent_name":"/root/reviewer"}}
			]`, string(req.Input))
			body = accountingResponse("resp_synthesis", "completed", "default", 200, 20, agentMessage,
				json.RawMessage(`{"type":"message","id":"msg_final","role":"assistant","status":"completed","phase":"final_answer","agent":{"agent_name":"/root"},"content":[{"type":"output_text","text":"Combined result","annotations":[]}]}`))
		default:
			t.Error("hosted collaboration action triggered client execution")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body["model"] = models.Default
		body["multi_agent"] = map[string]any{"enabled": true, "max_concurrent_subagents": 1}
		require.NoError(t, json.NewEncoder(w).Encode(body))
	})
	registerAccountingFunction(t, client, func(json.RawMessage) (string, error) {
		executions.Add(1)
		return "found", nil
	})

	resp, err := client.Send(&responses.Request{
		Model: models.Default, Input: "Review and combine the results", Tools: []string{"lookup"},
		MultiAgent: &responses.MultiAgentConfig{Enabled: true, MaxConcurrentSubagents: 1},
	})

	require.NoError(t, err)
	require.EqualValues(t, 2, requests.Load())
	require.EqualValues(t, 2, executions.Load())
	require.Equal(t, "Combined result", resp.FirstText())
	require.Len(t, resp.Calls, 2)
	require.JSONEq(t, string(spawn), resp.Calls[0].Outputs[0].String())
	require.JSONEq(t, string(spawnResult), resp.Calls[0].Outputs[1].String())
	require.JSONEq(t, string(agentMessage), resp.Calls[1].Outputs[0].String())
	require.Equal(t, "/root/reviewer", resp.Calls[0].Outputs[3].Agent.AgentName)
	message, ok := resp.ParsedOutputs[len(resp.ParsedOutputs)-1].(output.Message)
	require.True(t, ok)
	require.Equal(t, "final_answer", message.Phase)
	require.Equal(t, "/root", message.Agent.AgentName)
	usage, usageErr := resp.TotalUsage()
	require.NoError(t, usageErr)
	require.Equal(t, 300, usage.InputTokens)
	require.Equal(t, 30, usage.OutputTokens)
}

func TestSendToolContinuationUsesConversationWithoutPreviousResponse(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	client := newAccountingClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Conversation       string          `json:"conversation"`
			PreviousResponseID string          `json:"previous_response_id"`
			Input              json.RawMessage `json:"input"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "conv_lookup", req.Conversation)
		require.Empty(t, req.PreviousResponseID)
		var body map[string]any
		switch requests.Add(1) {
		case 1:
			body = accountingResponse("resp_lookup", "completed", "default", 100, 10, json.RawMessage(accountingFunctionCall))
		case 2:
			require.JSONEq(t, `[{"type":"function_call_output","call_id":"call_lookup","output":"found"}]`, string(req.Input))
			body = accountingResponse("resp_answer", "completed", "default", 120, 12, accountingMessage("Found in this conversation"))
		default:
			t.Error("conversation continuation repeated a completed tool")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body["model"] = models.Default
		require.NoError(t, json.NewEncoder(w).Encode(body))
	})
	registerAccountingFunction(t, client, func(json.RawMessage) (string, error) { return "found", nil })

	resp, err := client.Send(&responses.Request{
		Model: models.Default, Input: "Look up the value", Tools: []string{"lookup"}, Conversation: "conv_lookup",
	})

	require.NoError(t, err)
	require.EqualValues(t, 2, requests.Load())
	require.Equal(t, "Found in this conversation", resp.FirstText())
	require.Len(t, resp.Calls, 2)
}

func TestSendProgramReturnsPendingApplicationWork(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		tool tools.Tool
		item string
	}{
		{"MCP approval", tools.Tool{Type: "mcp", ServerLabel: "inventory", ServerURL: "https://example.com/mcp"}, `{"type":"mcp_approval_request","id":"approval","name":"lookup","server_label":"inventory","arguments":"{}"}`},
		{"local shell", tools.Tool{Type: "shell", Environment: map[string]any{"type": "local"}}, `{"type":"shell_call","id":"shell","call_id":"call_shell","status":"completed","action":{"commands":["pwd"]}}`},
		{"patch", tools.Tool{Type: "apply_patch"}, `{"type":"apply_patch_call","id":"patch","call_id":"call_patch","status":"completed","operation":{"type":"create_file","path":"note.txt","diff":"+note"}}`},
		{"computer", tools.Tool{Type: "computer_use_preview", DisplayWidth: 800, DisplayHeight: 600, Environment: "browser"}, `{"type":"computer_call","id":"computer","call_id":"call_computer","action":{"type":"screenshot"},"status":"completed"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			client := newAccountingClient(t, func(w http.ResponseWriter, r *http.Request) {
				if requests.Add(1) != 1 {
					t.Error("continued without the application result or approval")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				body := accountingResponse("paused", "completed", "default", 100, 10,
					json.RawMessage(`{"type":"program","id":"program","call_id":"program_call","code":"await tool();","fingerprint":"replay"}`), json.RawMessage(tc.item))
				body["model"] = models.Default
				require.NoError(t, json.NewEncoder(w).Encode(body))
			})
			require.NoError(t, client.Tools.RegisterTool(tools.Tool{Type: "programmatic_tool_calling"}))
			require.NoError(t, client.Tools.RegisterTool(tc.tool))
			resp, err := client.Send(&responses.Request{Model: models.Default, Input: "Run the program", Tools: []string{"programmatic_tool_calling", tc.tool.Type}})
			require.NoError(t, err)
			require.EqualValues(t, 1, requests.Load())
			require.Len(t, resp.Calls, 1)
			require.Len(t, resp.Outputs, 2)
			require.JSONEq(t, tc.item, resp.Outputs[1].String())
		})
	}
}

func TestSendProgramContinuesAfterResolvedHostedTool(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	client := newAccountingClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		switch requests.Add(1) {
		case 1:
			body = accountingResponse("resolved", "completed", "default", 100, 10,
				json.RawMessage(`{"type":"program_output","id":"program","call_id":"program_call","result":"done","status":"completed"}`),
				json.RawMessage(`{"type":"shell_call","id":"shell","call_id":"shell_call","status":"completed","action":{"commands":["pwd"]}}`),
				json.RawMessage(`{"type":"shell_call_output","call_id":"shell_call","output":[]}`))
		case 2:
			var req struct {
				PreviousResponseID string `json:"previous_response_id"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			require.Equal(t, "resolved", req.PreviousResponseID)
			body = accountingResponse("final", "completed", "default", 100, 10, accountingMessage("Finished"))
		default:
			t.Error("continued past the final answer")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		body["model"] = models.Default
		require.NoError(t, json.NewEncoder(w).Encode(body))
	})
	require.NoError(t, client.Tools.RegisterTool(tools.Tool{Type: "programmatic_tool_calling"}))
	require.NoError(t, client.Tools.RegisterTool(tools.Tool{Type: "shell", Environment: map[string]any{"type": "container_auto"}}))
	resp, err := client.Send(&responses.Request{Model: models.Default, Input: "Run the program", Tools: []string{"programmatic_tool_calling", "shell"}})
	require.NoError(t, err)
	require.EqualValues(t, 2, requests.Load())
	require.Equal(t, "Finished", resp.FirstText())
	require.Len(t, resp.Calls, 2)
}
