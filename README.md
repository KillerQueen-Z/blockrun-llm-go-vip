# blockrun-llm-go-vip

Genuine **native passthrough** for **Anthropic** and **OpenAI** through the BlockRun
gateway — pay per call in USDC (x402) on **Base**, with **zero model substitution and
zero response reshaping**.

Unlike a re-implemented client, the constructors here return the **official
`anthropic-sdk-go` and `openai-go` client types**. They only swap the transport (to add
x402 payment) and the base URL. The gateway returns the upstream provider's response
**verbatim**, so the official SDK parses the real signals:

- **Claude**: real thinking-block `Signature`, native content blocks (text / thinking /
  tool_use), `Usage` cache tokens, native streaming — routed to Anthropic's native
  `/v1/messages`.
- **GPT**: native `ID` (`chatcmpl-*`), `SystemFingerprint`, honest token usage and JSON
  mode — routed to `/v1/chat/completions`.

A Claude / OpenAI relay detector sees a direct upstream call.

This is the Go counterpart of the Python [`blockrun-llm-vip`](../blockrun-llm-vip) package,
reusing the x402 signing and wallet loading from
[`blockrun-llm-go`](../blockrun-llm-go).

## Install

```bash
go get github.com/BlockRunAI/blockrun-llm-go-vip
```

## Use — it's a drop-in

```go
import (
    "github.com/anthropics/anthropic-sdk-go"
    vip "github.com/BlockRunAI/blockrun-llm-go-vip"
)

// Claude — exactly the official anthropic-sdk-go API.
client, _ := vip.NewAnthropic()          // wallet auto-loaded from ~/.blockrun/.session
msg, _ := client.Messages.New(ctx, anthropic.MessageNewParams{
    Model:     anthropic.Model("claude-sonnet-4-6"),
    MaxTokens: 1024,
    Thinking:  anthropic.ThinkingConfigParamOfEnabled(1024),
    Messages:  []anthropic.MessageParam{
        anthropic.NewUserMessage(anthropic.NewTextBlock("What is 23*47?")),
    },
})
for _, b := range msg.Content {
    if b.Type == "thinking" {
        fmt.Println("signature:", b.Signature)   // real Anthropic signature
    }
}
```

```go
import (
    vip "github.com/BlockRunAI/blockrun-llm-go-vip"
    "github.com/openai/openai-go"
)

// GPT — exactly the official openai-go API.
gpt, _ := vip.NewOpenAI()
resp, _ := gpt.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
    Model:    openai.ChatModel("gpt-4o"),
    Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage("hi")},
})
fmt.Println(resp.SystemFingerprint, resp.Model)   // genuine OpenAI direct
```

Runnable examples: [`examples/anthropic`](examples/anthropic),
[`examples/openai`](examples/openai), and [`examples/seedance`](examples/seedance).

## Seedance video — incl. real-person (RealFace) & AI character (Portrait)

Generate short videos through **ByteDance Seedance**. `Video.Generate(...)` runs the
async submit→poll loop (x402-paid both legs by the same wallet) and returns the
gateway's verbatim completed-job JSON — `Data[0].URL` is a permanent BlockRun-hosted
MP4. These are not official-SDK passthrough (there is no upstream SDK to subclass) —
they reuse blockrun-llm-go's gateway clients, already x402-paid on Base.

```go
video, _ := vip.NewVideo()
job, _ := video.Generate(ctx, "a neon-lit cyberpunk street, slow dolly forward",
    &vip.VideoGenerateOptions{Model: "bytedance/seedance-2.0-fast", DurationSeconds: 5})
fmt.Println(job.Data[0].URL)
```

A specific, real person can appear consistently across clips: **enroll once** via
**RealFace** (one-time $0.01, ~1-min on-phone liveness for consent, no KYC), get a
`ta_xxxx`, and pass it as `RealFaceAssetID` on Seedance 2.0 / 2.0-fast. For an AI
character / mascot use **Portrait** instead (single enroll, no liveness).

```go
portrait, _ := vip.NewPortrait()
asset, _ := portrait.Enroll(ctx, "Mascot", "https://example.com/character.jpg") // $0.01
job, _ := video.Generate(ctx, "the mascot waves in soft studio light",
    &vip.VideoGenerateOptions{Model: "bytedance/seedance-2.0", RealFaceAssetID: asset.AssetID})
```

`RealFaceAssetID` is mutually exclusive with `ImageURL` and only works on Seedance
2.0 / 2.0-fast. Constructors: `vip.NewVideo`, `vip.NewRealFace`, `vip.NewPortrait`.

## Options

```go
vip.NewAnthropic(
    vip.WithWalletKey("0x..."),                  // explicit Base key (hex)
    vip.WithBaseURL("https://blockrun.ai/api"),  // override the gateway
    vip.WithAPIKey("blockrun"),                  // placeholder upstream key
)
```

All options apply to `NewOpenAI` too.

## Wallet

The private key is used **only for local EIP-712 signing** and never leaves your machine.
Resolution order:

1. `WithWalletKey(...)` option
2. `BLOCKRUN_WALLET_KEY` env
3. `BASE_CHAIN_WALLET_KEY` env
4. `~/.blockrun/.session`

The gateway base URL can also be overridden via the `BLOCKRUN_API_URL` env var.

## How it works

`NewAnthropic` / `NewOpenAI` build the official SDK client with two request options:

- `option.WithBaseURL(...)` — point at the BlockRun gateway.
- `option.WithMiddleware(...)` — a transport middleware that performs the x402
  handshake: the first request goes out unpaid; on a `402` the `payment-required`
  requirement is parsed, an EIP-712 USDC authorization is signed locally, and the
  original request is replayed **verbatim** with a `PAYMENT-SIGNATURE` header. On `200`
  the upstream body is handed back untouched.

Because the middleware never reshapes the success response, the official SDK does all
parsing — that is what makes the passthrough native.

## Scope

Covers **Anthropic + OpenAI native passthrough** and **Seedance video (incl.
RealFace real-person and Virtual Portrait)**, all on **Base**. Solana payment
(`chain="solana"` in the Python VIP package) is not yet ported.

## Access

Give BlockRun your wallet address to enable VIP, then pay per call from that wallet.

Contact: vicky@blockrun.ai
