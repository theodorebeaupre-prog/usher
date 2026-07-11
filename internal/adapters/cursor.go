package adapters

import "time"

type cursorAdapter struct{}

func (cursorAdapter) Name() string           { return "cursor" }
func (cursorAdapter) Bin() string            { return "cursor-agent" }
func (cursorAdapter) Detect() (bool, string) { return detect("cursor-agent") }
func (cursorAdapter) Installed() bool        { return installed("cursor-agent") }

func (cursorAdapter) LaunchArgs(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return []string{prompt}
}

func (cursorAdapter) HeadlessArgs(prompt string) []string {
	if prompt == "" {
		return nil
	}
	return []string{"-p", prompt}
}

var cursorQuotaPatterns = []string{
	"rate limit",
	"quota",
	"usage limit",
	"429",
}

func (cursorAdapter) QuotaError(exitCode int, output string) bool {
	return matchAny(output, cursorQuotaPatterns)
}

func (cursorAdapter) Profile() Profile {
	return Profile{
		Strengths: map[string]float64{
			"debug": 0.80, "feature": 0.85, "refactor": 0.85,
			"review": 0.75, "docs": 0.70, "test": 0.80, "other": 0.80,
		},
		QuotaWindow: 5 * time.Hour,
		InstallHint: "curl https://cursor.com/install -fsS | bash",
	}
}
