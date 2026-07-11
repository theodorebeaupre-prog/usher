package router

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct{ task, want string }{
		{"fix the flaky auth test in photocull", "debug"},
		{"debug why the ledger file is corrupt", "debug"},
		{"there's a crash on startup", "debug"},
		{"review this PR for security issues", "review"},
		{"audit the launch package", "review"},
		{"refactor the parser into two files", "refactor"},
		{"rename Scored to RankedAgent everywhere", "refactor"},
		{"write unit tests for the ledger", "test"},
		{"improve coverage of rank.go", "test"},
		{"update the readme install section", "docs"},
		{"document the adapter interface", "docs"},
		{"add dark mode to the settings pane", "feature"},
		{"implement quota pruning", "feature"},
		{"hello there", "other"},
	}
	for _, c := range cases {
		if got := Classify(c.task); got != c.want {
			t.Errorf("Classify(%q) = %q, want %q", c.task, got, c.want)
		}
	}
}
