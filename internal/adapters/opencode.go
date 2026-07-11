package adapters

import "time"

type opencodeAdapter struct{}

func (opencodeAdapter) Name() string           { return "opencode" }
func (opencodeAdapter) Bin() string            { return "opencode" }
func (opencodeAdapter) Detect() (bool, string) { return detect("opencode") }
func (opencodeAdapter) Installed() bool        { return installed("opencode") }

func (opencodeAdapter) LaunchArgs(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return []string{"--prompt", prompt}
}

var opencodeQuotaPatterns = []string{
	"rate limit",
	"quota",
	"429",
}

func (opencodeAdapter) QuotaError(exitCode int, output string) bool {
	return matchAny(output, opencodeQuotaPatterns)
}

func (opencodeAdapter) Profile() Profile {
	return Profile{
		Strengths: map[string]float64{
			"debug": 0.70, "feature": 0.70, "refactor": 0.70,
			"review": 0.70, "docs": 0.70, "test": 0.70, "other": 0.70,
		},
		QuotaWindow: time.Hour,
		InstallHint: "brew install sst/tap/opencode",
	}
}
