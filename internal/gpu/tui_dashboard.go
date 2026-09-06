package gpu

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Async Messages ─────────────────────────────────────────────────

type statusRefreshMsg struct {
	session    *GPUSession
	otherCount int
	err        error
}

type podActionDoneMsg struct {
	action string // "stop", "resume", "kill"
	err    error
}

type tickMsg struct{}

// ── Dashboard View ─────────────────────────────────────────────────

func (m model) viewDashboard() string {
	if m.dashLoading {
		frame := spinnerDots[int(time.Now().UnixMilli()/120)%len(spinnerDots)]
		return dimStyle.Render(frame+" Loading pod status...")
	}

	if m.dashError != nil {
		return errorStyle.Render("✗ Error fetching status") + "\n\n" +
			lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(m.dashError.Error()) + "\n\n" +
			helpStyle.Render("r refresh  ·  n new pod  ·  q quit")
	}

	if m.dashSession == nil {
		return dimStyle.Render("No active GPU pods") + "\n\n" +
			helpStyle.Render("press n to launch a new pod")
	}

	s := m.dashSession
	var out string

	// Header: status dot + pod ID + status
	dot := statusDot(s.Status)
	podID := valueStyle.Render(s.PodID)
	statusLabel := lipgloss.NewStyle().Foreground(statusColor(s.Status)).Bold(true).Render(s.Status)
	out += dot + " " + podID + "    " + statusLabel + "\n\n"

	// Info card
	var rows []string

	// GPU
	gpuVal := s.GPUType
	if s.Tier != "" {
		gpuVal += " (" + s.Tier + ")"
	}
	rows = append(rows, labelStyle.Render("GPU       ") + valueStyle.Render(gpuVal))

	// Template
	if s.TemplateName != "" {
		rows = append(rows, labelStyle.Render("Template  ") + valueStyle.Render(s.TemplateName))
	}

	// Uptime
	if s.Live != nil {
		rows = append(rows, labelStyle.Render("Uptime    ") + valueStyle.Render(formatUptime(s.Live.UptimeSeconds)))
	}

	// Cost
	if s.Live != nil {
		costStr := fmt.Sprintf("$%.2f total ($%.2f/hr)", s.Live.TotalCost, s.Live.CostPerHr)
		rows = append(rows, labelStyle.Render("Cost      ") + costStyle.Render(costStr))
	}

	// Volume & Disk
	volStr := ""
	if s.VolumeGB != nil {
		volStr += fmt.Sprintf("%d GB", *s.VolumeGB)
	}
	diskStr := ""
	if s.ContainerDiskGB != nil {
		diskStr += fmt.Sprintf("%d GB", *s.ContainerDiskGB)
	}
	if volStr != "" || diskStr != "" {
		storageRow := labelStyle.Render("Volume    ") + valueStyle.Render(volStr)
		if diskStr != "" {
			storageRow += dimStyle.Render("  |  ") + labelStyle.Render("Disk  ") + valueStyle.Render(diskStr)
		}
		rows = append(rows, storageRow)
	}

	// SSH
	if s.SSHHost != "" && s.SSHPort > 0 {
		sshVal := fmt.Sprintf("root@%s -p %d", s.SSHHost, s.SSHPort)
		rows = append(rows, labelStyle.Render("SSH       ") + valueStyle.Render(sshVal))
	}

	// Error message
	if s.ErrorMessage != "" {
		rows = append(rows, labelStyle.Render("Error     ") + errorStyle.Render(s.ErrorMessage))
	}

	cardContent := ""
	for i, row := range rows {
		cardContent += row
		if i < len(rows)-1 {
			cardContent += "\n"
		}
	}
	out += boxStyle.Render(cardContent) + "\n"

	// Other sessions count
	if m.dashOtherCount > 0 {
		out += "\n" + dimStyle.Render(fmt.Sprintf("+%d other session", m.dashOtherCount))
		if m.dashOtherCount > 1 {
			out += dimStyle.Render("s")
		}
		out += "\n"
	}

	// Help bar
	out += "\n" + helpStyle.Render("r refresh  ·  s stop  ·  k kill  ·  n new pod  ·  q quit")

	return out
}

// ── Commands ───────────────────────────────────────────────────────

func (m model) fetchStatus() tea.Cmd {
	return func() tea.Msg {
		sessions, err := m.manager.ListSessions()
		if err != nil {
			return statusRefreshMsg{err: err}
		}

		if len(sessions) == 0 {
			return statusRefreshMsg{}
		}

		// Find the most relevant session by priority:
		// running > creating > stopped > exited > error
		priorityOrder := map[string]int{
			"running":  0,
			"creating": 1,
			"stopped":  2,
			"exited":   3,
			"error":    4,
		}

		bestIdx := 0
		bestPriority := 99
		for i, s := range sessions {
			p, ok := priorityOrder[s.Status]
			if !ok {
				p = 5
			}
			if p < bestPriority {
				bestPriority = p
				bestIdx = i
			}
		}

		best := sessions[bestIdx]
		otherCount := len(sessions) - 1

		return statusRefreshMsg{
			session:    &best,
			otherCount: otherCount,
		}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m model) doPodAction(action string) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch action {
		case "stop":
			err = m.manager.Stop()
		case "resume":
			err = m.manager.Resume()
		case "kill":
			err = m.manager.Kill()
		}
		return podActionDoneMsg{action: action, err: err}
	}
}

// ── Helpers ────────────────────────────────────────────────────────

func statusDot(status string) string {
	switch status {
	case "running":
		return statusRunning
	case "creating":
		return statusCreating
	case "stopped", "exited":
		return statusStopped
	case "error":
		return statusError
	default:
		return dimStyle.Render("●")
	}
}

func statusColor(status string) lipgloss.Color {
	switch status {
	case "running":
		return catGreen
	case "creating":
		return catYellow
	case "stopped", "exited":
		return catRed
	case "error":
		return catRed
	default:
		return catDim
	}
}

func formatUptime(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d sec", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%d min", seconds/60)
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	return fmt.Sprintf("%dh %dm", h, m)
}
