// OpenAI native passthrough through BlockRun, paid per call in USDC (x402).
//
// Run with a funded Base wallet on the gateway:
//
//	export BLOCKRUN_WALLET_KEY=0x...   # or ~/.blockrun/.session
//	go run ./examples/openai
package main

import (
	"context"
	"fmt"
	"log"

	vip "github.com/BlockRunAI/blockrun-llm-go-vip"
	"github.com/openai/openai-go"
)

func main() {
	// Returns the official openai.Client — same API, x402-paid transport.
	client, err := vip.NewOpenAI()
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model: openai.ChatModel("gpt-4o"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("In one sentence, what is x402?"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Genuine OpenAI-direct signals survive the passthrough.
	fmt.Println(resp.Choices[0].Message.Content)
	fmt.Printf("id=%s model=%s fingerprint=%s\n", resp.ID, resp.Model, resp.SystemFingerprint)
}
