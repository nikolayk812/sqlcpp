package steps

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"
)

type ReferenceCode struct {
	templateFilename string
	references       []CodeReference
	referenceCodeKey string
}

func NewReferenceCode(templateFilename string, referenceCodeKey string, references []CodeReference) (ReferenceCode, error) {
	var s ReferenceCode

	if templateFilename == "" {
		return s, fmt.Errorf("templateFilename is empty")
	}
	if referenceCodeKey == "" {
		return s, fmt.Errorf("referenceCodeKey is empty")
	}
	if len(references) == 0 {
		return s, fmt.Errorf("references are empty")
	}

	return ReferenceCode{
		templateFilename: templateFilename,
		references:       references,
		referenceCodeKey: referenceCodeKey,
	}, nil
}

func (s ReferenceCode) Name() string {
	return "reference_code"
}

func (s ReferenceCode) Run(_ context.Context, dataCtx DataContext) error {
	var templateData []CodeReference

	for _, ref := range s.references {
		if ref.Code != "" {
			templateData = append(templateData, ref)
			continue
		}

		// resolve reference code if empty
		fileContent, err := os.ReadFile(ref.Filepath)
		if err != nil {
			return fmt.Errorf("os.ReadFile[%s]: %w", ref.Filepath, err)
		}

		ref.Code = string(fileContent)
		templateData = append(templateData, ref)
	}

	tmpl, err := template.ParseFiles(s.templateFilename)
	if err != nil {
		return fmt.Errorf("template.ParseFiles[%s]: %w", s.templateFilename, err)
	}

	var output strings.Builder
	if err := tmpl.Execute(&output, templateData); err != nil {
		return fmt.Errorf("tmpl.Execute: %w", err)
	}

	dataCtx[s.referenceCodeKey] = output.String()

	return nil
}

type CodeReference struct {
	Filepath    string
	Description string
	Type        string
	Code        string
}
