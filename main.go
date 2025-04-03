package main

import (
	"context"
	"fmt"
	"github.com/nikolayk812/sqlcpp/internal/template"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"log"
)

func main() {

	data := template.BuildRepositoryDataMap("cart")

	engine := template.NewEngine("internal/template/data/repository.tmpl", data)

	output, err := engine.Execute()
	if err != nil {
		log.Fatal(err)
	}

	llm, err := ollama.New(
		ollama.WithModel("llama3.2"),
		ollama.WithServerURL("http://localhost:11434"))
	if err != nil {
		log.Fatal(err)
	}

	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, output),
	}

	fmt.Println("calling LLM")

	// The response from the model happens when the model finishes processing the input, which it's usually slow.
	completion, err := llm.GenerateContent(context.Background(), content)
	if err != nil {
		log.Fatal(err)
	}

	for _, choice := range completion.Choices {
		if choice == nil {
			continue
		}
		fmt.Println(choice.Content)

		if choice.StopReason != "" {
			fmt.Println("Stop reason: ", choice.StopReason)
		}
	}

	// log.Println(output)
}
