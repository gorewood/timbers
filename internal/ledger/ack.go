package ledger

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// KindAck is the kind identifier for ack records — a "decision to skip"
// note that documents why a particular commit doesn't merit a content
// entry. Lives alongside entries under the same schema family
// (timbers.devlog/v1) but has a different shape.
const KindAck = "ack"

// ackIDPrefix is the prefix for all ack IDs (parallel to "tb_" for entries).
const ackIDPrefix = "ack_"

// Ack represents a "decision to skip" record. Records that the operator
// looked at a specific commit, deemed it not worth a content entry, and
// captured the reason in one line. Counts as documented for pending
// detection — a third bypass path alongside infrastructure rules and
// revert auto-skip.
//
// Use case: GitHub Action runs on `pull_request: closed && merged == true`
// and writes `ack <merge_sha> --reason "GitHub merge of PR #N"` so the
// merge SHA self-clears from everyone's pending list without requiring
// client-side discipline.
type Ack struct {
	Schema    string    `json:"schema"`
	Kind      string    `json:"kind"`
	ID        string    `json:"id"`
	AckedAt   time.Time `json:"acked_at"`
	Acker     Acker     `json:"acker"`
	TargetSHA string    `json:"target_sha"`
	Reason    string    `json:"reason"`
}

// Acker is the identity of whoever recorded the ack (typically from git
// config user.name + user.email).
type Acker struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GenerateAckID produces a deterministic ack ID from the target SHA and
// timestamp. Format: ack_<short-sha>_<ISO8601-timestamp>. The short SHA
// comes first so all acks for the same commit cluster together when
// listed; the timestamp disambiguates if the same commit is acked
// multiple times (e.g., a later ack with a more specific reason).
func GenerateAckID(targetSHA string, ackedAt time.Time) string {
	short := targetSHA
	if len(targetSHA) > shortSHALength {
		short = targetSHA[:shortSHALength]
	}
	return ackIDPrefix + short + "_" + ackedAt.UTC().Format(time.RFC3339)
}

// Validate checks that all required fields are present.
func (a *Ack) Validate() error {
	var missing []string
	if a.Schema == "" {
		missing = append(missing, "schema")
	}
	if a.Kind == "" {
		missing = append(missing, "kind")
	}
	if a.ID == "" {
		missing = append(missing, "id")
	}
	if a.AckedAt.IsZero() {
		missing = append(missing, "acked_at")
	}
	if a.TargetSHA == "" {
		missing = append(missing, "target_sha")
	}
	if a.Reason == "" {
		missing = append(missing, "reason")
	}
	if len(missing) > 0 {
		return &ValidationError{Fields: missing, Message: "missing required fields"}
	}
	return nil
}

// ToJSON serializes the ack to JSON.
func (a *Ack) ToJSON() ([]byte, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return nil, fmt.Errorf("serializing ack to JSON: %w", err)
	}
	return data, nil
}

// FromJSONAck deserializes an ack record from JSON. Returns ErrNotTimbersNote
// when the JSON is valid but doesn't have the timbers schema, or when the
// kind is not "ack" (use FromJSON for entries).
func FromJSONAck(data []byte) (*Ack, error) {
	if len(data) == 0 {
		return nil, errors.New("empty JSON data")
	}
	var ack Ack
	if err := json.Unmarshal(data, &ack); err != nil {
		return nil, fmt.Errorf("parsing ack JSON: %w", err)
	}
	if !strings.HasPrefix(ack.Schema, "timbers.devlog/") {
		return nil, ErrNotTimbersNote
	}
	if ack.Kind != KindAck {
		return nil, ErrNotTimbersNote
	}
	return &ack, nil
}

// AckDateDir extracts the YYYY/MM/DD relative path from an ack ID.
// Ack IDs have the format ack_<short-sha>_YYYY-MM-DDT... — the date
// comes after the second underscore. Returns empty string if the ID
// doesn't parse.
func AckDateDir(id string) string {
	if !strings.HasPrefix(id, ackIDPrefix) {
		return ""
	}
	// ack_<short>_<timestamp> — the timestamp portion starts after the
	// short SHA. Look for the second underscore.
	rest := id[len(ackIDPrefix):]
	idx := strings.IndexByte(rest, '_')
	if idx < 0 || idx+11 > len(rest) {
		return ""
	}
	datePart := rest[idx+1 : idx+11] // "2026-05-20"
	parts := strings.SplitN(datePart, "-", 3)
	if len(parts) != 3 {
		return ""
	}
	return parts[0] + "/" + parts[1] + "/" + parts[2]
}
