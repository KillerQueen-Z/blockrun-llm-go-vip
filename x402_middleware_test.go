package vip

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
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

// paymentBlockhash is called from httptest handler goroutines, so it reports
// via t.Errorf and returns "" rather than t.Fatalf — FailNow from a non-test
// goroutine runs runtime.Goexit there, killing the handler mid-response and
// hiding the real decode error behind a transport failure.
func paymentBlockhash(t *testing.T, signature string) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		t.Errorf("decode payment signature: %v", err)
		return ""
	}
	var envelope struct {
		Accepted blockrun.PaymentOption `json:"accepted"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Errorf("decode payment envelope: %v", err)
		return ""
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
	var mu sync.Mutex
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
		mu.Lock()
		signedBlockhashes = append(signedBlockhashes, blockhash)
		mu.Unlock()
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
	mu.Lock()
	defer mu.Unlock()
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

// staleClassifyServer drives one quote → sign → rejectBody cycle, then answers
// 200. It exists so each production 402 dialect can assert the same property:
// does this body shape actually reach the re-sign path?
func staleClassifyServer(t *testing.T, rejectBody string, calls *int32) *httptest.Server {
	t.Helper()
	var quotes int32
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(calls, 1)
		if r.Header.Get("PAYMENT-SIGNATURE") == "" {
			blockhash := testBlockhashA
			if atomic.AddInt32(&quotes, 1) > 1 {
				blockhash = testBlockhashB
			}
			w.Header().Set("payment-required", newSolanaRequirement("http://"+r.Host+r.URL.Path, blockhash))
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}
		if atomic.LoadInt32(&quotes) == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			_, _ = w.Write([]byte(rejectBody))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
}

func runStaleClassify(t *testing.T, rejectBody string) (*http.Response, int32) {
	t.Helper()
	var calls int32
	srv := staleClassifyServer(t, rejectBody, &calls)
	defer srv.Close()

	mw := x402MiddlewareWithBackoffs(func(paymentHeader, requestURL string) (string, error) {
		return signSolanaPayment(testSolanaKey, "", paymentHeader, requestURL)
	}, nil, []time.Duration{0})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/chat/completions", bytes.NewReader([]byte(`{}`)))
	resp, err := mw(req, http.DefaultClient.Do)
	if err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp, atomic.LoadInt32(&calls)
}

// The body sol.blockrun.ai actually ships today on /v1/chat/completions. The
// gateway stopped echoing the facilitator's invalidMessage (blockrun-sol
// c2a17bf) and now sends only the classified `reason`, so a client that keys
// solely on invalidMessage never recovers in production.
func TestX402Middleware_ResignsOnGatewayExpiredSignatureReason(t *testing.T) {
	resp, calls := runStaleClassify(t, `{"error":"Payment verification failed","message":"Message @bc1max on Telegram for help.","code":"PAYMENT_INVALID","reason":"expired_signature","payer":"Vote111111111111111111111111111111111111111"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 after re-sign on reason=expired_signature, got %d", resp.StatusCode)
	}
	if calls != 4 {
		t.Fatalf("want 4 calls (quote/sign twice), got %d", calls)
	}
}

// The Anthropic dialect (/v1/messages) nests the classification inside
// error.message and carries no top-level code or reason at all.
func TestX402Middleware_ResignsOnAnthropicExpiredSignature(t *testing.T) {
	resp, calls := runStaleClassify(t, `{"type":"error","error":{"type":"invalid_request_error","message":"Payment verification failed: expired_signature"}}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("want 200 after re-sign on Anthropic-shaped rejection, got %d", resp.StatusCode)
	}
	if calls != 4 {
		t.Fatalf("want 4 calls (quote/sign twice), got %d", calls)
	}
}

// insufficient_funds is the other classified reason. No re-sign can fund a
// wallet, so it must stay terminal on both dialects.
func TestX402Middleware_DoesNotRetryClassifiedTerminalReasons(t *testing.T) {
	for name, body := range map[string]string{
		"openai":    `{"error":"Payment verification failed","code":"PAYMENT_INVALID","reason":"insufficient_funds"}`,
		"anthropic": `{"type":"error","error":{"type":"invalid_request_error","message":"Payment verification failed: insufficient_funds"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			resp, calls := runStaleClassify(t, body)
			if resp.StatusCode != http.StatusPaymentRequired {
				t.Fatalf("want terminal 402, got %d", resp.StatusCode)
			}
			got, _ := io.ReadAll(resp.Body)
			if string(got) != body {
				t.Fatalf("terminal body changed:\n got %q\nwant %q", string(got), body)
			}
			if calls != 2 {
				t.Fatalf("insufficient_funds must not retry: got %d calls", calls)
			}
		})
	}
}

// A gateway holding a signed payment must not be able to make the client
// buffer an unbounded body while classifying it.
func TestX402Middleware_OversizedRejectionIsTerminalAndIntact(t *testing.T) {
	huge := `{"code":"PAYMENT_BLOCKHASH_STALE","pad":"` + strings.Repeat("x", maxStaleClassifyBytes+1024) + `"}`
	resp, calls := runStaleClassify(t, huge)
	if resp.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("oversized rejection must be terminal, got %d", resp.StatusCode)
	}
	if calls != 2 {
		t.Fatalf("oversized rejection must not retry: got %d calls", calls)
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	if string(got) != huge {
		t.Fatalf("oversized body not restored byte-for-byte: got %d bytes, want %d", len(got), len(huge))
	}
}

// Backoff must observe cancellation instead of sleeping through a dead context.
func TestX402Middleware_StaleRetryHonorsContextCancellation(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.Header.Get("PAYMENT-SIGNATURE") == "" {
			w.Header().Set("payment-required", newSolanaRequirement("http://"+r.Host+r.URL.Path, testBlockhashA))
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"code":"PAYMENT_BLOCKHASH_STALE"}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	mw := x402MiddlewareWithBackoffs(func(paymentHeader, requestURL string) (string, error) {
		cancel() // dead by the time the first backoff starts
		return signSolanaPayment(testSolanaKey, "", paymentHeader, requestURL)
	}, nil, []time.Duration{30 * time.Second})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL, bytes.NewReader([]byte(`{}`)))

	done := make(chan error, 1)
	go func() {
		resp, err := mw(req, http.DefaultClient.Do)
		if resp != nil {
			_ = resp.Body.Close()
		}
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("backoff slept through a cancelled context")
	}
}

// A proxy HTML error page or truncated body must be terminal, not a re-sign.
func TestX402Middleware_NonJSONRejectionIsTerminal(t *testing.T) {
	const page = `<html><body>502 Bad Gateway: BlockhashNotFound</body></html>`
	resp, calls := runStaleClassify(t, page)
	if resp.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("non-JSON rejection must be terminal, got %d", resp.StatusCode)
	}
	if calls != 2 {
		t.Fatalf("non-JSON rejection must not retry: got %d calls", calls)
	}
	got, _ := io.ReadAll(resp.Body)
	if string(got) != page {
		t.Fatalf("non-JSON body changed: got %q", string(got))
	}
}

// Settlement-phase 402s must never re-sign. Settle has already broadcast a
// transaction; if it landed and only the confirmation was lost, a re-sign is a
// second real charge. Both gateway spellings are covered: /v1/chat/completions
// sets code=SETTLEMENT_FAILED, while exa/audio/surf/phone/rpc/pm send a bare
// error label with no code at all.
func TestX402Middleware_NeverRetriesSettlementFailures(t *testing.T) {
	for name, body := range map[string]string{
		"chat_completions_coded": `{"error":"Payment settlement failed","message":"Message @bc1max on Telegram for help.","code":"SETTLEMENT_FAILED","reason":"expired_signature"}`,
		"partner_route_uncoded":  `{"error":"Payment settlement failed","reason":"expired_signature"}`,
		"settle_stale_code":      `{"error":"Payment settlement failed","code":"PAYMENT_BLOCKHASH_STALE"}`,
	} {
		t.Run(name, func(t *testing.T) {
			resp, calls := runStaleClassify(t, body)
			if resp.StatusCode != http.StatusPaymentRequired {
				t.Fatalf("settlement failure must be terminal, got %d", resp.StatusCode)
			}
			if calls != 2 {
				t.Fatalf("settlement failure must not re-sign: got %d calls", calls)
			}
			got, _ := io.ReadAll(resp.Body)
			if string(got) != body {
				t.Fatalf("settlement body changed:\n got %q\nwant %q", string(got), body)
			}
		})
	}
}
