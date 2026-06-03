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
