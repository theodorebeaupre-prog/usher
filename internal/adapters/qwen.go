package adapters

import "time"

type qwenAdapter struct{}

func (qwenAdapter) Name() string           { return "qwen" }
func (qwenAdapter) Bin() string            { return "qwen" }
func (qwenAdapter) Detect() (bool, string) { return detect("qwen") }
func (qwenAdapter) Installed() bool        { return installed("qwen") }

func (qwenAdapter) LaunchArgs(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return []string{"-i", prompt}
}

func (qwenAdapter) HeadlessArgs(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return []string{"-p", prompt}
}

var qwenQuotaPatterns = []string{
	"quota",
	"rate limit",
	"429",
	"resource_exhausted",
}

func (qwenAdapter) QuotaError(exitCode int, output string) bool {
	return matchAny(output, qwenQuotaPatterns)
}

func (qwenAdapter) Profile() Profile {
	return Profile{
		Strengths: map[string]float64{
			"debug": 0.70, "feature": 0.75, "refactor": 0.75,
			"review": 0.65, "docs": 0.75, "test": 0.70, "other": 0.70,
		},
		QuotaWindow: 24 * time.Hour,
		InstallHint: "npm install -g @qwen-code/qwen-code",
	}
}
