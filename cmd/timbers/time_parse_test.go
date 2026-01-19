package main

import (
	"testing"
	"time"
)

func TestParseSinceValue_Duration(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantDiff  time.Duration // approximate difference from now
		tolerance time.Duration // allowed tolerance
		wantErr   bool
	}{
		{"24 hours", "24h", 24 * time.Hour, time.Second, false},
		{"48 hours", "48h", 48 * time.Hour, time.Second, false},
		{"7 days", "7d", 7 * 24 * time.Hour, time.Second, false},
		{"2 weeks", "2w", 14 * 24 * time.Hour, time.Second, false},
		{"1 month", "1m", 30 * 24 * time.Hour, 48 * time.Hour, false}, // months vary (28-31 days)
		{"invalid unit", "5x", 0, 0, true},
		{"no number", "d", 0, 0, true},
		{"negative", "-5d", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now().UTC()
			got, err := parseSinceValue(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSinceValue(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("parseSinceValue(%q) unexpected error: %v", tt.input, err)
				return
			}

			// Check that the result is approximately the expected duration ago
			diff := now.Sub(got)
			if diff < tt.wantDiff-tt.tolerance || diff > tt.wantDiff+tt.tolerance {
				t.Errorf("parseSinceValue(%q) = %v ago, want ~%v ago (tolerance %v)", tt.input, diff, tt.wantDiff, tt.tolerance)
			}
		})
	}
}

func TestParseSinceValue_Date(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{"ISO date", "2026-01-15", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), false},
		{"ISO date 2", "2025-12-31", time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), false},
		{"invalid date", "2026-13-45", time.Time{}, true},
		{"wrong format", "01-15-2026", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSinceValue(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseSinceValue(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("parseSinceValue(%q) unexpected error: %v", tt.input, err)
				return
			}

			if !got.Equal(tt.want) {
				t.Errorf("parseSinceValue(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseSinceValue_RFC3339(t *testing.T) {
	input := "2026-01-15T10:30:00Z"
	want := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

	got, err := parseSinceValue(input)
	if err != nil {
		t.Fatalf("parseSinceValue(%q) unexpected error: %v", input, err)
	}

	if !got.Equal(want) {
		t.Errorf("parseSinceValue(%q) = %v, want %v", input, got, want)
	}
}
