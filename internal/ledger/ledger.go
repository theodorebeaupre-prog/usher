// Package ledger records observed launch and quota events per agent.
// It is a confidence signal, not accounting: vendors don't expose real
// quota, so usher only reacts to what it has actually seen happen.
package ledger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const retention = 7 * 24 * time.Hour

type Event struct {
	Agent string    `json:"agent"`
	Time  time.Time `json:"time"`
	Kind  string    `json:"kind"` // "launch" | "quota"
}

type Ledger struct {
	Events  []Event `json:"events"`
	path    string
	Warning string `json:"-"` // set when the file was corrupt and reset
}

// Load never fails: a missing file is a fresh ledger, a corrupt file is a
// fresh ledger with a Warning. Ledger state must never block a launch.
func Load(path string) *Ledger {
	l := &Ledger{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		return l
	}
	if err := json.Unmarshal(data, l); err != nil {
		return &Ledger{path: path, Warning: fmt.Sprintf("ledger was corrupt, starting fresh (%v)", err)}
	}
	return l
}

func (l *Ledger) record(agent, kind string, now time.Time) {
	kept := l.Events[:0]
	for _, e := range l.Events {
		if now.Sub(e.Time) < retention {
			kept = append(kept, e)
		}
	}
	l.Events = append(kept, Event{Agent: agent, Time: now, Kind: kind})
}

func (l *Ledger) RecordLaunch(agent string, now time.Time) { l.record(agent, "launch", now) }
func (l *Ledger) RecordQuota(agent string, now time.Time)  { l.record(agent, "quota", now) }

// Save writes atomically: temp file in the same directory, then rename.
func (l *Ledger) Save() error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(l)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(l.path), ".ledger-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), l.path)
}

// Confidence returns 0..1: 1.0 means no quota event within the window;
// otherwise it recovers linearly as the window elapses.
func (l *Ledger) Confidence(agent string, window time.Duration, now time.Time) float64 {
	var latest time.Time
	for _, e := range l.Events {
		if e.Agent == agent && e.Kind == "quota" && e.Time.After(latest) {
			latest = e.Time
		}
	}
	if latest.IsZero() {
		return 1.0
	}
	elapsed := now.Sub(latest)
	if elapsed >= window {
		return 1.0
	}
	if elapsed < 0 {
		return 0.0
	}
	return float64(elapsed) / float64(window)
}
