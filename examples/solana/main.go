// Solana native passthrough through BlockRun, paid per call in USDC on Solana (x402).
//
// Identical to the Base examples — just add vip.WithChain("solana"). The gateway
// is sol.blockrun.ai and payment is signed with your bs58 Solana key.
//
// Run with a funded Solana wallet on the gateway:
//
//	export SOLANA_WALLET_KEY=...        # or ~/.blockrun/.solana-session
//	go run ./examples/solana
package main

import (
	"context"
	"fmt"
	"log"

	vip "github.com/BlockRunAI/blockrun-llm-go-vip"
	"github.com/openai/openai-go"
)

func main() {
	// The official openai.Client — same API, but paid on Solana.
	client, err := vip.NewOpenAI(vip.WithChain("solana"))
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model:     openai.ChatModel("gpt-4o-mini"),
		MaxTokens: openai.Int(64),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("In one sentence, what is Solana?"),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Genuine OpenAI-direct signals survive the Solana passthrough.
	fmt.Println(resp.Choices[0].Message.Content)
	fmt.Printf("id=%s model=%s fingerprint=%s\n", resp.ID, resp.Model, resp.SystemFingerprint)
}
