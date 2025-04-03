package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/nikolayk812/sqlcpp/internal/template"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"log"
	"os"
	"strings"
)

func main() {

	data := template.BuildRepositoryDataMap("cart")

	engine := template.NewEngine("internal/template/data/repository_test.tmpl", data)

	request, err := engine.Execute()
	if err != nil {
		log.Fatal(err)
	}

	//llm, err := ollama.New(
	//	ollama.WithModel("llama3.2"),
	//	ollama.WithServerURL("http://localhost:11434"))
	//if err != nil {
	//	log.Fatal(err)
	//}

	llm, err := openai.New(
		openai.WithModel("gpt-4o"),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	content := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, request),
	}

	fmt.Println("calling LLM")

	// The response from the model happens when the model finishes processing the input, which it's usually slow.
	completion, err := llm.GenerateContent(context.Background(), content)
	if err != nil {
		log.Fatal(err)
	}

	var response strings.Builder

	for _, choice := range completion.Choices {
		if choice == nil {
			continue
		}

		response.WriteString(choice.Content)
		// fmt.Println(choice.Content)

		if choice.StopReason != "" && choice.StopReason != "stop" {
			fmt.Println("Stop reason: ", choice.StopReason)
		}
	}

	codeResponse, err := extractGoCode(response.String())
	if err != nil {
		log.Fatal(err)
	}

	log.Print(codeResponse)
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
