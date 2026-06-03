package vip

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	blockrun "github.com/BlockRunAI/blockrun-llm-go"
)

const (
	videoEndpoint = "/v1/videos/generations"
	// The gateway now runs video generation asynchronously: POST returns 202
	// with a poll_url, and the client must GET it (re-signing x402) until the
	// job is completed. These bound that poll loop.
	defaultVideoPollInterval = 12 * time.Second
	defaultVideoMaxWait      = 5 * time.Minute
)

// Video generates short videos through ByteDance Seedance (and Grok Imagine)
// via the BlockRun gateway, paid per call in USDC (x402) on Base.
//
// Unlike blockrun-llm-go's VideoClient, Generate handles the gateway's async
// submit→poll flow: it POSTs the job, and on a 202 polls the returned poll_url
// (re-signing x402 each time) until the job completes, then returns the verbatim
// completed-job JSON. Data[0].URL is a permanent BlockRun-hosted MP4.
type Video struct {
	priv         *ecdsa.PrivateKey
	apiURL       string
	httpClient   *http.Client
	pollInterval time.Duration
	maxWait      time.Duration
}

// NewVideo returns a Seedance video client paid per call via x402 on Base.
//
//	video, _ := vip.NewVideo()
//	job, _ := video.Generate(ctx, "a neon-lit cyberpunk street, slow dolly", &vip.VideoGenerateOptions{
//	    Model: "bytedance/seedance-2.0-fast", DurationSeconds: 5,
//	})
//	fmt.Println(job.Data[0].URL)
func NewVideo(opts ...Option) (*Video, error) {
	cfg, priv, err := resolve(opts...)
	if err != nil {
		return nil, err
	}
	return &Video{
		priv:         priv,
		apiURL:       cfg.apiURL,
		httpClient:   &http.Client{Timeout: 60 * time.Second},
		pollInterval: defaultVideoPollInterval,
		maxWait:      defaultVideoMaxWait,
	}, nil
}

// Generate submits a video job and waits for it to complete.
func (v *Video) Generate(ctx context.Context, prompt string, opts *VideoGenerateOptions) (*VideoResponse, error) {
	body := map[string]any{"prompt": prompt}
	if opts != nil {
		if opts.ImageURL != "" && opts.RealFaceAssetID != "" {
			return nil, fmt.Errorf("vip: ImageURL and RealFaceAssetID are mutually exclusive; pass at most one")
		}
		if opts.RealFaceAssetID != "" && !strings.HasPrefix(opts.RealFaceAssetID, "ta_") {
			return nil, fmt.Errorf("vip: RealFaceAssetID must start with 'ta_' (enroll via NewRealFace or NewPortrait)")
		}
		if opts.Model != "" {
			body["model"] = opts.Model
		}
		if opts.ImageURL != "" {
			body["image_url"] = opts.ImageURL
		}
		if opts.RealFaceAssetID != "" {
			body["real_face_asset_id"] = opts.RealFaceAssetID
		}
		if opts.DurationSeconds > 0 {
			body["duration_seconds"] = opts.DurationSeconds
		}
		if opts.Resolution != "" {
			body["resolution"] = opts.Resolution
		}
		if opts.GenerateAudio != nil {
			body["generate_audio"] = *opts.GenerateAudio
		}
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("vip: encode video request: %w", err)
	}

	// Submit.
	status, respBytes, err := v.doX402(ctx, http.MethodPost, v.apiURL+videoEndpoint, jsonBody)
	if err != nil {
		return nil, err
	}
	switch status {
	case http.StatusOK:
		return decodeVideoResponse(respBytes)
	case http.StatusAccepted:
		return v.pollUntilDone(ctx, respBytes)
	default:
		return nil, &blockrun.APIError{StatusCode: status, Message: fmt.Sprintf("video submit failed: %s", string(respBytes))}
	}
}

// pollUntilDone reads the poll_url from a 202 submit body and polls it (re-signing
// x402) until the job completes.
func (v *Video) pollUntilDone(ctx context.Context, submitBody []byte) (*VideoResponse, error) {
	var job struct {
		PollURL string `json:"poll_url"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(submitBody, &job); err != nil {
		return nil, fmt.Errorf("vip: parse 202 submit body: %w", err)
	}
	if job.PollURL == "" {
		return nil, fmt.Errorf("vip: 202 submit returned no poll_url: %s", string(submitBody))
	}

	pollURL, err := v.absoluteURL(job.PollURL)
	if err != nil {
		return nil, err
	}

	deadline := time.Now().Add(v.maxWait)
	for {
		status, body, err := v.doX402(ctx, http.MethodGet, pollURL, nil)
		if err != nil {
			return nil, err
		}
		if status == http.StatusOK {
			return decodeVideoResponse(body)
		}
		if status != http.StatusAccepted {
			return nil, &blockrun.APIError{StatusCode: status, Message: fmt.Sprintf("video poll failed: %s", string(body))}
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("vip: video not ready after %s (still generating; poll_url: %s)", v.maxWait, job.PollURL)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(v.pollInterval):
		}
	}
}

// doX402 performs an HTTP request and, on a 402, signs an x402 payment and
// retries once with the payment header.
func (v *Video) doX402(ctx context.Context, method, fullURL string, body []byte) (int, []byte, error) {
	newReq := func() (*http.Request, error) {
		var r io.Reader
		if body != nil {
			r = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, fullURL, r)
		if err != nil {
			return nil, err
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		return req, nil
	}

	req, err := newReq()
	if err != nil {
		return 0, nil, err
	}
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("vip: video request: %w", err)
	}
	if resp.StatusCode != http.StatusPaymentRequired {
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp.StatusCode, data, nil
	}

	paymentHeader := readPaymentRequirement(resp)
	resp.Body.Close()
	if paymentHeader == "" {
		return resp.StatusCode, nil, &blockrun.PaymentError{Message: "402 response but no payment requirements found"}
	}
	signature, err := signPayment(v.priv, paymentHeader, fullURL)
	if err != nil {
		return 0, nil, err
	}

	retry, err := newReq()
	if err != nil {
		return 0, nil, err
	}
	// The gateway accepts PAYMENT-SIGNATURE; the video poll docs also mention an
	// x-payment header, so set both for compatibility.
	retry.Header.Set("PAYMENT-SIGNATURE", signature)
	retry.Header.Set("X-PAYMENT", signature)
	resp2, err := v.httpClient.Do(retry)
	if err != nil {
		return 0, nil, fmt.Errorf("vip: video retry request: %w", err)
	}
	data, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	return resp2.StatusCode, data, nil
}

// absoluteURL resolves a possibly-relative poll_url against the gateway origin.
func (v *Video) absoluteURL(pollURL string) (string, error) {
	if strings.HasPrefix(pollURL, "http://") || strings.HasPrefix(pollURL, "https://") {
		return pollURL, nil
	}
	base, err := url.Parse(v.apiURL)
	if err != nil {
		return "", fmt.Errorf("vip: parse api url: %w", err)
	}
	return fmt.Sprintf("%s://%s%s", base.Scheme, base.Host, pollURL), nil
}

func decodeVideoResponse(b []byte) (*VideoResponse, error) {
	var resp VideoResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, fmt.Errorf("vip: decode video response: %w", err)
	}
	return &resp, nil
}
