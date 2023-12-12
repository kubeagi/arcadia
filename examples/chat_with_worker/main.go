package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	llm, err := openai.New(
		openai.WithToken("fake"),
		// update base url to fastchat api server
		openai.WithBaseURL("http://arcadia-fastchat.172.22.96.167.nip.io/v1"),
		openai.WithModel("fb219b5f-8f3e-49e1-8d5b-f0c6da481186"),
	)
	if err != nil {
		log.Fatal(err)
	}
	// llm call
	completion, err := llm.Call(context.Background(), "The first man to walk on the moon",
		llms.WithTemperature(0.8),
		llms.WithStopWords([]string{"Armstrong"}),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(completion)

	// create embeddings
	embeddings, err := llm.CreateEmbedding(context.Background(), []string{"hello,this is a embedding call"})
	if err != nil {
		panic(err)
	}
	fmt.Println(embeddings)
}
