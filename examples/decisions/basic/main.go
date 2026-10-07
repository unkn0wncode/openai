package main

import (
	"context"
	"fmt"
	"os"

	"github.com/unkn0wncode/openai"
	"github.com/unkn0wncode/openai/decisions"
)

func main() {
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		panic("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(token)

	decision, err := client.Decisions.Send(context.Background(), &decisions.Request{
		Input: decisions.TextInput("Export fails in Safari but works in Chrome. I need this report for a meeting in an hour."),
		Questions: []decisions.Question{
			{
				Type:         decisions.QuestionTypePredicate,
				Name:         "bug",
				Instructions: "Does the customer report a software defect?",
			},
			{
				Type:         decisions.QuestionTypeChoice,
				Name:         "department",
				Instructions: "Which department should handle this request?",
				Choices: []decisions.Choice{
					{Value: "billing", Description: "Payments, invoices, and refunds."},
					{Value: "technical", Description: "Problems using the product."},
					{Value: "other", Description: "Requests outside these categories."},
				},
			},
			{
				Type:         decisions.QuestionTypeScore,
				Name:         "severity",
				Instructions: "How severe is this issue?",
				Levels: []decisions.Level{
					{Label: "Cosmetic", Description: "Appearance only; no lost functionality."},
					{Label: "Workaround available", Description: "A task fails, but another way works."},
					{Label: "Fully blocked", Description: "A task fails with no workaround."},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	for _, name := range []string{"bug", "department", "severity"} {
		answer, ok := decision.Answer(name)
		switch {
		case !ok:
			fmt.Printf("%s: no answer\n", name)
		case answer.Type == decisions.AnswerTypeRefusal:
			fmt.Printf("%s: refused\n", name)
		case answer.Type == decisions.QuestionTypePredicate:
			fmt.Printf("%s: probability %.2f\n", name, answer.Probability)
		case answer.Type == decisions.QuestionTypeChoice:
			fmt.Printf("%s: %v (confidence %.2f)\n", name, answer.Choice, answer.Confidence)
		case answer.Type == decisions.QuestionTypeScore:
			fmt.Printf("%s: score %.2f (confidence %.2f)\n", name, answer.Score, answer.Confidence)
		}
	}

	cost, err := decision.EstimateCost()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Estimated cost: $%.9f\n", cost)
}
