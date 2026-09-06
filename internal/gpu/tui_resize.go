package gpu

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// resizeDoneMsg is sent when the async resize operation completes.
type resizeDoneMsg struct {
	err error
}

// viewResize renders the resize storage panel.
func (m model) viewResize() string {
	podID := ""
	if m.dashSession != nil {
		podID = m.dashSession.PodID
	}

	s := renderLogo() + "\n\n"
	s += renderDivider(0) + "\n\n"
	s += titleStyle.Render(fmt.Sprintf("Resize Storage — %s", podID)) + "\n\n"

	// Current sizes from session (default to 0 if nil)
	curVol := 0
	curDisk := 0
	if m.dashSession != nil {
		if m.dashSession.VolumeGB != nil {
			curVol = *m.dashSession.VolumeGB
		}
		if m.dashSession.ContainerDiskGB != nil {
			curDisk = *m.dashSession.ContainerDiskGB
		}
	}

	// Styles for the fields
	fieldActive := lipgloss.NewStyle().Foreground(catCyan).Bold(true)
	fieldNormal := lipgloss.NewStyle().Foreground(catWhite)
	inputStyle := lipgloss.NewStyle().Foreground(catYellow).Bold(true).Underline(true)
	warningStyle := lipgloss.NewStyle().Foreground(catYellow)

	// Volume field
	volLabel := "  Volume (persistent):  "
	volValue := fmt.Sprintf("Current: %d GB", curVol)
	volNew := ""
	if m.resizeVolGB > 0 {
		volNew = fmt.Sprintf(" → %d GB", m.resizeVolGB)
	} else if m.resizeCursor == 0 && m.resizeInput != "" {
		volNew = fmt.Sprintf(" → %s GB", m.resizeInput)
	} else if m.resizeCursor == 0 {
		volNew = " → __ GB"
	}

	if m.resizeCursor == 0 {
		cursor := " ← type number"
		if m.resizeInput != "" {
			s += fieldActive.Render("▸"+volLabel) + valueStyle.Render(volValue) + inputStyle.Render(volNew) + dimStyle.Render(cursor) + "\n"
		} else {
			s += fieldActive.Render("▸"+volLabel) + valueStyle.Render(volValue) + dimStyle.Render(volNew) + dimStyle.Render(cursor) + "\n"
		}
	} else {
		if m.resizeVolGB > 0 {
			s += fieldNormal.Render(" "+volLabel) + valueStyle.Render(volValue) + inputStyle.Render(volNew) + "\n"
		} else {
			s += fieldNormal.Render(" "+volLabel) + dimStyle.Render(volValue) + "\n"
		}
	}

	s += "\n"

	// Container disk field
	diskLabel := "  Container Disk:       "
	diskValue := fmt.Sprintf("Current: %d GB", curDisk)
	diskNew := ""
	if m.resizeDiskGB > 0 {
		diskNew = fmt.Sprintf(" → %d GB", m.resizeDiskGB)
	} else if m.resizeCursor == 1 && m.resizeInput != "" {
		diskNew = fmt.Sprintf(" → %s GB", m.resizeInput)
	} else if m.resizeCursor == 1 {
		diskNew = " → __ GB"
	}

	if m.resizeCursor == 1 {
		cursor := " ← type number"
		if m.resizeInput != "" {
			s += fieldActive.Render("▸"+diskLabel) + valueStyle.Render(diskValue) + inputStyle.Render(diskNew) + dimStyle.Render(cursor) + "\n"
		} else {
			s += fieldActive.Render("▸"+diskLabel) + valueStyle.Render(diskValue) + dimStyle.Render(diskNew) + dimStyle.Render(cursor) + "\n"
		}
	} else {
		if m.resizeDiskGB > 0 {
			s += fieldNormal.Render(" "+diskLabel) + valueStyle.Render(diskValue) + inputStyle.Render(diskNew) + "\n"
		} else {
			s += fieldNormal.Render(" "+diskLabel) + dimStyle.Render(diskValue) + "\n"
		}
	}
	s += warningStyle.Render("                         ⚠ Changing this will restart your pod") + "\n"

	s += "\n"

	// Apply button
	if m.resizeCursor == 2 {
		s += selectedStyle.Render("  [ Apply Changes ]  ") + "\n"
	} else {
		s += normalStyle.Render("  [ Apply Changes ]  ") + "\n"
	}

	s += "\n"

	// Error message
	if m.resizeError != "" {
		s += errorStyle.Render("  ✗ "+m.resizeError) + "\n\n"
	}

	// Confirmation dialog
	if m.resizeConfirm {
		confirmBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(catYellow).
			Padding(0, 2).
			Foreground(catYellow)
		s += confirmBox.Render("⚠ This will restart your pod. Continue? [enter confirm / esc cancel]") + "\n\n"
	}

	// Footer
	s += dimStyle.Render("  Sizes can only increase, never shrink.") + "\n\n"
	s += helpStyle.Render("  ↑/↓ navigate  0-9 set size  enter apply  esc back")

	return s
}

// updateResize handles key input for the resize panel.
func (m model) updateResize(msg tea.KeyMsg) (model, tea.Cmd) {
	key := msg.String()

	// Confirmation mode
	if m.resizeConfirm {
		switch key {
		case "enter":
			return m, m.doResize()
		case "esc":
			m.resizeConfirm = false
			return m, nil
		}
		return m, nil
	}

	// Normal mode
	switch key {
	case "up":
		if m.resizeCursor > 0 {
			m.resizeInput = ""
			m.resizeCursor--
		}
	case "down":
		if m.resizeCursor < 2 {
			m.resizeInput = ""
			m.resizeCursor++
		}
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if m.resizeCursor < 2 {
			m.resizeInput += key
			m.applyResizeInput()
		}
	case "backspace":
		if len(m.resizeInput) > 0 {
			m.resizeInput = m.resizeInput[:len(m.resizeInput)-1]
			m.applyResizeInput()
		}
	case "enter":
		if m.resizeCursor == 2 {
			// Validate
			if errMsg := m.validateResize(); errMsg != "" {
				m.resizeError = errMsg
				return m, nil
			}
			m.resizeError = ""

			// If disk changed, show confirmation
			curDisk := 0
			if m.dashSession != nil && m.dashSession.ContainerDiskGB != nil {
				curDisk = *m.dashSession.ContainerDiskGB
			}
			if m.resizeDiskGB > 0 && m.resizeDiskGB != curDisk {
				m.resizeConfirm = true
				return m, nil
			}

			// Volume-only change: execute immediately
			return m, m.doResize()
		}
	case "esc":
		m.activeScreen = screenDashboard
		m.resizeCursor = 0
		m.resizeVolGB = 0
		m.resizeDiskGB = 0
		m.resizeInput = ""
		m.resizeConfirm = false
		m.resizeError = ""
		return m, nil
	}

	return m, nil
}

// applyResizeInput parses the current input and sets the appropriate resize field.
func (m *model) applyResizeInput() {
	if m.resizeInput == "" {
		if m.resizeCursor == 0 {
			m.resizeVolGB = 0
		} else if m.resizeCursor == 1 {
			m.resizeDiskGB = 0
		}
		return
	}

	val, err := strconv.Atoi(m.resizeInput)
	if err != nil {
		return
	}

	// Cap at 10000
	if val > 10000 {
		val = 10000
		m.resizeInput = "10000"
	}

	switch m.resizeCursor {
	case 0:
		m.resizeVolGB = val
	case 1:
		m.resizeDiskGB = val
	}
}

// validateResize checks that the new sizes are valid.
func (m *model) validateResize() string {
	curVol := 0
	curDisk := 0
	if m.dashSession != nil {
		if m.dashSession.VolumeGB != nil {
			curVol = *m.dashSession.VolumeGB
		}
		if m.dashSession.ContainerDiskGB != nil {
			curDisk = *m.dashSession.ContainerDiskGB
		}
	}

	if m.resizeVolGB > 0 && m.resizeVolGB < curVol {
		return fmt.Sprintf("Volume must be ≥ %d GB", curVol)
	}
	if m.resizeDiskGB > 0 && m.resizeDiskGB < curDisk {
		return fmt.Sprintf("Disk must be ≥ %d GB", curDisk)
	}
	if m.resizeVolGB == 0 && m.resizeDiskGB == 0 {
		return "No changes to apply"
	}

	return ""
}

// doResize returns a tea.Cmd that calls the manager to resize the pod.
func (m model) doResize() tea.Cmd {
	sessionID := 0
	if m.dashSession != nil {
		sessionID = m.dashSession.ID
	}
	volGB := m.resizeVolGB
	diskGB := m.resizeDiskGB
	mgr := m.manager

	return func() tea.Msg {
		err := mgr.ResizePod(sessionID, volGB, diskGB)
		return resizeDoneMsg{err: err}
	}
}
