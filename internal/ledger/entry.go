// Package ledger provides the entry schema, validation, and serialization for the timbers development ledger.
package ledger

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SchemaVersion is the current schema version for timbers entries.
const SchemaVersion = "timbers.devlog/v1"

// KindEntry is the kind identifier for ledger entries.
const KindEntry = "entry"

// idPrefix is the prefix for all entry IDs.
const idPrefix = "tb_"

// shortSHALength is the number of characters to use from the anchor SHA.
const shortSHALength = 6

// Entry represents a development ledger entry.
type Entry struct {
	Schema    string     `json:"schema"`
	Kind      string     `json:"kind"`
	ID        string     `json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Workset   Workset    `json:"workset"`
	Summary   Summary    `json:"summary"`
	Tags      []string   `json:"tags,omitempty"`
	WorkItems []WorkItem `json:"work_items,omitempty"`
}

// Workset represents the set of commits documented by an entry.
type Workset struct {
	AnchorCommit string    `json:"anchor_commit"`
	Commits      []string  `json:"commits"`
	Range        string    `json:"range,omitempty"`
	Diffstat     *Diffstat `json:"diffstat,omitempty"`
}

// Summary represents the what/why/how summary of an entry.
type Summary struct {
	What string `json:"what"`
	Why  string `json:"why"`
	How  string `json:"how"`
}

// WorkItem represents a link to an external work tracking system.
type WorkItem struct {
	System string `json:"system"`
	ID     string `json:"id"`
}

// Diffstat represents file change statistics.
type Diffstat struct {
	Files      int `json:"files"`
	Insertions int `json:"insertions"`
	Deletions  int `json:"deletions"`
}

// ValidationError is returned when entry validation fails.
type ValidationError struct {
	Fields  []string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Message, strings.Join(e.Fields, ", "))
}

// AsValidationError checks if err is a ValidationError and extracts it.
func AsValidationError(err error, target **ValidationError) bool {
	return errors.As(err, target)
}

// GenerateID creates a deterministic entry ID from an anchor commit and timestamp.
// Format: tb_<ISO8601-timestamp>_<short-sha>
// The timestamp is formatted in UTC with second precision.
// The short SHA is the first 6 characters of the anchor commit.
func GenerateID(anchor string, timestamp time.Time) string {
	// Format timestamp as ISO8601 in UTC
	formattedTime := timestamp.UTC().Format(time.RFC3339)

	// Extract short SHA (first 6 chars or full if shorter)
	shortSHA := anchor
	if len(anchor) > shortSHALength {
		shortSHA = anchor[:shortSHALength]
	}

	return idPrefix + formattedTime + "_" + shortSHA
}

// Validate checks that all required fields are present.
// Returns a ValidationError with the list of missing fields if validation fails.
func (e *Entry) Validate() error {
	var missing []string
	missing = e.validateTopLevel(missing)
	missing = e.Workset.validate(missing)
	missing = e.Summary.validate(missing)

	if len(missing) > 0 {
		return &ValidationError{
			Fields:  missing,
			Message: "missing required fields",
		}
	}

	return nil
}

// validateTopLevel checks top-level required fields.
func (e *Entry) validateTopLevel(missing []string) []string {
	if e.Schema == "" {
		missing = append(missing, "schema")
	}
	if e.Kind == "" {
		missing = append(missing, "kind")
	}
	if e.ID == "" {
		missing = append(missing, "id")
	}
	if e.CreatedAt.IsZero() {
		missing = append(missing, "created_at")
	}
	if e.UpdatedAt.IsZero() {
		missing = append(missing, "updated_at")
	}
	return missing
}

// validate checks required fields in Workset.
func (w *Workset) validate(missing []string) []string {
	if w.AnchorCommit == "" {
		missing = append(missing, "workset.anchor_commit")
	}
	if len(w.Commits) == 0 {
		missing = append(missing, "workset.commits")
	}
	return missing
}

// validate checks required fields in Summary.
func (s *Summary) validate(missing []string) []string {
	if s.What == "" {
		missing = append(missing, "summary.what")
	}
	if s.Why == "" {
		missing = append(missing, "summary.why")
	}
	if s.How == "" {
		missing = append(missing, "summary.how")
	}
	return missing
}

// ToJSON serializes the entry to JSON.
func (e *Entry) ToJSON() ([]byte, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("serializing entry to JSON: %w", err)
	}
	return data, nil
}

// FromJSON deserializes an entry from JSON.
func FromJSON(data []byte) (*Entry, error) {
	if len(data) == 0 {
		return nil, errors.New("empty JSON data")
	}

	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("parsing entry JSON: %w", err)
	}

	return &entry, nil
}
