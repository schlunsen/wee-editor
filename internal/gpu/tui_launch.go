package gpu

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// tierInfo describes a GPU tier shown in the launch flow.
type tierInfo struct {
	Name  string
	Label string
	GPU   string
	VRAM  string
	Cost  string
	Color lipgloss.Color
	Emoji string
}

var gpuTiers = []tierInfo{
	{Name: "starter", Label: "Starter", GPU: "RTX 4090", VRAM: "24GB", Cost: "$0.39/hr", Color: lipgloss.Color("42"), Emoji: "⚡"},
	{Name: "pro", Label: "Pro", GPU: "A100 80GB", VRAM: "80GB", Cost: "$1.64/hr", Color: lipgloss.Color("33"), Emoji: "🔥"},
	{Name: "beast", Label: "Beast", GPU: "H100 80GB", VRAM: "80GB", Cost: "$3.49/hr", Color: lipgloss.Color("196"), Emoji: "🚀"},
}

// viewLaunchPanel dispatches to the right sub-view based on launchStep.
func (m model) viewLaunchPanel() string {
	switch m.launchStep {
	case 0:
		return m.viewSelectTier()
	case 1:
		return m.viewSelectTemplate()
	case 2:
		return m.viewConfigStorage()
	case 3:
		return m.viewLaunching()
	default:
		return ""
	}
}

// viewSelectTier renders the GPU tier selection screen.
func (m model) viewSelectTier() string {
	var s string
	s += titleStyle.Render("Select GPU Tier") + "\n"
	s += subtitleStyle.Render("Choose your compute power") + "\n\n"

	if m.mountVolume != nil {
		lockStyle := lipgloss.NewStyle().Foreground(catYellow).Bold(true)
		s += lockStyle.Render("🔒 Data center locked: ") +
			valueStyle.Render(m.mountVolume.DataCenterID) +
			dimStyle.Render(fmt.Sprintf("  (volume: %s)", m.mountVolume.Name)) + "\n\n"
	}

	for i, tier := range gpuTiers {
		tierColor := lipgloss.NewStyle().Foreground(tier.Color).Bold(true)

		line1 := tierColor.Render(fmt.Sprintf(" %s %-8s", tier.Emoji, tier.Label)) +
			dimStyle.Render(" │ ") +
			valueStyle.Render(fmt.Sprintf("%-10s", tier.GPU)) +
			dimStyle.Render(tier.VRAM)
		line2 := strings.Repeat(" ", 3) +
			costStyle.Render(fmt.Sprintf("~%s", tier.Cost))

		content := line1 + "\n" + line2

		box := tierBoxStyle.Copy()
		if i == m.tierCursor {
			box = box.BorderForeground(tier.Color)
			cursor := tierColor.Render("▸ ")
			s += cursor + box.Render(content) + "\n"
		} else {
			box = box.BorderForeground(lipgloss.Color("236"))
			s += "  " + box.Render(content) + "\n"
		}
	}

	s += "\n" + helpStyle.Render("↑/↓ navigate  enter select  esc back  q quit")
	return s
}

// viewSelectTemplate renders the template selection screen.
func (m model) viewSelectTemplate() string {
	var s string

	tierColor := lipgloss.NewStyle().Foreground(m.selectedTier.Color).Bold(true)
	s += titleStyle.Render("Select Template") + "  "
	s += tierColor.Render(fmt.Sprintf("%s %s", m.selectedTier.Emoji, m.selectedTier.Label))
	s += dimStyle.Render(fmt.Sprintf(" (%s)", m.selectedTier.GPU)) + "\n\n"

	if len(m.templates) == 0 {
		s += dimStyle.Render("No templates available") + "\n"
		s += helpStyle.Render("esc back  q quit")
		return s
	}

	for i, tmpl := range m.templates {
		name := tmpl.Name
		image := ""
		if tmpl.ImageName != "" {
			img := tmpl.ImageName
			if len(img) > 40 {
				img = img[:37] + "..."
			}
			image = dimStyle.Render("  " + img)
		}

		if i == m.templateCursor {
			cursor := lipgloss.NewStyle().Foreground(catCyan).Bold(true).Render("▸ ")
			s += cursor + selectedStyle.Render(name) + image + "\n"
		} else {
			s += "  " + normalStyle.Render(name) + image + "\n"
		}
	}

	s += "\n" + helpStyle.Render("↑/↓ navigate  enter select  esc back  q quit")
	return s
}

// viewConfigStorage renders the storage configuration screen.
func (m model) viewConfigStorage() string {
	var s string

	tierColor := lipgloss.NewStyle().Foreground(m.selectedTier.Color).Bold(true)
	s += titleStyle.Render("Storage Configuration") + "  "
	s += tierColor.Render(fmt.Sprintf("%s %s", m.selectedTier.Emoji, m.selectedTier.Label)) + "\n"
	s += subtitleStyle.Render("Customize disk sizes (or press Enter to use defaults)") + "\n\n"

	tierDefaults := map[string][2]int{
		"starter": {20, 20},
		"pro":     {50, 40},
		"beast":   {100, 50},
	}
	defaults := tierDefaults[m.selectedTier.Name]
	volDefault, diskDefault := defaults[0], defaults[1]

	hasNetworkVolume := m.mountVolume != nil

	// Build the list of configurable items.
	type storageItem struct {
		label string
		value string
		help  string
	}
	var items []storageItem

	if hasNetworkVolume {
		// Network volume replaces the volume field; show it as informational.
		volInfo := fmt.Sprintf("%s (%d GB, %s)", m.mountVolume.Name, m.mountVolume.SizeGB, m.mountVolume.DataCenterID)
		s += labelStyle.Render("Network Volume  ") + valueStyle.Render(volInfo) + "\n"
		s += strings.Repeat(" ", 4) + dimStyle.Render("Mounted from volumes screen") + "\n\n"

		// Only show container disk and launch button.
		diskDisplay := fmt.Sprintf("%d GB (default)", diskDefault)
		if m.containerDiskGB > 0 {
			diskDisplay = fmt.Sprintf("%d GB", m.containerDiskGB)
		}
		items = []storageItem{
			{"Container Disk", diskDisplay, "Temporary, lost on terminate"},
			{"▶ Launch with these settings", "", ""},
		}
	} else {
		volDisplay := fmt.Sprintf("%d GB (default)", volDefault)
		if m.volumeGB > 0 {
			volDisplay = fmt.Sprintf("%d GB", m.volumeGB)
		}
		diskDisplay := fmt.Sprintf("%d GB (default)", diskDefault)
		if m.containerDiskGB > 0 {
			diskDisplay = fmt.Sprintf("%d GB", m.containerDiskGB)
		}
		items = []storageItem{
			{"Volume (persistent)", volDisplay, "Persists across stop/resume"},
			{"Container Disk", diskDisplay, "Temporary, lost on terminate"},
			{"▶ Launch with these settings", "", ""},
		}
	}

	for i, item := range items {
		isLaunchButton := i == len(items)-1
		prefix := "  "
		style := normalStyle
		if i == m.storageCursor {
			prefix = lipgloss.NewStyle().Foreground(catCyan).Bold(true).Render("▸ ")
			style = selectedStyle
		}
		if !isLaunchButton {
			s += prefix + style.Render(fmt.Sprintf("%-22s", item.label)) + "  " + valueStyle.Render(item.value)
			if i == m.storageCursor {
				s += dimStyle.Render("  ← type number")
			}
			s += "\n" + strings.Repeat(" ", 4) + dimStyle.Render(item.help) + "\n\n"
		} else {
			s += prefix + successStyle.Render(item.label) + "\n"
		}
	}

	s += "\n" + helpStyle.Render("↑/↓ navigate  0-9 set size  enter confirm  esc back")
	return s
}

// viewLaunching renders the launching spinner screen.
func (m model) viewLaunching() string {
	var s string

	spinner := lipgloss.NewStyle().Foreground(catPurple).Bold(true).Render("◌ ")
	s += spinner + titleStyle.Render("Launching GPU Pod...") + "\n\n"

	if m.selectedTier.Label != "" {
		tierColor := lipgloss.NewStyle().Foreground(m.selectedTier.Color).Bold(true)
		s += labelStyle.Render("Tier      ") + tierColor.Render(fmt.Sprintf("%s %s", m.selectedTier.Emoji, m.selectedTier.Label)) + "\n"
		s += labelStyle.Render("GPU       ") + valueStyle.Render(m.selectedTier.GPU) + "\n"
	}
	if m.selectedTempl.Name != "" {
		s += labelStyle.Render("Template  ") + valueStyle.Render(m.selectedTempl.Name) + "\n"
	}
	if m.mountVolume != nil {
		s += labelStyle.Render("Volume    ") + valueStyle.Render(m.mountVolume.Name) + "\n"
	}
	s += "\n" + dimStyle.Render("Generating SSH key and deploying pod...")
	return s
}

// updateLaunch handles all key input for the launch flow and returns the
// updated model and any command to execute.
func (m model) updateLaunch(msg tea.KeyMsg) (model, tea.Cmd) {
	// Storage step has its own dedicated handler.
	if m.launchStep == 2 {
		return m.updateStorage(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc":
		switch m.launchStep {
		case 0:
			// At tier selection, go back to dashboard.
			m.activeScreen = screenDashboard
		case 1:
			m.launchStep = 0
			m.tierCursor = 0
		}
		return m, nil

	case "up", "k":
		switch m.launchStep {
		case 0:
			if m.tierCursor > 0 {
				m.tierCursor--
			}
		case 1:
			if m.templateCursor > 0 {
				m.templateCursor--
			}
		}

	case "down", "j":
		switch m.launchStep {
		case 0:
			if m.tierCursor < len(gpuTiers)-1 {
				m.tierCursor++
			}
		case 1:
			if m.templateCursor < len(m.templates)-1 {
				m.templateCursor++
			}
		}

	case "enter":
		switch m.launchStep {
		case 0:
			// Tier selected -- load templates.
			if m.tierCursor < len(gpuTiers) {
				m.selectedTier = gpuTiers[m.tierCursor]
				m.launchStep = 3 // show launching while templates load
				return m, m.loadTemplates()
			}
		case 1:
			// Template selected -- go to storage config.
			if m.templateCursor < len(m.templates) {
				m.selectedTempl = m.templates[m.templateCursor]
				m.volumeGB = 0
				m.containerDiskGB = 0
				m.storageInput = ""
				m.storageCursor = 0
				m.launchStep = 2
			}
		}
	}

	return m, nil
}

// updateStorage handles key input for the storage configuration step.
func (m model) updateStorage(msg tea.KeyMsg) (model, tea.Cmd) {
	hasNetworkVolume := m.mountVolume != nil
	// When a network volume is mounted the volume field is removed, so there
	// are only 2 items (container disk + launch button) instead of 3.
	maxCursor := 2
	if hasNetworkVolume {
		maxCursor = 1
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		m.launchStep = 1
		m.templateCursor = 0
		return m, nil
	case "up", "k":
		if m.storageCursor > 0 {
			m.storageCursor--
			m.storageInput = ""
		}
	case "down", "j":
		if m.storageCursor < maxCursor {
			m.storageCursor++
			m.storageInput = ""
		}
	case "enter":
		if m.storageCursor == maxCursor {
			m.launchStep = 3
			return m, m.launchPod()
		}
	case "backspace":
		if len(m.storageInput) > 0 {
			m.storageInput = m.storageInput[:len(m.storageInput)-1]
			m.applyStorageInput()
		}
	default:
		if len(msg.String()) == 1 && msg.String()[0] >= '0' && msg.String()[0] <= '9' {
			m.storageInput += msg.String()
			m.applyStorageInput()
		}
	}
	return m, nil
}

// applyStorageInput parses the current storageInput buffer and updates the
// corresponding storage field.
func (m *model) applyStorageInput() {
	val := 0
	for _, c := range m.storageInput {
		val = val*10 + int(c-'0')
	}
	if val > 10000 {
		val = 10000
	}

	hasNetworkVolume := m.mountVolume != nil
	if hasNetworkVolume {
		// With a network volume mounted, cursor 0 is container disk.
		if m.storageCursor == 0 {
			m.containerDiskGB = val
		}
	} else {
		if m.storageCursor == 0 {
			m.volumeGB = val
		} else if m.storageCursor == 1 {
			m.containerDiskGB = val
		}
	}
}

// loadTemplates returns a tea.Cmd that asynchronously fetches available templates.
func (m model) loadTemplates() tea.Cmd {
	return func() tea.Msg {
		templates, err := m.manager.ListTemplates()
		return templatesLoadedMsg{templates: templates, err: err}
	}
}

// launchPod returns a tea.Cmd that asynchronously launches a GPU pod.
// If a network volume is mounted, its ID is passed to the launch call.
func (m model) launchPod() tea.Cmd {
	return func() tea.Msg {
		pubKey, err := EnsureSSHKey(m.manager.SSHKeyPath())
		if err != nil {
			return launchDoneMsg{err: fmt.Errorf("SSH key error: %w", err)}
		}

		volumeGB := m.volumeGB
		if m.mountVolume != nil {
			// Network volume replaces the regular volume; use its size.
			volumeGB = m.mountVolume.SizeGB
		}

		session, err := m.manager.Launch(
			m.selectedTier.Name,
			m.selectedTempl.ID,
			pubKey,
			volumeGB,
			m.containerDiskGB,
		)
		return launchDoneMsg{session: session, err: err}
	}
}
