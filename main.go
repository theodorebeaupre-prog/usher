package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/theodorebeaupre-prog/usher/internal/adapters"
	"github.com/theodorebeaupre-prog/usher/internal/banner"
	"github.com/theodorebeaupre-prog/usher/internal/config"
	"github.com/theodorebeaupre-prog/usher/internal/launch"
	"github.com/theodorebeaupre-prog/usher/internal/ledger"
	"github.com/theodorebeaupre-prog/usher/internal/router"
)

var version = "dev"

const usage = `usage:
  usher [flags] "<task>"     route the task to the best agent and launch it
  usher doctor               show detected agents, quota confidence, paths
  usher list                 list supported agents
  usher version               print version

flags:
  --agent <name>   skip routing, launch this agent
  --why            print the scoring table before launching
  --no-banner      skip the animated banner

config: ` + "~" + `/.config/usher/config.toml   (pins, weights, default_agent, disabled)`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "usher:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("usher", flag.ContinueOnError)
	agentFlag := fs.String("agent", "", "launch this agent, skip routing")
	whyFlag := fs.Bool("why", false, "print the scoring table")
	noBanner := fs.Bool("no-banner", false, "skip the animated banner")
	fs.Usage = func() { fmt.Fprintln(os.Stderr, usage) }
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()

	if len(rest) > 0 {
		switch rest[0] {
		case "version", "--version":
			fmt.Println("usher", version)
			return nil
		case "list":
			return cmdList()
		case "doctor":
			return cmdDoctor()
		}
	}
	if len(rest) == 0 && *agentFlag == "" {
		banner.Play(os.Stdout, banner.ShouldAnimate(*noBanner))
		fmt.Println()
		fmt.Println(usage)
		return nil
	}

	task := strings.Join(rest, " ")
	return cmdLaunch(task, *agentFlag, *whyFlag)
}

// installedAgents detects all adapters and returns the installed ones with
// their ledger-derived quota confidence.
func installedAgents(cfg config.Config, led *ledger.Ledger, now time.Time) ([]router.AgentInfo, []adapters.Adapter) {
	var infos []router.AgentInfo
	var installed []adapters.Adapter
	for _, a := range adapters.All() {
		ok, _ := a.Detect()
		if !ok {
			continue
		}
		installed = append(installed, a)
		p := a.Profile()
		infos = append(infos, router.AgentInfo{
			Name:      a.Name(),
			Strengths: p.Strengths,
			Quota:     led.Confidence(a.Name(), p.QuotaWindow, now),
		})
	}
	return infos, installed
}

func cmdLaunch(task, forced string, why bool) error {
	now := time.Now()
	cfg, err := config.Load(filepath.Join(config.Dir(), "config.toml"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "usher: config error, using defaults:", err)
		cfg = config.Config{}
	}
	led := ledger.Load(filepath.Join(config.Dir(), "ledger.json"))
	if led.Warning != "" {
		fmt.Fprintln(os.Stderr, "usher:", led.Warning)
	}

	dir, _ := os.Getwd()
	infos, _ := installedAgents(cfg, led, now)
	if len(infos) == 0 {
		return noAgentsError()
	}
	ranked := router.Rank(task, dir, infos, cfg)
	if len(ranked) == 0 {
		return fmt.Errorf("all installed agents are disabled in config")
	}

	choice := ranked[0]
	if forced != "" {
		found := false
		for _, r := range ranked {
			if r.Name == forced {
				choice, found = r, true
				break
			}
		}
		if !found {
			return fmt.Errorf("--agent %s: not installed or disabled (try: usher doctor)", forced)
		}
	}

	if why {
		printWhy(ranked, choice.Name)
	}
	printDecision(choice, forced != "")

	led.RecordLaunch(choice.Name, now)
	if err := led.Save(); err != nil {
		fmt.Fprintln(os.Stderr, "usher: could not save ledger:", err)
	}

	return launchWithFallback(choice, ranked, task, dir, led)
}

// launchWithFallback runs the chosen agent; on a quota error it records the
// event and offers the runner-up.
func launchWithFallback(choice router.Scored, ranked []router.Scored, task, dir string, led *ledger.Ledger) error {
	a, _ := adapters.Get(choice.Name)
	exit, tail, err := launch.Run(a.Bin(), a.LaunchArgs(task), dir)
	if err != nil {
		// The CLI vanished between detect and exec — offer the runner-up
		// instead of dying (spec: error handling, adapter launch failure).
		if next, ok := runnerUp(ranked, choice.Name); ok {
			fmt.Fprintf(os.Stderr, "usher: %s failed to start (%v) — trying %s\n", choice.Name, err, next.Name)
			return launchWithFallback(next, remove(ranked, choice.Name), task, dir, led)
		}
		return err
	}
	if a.QuotaError(exit, tail) {
		led.RecordQuota(choice.Name, time.Now())
		if err := led.Save(); err != nil {
			fmt.Fprintln(os.Stderr, "usher: could not save ledger:", err)
		}
		if next, ok := runnerUp(ranked, choice.Name); ok && isTTY() {
			fmt.Fprintf(os.Stderr, "\n\x1b[93m→ %s hit its usage cap — relaunch with %s? [Y/n] \x1b[0m", choice.Name, next.Name)
			r := bufio.NewReader(os.Stdin)
			line, _ := r.ReadString('\n')
			ans := strings.ToLower(strings.TrimSpace(line))
			if ans == "" || ans == "y" || ans == "yes" {
				return launchWithFallback(next, remove(ranked, choice.Name), task, dir, led)
			}
		}
	}
	os.Exit(exit)
	return nil
}

func runnerUp(ranked []router.Scored, exclude string) (router.Scored, bool) {
	for _, r := range ranked {
		if r.Name != exclude {
			return r, true
		}
	}
	return router.Scored{}, false
}

func remove(ranked []router.Scored, name string) []router.Scored {
	var out []router.Scored
	for _, r := range ranked {
		if r.Name != name {
			out = append(out, r)
		}
	}
	return out
}

func isTTY() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func printDecision(c router.Scored, forced bool) {
	reason := c.TaskType + " task"
	switch {
	case forced:
		reason = "your call"
	case c.Pinned:
		reason = "pinned"
	case c.Quota < 1.0:
		reason += " · routing around a cap"
	default:
		reason += " · quota OK"
	}
	fmt.Fprintf(os.Stderr, "\x1b[96m→ %s\x1b[0m  \x1b[90m(%s · override with --agent)\x1b[0m\n", c.Name, reason)
}

func printWhy(ranked []router.Scored, winner string) {
	fmt.Fprintf(os.Stderr, "task type: \x1b[97m%s\x1b[0m\n\n", ranked[0].TaskType)
	fmt.Fprintf(os.Stderr, "  %-10s %9s %7s %6s %7s\n", "agent", "strength", "quota", "pin", "score")
	for _, r := range ranked {
		pin := "—"
		if r.Pinned {
			pin = "pin"
		}
		marker := ""
		if r.Name == winner {
			marker = "  ← launching"
		}
		fmt.Fprintf(os.Stderr, "  %-10s %9.2f %7.2f %6s %7.2f%s\n",
			r.Name, r.Strength, r.Quota, pin, r.Score, marker)
	}
	fmt.Fprintln(os.Stderr)
}

func noAgentsError() error {
	var sb strings.Builder
	sb.WriteString("no agent CLIs found. usher routes across the ones you install:\n")
	for _, a := range adapters.All() {
		sb.WriteString(fmt.Sprintf("  %-10s %s\n", a.Name(), a.Profile().InstallHint))
	}
	return fmt.Errorf("%s", sb.String())
}

func cmdList() error {
	for _, a := range adapters.All() {
		fmt.Println(a.Name())
	}
	return nil
}

func cmdDoctor() error {
	now := time.Now()
	led := ledger.Load(filepath.Join(config.Dir(), "ledger.json"))
	fmt.Println("usher doctor")
	for _, a := range adapters.All() {
		ok, ver := a.Detect()
		if !ok {
			fmt.Printf("  \x1b[90m%-10s not installed → %s\x1b[0m\n", a.Name(), a.Profile().InstallHint)
			continue
		}
		conf := led.Confidence(a.Name(), a.Profile().QuotaWindow, now)
		fmt.Printf("  \x1b[97m%-10s\x1b[0m %-14s quota %s %3.0f%%\n", a.Name(), ver, bar(conf), conf*100)
	}
	cfgPath := filepath.Join(config.Dir(), "config.toml")
	if _, err := os.Stat(cfgPath); err != nil {
		fmt.Printf("  config: %s \x1b[90m(not found — using defaults)\x1b[0m\n", cfgPath)
	} else {
		fmt.Printf("  config: %s\n", cfgPath)
	}
	fmt.Printf("  ledger: %s \x1b[90m(%d events, confidence not accounting)\x1b[0m\n",
		filepath.Join(config.Dir(), "ledger.json"), len(led.Events))
	return nil
}

func bar(v float64) string {
	full := int(v * 10)
	return "\x1b[92m" + strings.Repeat("█", full) + "\x1b[90m" + strings.Repeat("░", 10-full) + "\x1b[0m"
}
