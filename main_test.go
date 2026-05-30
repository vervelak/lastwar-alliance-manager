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

func TestAbsInt64(t *testing.T) {
	tests := []struct{ in, want int64 }{
		{5, 5}, {-5, 5}, {0, 0}, {-1000000, 1000000},
	}
	for _, tt := range tests {
		if got := absInt64(tt.in); got != tt.want {
			t.Errorf("absInt64(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestCleanVSRowName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Leading single-letter artefact stripped.
		{"I Gargoland", "Gargoland"},
		{"l MTee689", "MTee689"},
		// Normal names unchanged.
		{"Gargoland", "Gargoland"},
		{"Mando 78", "Mando 78"},
		// Alliance tags removed (handled by cleanPlayerName inside cleanVSRowName).
		{"[RSRP]Gargoland", "Gargoland"},
		// Trailing punctuation stripped.
		{"WoodWould.", "WoodWould"},
	}
	for _, tt := range tests {
		got := cleanVSRowName(tt.input)
		if got != tt.want {
			t.Errorf("cleanVSRowName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAdjustVSPointsByOrder_NoOp_WhenNoCandidates(t *testing.T) {
	pts, raw := adjustVSPointsByOrder(5000000, "5,000,000", nil, 6000000)
	if pts != 5000000 || raw != "5,000,000" {
		t.Errorf("expected unchanged; got pts=%d raw=%q", pts, raw)
	}
}

func TestAdjustVSPointsByOrder_RejectsTooHigh(t *testing.T) {
	// Previous = 6 M; selected = 7 M (invalid increase); candidate 5 M is valid.
	candidates := []vsPointsCandidate{
		{Value: 7000000, Raw: "7000000", Votes: 2, Score: 13},
		{Value: 5000000, Raw: "5000000", Votes: 1, Score: 7},
	}
	pts, _ := adjustVSPointsByOrder(7000000, "7000000", candidates, 6000000)
	if pts != 5000000 {
		t.Errorf("expected 5000000 (fits descending order), got %d", pts)
	}
}

func TestVsCandidateValues_Deduplication(t *testing.T) {
	candidates := []vsPointsCandidate{
		{Value: 100, Votes: 2},
		{Value: 200, Votes: 1},
		{Value: 100, Votes: 1}, // duplicate
	}
	got := vsCandidateValues(candidates)
	if len(got) != 2 {
		t.Errorf("expected 2 unique values, got %d: %v", len(got), got)
	}
	// Should be sorted descending.
	if got[0] != 200 || got[1] != 100 {
		t.Errorf("expected [200 100], got %v", got)
	}
}

func TestReconcileVSPointCandidates_GlobalAdjustment(t *testing.T) {
	// Row 0: 10 M, row 1: 9 M, row 2: erroneously 12 M but has 8 M candidate.
	rows := []vsRowOCRResult{
		{MemberName: "Alpha", Points: 10000000},
		{MemberName: "Beta", Points: 9000000},
		{
			MemberName: "Gamma",
			Points:     12000000,
			Candidates: []vsPointsCandidate{
				{Value: 12000000, Raw: "12000000", Votes: 1, Score: 7},
				{Value: 8000000, Raw: "8000000", Votes: 2, Score: 13},
			},
		},
	}
	result := reconcileVSPointCandidates(rows)
	// Gamma's 12M is higher than previous (9M) so the 8M candidate should win.
	if result[2].Points != 8000000 {
		t.Errorf("expected Gamma adjusted to 8000000, got %d", result[2].Points)
	}
}

func TestFinalizeVSOCRRecords_ConfidenceHigh(t *testing.T) {
	rows := []vsRowOCRResult{
		{MemberName: "Alpha", Points: 10000000, Notes: nil},
		{MemberName: "Beta", Points: 9000000, Notes: []string{"adjusted"}},
	}
	records := finalizeVSOCRRecords(rows)
	if records[0].Confidence != "high" {
		t.Errorf("Alpha confidence = %q, want high", records[0].Confidence)
	}
	if records[1].Confidence != "review" {
		t.Errorf("Beta confidence = %q, want review", records[1].Confidence)
	}
}

func TestMergeVSRecordsByName_NoSupplemental(t *testing.T) {
	primary := []VSOCRRecord{{MemberName: "Alpha", Points: 10000000}}
	result := mergeVSRecordsByName(primary, nil)
	if len(result) != 1 {
		t.Errorf("expected 1, got %d", len(result))
	}
}

func TestMergeVSRecordsByName_AddsNew(t *testing.T) {
	primary := []VSOCRRecord{{MemberName: "Alpha", Points: 10000000}}
	supplemental := []VSOCRRecord{
		{MemberName: "Beta", Points: 8000000},
		{MemberName: "Alpha", Points: 10000000}, // duplicate — should be skipped
	}
	result := mergeVSRecordsByName(primary, supplemental)
	if len(result) != 2 {
		t.Errorf("expected 2, got %d", len(result))
	}
	if result[1].MemberName != "Beta" {
		t.Errorf("expected Beta, got %q", result[1].MemberName)
	}
	if result[1].Confidence != "review" {
		t.Errorf("supplemental confidence = %q, want review", result[1].Confidence)
	}
}
