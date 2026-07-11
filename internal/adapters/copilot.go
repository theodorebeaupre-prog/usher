package adapters

import "time"

type copilotAdapter struct{}

func (copilotAdapter) Name() string           { return "copilot" }
func (copilotAdapter) Bin() string            { return "copilot" }
func (copilotAdapter) Detect() (bool, string) { return detect("copilot") }
func (copilotAdapter) Installed() bool        { return installed("copilot") }

func (copilotAdapter) LaunchArgs(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return []string{prompt}
}

func (copilotAdapter) HeadlessArgs(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return []string{"-p", prompt}
}

var copilotQuotaPatterns = []string{
	"rate limit",
	"quota",
	"usage limit",
	"premium request",
}

func (copilotAdapter) QuotaError(exitCode int, output string) bool {
	return matchAny(output, copilotQuotaPatterns)
}

func (copilotAdapter) Profile() Profile {
	return Profile{
		Strengths: map[string]float64{
			"debug": 0.80, "feature": 0.80, "refactor": 0.75,
			"review": 0.85, "docs": 0.75, "test": 0.85, "other": 0.75,
		},
		QuotaWindow: 6 * time.Hour,
		InstallHint: "npm install -g @github/copilot",
	}
}
