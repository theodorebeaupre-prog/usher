package main

import (
	"bufio"
	"errors"
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
  usher -p "<task>"          headless: answer on stdout, exit code from the agent
  usher doctor               show detected agents, quota confidence, paths
  usher list                 list supported agents
  usher version              print version

flags:
  --agent <name>   skip routing, launch this agent
  --why            print the scoring table before launching
  --no-banner      skip the animated banner
  -p, --print      headless: run the agent's print-and-exit mode (for scripts/CI)

config: ` + "~" + `/.config/usher/config.toml   (pins, weights, default_agent, disabled)`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "usher:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 && (args[0] == "version" || args[0] == "--version") {
		fmt.Println("usher", version)
		return nil
	}

	fs := flag.NewFlagSet("usher", flag.ContinueOnError)
	agentFlag := fs.String("agent", "", "launch this agent, skip routing")
	whyFlag := fs.Bool("why", false, "print the scoring table")
	noBanner := fs.Bool("no-banner", false, "skip the animated banner")
	var headless bool
	fs.BoolVar(&headless, "p", false, "headless: print the agent's answer and exit (no chat UI)")
	fs.BoolVar(&headless, "print", false, "alias of -p")
	fs.Usage = func() { fmt.Fprintln(os.Stderr, usage) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// fs.Usage already printed the usage text.
			return nil
		}
		return err
	}
	rest := fs.Args()

	if len(rest) > 0 {
		switch rest[0] {
		case "list":
			return cmdList()
		case "doctor":
			return cmdDoctor()
		}
	}
	if len(rest) == 0 && *agentFlag == "" && !headless {
		banner.Play(os.Stdout, banner.ShouldAnimate(*noBanner))
		fmt.Println()
		fmt.Println(usage)
		return nil
	}

	for _, w := range rest {
		if strings.HasPrefix(w, "-") {
			return fmt.Errorf("flag %q must come before the task — usage: usher [flags] \"<task>\"", w)
		}
	}

	task := strings.Join(rest, " ")
	if headless {
		if task == "" {
			return fmt.Errorf("headless mode needs a task: usher -p \"<task>\"")
		}
		return cmdHeadless(task, *agentFlag, *whyFlag)
	}
	return cmdLaunch(task, *agentFlag, *whyFlag)
}

// installedAgents detects all adapters and returns router inputs for the
// installed ones, with their ledger-derived quota confidence. Disabled
// agents are filtered later by router.Rank.
func installedAgents(led *ledger.Ledger, now time.Time) []router.AgentInfo {
	var infos []router.AgentInfo
	for _, a := range adapters.All() {
		if !a.Installed() {
			continue
		}
		p := a.Profile()
		infos = append(infos, router.AgentInfo{
			Name:      a.Name(),
			Strengths: p.Strengths,
			Quota:     led.Confidence(a.Name(), p.QuotaWindow, now),
		})
	}
	return infos
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
	infos := installedAgents(led, now)
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
	routedAround := ""
	if forced == "" && !choice.Pinned {
		routedAround = strongestCappedLoser(ranked, choice)
	}
	printDecision(choice, forced != "", routedAround)

	return launchWithFallback(choice, ranked, task, dir, led)
}

// cmdHeadless routes and runs the winner's print-and-exit mode. usher's own
// lines go to stderr; stdout belongs entirely to the agent. On a quota error
// it fails over to the next ranked agent automatically — each agent is
// attempted at most once. A forced --agent skips ranking and failover
// entirely.
func cmdHeadless(task, forced string, why bool) error {
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
	var infos []router.AgentInfo
	for _, info := range installedAgents(led, now) {
		a, _ := adapters.Get(info.Name)
		if a.HeadlessArgs(task) == nil {
			continue // no print-and-exit mode; not a candidate for -p
		}
		infos = append(infos, info)
	}
	if len(infos) == 0 {
		return noHeadlessAgentsError()
	}
	ranked := router.Rank(task, dir, infos, cfg)
	if len(ranked) == 0 {
		return fmt.Errorf("all headless-capable agents are disabled in config")
	}

	if forced != "" {
		var pick *router.Scored
		for i := range ranked {
			if ranked[i].Name == forced {
				pick = &ranked[i]
				break
			}
		}
		if pick == nil {
			return fmt.Errorf("--agent %s: not installed, disabled, or no headless mode (try: usher doctor)", forced)
		}
		ranked = []router.Scored{*pick} // single attempt, no failover
	}

	if why {
		printWhy(ranked, ranked[0].Name)
	}

	color := useColor()
	exit := 0
	for i, choice := range ranked {
		a, _ := adapters.Get(choice.Name)
		fmt.Fprintf(os.Stderr, "%s  %s\n",
			sgr(color, 96, "→ "+choice.Name),
			sgr(color, 90, fmt.Sprintf("(%s task · headless)", choice.TaskType)))

		led.RecordLaunch(choice.Name, time.Now())
		if err := led.Save(); err != nil {
			fmt.Fprintln(os.Stderr, "usher: could not save ledger:", err)
		}

		var tail string
		exit, tail, err = launch.Run(a.Bin(), a.HeadlessArgs(task), dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "usher: %s failed to start: %v\n", choice.Name, err)
			exit = 1
			continue
		}
		if exit == 0 || !a.QuotaError(exit, tail) {
			os.Exit(exit)
		}
		led.RecordQuota(choice.Name, time.Now())
		if err := led.Save(); err != nil {
			fmt.Fprintln(os.Stderr, "usher: could not save ledger:", err)
		}
		if i < len(ranked)-1 {
			fmt.Fprintf(os.Stderr, "%s\n", sgr(color, 93,
				fmt.Sprintf("→ %s hit its cap — failing over to %s", choice.Name, ranked[i+1].Name)))
		}
	}
	fmt.Fprintln(os.Stderr, "usher: every available agent is capped or failed — exiting with the last agent's code")
	os.Exit(exit)
	return nil
}

func noHeadlessAgentsError() error {
	var sb strings.Builder
	sb.WriteString("no installed agent supports headless mode. These would:\n")
	for _, a := range adapters.All() {
		if a.HeadlessArgs("x") != nil {
			sb.WriteString(fmt.Sprintf("  %-10s %s\n", a.Name(), a.Profile().InstallHint))
		}
	}
	return fmt.Errorf("%s", sb.String())
}

// launchWithFallback runs the chosen agent; on a quota error it records the
// event and offers the runner-up. Every attempt — including recursive
// relaunches down the quota Y/n path and the start-failure path — records
// its own launch event first.
func launchWithFallback(choice router.Scored, ranked []router.Scored, task, dir string, led *ledger.Ledger) error {
	led.RecordLaunch(choice.Name, time.Now())
	if err := led.Save(); err != nil {
		fmt.Fprintln(os.Stderr, "usher: could not save ledger:", err)
	}
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
			color := useColor()
			prompt := sgr(color, 93, fmt.Sprintf("→ %s hit its usage cap — relaunch with %s? [Y/n] ", choice.Name, next.Name))
			fmt.Fprintf(os.Stderr, "\n%s", prompt)
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

// useColor reports whether ANSI escapes should be emitted, per no-color.org:
// any non-empty NO_COLOR disables color.
func useColor() bool { return os.Getenv("NO_COLOR") == "" }

// sgr wraps s in the given SGR code when color is true; otherwise it returns
// s unchanged.
func sgr(color bool, code int, s string) string {
	if !color {
		return s
	}
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", code, s)
}

// strongestCappedLoser finds the highest-strength ranked agent (other than
// choice) that is stronger than choice but capped (Quota < 1.0) — the agent
// usher routed around by picking choice instead. Returns "" if none.
func strongestCappedLoser(ranked []router.Scored, choice router.Scored) string {
	best := ""
	bestStrength := choice.Strength
	for _, r := range ranked {
		if r.Name == choice.Name {
			continue
		}
		if r.Strength > choice.Strength && r.Quota < 1.0 && r.Strength > bestStrength {
			best = r.Name
			bestStrength = r.Strength
		}
	}
	return best
}

func printDecision(c router.Scored, forced bool, routedAround string) {
	reason := c.TaskType + " task"
	switch {
	case forced:
		reason = "your call"
	case c.Pinned:
		reason = "pinned"
	case routedAround != "":
		reason = fmt.Sprintf("%s task · %s is capped — routed around it", c.TaskType, routedAround)
	case c.Quota < 1.0:
		reason += " · cap cooling down"
	default:
		reason += " · quota OK"
	}
	color := useColor()
	arrow := sgr(color, 96, "→ "+c.Name)
	meta := sgr(color, 90, fmt.Sprintf("(%s · override with --agent)", reason))
	fmt.Fprintf(os.Stderr, "%s  %s\n", arrow, meta)
}

func printWhy(ranked []router.Scored, winner string) {
	color := useColor()
	fmt.Fprintf(os.Stderr, "task type: %s\n\n", sgr(color, 97, ranked[0].TaskType))
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
	color := useColor()
	fmt.Println("usher doctor")
	if led.Warning != "" {
		fmt.Println("  ⚠ " + led.Warning)
	}
	for _, a := range adapters.All() {
		ok, ver := a.Detect()
		if !ok {
			fmt.Printf("  %s\n", sgr(color, 90, fmt.Sprintf("%-10s not installed → %s", a.Name(), a.Profile().InstallHint)))
			continue
		}
		conf := led.Confidence(a.Name(), a.Profile().QuotaWindow, now)
		fmt.Printf("  %s %-14s quota %s %3.0f%%\n", sgr(color, 97, fmt.Sprintf("%-10s", a.Name())), ver, bar(conf, color), conf*100)
	}
	cfgPath := filepath.Join(config.Dir(), "config.toml")
	if _, err := os.Stat(cfgPath); err != nil {
		fmt.Printf("  config: %s %s\n", cfgPath, sgr(color, 90, "(not found — using defaults)"))
	} else {
		fmt.Printf("  config: %s\n", cfgPath)
	}
	fmt.Printf("  ledger: %s %s\n",
		filepath.Join(config.Dir(), "ledger.json"),
		sgr(color, 90, fmt.Sprintf("(%d events, confidence not accounting)", len(led.Events))))
	return nil
}

func bar(v float64, color bool) string {
	full := int(v * 10)
	if !color {
		return strings.Repeat("█", full) + strings.Repeat("░", 10-full)
	}
	return "\x1b[92m" + strings.Repeat("█", full) + "\x1b[90m" + strings.Repeat("░", 10-full) + "\x1b[0m"
}
