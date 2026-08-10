package vip

import (
	"encoding/json"
	"testing"
)

func TestRouterV3GoldenTasks(t *testing.T) {
	weather := []RouterTool{{Name: "get_weather"}}
	terminal := []RouterTool{{Name: "terminalExec"}}
	orders := []RouterTool{{Name: "get_order"}, {Name: "update_address"}}
	web := []RouterTool{{Name: "web_search"}, {Name: "web_fetch"}}
	cases := []struct {
		name  string
		req   RouterRequest
		model string
		task  string
		tier  RoutingTier
	}{
		{"chat", RouterRequest{Prompt: "Hello, how are you?"}, "google/gemini-2.5-flash", "chat", TierSimple},
		{"structured-zh", RouterRequest{Prompt: "请从文本提取姓名并严格返回 JSON。", RequiresStructuredOutput: true}, "google/gemini-2.5-flash", "extraction", TierMedium},
		{"code-edit", RouterRequest{Prompt: "Implement a TypeScript API function and add tests."}, "google/gemini-3-flash-preview", "code_edit", TierMedium},
		{"debug", RouterRequest{Prompt: "Debug the failing Python tests, identify the regression, edit the files, and verify the fix."}, "openai/gpt-4o-mini", "debug", TierMedium},
		{"reasoning", RouterRequest{Prompt: "Prove formally that the square root of 2 is irrational."}, "deepseek/deepseek-v4-pro", "reasoning", TierReasoning},
		{"mcq", RouterRequest{Prompt: "Which is correct?\nA. 1\nB. 2\nC. 3\nD. 4"}, "google/gemini-3-flash-preview", "reasoning_mcq", TierReasoning},
		{"math", RouterRequest{Prompt: "A shop sells 3 books at $12 each with a 25% discount. How much is the total?"}, "deepseek/deepseek-v4-pro", "reasoning_math", TierReasoning},
		{"code-agent", RouterRequest{Prompt: "Use terminal commands to inspect the repository and edit the failing endpoint.", Tools: terminal, ToolChoice: "required"}, "openai/gpt-5-mini", "code_agent", TierSimple},
		{"tool-agent", RouterRequest{Prompt: "Check my order status and update its delivery address.", Tools: orders, ToolChoice: "required"}, "openai/gpt-5-mini", "tool_agent", TierSimple},
		{"parallel-agent", RouterRequest{Prompt: "Check London, Paris, and Tokyo weather in parallel.", Tools: weather, ToolChoice: "required"}, "anthropic/claude-opus-4.8", "tool_agent_parallel", TierSimple},
		{"deep-research", RouterRequest{Prompt: "Using multiple public sources, identify the person described by the following clues and return one exact best-supported answer: they founded a company after 2010, later joined another lab, and published work in 2024.", Tools: web, ToolChoice: "required"}, "anthropic/claude-sonnet-5", "tool_agent", TierMedium},
		{"vision", RouterRequest{Prompt: "Describe this image accurately.", HasVision: true}, "google/gemini-2.5-flash", "vision", TierSimple},
		{"premium", RouterRequest{Prompt: "Review this production payment implementation for security vulnerabilities.", Profile: RoutingPremium}, "google/gemini-2.5-flash", "chat", TierSimple},
		{"eco", RouterRequest{Prompt: "Summarize this note in one sentence.", Profile: RoutingEco}, "google/gemini-3.1-flash-lite", "chat", TierSimple},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := Route(tc.req)
			if err != nil {
				t.Fatal(err)
			}
			if decision.TaskType != tc.task || decision.Tier != tc.tier {
				t.Fatalf("got task=%s tier=%s, want task=%s tier=%s", decision.TaskType, decision.Tier, tc.task, tc.tier)
			}
			if decision.Model != tc.model {
				t.Fatalf("got model=%s, want %s", decision.Model, tc.model)
			}
			if decision.Model == "" || len(decision.Candidates) == 0 || decision.Candidates[0] != decision.Model {
				t.Fatalf("invalid ranked decision: %#v", decision)
			}
			if decision.RouterVersion != RouterVersion || decision.Method != "portfolio" {
				t.Fatalf("unexpected router contract: %#v", decision)
			}
		})
	}
}

func TestRouteOpenAIRequestRewritesAliasBeforePayment(t *testing.T) {
	body := []byte(`{"model":"blockrun/auto","messages":[{"role":"system","content":"Use tools safely"},{"role":"user","content":"Check the weather in London and Paris in parallel"}],"tools":[{"type":"function","function":{"name":"get_weather","description":"weather"}}],"tool_choice":"required","max_tokens":512}`)
	routed, decision, err := routeOpenAIRequest(body, "/v1/chat/completions")
	if err != nil {
		t.Fatal(err)
	}
	if decision == nil || decision.TaskType != "tool_agent_parallel" {
		t.Fatalf("unexpected decision: %#v", decision)
	}
	var request map[string]any
	if err := json.Unmarshal(routed, &request); err != nil {
		t.Fatal(err)
	}
	if request["model"] != decision.Model || request["model"] == "blockrun/auto" {
		t.Fatalf("alias was not rewritten: %#v", request["model"])
	}
	if _, ok := request["tools"]; !ok {
		t.Fatal("tool schema was dropped while routing")
	}
}

func TestRouteOpenAIRequestLeavesExplicitModelUntouched(t *testing.T) {
	body := []byte(`{"model":"openai/gpt-5-mini","messages":[{"role":"user","content":"hello"}]}`)
	routed, decision, err := routeOpenAIRequest(body, "/v1/chat/completions")
	if err != nil {
		t.Fatal(err)
	}
	if decision != nil || string(routed) != string(body) {
		t.Fatal("explicit model request was changed")
	}
}
