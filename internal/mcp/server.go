// Package mcp provides a Model Context Protocol server for timbers.
// It exposes ledger operations as MCP tools that any MCP-capable agent can use.
package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/gorewood/timbers/internal/ledger"
)

// NewServer creates an MCP server with all timbers tools registered.
func NewServer(version string, storage *ledger.Storage) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "timbers",
		Version: version,
	}, nil)
	registerTools(server, storage)
	return server
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}

// readOnlyAnnotations returns annotations for read-only tools.
func readOnlyAnnotations() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  boolPtr(false),
	}
}

// writeAnnotations returns annotations for write tools (additive, not destructive).
func writeAnnotations() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		DestructiveHint: boolPtr(false),
		OpenWorldHint:   boolPtr(false),
	}
}

// registerTools adds all timbers tools to the server.
func registerTools(server *mcp.Server, storage *ledger.Storage) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "pending",
		Description: "Show undocumented commits since the last ledger entry. Returns the count and list of commits that need to be documented.",
		Annotations: readOnlyAnnotations(),
	}, handlePending(storage))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "prime",
		Description: "Get session bootstrapping context: repo info, recent entries, pending commits, and workflow instructions.",
		Annotations: readOnlyAnnotations(),
	}, handlePrime(storage))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query",
		Description: "Search and retrieve ledger entries with filters. Supports --last N, --since/--until time ranges, and --tags filtering.",
		Annotations: readOnlyAnnotations(),
	}, handleQuery(storage))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "show",
		Description: "Display a single ledger entry by ID, or the most recent entry with latest=true.",
		Annotations: readOnlyAnnotations(),
	}, handleShow(storage))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "status",
		Description: "Show repository and ledger state: repo name, branch, HEAD, entry count, and directory status.",
		Annotations: readOnlyAnnotations(),
	}, handleStatus(storage))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "log",
		Description: "Record work as a ledger entry with what/why/how. Writes the entry file and stages it.",
		Annotations: writeAnnotations(),
	}, handleLog(storage))
}
