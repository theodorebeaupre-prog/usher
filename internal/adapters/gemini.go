package adapters

import "time"

type geminiAdapter struct{}

func (geminiAdapter) Name() string           { return "gemini" }
func (geminiAdapter) Bin() string            { return "gemini" }
func (geminiAdapter) Detect() (bool, string) { return detect("gemini") }
func (geminiAdapter) Installed() bool        { return installed("gemini") }

func (geminiAdapter) LaunchArgs(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return []string{"-i", prompt}
}

func (geminiAdapter) HeadlessArgs(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return []string{"-p", prompt}
}

var geminiQuotaPatterns = []string{
	"resource_exhausted",
	"quota",
	"rate limit",
	"429",
}

func (geminiAdapter) QuotaError(exitCode int, output string) bool {
	return matchAny(output, geminiQuotaPatterns)
}

func (geminiAdapter) Profile() Profile {
	return Profile{
		Strengths: map[string]float64{
			"debug": 0.70, "feature": 0.75, "refactor": 0.70,
			"review": 0.70, "docs": 0.90, "test": 0.70, "other": 0.75,
		},
		QuotaWindow: 24 * time.Hour,
		InstallHint: "npm install -g @google/gemini-cli",
	}
}
