package setup

import "slices"

// AgentEnv describes an agent coding environment that timbers can integrate with.
// Each implementation handles detection, installation, and removal of timbers
// hooks for a specific tool (Claude Code, Gemini CLI, Codex, etc.).
type AgentEnv interface {
	// Name returns the short identifier used in CLI commands (e.g., "claude").
	Name() string

	// DisplayName returns the human-readable name (e.g., "Claude Code").
	DisplayName() string

	// Detect checks whether this agent environment's integration is installed.
	// Returns the settings path, scope ("project"/"global"), and whether installed.
	Detect() (path, scope string, installed bool)

	// Install adds timbers hooks to this agent environment's settings.
	// If project is true, installs to project-local settings; otherwise global.
	Install(project bool) (path string, err error)

	// Remove removes timbers hooks from this agent environment's settings.
	// If project is true, targets project-local settings; otherwise global.
	Remove(project bool) error

	// Check returns the settings path and scope for the given scope.
	// project=true returns the project-local path; false returns the global path.
	Check(project bool) (path, scope string, installed bool, err error)
}

// registry holds all known agent environments, keyed by name.
var registry = map[string]AgentEnv{}

// RegisterAgentEnv registers an agent environment implementation.
func RegisterAgentEnv(env AgentEnv) {
	registry[env.Name()] = env
}

// GetAgentEnv returns a registered agent environment by name, or nil if not found.
func GetAgentEnv(name string) AgentEnv {
	return registry[name]
}

// AllAgentEnvs returns all registered agent environments in a stable order.
func AllAgentEnvs() []AgentEnv {
	// Return in a deterministic order for consistent output.
	order := []string{"claude"}
	var result []AgentEnv
	for _, name := range order {
		if env, ok := registry[name]; ok {
			result = append(result, env)
		}
	}
	// Append any environments not in the explicit order (future additions).
	for name, env := range registry {
		if !slices.Contains(order, name) {
			result = append(result, env)
		}
	}
	return result
}

// DetectedAgentEnvs returns agent environments that have timbers installed.
func DetectedAgentEnvs() []AgentEnv {
	var detected []AgentEnv
	for _, env := range AllAgentEnvs() {
		if _, _, installed := env.Detect(); installed {
			detected = append(detected, env)
		}
	}
	return detected
}
