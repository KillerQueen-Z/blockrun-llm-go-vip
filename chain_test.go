package vip

import (
	"testing"

	blockrun "github.com/BlockRunAI/blockrun-llm-go"
)

// testSolanaKey is a throwaway Solana keypair (bs58). It only signs locally in
// tests; no funds, no network. Its public key is testSolanaPubkey.
const (
	testSolanaKey    = "2kDj4SxF8BBxdJZAGUFtB6mLKcTb8QwNzEBJWo2Qd3fr99d2V9HW4tcxjou8M9mkFK682ZykSiNv8vdQH2b95Bpp"
	testSolanaPubkey = "DpaAntLAPfQ7aAbW33inWAQvE3S5RvYm8HeVChETASHJ"
)

func TestResolveKey_BaseDefault(t *testing.T) {
	cfg, key, err := resolveKey(WithWalletKey(testWalletKey))
	if err != nil {
		t.Fatalf("resolveKey: %v", err)
	}
	if cfg.chain != chainBase {
		t.Errorf("chain = %q, want base", cfg.chain)
	}
	if cfg.apiURL != blockrun.DefaultAPIURL {
		t.Errorf("apiURL = %q, want %q", cfg.apiURL, blockrun.DefaultAPIURL)
	}
	if key != testWalletKey {
		t.Errorf("key = %q, want the base hex key", key)
	}
	if cfg.isSolana() {
		t.Error("isSolana() should be false for base")
	}
}

func TestResolveKey_Solana(t *testing.T) {
	cfg, key, err := resolveKey(WithChain("solana"), WithWalletKey(testSolanaKey))
	if err != nil {
		t.Fatalf("resolveKey: %v", err)
	}
	if !cfg.isSolana() {
		t.Fatal("isSolana() should be true")
	}
	if cfg.apiURL != blockrun.DefaultSolanaAPIURL {
		t.Errorf("apiURL = %q, want %q", cfg.apiURL, blockrun.DefaultSolanaAPIURL)
	}
	if key != testSolanaKey {
		t.Errorf("key = %q, want the bs58 solana key", key)
	}
}

func TestResolveKey_SolanaCustomURLAndRPC(t *testing.T) {
	cfg, _, err := resolveKey(
		WithChain("solana"),
		WithWalletKey(testSolanaKey),
		WithBaseURL("https://custom.example/api"),
		WithSolanaRPCURL("https://rpc.example"),
	)
	if err != nil {
		t.Fatalf("resolveKey: %v", err)
	}
	if cfg.apiURL != "https://custom.example/api" {
		t.Errorf("apiURL = %q, want the explicit override", cfg.apiURL)
	}
	if cfg.solanaRPCURL != "https://rpc.example" {
		t.Errorf("solanaRPCURL = %q, want the explicit override", cfg.solanaRPCURL)
	}
}

func TestResolveKey_UnknownChain(t *testing.T) {
	if _, _, err := resolveKey(WithChain("ethereum"), WithWalletKey(testWalletKey)); err == nil {
		t.Fatal("expected an error for an unknown chain")
	}
}

func TestResolveSigner_ChainSelection(t *testing.T) {
	// Base → non-nil signer that actually signs a base requirement.
	cfg, sign, err := resolveSigner(WithWalletKey(testWalletKey))
	if err != nil || sign == nil {
		t.Fatalf("base resolveSigner: sign=%v err=%v", sign, err)
	}
	if cfg.isSolana() {
		t.Error("base cfg should not be solana")
	}
	payload, err := sign(newRequirement("http://x/y"), "http://x/y")
	if err != nil || payload == "" {
		t.Errorf("base signer failed: payload=%q err=%v", payload, err)
	}

	// Solana → non-nil signer selected (invoking it needs RPC, covered live).
	cfg, sign, err = resolveSigner(WithChain("solana"), WithWalletKey(testSolanaKey))
	if err != nil || sign == nil {
		t.Fatalf("solana resolveSigner: sign=%v err=%v", sign, err)
	}
	if !cfg.isSolana() {
		t.Error("solana cfg.isSolana() should be true")
	}
}

// TestSolanaMediaClientWiring proves a media constructor builds a Solana-backed
// client whose wallet address is the bs58 pubkey (not a 0x Base address).
func TestSolanaMediaClientWiring(t *testing.T) {
	img, err := NewImage(WithChain("solana"), WithWalletKey(testSolanaKey))
	if err != nil {
		t.Fatalf("NewImage solana: %v", err)
	}
	if got := img.GetWalletAddress(); got != testSolanaPubkey {
		t.Errorf("wallet address = %q, want the solana pubkey %q", got, testSolanaPubkey)
	}

	// Base still yields a 0x address from the same wrapper.
	imgBase, err := NewImage(WithWalletKey(testWalletKey))
	if err != nil {
		t.Fatalf("NewImage base: %v", err)
	}
	if addr := imgBase.GetWalletAddress(); len(addr) < 2 || addr[:2] != "0x" {
		t.Errorf("base wallet address = %q, want a 0x address", addr)
	}
}
