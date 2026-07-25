// Anthropic native passthrough through BlockRun, paid per call in USDC (x402).
//
// Run with a funded Base wallet on the gateway:
//
//	export BLOCKRUN_WALLET_KEY=0x...   # or ~/.blockrun/.session
//	go run ./examples/anthropic
package main

import (
	"context"
	"fmt"
	"log"

	vip "github.com/BlockRunAI/blockrun-llm-go-vip"
	"github.com/anthropics/anthropic-sdk-go"
)

func main() {
	// Returns the official anthropic.Client — same API, x402-paid transport.
	client, err := vip.NewAnthropic()
	if err != nil {
		log.Fatal(err)
	}

	msg, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.Model("claude-opus-5"),
		MaxTokens: 1024,
		Thinking:  anthropic.ThinkingConfigParamOfEnabled(1024),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("What is 23*47? Think first.")),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// The response is parsed by the official SDK from the verbatim upstream
	// payload, so native signals survive — e.g. real thinking signatures.
	for _, block := range msg.Content {
		switch block.Type {
		case "thinking":
			fmt.Printf("thinking signature: %s\n", block.Signature)
		case "text":
			fmt.Println(block.Text)
		}
	}
	fmt.Printf("usage: in=%d out=%d\n", msg.Usage.InputTokens, msg.Usage.OutputTokens)
}
