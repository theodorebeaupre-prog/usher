package adapters

import "testing"

func TestRegistryOrderAndLookup(t *testing.T) {
	all := All()
	want := []string{"claude"}
	if len(all) != len(want) {
		t.Fatalf("want %d adapters, got %d", len(want), len(all))
	}
	for i, a := range all {
		if a.Name() != want[i] {
			t.Errorf("All()[%d] = %s, want %s", i, a.Name(), want[i])
		}
	}
	if _, ok := Get("claude"); !ok {
		t.Error("Get(claude) not found")
	}
	if _, ok := Get("nope"); ok {
		t.Error("Get(nope) should not be found")
	}
}

func TestDetectUsesLookPath(t *testing.T) {
	orig, origV := lookPath, versionCmd
	defer func() { lookPath, versionCmd = orig, origV }()

	lookPath = func(bin string) (string, error) { return "/fake/" + bin, nil }
	versionCmd = func(bin string) string { return "9.9.9 (fake)" }
	a, _ := Get("claude")
	installed, version := a.Detect()
	if !installed || version != "9.9.9 (fake)" {
		t.Fatalf("Detect() = %v %q", installed, version)
	}
}

func TestClaudeQuotaError(t *testing.T) {
	a, _ := Get("claude")
	cases := []struct {
		exit   int
		output string
		want   bool
	}{
		{1, "Claude AI usage limit reached|1752241200", true},
		{1, "API Error: 429 rate_limit_error", true},
		{0, "5-hour limit reached", true},
		{1, "panic: nil pointer dereference", false},
		{0, "", false},
	}
	for _, c := range cases {
		if got := a.QuotaError(c.exit, c.output); got != c.want {
			t.Errorf("QuotaError(%d, %q) = %v, want %v", c.exit, c.output, got, c.want)
		}
	}
}

func TestClaudeLaunchArgs(t *testing.T) {
	a, _ := Get("claude")
	if got := a.LaunchArgs("fix it"); len(got) != 1 || got[0] != "fix it" {
		t.Errorf("LaunchArgs = %v", got)
	}
	if got := a.LaunchArgs(""); len(got) != 0 {
		t.Errorf("empty prompt should mean no args, got %v", got)
	}
}
