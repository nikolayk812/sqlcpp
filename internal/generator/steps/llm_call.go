package steps

import (
	"context"
	"fmt"
	"github.com/tmc/langchaingo/llms"
	"log/slog"
	"strings"
)

type LLMCall struct {
	llm         llms.Model
	promptKey   string
	responseKey string
}

func NewLLMCall(llm llms.Model, promptKey, llmResponseKey string) (LLMCall, error) {
	var s LLMCall

	if llm == nil {
		return s, fmt.Errorf("llm is nil")
	}
	if promptKey == "" {
		return s, fmt.Errorf("promptKey is empty")
	}
	if llmResponseKey == "" {
		return s, fmt.Errorf("llmResponseKey is empty")
	}

	return LLMCall{
		llm:         llm,
		promptKey:   promptKey,
		responseKey: llmResponseKey,
	}, nil
}

func (s LLMCall) Name() string {
	return "llm_call"
}

func (s LLMCall) Run(ctx context.Context, dataCtx DataContext) error {
	prompt, ok := dataCtx[s.promptKey]
	if !ok {
		return fmt.Errorf("key[%s] not found in data context", s.promptKey)
	}

	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	completion, err := s.llm.GenerateContent(ctx, content)
	if err != nil {
		return fmt.Errorf("llm.GenerateContent: %w", err)
	}

	var response strings.Builder
	for _, choice := range completion.Choices {
		if choice == nil {
			continue
		}

		response.WriteString(choice.Content)

		// "stop" stop reason is expected at least in OpenAI
		if choice.StopReason != "" && choice.StopReason != "stop" {
			slog.Warn("Unexpected stop reason",
				"method", "LLMCall.Run",
				"stop_reason", choice.StopReason)
		}
	}

	dataCtx[s.responseKey] = response.String()

	return nil
}
