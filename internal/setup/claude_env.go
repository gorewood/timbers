package setup

// ClaudeEnv implements AgentEnv for Claude Code.
type ClaudeEnv struct{}

func init() {
	RegisterAgentEnv(&ClaudeEnv{})
}

// Name returns the CLI identifier.
func (c *ClaudeEnv) Name() string { return "claude" }

// DisplayName returns the human-readable name.
func (c *ClaudeEnv) DisplayName() string { return "Claude Code" }

// Detect checks whether Claude Code integration is installed at either scope.
func (c *ClaudeEnv) Detect() (path, scope string, installed bool) {
	// Check project first, then global.
	for _, project := range []bool{true, false} {
		hookPath, s, err := ResolveClaudeSettingsPath(project)
		if err != nil {
			continue
		}
		if IsTimbersSectionInstalled(hookPath) {
			return hookPath, s, true
		}
	}
	return "", "", false
}

// Install adds timbers hooks to Claude Code settings.
func (c *ClaudeEnv) Install(project bool) (string, error) {
	hookPath, _, err := ResolveClaudeSettingsPath(project)
	if err != nil {
		return "", err
	}
	if err := InstallTimbersSection(hookPath); err != nil {
		return "", err
	}
	return hookPath, nil
}

// Remove removes timbers hooks from Claude Code settings.
func (c *ClaudeEnv) Remove(project bool) error {
	hookPath, _, err := ResolveClaudeSettingsPath(project)
	if err != nil {
		return err
	}
	return RemoveTimbersSectionFromHook(hookPath)
}

// Check returns installation status for a specific scope.
func (c *ClaudeEnv) Check(project bool) (path, scope string, installed bool, err error) {
	hookPath, s, resolveErr := ResolveClaudeSettingsPath(project)
	if resolveErr != nil {
		return "", "", false, resolveErr
	}
	return hookPath, s, IsTimbersSectionInstalled(hookPath), nil
}
