// AI generation clients through BlockRun, paid per call in USDC (x402) on Base:
// image, speech/music, and web search (Grok + Exa). Voice/Phone are shown but
// commented out because they provision real numbers and place real calls.
//
// Run with a funded Base wallet on the gateway:
//
//	export BLOCKRUN_WALLET_KEY=0x...   # or ~/.blockrun/.session
//	go run ./examples/generation
package main

import (
	"context"
	"fmt"
	"log"

	vip "github.com/BlockRunAI/blockrun-llm-go-vip"
)

func main() {
	ctx := context.Background()

	// ---- Image: text-to-image ----
	img, err := vip.NewImage()
	if err != nil {
		log.Fatal(err)
	}
	pic, err := img.Generate(ctx, "a red fox in fresh snow, soft studio light",
		&vip.ImageGenerateOptions{Model: "openai/gpt-image-1"})
	if err != nil {
		log.Fatal(err)
	}
	if len(pic.Data) > 0 {
		fmt.Println("image:", pic.Data[0].URL)
	}

	// ---- Audio: speech, sound effect, music ----
	sp, err := vip.NewSpeech()
	if err != nil {
		log.Fatal(err)
	}
	speech, err := sp.Generate(ctx, "Hello there, this is BlockRun.",
		&vip.SpeechGenerateOptions{Voice: "sarah"})
	if err != nil {
		log.Fatal(err)
	}
	if len(speech.Data) > 0 {
		fmt.Println("speech:", speech.Data[0].URL)
	}

	mu, err := vip.NewMusic()
	if err != nil {
		log.Fatal(err)
	}
	instrumental := true
	track, err := mu.Generate(ctx, "dreamy lo-fi beat", // ~1-3 min, blocks until ready
		&vip.MusicGenerateOptions{Instrumental: &instrumental})
	if err != nil {
		log.Fatal(err)
	}
	if len(track.Data) > 0 {
		fmt.Println("music:", track.Data[0].URL)
	}

	// ---- Search: Grok Live Search + Exa ----
	s, err := vip.NewSearch()
	if err != nil {
		log.Fatal(err)
	}
	res, err := s.Search(ctx, "latest on x402 micropayments",
		&vip.SearchOptions{Sources: []string{"x", "news"}, MaxResults: 10})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("search: %+v\n", res)

	exa, err := vip.NewExa()
	if err != nil {
		log.Fatal(err)
	}
	hits, err := exa.Search(ctx, "x402 protocol", map[string]any{"numResults": 5})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("exa hits:", hits)

	// ---- Voice & Phone (costs real money — buy a number, place a call) ----
	// phone, _ := vip.NewPhone()
	// num, _ := phone.BuyNumber(ctx, vip.BuyNumberOptions{Country: "US", AreaCode: "415"}) // $5
	// fmt.Println("number:", num.PhoneNumber)
	// voice, _ := vip.NewVoice()
	// call, _ := voice.Call(ctx, vip.CallOptions{
	//     To:          "+14155551234",
	//     Task:        "Ask if they're open Sunday, confirm hours, then thank them and end.",
	//     MaxDuration: 3,
	// })
	// status, _ := voice.GetCallStatus(ctx, call.CallID)
	// fmt.Println("call status:", status)
}
