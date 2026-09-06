package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/schlunsen/wee-editor/internal/database"
	mcpserver "github.com/schlunsen/wee-editor/internal/mcp"
	"github.com/schlunsen/wee-editor/internal/server/agents"
	"github.com/spf13/cobra"
)

var mcpServerCmd = &cobra.Command{
	Use:   "mcp-server",
	Short: "Run standalone MCP server for agent handovers",
	Long: `Run a standalone Model Context Protocol (MCP) server that provides
agent session handover and ecosystem tools via stdio interface.

This command reads JSON-RPC requests from stdin and writes responses to stdout,
allowing Claude Code to use Wee tools natively through the MCP protocol.

The MCP server provides these tools:

  Session tools:
  - get_current_session: Auto-discover your session ID
  - list_sessions: List all agent sessions
  - create_handover: Create a handover package
  - accept_handover: Accept a handover token

  Ecosystem tools:
  - list_skills: Discover available skills
  - invoke_skill: Get skill instructions by name
  - get_hook_status: Query hook execution history
  - create_skill_from_handover: Package a session as a reusable skill

Configuration in .mcp.json:
{
  "mcpServers": {
    "wee-agent-ecosystem": {
      "command": "/path/to/wee",
      "args": ["mcp-server"]
    }
  }
}`,
	RunE: runMCPServer,
}

func init() {
	rootCmd.AddCommand(mcpServerCmd)
}

func runMCPServer(cmd *cobra.Command, args []string) error {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Initialize database
	claudeDir := filepath.Join(homeDir, ".claude")
	dataDir := filepath.Join(claudeDir, "wee")

	db, err := database.Initialize(dataDir)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Get API key for agent operations (optional for MCP server)
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("CLAUDE_API_KEY")
	}

	// Create agent config
	agentConfig := &agents.Config{
		Model:                 "claude-sonnet-4-20250514",
		APIKey:                apiKey,
		MaxConcurrentSessions: 10,
		Verbose:               false,
		SessionRetentionDays:  30,
		CleanupEnabled:        true,
		CleanupIntervalHours:  24,
	}

	// Create repository for skill/hook access
	repo := database.NewRepository(db)

	// Create session manager
	sessionManager, err := agents.NewSessionManager(agentConfig, db.GetDB(), repo)
	if err != nil {
		return fmt.Errorf("failed to create session manager: %w", err)
	}

	// Create MCP server with repository for ecosystem tools
	mcpServer := mcpserver.NewMCPServer(sessionManager, repo)

	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start reading in a goroutine
	errChan := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				errChan <- nil
				return
			default:
			}

			line := scanner.Bytes()

			// Parse MCP request
			var req mcpserver.MCPRequest
			if err := json.Unmarshal(line, &req); err != nil {
				// Send parse error response
				response := mcpserver.MCPResponse{
					JSONRPC: "2.0",
					ID:      nil,
					Error: &mcpserver.MCPError{
						Code:    -32700,
						Message: "Parse error: " + err.Error(),
					},
				}
				responseJSON, _ := json.Marshal(response)
				fmt.Println(string(responseJSON))
				continue
			}

			// Handle request
			response := mcpServer.HandleRequest(req)

			// Write response to stdout
			responseJSON, err := json.Marshal(response)
			if err != nil {
				// This should never happen, but handle it anyway
				errorResponse := mcpserver.MCPResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &mcpserver.MCPError{
						Code:    -32603,
						Message: "Internal error: failed to marshal response",
					},
				}
				errorJSON, _ := json.Marshal(errorResponse)
				fmt.Println(string(errorJSON))
				continue
			}

			fmt.Println(string(responseJSON))
		}

		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("error reading from stdin: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// Wait for either a signal or error
	select {
	case <-sigChan:
		cancel()
		return nil
	case err := <-errChan:
		return err
	}
}
