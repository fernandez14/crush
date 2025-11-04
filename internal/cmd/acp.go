package cmd

import (
	"context"
	"log/slog"
	"os"

	"github.com/charmbracelet/crush/internal/acp"
	"github.com/spf13/cobra"
)

var acpCmd = &cobra.Command{
	Use:   "acp",
	Short: "Run Crush as an Agent Client Protocol (ACP) agent",
	Long: `Run Crush as an ACP-compatible agent that communicates via JSON-RPC 2.0 over stdio.

This allows Crush to be used as an AI coding agent in ACP-compatible IDEs like Zed.

The agent listens for JSON-RPC messages on stdin and sends responses on stdout.`,
	Example: `
# Run Crush as an ACP agent
crush acp

# Run with debug logging (logs go to stderr)
crush acp -d

# Use with an ACP-compatible IDE
# In Zed: Configure the agent path to point to 'crush acp'
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Setup app without TUI
		app, err := setupApp(cmd)
		if err != nil {
			return err
		}
		defer app.Shutdown()

		slog.Info("Starting Crush ACP agent", "protocol_version", acp.ProtocolVersion)

		// Create the Crush ACP agent
		crushAgent := acp.NewCrushAgent(app)

		// Create the ACP server
		server := acp.NewServer(crushAgent, os.Stdin, os.Stdout)
		crushAgent.SetServer(server)

		// Run the server (blocks until stdin is closed)
		ctx := context.Background()
		if err := server.Run(ctx); err != nil {
			slog.Error("ACP server error", "error", err)
			return err
		}

		slog.Info("ACP agent shutdown")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(acpCmd)
}
