package llm

type Preset struct {
	BaseURL string
	EnvVar  string
}

var presets = map[string]Preset{
	"openai": {
		BaseURL: "https://api.openai.com/v1",
		EnvVar:  "OPENAI_API_KEY",
	},
	"gemini": {
		BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
		EnvVar:  "GEMINI_API_KEY",
	},
	"mistral": {
		BaseURL: "https://api.mistral.ai/v1",
		EnvVar:  "MISTRAL_API_KEY",
	},
	"groq": {
		BaseURL: "https://api.groq.com/openai/v1",
		EnvVar:  "GROQ_API_KEY",
	},
	"grok": {
		BaseURL: "https://api.x.ai/v1",
		EnvVar:  "XAI_API_KEY",
	},
}
