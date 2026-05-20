package ledger

import (
	"github.com/gorewood/timbers/internal/output"
)

// WriteAck writes a "decision to skip" ack record. Counts as documented
// for pending detection — acked SHAs are dropped from both gate and
// display paths with reason "ack".
func (s *Storage) WriteAck(ack *Ack) error {
	if s.files == nil {
		return output.NewSystemError("storage not configured for writes")
	}
	return s.files.WriteAck(ack)
}

// ListAcks returns every ack record under the storage directory.
func (s *Storage) ListAcks() ([]*Ack, error) {
	if s.files == nil {
		return nil, nil
	}
	return s.files.ListAcks()
}

// AckedSet returns the set of SHAs that have an ack record. Built from
// a fresh ListAcks scan. Returns an empty (non-nil) map on any error so
// pending detection degrades gracefully rather than failing.
func (s *Storage) AckedSet() map[string]bool {
	acked := make(map[string]bool)
	acks, err := s.ListAcks()
	if err != nil {
		return acked
	}
	for _, ack := range acks {
		acked[ack.TargetSHA] = true
	}
	return acked
}
