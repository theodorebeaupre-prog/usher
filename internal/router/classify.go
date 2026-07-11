// Package router ranks available agents for a task. Pure functions only —
// no I/O, no clock, no environment reads.
package router

import (
	"regexp"
	"strings"
)

// Task types, in priority order: the first category whose keywords match wins.
// "fix the flaky test" is debugging, not testing, so debug outranks test.
var taskTypes = []struct {
	name string
	re   *regexp.Regexp
}{
	{"debug", regexp.MustCompile(`\b(fix|bug|bugs|broken|crash|crashes|error|errors|fail|fails|failing|flaky|debug|regression)\b`)},
	{"review", regexp.MustCompile(`\b(review|audit)\b`)},
	{"refactor", regexp.MustCompile(`\b(refactor|rename|extract|restructure|simplify|cleanup|clean)\b`)},
	{"test", regexp.MustCompile(`\b(test|tests|testing|coverage|spec|specs)\b`)},
	{"docs", regexp.MustCompile(`\b(doc|docs|document|documentation|readme|comment|comments|changelog)\b`)},
	{"feature", regexp.MustCompile(`\b(add|implement|create|build|support|feature|new)\b`)},
}

// Classify buckets a task description into a task type.
func Classify(task string) string {
	lower := strings.ToLower(task)
	for _, t := range taskTypes {
		if t.re.MatchString(lower) {
			return t.name
		}
	}
	return "other"
}
