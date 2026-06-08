package main

import "testing"

func TestBuildCQL(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		space string
		want  string
	}{
		{"text only", "release notes", "", `text ~ "release notes"`},
		{"space only", "", "ENG", `space = "ENG"`},
		{"both", "design", "ENG", `space = "ENG" AND text ~ "design"`},
		{"empty", "", "", ""},
		{"quote escaping", `say "hi"`, "", `text ~ "say \"hi\""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildCQL(tt.text, tt.space); got != tt.want {
				t.Errorf("buildCQL(%q,%q) = %q, want %q", tt.text, tt.space, got, tt.want)
			}
		})
	}
}

func TestSplitExpand(t *testing.T) {
	got := splitExpand(" space, version ,, body.storage ")
	want := []string{"space", "version", "body.storage"}
	if len(got) != len(want) {
		t.Fatalf("splitExpand len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("splitExpand[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
