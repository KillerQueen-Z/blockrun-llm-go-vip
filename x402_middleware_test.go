package vip

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	blockrun "github.com/BlockRunAI/blockrun-llm-go"
	"github.com/openai/openai-go"
)

// testWalletKey is a throwaway, well-known Hardhat account #0 private key.
// It only signs locally in this test; no funds, no network.
const testWalletKey = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

// newRequirement builds a base64-encoded x402 payment requirement matching what
// the BlockRun gateway returns in the `payment-required` header on a 402.
func newRequirement(resourceURL string) string {
	req := blockrun.PaymentRequirement{
		X402Version: 2,
		Accepts: []blockrun.PaymentOption{{
			Scheme:            "exact",
			Network:           "base",
			Amount:            "1000",
			Asset:             blockrun.USDCBase,
			PayTo:             "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
			MaxTimeoutSeconds: 60,
		}},
		Resource: blockrun.ResourceInfo{URL: resourceURL, Description: "test"},
	}
	raw, _ := json.Marshal(req)
	return base64.StdEncoding.EncodeToString(raw)
}

const (
	testBlockhashA = "SysvarRent111111111111111111111111111111111"
	testBlockhashB = "SysvarC1ock11111111111111111111111111111111"
)

func newSolanaRequirement(resourceURL, blockhash string) string {
	req := blockrun.PaymentRequirement{
		X402Version: 2,
		Accepts: []blockrun.PaymentOption{{
			Scheme:            "exact",
			Network:           "solana",
			Amount:            "1000",
			Asset:             blockrun.USDCSolanaMainnet,
			PayTo:             "Vote111111111111111111111111111111111111111",
			MaxTimeoutSeconds: 60,
			Extra: map[string]any{
				"feePayer":             "Stake11111111111111111111111111111111111111",
				"recentBlockhash":      blockhash,
				"lastValidBlockHeight": "123456789",
			},
		}},
		Resource: blockrun.ResourceInfo{URL: resourceURL, Description: "test"},
	}
	raw, _ := json.Marshal(req)
	return base64.StdEncoding.EncodeToString(raw)
}

func paymentBlockhash(t *testing.T, signature string) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		t.Fatalf("decode payment signature: %v", err)
	}
	var envelope struct {
		Accepted blockrun.PaymentOption `json:"accepted"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("decode payment envelope: %v", err)
	}
	blockhash, _ := envelope.Accepted.Extra["recentBlockhash"].(string)
	return blockhash
}

func TestX402Middleware_SignsAndRetries(t *testing.T) {
	var calls int32
	var sawSignature bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		body, _ := io.ReadAll(r.Body)

		if r.Header.Get("PAYMENT-SIGNATURE") == "" {
			// First, unpaid attempt -> 402 with the requirement header.
			w.Header().Set("payment-required", newRequirement("http://"+r.Host+r.URL.Path))
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}

		// Paid retry -> verbatim 200. The body must match the original request
		// so we can prove it was replayed unchanged.
		sawSignature = true
		if string(body) != `{"hello":"world"}` {
			t.Errorf("retry body mutated: got %q", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
		_ = n
	}))
	defer srv.Close()

	priv, err := blockrun.GetPrivateKeyFromHex(testWalletKey)
	if err != nil {
		t.Fatalf("load key: %v", err)
	}

	mw := x402Middleware(func(paymentHeader, requestURL string) (string, error) {
		return signPayment(priv, paymentHeader, requestURL)
	}, nil)
	next := func(req *http.Request) (*http.Response, error) {
		return http.DefaultClient.Do(req)
	}

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/messages",
		bytes.NewReader([]byte(`{"hello":"world"}`)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := mw(req, next)
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 after payment, got %d", resp.StatusCode)
	}
	out, _ := io.ReadAll(resp.Body)
	if string(out) != `{"ok":true}` {
		t.Fatalf("unexpected passthrough body: %q", string(out))
	}
	if !sawSignature {
		t.Fatal("retry did not carry PAYMENT-SIGNATURE")
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("want exactly 2 upstream calls (402 then paid), got %d", got)
	}
}

func TestX402Middleware_PassthroughOn200(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"native":"verbatim"}`))
	}))
	defer srv.Close()

	priv, _ := blockrun.GetPrivateKeyFromHex(testWalletKey)
	mw := x402Middleware(func(paymentHeader, requestURL string) (string, error) {
		return signPayment(priv, paymentHeader, requestURL)
	}, nil)
	next := func(req *http.Request) (*http.Response, error) { return http.DefaultClient.Do(req) }

	req, _ := http.NewRequest(http.MethodPost, srv.URL, bytes.NewReader([]byte(`{}`)))
	resp, err := mw(req, next)
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	if string(out) != `{"native":"verbatim"}` {
		t.Fatalf("200 response was reshaped: %q", string(out))
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("want 1 call (no payment needed), got %d", got)
	}
}

func TestX402Middleware_ResignsAfterStaleBlockhash(t *testing.T) {
	var calls int32
	var quotes int32
	var signedBlockhashes []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"hello":"solana"}` {
			t.Errorf("request body mutated: got %q", string(body))
		}

		signature := r.Header.Get("PAYMENT-SIGNATURE")
		if signature == "" {
			quote := atomic.AddInt32(&quotes, 1)
			blockhash := testBlockhashA
			if quote > 1 {
				blockhash = testBlockhashB
			}
			w.Header().Set("payment-required", newSolanaRequirement("http://"+r.Host+r.URL.Path, blockhash))
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}

		blockhash := paymentBlockhash(t, signature)
		signedBlockhashes = append(signedBlockhashes, blockhash)
		if blockhash == testBlockhashA {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			_, _ = w.Write([]byte(`{"code":"PAYMENT_INVALID","debug":"transaction_simulation_failed","invalidMessage":"BlockhashNotFound"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	mw := x402MiddlewareWithBackoffs(func(paymentHeader, requestURL string) (string, error) {
		return signSolanaPayment(testSolanaKey, "", paymentHeader, requestURL)
	}, nil, []time.Duration{0})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/chat/completions", bytes.NewReader([]byte(`{"hello":"solana"}`)))
	resp, err := mw(req, http.DefaultClient.Do)
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 after fresh re-sign, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 4 {
		t.Fatalf("want 4 calls (quote/sign twice), got %d", got)
	}
	if want := []string{testBlockhashA, testBlockhashB}; !reflect.DeepEqual(signedBlockhashes, want) {
		t.Fatalf("signed blockhashes = %v, want %v", signedBlockhashes, want)
	}
}

func TestX402Middleware_DoesNotRetryTerminalSimulationFailure(t *testing.T) {
	var calls int32
	const failureBody = `{"code":"PAYMENT_INVALID","debug":"transaction_simulation_failed","invalidMessage":"InvalidAccountData"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.Header.Get("PAYMENT-SIGNATURE") == "" {
			w.Header().Set("payment-required", newSolanaRequirement("http://"+r.Host+r.URL.Path, testBlockhashA))
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(failureBody))
	}))
	defer srv.Close()

	mw := x402MiddlewareWithBackoffs(func(paymentHeader, requestURL string) (string, error) {
		return signSolanaPayment(testSolanaKey, "", paymentHeader, requestURL)
	}, nil, []time.Duration{0, 0})
	req, _ := http.NewRequest(http.MethodPost, srv.URL, bytes.NewReader([]byte(`{}`)))
	resp, err := mw(req, http.DefaultClient.Do)
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("want terminal 402 passthrough, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != failureBody {
		t.Fatalf("terminal response body changed: got %q", string(body))
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("terminal failure must not retry: got %d calls", got)
	}
}

func TestX402Middleware_BoundsStaleBlockhashRetries(t *testing.T) {
	var calls int32
	var quotes int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.Header.Get("PAYMENT-SIGNATURE") == "" {
			quote := atomic.AddInt32(&quotes, 1)
			blockhash := testBlockhashA
			if quote%2 == 0 {
				blockhash = testBlockhashB
			}
			w.Header().Set("payment-required", newSolanaRequirement("http://"+r.Host+r.URL.Path, blockhash))
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"code":"PAYMENT_BLOCKHASH_STALE","retryable":true}`))
	}))
	defer srv.Close()

	mw := x402MiddlewareWithBackoffs(func(paymentHeader, requestURL string) (string, error) {
		return signSolanaPayment(testSolanaKey, "", paymentHeader, requestURL)
	}, nil, []time.Duration{0, 0})
	req, _ := http.NewRequest(http.MethodPost, srv.URL, bytes.NewReader([]byte(`{}`)))
	resp, err := mw(req, http.DefaultClient.Do)
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("want final 402 after bounded retries, got %d", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 6 {
		t.Fatalf("want 6 calls (3 bounded quote/sign attempts), got %d", got)
	}
}

func TestNewOpenAI_SolanaRecoversFromStaleBlockhash(t *testing.T) {
	var calls int32
	var quotes int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		signature := r.Header.Get("PAYMENT-SIGNATURE")
		if signature == "" {
			quote := atomic.AddInt32(&quotes, 1)
			blockhash := testBlockhashA
			if quote > 1 {
				blockhash = testBlockhashB
			}
			w.Header().Set("payment-required", newSolanaRequirement("http://"+r.Host+r.URL.Path, blockhash))
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}

		if paymentBlockhash(t, signature) == testBlockhashA {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			_, _ = w.Write([]byte(`{"code":"PAYMENT_INVALID","debug":"transaction_simulation_failed","invalidMessage":"BlockhashNotFound"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id":"chatcmpl-local",
			"object":"chat.completion",
			"created":1,
			"model":"gpt-local",
			"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`))
	}))
	defer srv.Close()

	client, err := NewOpenAI(
		WithChain("solana"),
		WithWalletKey(testSolanaKey),
		WithFacilitator("payai"),
		WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewOpenAI: %v", err)
	}
	completion, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model: "gpt-local",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("hello"),
		},
	})
	if err != nil {
		t.Fatalf("Chat.Completions.New: %v", err)
	}
	if got := completion.Choices[0].Message.Content; got != "ok" {
		t.Fatalf("completion content = %q, want ok", got)
	}
	if got := atomic.LoadInt32(&calls); got != 4 {
		t.Fatalf("want 4 gateway calls through the public SDK, got %d", got)
	}
}
