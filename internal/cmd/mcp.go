package cmd

import (
	"github.com/spf13/cobra"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/basecamp/hey-cli/internal/version"
)

type mcpCommand struct {
	cmd *cobra.Command
}

func newMCPCommand() *mcpCommand {
	c := &mcpCommand{}
	c.cmd = &cobra.Command{
		Use:   "mcp",
		Short: "Run an MCP (Model Context Protocol) server over stdio",
		Long: `Run an MCP server that exposes HEY functionality (mail, search, calendar, todos,
habits, time tracking, journal) as tools callable by an MCP-aware host
(e.g. Claude Desktop, Cowork). The server speaks JSON-RPC over stdin/stdout
until the client disconnects.

Configure your MCP host with:

  {
    "mcpServers": {
      "hey": { "command": "hey", "args": ["mcp"] }
    }
  }`,
		Annotations: map[string]string{
			"agent_notes": "Starts a stdio MCP server. Blocks until the client disconnects. Requires authentication.",
		},
		RunE:          c.run,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	return c
}

func (c *mcpCommand) run(cmd *cobra.Command, _ []string) error {
	if err := requireAuth(); err != nil {
		return err
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "hey",
		Version: version.Version,
	}, nil)

	registerMCPTools(server)

	return server.Run(cmd.Context(), &mcp.StdioTransport{})
}
