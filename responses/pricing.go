// Package responses / pricing.go provides request previews and estimates from observed response usage.
package responses

import (
	"errors"
	"fmt"
	"strings"

	"github.com/unkn0wncode/openai/models"
)

// Cost estimates standard-tier token charges in USD. The selected Pricing may
// instead come from Pricing.ForTier. Hosted tools and regional charges are excluded.
func (u *Usage) Cost(pricing models.Pricing) (float64, error) {
	return (*models.Usage)(u).Cost(pricing)
}

// PreviewCost estimates token charges using only the model and service tier
// from req; all token counts, including cache reads and writes, come from
// assumedUsage. Populate assumedUsage.InputTokens with the result of
// CountInputTokens or a manual estimate, and supply the other assumed counts.
// The request's input and output limits are not used.
// An empty model uses models.Default. Set an explicit service tier because
// auto depends on project settings. Hosted tools, regional charges, and
// automatic follow-up requests are excluded.
func (req *Request) PreviewCost(assumedUsage *Usage) (float64, error) {
	if req == nil {
		return 0, errors.New("request is nil")
	}
	if req.ServiceTier == "" || req.ServiceTier == ServiceTierAuto {
		return 0, errors.New("preview requires an explicit service tier; auto depends on project settings")
	}
	model := req.Model
	if model == "" {
		model = models.Default
	}
	pricing, ok := models.Data[model]
	if !ok {
		return 0, fmt.Errorf("no pricing found for model %q", model)
	}
	pricing, err := pricing.ForTier(req.ServiceTier)
	if err != nil {
		return 0, err
	}
	return pricing.Cost(assumedUsage)
}

// EstimateCost estimates charges using the actual model, tier, usage, and tool
// calls in resp. It records each call's EstimatedCost and CostError, then sums
// amounts and joins errors. A non-nil error means the amount is a known subtotal.
// Estimation errors do not indicate whether execution succeeded.
func (req *Request) EstimateCost(resp *Response) (float64, error) {
	if resp == nil {
		return 0, errors.New("response is nil")
	}
	var total float64
	var errs []error
	if len(resp.Calls) == 0 {
		total, resp.CostError = req.estimateCall(resp)
		errs = append(errs, resp.CostError)
	} else {
		for i := range resp.Calls {
			call := &resp.Calls[i]
			call.EstimatedCost, call.CostError = req.estimateCall(call)
			total += call.EstimatedCost
			if call.CostError != nil {
				errs = append(errs, fmt.Errorf("API call %d (response %q): %w", i+1, call.ID, call.CostError))
			}
		}
	}
	if resp.BillingIncomplete {
		errs = append(errs, errors.New("usage is unavailable for an attempted API request"))
	}
	resp.EstimatedCost = total
	resp.CostError = errors.Join(errs...)
	return resp.EstimatedCost, resp.CostError
}

func (req *Request) estimateCall(resp *Response) (float64, error) {
	toolCost, unknownInput, toolErr := req.toolCost(resp)
	if resp.Status == "queued" || resp.Status == "in_progress" {
		toolErr = errors.Join(toolErr, fmt.Errorf("response %q is still %s", resp.ID, resp.Status))
	}
	if resp.BillingIncomplete {
		toolErr = errors.Join(toolErr, errors.New("billing evidence is incomplete"))
	}
	pricing, ok := models.Data[resp.Model]
	if !ok || resp.Model == "" {
		return toolCost, errors.Join(fmt.Errorf("no pricing found for model %q", resp.Model), toolErr)
	}
	if resp.ServiceTier == "" {
		return toolCost, errors.Join(errors.New("response service tier is unavailable"), toolErr)
	}
	pricing, err := pricing.ForTier(resp.ServiceTier)
	if err != nil {
		return toolCost, errors.Join(err, toolErr)
	}
	if resp.Model == models.GPT6Astra && resp.ProcessingRegion == "eu" &&
		(resp.ServiceTier == ServiceTierFast || resp.ServiceTier == ServiceTierPriority) {
		return toolCost, errors.Join(errors.New("fast pricing is unavailable for GPT-6 Astra with EU data residency"), toolErr)
	}
	tokenCost, tokenErr := pricing.Cost(resp.Usage)
	if unknownInput && resp.Usage != nil {
		// Keep output charges at the context rate selected by the original input.
		inputUsage := *resp.Usage
		inputUsage.OutputTokens = 0
		inputCost, inputErr := pricing.Cost(&inputUsage)
		tokenErr = errors.Join(tokenErr, inputErr)
		tokenCost -= inputCost
	}
	if pricing.RegionalUplift != 0 {
		switch resp.ProcessingRegion {
		case "global":
		case "us", "eu":
			tokenCost *= 1 + pricing.RegionalUplift
		default:
			err = fmt.Errorf("regional processing charges are unavailable for region %q", resp.ProcessingRegion)
		}
	}
	return tokenCost + toolCost, errors.Join(tokenErr, err, toolErr)
}

// toolCost reads raw output discriminators, so unfamiliar outputs can be
// reported as unpriced without discarding the rest of the usage evidence.
func (req *Request) toolCost(resp *Response) (cost float64, unknownInput bool, err error) {
	var errs []error
	searchType := ""
	for _, tool := range resp.Tools {
		if tool.Type == "web_search" || tool.Type == "web_search_preview" {
			if searchType != "" && searchType != tool.Type {
				searchType = "ambiguous"
				break
			}
			searchType = tool.Type
		}
	}
	// A standalone response may omit echoed tools; use request names only when
	// they identify a built-in tool unambiguously.
	if len(resp.Tools) == 0 && req != nil {
		for _, name := range req.Tools {
			if name == "web_search" || name == "web_search_preview" {
				if searchType != "" && searchType != name {
					searchType = "ambiguous"
					break
				}
				searchType = name
			}
		}
	}
	for _, item := range resp.Outputs {
		switch item.Type {
		case "web_search_call":
			var call struct {
				ID     string `json:"id"`
				Action struct {
					Type string `json:"type"`
				} `json:"action"`
			}
			if decodeErr := item.UnmarshalToTarget(&call); decodeErr != nil {
				errs = append(errs, fmt.Errorf("decode web search usage: %w", decodeErr))
				unknownInput = true
				continue
			}
			switch call.Action.Type {
			case "open_page", "find_in_page", "find":
				continue
			case "search":
			default:
				errs = append(errs, fmt.Errorf("web search action %q has no known pricing", call.Action.Type))
				unknownInput = true
				continue
			}
			switch searchType {
			case "web_search":
				cost += 0.01
				if strings.HasPrefix(resp.Model, "gpt-4o-mini") || strings.HasPrefix(resp.Model, "gpt-4.1-mini") {
					unknownInput = true
					errs = append(errs, fmt.Errorf("web search %q: fixed search-content token billing is not itemized in usage", call.ID))
				}
			case "web_search_preview":
				switch {
				case strings.HasPrefix(resp.Model, "gpt-5"), strings.HasPrefix(resp.Model, "gpt-6"),
					strings.HasPrefix(resp.Model, "o1"), strings.HasPrefix(resp.Model, "o3"), strings.HasPrefix(resp.Model, "o4"):
					cost += 0.01
				case strings.HasPrefix(resp.Model, "gpt-4"):
					cost += 0.025
					unknownInput = true
					errs = append(errs, fmt.Errorf("web search %q: free search-content tokens are not itemized in usage", call.ID))
				default:
					unknownInput = true
					errs = append(errs, fmt.Errorf("web search preview pricing is unavailable for model %q", resp.Model))
				}
			default:
				unknownInput = true
				errs = append(errs, fmt.Errorf("web search %q: tool variant is unavailable", call.ID))
			}
		case "file_search_call":
			cost += 0.0025
			errs = append(errs, errors.New("file search storage charges are not available from response usage"))
		case "code_interpreter_call":
			errs = append(errs, errors.New("container session charges are not available from response usage"))
		case "shell_call":
			knownLocal := false
			for _, tool := range resp.Tools {
				if tool.Type != "shell" {
					continue
				}
				if env, ok := tool.Environment.(string); ok && env == "local" {
					knownLocal = true
					continue
				}
				if env, ok := tool.Environment.(map[string]any); ok && env["type"] == "local" {
					knownLocal = true
					continue
				}
			}
			if !knownLocal {
				errs = append(errs, errors.New("shell container charges are not available without local execution evidence or session billing"))
			}
		case "message", "reasoning", "compaction", "function_call", "custom_tool_call", "computer_call",
			"apply_patch_call", "local_shell_call", "mcp_call", "mcp_list_tools", "mcp_approval_request":
			// No separate OpenAI call charge for these outputs.
		default:
			errs = append(errs, fmt.Errorf("output type %q has no known tool pricing", item.Type))
		}
	}
	return cost, unknownInput, errors.Join(errs...)
}

// TotalUsage sums observed usage across this Send's API calls. Its result is
// for reporting; context thresholds and service tiers must be priced per call.
func (resp *Response) TotalUsage() (Usage, error) {
	var total Usage
	if resp == nil {
		return total, errors.New("response is nil")
	}
	calls := resp.Calls
	if len(calls) == 0 {
		calls = []Response{*resp}
	}
	var errs []error
	for i := range calls {
		if calls[i].Status == "queued" || calls[i].Status == "in_progress" {
			errs = append(errs, fmt.Errorf("API call %d (response %q) is still %s", i+1, calls[i].ID, calls[i].Status))
		}
		u := calls[i].Usage
		if u == nil {
			errs = append(errs, fmt.Errorf("API call %d (response %q): usage is unavailable", i+1, calls[i].ID))
			continue
		}
		total.InputTokens += u.InputTokens
		total.InputTokensDetails.CachedTokens += u.InputTokensDetails.CachedTokens
		total.InputTokensDetails.CacheWriteTokens += u.InputTokensDetails.CacheWriteTokens
		total.OutputTokens += u.OutputTokens
		total.OutputTokensDetails.ReasoningTokens += u.OutputTokensDetails.ReasoningTokens
		total.TotalTokens += u.TotalTokens
	}
	if resp.BillingIncomplete {
		errs = append(errs, errors.New("usage is unavailable for an attempted API request"))
	}
	return total, errors.Join(errs...)
}
