package protocol

import (
	"strings"
	"testing"
)

// TestSessionProtocol_ContainsOrderingRule sanity-checks that the
// canonical session-protocol text includes the load-bearing ordering
// rule. If a future edit accidentally removes it, the consumers (prime
// and MCP) silently lose the guidance and the push-before-log race
// reappears.
func TestSessionProtocol_ContainsOrderingRule(t *testing.T) {
	wantSubstrs := []string{
		"commit → timbers log → push",
		"Never push between",
		"<protocol>",
		"</protocol>",
	}
	for _, want := range wantSubstrs {
		if !strings.Contains(SessionProtocol, want) {
			t.Errorf("SessionProtocol missing %q", want)
		}
	}
}

// TestSessionProtocol_ChecklistOrdering locks down the position of the
// three checklist anchors. Agents read these mechanically, so a future
// edit that reorders the bullets (e.g. "push → log") would silently
// re-introduce the push-before-log race even though the prose
// explanation above still reads correctly. Catch ordering drift here.
func TestSessionProtocol_ChecklistOrdering(t *testing.T) {
	commitPos := strings.Index(SessionProtocol, "git add && git commit")
	logPos := strings.Index(SessionProtocol, "timbers log \"what\"")
	pushPos := strings.Index(SessionProtocol, "git push (sends")
	if commitPos < 0 || logPos < 0 || pushPos < 0 {
		t.Fatalf("checklist anchors missing: commit=%d log=%d push=%d", commitPos, logPos, pushPos)
	}
	if commitPos >= logPos || logPos >= pushPos {
		t.Errorf("checklist ordering violated: commit=%d log=%d push=%d (want strictly ascending)",
			commitPos, logPos, pushPos)
	}
}

// TestStaleAnchorGuidance_ContainsCriticalRule sanity-checks that the
// canonical stale-anchor text includes the "do not re-document" rule.
// This guidance prevents agents from creating duplicate entries after
// squash merges.
func TestStaleAnchorGuidance_ContainsCriticalRule(t *testing.T) {
	wantSubstrs := []string{
		"Do NOT try to catch up",
		"anchor self-heals",
		"<stale-anchor>",
		"</stale-anchor>",
	}
	for _, want := range wantSubstrs {
		if !strings.Contains(StaleAnchorGuidance, want) {
			t.Errorf("StaleAnchorGuidance missing %q", want)
		}
	}
}

// TestRebaseRelinkGuidance_ContainsAckPattern sanity-checks that the
// rebase-relink text steers agents to ack the new SHA rather than write a
// duplicate entry. If a future edit drops the ack command or the
// content-preserved framing, agents lose the low-friction path and fall back
// to re-logging rebased work.
func TestRebaseRelinkGuidance_ContainsAckPattern(t *testing.T) {
	wantSubstrs := []string{
		"timbers ack <new-sha>",
		"content in <original-entry-id>",
		"does NOT self-heal",
		"<rebase-relink>",
		"</rebase-relink>",
	}
	for _, want := range wantSubstrs {
		if !strings.Contains(RebaseRelinkGuidance, want) {
			t.Errorf("RebaseRelinkGuidance missing %q", want)
		}
	}
}
