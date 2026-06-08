package main

import (
	"strings"
	"testing"
)

func TestBuildSkillContainsReference(t *testing.T) {
	for _, flavor := range []string{"", "generic", "claude", "codex", "gemini", "opencode"} {
		out := buildSkill(flavor)
		for _, want := range []string{
			"CONFLUENCE_BASE_URL",
			"confluence-cli search",
			"confluence-cli create",
			"storage format",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("flavor %q missing %q", flavor, want)
			}
		}
	}
}

func TestClaudeFlavorHasFrontmatter(t *testing.T) {
	out := buildSkill("claude")
	if !strings.HasPrefix(out, "---\nname: confluence-cli\n") {
		t.Errorf("claude flavor should start with YAML frontmatter, got:\n%.80s", out)
	}
	if !strings.Contains(out, "description:") {
		t.Error("claude frontmatter missing description")
	}
}

func TestNonClaudeFlavorsHaveNoFrontmatter(t *testing.T) {
	for _, flavor := range []string{"", "generic", "codex", "gemini", "opencode"} {
		if strings.HasPrefix(buildSkill(flavor), "---") {
			t.Errorf("flavor %q should not have YAML frontmatter", flavor)
		}
	}
}

func TestFlavorNamesExcludesEmpty(t *testing.T) {
	names := flavorNames()
	want := map[string]bool{"claude": true, "codex": true, "gemini": true, "opencode": true}
	if len(names) != len(want) {
		t.Fatalf("flavorNames = %v", names)
	}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected flavor %q", n)
		}
	}
}
