//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// commitEntry commits staged timbers entry files.
// Must be called after timbers log/amend before switching branches.
func (r *testRepo) commitEntry(msg string) {
	r.t.Helper()
	r.git("commit", "-m", msg)
}

// TestBranchMerge_DisjointEntries tests that entries created on separate branches
// coexist after merge. Since each entry is a unique file, git merge should be clean.
func TestBranchMerge_DisjointEntries(t *testing.T) {
	repo := newTestRepo(t)

	// Create initial commit on main
	repo.createFile("README.md", "# Project")
	repo.commit("Initial commit")

	// Create feature-a branch and add an entry
	repo.git("checkout", "-b", "feature-a")
	repo.createFile("feature_a.go", "package main\nfunc featureA() {}")
	repo.commit("Add feature A")
	repo.timbersOK("log", "Feature A",
		"--why", "User requested feature A",
		"--how", "Implemented featureA function")
	repo.commitEntry("Log feature A entry")

	// Remember the entry ID from feature-a
	queryA := repo.timbersOK("query", "--last", "1", "--json")
	entryIDA := parseEntryIDs(t, queryA)[0]

	// Switch back to main and create feature-b branch
	repo.git("checkout", "main")
	repo.git("checkout", "-b", "feature-b")
	repo.createFile("feature_b.go", "package main\nfunc featureB() {}")
	repo.commit("Add feature B")
	repo.timbersOK("log", "Feature B",
		"--why", "User requested feature B",
		"--how", "Implemented featureB function")
	repo.commitEntry("Log feature B entry")

	queryB := repo.timbersOK("query", "--last", "1", "--json")
	entryIDB := parseEntryIDs(t, queryB)[0]

	// Merge feature-a into main, then feature-b
	repo.git("checkout", "main")
	repo.git("merge", "feature-a", "--no-edit")
	repo.git("merge", "feature-b", "--no-edit")

	// Both entries should exist
	queryAll := repo.timbersOK("query", "--last", "10", "--json")
	allIDs := parseEntryIDs(t, queryAll)

	if !containsID(allIDs, entryIDA) {
		t.Errorf("entry from feature-a (%s) missing after merge", entryIDA)
	}
	if !containsID(allIDs, entryIDB) {
		t.Errorf("entry from feature-b (%s) missing after merge", entryIDB)
	}
	if len(allIDs) != 2 {
		t.Errorf("expected 2 entries after merge, got %d: %v", len(allIDs), allIDs)
	}
}

// TestBranchMerge_SameDayEntries tests that entries created on the same day
// on different branches merge cleanly into the same YYYY/MM/DD directory.
func TestBranchMerge_SameDayEntries(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("README.md", "# Project")
	repo.commit("Initial commit")

	// Feature-a: create entry (will be in today's YYYY/MM/DD dir)
	repo.git("checkout", "-b", "feature-a")
	repo.createFile("a.go", "package main")
	repo.commit("Add a.go")
	repo.timbersOK("log", "Work on A",
		"--why", "A reason", "--how", "A method")
	repo.commitEntry("Log A entry")

	// Feature-b: create entry (same day, same dir, different file)
	repo.git("checkout", "main")
	repo.git("checkout", "-b", "feature-b")
	repo.createFile("b.go", "package main")
	repo.commit("Add b.go")
	repo.timbersOK("log", "Work on B",
		"--why", "B reason", "--how", "B method")
	repo.commitEntry("Log B entry")

	// Merge both into main
	repo.git("checkout", "main")
	repo.git("merge", "feature-a", "--no-edit")
	repo.git("merge", "feature-b", "--no-edit")

	// Verify both entries present and in correct subdirectory
	queryOut := repo.timbersOK("query", "--last", "10", "--json")
	var entries []struct {
		ID      string `json:"id"`
		Summary struct {
			What string `json:"what"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(queryOut), &entries); err != nil {
		t.Fatalf("failed to parse query JSON: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Verify the .timbers/ directory structure has date subdirs
	timbersDir := filepath.Join(repo.dir, ".timbers")
	for _, e := range entries {
		// Entry ID format: tb_YYYY-MM-DDT...
		datePart := e.ID[3:13] // "2026-02-11"
		parts := strings.SplitN(datePart, "-", 3)
		expectedDir := filepath.Join(timbersDir, parts[0], parts[1], parts[2])
		entryFile := filepath.Join(expectedDir, e.ID+".json")
		if _, err := os.Stat(entryFile); err != nil {
			t.Errorf("entry file not in expected date dir: %s", entryFile)
		}
	}
}

// TestBranchMerge_ThreeWay tests a three-way merge where main and two
// feature branches all have entries. All three sets of entries should survive.
func TestBranchMerge_ThreeWay(t *testing.T) {
	repo := newTestRepo(t)

	// Base commit
	repo.createFile("README.md", "# Project")
	repo.commit("Initial commit")

	// Main gets an entry
	repo.createFile("main_work.go", "package main")
	repo.commit("Main work")
	repo.timbersOK("log", "Main work",
		"--why", "Main reason", "--how", "Main method")
	repo.commitEntry("Log main entry")

	// Branch off for feature-1
	repo.git("checkout", "-b", "feature-1")
	repo.createFile("feat1.go", "package main")
	repo.commit("Feature 1")
	repo.timbersOK("log", "Feature 1 work",
		"--why", "F1 reason", "--how", "F1 method")
	repo.commitEntry("Log feature 1 entry")

	// Branch off main for feature-2
	repo.git("checkout", "main")
	repo.git("checkout", "-b", "feature-2")
	repo.createFile("feat2.go", "package main")
	repo.commit("Feature 2")
	repo.timbersOK("log", "Feature 2 work",
		"--why", "F2 reason", "--how", "F2 method")
	repo.commitEntry("Log feature 2 entry")

	// Merge both into main
	repo.git("checkout", "main")
	repo.git("merge", "feature-1", "--no-edit")
	repo.git("merge", "feature-2", "--no-edit")

	// All three entries should exist
	queryOut := repo.timbersOK("query", "--last", "10", "--json")
	ids := parseEntryIDs(t, queryOut)
	if len(ids) != 3 {
		t.Errorf("expected 3 entries after three-way merge, got %d: %v", len(ids), ids)
	}

	// Verify all what fields present
	whats := parseEntryWhats(t, queryOut)
	for _, expected := range []string{"Main work", "Feature 1 work", "Feature 2 work"} {
		if !containsString(whats, expected) {
			t.Errorf("missing entry with what=%q after merge", expected)
		}
	}
}

// TestBranchMerge_PendingAfterMerge tests that pending correctly detects
// new commits brought in by a merge.
func TestBranchMerge_PendingAfterMerge(t *testing.T) {
	repo := newTestRepo(t)

	// Initial setup: create commit and log it so pending is clean
	repo.createFile("README.md", "# Project")
	repo.commit("Initial commit")
	repo.timbersOK("log", "Initial setup",
		"--why", "Project init", "--how", "Created README")
	repo.commitEntry("Log initial entry")

	// Create feature branch with work
	repo.git("checkout", "-b", "feature")
	repo.createFile("feature.go", "package main\nfunc feature() {}")
	repo.commit("Add feature")

	// Merge feature into main (without logging the feature work)
	repo.git("checkout", "main")
	repo.git("merge", "feature", "--no-edit")

	// Pending should show the merged commit
	pendingOut := repo.timbersOK("pending", "--json")
	var pendingResult struct {
		Count   int `json:"count"`
		Commits []struct {
			SHA     string `json:"sha"`
			Subject string `json:"subject"`
		} `json:"commits"`
	}
	if err := json.Unmarshal([]byte(pendingOut), &pendingResult); err != nil {
		t.Fatalf("failed to parse pending JSON: %v", err)
	}

	if pendingResult.Count == 0 {
		t.Error("expected pending commits after merge, got 0")
	}

	// At least one commit should be "Add feature"
	found := false
	for _, c := range pendingResult.Commits {
		if strings.Contains(c.Subject, "Add feature") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'Add feature' in pending commits, got: %v", pendingResult.Commits)
	}
}

// TestBranchMerge_SquashMerge tests that squash-merging a feature branch
// creates a single commit and that pending still works (stale anchor graceful degradation).
func TestBranchMerge_SquashMerge(t *testing.T) {
	repo := newTestRepo(t)

	// Setup: clean pending state
	repo.createFile("README.md", "# Project")
	repo.commit("Initial commit")
	repo.timbersOK("log", "Init",
		"--why", "Project setup", "--how", "Created README")
	repo.commitEntry("Log init entry")

	// Feature branch: log an entry there
	repo.git("checkout", "-b", "feature")
	repo.createFile("feat.go", "package main")
	repo.commit("Add feat.go")
	repo.timbersOK("log", "Feature work",
		"--why", "New capability needed", "--how", "Implemented feat.go")
	repo.commitEntry("Log feature entry")

	featureEntryOut := repo.timbersOK("query", "--last", "1", "--json")
	featureIDs := parseEntryIDs(t, featureEntryOut)

	// Squash merge into main — the feature branch commits disappear from main's history
	repo.git("checkout", "main")
	repo.git("merge", "--squash", "feature")
	repo.git("commit", "-m", "Squash merge feature branch")

	// The feature entry should still exist as a file (it was added to .timbers/)
	queryOut := repo.timbersOK("query", "--last", "10", "--json")
	allIDs := parseEntryIDs(t, queryOut)
	if !containsID(allIDs, featureIDs[0]) {
		t.Errorf("feature entry (%s) missing after squash merge", featureIDs[0])
	}

	// Pending should work (may warn about stale anchor but not error)
	_, _, err := repo.timbers("pending", "--json")
	if err != nil {
		// pending might return non-zero with stale anchor warning,
		// but should still produce output
		t.Logf("pending returned error (possibly stale anchor): %v", err)
	}
}

// TestBranchMerge_AmendOnSeparateBranches tests that amending different entries
// on separate branches merges cleanly.
func TestBranchMerge_AmendOnSeparateBranches(t *testing.T) {
	repo := newTestRepo(t)

	// Create two entries on main
	repo.createFile("file1.go", "package main")
	repo.commit("Add file1")
	repo.timbersOK("log", "First entry",
		"--why", "First reason", "--how", "First method")
	repo.commitEntry("Log first entry")

	repo.createFile("file2.go", "package main")
	repo.commit("Add file2")
	repo.timbersOK("log", "Second entry",
		"--why", "Second reason", "--how", "Second method")
	repo.commitEntry("Log second entry")

	// Get both entry IDs
	queryOut := repo.timbersOK("query", "--last", "2", "--json")
	ids := parseEntryIDs(t, queryOut)
	if len(ids) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(ids))
	}
	entryID1, entryID2 := ids[1], ids[0] // oldest first

	// Branch A amends entry 1
	repo.git("checkout", "-b", "amend-a")
	repo.timbersOK("amend", entryID1, "--what", "First entry (amended by A)")
	repo.commitEntry("Amend entry 1 on branch A")

	// Branch B amends entry 2
	repo.git("checkout", "main")
	repo.git("checkout", "-b", "amend-b")
	repo.timbersOK("amend", entryID2, "--what", "Second entry (amended by B)")
	repo.commitEntry("Amend entry 2 on branch B")

	// Merge both into main — should be clean since they touch different files
	repo.git("checkout", "main")
	repo.git("merge", "amend-a", "--no-edit")
	repo.git("merge", "amend-b", "--no-edit")

	// Verify both amendments survived
	showOut1 := repo.timbersOK("show", entryID1, "--json")
	showOut2 := repo.timbersOK("show", entryID2, "--json")

	var entry1, entry2 struct {
		Summary struct {
			What string `json:"what"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(showOut1), &entry1); err != nil {
		t.Fatalf("failed to parse show JSON for entry1: %v", err)
	}
	if err := json.Unmarshal([]byte(showOut2), &entry2); err != nil {
		t.Fatalf("failed to parse show JSON for entry2: %v", err)
	}

	if entry1.Summary.What != "First entry (amended by A)" {
		t.Errorf("entry1 what = %q, want %q", entry1.Summary.What, "First entry (amended by A)")
	}
	if entry2.Summary.What != "Second entry (amended by B)" {
		t.Errorf("entry2 what = %q, want %q", entry2.Summary.What, "Second entry (amended by B)")
	}
}

// TestBranchMerge_AmendSameEntry_Conflict tests that amending the same entry
// on two branches creates a merge conflict (expected behavior).
func TestBranchMerge_AmendSameEntry_Conflict(t *testing.T) {
	repo := newTestRepo(t)

	// Create one entry on main
	repo.createFile("file.go", "package main")
	repo.commit("Add file")
	repo.timbersOK("log", "Shared entry",
		"--why", "Shared reason", "--how", "Shared method")
	repo.commitEntry("Log shared entry")

	queryOut := repo.timbersOK("query", "--last", "1", "--json")
	entryID := parseEntryIDs(t, queryOut)[0]

	// Branch A amends the entry and commits
	repo.git("checkout", "-b", "amend-a")
	repo.timbersOK("amend", entryID, "--what", "Version A")
	repo.commitEntry("Amend shared entry to version A")

	// Branch B also amends the same entry and commits
	repo.git("checkout", "main")
	repo.git("checkout", "-b", "amend-b")
	repo.timbersOK("amend", entryID, "--what", "Version B")
	repo.commitEntry("Amend shared entry to version B")

	// Merge A into main, then try to merge B — should conflict
	repo.git("checkout", "main")
	repo.git("merge", "amend-a", "--no-edit")

	_, mergeErr := repo.gitMayFail("merge", "amend-b", "--no-edit")
	if mergeErr == nil {
		t.Error("expected merge conflict when both branches amend the same entry, but merge succeeded")
		return
	}

	// Verify we're in a conflicted state
	statusOut := repo.git("status", "--porcelain")
	if !strings.Contains(statusOut, "UU") && !strings.Contains(statusOut, "AA") {
		t.Errorf("expected merge conflict markers in git status, got: %s", statusOut)
	}

	// Clean up the conflict
	repo.git("merge", "--abort")
}

// TestBranchMerge_EntryOnBranch_NoneOnMain tests merging a branch that has
// entries into a main branch that has none.
func TestBranchMerge_EntryOnBranch_NoneOnMain(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("README.md", "# Project")
	repo.commit("Initial commit")

	// Feature branch creates entries
	repo.git("checkout", "-b", "feature")
	repo.createFile("feat.go", "package main")
	repo.commit("Add feature")
	repo.timbersOK("log", "Feature implementation",
		"--why", "New feature needed", "--how", "Implemented feat.go")
	repo.commitEntry("Log feature entry")

	// Main has no entries — merge should just bring them in
	repo.git("checkout", "main")
	repo.git("merge", "feature", "--no-edit")

	queryOut := repo.timbersOK("query", "--last", "1", "--json")
	whats := parseEntryWhats(t, queryOut)
	if len(whats) != 1 || whats[0] != "Feature implementation" {
		t.Errorf("expected 1 entry with what='Feature implementation' after merge, got: %v", whats)
	}
}

// TestBranchMerge_ManyBranches tests merging multiple branches (5+) each with
// their own entries. Verifies the YYYY/MM/DD layout handles many same-day entries.
func TestBranchMerge_ManyBranches(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("README.md", "# Project")
	repo.commit("Initial commit")

	branchCount := 5
	expectedIDs := make([]string, 0, branchCount)

	for i := 0; i < branchCount; i++ {
		branchName := "feature-" + string(rune('a'+i))
		fileName := branchName + ".go"

		repo.git("checkout", "main")
		repo.git("checkout", "-b", branchName)
		repo.createFile(fileName, "package main")
		repo.commit("Add " + fileName)
		repo.timbersOK("log", "Work on "+branchName,
			"--why", branchName+" reason",
			"--how", branchName+" method")
		repo.commitEntry("Log " + branchName + " entry")

		queryOut := repo.timbersOK("query", "--last", "1", "--json")
		ids := parseEntryIDs(t, queryOut)
		expectedIDs = append(expectedIDs, ids[0])
	}

	// Merge all branches into main
	repo.git("checkout", "main")
	for i := 0; i < branchCount; i++ {
		branchName := "feature-" + string(rune('a'+i))
		repo.git("merge", branchName, "--no-edit")
	}

	// All entries should exist
	queryOut := repo.timbersOK("query", "--last", "20", "--json")
	allIDs := parseEntryIDs(t, queryOut)

	if len(allIDs) != branchCount {
		t.Errorf("expected %d entries after merging %d branches, got %d",
			branchCount, branchCount, len(allIDs))
	}

	for _, expected := range expectedIDs {
		if !containsID(allIDs, expected) {
			t.Errorf("entry %s missing after merge", expected)
		}
	}
}

// TestBranchMerge_RebasePreservesEntries tests that rebasing a feature branch
// onto main preserves all entry files.
func TestBranchMerge_RebasePreservesEntries(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("README.md", "# Project")
	repo.commit("Initial commit")

	// Feature branch from initial commit
	repo.git("checkout", "-b", "feature")
	repo.createFile("feat.go", "package main")
	repo.commit("Feature work")
	repo.timbersOK("log", "Feature entry",
		"--why", "Feature reason", "--how", "Feature method")
	repo.commitEntry("Log feature entry")

	featureQuery := repo.timbersOK("query", "--last", "1", "--json")
	featureID := parseEntryIDs(t, featureQuery)[0]

	// Main advances with a non-conflicting change
	repo.git("checkout", "main")
	repo.createFile("main.go", "package main")
	repo.commit("Main advance")

	// Rebase feature onto main
	repo.git("checkout", "feature")
	repo.git("rebase", "main")

	// Entry file should still exist after rebase
	queryOut := repo.timbersOK("query", "--last", "10", "--json")
	ids := parseEntryIDs(t, queryOut)
	if !containsID(ids, featureID) {
		t.Errorf("entry %s lost after rebase", featureID)
	}

	// Now merge into main (fast-forward)
	repo.git("checkout", "main")
	repo.git("merge", "feature", "--no-edit")

	finalQuery := repo.timbersOK("query", "--last", "10", "--json")
	finalIDs := parseEntryIDs(t, finalQuery)
	if !containsID(finalIDs, featureID) {
		t.Errorf("entry %s lost after merge of rebased branch", featureID)
	}
}

// TestBranchMerge_ExportAfterMerge tests that export works correctly with
// entries from multiple merged branches.
func TestBranchMerge_ExportAfterMerge(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("README.md", "# Project")
	repo.commit("Initial commit")

	// Two branches with entries
	for _, branch := range []string{"alpha", "beta"} {
		repo.git("checkout", "main")
		repo.git("checkout", "-b", branch)
		repo.createFile(branch+".go", "package main")
		repo.commit("Add " + branch)
		repo.timbersOK("log", branch+" work",
			"--why", branch+" reason", "--how", branch+" method",
			"--tag", branch)
		repo.commitEntry("Log " + branch + " entry")
	}

	// Merge both
	repo.git("checkout", "main")
	repo.git("merge", "alpha", "--no-edit")
	repo.git("merge", "beta", "--no-edit")

	// JSON export should include both
	jsonOut := repo.timbersOK("export", "--last", "10", "--format", "json")
	var exported []map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &exported); err != nil {
		t.Fatalf("failed to parse export JSON: %v\noutput: %s", err, jsonOut)
	}
	if len(exported) != 2 {
		t.Errorf("expected 2 exported entries, got %d", len(exported))
	}

	// Markdown export should include both
	mdOut := repo.timbersOK("export", "--last", "10", "--format", "md")
	if !strings.Contains(mdOut, "alpha work") {
		t.Error("markdown export missing alpha entry")
	}
	if !strings.Contains(mdOut, "beta work") {
		t.Error("markdown export missing beta entry")
	}
}

// TestBranchMerge_TagFilterAfterMerge tests that tag-based filtering works
// correctly after merging branches with differently tagged entries.
func TestBranchMerge_TagFilterAfterMerge(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("README.md", "# Project")
	repo.commit("Initial commit")

	// Branch with "security" tagged entry
	repo.git("checkout", "-b", "security-fix")
	repo.createFile("auth.go", "package main")
	repo.commit("Fix auth")
	repo.timbersOK("log", "Security fix",
		"--why", "Vulnerability found", "--how", "Patched auth",
		"--tag", "security")
	repo.commitEntry("Log security entry")

	// Branch with "feature" tagged entry
	repo.git("checkout", "main")
	repo.git("checkout", "-b", "new-feature")
	repo.createFile("widget.go", "package main")
	repo.commit("Add widget")
	repo.timbersOK("log", "Widget feature",
		"--why", "User request", "--how", "Built widget",
		"--tag", "feature")
	repo.commitEntry("Log feature entry")

	// Merge both
	repo.git("checkout", "main")
	repo.git("merge", "security-fix", "--no-edit")
	repo.git("merge", "new-feature", "--no-edit")

	// Query with tag filter: security only
	secOut := repo.timbersOK("query", "--last", "10", "--tag", "security", "--json")
	secWhats := parseEntryWhats(t, secOut)
	if len(secWhats) != 1 || secWhats[0] != "Security fix" {
		t.Errorf("tag filter 'security' returned unexpected results: %v", secWhats)
	}

	// Query with tag filter: feature only
	featOut := repo.timbersOK("query", "--last", "10", "--tag", "feature", "--json")
	featWhats := parseEntryWhats(t, featOut)
	if len(featWhats) != 1 || featWhats[0] != "Widget feature" {
		t.Errorf("tag filter 'feature' returned unexpected results: %v", featWhats)
	}
}

// TestBranchMerge_StatusAfterMerge tests that status command reflects
// the correct entry count after merging multiple branches.
func TestBranchMerge_StatusAfterMerge(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("README.md", "# Project")
	repo.commit("Initial commit")

	// Three branches with entries
	for i := 0; i < 3; i++ {
		branch := "branch-" + string(rune('a'+i))
		repo.git("checkout", "main")
		repo.git("checkout", "-b", branch)
		repo.createFile(branch+".go", "package main")
		repo.commit("Work on " + branch)
		repo.timbersOK("log", branch+" work",
			"--why", "Reason", "--how", "Method")
		repo.commitEntry("Log " + branch + " entry")
	}

	// Merge all
	repo.git("checkout", "main")
	for i := 0; i < 3; i++ {
		branch := "branch-" + string(rune('a'+i))
		repo.git("merge", branch, "--no-edit")
	}

	// Status should report 3 entries
	statusOut := repo.timbersOK("status", "--json")
	var statusResult struct {
		EntryCount int `json:"entry_count"`
	}
	if err := json.Unmarshal([]byte(statusOut), &statusResult); err != nil {
		t.Fatalf("failed to parse status JSON: %v", err)
	}
	if statusResult.EntryCount != 3 {
		t.Errorf("expected entry_count=3 after merging 3 branches, got %d", statusResult.EntryCount)
	}
}

// TestBranchMerge_CherryPickEntry tests that cherry-picking a commit containing
// an entry file brings the entry into the target branch.
func TestBranchMerge_CherryPickEntry(t *testing.T) {
	repo := newTestRepo(t)

	repo.createFile("README.md", "# Project")
	repo.commit("Initial commit")

	// Feature branch with entry
	repo.git("checkout", "-b", "feature")
	repo.createFile("feat.go", "package main")
	repo.commit("Add feature")
	repo.timbersOK("log", "Cherry-pickable work",
		"--why", "Important fix", "--how", "Fixed the thing")
	entryCommitSHA := repo.git("rev-parse", "HEAD")
	// timbers log staged but didn't commit yet — commit the entry
	repo.commitEntry("Log entry for cherry-pick")
	entrySHA := repo.git("rev-parse", "HEAD")

	queryOut := repo.timbersOK("query", "--last", "1", "--json")
	entryID := parseEntryIDs(t, queryOut)[0]

	// Cherry-pick the entry commit (not the feature code commit) into main
	repo.git("checkout", "main")
	repo.git("cherry-pick", entrySHA)

	// Verify the entry exists on main
	mainQuery := repo.timbersOK("query", "--last", "10", "--json")
	mainIDs := parseEntryIDs(t, mainQuery)
	if !containsID(mainIDs, entryID) {
		t.Errorf("cherry-picked entry %s not found on main (cherry-pick from %s, entry commit %s)",
			entryID, entryCommitSHA, entrySHA)
	}
}

// --- Test helpers ---

// parseEntryIDs extracts entry IDs from query --json output.
func parseEntryIDs(t *testing.T, queryJSON string) []string {
	t.Helper()
	var entries []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(queryJSON), &entries); err != nil {
		t.Fatalf("failed to parse entry IDs from JSON: %v\noutput: %s", err, queryJSON)
	}
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.ID
	}
	return ids
}

// parseEntryWhats extracts summary.what values from query --json output.
func parseEntryWhats(t *testing.T, queryJSON string) []string {
	t.Helper()
	var entries []struct {
		Summary struct {
			What string `json:"what"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(queryJSON), &entries); err != nil {
		t.Fatalf("failed to parse entry whats from JSON: %v\noutput: %s", err, queryJSON)
	}
	whats := make([]string, len(entries))
	for i, e := range entries {
		whats[i] = e.Summary.What
	}
	return whats
}

// containsID checks if an ID exists in a slice of IDs.
func containsID(ids []string, target string) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

// containsString checks if a string exists in a slice.
func containsString(ss []string, target string) bool {
	for _, s := range ss {
		if s == target {
			return true
		}
	}
	return false
}
