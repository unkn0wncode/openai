package chat

import (
	"macbot/commands"
	"macbot/framework"
	"macbot/util"

	"github.com/unkn0wncode/brocord"
)

// Register commands.
func init() {
	commands.Register(
		framework.Command{
			Name:                     "gpt",
			Function:                 GPTPrompt,
			Aliases:                  []string{},
			Permission:               "admin.test",
			Description:              "sends given text to ChatGPT, writes response",
			DefaultMemberPermissions: &commands.PermAdministrator,
			SlashCommand:             true,
			Options: []*brocord.ApplicationCommandOption{
				{
					Type:        brocord.ApplicationCommandOptionString,
					Name:        "prompt",
					Description: "what do you say to an AI",
					Required:    true,
				},
			},
		},
	)
}

// GPTPrompt sends a sumple prompt without any context to AI and responds with an AI reply.
func GPTPrompt(ctx *framework.Context) {
	var prompt string
	for _, opt := range ctx.Options() {
		switch opt.Name {
		case "prompt":
			prompt = opt.StringValue()
		}
	}

	response, err := SinglePrompt(prompt, ctx.User.ID)
	if err != nil {
		err = ctx.Respond(&brocord.InteractionResponse{
			Type: brocord.InteractionResponseChannelMessageWithSource,
			Data: &brocord.InteractionResponseData{
				Content: "Error: " + err.Error(),
				Flags:   brocord.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			util.PatternText(framework.LogMainStd, "dCantRespondInteraction", err)
			return
		}
		return
	}

	err = ctx.Respond(&brocord.InteractionResponse{
		Type: brocord.InteractionResponseChannelMessageWithSource,
		Data: &brocord.InteractionResponseData{
			Content: response,
		},
	})
	if err != nil {
		util.PatternText(framework.LogMainStd, "dCantRespondInteraction", err)
		return
	}
}
