package debate

import (
	"context"
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
)

var client = anthropic.NewClient()

// call sends a single message to the specified model and returns the text response.
func call(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 2048,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("api call failed: %w", err)
	}
	if len(msg.Content) == 0 {
		return "", fmt.Errorf("empty response from model %s", model)
	}
	return msg.Content[0].Text, nil
}
