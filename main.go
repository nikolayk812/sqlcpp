package main

import (
	"context"
	"github.com/nikolayk812/sqlcpp/internal/generator"
	"github.com/tmc/langchaingo/llms/openai"
	"log"
	"os"
)

func main() {
	llm, err := openai.New(
		openai.WithModel("gpt-4o"),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	pipeline, err := generator.NewPipeline(llm)
	if err != nil {
		log.Fatal(err)
	}

	if err := pipeline.Run(context.Background()); err != nil {
		log.Fatal(err)
	}

}
