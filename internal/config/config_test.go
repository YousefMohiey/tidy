package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

func TestLoad_ValidYAML(t *testing.T) {
	yaml := `rules:
  - name: Images
    extensions: [jpg, png]
    destination: Images
    patterns: ["*.jpeg"]
  - name: Docs
    extensions: [pdf]
    magic_bytes: [application/pdf]
    destination: Docs
`
	path := writeTempFile(t, "config.yaml", yaml)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if got := len(cfg.Rules); got != 2 {
		t.Fatalf("expected 2 rules, got %d", got)
	}

	if cfg.Rules[0].Name != "Images" {
		t.Errorf("rule[0].Name = %q, want %q", cfg.Rules[0].Name, "Images")
	}
	if cfg.Rules[0].Destination != "Images" {
		t.Errorf("rule[0].Destination = %q, want %q", cfg.Rules[0].Destination, "Images")
	}
	if len(cfg.Rules[0].Extensions) != 2 {
		t.Errorf("rule[0].Extensions len = %d, want 2", len(cfg.Rules[0].Extensions))
	}
	if len(cfg.Rules[0].Patterns) != 1 || cfg.Rules[0].Patterns[0] != "*.jpeg" {
		t.Errorf("rule[0].Patterns = %v, want [*.jpeg]", cfg.Rules[0].Patterns)
	}

	if len(cfg.Rules[1].MagicBytes) != 1 || cfg.Rules[1].MagicBytes[0] != "application/pdf" {
		t.Errorf("rule[1].MagicBytes = %v, want [application/pdf]", cfg.Rules[1].MagicBytes)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("error = %q, want substring %q", err.Error(), "failed to read file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantSub string
	}{
		{
			name:    "malformed structure",
			content: "rules:\n  - name: [broken",
			wantSub: "failed to parse file",
		},
		{
			name:    "invalid tabs in mapping",
			content: "rules:\n  name: foo\n\textensions: [a]",
			wantSub: "failed to parse file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTempFile(t, "bad.yaml", tc.content)
			_, err := Load(path)
			if err == nil {
				t.Fatal("expected error for invalid YAML")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestLoad_EmptyRules(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{
			name:    "empty file",
			content: "",
		},
		{
			name:    "rules key missing",
			content: "other: value\n",
		},
		{
			name:    "rules array empty",
			content: "rules: []\n",
		},
		{
			name:    "rules key null",
			content: "rules: ~\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTempFile(t, "empty.yaml", tc.content)
			cfg, err := Load(path)
			if err == nil {
				t.Fatalf("expected error for empty rules, got cfg=%+v", cfg)
			}
			if !strings.Contains(err.Error(), "no rules") {
				t.Errorf("error = %q, want substring %q", err.Error(), "no rules")
			}
		})
	}
}

func TestDefault_ReturnsNonEmptyConfig(t *testing.T) {
	cfg := Default()
	if cfg == nil {
		t.Fatal("Default() returned nil")
	}
	if len(cfg.Rules) == 0 {
		t.Fatal("Default() returned config with no rules")
	}
}

func TestDefault_RulesHaveRequiredFields(t *testing.T) {
	cfg := Default()

	for i, r := range cfg.Rules {
		if strings.TrimSpace(r.Name) == "" {
			t.Errorf("rule[%d] has empty Name", i)
		}
		if strings.TrimSpace(r.Destination) == "" {
			t.Errorf("rule[%d] (%s) has empty Destination", i, r.Name)
		}

		hasCriterion := len(r.Extensions) > 0 || len(r.MagicBytes) > 0 || len(r.Patterns) > 0
		if !hasCriterion {
			t.Errorf("rule[%d] (%s) has no Extensions, MagicBytes, or Patterns", i, r.Name)
		}
	}
}

func TestDefault_RulesHaveUniqueNames(t *testing.T) {
	cfg := Default()
	seen := make(map[string]bool, len(cfg.Rules))
	for _, r := range cfg.Rules {
		if seen[r.Name] {
			t.Errorf("duplicate rule name: %q", r.Name)
		}
		seen[r.Name] = true
	}
}

func TestDefault_ImagesRuleIncludesJpg(t *testing.T) {
	cfg := Default()
	var images *Rule
	for i := range cfg.Rules {
		if cfg.Rules[i].Name == "Images" {
			images = &cfg.Rules[i]
			break
		}
	}
	if images == nil {
		t.Fatal("Default() missing Images rule")
	}

	found := false
	for _, ext := range images.Extensions {
		if ext == "jpg" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Images rule Extensions = %v, want to include %q", images.Extensions, "jpg")
	}
	if images.Destination == "" {
		t.Error("Images rule has empty Destination")
	}
}
