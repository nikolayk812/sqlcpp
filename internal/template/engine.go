package template

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

type Engine struct {
	filename string
	data     map[string]string
}

func NewEngine(filename string, data map[string]string) *Engine {
	e := &Engine{
		filename: filename,
		data:     data,
	}

	return e
}

func (e *Engine) Execute() (string, error) {
	resolvedData, err := e.resolveFiles()
	if err != nil {
		return "", fmt.Errorf("e.resolveFiles: %w", err)
	}

	tmpl, err := template.ParseFiles(e.filename)
	if err != nil {
		return "", fmt.Errorf("template.ParseFiles: %w", err)
	}

	var output strings.Builder
	if err := tmpl.Execute(&output, resolvedData); err != nil {
		return "", fmt.Errorf("tmpl.Execute: %w", err)
	}

	outputStr := output.String()

	// Check if there are any unresolved placeholders
	if strings.Contains(outputStr, "<no value>") {
		return "", fmt.Errorf("unresolved placeholders found in the template")
	}

	return outputStr, nil
}

func (e *Engine) resolveFiles() (map[string]string, error) {
	result := make(map[string]string, len(e.data))

	for k, v := range e.data {
		if !strings.HasSuffix(v, ".go") {
			result[k] = v
			continue
		}

		content, err := os.ReadFile(v)
		if err != nil {
			return nil, fmt.Errorf("os.ReadFile[%s][%s]: %w", k, v, err)
		}
		result[k] = string(content)
	}

	return result, nil
}
