package adapters

import "time"

type claudeAdapter struct{}

func (claudeAdapter) Name() string { return "claude" }
func (claudeAdapter) Bin() string  { return "claude" }

func (claudeAdapter) Detect() (bool, string) { return detect("claude") }

func (claudeAdapter) LaunchArgs(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return []string{prompt}
}

// Patterns observed from Claude Code rate-limit failures.
var claudeQuotaPatterns = []string{
	"usage limit reached",
	"rate_limit_error",
	"rate limit",
	"limit reached",
}

func (claudeAdapter) QuotaError(exitCode int, output string) bool {
	return matchAny(output, claudeQuotaPatterns)
}

func (claudeAdapter) Profile() Profile {
	return Profile{
		Strengths: map[string]float64{
			"debug": 0.90, "feature": 0.90, "refactor": 0.95,
			"review": 0.85, "docs": 0.85, "test": 0.85, "other": 0.85,
		},
		QuotaWindow: 5 * time.Hour,
		InstallHint: "npm install -g @anthropic-ai/claude-code",
	}
}
