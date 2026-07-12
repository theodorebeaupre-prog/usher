package adapters

import "time"

type ampAdapter struct{}

func (ampAdapter) Name() string           { return "amp" }
func (ampAdapter) Bin() string            { return "amp" }
func (ampAdapter) Detect() (bool, string) { return detect("amp") }
func (ampAdapter) Installed() bool        { return installed("amp") }

func (ampAdapter) LaunchArgs(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return []string{prompt}
}

func (ampAdapter) HeadlessArgs(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return []string{"-x", prompt}
}

var ampQuotaPatterns = []string{
	"rate limit",
	"quota",
	"usage limit",
	"out of free credits",
}

func (ampAdapter) QuotaError(exitCode int, output string) bool {
	return matchAny(output, ampQuotaPatterns)
}

func (ampAdapter) Profile() Profile {
	return Profile{
		Strengths: map[string]float64{
			"debug": 0.80, "feature": 0.80, "refactor": 0.80,
			"review": 0.80, "docs": 0.70, "test": 0.80, "other": 0.75,
		},
		QuotaWindow: 5 * time.Hour,
		InstallHint: "npm install -g @sourcegraph/amp",
	}
}
