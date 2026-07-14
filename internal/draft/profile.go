package draft

import (
	"errors"
	"fmt"
)

const (
	// ProjectionNarrative includes authored what/why/how report fields.
	ProjectionNarrative = "narrative"
	// ProjectionDecision excludes implementation detail from decision reports.
	ProjectionDecision = "decision"
)

// ReportProfile defines portable report behavior in template frontmatter.
type ReportProfile struct {
	Scope       ReportScope `yaml:"scope"`
	Projection  string      `yaml:"projection"`
	Format      string      `yaml:"format"`
	QuietOutput string      `yaml:"quiet_output,omitempty"`
}

// ReportScope provides the profile's default entry selection.
type ReportScope struct {
	Last  string `yaml:"last,omitempty"`
	Since string `yaml:"since,omitempty"`
}

// Validate checks the supported first-slice profile grammar.
func (p *ReportProfile) Validate() error {
	if p == nil {
		return nil
	}
	if (p.Scope.Last == "") == (p.Scope.Since == "") {
		return errors.New("scope must contain exactly one of last or since")
	}
	if p.Projection != ProjectionNarrative && p.Projection != ProjectionDecision {
		return fmt.Errorf("unsupported projection %q", p.Projection)
	}
	if p.Format != "markdown" {
		return fmt.Errorf("unsupported format %q", p.Format)
	}
	return nil
}
