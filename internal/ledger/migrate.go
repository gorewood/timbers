package ledger

// MigrateLegacyFilenames renames pre-v0.18 colon-encoded entry files in the
// underlying FileStorage to the canonical (dashed) form. Returns the IDs that
// were migrated, or an empty slice if file storage is not configured.
func (s *Storage) MigrateLegacyFilenames() ([]string, error) {
	if s.files == nil {
		return nil, nil
	}
	return s.files.MigrateLegacyFilenames()
}
