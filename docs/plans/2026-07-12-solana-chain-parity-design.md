# Solana chain parity for `blockrun-llm-go-vip`

**Date:** 2026-07-12
**Goal:** Give the Go VIP package the same Solana experience the Python `blockrun-llm-vip`
already has: pass `WithChain("solana")` to any constructor and pay USDC on Solana via
`sol.blockrun.ai`, with zero change to the default Base behaviour.

## Background

- The Python VIP exposes `chain="solana"` on every client. It routes to
  `sol.blockrun.ai/api`, loads a bs58 Solana key, and signs with the x402 **SVM "exact"
  scheme** (a real signed Solana `VersionedTransaction`), via the PayAI facilitator.
- The Go VIP has **no** Solana support — `resolve()` yields a secp256k1 key, `signPayment`
  is EIP-712 on Base USDC, base URL is `blockrun.ai/api`.
- The shared `blockrun-llm-go` (through local 0.18.0) has **no** Solana signing either —
  only a read-only RPC passthrough. So SVM x402 signing must be built from scratch in Go.

Decisions (confirmed with the user):
1. SVM signing lives in `blockrun-llm-go` first (mirrors Python's x402-lib layering),
   then this package wires it.
2. API shape: a `WithChain("base"|"solana")` functional option on existing constructors.
3. Live verification: user has a funded Solana wallet at `~/.blockrun/.solana-session`.

## The x402 SVM "exact" scheme (ground truth)

Reference: the working Python `x402` SDK
(`x402/mechanisms/svm/exact/client.py`, `x402/schemas/payments.py`).

**Transaction** — a `MessageV0` compiled with `feePayer` (from `requirement.extra.feePayer`)
as payer, four instructions in order:
1. ComputeBudget `SetComputeUnitLimit` — disc `0x02` + u32 LE `20000`.
2. ComputeBudget `SetComputeUnitPrice` — disc `0x03` + u64 LE `1` (microlamports).
3. SPL Token `TransferChecked` — disc `0x0c` + u64 LE `amount` + u8 `decimals`;
   accounts `[sourceATA(w), mint(r), destATA(w), payer(signer,r)]`.
4. SPL Memo — program `MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr`,
   data = hex of 16 random bytes (uniqueness nonce), no accounts.

Token program + decimals come from the mint account (owner → Token vs Token-2022,
`data[44]` → decimals). USDC mainnet mint `EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v`,
classic Token program, 6 decimals. ATAs derived with the standard ATA program.

Signing: prepend `0x80` version byte to the serialized `MessageV0`, ed25519-sign with the
client key. The `VersionedTransaction` carries signatures `[Signature.default (feePayer
placeholder — facilitator fills it), clientSignature]`. Serialize → base64.

**Envelope** (`PaymentPayload`, camelCase, `exclude_none`, then base64 → header):
```json
{"x402Version":2,
 "payload":{"transaction":"<base64 tx>"},
 "accepted":{"scheme":"exact","network":"<from 402>","asset":"<mint>",
             "amount":"<smallest unit>","payTo":"<recipient>",
             "maxTimeoutSeconds":<n>,"extra":{"feePayer":"<pk>", ...}}}
```
`accepted` echoes the fulfilled requirement from the 402. Sent as both `PAYMENT-SIGNATURE`
and `X-Payment` headers (matches the Base transport + video/realface endpoints).

RPC (blockhash + mint info): default `https://sol.blockrun.ai/api/v1/solana/rpc`,
override with `SOLANA_RPC_URL`.

## Repo 1 — `blockrun-llm-go` (v0.18.0 → v0.19.0)

New dependency: `github.com/gagliardetto/solana-go v1.12.0` (Go 1.19-compatible; provides
MessageV0, ATA derivation, TransferChecked, compute-budget, memo, base58/ed25519).

**New `solana_wallet.go`**
- `LoadSolanaWallet() (string, error)` — order: `SOLANA_WALLET_KEY` env → scan
  `~/.*/solana-wallet.json` (most-recent, `privateKey`+`address`) → `~/.blockrun/.solana-session`.
- `GetSolanaPublicKey(bs58Key string) (string, error)` — accepts 32-byte seed or 64-byte keypair.
- Const `DefaultSolanaAPIURL = "https://sol.blockrun.ai/api"`,
  `DefaultSolanaRPCURL = "https://sol.blockrun.ai/api/v1/solana/rpc"`,
  `SolanaSessionFile = ~/.blockrun/.solana-session`, USDC/program constants.

**New `solana_x402.go`**
- `CreateSolanaPaymentPayload(bs58Key string, opt *PaymentOption, resourceURL, description string, extra, extensions map[string]any, rpcURL string) (string, error)`
  building the tx + envelope above. Minimal `net/http` JSON-RPC helpers
  `solanaGetLatestBlockhash`, `solanaGetMintInfo` (owner + decimals).

**`base_client.go`**
- `baseClient` gains `chain string`, `solanaKey string`, `solanaRPCURL string`
  (privateKey nil on Solana; `address` = bs58 pubkey).
- `newSolanaBaseClient(bs58Key, apiURL, rpcURL string, timeout) (*baseClient, error)`.
- Centralized `bc.signX402(opt *PaymentOption, resourceURL, description string, extra, extensions map[string]any) (string, error)` branching base(EIP-712)/solana(SVM).
- Replace the **5** `CreatePaymentPayload` call sites (`base_client.go` ×2, `image.go`,
  `video.go`, `stream.go`) with `bc.signX402(...)`.
- `checkEnvAPIURL` becomes chain-aware (`BLOCKRUN_SOLANA_API_URL` vs `DefaultSolanaAPIURL`).

**Solana sibling constructors** (~3 lines each): `NewLLMClientSolana`,
`NewAnthropicClientSolana`, `NewImageClientSolana`, `NewVideoClientSolana`,
`NewSpeechClientSolana`, `NewMusicClientSolana`, `NewVoiceClientSolana`,
`NewPhoneClientSolana`, `NewRealFaceClientSolana`, `NewPortraitClientSolana`.

Bump `VERSION`, `CHANGELOG.md`, README Solana section.

## Repo 2 — `blockrun-llm-go-vip`

- `vip.go`: `config` gains `chain` (default `"base"`) + `solanaRPCURL`; new
  `WithChain(string)` and `WithSolanaRPCURL(string)`. `resolve*` on Solana loads bs58 via
  `blockrun.LoadSolanaWallet`, defaults apiURL to `sol.blockrun.ai/api`, builds a
  chain-aware signer closure.
- Signer refactor: `x402Middleware` and `Video` hold a
  `signer func(paymentHeader, url string) (string, error)` instead of `*ecdsa.PrivateKey`.
  Base → current `signPayment`; Solana → `blockrun.CreateSolanaPaymentPayload`.
- `anthropic.go`/`openai.go`/`video.go`: chain-aware base URL + signer.
- `media.go`/`generation.go`: each `New*` wrapper dispatches to the base-lib Solana
  sibling constructor when `chain=="solana"`.
- README + `examples/solana` + docs.

## Testing

- Base-lib unit tests decode the produced `VersionedTransaction` and assert: 4
  instructions in order, USDC mint, `destATA == ATA(payTo)`, amount + decimals,
  memo present, feePayer slot = placeholder signature, client signature valid over
  `0x80||message`, and the base64 envelope round-trips to the pinned JSON.
- vip routing tests: URL + signer selection per chain; `buildVideoBody` unchanged.
- Live: one real paid call each on Base (regression) and Solana (chat + one media)
  using `~/.blockrun/.solana-session`.

## Rollout

Develop the base lib with a temporary `replace github.com/BlockRunAI/blockrun-llm-go =>
../blockrun-llm-go` in vip's `go.mod`. Verify live. Then tag/release
`blockrun-llm-go v0.19.0`, bump vip's require, drop the replace, release the vip package.

## Risk

The exact envelope field names / `network` value must match the PayAI facilitator. Pinned
from the x402 Python schema; confirmed with the live Solana call before release.
