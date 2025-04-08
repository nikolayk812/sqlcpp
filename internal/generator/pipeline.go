package generator

import (
	"context"
	"fmt"
	"github.com/nikolayk812/sqlcpp/internal/generator/steps"
	"github.com/tmc/langchaingo/llms"
)

type Pipeline struct {
	steps   []steps.Step
	dataCtx steps.DataContext
}

func NewPipeline(llm llms.Model) (Pipeline, error) {
	var p Pipeline

	if llm == nil {
		return p, fmt.Errorf("llm is nil")
	}

	pSteps, err := buildSteps(llm)
	if err != nil {
		return p, fmt.Errorf("buildSteps: %w", err)
	}

	return Pipeline{
		steps:   pSteps,
		dataCtx: make(map[string]string),
	}, nil
}

func (p Pipeline) Run(ctx context.Context) error {
	for idx, step := range p.steps {
		if err := step.Run(ctx, p.dataCtx); err != nil {
			return fmt.Errorf("step.Run[%d][%s]: %w", idx, step.Name(), err)
		}
	}

	return nil
}

func buildSteps(llm llms.Model) ([]steps.Step, error) {
	const (
		toGenerateCodeKey = "to_generate_code"
		referenceCodeKey  = "reference_code"
		promptKey         = "prompt"
		llmResponseKey    = "llm_response"
		generatedCodeKey  = "generated_code"
	)

	var results []steps.Step

	step0, err := steps.NewReferenceCode("internal/template/data/reference_code.tmpl", toGenerateCodeKey,
		[]steps.CodeReference{
			{
				Filepath:    "internal/domain/cart.go",
				Description: "Cart domain model",
			},
			{
				Filepath:    "internal/domain/money.go",
				Description: "Money domain model",
			},

			{
				Filepath:    "internal/db/cart.sql.go",
				Description: "sqlc generated cart queries",
			},

			{
				Filepath:    "internal/port/cart_port.go",
				Description: "Cart port interface. You have to generate implementation for it",
			},
		})
	if err != nil {
		return nil, fmt.Errorf("step.NewReferenceCode: %w", err)
	}
	results = append(results, step0)

	step1, err := steps.NewReferenceCode("internal/template/data/reference_code.tmpl", referenceCodeKey,
		[]steps.CodeReference{
			{
				Filepath:    "internal/db/models.go",
				Description: "Reference: sqlc generated all records",
			},

			{
				Filepath:    "internal/db/order.sql.go",
				Description: "Reference: sqlc generated order queries",
			},

			// TODO: special step to read all domain models!?
			{
				Filepath:    "internal/domain/order.go",
				Description: " Reference: Order domain model",
			},
			{
				Filepath:    "internal/domain/order_status.go",
				Description: "Reference: Order status domain model",
			},
			{
				Filepath:    "internal/domain/order_filter.go",
				Description: "Reference: Order filter domain model",
			},

			{
				Filepath:    "internal/port/order_port.go",
				Description: "Reference: Order port interface",
			},
			{
				Filepath:    "internal/repository/order_repository.go",
				Description: "Reference: Order repository",
			},
		})
	if err != nil {
		return nil, fmt.Errorf("step.NewReferenceCode: %w", err)
	}
	results = append(results, step1)

	step2, err := steps.NewCreatePrompt("internal/template/data/repository.tmpl", toGenerateCodeKey, referenceCodeKey, promptKey)
	if err != nil {
		return nil, fmt.Errorf("step.NewCreatePrompt: %w", err)
	}
	results = append(results, step2)

	step3, err := steps.NewLLMCall(llm, promptKey, llmResponseKey)
	if err != nil {
		return nil, fmt.Errorf("step.NewLLMCall: %w", err)
	}
	results = append(results, step3)

	step4, err := steps.NewExtractCode(llmResponseKey, generatedCodeKey)
	if err != nil {
		return nil, fmt.Errorf("step.NewExtractCode: %w", err)
	}
	results = append(results, step4)

	step5, err := steps.NewWriteStep("internal/repository/cart_repository.go", generatedCodeKey)
	if err != nil {
		return nil, fmt.Errorf("step.NewWriteStep: %w", err)
	}
	results = append(results, step5)

	return results, nil
}
