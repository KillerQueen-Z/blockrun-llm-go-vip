package vip

import "testing"

// Construction smoke tests — no network, no spend. They verify the wallet key
// is wired into each reused blockrun-llm-go client and the gateway base URL
// override is accepted.
func TestNewMediaClients_Construct(t *testing.T) {
	opts := []Option{
		WithWalletKey(testWalletKey),
		WithBaseURL("https://blockrun.ai/api"),
	}

	video, err := NewVideo(opts...)
	if err != nil || video == nil {
		t.Fatalf("NewVideo: client=%v err=%v", video, err)
	}
	rf, err := NewRealFace(opts...)
	if err != nil || rf == nil {
		t.Fatalf("NewRealFace: client=%v err=%v", rf, err)
	}
	portrait, err := NewPortrait(opts...)
	if err != nil || portrait == nil {
		t.Fatalf("NewPortrait: client=%v err=%v", portrait, err)
	}
}

func TestNewMediaClients_BadKey(t *testing.T) {
	if _, err := NewVideo(WithWalletKey("not-a-hex-key")); err == nil {
		t.Fatal("NewVideo: want error on invalid wallet key, got nil")
	}
}

func TestNewGenerationClients_Construct(t *testing.T) {
	opts := []Option{
		WithWalletKey(testWalletKey),
		WithBaseURL("https://blockrun.ai/api"),
	}

	if c, err := NewImage(opts...); err != nil || c == nil {
		t.Fatalf("NewImage: client=%v err=%v", c, err)
	}
	if c, err := NewSpeech(opts...); err != nil || c == nil {
		t.Fatalf("NewSpeech: client=%v err=%v", c, err)
	}
	if c, err := NewMusic(opts...); err != nil || c == nil {
		t.Fatalf("NewMusic: client=%v err=%v", c, err)
	}
	if c, err := NewVoice(opts...); err != nil || c == nil {
		t.Fatalf("NewVoice: client=%v err=%v", c, err)
	}
	if c, err := NewPhone(opts...); err != nil || c == nil {
		t.Fatalf("NewPhone: client=%v err=%v", c, err)
	}
	if c, err := NewSearch(opts...); err != nil || c == nil {
		t.Fatalf("NewSearch: client=%v err=%v", c, err)
	}
	if c, err := NewExa(opts...); err != nil || c == nil {
		t.Fatalf("NewExa: client=%v err=%v", c, err)
	}
}

func TestBuildVideoBody(t *testing.T) {
	// minimal: just the prompt
	b, err := buildVideoBody("a corgi surfing", nil)
	if err != nil || b["prompt"] != "a corgi surfing" || len(b) != 1 {
		t.Fatalf("minimal: body=%v err=%v", b, err)
	}

	// new Seedance fields are wired through
	seed := 7
	b, err = buildVideoBody("x", &VideoGenerateOptions{
		Model:        "bytedance/seedance-2.0",
		ImageURL:     "https://e/first.jpg",
		LastFrameURL: "https://e/last.jpg",
		AspectRatio:  "9:16",
		Seed:         &seed,
	})
	if err != nil {
		t.Fatalf("fields: %v", err)
	}
	if b["last_frame_url"] != "https://e/last.jpg" || b["aspect_ratio"] != "9:16" || b["seed"] != 7 {
		t.Fatalf("new fields not wired: %v", b)
	}

	// omni references pass through
	b, err = buildVideoBody("x", &VideoGenerateOptions{
		Model:              "bytedance/seedance-2.0",
		ReferenceImageURLs: []string{"https://e/a.jpg", "https://e/b.jpg"},
	})
	if err != nil || len(b["reference_image_urls"].([]string)) != 2 {
		t.Fatalf("omni: body=%v err=%v", b, err)
	}

	// LastFrameURL requires ImageURL
	if _, err := buildVideoBody("x", &VideoGenerateOptions{LastFrameURL: "https://e/last.jpg"}); err == nil {
		t.Fatal("want error: LastFrameURL without ImageURL")
	}

	// the three seed inputs are mutually exclusive
	if _, err := buildVideoBody("x", &VideoGenerateOptions{ImageURL: "u", RealFaceAssetID: "ta_x"}); err == nil {
		t.Fatal("want error: ImageURL + RealFaceAssetID")
	}
	if _, err := buildVideoBody("x", &VideoGenerateOptions{ImageURL: "u", ReferenceImageURLs: []string{"a"}}); err == nil {
		t.Fatal("want error: ImageURL + ReferenceImageURLs")
	}

	// at most 9 reference images
	if _, err := buildVideoBody("x", &VideoGenerateOptions{ReferenceImageURLs: make([]string, 10)}); err == nil {
		t.Fatal("want error: >9 reference images")
	}

	// RealFaceAssetID must carry the ta_ prefix
	if _, err := buildVideoBody("x", &VideoGenerateOptions{RealFaceAssetID: "bad"}); err == nil {
		t.Fatal("want error: RealFaceAssetID without ta_ prefix")
	}
}
