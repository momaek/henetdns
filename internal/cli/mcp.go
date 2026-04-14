package cli

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	"github.com/momaek/henetdns/internal/app"
	mcpserver "github.com/momaek/henetdns/internal/mcp"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "mcp", Short: "MCP server operations"}
	cmd.AddCommand(newMCPServeCmd())
	return cmd
}

func newMCPServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start an MCP stdio server exposing DNS tools",
		Long: `Start a Model Context Protocol (MCP) server over stdio.

Exposes four tools: list_zones, list_records, upsert_record, delete_record.

Authentication: run 'henetdns login' first to persist a session cookie.
The server starts even without an active session; tools that require auth
return an actionable error message if no session is found.

Claude Desktop configuration (~/.config/claude/claude_desktop_config.json):
  {
    "mcpServers": {
      "henetdns": {
        "command": "henetdns",
        "args": ["mcp", "serve"]
      }
    }
  }`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := app.NewRuntime(cfg)
			if err != nil {
				return err
			}
			defer rt.Close()

			s := mcpserver.NewServer(rt, cfg.Username)
			return server.ServeStdio(s)
		},
	}
}
