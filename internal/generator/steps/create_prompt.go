package steps

import (
	"context"
	"fmt"
	"strings"
	"text/template"
)

type CreatePrompt struct {
	templateFilename  string
	toGenerateCodeKey string
	referenceCodeKey  string
	promptKey         string
}

func NewCreatePrompt(templateFilename, toGenerateCodeKey, referenceCodeKey, promptKey string) (CreatePrompt, error) {
	var s CreatePrompt

	if templateFilename == "" {
		return s, fmt.Errorf("templateFilename is empty")
	}
	if toGenerateCodeKey == "" {
		return s, fmt.Errorf("toGenerateCodeKey is empty")
	}
	if referenceCodeKey == "" {
		return s, fmt.Errorf("referenceCodeKey is empty")
	}
	if promptKey == "" {
		return s, fmt.Errorf("promptKey is empty")
	}

	return CreatePrompt{
		templateFilename:  templateFilename,
		toGenerateCodeKey: toGenerateCodeKey,
		referenceCodeKey:  referenceCodeKey,
		promptKey:         promptKey,
	}, nil
}

func (s CreatePrompt) Name() string {
	return "create_prompt"
}

func (s CreatePrompt) Run(_ context.Context, dataCtx DataContext) error {
	toGenerateCode, ok := dataCtx[s.toGenerateCodeKey]
	if !ok {
		return fmt.Errorf("key[%s] not found in data context", s.toGenerateCodeKey)
	}

	referenceCode, ok := dataCtx[s.referenceCodeKey]
	if !ok {
		return fmt.Errorf("key[%s] not found in data context", s.referenceCodeKey)
	}

	tmpl, err := template.ParseFiles(s.templateFilename)
	if err != nil {
		return fmt.Errorf("template.ParseFiles[%s]: %w", s.templateFilename, err)
	}

	templateData := map[string]string{
		"DomainModelName": "cart",
		"ToGenerateCode":  toGenerateCode,
		"ReferenceCode":   referenceCode,
	}

	var output strings.Builder
	if err := tmpl.Execute(&output, templateData); err != nil {
		return fmt.Errorf("tmpl.Execute: %w", err)
	}

	dataCtx[s.promptKey] = output.String()

	return nil
}
