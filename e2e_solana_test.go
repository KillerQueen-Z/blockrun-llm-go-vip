package vip

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/openai/openai-go"
)

// Live-gateway tests. Skipped unless BLOCKRUN_E2E=1 because they sign real
// USDC payments on Solana mainnet (~$0.001 per successful call).
//
// These exist because the stale-blockhash recovery is classified from the
// gateway's 402 body, and that body is owned by another repo (blockrun-sol).
// A unit test asserts against a body WE wrote, so it stays green even when the
// gateway renames a field and the recovery silently stops firing. Only a live
// run proves the classification still matches production.
const (
	e2eUpstream = "https://sol.blockrun.ai/api"
	e2eModel    = "gpt-4o-mini"

	// A valid 32-byte base58 value that is certainly not a recent blockhash, so
	// the facilitator's simulation fails with BlockhashNotFound — the exact
	// production symptom this recovery path exists for.
	e2eDeadBlockhash = testBlockhashA
)

func requireE2E(t *testing.T) {
	t.Helper()
	if os.Getenv("BLOCKRUN_E2E") != "1" {
		t.Skip("live gateway test; set BLOCKRUN_E2E=1 (signs real USDC, ~$0.001 per paid call)")
	}
}

// gatewayProxy forwards verbatim to the live Solana gateway, optionally
// poisoning the recentBlockhash of the FIRST 402 challenge. Everything else
// passes through untouched, so the rejection the SDK classifies on is the
// gateway's own, not one this test invented.
type gatewayProxy struct {
	poisonFirst bool

	mu         sync.Mutex
	challenges int
	paidLegs   int
	rejections []string
}

func (p *gatewayProxy) stats() (challenges, paidLegs int, rejections []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.challenges, p.paidLegs, append([]string(nil), p.rejections...)
}

func (p *gatewayProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := io.ReadAll(r.Body)
	out, err := http.NewRequestWithContext(r.Context(), r.Method, e2eUpstream+r.URL.Path, bytes.NewReader(reqBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for k, vs := range r.Header {
		switch {
		case strings.EqualFold(k, "Accept-Encoding"),
			strings.EqualFold(k, "Content-Length"),
			strings.EqualFold(k, "Host"):
			continue
		}
		for _, v := range vs {
			out.Header.Add(k, v)
		}
	}
	paid := r.Header.Get("PAYMENT-SIGNATURE") != ""

	resp, err := http.DefaultClient.Do(out)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	challenge := resp.Header.Get("payment-required")

	p.mu.Lock()
	if paid {
		p.paidLegs++
	}
	if challenge != "" {
		p.challenges++
		if p.poisonFirst && p.challenges == 1 {
			if poisoned, ok := poisonChallengeHeader(challenge); ok {
				resp.Header.Set("payment-required", poisoned)
				if resp.Header.Get("x-payment-required") != "" {
					resp.Header.Set("x-payment-required", poisoned)
				}
				if body, ok := poisonChallengeJSON(respBody); ok {
					respBody = body
				}
			}
		}
	}
	if paid && resp.StatusCode == http.StatusPaymentRequired && challenge == "" {
		p.rejections = append(p.rejections, string(respBody))
	}
	p.mu.Unlock()

	for k, vs := range resp.Header {
		if strings.EqualFold(k, "Content-Length") || strings.EqualFold(k, "Content-Encoding") {
			continue
		}
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}

// poisonChallengeJSON rewrites every accepts[].extra.recentBlockhash. It works
// on a generic map so an unrelated gateway field change cannot break it.
func poisonChallengeJSON(raw []byte) ([]byte, bool) {
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return nil, false
	}
	accepts, ok := doc["accepts"].([]any)
	if !ok {
		return nil, false
	}
	changed := false
	for _, a := range accepts {
		opt, ok := a.(map[string]any)
		if !ok {
			continue
		}
		extra, ok := opt["extra"].(map[string]any)
		if !ok {
			continue
		}
		if _, has := extra["recentBlockhash"]; has {
			extra["recentBlockhash"] = e2eDeadBlockhash
			changed = true
		}
	}
	if !changed {
		return nil, false
	}
	outRaw, err := json.Marshal(doc)
	if err != nil {
		return nil, false
	}
	return outRaw, true
}

func poisonChallengeHeader(h string) (string, bool) {
	raw, err := base64.StdEncoding.DecodeString(h)
	if err != nil {
		return "", false
	}
	out, ok := poisonChallengeJSON(raw)
	if !ok {
		return "", false
	}
	return base64.StdEncoding.EncodeToString(out), true
}

func e2eAsk(t *testing.T, client openai.Client) (string, error) {
	t.Helper()
	completion, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model:     e2eModel,
		MaxTokens: openai.Int(5),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Reply with the single word: ok"),
		},
	})
	if err != nil {
		return "", err
	}
	if len(completion.Choices) == 0 {
		return "", nil
	}
	return completion.Choices[0].Message.Content, nil
}

// Baseline: the rewritten request loop still pays and returns on the happy
// path. This is the regression this PR could most easily break, since it
// replaced the whole single-shot flow with a loop.
func TestE2E_SolanaPaidCallBaseline(t *testing.T) {
	requireE2E(t)

	client, err := NewOpenAI(WithChain("solana"))
	if err != nil {
		t.Fatalf("NewOpenAI(solana): %v", err)
	}
	content, err := e2eAsk(t, client)
	if err != nil {
		t.Fatalf("live paid call failed: %v", err)
	}
	if strings.TrimSpace(content) == "" {
		t.Fatal("live paid call returned empty content")
	}
	t.Logf("baseline paid call OK, content=%q", content)
}

// The real thing: a genuine stale blockhash, rejected by the real facilitator,
// recovered by re-signing against a fresh live challenge.
func TestE2E_SolanaStaleBlockhashRecovery(t *testing.T) {
	requireE2E(t)

	proxy := &gatewayProxy{poisonFirst: true}
	srv := httptest.NewServer(proxy)
	defer srv.Close()

	client, err := NewOpenAI(WithChain("solana"), WithBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewOpenAI(solana, proxy): %v", err)
	}

	content, err := e2eAsk(t, client)
	challenges, paidLegs, rejections := proxy.stats()
	for i, r := range rejections {
		t.Logf("live gateway rejection #%d: %s", i+1, r)
	}
	t.Logf("challenges=%d paidLegs=%d", challenges, paidLegs)

	if err != nil {
		t.Fatalf("stale-blockhash recovery failed end to end: %v", err)
	}
	if strings.TrimSpace(content) == "" {
		t.Fatal("recovered call returned empty content")
	}
	if len(rejections) == 0 {
		t.Fatal("poisoned blockhash was not rejected — the test never exercised recovery")
	}
	if challenges < 2 {
		t.Fatalf("recovery must re-quote a fresh challenge, saw %d", challenges)
	}
	if paidLegs < 2 {
		t.Fatalf("recovery must re-sign, saw %d paid legs", paidLegs)
	}
	t.Logf("recovered after real BlockhashNotFound, content=%q", content)
}
