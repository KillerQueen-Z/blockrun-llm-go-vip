package vip

// Model describes a model published by BlockRun's public catalog.
//
// Prices are provider list prices in USD. Token prices are per one million
// tokens; the media fields use the unit named by their field. BlockRun applies
// its platform fee at request time. Treat this as a useful SDK snapshot and
// use the gateway's GET /v1/models response when an application needs the
// live catalog.
type Model struct {
	ID         string
	Name       string
	Provider   string
	Categories []string
	Pricing    ModelPricing
}

// ModelPricing holds the supported billing units for a model. A zero value
// means that unit does not apply to the model.
type ModelPricing struct {
	InputPerMillion  float64
	OutputPerMillion float64
	PerImage         float64
	PerSecond        float64
	PerThousandChars float64
	PerTrack         float64
	PerGeneration    float64
}

// Models returns a defensive copy of the BlockRun model catalog snapshot.
// It includes chat, image, video, music, speech, and sound-effect models.
func Models() []Model {
	models := make([]Model, len(modelCatalog))
	copy(models, modelCatalog)
	for i := range models {
		models[i].Categories = append([]string(nil), models[i].Categories...)
	}
	return models
}

// FindModel returns the catalog entry for id.
func FindModel(id string) (Model, bool) {
	for _, model := range modelCatalog {
		if model.ID == id {
			model.Categories = append([]string(nil), model.Categories...)
			return model, true
		}
	}
	return Model{}, false
}

// Catalog snapshot from https://blockrun.ai/api/v1/models on 2026-07-25.
// Refresh this list together with the LiteLLM catalog when the gateway adds
// models. The gateway continues to accept any supported model id verbatim.
var modelCatalog = []Model{
	{ID: "openai/gpt-5.6-sol", Name: "GPT-5.6 Sol", Provider: "openai", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 5, OutputPerMillion: 30}},
	{ID: "openai/gpt-5.6-terra", Name: "GPT-5.6 Terra", Provider: "openai", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 2.5, OutputPerMillion: 15}},
	{ID: "openai/gpt-5.6-luna", Name: "GPT-5.6 Luna", Provider: "openai", Categories: []string{"chat", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 1, OutputPerMillion: 6}},
	{ID: "openai/gpt-5.5", Name: "GPT-5.5", Provider: "openai", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 5, OutputPerMillion: 30}},
	{ID: "openai/gpt-5.5-pro", Name: "GPT-5.5 Pro", Provider: "openai", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 30, OutputPerMillion: 180}},
	{ID: "openai/chat-latest", Name: "ChatGPT Instant (GPT-5.5)", Provider: "openai", Categories: []string{"chat", "vision"}, Pricing: ModelPricing{InputPerMillion: 5, OutputPerMillion: 30}},
	{ID: "openai/gpt-5.4", Name: "GPT-5.4", Provider: "openai", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 2.5, OutputPerMillion: 15}},
	{ID: "openai/gpt-5.4-pro", Name: "GPT-5.4 Pro", Provider: "openai", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 30, OutputPerMillion: 180}},
	{ID: "openai/gpt-5.3", Name: "GPT-5.3", Provider: "openai", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 1.75, OutputPerMillion: 14}},
	{ID: "openai/gpt-5.2", Name: "GPT-5.2", Provider: "openai", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 1.75, OutputPerMillion: 14}},
	{ID: "openai/gpt-5.4-mini", Name: "GPT-5.4 Mini", Provider: "openai", Categories: []string{"chat", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 0.75, OutputPerMillion: 4.5}},
	{ID: "openai/gpt-5-mini", Name: "GPT-5 Mini", Provider: "openai", Categories: []string{"chat", "coding"}, Pricing: ModelPricing{InputPerMillion: 0.25, OutputPerMillion: 2}},
	{ID: "openai/gpt-5.4-nano", Name: "GPT-5.4 Nano", Provider: "openai", Categories: []string{"chat"}, Pricing: ModelPricing{InputPerMillion: 0.2, OutputPerMillion: 1.25}},
	{ID: "openai/gpt-5.2-pro", Name: "GPT-5.2 Pro", Provider: "openai", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 21, OutputPerMillion: 168}},
	{ID: "openai/gpt-5.3-codex", Name: "GPT-5.3 Codex", Provider: "openai", Categories: []string{"coding", "reasoning", "chat"}, Pricing: ModelPricing{InputPerMillion: 1.75, OutputPerMillion: 14}},
	{ID: "openai/gpt-4.1", Name: "GPT-4.1", Provider: "openai", Categories: []string{"chat", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 2, OutputPerMillion: 8}},
	{ID: "openai/gpt-4.1-mini", Name: "GPT-4.1 Mini", Provider: "openai", Categories: []string{"chat", "coding"}, Pricing: ModelPricing{InputPerMillion: 0.4, OutputPerMillion: 1.6}},
	{ID: "openai/gpt-4.1-nano", Name: "GPT-4.1 Nano", Provider: "openai", Categories: []string{"chat"}, Pricing: ModelPricing{InputPerMillion: 0.1, OutputPerMillion: 0.4}},
	{ID: "openai/gpt-4o", Name: "GPT-4o", Provider: "openai", Categories: []string{"chat", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 2.5, OutputPerMillion: 10}},
	{ID: "openai/gpt-4o-mini", Name: "GPT-4o Mini", Provider: "openai", Categories: []string{"chat", "coding"}, Pricing: ModelPricing{InputPerMillion: 0.15, OutputPerMillion: 0.6}},
	{ID: "openai/o1", Name: "o1", Provider: "openai", Categories: []string{"chat", "reasoning", "coding"}, Pricing: ModelPricing{InputPerMillion: 15, OutputPerMillion: 60}},
	{ID: "openai/o3", Name: "o3", Provider: "openai", Categories: []string{"chat", "reasoning", "coding"}, Pricing: ModelPricing{InputPerMillion: 2, OutputPerMillion: 8}},
	{ID: "openai/o3-mini", Name: "o3-mini", Provider: "openai", Categories: []string{"chat", "reasoning", "coding"}, Pricing: ModelPricing{InputPerMillion: 1.1, OutputPerMillion: 4.4}},
	{ID: "openai/o4-mini", Name: "o4-mini", Provider: "openai", Categories: []string{"reasoning", "coding"}, Pricing: ModelPricing{InputPerMillion: 1.1, OutputPerMillion: 4.4}},
	{ID: "anthropic/claude-haiku-4.5", Name: "Claude Haiku 4.5", Provider: "anthropic", Categories: []string{"chat", "coding"}, Pricing: ModelPricing{InputPerMillion: 1, OutputPerMillion: 5}},
	{ID: "anthropic/claude-sonnet-5", Name: "Claude Sonnet 5", Provider: "anthropic", Categories: []string{"chat", "coding", "reasoning", "vision"}, Pricing: ModelPricing{InputPerMillion: 3, OutputPerMillion: 15}},
	{ID: "anthropic/claude-sonnet-4.6", Name: "Claude Sonnet 4.6", Provider: "anthropic", Categories: []string{"chat", "coding", "reasoning"}, Pricing: ModelPricing{InputPerMillion: 3, OutputPerMillion: 15}},
	{ID: "anthropic/claude-sonnet-4.5", Name: "Claude Sonnet 4.5", Provider: "anthropic", Categories: []string{"chat", "coding", "reasoning", "vision"}, Pricing: ModelPricing{InputPerMillion: 3, OutputPerMillion: 15}},
	{ID: "anthropic/claude-opus-4.5", Name: "Claude Opus 4.5", Provider: "anthropic", Categories: []string{"chat", "coding", "reasoning", "vision"}, Pricing: ModelPricing{InputPerMillion: 5, OutputPerMillion: 25}},
	{ID: "anthropic/claude-opus-4.7", Name: "Claude Opus 4.7", Provider: "anthropic", Categories: []string{"chat", "coding", "reasoning", "vision"}, Pricing: ModelPricing{InputPerMillion: 5, OutputPerMillion: 25}},
	{ID: "anthropic/claude-fable-5", Name: "Claude Fable 5", Provider: "anthropic", Categories: []string{"chat", "coding", "reasoning", "vision"}, Pricing: ModelPricing{InputPerMillion: 10, OutputPerMillion: 50}},
	{ID: "anthropic/claude-opus-4.8", Name: "Claude Opus 4.8", Provider: "anthropic", Categories: []string{"chat", "coding", "reasoning", "vision"}, Pricing: ModelPricing{InputPerMillion: 5, OutputPerMillion: 25}},
	{ID: "anthropic/claude-opus-5", Name: "Claude Opus 5", Provider: "anthropic", Categories: []string{"chat", "coding", "reasoning", "vision"}, Pricing: ModelPricing{InputPerMillion: 5, OutputPerMillion: 25}},
	{ID: "google/gemini-3.1-pro", Name: "Gemini 3.1 Pro", Provider: "google", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 2, OutputPerMillion: 12}},
	{ID: "google/gemini-3-flash-preview", Name: "Gemini 3 Flash Preview", Provider: "google", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 0.5, OutputPerMillion: 3}},
	{ID: "google/gemini-3.5-flash", Name: "Gemini 3.5 Flash", Provider: "google", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 0.5, OutputPerMillion: 3}},
	{ID: "google/gemini-2.5-pro", Name: "Gemini 2.5 Pro", Provider: "google", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 1.25, OutputPerMillion: 10}},
	{ID: "google/gemini-2.5-flash", Name: "Gemini 2.5 Flash", Provider: "google", Categories: []string{"chat", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 0.3, OutputPerMillion: 2.5}},
	{ID: "google/gemini-3.1-flash-lite", Name: "Gemini 3.1 Flash Lite", Provider: "google", Categories: []string{"chat", "reasoning"}, Pricing: ModelPricing{InputPerMillion: 0.25, OutputPerMillion: 1.5}},
	{ID: "google/gemini-2.5-flash-lite", Name: "Gemini 2.5 Flash Lite", Provider: "google", Categories: []string{"chat"}, Pricing: ModelPricing{InputPerMillion: 0.1, OutputPerMillion: 0.4}},
	{ID: "deepseek/deepseek-v4-pro", Name: "DeepSeek V4 Pro", Provider: "deepseek", Categories: []string{"chat", "reasoning", "coding"}, Pricing: ModelPricing{InputPerMillion: 0.435, OutputPerMillion: 0.87}},
	{ID: "deepseek/deepseek-chat", Name: "DeepSeek V4 Flash Chat", Provider: "deepseek", Categories: []string{"chat", "coding"}, Pricing: ModelPricing{InputPerMillion: 0.2, OutputPerMillion: 0.4}},
	{ID: "deepseek/deepseek-reasoner", Name: "DeepSeek V4 Flash Reasoner", Provider: "deepseek", Categories: []string{"chat", "reasoning", "coding"}, Pricing: ModelPricing{InputPerMillion: 0.2, OutputPerMillion: 0.4}},
	{ID: "moonshot/kimi-k3", Name: "Kimi K3", Provider: "moonshot", Categories: []string{"chat", "reasoning", "coding", "vision"}, Pricing: ModelPricing{InputPerMillion: 3, OutputPerMillion: 15}},
	{ID: "zai/glm-5.2", Name: "GLM-5.2", Provider: "zai", Categories: []string{"chat", "reasoning", "coding"}, Pricing: ModelPricing{InputPerMillion: 1.4, OutputPerMillion: 4.4}},
	{ID: "zai/glm-5.1", Name: "GLM-5.1", Provider: "zai", Categories: []string{"chat", "reasoning", "coding"}, Pricing: ModelPricing{InputPerMillion: 1.4, OutputPerMillion: 4.4}},
	{ID: "zai/glm-5", Name: "GLM-5", Provider: "zai", Categories: []string{"chat", "reasoning", "coding"}, Pricing: ModelPricing{InputPerMillion: 0.6, OutputPerMillion: 1.92}},
	{ID: "zai/glm-5-turbo", Name: "GLM-5 Turbo", Provider: "zai", Categories: []string{"chat", "reasoning"}, Pricing: ModelPricing{InputPerMillion: 1.2, OutputPerMillion: 4}},
	{ID: "xai/grok-4.3", Name: "Grok 4.3", Provider: "xai", Categories: []string{"reasoning", "coding", "vision", "chat"}, Pricing: ModelPricing{InputPerMillion: 1.5, OutputPerMillion: 4}},
	{ID: "xai/grok-build-0.1", Name: "Grok Build 0.1", Provider: "xai", Categories: []string{"coding", "chat"}, Pricing: ModelPricing{InputPerMillion: 1.5, OutputPerMillion: 3}},
	{ID: "xai/grok-4.5", Name: "Grok 4.5", Provider: "xai", Categories: []string{"reasoning", "coding", "vision", "chat", "search"}, Pricing: ModelPricing{InputPerMillion: 2.5, OutputPerMillion: 9}},
	{ID: "minimax/minimax-m2.7", Name: "MiniMax M2.7", Provider: "minimax", Categories: []string{"chat", "reasoning", "coding"}, Pricing: ModelPricing{InputPerMillion: 0.3, OutputPerMillion: 1.2}},
	{ID: "minimax/minimax-m3", Name: "MiniMax M3", Provider: "minimax", Categories: []string{"chat", "reasoning", "coding"}, Pricing: ModelPricing{InputPerMillion: 0.3, OutputPerMillion: 1.2}},
	{ID: "qwen/qwen3.7-max", Name: "Qwen3.7 Max", Provider: "qwen", Categories: []string{"chat", "reasoning", "coding"}, Pricing: ModelPricing{InputPerMillion: 1.475, OutputPerMillion: 4.425}},
	{ID: "nvidia/deepseek-v4-flash", Name: "DeepSeek V4 Flash (Free)", Provider: "nvidia", Categories: []string{"chat", "reasoning", "coding"}},
	{ID: "nvidia/nemotron-3-nano-omni-30b-a3b-reasoning", Name: "Nemotron 3 Nano Omni (Free)", Provider: "nvidia", Categories: []string{"chat", "reasoning", "vision"}},
	{ID: "nvidia/qwen3-next-80b-a3b-instruct", Name: "Qwen3-Next 80B Instruct (Free)", Provider: "nvidia", Categories: []string{"chat", "reasoning", "coding"}},
	{ID: "nvidia/mistral-nemotron", Name: "Mistral Nemotron (Free)", Provider: "nvidia", Categories: []string{"chat", "coding"}},
	{ID: "nvidia/step-3.7-flash", Name: "StepFun Step 3.7 Flash (Free)", Provider: "nvidia", Categories: []string{"chat", "reasoning"}},
	{ID: "nvidia/seed-oss-36b", Name: "ByteDance Seed-OSS 36B (Free)", Provider: "nvidia", Categories: []string{"chat", "coding"}},
	{ID: "nvidia/nemotron-nano-9b-v2", Name: "Nemotron Nano 9B v2 (Free)", Provider: "nvidia", Categories: []string{"chat", "reasoning"}},
	{ID: "nvidia/nemotron-nano-12b-v2-vl", Name: "Nemotron Nano 12B v2 VL (Free)", Provider: "nvidia", Categories: []string{"chat", "reasoning", "vision"}},
	{ID: "openai/gpt-image-1", Name: "GPT Image 1", Provider: "openai", Categories: []string{"image"}, Pricing: ModelPricing{PerImage: 0.02}},
	{ID: "openai/gpt-image-2", Name: "ChatGPT Images 2.0", Provider: "openai", Categories: []string{"image"}, Pricing: ModelPricing{PerImage: 0.06}},
	{ID: "google/nano-banana", Name: "Nano Banana", Provider: "google", Categories: []string{"image"}, Pricing: ModelPricing{PerImage: 0.05}},
	{ID: "google/nano-banana-pro", Name: "Nano Banana Pro", Provider: "google", Categories: []string{"image"}, Pricing: ModelPricing{PerImage: 0.1}},
	{ID: "xai/grok-imagine-image", Name: "Grok Imagine", Provider: "xai", Categories: []string{"image"}, Pricing: ModelPricing{PerImage: 0.02}},
	{ID: "xai/grok-imagine-image-pro", Name: "Grok Imagine Pro", Provider: "xai", Categories: []string{"image"}, Pricing: ModelPricing{PerImage: 0.07}},
	{ID: "bytedance/seedream-5-pro", Name: "Seedream 5.0 Pro", Provider: "bytedance", Categories: []string{"image"}, Pricing: ModelPricing{PerImage: 0.045}},
	{ID: "zai/cogview-4", Name: "CogView-4", Provider: "zai", Categories: []string{"image"}, Pricing: ModelPricing{PerImage: 0.015}},
	{ID: "minimax/music-2.5+", Name: "MiniMax Music 2.5+", Provider: "minimax", Categories: []string{"music"}, Pricing: ModelPricing{PerTrack: 0.15}},
	{ID: "elevenlabs/flash-v2.5", Name: "ElevenLabs Flash v2.5", Provider: "elevenlabs", Categories: []string{"speech", "tts"}, Pricing: ModelPricing{PerThousandChars: 0.05}},
	{ID: "elevenlabs/turbo-v2.5", Name: "ElevenLabs Turbo v2.5", Provider: "elevenlabs", Categories: []string{"speech", "tts"}, Pricing: ModelPricing{PerThousandChars: 0.05}},
	{ID: "elevenlabs/multilingual-v2", Name: "ElevenLabs Multilingual v2", Provider: "elevenlabs", Categories: []string{"speech", "tts"}, Pricing: ModelPricing{PerThousandChars: 0.1}},
	{ID: "elevenlabs/v3", Name: "ElevenLabs v3", Provider: "elevenlabs", Categories: []string{"speech", "tts"}, Pricing: ModelPricing{PerThousandChars: 0.1}},
	{ID: "bytedance/seed-audio-1.0", Name: "Seed Audio 1.0", Provider: "bytedance", Categories: []string{"speech", "tts"}, Pricing: ModelPricing{PerThousandChars: 0.3}},
	{ID: "elevenlabs/sound-effects", Name: "ElevenLabs Sound Effects", Provider: "elevenlabs", Categories: []string{"sound_effect"}, Pricing: ModelPricing{PerGeneration: 0.05}},
	{ID: "xai/grok-imagine-video", Name: "Grok Imagine Video", Provider: "xai", Categories: []string{"video"}, Pricing: ModelPricing{PerSecond: 0.05}},
	{ID: "bytedance/seedance-1.5-pro", Name: "Seedance 1.5 Pro", Provider: "bytedance", Categories: []string{"video"}, Pricing: ModelPricing{PerSecond: 0.098}},
	{ID: "bytedance/seedance-2.0-fast", Name: "Seedance 2.0 Fast", Provider: "bytedance", Categories: []string{"video"}, Pricing: ModelPricing{PerSecond: 0.255}},
	{ID: "bytedance/seedance-2.0", Name: "Seedance 2.0 Pro", Provider: "bytedance", Categories: []string{"video"}, Pricing: ModelPricing{PerSecond: 0.319}},
	{ID: "azure/sora-2", Name: "Sora 2", Provider: "azure", Categories: []string{"video"}, Pricing: ModelPricing{PerSecond: 0.1}},
}
