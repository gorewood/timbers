package ledger

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestGenerateAckID(t *testing.T) {
	ackedAt := time.Date(2026, 5, 20, 12, 30, 45, 0, time.UTC)
	tests := []struct {
		name      string
		targetSHA string
		want      string
	}{
		{
			name:      "long SHA truncated to short form",
			targetSHA: "abc123def456789012345678901234567890abcd",
			want:      "ack_abc123_2026-05-20T12:30:45Z",
		},
		{
			name:      "short SHA preserved",
			targetSHA: "abc12",
			want:      "ack_abc12_2026-05-20T12:30:45Z",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateAckID(tt.targetSHA, ackedAt)
			if got != tt.want {
				t.Errorf("GenerateAckID = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAckDateDir(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "canonical ack ID",
			id:   "ack_abc123_2026-05-20T12:30:45Z",
			want: "2026/05/20",
		},
		{
			name: "dashed (filename-safe) ack ID also parses",
			id:   "ack_abc123_2026-05-20T12-30-45Z",
			want: "2026/05/20",
		},
		{
			name: "non-ack ID returns empty",
			id:   "tb_2026-05-20T12:30:45Z_abc123",
			want: "",
		},
		{
			name: "malformed ID returns empty",
			id:   "ack_short",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AckDateDir(tt.id)
			if got != tt.want {
				t.Errorf("AckDateDir(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestAckValidate(t *testing.T) {
	valid := &Ack{
		Schema:    SchemaVersion,
		Kind:      KindAck,
		ID:        "ack_abc123_2026-05-20T12:30:45Z",
		AckedAt:   time.Now().UTC(),
		Acker:     Acker{Name: "Test", Email: "test@example.com"},
		TargetSHA: "abc123def456",
		Reason:    "Reviewed; no entry needed",
	}
	if err := valid.Validate(); err != nil {
		t.Errorf("valid ack should pass: %v", err)
	}

	t.Run("missing reason", func(t *testing.T) {
		bad := *valid
		bad.Reason = ""
		if err := bad.Validate(); err == nil {
			t.Error("expected validation error for missing reason")
		}
	})

	t.Run("missing target_sha", func(t *testing.T) {
		bad := *valid
		bad.TargetSHA = ""
		if err := bad.Validate(); err == nil {
			t.Error("expected validation error for missing target_sha")
		}
	})

	t.Run("zero acked_at", func(t *testing.T) {
		bad := *valid
		bad.AckedAt = time.Time{}
		if err := bad.Validate(); err == nil {
			t.Error("expected validation error for zero acked_at")
		}
	})
}

func TestAckJSONRoundtrip(t *testing.T) {
	original := &Ack{
		Schema:    SchemaVersion,
		Kind:      KindAck,
		ID:        "ack_abc123_2026-05-20T12:30:45Z",
		AckedAt:   time.Date(2026, 5, 20, 12, 30, 45, 0, time.UTC),
		Acker:     Acker{Name: "Test", Email: "test@example.com"},
		TargetSHA: "abc123def456789012345678901234567890abcd",
		Reason:    "GitHub merge of PR #N; content in entry tb_xyz",
	}

	data, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	got, err := FromJSONAck(data)
	if err != nil {
		t.Fatalf("FromJSONAck: %v", err)
	}
	if got.ID != original.ID || got.TargetSHA != original.TargetSHA || got.Reason != original.Reason {
		t.Errorf("roundtrip mismatch: got %+v, want %+v", got, original)
	}
}

func TestFromJSONAck_RejectsEntries(t *testing.T) {
	// An entry document — same schema family, but kind="entry" — must be
	// rejected so FromJSONAck only loads ack records.
	entry := map[string]any{
		"schema":     SchemaVersion,
		"kind":       KindEntry,
		"id":         "tb_2026-05-20T12:30:45Z_abc123",
		"created_at": "2026-05-20T12:30:45Z",
		"updated_at": "2026-05-20T12:30:45Z",
		"workset":    map[string]any{"anchor_commit": "abc", "commits": []string{"abc"}},
		"summary":    map[string]any{"what": "x", "why": "y", "how": "z"},
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	_, err = FromJSONAck(data)
	if err == nil {
		t.Error("FromJSONAck must reject entry documents")
	}
	if !strings.Contains(err.Error(), "not a timbers note") {
		// ErrNotTimbersNote uses this message
		t.Errorf("expected not-a-timbers-note error, got %v", err)
	}
}
