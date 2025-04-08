package steps

import (
	"context"
	"fmt"
	"os"
)

type WriteFile struct {
	filename         string
	generatedCodeKey string
}

func NewWriteStep(filename, generatedCodeKey string) (WriteFile, error) {
	var s WriteFile

	if filename == "" {
		return s, fmt.Errorf("filename is empty")
	}
	if generatedCodeKey == "" {
		return s, fmt.Errorf("generatedCodeKey is empty")
	}

	return WriteFile{
		filename:         filename,
		generatedCodeKey: generatedCodeKey,
	}, nil
}

func (s WriteFile) Name() string {
	return "write_file"
}

func (s WriteFile) Run(_ context.Context, dataCtx DataContext) error {
	data, ok := dataCtx[s.generatedCodeKey]
	if !ok {
		return fmt.Errorf("key[%s] not found in data context", s.generatedCodeKey)
	}

	if err := os.WriteFile(s.filename, []byte(data), 0644); err != nil {
		return fmt.Errorf("os.WriteFile: %w", err)
	}

	return nil
}
