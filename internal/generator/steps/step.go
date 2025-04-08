package steps

import (
	"context"
)

type Step interface {
	Name() string
	Run(ctx context.Context, dataCtx DataContext) error
}

type DataContext map[string]string

// validate: compile project, run tests, AST, linters?
