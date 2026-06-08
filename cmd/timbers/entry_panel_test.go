package main

import (
	"testing"
	"time"

	"github.com/gorewood/timbers/internal/ledger"
	"github.com/gorewood/timbers/internal/output"
)

func sampleEntry() *ledger.Entry {
	return &ledger.Entry{
		ID:        "tb_2026-06-08T17:42:10Z_a3f9c2",
		CreatedAt: time.Date(2026, 6, 8, 17, 42, 10, 0, time.UTC),
		Summary: ledger.Summary{
			What: "Fixed auth bypass",
			Why:  "input not sanitized",
			How:  "added middleware",
		},
		Workset: ledger.Workset{
			AnchorCommit: "a3f9c2dabcdef",
			Commits:      []string{"a3f9c2dabcdef"},
			Diffstat:     &ledger.Diffstat{Files: 3, Insertions: 45, Deletions: 12},
		},
	}
}

// keysOf returns the non-separator keys in order.
func keysOf(fields []output.Field) []string {
	var keys []string
	for _, f := range fields {
		if f.Key != "" {
			keys = append(keys, f.Key)
		}
	}
	return keys
}

func hasKey(fields []output.Field, key string) bool {
	for _, f := range fields {
		if f.Key == key {
			return true
		}
	}
	return false
}

// TestDryRunFieldsIncludeNotes guards the bug where dry-run silently dropped
// the Notes field.
func TestDryRunFieldsIncludeNotes(t *testing.T) {
	entry := sampleEntry()
	entry.Notes = "considered rate limiting"
	if !hasKey(dryRunFields(entry), "Notes") {
		t.Error("dry-run panel must include Notes when present")
	}

	entry.Notes = ""
	if hasKey(dryRunFields(entry), "Notes") {
		t.Error("dry-run panel must omit Notes when empty")
	}
}

// TestDryRunFieldsOrder verifies substance leads and bookkeeping (ID, Anchor)
// trails after a separator.
func TestDryRunFieldsOrder(t *testing.T) {
	fields := dryRunFields(sampleEntry())
	keys := keysOf(fields)

	if keys[0] != "What" {
		t.Errorf("first field = %q, want What", keys[0])
	}
	if n := len(keys); keys[n-2] != "ID" || keys[n-1] != "Anchor" {
		t.Errorf("bookkeeping not at bottom: ...%v", keys[len(keys)-2:])
	}

	// A separator must sit immediately before the ID row.
	idIdx := -1
	for i, f := range fields {
		if f.Key == "ID" {
			idIdx = i
			break
		}
	}
	if idIdx <= 0 || fields[idIdx-1] != output.Separator() {
		t.Errorf("expected separator before ID row, fields=%v", fields)
	}
}

// TestShowFieldsTitleNotInBody verifies the ID is not duplicated in the body
// (it is the panel title), and substance leads with bookkeeping trailing.
func TestShowFieldsTitleNotInBody(t *testing.T) {
	fields := showFields(sampleEntry())
	if hasKey(fields, "ID") {
		t.Error("show body must not repeat the ID (it is the title)")
	}
	keys := keysOf(fields)
	if keys[0] != "What" {
		t.Errorf("first field = %q, want What", keys[0])
	}
	for _, want := range []string{"Anchor", "Commits", "Files", "Created"} {
		if !hasKey(fields, want) {
			t.Errorf("show panel missing bookkeeping field %q", want)
		}
	}
}

// TestSubstanceFieldsOmitEmptyOptionals verifies optional rows only appear when
// set, and What/Why are emphasized.
func TestSubstanceFieldsOmitEmptyOptionals(t *testing.T) {
	fields := substanceFields(sampleEntry()) // no notes/tags/work
	for _, absent := range []string{"Notes", "Tags", "Work"} {
		if hasKey(fields, absent) {
			t.Errorf("unexpected optional field %q when empty", absent)
		}
	}
	for _, f := range fields {
		if (f.Key == "What" || f.Key == "Why") && !f.Emphasis {
			t.Errorf("field %q should be emphasized", f.Key)
		}
	}
}
