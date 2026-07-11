package router

import (
	"sort"
	"strings"

	"github.com/theodorebeaupre-prog/usher/internal/config"
)

// AgentInfo is what the router needs to know about one installed agent.
type AgentInfo struct {
	Name      string
	Strengths map[string]float64 // task type -> 0..1
	Quota     float64            // ledger confidence, 0..1
}

// Scored is one row of a ranking, with the components exposed for --why.
type Scored struct {
	Name     string
	TaskType string
	Strength float64
	Quota    float64
	Score    float64
	Pinned   bool
}

const defaultStrength = 0.75
const defaultAgentBonus = 0.01

// Rank orders agents for a task, best first. Pinned agents sort above
// everything; ties break on name for determinism.
func Rank(task, dir string, agents []AgentInfo, cfg config.Config) []Scored {
	taskType := Classify(task)
	pinnedAgent := cfg.Pins.Types[taskType]
	bestLen := -1
	for prefix, agent := range cfg.Pins.Paths {
		if prefix == "" {
			continue
		}
		p := strings.TrimSuffix(prefix, "/")
		if (dir == p || strings.HasPrefix(dir, p+"/")) && len(p) > bestLen {
			bestLen, pinnedAgent = len(p), agent
		}
	}

	var out []Scored
	for _, a := range agents {
		if cfg.IsDisabled(a.Name) {
			continue
		}
		strength, ok := a.Strengths[taskType]
		if !ok {
			strength = defaultStrength
		}
		if w, ok := cfg.Weights[a.Name][taskType]; ok {
			strength = w
		}
		score := strength * a.Quota
		if a.Name == cfg.DefaultAgent {
			score += defaultAgentBonus
		}
		out = append(out, Scored{
			Name: a.Name, TaskType: taskType,
			Strength: strength, Quota: a.Quota, Score: score,
			Pinned: a.Name == pinnedAgent,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pinned != out[j].Pinned {
			return out[i].Pinned
		}
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Name < out[j].Name
	})
	return out
}
