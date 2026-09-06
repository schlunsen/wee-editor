package terminal

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
)

// TerminalHandler manages WebSocket connections for terminal I/O.
type TerminalHandler struct {
	ptyManager *PTYManager
	store      TerminalSessionStore
	mu         sync.RWMutex
	// Map of terminal ID to active WebSocket connections
	connections map[string]*websocket.Conn
	// Map of terminal ID to write mutex for that connection
	writeMutex map[string]*sync.Mutex
	// Map of terminal ID to whether an output goroutine already exists
	hasOutputGoroutine map[string]bool
}

// NewTerminalHandler creates a new terminal handler.
func NewTerminalHandler(ptyManager *PTYManager, store TerminalSessionStore) *TerminalHandler {
	return &TerminalHandler{
		ptyManager:         ptyManager,
		store:              store,
		connections:        make(map[string]*websocket.Conn),
		writeMutex:         make(map[string]*sync.Mutex),
		hasOutputGoroutine: make(map[string]bool),
	}
}

// HandleWebSocket handles a WebSocket connection for a terminal session.
func (h *TerminalHandler) HandleWebSocket(c *websocket.Conn, agentSessionID string) error {
	defer c.Close()

	// Extract terminal_id from query parameters (set by frontend when connecting to existing terminal)
	terminalID := c.Query("terminal_id")
	var ptyFile interface{}

	// Variables for I/O handling (created when PTY is available)
	var inputChan chan []byte
	var outputDone chan error
	var inputDone chan error

	// If terminal_id is provided, try to retrieve the existing terminal session
	if terminalID != "" {
		session, err := h.ptyManager.GetSession(terminalID)
		if err != nil {
			log.Printf("failed to retrieve terminal session: %v", err)
			errMsg := WebSocketMessage{
				Type:  "terminal.error",
				Error: fmt.Sprintf("terminal session not found: %v", err),
			}
			c.WriteJSON(errMsg)
			return fmt.Errorf("failed to retrieve terminal: %w", err)
		}
		ptyFile = session.PTYFile
		log.Printf("retrieved existing terminal session: %s", terminalID)

		// Initialize I/O channels for existing terminal
		inputChan = make(chan []byte, 100)
		outputDone = make(chan error, 1)
		inputDone = make(chan error, 1)

		// Close and replace old connection if it exists (client reconnecting after tab switch)
		h.mu.Lock()
		if oldConn, exists := h.connections[terminalID]; exists {
			log.Printf("closing old connection for terminal %s (client reconnecting)", terminalID)
			oldConn.Close()
		}
		// Create or reuse write mutex for this terminal
		if _, exists := h.writeMutex[terminalID]; !exists {
			h.writeMutex[terminalID] = &sync.Mutex{}
		}
		h.connections[terminalID] = c
		h.mu.Unlock()
	}

	// Send ready message
	readyMsg := WebSocketMessage{
		Type: "terminal.ready",
	}
	if err := c.WriteJSON(readyMsg); err != nil {
		return fmt.Errorf("failed to send ready message: %w", err)
	}

	// Goroutine for reading WebSocket messages
	go func() {
		for {
			var msg WebSocketMessage
			if err := c.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					inputDone <- fmt.Errorf("websocket connection closed: %w", err)
				} else {
					inputDone <- nil
				}
				return
			}

			switch msg.Type {
			case "terminal.start":
				// Start a new terminal
				if workingDir, ok := msg.Data.(string); ok {
					// Spawn terminal with working directory passed from frontend
					session, err := h.ptyManager.SpawnTerminal(agentSessionID, workingDir, "")
					if err != nil {
						errMsg := WebSocketMessage{
							Type:  "terminal.error",
							Error: fmt.Sprintf("failed to start terminal: %v", err),
						}
						c.WriteJSON(errMsg)
						continue
					}

					terminalID = session.ID
					ptyFile = session.PTYFile

					// Initialize I/O channels for new terminal
					inputChan = make(chan []byte, 100)
					outputDone = make(chan error, 1)
					inputDone = make(chan error, 1)

					// Save to database
					if err := h.store.SaveTerminalSession(session); err != nil {
						log.Printf("failed to save terminal session: %v", err)
					}

					// Send started message
					startMsg := TerminalStartResponse{
						TerminalID: terminalID,
						Shell:      session.Shell,
						CWD:        session.WorkingDir,
						Rows:       session.Rows,
						Cols:       session.Cols,
					}
					response := WebSocketMessage{
						Type: "terminal.started",
						Data: startMsg,
					}
					c.WriteJSON(response)

					// Register connection
					h.mu.Lock()
					// Create write mutex for this terminal
					if _, exists := h.writeMutex[terminalID]; !exists {
						h.writeMutex[terminalID] = &sync.Mutex{}
					}
					h.connections[terminalID] = c
					h.mu.Unlock()
				}

			case "terminal.input":
				// Send input to terminal
				if data, ok := msg.Data.(string); ok {
					inputChan <- []byte(data)
				}

			case "terminal.resize":
				// Resize terminal
				if terminalID != "" {
					rows := int(msg.Rows)
					cols := int(msg.Cols)
					if err := h.ptyManager.ResizeTerminal(terminalID, rows, cols); err != nil {
						errMsg := WebSocketMessage{
							Type:  "terminal.error",
							Error: fmt.Sprintf("resize failed: %v", err),
						}
						c.WriteJSON(errMsg)
					}
				}

			case "terminal.close":
				// Close terminal
				if terminalID != "" {
					h.ptyManager.KillTerminal(terminalID)
					h.mu.Lock()
					delete(h.connections, terminalID)
					delete(h.writeMutex, terminalID)
					delete(h.hasOutputGoroutine, terminalID)
					h.mu.Unlock()

					// Mark terminal as ended in database
					session, err := h.store.GetTerminalSession(terminalID)
					if err == nil && session.EndedAt == nil {
						h.store.UpdateTerminalEnd(terminalID, session.ExitCode)
					}

					exitMsg := WebSocketMessage{
						Type: "terminal.exited",
						Data: map[string]interface{}{
							"terminal_id": terminalID,
						},
					}
					c.WriteJSON(exitMsg)
				}
				return
			}
		}
	}()

	// Goroutine for writing PTY output to WebSocket
	// Only create ONE output goroutine per terminal (prevents concurrent writes when reconnecting)
	if ptyFile != nil && terminalID != "" {
		h.mu.Lock()
		hasOutput := h.hasOutputGoroutine[terminalID]
		h.mu.Unlock()

		if !hasOutput {
			// Mark that we're creating an output goroutine for this terminal
			h.mu.Lock()
			h.hasOutputGoroutine[terminalID] = true
			h.mu.Unlock()

			go func() {
				buf := make([]byte, 1024)
				for {
					n, err := ptyFile.(interface {
						Read([]byte) (int, error)
					}).Read(buf)
					if err != nil {
						outputDone <- err
						return
					}

					if n > 0 {
						// Update activity
						if terminalID != "" {
							h.store.UpdateTerminalActivity(terminalID)
						}

						// Get current connection and write mutex
						h.mu.RLock()
						currentConn := h.connections[terminalID]
						writeMutex := h.writeMutex[terminalID]
						h.mu.RUnlock()

						if currentConn != nil && writeMutex != nil {
							// Send output with mutex protection to prevent concurrent writes
							outMsg := WebSocketMessage{
								Type: "terminal.output",
								Data: string(buf[:n]),
							}
							writeMutex.Lock()
							err := currentConn.WriteJSON(outMsg)
							writeMutex.Unlock()
							if err != nil {
								outputDone <- err
								return
							}
						}
					}
				}
			}()
		}
	}

	// Goroutine for writing input to PTY
	if ptyFile != nil {
		go func() {
			for data := range inputChan {
				if _, err := ptyFile.(interface {
					Write([]byte) (int, error)
				}).Write(data); err != nil {
					inputDone <- err
					return
				}

				// Record command if it looks like a complete command
				if terminalID != "" && len(data) > 0 && data[len(data)-1] == '\n' {
					command := string(data[:len(data)-1])
					if err := h.store.RecordCommand(terminalID, command); err != nil {
						log.Printf("failed to record command: %v", err)
					}
				}
			}
		}()
	}

	// Wait for errors or completion (only if terminal was started)
	if ptyFile != nil && inputDone != nil && outputDone != nil {
		select {
		case err := <-outputDone:
			if err != nil && err != io.EOF {
				log.Printf("output error: %v", err)
				errMsg := WebSocketMessage{
					Type:  "terminal.error",
					Error: fmt.Sprintf("output error: %v", err),
				}
				c.WriteJSON(errMsg)
			}
		case err := <-inputDone:
			if err != nil {
				log.Printf("input error: %v", err)
			}
		case <-time.After(24 * time.Hour): // Maximum session duration
			if terminalID != "" {
				h.ptyManager.KillTerminal(terminalID)
			}
		}
	}

	// Cleanup
	if terminalID != "" {
		h.mu.Lock()
		delete(h.connections, terminalID)
		h.mu.Unlock()

		// NOTE: We do NOT mark the terminal as ended on WebSocket disconnect
		// This allows the user to switch tabs and reconnect to the same terminal
		// The terminal is only marked as ended on explicit close or timeout
		// (handled in the "terminal.close" case and the 24-hour timeout)
	}

	return nil
}


// ListTerminals returns all terminals for an agent session.
func (h *TerminalHandler) ListTerminals(agentSessionID string) []*TerminalSession {
	return h.ptyManager.ListSessionsByAgent(agentSessionID)
}

// KillTerminal kills a terminal session.
func (h *TerminalHandler) KillTerminal(terminalID string) error {
	return h.ptyManager.KillTerminal(terminalID)
}

// CloseAll gracefully closes all active WebSocket connections and terminal sessions.
func (h *TerminalHandler) CloseAll() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close all WebSocket connections
	for terminalID, conn := range h.connections {
		if conn != nil {
			conn.Close()
		}
		log.Printf("Closed WebSocket connection for terminal: %s", terminalID)
	}

	// Clear maps
	h.connections = make(map[string]*websocket.Conn)
	h.writeMutex = make(map[string]*sync.Mutex)
	h.hasOutputGoroutine = make(map[string]bool)

	return nil
}
