package rules

import (
	"path"
	"strings"

	"github.com/YousefMohiey/tidy/internal/config"
)

// MatchResult holds the outcome of a successful rule match.
type MatchResult struct {
	Category    string // rule Name (e.g. "Images")
	Destination string // folder name (e.g. "Images")
}

// Engine evaluates files against a set of configuration rules.
// Match priority: extensions (fast) → magic bytes (MIME) → patterns (glob).
type Engine struct {
	config *config.Config
}

// NewEngine creates a rule-matching engine from the given configuration.
func NewEngine(cfg *config.Config) *Engine {
	return &Engine{config: cfg}
}

// Match evaluates a filename and its detected MIME type against all rules.
// Returns nil if no rule matches (caller decides the "Other" fallback).
//
// Priority order:
//  1. Extensions — case-insensitive, leading dots stripped from both sides.
//  2. MagicBytes — exact string match against the mimeType parameter.
//  3. Patterns   — path.Match glob matching on the filename.
func (e *Engine) Match(filename string, mimeType string) *MatchResult {
	// Phase 1: extension matching (fastest)
	fileExt := strings.TrimPrefix(strings.ToLower(extFromName(filename)), ".")
	if fileExt != "" {
		for _, rule := range e.config.Rules {
			for _, ruleExt := range rule.Extensions {
				cleaned := strings.TrimPrefix(strings.ToLower(ruleExt), ".")
				if cleaned == fileExt {
					return &MatchResult{
						Category:    rule.Name,
						Destination: rule.Destination,
					}
				}
			}
		}
	}

	// Phase 2: MIME type / magic bytes matching
	if mimeType != "" {
		for _, rule := range e.config.Rules {
			for _, mime := range rule.MagicBytes {
				if mime == mimeType {
					return &MatchResult{
						Category:    rule.Name,
						Destination: rule.Destination,
					}
				}
			}
		}
	}

	// Phase 3: glob pattern matching on filename
	for _, rule := range e.config.Rules {
		for _, pattern := range rule.Patterns {
			matched, err := path.Match(pattern, filename)
			if err == nil && matched {
				return &MatchResult{
					Category:    rule.Name,
					Destination: rule.Destination,
				}
			}
		}
	}

	return nil
}

// extFromName extracts the file extension (including the dot) from a filename.
// Returns empty string for dotfiles with no further extension (e.g. ".gitignore").
func extFromName(filename string) string {
	// Use path.Ext which handles the last dot correctly.
	return path.Ext(filename)
}
