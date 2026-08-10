package vip

// This file is the native Go adapter for BlockRun Router Core V3.  The
// product-neutral source contract is pinned to the commit below; golden tests
// keep this port aligned with the TypeScript package used by the web SDK.

import (
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	RouterCoreCommit = "d4308049348e11e17ed08a254676a34949be80f9"
	RouterVersion    = "v3-portfolio"
)

type RoutingProfile string

const (
	RoutingEco     RoutingProfile = "eco"
	RoutingAuto    RoutingProfile = "auto"
	RoutingPremium RoutingProfile = "premium"
)

type RoutingTier string

const (
	TierSimple    RoutingTier = "SIMPLE"
	TierMedium    RoutingTier = "MEDIUM"
	TierComplex   RoutingTier = "COMPLEX"
	TierReasoning RoutingTier = "REASONING"
)

type RouterTool struct {
	Name        string
	Description string
}

type RouterRequest struct {
	Prompt                   string
	SystemPrompt             string
	MaxOutputTokens          int
	Profile                  RoutingProfile
	Tools                    []RouterTool
	ToolChoice               string
	RequiresStructuredOutput bool
	HasVision                bool
	MinimumPaymentUSD        float64
	Models                   []Model
}

type CandidateScore struct {
	Model       string  `json:"model"`
	Score       float64 `json:"score"`
	Quality     float64 `json:"quality"`
	Cost        float64 `json:"cost"`
	Speed       float64 `json:"speed"`
	Reliability float64 `json:"reliability"`
}

type RoutingDecision struct {
	Model           string           `json:"model"`
	Tier            RoutingTier      `json:"tier"`
	Confidence      float64          `json:"confidence"`
	Method          string           `json:"method"`
	Reasoning       string           `json:"reasoning"`
	CostEstimate    float64          `json:"cost_estimate"`
	BaselineCost    float64          `json:"baseline_cost"`
	Savings         float64          `json:"savings"`
	Profile         string           `json:"profile"`
	TaskType        string           `json:"task_type"`
	RouterVersion   string           `json:"router_version"`
	Candidates      []string         `json:"candidates"`
	CandidateScores []CandidateScore `json:"candidate_scores"`
	Fallbacks       []string         `json:"fallbacks"`
}

type routeFeatures struct {
	task          string
	inputTokens   int
	needsTools    bool
	parallel      bool
	languageZH    bool
	terminal      bool
	highStakes    bool
	structured    bool
	vision        bool
	deepResearch  bool
	domain        string
	complexAction bool
	agenticIntent bool
}

var (
	codePattern       = regexp.MustCompile(`(?i)\x60\x60\x60|\b(typescript|javascript|python|rust|java|sql|stack trace|traceback|exception)\b|\.(ts|tsx|js|py|go|rs)\b`)
	codeActionPattern = regexp.MustCompile(`(?is)\b(implement|refactor|debug|write|edit|modify|create|define|review|fix)\b.{0,48}\b(api|function|class|method)\b`)
	toolVerbPattern   = regexp.MustCompile(`(?i)\b(get|fetch|search|look up|check|update|change|create|delete|cancel|book|send|run|execute|open|read|write|edit|deploy|install)\b|查询|搜索|查看|获取|更新|修改|创建|删除|取消|预订|发送|执行|打开|读取|写入|部署|安装`)
	parallelPattern   = regexp.MustCompile(`(?i)\b(in parallel|simultaneously|concurrently|for each|each of|every one|both|(two|three|multiple|several)\s+(cities|locations|items|tasks|orders|users|files))\b|并行|同时|分别|每个|各自|(两个|三个|多个)(城市|地点|项目|任务|订单|用户|文件)`)
	highRiskPattern   = regexp.MustCompile(`(?i)\b(production|security|payment|legal|medical|financial|audit)\b|生产|安全|支付|法律|医疗|财务|审计`)
)

var autoPortfolio = map[RoutingTier][]string{
	TierSimple:    {"google/gemini-2.5-flash", "google/gemini-3-flash-preview", "deepseek/deepseek-chat", "moonshot/kimi-k2.5", "google/gemini-3.1-flash-lite", "google/gemini-2.5-flash-lite", "openai/gpt-5.4-nano", "xai/grok-4-fast-non-reasoning", "free/gpt-oss-120b"},
	TierMedium:    {"moonshot/kimi-k2.7", "moonshot/kimi-k2.6", "moonshot/kimi-k2.5", "google/gemini-3-flash-preview", "deepseek/deepseek-chat", "google/gemini-2.5-flash", "google/gemini-3.1-flash-lite", "google/gemini-2.5-flash-lite", "xai/grok-4-1-fast-non-reasoning", "xai/grok-3-mini"},
	TierComplex:   {"google/gemini-3.1-pro", "google/gemini-3-flash-preview", "xai/grok-4-0709", "google/gemini-2.5-pro", "anthropic/claude-sonnet-5", "anthropic/claude-sonnet-4.6", "deepseek/deepseek-chat", "google/gemini-2.5-flash", "openai/gpt-5.6-terra", "openai/gpt-5.5", "openai/gpt-5.4"},
	TierReasoning: {"xai/grok-4-1-fast-reasoning", "xai/grok-4-fast-reasoning", "deepseek/deepseek-reasoner", "deepseek/deepseek-v4-pro", "openai/o4-mini", "openai/o3"},
}

var ecoPortfolio = map[RoutingTier][]string{
	TierSimple:    {"free/gpt-oss-120b", "free/gpt-oss-20b", "free/deepseek-v4-flash", "google/gemini-3.1-flash-lite", "openai/gpt-5.4-nano", "google/gemini-2.5-flash-lite", "xai/grok-4-fast-non-reasoning"},
	TierMedium:    {"google/gemini-3.1-flash-lite", "openai/gpt-5.4-nano", "google/gemini-2.5-flash-lite", "xai/grok-4-fast-non-reasoning", "google/gemini-2.5-flash"},
	TierComplex:   {"google/gemini-3.1-flash-lite", "google/gemini-2.5-flash-lite", "xai/grok-4-0709", "google/gemini-2.5-flash", "deepseek/deepseek-chat"},
	TierReasoning: {"xai/grok-4-1-fast-reasoning", "xai/grok-4-fast-reasoning", "deepseek/deepseek-reasoner", "deepseek/deepseek-v4-pro"},
}

var premiumPortfolio = map[RoutingTier][]string{
	TierSimple:    {"moonshot/kimi-k2.7", "moonshot/kimi-k2.6", "moonshot/kimi-k2.5", "google/gemini-2.5-flash", "anthropic/claude-haiku-4.5", "google/gemini-2.5-flash-lite", "deepseek/deepseek-chat"},
	TierMedium:    {"openai/gpt-5.3-codex", "moonshot/kimi-k2.7", "moonshot/kimi-k2.6", "moonshot/kimi-k2.5", "google/gemini-2.5-flash", "google/gemini-2.5-pro", "xai/grok-4-0709", "anthropic/claude-sonnet-5", "anthropic/claude-sonnet-4.6"},
	TierComplex:   {"anthropic/claude-fable-5", "anthropic/claude-opus-5", "anthropic/claude-opus-4.8", "anthropic/claude-opus-4.7", "anthropic/claude-sonnet-5", "anthropic/claude-sonnet-4.6", "xai/grok-4.5", "moonshot/kimi-k2.7", "openai/gpt-5.6-terra", "openai/gpt-5.5", "openai/gpt-5.4", "openai/gpt-5.3-codex", "deepseek/deepseek-chat"},
	TierReasoning: {"anthropic/claude-sonnet-4.6", "anthropic/claude-sonnet-5", "anthropic/claude-opus-5", "anthropic/claude-opus-4.8", "anthropic/claude-opus-4.7", "xai/grok-4-1-fast-reasoning", "openai/o4-mini", "openai/o3"},
}

var agentPortfolio = map[RoutingTier][]string{
	TierSimple:    {"openai/gpt-4o-mini", "moonshot/kimi-k2.5", "anthropic/claude-haiku-4.5", "xai/grok-4-1-fast-non-reasoning"},
	TierMedium:    {"moonshot/kimi-k2.7", "moonshot/kimi-k2.6", "moonshot/kimi-k2.5", "xai/grok-4-1-fast-non-reasoning", "openai/gpt-4o-mini", "anthropic/claude-haiku-4.5", "deepseek/deepseek-chat"},
	TierComplex:   {"anthropic/claude-sonnet-4.6", "anthropic/claude-sonnet-5", "anthropic/claude-opus-5", "anthropic/claude-opus-4.8", "anthropic/claude-opus-4.7", "anthropic/claude-opus-4.6", "xai/grok-4-0709", "moonshot/kimi-k2.7", "moonshot/kimi-k2.5", "openai/gpt-5.6-terra", "openai/gpt-5.5", "openai/gpt-5.4", "deepseek/deepseek-chat", "free/gpt-oss-120b"},
	TierReasoning: {"anthropic/claude-sonnet-4.6", "anthropic/claude-sonnet-5", "anthropic/claude-opus-5", "anthropic/claude-opus-4.8", "anthropic/claude-opus-4.7", "anthropic/claude-opus-4.6", "xai/grok-4-1-fast-reasoning", "deepseek/deepseek-reasoner"},
}

var evidenceCandidates = map[string][]string{
	"code_agent":          {"openai/gpt-5-mini", "openai/gpt-5.3-codex", "anthropic/claude-sonnet-5", "google/gemini-3.5-flash", "moonshot/kimi-k3", "deepseek/deepseek-v4-pro", "zai/glm-5.2"},
	"tool_agent":          {"openai/gpt-5-mini", "anthropic/claude-sonnet-5", "google/gemini-3.5-flash", "deepseek/deepseek-v4-pro", "openai/gpt-5.3-codex"},
	"tool_agent_parallel": {"anthropic/claude-opus-4.8", "anthropic/claude-sonnet-5", "xai/grok-4.5", "google/gemini-3.5-flash", "deepseek/deepseek-v4-pro", "openai/gpt-5-mini"},
	"reasoning_mcq":       {"google/gemini-3-flash-preview", "google/gemini-3.5-flash", "xai/grok-4.5", "anthropic/claude-sonnet-5", "deepseek/deepseek-v4-pro"},
	"reasoning_math":      {"deepseek/deepseek-v4-pro", "google/gemini-3.5-flash", "xai/grok-4.5", "anthropic/claude-sonnet-5", "moonshot/kimi-k3"},
	"long_context":        {"google/gemini-3.1-pro", "google/gemini-3.5-flash", "deepseek/deepseek-v4-pro", "zai/glm-5.2"},
}

var noToolModels = map[string]bool{
	"free/deepseek-v4-flash":             true,
	"free/gpt-oss-120b":                  true,
	"free/gpt-oss-20b":                   true,
	"free/seed-oss-36b":                  true,
	"google/gemini-3-flash-preview":      true,
	"nvidia/deepseek-v4-flash":           true,
	"nvidia/step-3.7-flash":              true,
	"nvidia/nemotron-nano-9b-v2":         true,
	"nvidia/qwen3-next-80b-a3b-instruct": true,
}

func RouterProfileForModel(model string) (RoutingProfile, bool) {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "blockrun/auto":
		return RoutingAuto, true
	case "blockrun/eco":
		return RoutingEco, true
	case "blockrun/premium":
		return RoutingPremium, true
	default:
		return "", false
	}
}

func Route(req RouterRequest) (RoutingDecision, error) {
	if req.MaxOutputTokens <= 0 {
		req.MaxOutputTokens = 1024
	}
	if req.Profile == "" {
		req.Profile = RoutingAuto
	}
	if req.Profile != RoutingAuto && req.Profile != RoutingEco && req.Profile != RoutingPremium {
		return RoutingDecision{}, errors.New("router: profile must be auto, eco, or premium")
	}
	if req.MinimumPaymentUSD <= 0 {
		req.MinimumPaymentUSD = 0.001
	}
	models := req.Models
	if len(models) == 0 {
		models = Models()
	}
	f := classifyRouterRequest(req)
	tier := tierForFeatures(f)
	portfolio, decisionProfile := autoPortfolio, string(req.Profile)
	if req.Profile == RoutingEco {
		portfolio = ecoPortfolio
	} else if req.Profile == RoutingPremium {
		portfolio = premiumPortfolio
	} else if f.needsTools || f.agenticIntent {
		portfolio, decisionProfile = agentPortfolio, "agentic"
	}
	available := make(map[string]Model, len(models))
	for _, model := range models {
		if isChatModel(model) {
			available[model.ID] = model
		}
	}
	chain := uniqueStrings(append(append([]string{}, portfolio[tier]...), evidenceCandidates[f.task]...))
	eligible := make([]string, 0, len(chain))
	for _, id := range chain {
		model, ok := available[id]
		if ok && eligibleModel(model, f, req.MaxOutputTokens) {
			eligible = append(eligible, id)
		}
	}
	if len(eligible) == 0 {
		return RoutingDecision{}, errors.New("router: no capability-eligible model in the current catalog")
	}

	quality := make(map[string]float64, len(eligible))
	best := 0.0
	for _, id := range eligible {
		quality[id] = modelAffinity(id, f)
		best = math.Max(best, quality[id])
	}
	gap := map[RoutingProfile]float64{RoutingAuto: .10, RoutingEco: .22, RoutingPremium: .05}[req.Profile]
	if f.terminal {
		gap = math.Max(gap, .12)
	}
	pool := make([]string, 0, len(eligible))
	for _, id := range eligible {
		if quality[id] > .68 && quality[id] >= best-gap {
			pool = append(pool, id)
		}
	}
	if len(pool) == 0 {
		pool = eligible[:1]
	}
	minCost, maxCost := math.Inf(1), 0.0
	costs := make(map[string]float64, len(pool))
	for _, id := range pool {
		costs[id] = estimatedModelCost(available[id], f.inputTokens, req.MaxOutputTokens)
		minCost = math.Min(minCost, costs[id])
		maxCost = math.Max(maxCost, costs[id])
	}
	weights := map[RoutingProfile][6]float64{
		RoutingAuto:    {.47, .20, .18, .07, .03, .05},
		RoutingEco:     {.36, .20, .28, .10, .04, .02},
		RoutingPremium: {.58, .20, .08, .06, .06, .02},
	}[req.Profile]
	scores := make([]CandidateScore, 0, len(pool))
	for i, id := range pool {
		costScore := .5
		if maxCost > minCost {
			costScore = 1 - (costs[id]-minCost)/(maxCost-minCost)
		}
		legacy := 1 - float64(i)/math.Max(1, float64(len(pool)-1))
		qualityWeight := weights[0]
		if f.highStakes {
			qualityWeight += .08
		}
		scores = append(scores, CandidateScore{
			Model: id, Quality: quality[id], Cost: costScore, Speed: .5, Reliability: 1,
			Score: quality[id]*qualityWeight + weights[1] + costScore*weights[2] + .5*weights[3] + weights[4] + legacy*weights[5],
		})
	}
	sort.SliceStable(scores, func(i, j int) bool { return scores[i].Score > scores[j].Score })
	ranked := make([]string, 0, len(eligible))
	for _, score := range scores {
		ranked = append(ranked, score.Model)
	}
	for _, id := range eligible {
		if !contains(ranked, id) {
			ranked = append(ranked, id)
		}
	}
	selected := ranked[0]
	costEstimate := math.Max(costs[selected]*1.05, req.MinimumPaymentUSD)
	baseline := (float64(f.inputTokens)*5 + float64(req.MaxOutputTokens)*30) / 1_000_000
	savings := 0.0
	if req.Profile != RoutingPremium && baseline > 0 {
		savings = math.Max(0, (baseline-costEstimate)/baseline)
	}
	return RoutingDecision{
		Model: selected, Tier: tier, Confidence: routerConfidence(f), Method: "portfolio",
		Reasoning: "constraint-first local portfolio ranking", CostEstimate: costEstimate,
		BaselineCost: baseline, Savings: savings, Profile: decisionProfile, TaskType: f.task,
		RouterVersion: RouterVersion, Candidates: ranked, CandidateScores: scores,
		Fallbacks: append([]string(nil), ranked[1:]...),
	}, nil
}

func classifyRouterRequest(req RouterRequest) routeFeatures {
	text := req.SystemPrompt + " " + req.Prompt
	lower := strings.ToLower(text)
	promptLower := strings.ToLower(req.Prompt)
	f := routeFeatures{
		task: "chat", inputTokens: (utf8.RuneCountInString(text) + 3) / 4,
		languageZH: regexp.MustCompile(`[\x{3400}-\x{9fff}]`).MatchString(text),
		structured: req.RequiresStructuredOutput, vision: req.HasVision,
		highStakes: highRiskPattern.MatchString(text),
	}
	toolNames := make([]string, 0, len(req.Tools))
	for _, tool := range req.Tools {
		toolNames = append(toolNames, strings.ToLower(tool.Name))
	}
	f.needsTools = len(req.Tools) > 0 && req.ToolChoice != "none" && (req.ToolChoice == "required" || toolVerbPattern.MatchString(req.Prompt))
	f.parallel = f.needsTools && isParallelRequest(req.Prompt)
	f.terminal = anyName(toolNames, "terminalexec", "terminalinspect", "terminalsendkeys")
	f.domain = toolDomain(toolNames)
	f.deepResearch = f.domain == "web_research" && (strings.Contains(lower, "multiple public sources") || strings.Contains(lower, "best-supported answer") || len(req.Prompt) >= 320)
	f.complexAction = f.needsTools && (strings.Contains(lower, "all reservations") || strings.Contains(lower, "every passenger") || strings.Contains(lower, "multiple files") || strings.Contains(lower, "fix all"))
	f.agenticIntent = countAgenticSignals(lower) >= 3
	hasCode := codePattern.MatchString(req.Prompt) || codeActionPattern.MatchString(req.Prompt)
	switch {
	case f.vision:
		f.task = "vision"
	case f.inputTokens > 80_000:
		f.task = "long_context"
	case f.needsTools && (hasCode || f.terminal):
		f.task = "code_agent"
	case f.needsTools && f.parallel:
		f.task = "tool_agent_parallel"
	case f.needsTools:
		f.task = "tool_agent"
	case multipleChoice(req.Prompt):
		f.task = "reasoning_mcq"
	case compactMath(req.Prompt, hasCode):
		f.task = "reasoning_math"
	case regexp.MustCompile(`(?i)\b(bug|debug|error|failure|failing|regression|crash)\b|修复|报错|错误|调试`).MatchString(promptLower):
		f.task = "debug"
	case hasCode || regexp.MustCompile(`(?i)\b(refactor|implement|patch|edit|rewrite)\b|重构|实现|修改`).MatchString(promptLower):
		f.task = "code_edit"
	case f.structured || regexp.MustCompile(`(?i)\b(extract|json|schema|csv)\b|字段|提取`).MatchString(text):
		f.task = "extraction"
	case regexp.MustCompile(`(?i)\b(prove|derive|theorem|formal|mathematical|reasoning)\b|证明|推导|定理|数学`).MatchString(text):
		f.task = "reasoning"
	}
	return f
}

func isParallelRequest(prompt string) bool {
	if parallelPattern.MatchString(prompt) {
		return true
	}
	lookup := regexp.MustCompile(`(?i)\b(weather|climate|temperature|news|report)\b|天气|气象|温度|新闻|报告`).MatchString(prompt)
	return lookup && (len(regexp.MustCompile(`[,，]`).FindAllString(prompt, -1)) >= 2 || regexp.MustCompile(`(?i)\band\b|以及|和|、`).MatchString(prompt))
}

func countAgenticSignals(text string) int {
	keywords := []string{
		"read file", "read the file", "look at", "check the", "open the", "edit", "modify",
		"update the", "change the", "write to", "create file", "execute", "deploy", "install",
		"npm", "pip", "compile", "after that", "and also", "once done", "step 1", "step 2",
		"fix", "debug", "until it works", "keep trying", "iterate", "make sure", "verify", "confirm",
	}
	count := 0
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			count++
		}
	}
	return count
}

func tierForFeatures(f routeFeatures) RoutingTier {
	switch f.task {
	case "reasoning", "reasoning_mcq", "reasoning_math":
		return TierReasoning
	case "long_context":
		return TierComplex
	case "code_edit", "debug", "extraction":
		return TierMedium
	case "tool_agent", "tool_agent_parallel", "code_agent":
		if f.complexAction || f.deepResearch {
			return TierMedium
		}
		return TierSimple
	default:
		return TierSimple
	}
}

func modelAffinity(id string, f routeFeatures) float64 {
	name := strings.TrimPrefix(id, strings.SplitN(id, "/", 2)[0]+"/")
	lookup := func(values map[string]float64) float64 {
		if value := values[name]; value > .68 {
			return value
		}
		return .68
	}
	switch f.task {
	case "code_agent":
		return lookup(map[string]float64{"gpt-5.3-codex": 1, "claude-sonnet-5": .98, "gpt-5-mini": .96, "gemini-3.5-flash": .92, "kimi-k3": .90, "deepseek-v4-pro": .88, "glm-5.2": .88})
	case "tool_agent":
		if f.domain == "web_research" {
			return lookup(map[string]float64{"claude-sonnet-5": 1, "gpt-5-mini": .88, "gemini-3.5-flash": .86})
		}
		return lookup(map[string]float64{"gpt-5-mini": 1, "claude-sonnet-5": .90, "gemini-3.5-flash": .82, "deepseek-v4-pro": .82})
	case "tool_agent_parallel":
		return lookup(map[string]float64{"claude-opus-4.8": 1, "claude-sonnet-5": .84, "grok-4.5": .82, "gemini-3.5-flash": .80, "deepseek-v4-pro": .78})
	case "code_edit", "debug":
		return lookup(map[string]float64{"gpt-5.3-codex": 1, "claude-sonnet-4.6": .94, "glm-5.2": .90, "kimi-k2.7": .86, "deepseek-v4-pro": .86})
	case "reasoning":
		return lookup(map[string]float64{"claude-sonnet-5": .98, "claude-sonnet-4.6": .98, "deepseek-v4-pro": .95, "grok-4.5": .94, "gemini-3.1-pro": .92, "gemini-3.5-flash": .92})
	case "reasoning_mcq":
		return lookup(map[string]float64{"gemini-3-flash-preview": 1, "gemini-3.5-flash": .91, "grok-4.5": .90, "claude-sonnet-5": .88, "deepseek-v4-pro": .84})
	case "reasoning_math":
		return lookup(map[string]float64{"gemini-3.5-flash": 1, "grok-4.5": .93, "claude-sonnet-5": .90, "deepseek-v4-pro": .90, "kimi-k3": .90})
	case "vision":
		return lookup(map[string]float64{"gemini-3.1-pro": .96, "claude-sonnet-4.6": .90, "kimi-k3": .90, "grok-4.3": .90})
	case "long_context":
		return lookup(map[string]float64{"gemini-3.1-pro": 1, "glm-5.2": .89, "gemini-3.5-flash": .88, "deepseek-v4-pro": .85})
	case "extraction":
		kimi := .90
		if f.languageZH {
			kimi = 1
		}
		return lookup(map[string]float64{"gemini-3.5-flash": .90, "gemini-2.5-flash": .90, "gpt-4o-mini": .90, "claude-sonnet-5": .90, "kimi-k3": kimi, "kimi-k2.7": kimi})
	default:
		return lookup(map[string]float64{"gemini-3.5-flash": .86, "gemini-2.5-flash": .86, "kimi-k3": .86, "kimi-k2.7": .86})
	}
}

func eligibleModel(model Model, f routeFeatures, maxOutput int) bool {
	if model.MaxOutput > 0 && model.MaxOutput < maxOutput {
		return false
	}
	if model.ContextWindow > 0 && float64(model.ContextWindow) < float64(f.inputTokens+maxOutput)*1.1 {
		return false
	}
	if f.vision && !contains(model.Categories, "vision") {
		return false
	}
	if f.needsTools && (strings.HasPrefix(model.ID, "free/") || noToolModels[model.ID]) {
		return false
	}
	return true
}

func isChatModel(model Model) bool {
	return contains(model.Categories, "chat") || contains(model.Categories, "coding") || contains(model.Categories, "reasoning")
}

func estimatedModelCost(model Model, input, output int) float64 {
	return (float64(input)*model.Pricing.InputPerMillion + float64(output)*model.Pricing.OutputPerMillion) / 1_000_000
}

func routerConfidence(f routeFeatures) float64 {
	if f.inputTokens > 100_000 {
		return .95
	}
	if f.task == "chat" {
		return .72
	}
	return .84
}

func multipleChoice(text string) bool {
	return len(regexp.MustCompile(`(?im)^\s*[A-D][.)]\s+`).FindAllString(text, -1)) >= 3
}

func compactMath(text string, hasCode bool) bool {
	if hasCode || len(text) >= 2500 {
		return false
	}
	numbers := regexp.MustCompile(`-?\d+(?:[.,]\d+)?`).FindAllString(text, -1)
	return len(numbers) >= 2 && regexp.MustCompile(`(?i)[+×÷=%$€£¥]|\b(total|each|per|times|half|twice|percent|how many|how much|calculate)\b|[?？]\s*$`).MatchString(text)
}

func toolDomain(names []string) string {
	for _, name := range names {
		if name == "web_search" || name == "websearch" || name == "web_fetch" || name == "webfetch" {
			return "web_research"
		}
	}
	for _, name := range names {
		if strings.Contains(name, "flight") || strings.Contains(name, "reservation") || strings.Contains(name, "airport") {
			return "airline"
		}
		if strings.Contains(name, "order") || strings.Contains(name, "product") || strings.Contains(name, "return") {
			return "retail"
		}
	}
	return "other"
}

func anyName(names []string, targets ...string) bool {
	for _, name := range names {
		for _, target := range targets {
			if name == target {
				return true
			}
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !contains(out, value) {
			out = append(out, value)
		}
	}
	return out
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// routeOpenAIRequest resolves an alias in an OpenAI-compatible request body.
// It is deliberately called before the first unpaid x402 probe.
func routeOpenAIRequest(body []byte, path string) ([]byte, *RoutingDecision, error) {
	if !strings.Contains(path, "/chat/completions") {
		return body, nil, nil
	}
	var request map[string]any
	if err := json.Unmarshal(body, &request); err != nil {
		return body, nil, nil
	}
	model, _ := request["model"].(string)
	profile, ok := RouterProfileForModel(model)
	if !ok {
		return body, nil, nil
	}
	prompt, system, vision := openAIMessages(request["messages"])
	tools := openAITools(request["tools"])
	toolChoice, _ := request["tool_choice"].(string)
	if _, requiredObject := request["tool_choice"].(map[string]any); requiredObject {
		toolChoice = "required"
	}
	maxOutput := numberAsInt(request["max_tokens"])
	if maxOutput == 0 {
		maxOutput = numberAsInt(request["max_completion_tokens"])
	}
	decision, err := Route(RouterRequest{
		Prompt: prompt, SystemPrompt: system, MaxOutputTokens: maxOutput, Profile: profile,
		Tools: tools, ToolChoice: toolChoice, RequiresStructuredOutput: request["response_format"] != nil,
		HasVision: vision,
	})
	if err != nil {
		return nil, nil, err
	}
	request["model"] = decision.Model
	routed, err := json.Marshal(request)
	return routed, &decision, err
}

func openAIMessages(raw any) (string, string, bool) {
	messages, _ := raw.([]any)
	var prompt string
	var systems []string
	vision := false
	for _, rawMessage := range messages {
		message, _ := rawMessage.(map[string]any)
		role, _ := message["role"].(string)
		text, hasVision := messageContent(message["content"])
		vision = vision || hasVision
		if role == "user" && text != "" {
			prompt = text
		} else if (role == "system" || role == "developer") && text != "" {
			systems = append(systems, text)
		}
	}
	return prompt, strings.Join(systems, "\n"), vision
}

func messageContent(raw any) (string, bool) {
	if text, ok := raw.(string); ok {
		return text, false
	}
	parts, _ := raw.([]any)
	texts := make([]string, 0, len(parts))
	vision := false
	for _, rawPart := range parts {
		part, _ := rawPart.(map[string]any)
		kind, _ := part["type"].(string)
		if kind == "image_url" || kind == "input_image" || kind == "image" {
			vision = true
		}
		if text, ok := part["text"].(string); ok {
			texts = append(texts, text)
		}
	}
	return strings.Join(texts, "\n"), vision
}

func openAITools(raw any) []RouterTool {
	items, _ := raw.([]any)
	tools := make([]RouterTool, 0, len(items))
	for _, rawItem := range items {
		item, _ := rawItem.(map[string]any)
		fn, _ := item["function"].(map[string]any)
		name, _ := fn["name"].(string)
		description, _ := fn["description"].(string)
		if name != "" {
			tools = append(tools, RouterTool{Name: name, Description: description})
		}
	}
	return tools
}

func numberAsInt(raw any) int {
	switch value := raw.(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return 0
	}
}
