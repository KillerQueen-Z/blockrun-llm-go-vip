// Package vip is a genuine native passthrough for Anthropic and OpenAI through
// the BlockRun gateway, paid per call in USDC (x402) on Base.
//
// Unlike a re-implemented client, the constructors here return the *official*
// anthropic-sdk-go and openai-go client types. They only swap the transport (to
// add x402 payment) and the base URL — the gateway returns the upstream
// provider's response verbatim, so the official SDK parses the real signals:
// Claude thinking-block signatures and native content blocks, GPT
// system_fingerprint and token-detail usage, native streaming, and so on. A
// relay detector sees a direct upstream call.
//
// Wallet keys are used ONLY for local EIP-712 signing and never leave the
// machine. Payment runs on Base (USDC).
package vip

import (
	"crypto/ecdsa"
	"fmt"

	blockrun "github.com/BlockRunAI/blockrun-llm-go"
)

// apiKeySentinel is sent as the upstream API key. The gateway authorizes by
// x402 payment, not by this value, but the official SDKs require a non-empty
// key, so we supply a placeholder rather than leak any real provider key.
const apiKeySentinel = "blockrun"

// config holds resolved client settings shared by the Anthropic and OpenAI
// constructors.
type config struct {
	apiURL  string
	apiKey  string
	privHex string // optional explicit wallet key (hex); empty means auto-load
}

// Option customises a VIP client.
type Option func(*config)

// WithBaseURL overrides the BlockRun gateway base URL
// (default: https://blockrun.ai/api). The BLOCKRUN_API_URL env var is honoured
// by the underlying loader when no explicit URL is set.
func WithBaseURL(url string) Option {
	return func(c *config) { c.apiURL = url }
}

// WithWalletKey sets the Base wallet private key (hex) explicitly. When unset,
// the key is loaded from BLOCKRUN_WALLET_KEY / BASE_CHAIN_WALLET_KEY or
// ~/.blockrun/.session.
func WithWalletKey(hexKey string) Option {
	return func(c *config) { c.privHex = hexKey }
}

// WithAPIKey overrides the placeholder upstream API key. Rarely needed —
// authorization is by x402 payment.
func WithAPIKey(key string) Option {
	return func(c *config) { c.apiKey = key }
}

// resolveKey applies options and resolves the wallet key as a hex string. The
// media clients (Video / RealFace / Portrait) reuse blockrun-llm-go's clients,
// which take the hex key directly.
func resolveKey(opts ...Option) (cfg config, hexKey string, err error) {
	cfg = config{
		apiURL: blockrun.DefaultAPIURL,
		apiKey: apiKeySentinel,
	}
	for _, o := range opts {
		o(&cfg)
	}

	hexKey = cfg.privHex
	if hexKey == "" {
		hexKey, err = blockrun.LoadWallet()
		if err != nil {
			return cfg, "", fmt.Errorf("vip: no wallet key (set BLOCKRUN_WALLET_KEY or ~/.blockrun/.session, or pass WithWalletKey): %w", err)
		}
	}
	return cfg, hexKey, nil
}

// resolve applies options and loads the signing key as an *ecdsa.PrivateKey for
// the x402 middleware used by the Anthropic / OpenAI passthrough clients.
func resolve(opts ...Option) (cfg config, priv *ecdsa.PrivateKey, err error) {
	cfg, hexKey, err := resolveKey(opts...)
	if err != nil {
		return cfg, nil, err
	}

	priv, err = blockrun.GetPrivateKeyFromHex(hexKey)
	if err != nil {
		return cfg, nil, fmt.Errorf("vip: invalid wallet key: %w", err)
	}
	return cfg, priv, nil
}
