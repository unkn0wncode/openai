// Command getmodels fetches OpenAI model pricing/limit data from models.dev
// and rewrites models/text.go with the refreshed information. Only the code section
// below the "GENERATED" marker is modified; all hand-written code above it remains
// untouched.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"
)

const (
	modelsAPI      = "https://models.dev/api.json"
	providerOpenAI = "openai"
	modelsFile     = "text.go"
)

const dataBlockFormat = `// CODE BELOW THIS LINE IS GENERATED. ONLY EDIT IF YOU KNOW HOW.

// Data contains price per 1 token for each model, separately for input and output, and token limits.
// Note that pricing page https://openai.com/pricing lists price per 1k tokens and here it's per 1 token.
// The "" denotes default values.
var Data = map[string]struct {
	PriceIn       float64
	PriceCachedIn float64
	PriceOut      float64
	LimitContext  int
	LimitOutput   int
}{
	// Zeroes in the end of prices are added to align it and make it easier to read.
	// Can be read as "0.00000450 = 4.5 micro dollars per token = $4.50 per 1M tokens".
	"":                          {0.00000000, 0.00000000, 0.00000000, 4096, 4096},
	{{- define "modelList"}}{{range .}}{{with .ConstantName}}{{.}}{{else}}"{{.ID}}"{{end}}: { {{- .PriceInStr}}, {{.PriceCachedInStr}}, {{.PriceOutStr}}, {{.LimitContext}}, {{.LimitOutput -}} },
	{{end}}{{end}}
	{{template "modelList" .Models}}

	// Deprecated or unused models
	{{template "modelList" .Deprecated}}
}
`

var (
	logger = slog.Default()

	// reConstantDefinition matches a line declaring a model constant, e.g.
	//   GPT3Turbo = "gpt-3.5-turbo"
	reConstantDefinition = regexp.MustCompile(`^\s+(\w+)\s+=\s"([-\.\w]+)"$`)

	// reConstantData matches a line assigning pricing data to a previously
	// declared constant.
	reConstantData = regexp.MustCompile(`^\s+(\w+):\s+\{([\.\d]+),\s*([\.\d]+),\s*([\.\d]+),\s*(\d+),\s*(\d+)\},`)

	// reLiteralData matches a pricing line that uses a model ID literal (no
	// constant).
	reLiteralData = regexp.MustCompile(`^\s+"([-\.\w]+)":\s+\{([\.\d]+),\s*([\.\d]+),\s*([\.\d]+),\s*(\d+),\s*(\d+)\},`)

	// outputBuilder accumulates the contents of the rewritten models/text.go file.
	outputBuilder = strings.Builder{}
)

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

type outputData struct {
	ID           modelID
	ConstantName string

	IsDeprecated bool

	PriceInStr       string
	PriceCachedInStr string
	PriceOutStr      string
	LimitContext     int
	LimitOutput      int
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

// parseModelsFile parses the models/text.go file and returns a list of outputData
// structs, one for each found model.
func parseModelsFile() ([]outputData, error) {
	content, err := os.ReadFile(modelsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read models file: %w", err)
	}
	lines := strings.Split(string(content), "\n")
	logger.Info(fmt.Sprintf("Got %d lines from models file", len(lines)))

	var data []outputData
	endOfConstants := false
	isDeprecated := false
	generated := false
	for _, line := range lines {
		if !generated && strings.Contains(line, "GENERATED") {
			generated = true
		}
		if !generated {
			outputBuilder.WriteString(line + "\n")
		}

		if matches := reConstantDefinition.FindStringSubmatch(line); len(matches) > 0 && !endOfConstants {
			// Found a constant declaration (e.g. GPT3Turbo = "gpt-3.5-turbo").
			logger.Info(fmt.Sprintf("Found constant: %s = %s", matches[1], matches[2]))
			data = append(data, outputData{
				ConstantName: matches[1],
				ID:           modelID(matches[2]),
			})
		} else if matches := reConstantData.FindStringSubmatch(line); len(matches) > 0 {
			// Found pricing data that references a constant.
			logger.Info(fmt.Sprintf("Found data for constant %s: %v", matches[1], matches[2:]))
			foundIndex := slices.IndexFunc(data, func(data outputData) bool {
				return data.ConstantName == matches[1]
			})
			if foundIndex == -1 {
				logger.Error(fmt.Sprintf("Unknown constant %s", matches[1]))
				continue
			}
			data[foundIndex].PriceInStr = matches[2]
			data[foundIndex].PriceCachedInStr = matches[3]
			data[foundIndex].PriceOutStr = matches[4]
			data[foundIndex].LimitContext, _ = strconv.Atoi(matches[5])
			data[foundIndex].LimitOutput, _ = strconv.Atoi(matches[6])
		} else if matches := reLiteralData.FindStringSubmatch(line); len(matches) > 0 {
			// Found pricing data that references a model literal.
			logger.Info(fmt.Sprintf("Found data for literal %s: %v", matches[1], matches[2:]))
			foundIndex := slices.IndexFunc(data, func(data outputData) bool {
				return data.ID == modelID(matches[1])
			})
			if foundIndex == -1 {
				data = append(data, outputData{
					ID:               modelID(matches[1]),
					PriceInStr:       matches[2],
					PriceCachedInStr: matches[3],
					PriceOutStr:      matches[4],
					IsDeprecated:     isDeprecated,
				})
				foundIndex = len(data) - 1
				data[foundIndex].LimitContext, _ = strconv.Atoi(matches[5])
				data[foundIndex].LimitOutput, _ = strconv.Atoi(matches[6])
				continue
			}
			data[foundIndex].PriceInStr = matches[2]
			data[foundIndex].PriceCachedInStr = matches[3]
			data[foundIndex].PriceOutStr = matches[4]
			data[foundIndex].LimitContext, _ = strconv.Atoi(matches[5])
			data[foundIndex].LimitOutput, _ = strconv.Atoi(matches[6])
		} else if strings.Contains(line, "Completion models") {
			endOfConstants = true
		} else if strings.Contains(line, "// Deprecated or unused models") {
			isDeprecated = true
		}
	}

	return data, nil
}

// formatWithGoFmt pipes the given Go source through `gofmt` so that the generated
// section follows standard formatting.
func formatWithGoFmt(data string) (string, error) {
	cmd := exec.Command("bash", "-c", "cat - | gofmt")
	cmd.Stdin = strings.NewReader(data)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func main() {
	dataTemplate, err := template.New("data").Parse(dataBlockFormat)
	if err != nil {
		panic(err)
	}

	formattedData, err := parseModelsFile()
	if err != nil {
		panic(err)
	}

	apiData, err := fetchOpenAIData()
	if err != nil {
		panic(err)
	}

	for _, model := range apiData.Models {
		foundIndex := slices.IndexFunc(formattedData, func(data outputData) bool {
			return data.ID == model.ID
		})
		if foundIndex != -1 {
			formattedData[foundIndex] = outputData{
				ID:           model.ID,
				ConstantName: formattedData[foundIndex].ConstantName,

				// prices are formatted as USD/token float64 with 8 decimal places
				PriceInStr:       fmt.Sprintf("%.8f", model.Cost.Input*1e-6),
				PriceCachedInStr: fmt.Sprintf("%.8f", model.Cost.CacheRead*1e-6),
				PriceOutStr:      fmt.Sprintf("%.8f", model.Cost.Output*1e-6),

				LimitContext: model.Limit.Context,
				LimitOutput:  model.Limit.Output,
			}
			continue
		}

		formattedData = append(formattedData, outputData{
			ID:           model.ID,
			ConstantName: "",

			PriceInStr:       fmt.Sprintf("%.8f", model.Cost.Input*1e-6),
			PriceCachedInStr: fmt.Sprintf("%.8f", model.Cost.CacheRead*1e-6),
			PriceOutStr:      fmt.Sprintf("%.8f", model.Cost.Output*1e-6),

			LimitContext: model.Limit.Context,
			LimitOutput:  model.Limit.Output,
		})
	}

	groupedData := struct {
		Models     []outputData
		Deprecated []outputData
	}{}
	for _, data := range formattedData {
		if data.IsDeprecated {
			groupedData.Deprecated = append(groupedData.Deprecated, data)
		} else {
			groupedData.Models = append(groupedData.Models, data)
		}
	}

	var dataBlock bytes.Buffer
	err = dataTemplate.Execute(&dataBlock, groupedData)
	if err != nil {
		panic(err)
	}
	logger.Info("Template executed")

	output, err := formatWithGoFmt(dataBlock.String())
	if err != nil {
		fmt.Println(dataBlock.String())
		panic(err)
	}
	logger.Info("Gofmt executed")

	outputBuilder.WriteString(output)

	err = os.WriteFile(modelsFile, []byte(outputBuilder.String()), 0644)
	if err != nil {
		panic(err)
	}
	logger.Info("File written, done")
}
