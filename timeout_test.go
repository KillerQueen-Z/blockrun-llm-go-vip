package vip

import (
	"os"
	"testing"
	"time"
)

// TestDefaultChatTimeout verifies the env-configurable default chat timeout:
// 600s when BLOCKRUN_CHAT_TIMEOUT is unset, an integer-seconds override when
// set, and a fallback to the default when the value is invalid. Reasoning
// models (opus-4.8 / deepseek-v4-pro) think 200-300s+, so the default must be
// well above the official SDKs' lower default.
func TestDefaultChatTimeout(t *testing.T) {
	if DefaultChatTimeout != 600*time.Second {
		t.Fatalf("DefaultChatTimeout = %s, want 600s", DefaultChatTimeout)
	}

	cases := []struct {
		name string
		env  string // value to set; "" means unset
		set  bool
		want time.Duration
	}{
		{name: "unset falls back to 600s", set: false, want: 600 * time.Second},
		{name: "valid override 240s", env: "240", set: true, want: 240 * time.Second},
		{name: "valid override with whitespace", env: " 300 ", set: true, want: 300 * time.Second},
		{name: "non-numeric falls back", env: "abc", set: true, want: 600 * time.Second},
		{name: "zero falls back", env: "0", set: true, want: 600 * time.Second},
		{name: "negative falls back", env: "-5", set: true, want: 600 * time.Second},
		{name: "empty falls back", env: "", set: true, want: 600 * time.Second},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv("BLOCKRUN_CHAT_TIMEOUT", tc.env)
			} else {
				// t.Setenv with unset isn't available; rely on the test process
				// not having it set. Guard by skipping if it is.
				if v := os.Getenv("BLOCKRUN_CHAT_TIMEOUT"); v != "" {
					t.Skipf("BLOCKRUN_CHAT_TIMEOUT set in env (%q); skipping unset case", v)
				}
			}
			if got := defaultChatTimeout(); got != tc.want {
				t.Fatalf("defaultChatTimeout() = %s, want %s", got, tc.want)
			}
		})
	}
}

// TestChatClientsConstructWithTimeout is a no-network smoke test that both
// passthrough chat constructors build successfully with the default timeout
// wired in (option.WithRequestTimeout is applied first inside the constructors).
func TestChatClientsConstructWithTimeout(t *testing.T) {
	if _, err := NewOpenAI(WithWalletKey(testWalletKey)); err != nil {
		t.Fatalf("NewOpenAI: %v", err)
	}
	if _, err := NewAnthropic(WithWalletKey(testWalletKey)); err != nil {
		t.Fatalf("NewAnthropic: %v", err)
	}
}
