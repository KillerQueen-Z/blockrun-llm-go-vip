package vip

import (
	blockrun "github.com/BlockRunAI/blockrun-llm-go"
)

// Seedance video, RealFace (real-person), and Virtual Portrait support.
//
// Unlike the Anthropic / OpenAI clients, these are not official-SDK passthrough
// (there is no upstream SDK to subclass) — they are BlockRun gateway clients,
// already x402-paid on Base. The VIP package reuses blockrun-llm-go's
// implementations verbatim and only adds wallet resolution consistent with the
// rest of this package, so VIP users get the same clients from the same package
// and the same wallet.

// Client types, re-exported so callers never import blockrun-llm-go directly.
type (
	// Video generates short videos through ByteDance Seedance (and Grok
	// Imagine), running the async submit→poll loop and returning the gateway's
	// verbatim completed-job JSON.
	Video = blockrun.VideoClient
	// RealFace enrolls a real, specific person (one-time on-phone liveness for
	// consent, no KYC) and yields a ta_ asset for identity-consistent Seedance
	// 2.0 generation.
	RealFace = blockrun.RealFaceClient
	// Portrait enrolls an AI character / mascot (no liveness) and yields a ta_
	// asset for identity-consistent Seedance 2.0 generation.
	Portrait = blockrun.PortraitClient
)

// Option/result types re-exported for ergonomic use.
type (
	VideoGenerateOptions = blockrun.VideoGenerateOptions
	VideoResponse        = blockrun.VideoResponse
	WaitForActiveOptions = blockrun.WaitForActiveOptions
	RealFaceInit         = blockrun.RealFaceInit
	RealFaceStatus       = blockrun.RealFaceStatus
	RealFaceEnrollment   = blockrun.RealFaceEnrollment
	RealFaceList         = blockrun.RealFaceList
	PortraitEnrollment   = blockrun.PortraitEnrollment
	PortraitList         = blockrun.PortraitList
)

// NewVideo returns a Seedance video client paid per call via x402 on Base.
//
//	video, _ := vip.NewVideo()
//	job, _ := video.Generate(ctx, "a neon-lit cyberpunk street, slow dolly", &vip.VideoGenerateOptions{
//	    Model:           "bytedance/seedance-2.0-fast",
//	    DurationSeconds: 5,
//	})
//	fmt.Println(job.Data[0].URL)
func NewVideo(opts ...Option) (*Video, error) {
	cfg, hexKey, err := resolveKey(opts...)
	if err != nil {
		return nil, err
	}
	return blockrun.NewVideoClient(hexKey, blockrun.WithVideoAPIURL(cfg.apiURL))
}

// NewRealFace returns a RealFace client for enrolling a real, specific person
// (init → on-phone liveness → enroll), paid per call via x402 on Base.
func NewRealFace(opts ...Option) (*RealFace, error) {
	cfg, hexKey, err := resolveKey(opts...)
	if err != nil {
		return nil, err
	}
	return blockrun.NewRealFaceClient(hexKey, blockrun.WithRealFaceAPIURL(cfg.apiURL))
}

// NewPortrait returns a Virtual Portrait client for enrolling an AI character /
// mascot (single enroll call, no liveness), paid per call via x402 on Base.
func NewPortrait(opts ...Option) (*Portrait, error) {
	cfg, hexKey, err := resolveKey(opts...)
	if err != nil {
		return nil, err
	}
	return blockrun.NewPortraitClient(hexKey, blockrun.WithPortraitAPIURL(cfg.apiURL))
}
