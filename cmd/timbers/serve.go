// Package main provides the entry point for the timbers CLI.
package main

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/gorewood/timbers/internal/ledger"
	timbersmcp "github.com/gorewood/timbers/internal/mcp"
)

// newServeCmd creates the serve command for running as an MCP server.
func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run as MCP server (stdio transport)",
		Long: `Run timbers as a Model Context Protocol (MCP) server over stdio.

This exposes timbers operations as MCP tools that any MCP-capable agent
environment can use (Claude Code, Cursor, Windsurf, Gemini CLI, etc).

Configure in your agent's MCP settings:
  {
    "mcpServers": {
      "timbers": {
        "command": "timbers",
        "args": ["serve"]
      }
    }
  }

Available tools: pending, prime, query, show, status, log`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			storage, err := ledger.NewDefaultStorage()
			if err != nil {
				return err
			}
			server := timbersmcp.NewServer(buildVersion(), storage)
			return server.Run(cmd.Context(), &mcp.StdioTransport{})
		},
	}
}
