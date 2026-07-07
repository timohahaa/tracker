package service

import (
	"context"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Acme":            "acme",
		"Acme Inc":        "acme-inc",
		"  Acme  Inc  ":   "acme-inc",
		"Acme, Inc.":      "acme-inc",
		"ACME--INC":       "acme-inc",
		"Team #1 (2026)!": "team-1-2026",
		"---":             "",
		"":                "",
		"Привет":          "privet", // кириллица транслитерируется
		"Café":            "cafe",   // диакритика транслитерируется
		"a1b2":            "a1b2",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUniqueSlug_FirstFree(t *testing.T) {
	got, err := uniqueSlug(context.Background(), "Acme Inc", func(context.Context, string) (bool, error) {
		return false, nil
	})
	if err != nil {
		t.Fatalf("uniqueSlug: %v", err)
	}
	if got != "acme-inc" {
		t.Fatalf("got %q, want acme-inc", got)
	}
}

func TestUniqueSlug_EmptyBaseFallback(t *testing.T) {
	got, err := uniqueSlug(context.Background(), "!!!", func(context.Context, string) (bool, error) {
		return false, nil
	})
	if err != nil {
		t.Fatalf("uniqueSlug: %v", err)
	}
	if got != "workspace" {
		t.Fatalf("got %q, want workspace", got)
	}
}

func TestUniqueSlug_NumericSuffixOnCollision(t *testing.T) {
	taken := map[string]bool{"acme": true, "acme-2": true, "acme-3": true}
	got, err := uniqueSlug(context.Background(), "Acme", func(_ context.Context, s string) (bool, error) {
		return taken[s], nil
	})
	if err != nil {
		t.Fatalf("uniqueSlug: %v", err)
	}
	if got != "acme-4" {
		t.Fatalf("got %q, want acme-4", got)
	}
}

func TestDefaultUsername(t *testing.T) {
	if got := defaultUsername("John Doe", "j@x.io"); got != "john-doe" {
		t.Errorf("from display name: got %q, want john-doe", got)
	}
	if got := defaultUsername("", "jane.roe@x.io"); got != "jane-roe" {
		t.Errorf("from email local part: got %q, want jane-roe", got)
	}
	if got := defaultUsername("", "@x.io"); got != "user" {
		t.Errorf("fallback: got %q, want user", got)
	}
	if got := defaultUsername("Привет", ""); got != "privet" {
		t.Errorf("transliteration: got %q, want privet", got)
	}
	if got := defaultUsername("😀", "😀@x.io"); got != "user" {
		t.Errorf("non-transliterable fallback: got %q, want user", got)
	}
	if got := defaultUsername("Ops Team", ""); !strings.HasPrefix(got, "ops-team") {
		t.Errorf("got %q, want ops-team prefix", got)
	}
}
