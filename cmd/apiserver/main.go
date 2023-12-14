package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"gptapiserver/pkg/openai"

	"github.com/caarlos0/env/v7"
	"github.com/valyala/fasthttp"
)

var cfg struct {
	AccessKey        string  `env:"GPT_KEY,required"`
	OpenAIAPIKey     string  `env:"OPENAI_API_KEY,required"`
	ModelTemperature float32 `env:"MODEL_TEMPERATURE" envDefault:"1.0"`
}

var openAIClient *openai.Client

func main() {
	var port string

	flag.StringVar(&port, "port", "8080", "port to run the server on")
	flag.Parse()

	if err := env.Parse(&cfg); err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}

	openAIClient = openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	server := &fasthttp.Server{
		Handler: requestHandler,
	}

	fmt.Printf("Server is running on http://localhost:%s\n", port)
	if err := server.ListenAndServe(":" + port); err != nil {
		log.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/api/gpt":
		handleGPTRequest(ctx)
	default:
		ctx.Error("Unsupported path", fasthttp.StatusNotFound)
	}
}

func handleGPTRequest(ctx *fasthttp.RequestCtx) {
	key := string(ctx.FormValue("key"))
	if key != cfg.AccessKey {
		ctx.Error("incorrect key", fasthttp.StatusBadRequest)
		return
	}

	command := string(ctx.FormValue("command"))
	if command == "" {
		ctx.Error("Command is required", fasthttp.StatusBadRequest)
		return
	}

	response, err := askGPT(command)
	if err != nil {
		ctx.SetContentType("application/json")
		ctx.SetStatusCode(fasthttp.StatusOK)
		fmt.Fprintf(ctx, `{"error": %q}`, err.Error())
		return
	}

	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	fmt.Fprintf(ctx, `{"answer": %q}`, response)
}

func askGPT(command string) (string, error) {
	resp, err := openAIClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: command,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("ERR: %v RESP: %v", err, resp)
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from GPT")
}
