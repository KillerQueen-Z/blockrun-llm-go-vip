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
    Model:     anthropic.Model("claude-opus-4.8"), // current flagship (adaptive thinking)
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

### Models

You name the model; the gateway never substitutes it. Pass any current id verbatim:

- **Claude**: `claude-opus-4.8` · `claude-opus-4.7` · `claude-opus-4.6` ·
  `claude-opus-4.5` · `claude-sonnet-4.6` · `claude-sonnet-4.5` · `claude-haiku-4.5`.
  Opus 4.7/4.8 use adaptive thinking — `anthropic.ThinkingConfigParamOfEnabled(N)` is honored.
- **GPT**: `gpt-5.5` · `gpt-5.4` · `gpt-5.3` · `gpt-5.2` · `gpt-4.1` · `gpt-4o` ·
  `gpt-4o-mini`, reasoning `o3` / `o4-mini`, and more. GPT‑5.x / o-series are reasoning
  models — leave `MaxTokens`/`Temperature` unset (the gateway normalizes them);
  `gpt-4o` / `gpt-4o-mini` are served OpenAI-direct.

Full live catalog (66+ models incl. xAI Grok, DeepSeek, Llama, Mistral, Gemini):
`https://blockrun.ai/api/v1/models`.

## Seedance video — incl. real-person (RealFace) & AI character (Portrait)

Generate short videos through **ByteDance Seedance**. The gateway runs generation
asynchronously, and the client mirrors that — `Submit` signs the x402 payment
authorization and returns immediately, then you `Poll` or `Wait`. USDC settles
on-chain only on the first poll that observes `completed`; failed or abandoned
jobs are never charged:

```go
video, _ := vip.NewVideo()

// Async: submit returns a job handle without blocking.
job, _ := video.Submit(ctx, "a neon-lit cyberpunk street, slow dolly forward",
    &vip.VideoGenerateOptions{Model: "bytedance/seedance-2.0-fast", DurationSeconds: 5})
fmt.Println(job.ID, job.Status) // e.g. "bytedance:video_…", "queued"

for !job.Done() {
    time.Sleep(12 * time.Second)
    video.Poll(ctx, job) // re-signs x402, advances job.Status
}
fmt.Println(job.Response.Data[0].URL) // permanent BlockRun-hosted MP4
```

`video.Wait(ctx, job)` blocks until done, and `video.Generate(ctx, prompt, opts)` is
shorthand for `Submit`+`Wait` (one blocking call). `Video` is implemented natively in
this package; `RealFace` and `Portrait` reuse blockrun-llm-go's gateway clients. All
x402-paid on Base.

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

Full real-person flow (RealFace state machine, on-phone liveness, error states):
**[docs/real-person-flow.md](docs/real-person-flow.md)**.

## Image — generate & edit

```go
img, _ := vip.NewImage()
out, _ := img.Generate(ctx, "a red fox in fresh snow, soft studio light",
    &vip.ImageGenerateOptions{Model: "openai/gpt-image-1"})
fmt.Println(out.Data[0].URL)

// edit / multi-image fusion:
edited, _ := img.Edit(ctx, "make it night", []string{dataURI}, nil)
```

Models: `openai/gpt-image-1` · `gpt-image-2` · `google/nano-banana` · `nano-banana-pro` ·
`xai/grok-imagine-image` · `zai/cogview-4`. `img.ListImageModels(ctx)` lists them (free).

## Audio — speech, music, sound effects

```go
sp, _ := vip.NewSpeech()
speech, _ := sp.Generate(ctx, "Hello there.", &vip.SpeechGenerateOptions{Voice: "sarah"})
sfx, _ := sp.SoundEffect(ctx, "distant thunder over rain", nil)

mu, _ := vip.NewMusic()
instrumental := true
track, _ := mu.Generate(ctx, "dreamy lo-fi beat", &vip.MusicGenerateOptions{Instrumental: &instrumental})
fmt.Println(speech.Data[0].URL, sfx.Data[0].URL, track.Data[0].URL)
```

`sp.ListVoices(ctx)` lists TTS voices (free). Music runs ~1-3 min and blocks until ready.

## Search — Grok Live Search & Exa

```go
s, _ := vip.NewSearch()
r, _ := s.Search(ctx, "latest on x402 micropayments",
    &vip.SearchOptions{Sources: []string{"x", "news"}, MaxResults: 15})

exa, _ := vip.NewExa()
hits, _ := exa.Search(ctx, "x402 protocol", map[string]any{"numResults": 5})
text, _ := exa.Contents(ctx, []string{"https://example.com"}, nil)
ans, _ := exa.Answer(ctx, "what is the x402 payment header?", nil)
```

## Voice & Phone — AI phone calls

Lease a wallet-bound number, then place an AI-driven outbound call. `Call` returns a
call id; poll it with `GetCallStatus` for the transcript + recording.

```go
phone, _ := vip.NewPhone()
num, _ := phone.BuyNumber(ctx, vip.BuyNumberOptions{Country: "US", AreaCode: "415"}) // $5, 30-day lease

voice, _ := vip.NewVoice()
call, _ := voice.Call(ctx, vip.CallOptions{
    To:          "+14155551234",
    Task:        "Ask if they're open Sunday, confirm hours, then thank them and end the call.",
    MaxDuration: 3,
})
status, _ := voice.GetCallStatus(ctx, call.CallID)
fmt.Println(status)
```

`Phone` also does `Lookup` / `LookupFraud`, `ListNumbers`, `RenewNumber`, `ReleaseNumber`.

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

Covers, all on **Base**:

- **Anthropic + OpenAI** native passthrough (`NewAnthropic`, `NewOpenAI`).
- **Seedance video** incl. RealFace real-person + Virtual Portrait (`NewVideo`,
  `NewRealFace`, `NewPortrait`).
- **Image** generate + edit (`NewImage`).
- **Audio**: ElevenLabs speech + sound effects (`NewSpeech`), MiniMax music (`NewMusic`).
- **Search**: Grok Live Search (`NewSearch`) + Exa web search (`NewExa`).
- **Voice & Phone**: AI phone calls (`NewVoice`) + number provisioning (`NewPhone`).

Solana payment (`chain="solana"` in the Python VIP package) is not yet ported.

## Access

Give BlockRun your wallet address to enable VIP, then pay per call from that wallet.

Contact: vicky@blockrun.ai
