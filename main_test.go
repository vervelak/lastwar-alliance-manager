package main

import (
	"testing"
)

func TestNormalizeName_GermanDiacritics(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Müller", "muller"},
		{"Größe", "grosse"},
		{"Ärger", "arger"},
		{"Übung", "ubung"},
		{"straße", "strasse"},
		// Already-ASCII names should be unchanged (after lowering).
		{"Gargoland", "gargoland"},
		{"MTee689", "mtee689"},
	}
	for _, tt := range tests {
		got := normalizeName(tt.input)
		if got != tt.want {
			t.Errorf("normalizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCalculateSimilarity_ShortContainmentGuard(t *testing.T) {
	tests := []struct {
		a, b      string
		wantBelow int // result must be strictly less than this
		wantAbove int // result must be >= this (use 0 to skip)
	}{
		// Short substring "kim" inside a long name should NOT get the 90 shortcut.
		{"kim", "kimberlyXyz", 90, 0},
		// "alpha" is exactly 5 chars — should still qualify for the 90 shortcut
		// because after normalization both are >= 5 and one contains the other.
		{"alpha", "alphaBetaGamma", 0, 90},
		// "alphabet" inside "alphabetagamma" — both >= 5.
		{"alphabet", "alphabetagamma", 0, 90},
		// Identical strings → 100.
		{"Gargoland", "gargoland", 0, 100},
	}
	for _, tt := range tests {
		got := calculateSimilarity(tt.a, tt.b)
		if tt.wantBelow > 0 && got >= tt.wantBelow {
			t.Errorf("calculateSimilarity(%q, %q) = %d, want < %d", tt.a, tt.b, got, tt.wantBelow)
		}
		if tt.wantAbove > 0 && got < tt.wantAbove {
			t.Errorf("calculateSimilarity(%q, %q) = %d, want >= %d", tt.a, tt.b, got, tt.wantAbove)
		}
	}
}

func TestVSDayPatterns_CoversGermanAndEnglish(t *testing.T) {
	patterns := vsDayPatterns()

	// Must have exactly 6 days (Monday-Saturday).
	if len(patterns) != 6 {
		t.Fatalf("vsDayPatterns() returned %d days, want 6", len(patterns))
	}

	expectedDays := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	for i, want := range expectedDays {
		if patterns[i].Name != want {
			t.Errorf("vsDayPatterns()[%d].Name = %q, want %q", i, patterns[i].Name, want)
		}
	}

	// Every day must have at least one German pattern.
	germanTokens := map[string]string{
		"monday": "mo", "tuesday": "di", "wednesday": "mi",
		"thursday": "do", "friday": "fr", "saturday": "sa",
	}
	for _, dp := range patterns {
		token := germanTokens[dp.Name]
		found := false
		for _, p := range dp.Patterns {
			if p == token {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("vsDayPatterns() day %q missing German token %q", dp.Name, token)
		}
	}

	// No duplicate patterns across days.
	seen := map[string]string{}
	for _, dp := range patterns {
		for _, p := range dp.Patterns {
			if prev, ok := seen[p]; ok {
				t.Errorf("duplicate pattern %q in days %q and %q", p, prev, dp.Name)
			}
			seen[p] = dp.Name
		}
	}
}

func TestIsVSUILabel(t *testing.T) {
	positives := []string{
		"Rang", "Kommandant", "Punkte",
		"Daily Rank", "Tagesrang", "Tages-Rang",
		"Mo", "di.", "Fr",
		"commander", "RANKING", "Points",
		"Your Alliance", "Deine Allianz",
		"Wochenrang", "Nova Sapphire", "Reset Reapers",
	}
	for _, text := range positives {
		if !isVSUILabel(text) {
			t.Errorf("isVSUILabel(%q) = false, want true", text)
		}
	}

	negatives := []string{
		"Gargoland", "MTee689", "", "Precious Reaper",
		"WoodWould", "Ash7860", "Müller42",
	}
	for _, text := range negatives {
		if isVSUILabel(text) {
			t.Errorf("isVSUILabel(%q) = true, want false", text)
		}
	}
}

func TestParseVSPointsText_FiltersGermanDayRows(t *testing.T) {
	// Input text simulating OCR output with German day abbreviations
	// that should be filtered, and one real player row.
	input := "Mo\nDi\nMi\nGargoland 12,345,678\n"
	records := parseVSPointsText(input)

	// Only the player row should survive.
	if len(records) != 1 {
		t.Fatalf("parseVSPointsText returned %d records, want 1; records: %v", len(records), records)
	}
	if records[0].MemberName != "Gargoland" {
		t.Errorf("record name = %q, want %q", records[0].MemberName, "Gargoland")
	}
	if records[0].Points != 12345678 {
		t.Errorf("record points = %d, want %d", records[0].Points, 12345678)
	}
}

func TestParseVSPointsText_GermanDotSeparator(t *testing.T) {
	// German-locale numbers use dots as thousand separators (19.291.992).
	input := "Gargoland 19.291.992\n"
	records := parseVSPointsText(input)

	if len(records) != 1 {
		t.Fatalf("parseVSPointsText returned %d records, want 1; records: %v", len(records), records)
	}
	if records[0].Points != 19291992 {
		t.Errorf("record points = %d, want %d", records[0].Points, 19291992)
	}
}
