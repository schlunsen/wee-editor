package gpu

import (
	"fmt"
	"sort"
	"strconv"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Async Messages ─────────────────────────────────────────────────

type volumesLoadedMsg struct {
	volumes []NetworkVolume
	err     error
}

type dataCentersLoadedMsg struct {
	dataCenters []DataCenter
	err         error
}

type volumeCreatedMsg struct {
	volume *NetworkVolume
	err    error
}

type volumeDeletedMsg struct {
	err error
}

// ── Volume List View ───────────────────────────────────────────────

func (m model) viewVolumes() string {
	if m.volumesLoading {
		frame := spinnerDots[int(time.Now().UnixMilli()/120)%len(spinnerDots)]
		return dimStyle.Render(frame + " Loading network volumes...")
	}

	if m.volumesError != nil {
		return errorStyle.Render("✗ Error loading volumes") + "\n\n" +
			lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(m.volumesError.Error()) + "\n\n" +
			helpStyle.Render("r refresh  ·  esc back  ·  q quit")
	}

	if m.volumeCreateMode {
		return m.viewCreateVolume()
	}

	if m.volumeDeleteConfirm {
		return m.viewDeleteConfirm()
	}

	var s string
	s += titleStyle.Render("Network Volumes") + "\n\n"

	if len(m.volumes) == 0 {
		s += dimStyle.Render("No network volumes yet") + "\n\n"
		s += helpStyle.Render("c create  ·  esc back  ·  q quit")
		return s
	}

	grouped := groupVolumesByDC(m.volumes)
	keys := sortedDCKeys(grouped)

	// Build a flat list of (dcHeader, volume) entries to map cursor positions
	flatIdx := 0
	for _, dc := range keys {
		vols := grouped[dc]

		// Data center header
		s += dimStyle.Render(dc) + "\n"

		for _, vol := range vols {
			isSelected := flatIdx == m.volumeCursor

			prefix := "  "
			if isSelected {
				prefix = lipgloss.NewStyle().Foreground(catCyan).Bold(true).Render("▸ ")
			}

			name := valueStyle.Render(vol.Name)
			size := costStyle.Render(fmt.Sprintf("%d GB", vol.SizeGB))

			var status string
			if vol.MountedPodID != "" {
				status = lipgloss.NewStyle().Foreground(catGreen).Render("●") + " " +
					lipgloss.NewStyle().Foreground(catGreen).Render("mounted")
			} else {
				status = dimStyle.Render("○") + " " + dimStyle.Render("free")
			}

			line := fmt.Sprintf("%s%-18s %8s   %s", prefix, name, size, status)
			if isSelected {
				line = selectedStyle.Render(
					fmt.Sprintf("▸ %-18s %8s   %s",
						vol.Name,
						fmt.Sprintf("%d GB", vol.SizeGB),
						statusLabel(vol.MountedPodID),
					),
				)
			}

			s += line + "\n"
			flatIdx++
		}
		s += "\n"
	}

	s += helpStyle.Render("↑/↓ navigate  ·  c create  ·  d delete  ·  m mount  ·  esc back")
	return s
}

func statusLabel(mountedPodID string) string {
	if mountedPodID != "" {
		return "● mounted"
	}
	return "○ free"
}

func (m model) viewDeleteConfirm() string {
	var s string
	s += titleStyle.Render("Delete Network Volume") + "\n\n"

	if m.volumeCursor >= 0 && m.volumeCursor < len(m.volumes) {
		vol := m.volumeAtCursor()
		if vol != nil {
			s += labelStyle.Render("Volume: ") + valueStyle.Render(vol.Name) + "\n"
			s += labelStyle.Render("Size:   ") + costStyle.Render(fmt.Sprintf("%d GB", vol.SizeGB)) + "\n"
			s += labelStyle.Render("DC:     ") + valueStyle.Render(vol.DataCenterID) + "\n\n"
		}
	}

	s += errorStyle.Render("Are you sure you want to delete this volume?") + "\n"
	s += errorStyle.Render("This action cannot be undone.") + "\n\n"
	s += helpStyle.Render("y confirm  ·  n/esc cancel")
	return s
}

// ── Create Volume Form ─────────────────────────────────────────────

func (m model) viewCreateVolume() string {
	var s string
	s += titleStyle.Render("Create Network Volume") + "\n\n"

	// Name field
	nameLabel := labelStyle.Render("Name          ")
	nameValue := m.newVolumeName
	if nameValue == "" {
		nameValue = "________"
	}
	if m.volumeCreateStep == 0 {
		nameLabel = lipgloss.NewStyle().Foreground(catCyan).Bold(true).Render("Name          ")
		nameValue = lipgloss.NewStyle().Foreground(catWhite).Bold(true).
			Background(lipgloss.Color("57")).Padding(0, 1).Render(m.newVolumeName + "_")
	} else {
		nameValue = valueStyle.Render(nameValue)
	}
	s += nameLabel + nameValue + "\n\n"

	// Size field
	sizeLabel := labelStyle.Render("Size (GB)     ")
	sizeValue := m.newVolumeSizeGB
	if sizeValue == "" {
		sizeValue = "__"
	}
	if m.volumeCreateStep == 1 {
		sizeLabel = lipgloss.NewStyle().Foreground(catCyan).Bold(true).Render("Size (GB)     ")
		sizeValue = lipgloss.NewStyle().Foreground(catWhite).Bold(true).
			Background(lipgloss.Color("57")).Padding(0, 1).Render(m.newVolumeSizeGB+"_") +
			dimStyle.Render(" GB")
	} else {
		sizeValue = valueStyle.Render(sizeValue) + dimStyle.Render(" GB")
	}
	s += sizeLabel + sizeValue + "\n\n"

	// Data center picker
	dcLabel := labelStyle.Render("Data Center   ")
	if m.volumeCreateStep == 2 {
		dcLabel = lipgloss.NewStyle().Foreground(catCyan).Bold(true).Render("Data Center   ")
	}
	s += dcLabel

	if len(m.dataCenters) == 0 {
		s += dimStyle.Render("(loading...)") + "\n"
	} else {
		s += "\n"
		// Show a scrollable window of max 8 data centers
		maxVisible := 8
		total := len(m.dataCenters)
		start := 0
		if total > maxVisible {
			start = m.dcCursor - maxVisible/2
			if start < 0 {
				start = 0
			}
			if start+maxVisible > total {
				start = total - maxVisible
			}
		}
		end := start + maxVisible
		if end > total {
			end = total
		}

		if start > 0 {
			s += "              " + dimStyle.Render(fmt.Sprintf("  ↑ %d more", start)) + "\n"
		}

		for i := start; i < end; i++ {
			dc := m.dataCenters[i]
			prefix := "  "
			dcDisplay := dimStyle.Render(dc.ID)
			if dc.Name != "" && dc.Name != dc.ID {
				dcDisplay += dimStyle.Render(" (" + dc.Name + ")")
			}

			if m.volumeCreateStep == 2 && i == m.dcCursor {
				prefix = lipgloss.NewStyle().Foreground(catCyan).Bold(true).Render("▸ ")
				dcDisplay = selectedStyle.Render(dc.ID)
				if dc.Name != "" && dc.Name != dc.ID {
					dcDisplay += " " + dimStyle.Render("(" + dc.Name + ")")
				}
			}
			s += "              " + prefix + dcDisplay + "\n"
		}

		if end < total {
			s += "              " + dimStyle.Render(fmt.Sprintf("  ↓ %d more", total-end)) + "\n"
		}
	}
	s += "\n"

	// Create button
	if m.volumeCreateStep == 3 {
		s += selectedStyle.Render("▶ Create") + "\n"
	} else {
		s += dimStyle.Render("  Create") + "\n"
	}

	s += "\n" + helpStyle.Render("↑/↓ fields  ·  enter select  ·  esc cancel")
	return s
}

// ── Key Handling ───────────────────────────────────────────────────

func (m model) updateVolumes(msg tea.KeyMsg) (model, tea.Cmd) {
	if m.volumeCreateMode {
		return m.updateCreateVolume(msg)
	}

	if m.volumeDeleteConfirm {
		switch msg.String() {
		case "y", "Y":
			m.volumeDeleteConfirm = false
			return m, m.deleteVolume()
		case "n", "N", "esc":
			m.volumeDeleteConfirm = false
			return m, nil
		}
		return m, nil
	}

	// Normal volume list navigation
	switch msg.String() {
	case "up", "k":
		if m.volumeCursor > 0 {
			m.volumeCursor--
		}
	case "down", "j":
		maxIdx := len(m.volumes) - 1
		if m.volumeCursor < maxIdx {
			m.volumeCursor++
		}
	case "esc":
		m.activeScreen = screenDashboard
		return m, nil
	case "c":
		m.volumeCreateMode = true
		m.volumeCreateStep = 0
		m.newVolumeName = ""
		m.newVolumeSizeGB = ""
		m.dcCursor = 0
		return m, m.loadDataCenters()
	case "d":
		if len(m.volumes) > 0 {
			vol := m.volumeAtCursor()
			if vol != nil && vol.MountedPodID == "" {
				m.volumeDeleteConfirm = true
			}
		}
	case "m":
		if len(m.volumes) > 0 {
			vol := m.volumeAtCursor()
			if vol != nil && vol.MountedPodID == "" {
				m.mountVolume = vol
				m.activeScreen = screenLaunch
				m.launchStep = 0
			}
		}
	case "r":
		m.volumesLoading = true
		return m, m.loadVolumes()
	}

	return m, nil
}

func (m model) updateCreateVolume(msg tea.KeyMsg) (model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		m.volumeCreateMode = false
		return m, nil

	case "up", "shift+tab":
		// When on data center picker, up/down navigate the DC list
		if m.volumeCreateStep == 2 && key == "up" {
			if m.dcCursor > 0 {
				m.dcCursor--
			} else {
				// At top of DC list, move to previous form field
				m.volumeCreateStep--
			}
			return m, nil
		}
		if m.volumeCreateStep > 0 {
			m.volumeCreateStep--
		}
		return m, nil

	case "down", "tab":
		if m.volumeCreateStep == 2 && key == "down" {
			if m.dcCursor < len(m.dataCenters)-1 {
				m.dcCursor++
			} else {
				// At bottom of DC list, move to next form field
				m.volumeCreateStep++
			}
			return m, nil
		}
		if m.volumeCreateStep < 3 {
			m.volumeCreateStep++
		}
		return m, nil

	case "enter":
		if m.volumeCreateStep < 3 {
			// Move to next field
			m.volumeCreateStep++
			return m, nil
		}
		// On create button: validate and submit
		return m.validateAndCreate()

	case "backspace":
		switch m.volumeCreateStep {
		case 0:
			if len(m.newVolumeName) > 0 {
				m.newVolumeName = m.newVolumeName[:len(m.newVolumeName)-1]
			}
		case 1:
			if len(m.newVolumeSizeGB) > 0 {
				m.newVolumeSizeGB = m.newVolumeSizeGB[:len(m.newVolumeSizeGB)-1]
			}
		}
		return m, nil

	default:
		// Handle text input for name and size fields
		if len(key) == 1 {
			r := rune(key[0])
			switch m.volumeCreateStep {
			case 0:
				if unicode.IsPrint(r) {
					m.newVolumeName += key
				}
			case 1:
				if r >= '0' && r <= '9' {
					m.newVolumeSizeGB += key
				}
			case 2:
				// On datacenter picker, handle up/down via j/k
				if key == "j" && m.dcCursor < len(m.dataCenters)-1 {
					m.dcCursor++
				} else if key == "k" && m.dcCursor > 0 {
					m.dcCursor--
				}
			}
		}
		return m, nil
	}
}

func (m model) validateAndCreate() (model, tea.Cmd) {
	if m.newVolumeName == "" {
		m.volumesError = fmt.Errorf("volume name is required")
		return m, nil
	}

	size, err := strconv.Atoi(m.newVolumeSizeGB)
	if err != nil || size <= 0 {
		m.volumesError = fmt.Errorf("volume size must be a positive number")
		return m, nil
	}

	if len(m.dataCenters) == 0 {
		m.volumesError = fmt.Errorf("no data center selected")
		return m, nil
	}

	if m.dcCursor < 0 || m.dcCursor >= len(m.dataCenters) {
		m.volumesError = fmt.Errorf("invalid data center selection")
		return m, nil
	}

	m.volumeCreateMode = false
	m.volumesLoading = true
	return m, m.createVolume()
}

// ── Async Commands ─────────────────────────────────────────────────

func (m model) loadVolumes() tea.Cmd {
	return func() tea.Msg {
		volumes, err := m.manager.ListNetworkVolumes()
		return volumesLoadedMsg{volumes: volumes, err: err}
	}
}

func (m model) loadDataCenters() tea.Cmd {
	return func() tea.Msg {
		dcs, err := m.manager.ListDataCenters()
		return dataCentersLoadedMsg{dataCenters: dcs, err: err}
	}
}

func (m model) createVolume() tea.Cmd {
	name := m.newVolumeName
	sizeStr := m.newVolumeSizeGB
	var dcID string
	if m.dcCursor >= 0 && m.dcCursor < len(m.dataCenters) {
		dcID = m.dataCenters[m.dcCursor].ID
	}

	return func() tea.Msg {
		size, err := strconv.Atoi(sizeStr)
		if err != nil {
			return volumeCreatedMsg{err: fmt.Errorf("invalid size: %s", sizeStr)}
		}
		vol, err := m.manager.CreateNetworkVolume(name, size, dcID)
		return volumeCreatedMsg{volume: vol, err: err}
	}
}

func (m model) deleteVolume() tea.Cmd {
	vol := m.volumeAtCursor()
	if vol == nil {
		return func() tea.Msg {
			return volumeDeletedMsg{err: fmt.Errorf("no volume selected")}
		}
	}
	volumeID := vol.ID

	return func() tea.Msg {
		err := m.manager.DeleteNetworkVolume(volumeID)
		return volumeDeletedMsg{err: err}
	}
}

// ── Helpers ────────────────────────────────────────────────────────

// volumeAtCursor returns the volume at the current cursor position
// using the flat index across all data center groups.
func (m model) volumeAtCursor() *NetworkVolume {
	if len(m.volumes) == 0 || m.volumeCursor < 0 {
		return nil
	}

	grouped := groupVolumesByDC(m.volumes)
	keys := sortedDCKeys(grouped)

	flatIdx := 0
	for _, dc := range keys {
		vols := grouped[dc]
		for i := range vols {
			if flatIdx == m.volumeCursor {
				return &vols[i]
			}
			flatIdx++
		}
	}
	return nil
}

// groupVolumesByDC groups volumes by their DataCenterID.
func groupVolumesByDC(volumes []NetworkVolume) map[string][]NetworkVolume {
	grouped := make(map[string][]NetworkVolume)
	for _, vol := range volumes {
		dc := vol.DataCenterID
		if dc == "" {
			dc = "Unknown"
		}
		grouped[dc] = append(grouped[dc], vol)
	}
	return grouped
}

// sortedDCKeys returns the data center keys in sorted order.
func sortedDCKeys(grouped map[string][]NetworkVolume) []string {
	keys := make([]string, 0, len(grouped))
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
