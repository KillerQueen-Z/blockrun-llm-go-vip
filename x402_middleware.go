package vip

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	blockrun "github.com/BlockRunAI/blockrun-llm-go"
)

// transportMiddleware is the shared underlying type of the Anthropic and OpenAI
// SDK option.Middleware aliases. Both are defined as
//
//	type Middleware = func(*http.Request, MiddlewareNext) (*http.Response, error)
//
// over the same MiddlewareNext alias, so a single implementation can be passed
// to option.WithMiddleware on either SDK without conversion.
type transportMiddleware = func(*http.Request, func(*http.Request) (*http.Response, error)) (*http.Response, error)

// x402Middleware builds the native-passthrough payment middleware.
//
// It performs the x402 negotiation transparently: the first request goes out
// unpaid, and on a 402 the requirement is parsed, an EIP-712 USDC authorization
// is signed locally, and the original request is replayed verbatim with the
// PAYMENT-SIGNATURE header. Everything else flows through untouched, so the
// official SDK parses the upstream provider's response byte-for-byte.
//
// SECURITY: priv is used ONLY for local EIP-712 signing. The key never leaves
// the machine; only the signature is transmitted.
func x402Middleware(priv *ecdsa.PrivateKey) transportMiddleware {
	return func(req *http.Request, next func(*http.Request) (*http.Response, error)) (*http.Response, error) {
		// Buffer the body so the 402 retry can replay it verbatim.
		var body []byte
		if req.Body != nil {
			b, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, fmt.Errorf("x402: read request body: %w", err)
			}
			_ = req.Body.Close()
			body = b
			req.Body = io.NopCloser(bytes.NewReader(body))
			req.ContentLength = int64(len(body))
		}

		resp, err := next(req)
		if err != nil {
			return resp, err
		}
		if resp.StatusCode != http.StatusPaymentRequired {
			// Native passthrough: hand the upstream response back untouched.
			return resp, nil
		}

		paymentHeader := readPaymentRequirement(resp)
		_ = resp.Body.Close()
		if paymentHeader == "" {
			return resp, &blockrun.PaymentError{Message: "402 response but no payment requirements found"}
		}

		signature, err := signPayment(priv, paymentHeader, req.URL.String())
		if err != nil {
			return resp, err
		}

		retry := req.Clone(req.Context())
		if body != nil {
			retry.Body = io.NopCloser(bytes.NewReader(body))
			retry.ContentLength = int64(len(body))
			retry.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(body)), nil
			}
		}
		retry.Header.Set("PAYMENT-SIGNATURE", signature)
		return next(retry)
	}
}

// readPaymentRequirement extracts the base64 payment requirement, preferring the
// `payment-required` header and falling back to an x402 body, mirroring the
// blockrun-llm-go base client behaviour.
func readPaymentRequirement(resp *http.Response) string {
	if h := resp.Header.Get("payment-required"); h != "" {
		return h
	}
	var rb map[string]any
	if json.NewDecoder(resp.Body).Decode(&rb) == nil {
		if _, ok := rb["x402"]; ok {
			if jb, err := json.Marshal(rb); err == nil {
				return string(jb)
			}
		}
	}
	return ""
}

// signPayment parses the requirement and returns a signed x402 v2 payload.
func signPayment(priv *ecdsa.PrivateKey, paymentHeader, requestURL string) (string, error) {
	paymentReq, err := blockrun.ParsePaymentRequired(paymentHeader)
	if err != nil {
		return "", &blockrun.PaymentError{Message: fmt.Sprintf("failed to parse payment requirements: %v", err)}
	}
	option, err := blockrun.ExtractPaymentDetails(paymentReq)
	if err != nil {
		return "", &blockrun.PaymentError{Message: fmt.Sprintf("failed to extract payment details: %v", err)}
	}

	resourceURL := paymentReq.Resource.URL
	if resourceURL == "" {
		resourceURL = requestURL
	}

	payload, err := blockrun.CreatePaymentPayload(
		priv,
		option.PayTo,
		option.Amount,
		option.Network,
		resourceURL,
		paymentReq.Resource.Description,
		option.MaxTimeoutSeconds,
		option.Extra,
		paymentReq.Extensions,
	)
	if err != nil {
		return "", &blockrun.PaymentError{Message: fmt.Sprintf("failed to create payment: %v", err)}
	}
	return payload, nil
}
