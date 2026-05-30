package providers

import (
	"strings"

	"picoclaw/pkg/config"
)

// compatDefaults maps a protocol name to the default base URL of an
// OpenAI-compatible backend. Each entry reuses the OpenAI adapter; a model
// entry may still override base_url. The api_key env var, if any, is also
// honoured by newOpenAI's OPENAI_API_KEY fallback, so explicit keys (config or
// the shared credentials map) are preferred.
var compatDefaults = map[string]string{
	"openrouter": "https://openrouter.ai/api/v1",
	"deepseek":   "https://api.deepseek.com/v1",
	"groq":       "https://api.groq.com/openai/v1",
	"mistral":    "https://api.mistral.ai/v1",
	"moonshot":   "https://api.moonshot.cn/v1",
	"kimi":       "https://api.moonshot.cn/v1",
	"qwen":       "https://dashscope.aliyuncs.com/compatible-mode/v1",
	"zhipu":      "https://open.bigmodel.cn/api/paas/v4",
	"glm":        "https://open.bigmodel.cn/api/paas/v4",
	"xai":        "https://api.x.ai/v1",
	"together":   "https://api.together.xyz/v1",
	"novita":     "https://api.novita.ai/v3/openai",
	"cerebras":   "https://api.cerebras.ai/v1",
	"nvidia":     "https://integrate.api.nvidia.com/v1",
	"ollama":     "http://localhost:11434/v1",
	"vllm":       "http://localhost:8000/v1",
	"litellm":    "http://localhost:4000/v1",
}

func init() {
	for proto, base := range compatDefaults {
		Register(proto, compatConstructor(base))
	}
}

// compatConstructor returns a Constructor that fills in a default base URL
// before delegating to the OpenAI adapter.
func compatConstructor(defaultBase string) Constructor {
	return func(entry config.ModelEntry) (Provider, error) {
		if strings.TrimSpace(entry.BaseURL) == "" {
			entry.BaseURL = defaultBase
		}
		// Local OpenAI-compatible servers accept any bearer token; supply a
		// placeholder so the adapter doesn't reject a keyless config.
		if entry.APIKey == "" && strings.Contains(entry.BaseURL, "localhost") {
			entry.APIKey = "local"
		}
		return newOpenAI(entry)
	}
}
