package vip

import (
	blockrun "github.com/BlockRunAI/blockrun-llm-go"
)

// Image, Speech, Music, Voice, and Phone generation through the BlockRun gateway,
// paid per call in USDC (x402) on Base.
//
// Like RealFace / Portrait (see media.go), these are not official-SDK passthrough —
// there is no upstream SDK to subclass. They are BlockRun gateway clients reused
// verbatim from blockrun-llm-go, with VIP wallet resolution layered on so callers get
// the same clients, from the same package, paid by the same wallet. Every method
// returns the gateway's response verbatim.

// Client types, re-exported so callers never import blockrun-llm-go directly.
type (
	// Image generates and edits images (OpenAI gpt-image, Google nano-banana,
	// xAI grok-imagine, …). Generate runs text-to-image; Edit does image-to-image
	// and multi-image fusion.
	Image = blockrun.ImageClient
	// Speech is ElevenLabs text-to-speech (Generate) and sound effects (SoundEffect).
	Speech = blockrun.SpeechClient
	// Music generates full tracks (MiniMax) from a prompt, optional lyrics, and an
	// instrumental toggle.
	Music = blockrun.MusicClient
	// Voice places outbound AI phone calls (Bland) — Call initiates, GetCallStatus
	// polls for transcript + recording. Requires an active number from Phone.
	Voice = blockrun.VoiceClient
	// Phone provisions wallet-bound numbers (buy / renew / list / release) and runs
	// carrier + fraud lookups (Twilio).
	Phone = blockrun.PhoneClient
)

// Option/result types re-exported for ergonomic use.
type (
	ImageGenerateOptions = blockrun.ImageGenerateOptions
	ImageEditOptions     = blockrun.ImageEditOptions
	ImageData            = blockrun.ImageData
	ImageResponse        = blockrun.ImageResponse
	ImageModel           = blockrun.ImageModel

	SpeechGenerateOptions = blockrun.SpeechGenerateOptions
	SoundEffectOptions    = blockrun.SoundEffectOptions
	SpeechAudio           = blockrun.SpeechAudio
	SpeechResponse        = blockrun.SpeechResponse
	VoiceInfo             = blockrun.VoiceInfo

	MusicGenerateOptions = blockrun.MusicGenerateOptions
	AudioTrack           = blockrun.AudioTrack
	MusicResponse        = blockrun.MusicResponse
	AudioModel           = blockrun.AudioModel

	CallModel             = blockrun.CallModel
	CallOptions           = blockrun.CallOptions
	CallInitiatedResponse = blockrun.CallInitiatedResponse
	CallStatusResponse    = blockrun.CallStatusResponse

	BuyNumberOptions      = blockrun.BuyNumberOptions
	PhoneLookupResponse   = blockrun.PhoneLookupResponse
	NumberBuyResponse     = blockrun.NumberBuyResponse
	NumberRenewResponse   = blockrun.NumberRenewResponse
	NumberListResponse    = blockrun.NumberListResponse
	NumberReleaseResponse = blockrun.NumberReleaseResponse
	OwnedNumber           = blockrun.OwnedNumber
)

// NewImage returns an Image client (text-to-image + image editing), paid per call
// via x402 on Base.
//
//	img, _ := vip.NewImage()
//	out, _ := img.Generate(ctx, "a red fox in fresh snow", &vip.ImageGenerateOptions{
//	    Model: "openai/gpt-image-1",
//	})
//	fmt.Println(out.Data[0].URL)
func NewImage(opts ...Option) (*Image, error) {
	cfg, hexKey, err := resolveKey(opts...)
	if err != nil {
		return nil, err
	}
	return blockrun.NewImageClient(hexKey, blockrun.WithImageAPIURL(cfg.apiURL))
}

// NewSpeech returns a Speech client (ElevenLabs TTS + sound effects), paid per call
// via x402 on Base.
//
//	sp, _ := vip.NewSpeech()
//	out, _ := sp.Generate(ctx, "Hello there.", &vip.SpeechGenerateOptions{Voice: "sarah"})
//	fmt.Println(out.Data[0].URL)
func NewSpeech(opts ...Option) (*Speech, error) {
	cfg, hexKey, err := resolveKey(opts...)
	if err != nil {
		return nil, err
	}
	return blockrun.NewSpeechClient(hexKey, blockrun.WithSpeechAPIURL(cfg.apiURL))
}

// NewMusic returns a Music client (MiniMax track generation), paid per call via
// x402 on Base. Generation runs ~1-3 minutes; the call blocks until the track is ready.
func NewMusic(opts ...Option) (*Music, error) {
	cfg, hexKey, err := resolveKey(opts...)
	if err != nil {
		return nil, err
	}
	return blockrun.NewMusicClient(hexKey, blockrun.WithMusicAPIURL(cfg.apiURL))
}

// NewVoice returns a Voice client for outbound AI phone calls (Bland), paid per call
// via x402 on Base. Buy a number with NewPhone first; Call returns a call id, and
// GetCallStatus polls it for the transcript and recording.
func NewVoice(opts ...Option) (*Voice, error) {
	cfg, hexKey, err := resolveKey(opts...)
	if err != nil {
		return nil, err
	}
	return blockrun.NewVoiceClient(hexKey, blockrun.WithVoiceAPIURL(cfg.apiURL))
}

// NewPhone returns a Phone client for number provisioning + carrier/fraud lookups
// (Twilio), paid per call via x402 on Base.
func NewPhone(opts ...Option) (*Phone, error) {
	cfg, hexKey, err := resolveKey(opts...)
	if err != nil {
		return nil, err
	}
	return blockrun.NewPhoneClient(hexKey, blockrun.WithPhoneAPIURL(cfg.apiURL))
}
