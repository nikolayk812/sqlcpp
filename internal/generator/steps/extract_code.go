package steps

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type ExtractCode struct {
	llmResponseKey   string
	generatedCodeKey string
}

func NewExtractCode(llmResponseKey, generatedCodeKey string) (ExtractCode, error) {
	var s ExtractCode

	if llmResponseKey == "" {
		return s, fmt.Errorf("llmResponseKey is empty")
	}
	if generatedCodeKey == "" {
		return s, fmt.Errorf("generatedCodeKey is empty")
	}

	return ExtractCode{
		llmResponseKey:   llmResponseKey,
		generatedCodeKey: generatedCodeKey,
	}, nil
}

func (s ExtractCode) Name() string {
	return "extract_code"
}

func (s ExtractCode) Run(_ context.Context, dataCtx DataContext) error {
	llmResponse, ok := dataCtx[s.llmResponseKey]
	if !ok {
		return fmt.Errorf("key[%s] not found in data context", s.llmResponseKey)
	}

	code, err := extractGoCode(llmResponse)
	if err != nil {
		return fmt.Errorf("extractGoCode: %w", err)
	}

	dataCtx[s.generatedCodeKey] = code

	return nil
}

func extractGoCode(text string) (string, error) {
	startMarker := "```go"
	endMarker := "```"

	start := strings.Index(text, startMarker)
	if start == -1 {
		return "", errors.New("start marker not found")
	}
	start += len(startMarker) // Move past the start marker

	end := strings.Index(text[start:], endMarker)
	if end == -1 {
		return "", errors.New("end marker not found")
	}

	return text[start : start+end], nil
}
