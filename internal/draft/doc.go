// Package draft provides template loading, resolution, and rendering for LLM prompts.
//
// Templates are resolved in order:
//  1. .timbers/templates/<name>.md (project-local)
//  2. ~/.config/timbers/templates/<name>.md (user global)
//  3. Built-in templates (embedded in binary)
package draft
