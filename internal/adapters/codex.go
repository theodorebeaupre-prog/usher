package adapters

import "time"

type codexAdapter struct{}

func (codexAdapter) Name() string           { return "codex" }
func (codexAdapter) Bin() string            { return "codex" }
func (codexAdapter) Detect() (bool, string) { return detect("codex") }

func (codexAdapter) LaunchArgs(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return []string{prompt}
}

var codexQuotaPatterns = []string{
	"usage limit",
	"rate limit",
	"429",
	"plan limit",
}

func (codexAdapter) QuotaError(exitCode int, output string) bool {
	return matchAny(output, codexQuotaPatterns)
}

func (codexAdapter) Profile() Profile {
	return Profile{
		Strengths: map[string]float64{
			"debug": 0.85, "feature": 0.80, "refactor": 0.80,
			"review": 0.95, "docs": 0.70, "test": 0.90, "other": 0.80,
		},
		QuotaWindow: 5 * time.Hour,
		InstallHint: "npm install -g @openai/codex",
	}
}
