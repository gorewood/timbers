package main

import "github.com/gorewood/timbers/internal/output"

// outputDryRunHumanInit prints dry-run output in human format.
func outputDryRunHumanInit(printer *output.Printer, styles initStyleSet, repoName string, steps []initStepResult) {
	printer.Println()
	printer.Print("%s %s\n", styles.heading.Render("Dry run: timbers init in"), styles.dim.Render(repoName))
	printer.Println()

	for _, step := range steps {
		icon := styledDryRunIcon(styles, step.Status)
		printer.Print("  %s %s: %s\n", icon, step.Name, step.Message)
	}
}

// styledDryRunIcon returns a styled icon for a dry-run step status.
func styledDryRunIcon(styles initStyleSet, status string) string {
	switch status {
	case "skipped":
		return styles.dim.Render("--")
	case "dry_run":
		return styles.accent.Render(">")
	default:
		return "?"
	}
}

// printNextSteps outputs the next steps message.
func printNextSteps(printer *output.Printer, styles initStyleSet) {
	printer.Println()
	printer.Print("%s\n", styles.heading.Render(styles.pass.Render("Timbers initialized!")))
	printer.Println()
	printer.Print("Next steps:\n")
	printer.Print("  1. %s\n", styles.dim.Render("Add the timbers snippet to CLAUDE.md:"))
	printer.Print("     %s\n", styles.accent.Render("timbers onboard >> CLAUDE.md"))
	printer.Println()
	printer.Print("  2. %s\n", styles.dim.Render("Start documenting work:"))
	printer.Print("     %s\n", styles.accent.Render("timbers log \"what\" --why \"why\" --how \"how\""))
	printer.Println()
	printer.Print("  3. %s\n", styles.dim.Render("Verify setup:"))
	printer.Print("     %s\n", styles.accent.Render("timbers doctor"))
}

// printStepResult prints a single step result in human format.
func printStepResult(printer *output.Printer, styles initStyleSet, step initStepResult) {
	icon := styledStepIcon(styles, step.Status)
	name := formatStepName(step.Name)
	printer.Print("  %s %s", icon, name)
	if step.Message != "" {
		printer.Print(" %s", styles.dim.Render("("+step.Message+")"))
	}
	printer.Println()
}

// styledStepIcon returns a styled icon for a step status.
func styledStepIcon(styles initStyleSet, status string) string {
	switch status {
	case "ok":
		return styles.pass.Render("ok")
	case "skipped":
		return styles.skip.Render("--")
	case "failed":
		return styles.fail.Render("XX")
	default:
		return "??"
	}
}

// formatStepName converts internal step names to display names.
func formatStepName(name string) string {
	switch name {
	case "timbers_dir":
		return ".timbers directory"
	case "gitattributes":
		return ".gitattributes"
	case "hooks":
		return "Git hooks"
	case "post_rewrite":
		return "Post-rewrite hook"
	case "claude":
		return "Claude integration"
	default:
		return name
	}
}

// generatePostRewriteHook returns the full post-rewrite hook script.
func generatePostRewriteHook() string {
	return "#!/bin/sh\n" + postRewriteTimbersSection()
}

// postRewriteTimbersSection returns the timbers SHA remapping section for the post-rewrite hook.
func postRewriteTimbersSection() string {
	return `# timbers post-rewrite hook
# Remaps SHAs in .timbers/ entries after rebase
while IFS=' ' read -r old_sha new_sha _extra; do
  old_short="${old_sha%"${old_sha#???????}"}"
  new_short="${new_sha%"${new_sha#???????}"}"
  for f in .timbers/*.json; do
    [ -f "$f" ] || continue
    if grep -q "$old_sha\|$old_short" "$f"; then
      sed -i.bak \
        -e "s/$old_sha/$new_sha/g" \
        -e "s/$old_short/$new_short/g" \
        "$f"
      rm -f "$f.bak"
    fi
  done
done
`
}
