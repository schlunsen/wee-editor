package gpu

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenDashboard screen = iota
	screenLaunch
	screenVolumes
	screenResize
)

// Messages for async operations
type templatesLoadedMsg struct {
	templates []Template
	err       error
}

type launchDoneMsg struct {
	session *GPUSession
	err     error
}

// sidebarItem represents a clickable item in the sidebar.
type sidebarItem struct {
	label    string
	action   string // identifier for what this item does
	isHeader bool   // section header (not selectable)
	isSep    bool   // separator line
}

type model struct {
	manager      *Manager
	activeScreen screen

	// Sidebar
	sidebarFocus  bool
	sidebarCursor int
	sidebarItems  []sidebarItem

	// Launch flow state
	launchStep      int // 0=tier, 1=template, 2=storage, 3=launching
	tierCursor      int
	selectedTier    tierInfo
	templates       []Template
	templateCursor  int
	selectedTempl   Template
	volumeGB        int
	containerDiskGB int
	storageInput    string
	storageCursor   int // 0=volume, 1=disk, 2=launch button

	// Network volume to mount (set from volumes screen)
	mountVolume *NetworkVolume

	// Volumes state
	volumes             []NetworkVolume
	volumeCursor        int
	volumesLoading      bool
	volumesError        error
	volumeCreateMode    bool
	volumeCreateStep    int // 0=name, 1=size, 2=datacenter, 3=create button
	newVolumeName       string
	newVolumeSizeGB     string
	dataCenters         []DataCenter
	dcCursor            int
	volumeDeleteConfirm bool

	// Dashboard state
	dashLoading    bool
	dashSession    *GPUSession
	dashError      error
	dashOtherCount int

	// Resize state
	resizeCursor  int
	resizeInput   string
	resizeVolGB   int
	resizeDiskGB  int
	resizeError   string
	resizeConfirm bool

	// Legacy launch result (used for done/error in launch flow)
	session *GPUSession
	err     error

	width  int
	height int
}

func initialModel(m *Manager) model {
	mdl := model{
		manager:      m,
		activeScreen: screenDashboard,
		sidebarFocus: true,
		dashLoading:  true,
	}
	mdl.rebuildSidebar()
	return mdl
}

func (m model) Init() tea.Cmd {
	return m.fetchStatus()
}

// rebuildSidebar rebuilds the sidebar items based on current state.
func (m *model) rebuildSidebar() {
	items := []sidebarItem{
		{label: "Dashboard", action: "dashboard"},
		{label: "Volumes", action: "volumes"},
		{label: "Launch New", action: "launch"},
	}

	// Context-sensitive pod actions
	if m.dashSession != nil {
		items = append(items, sidebarItem{isSep: true})
		items = append(items, sidebarItem{label: "Pod Actions", isHeader: true})

		switch m.dashSession.Status {
		case "running":
			items = append(items,
				sidebarItem{label: "Stop", action: "stop"},
				sidebarItem{label: "Resize Disk", action: "resize"},
				sidebarItem{label: "Kill", action: "kill"},
			)
		case "stopped", "exited":
			items = append(items,
				sidebarItem{label: "Resume", action: "resume"},
				sidebarItem{label: "Kill", action: "kill"},
			)
		case "creating":
			items = append(items,
				sidebarItem{label: "Kill", action: "kill"},
			)
		case "error":
			items = append(items,
				sidebarItem{label: "Kill", action: "kill"},
			)
		}
	}

	// Volume actions when on volumes screen
	if m.activeScreen == screenVolumes && !m.volumeCreateMode {
		items = append(items, sidebarItem{isSep: true})
		items = append(items, sidebarItem{label: "Volume Actions", isHeader: true})
		items = append(items, sidebarItem{label: "Create New", action: "vol_create"})
		if len(m.volumes) > 0 {
			items = append(items, sidebarItem{label: "Mount to Pod", action: "vol_mount"})
			items = append(items, sidebarItem{label: "Delete", action: "vol_delete"})
		}
	}

	m.sidebarItems = items

	// Clamp cursor
	if m.sidebarCursor >= len(items) {
		m.sidebarCursor = len(items) - 1
	}
	if m.sidebarCursor < 0 {
		m.sidebarCursor = 0
	}
	// Skip non-selectable items
	m.sidebarCursor = m.nextSelectableSidebar(m.sidebarCursor, 1)
}

// nextSelectableSidebar finds the next selectable item in the given direction.
func (m model) nextSelectableSidebar(from, dir int) int {
	for i := from; i >= 0 && i < len(m.sidebarItems); i += dir {
		item := m.sidebarItems[i]
		if !item.isHeader && !item.isSep {
			return i
		}
	}
	return from
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// q to quit — but not when typing text
		if msg.String() == "q" && !m.isTypingText() {
			return m, tea.Quit
		}

		// Tab toggles focus between sidebar and content
		if msg.String() == "tab" {
			m.sidebarFocus = !m.sidebarFocus
			return m, nil
		}

		if m.sidebarFocus {
			// Forward shortcut keys to content panel instead of swallowing them
			key := msg.String()
			isSidebarNav := key == "up" || key == "down" || key == "k" || key == "j" || key == "enter"
			if !isSidebarNav {
				// Let the active screen handle shortcuts (c/d/m/r on volumes, etc.)
				switch m.activeScreen {
				case screenVolumes:
					mdl, cmd := m.updateVolumes(msg)
					// If we entered create or delete mode, switch focus to content
					if mdl.volumeCreateMode || mdl.volumeDeleteConfirm {
						mdl.sidebarFocus = false
					}
					return mdl, cmd
				case screenDashboard:
					// Dashboard shortcuts: n for new pod
					if key == "n" {
						return m.executeSidebarAction("launch")
					}
					if key == "s" {
						return m.executeSidebarAction("stop")
					}
					if key == "r" && m.dashSession != nil && (m.dashSession.Status == "stopped" || m.dashSession.Status == "exited") {
						return m.executeSidebarAction("resume")
					}
				}
			}
			return m.updateSidebar(msg)
		}

		// Dispatch to active screen content handler
		switch m.activeScreen {
		case screenDashboard:
			// Dashboard content has no interactive elements
			return m, nil
		case screenLaunch:
			return m.updateLaunch(msg)
		case screenVolumes:
			return m.updateVolumes(msg)
		case screenResize:
			return m.updateResize(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	// Dashboard messages
	case statusRefreshMsg:
		m.dashLoading = false
		m.dashError = msg.err
		m.dashSession = msg.session
		m.dashOtherCount = msg.otherCount
		m.rebuildSidebar()
		if m.activeScreen == screenDashboard {
			return m, tickCmd()
		}

	case tickMsg:
		if m.activeScreen == screenDashboard {
			return m, m.fetchStatus()
		}

	case podActionDoneMsg:
		if msg.err != nil {
			m.dashError = msg.err
		}
		// Refresh status after any action
		m.dashLoading = true
		return m, m.fetchStatus()

	// Launch messages
	case templatesLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.templates = msg.templates
		m.launchStep = 1
		m.templateCursor = 0

	case launchDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.session = msg.session
		// After successful launch, go to dashboard to see the pod
		m.activeScreen = screenDashboard
		m.sidebarFocus = true
		m.dashLoading = true
		m.session = nil
		m.err = nil
		m.launchStep = 0
		m.rebuildSidebar()
		return m, m.fetchStatus()

	// Resize messages
	case resizeDoneMsg:
		if msg.err != nil {
			m.resizeError = fmt.Sprintf("Resize failed: %v", msg.err)
			m.resizeConfirm = false
			return m, nil
		}
		// Success — go back to dashboard
		m.activeScreen = screenDashboard
		m.sidebarFocus = true
		m.dashLoading = true
		m.rebuildSidebar()
		return m, m.fetchStatus()

	// Volume messages
	case volumesLoadedMsg:
		m.volumesLoading = false
		if msg.err != nil {
			m.volumesError = msg.err
			return m, nil
		}
		m.volumes = msg.volumes
		m.volumesError = nil
		m.rebuildSidebar()

	case dataCentersLoadedMsg:
		if msg.err != nil {
			m.volumesError = msg.err
			return m, nil
		}
		m.dataCenters = msg.dataCenters

	case volumeCreatedMsg:
		if msg.err != nil {
			m.volumesError = msg.err
			return m, nil
		}
		m.volumeCreateMode = false
		m.newVolumeName = ""
		m.newVolumeSizeGB = ""
		m.volumeCreateStep = 0
		m.volumesLoading = true
		m.rebuildSidebar()
		return m, m.loadVolumes()

	case volumeDeletedMsg:
		m.volumeDeleteConfirm = false
		if msg.err != nil {
			m.volumesError = msg.err
			return m, nil
		}
		m.volumesLoading = true
		m.rebuildSidebar()
		return m, m.loadVolumes()
	}

	return m, nil
}

// isTypingText returns true when the user is in a text input context.
func (m model) isTypingText() bool {
	if m.activeScreen == screenLaunch && m.launchStep == 2 {
		return true
	}
	if m.activeScreen == screenResize {
		return true
	}
	if m.activeScreen == screenVolumes && m.volumeCreateMode {
		return true
	}
	return false
}

// updateSidebar handles key input when the sidebar is focused.
func (m model) updateSidebar(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		for i := m.sidebarCursor - 1; i >= 0; i-- {
			if !m.sidebarItems[i].isHeader && !m.sidebarItems[i].isSep {
				m.sidebarCursor = i
				break
			}
		}
	case "down", "j":
		for i := m.sidebarCursor + 1; i < len(m.sidebarItems); i++ {
			if !m.sidebarItems[i].isHeader && !m.sidebarItems[i].isSep {
				m.sidebarCursor = i
				break
			}
		}
	case "enter":
		if m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sidebarItems) {
			return m.executeSidebarAction(m.sidebarItems[m.sidebarCursor].action)
		}
	}
	return m, nil
}

// executeSidebarAction performs the action associated with a sidebar item.
func (m model) executeSidebarAction(action string) (model, tea.Cmd) {
	switch action {
	case "dashboard":
		m.activeScreen = screenDashboard
		m.dashLoading = true
		m.rebuildSidebar()
		return m, tea.Batch(m.fetchStatus(), tickCmd())

	case "volumes":
		m.activeScreen = screenVolumes
		m.volumesLoading = true
		m.volumeCreateMode = false
		m.volumeDeleteConfirm = false
		m.volumeCursor = 0
		m.rebuildSidebar()
		return m, tea.Batch(m.loadVolumes(), m.loadDataCenters())

	case "launch":
		m.activeScreen = screenLaunch
		m.launchStep = 0
		m.tierCursor = 0
		m.session = nil
		m.err = nil
		m.sidebarFocus = false // focus content for tier selection
		m.rebuildSidebar()
		return m, nil

	case "resize":
		if m.dashSession == nil {
			return m, nil
		}
		m.activeScreen = screenResize
		m.resizeCursor = 0
		m.resizeInput = ""
		m.resizeVolGB = 0
		m.resizeDiskGB = 0
		m.resizeError = ""
		m.resizeConfirm = false
		m.sidebarFocus = false
		m.rebuildSidebar()
		return m, nil

	case "stop":
		return m, m.doPodAction("stop")
	case "resume":
		return m, m.doPodAction("resume")
	case "kill":
		return m, m.doPodAction("kill")

	case "vol_create":
		m.volumeCreateMode = true
		m.volumeCreateStep = 0
		m.newVolumeName = ""
		m.newVolumeSizeGB = ""
		m.dcCursor = 0
		m.sidebarFocus = false
		m.rebuildSidebar()
		return m, nil

	case "vol_mount":
		if len(m.volumes) > 0 && m.volumeCursor < len(m.volumes) {
			vol := m.volumes[m.volumeCursor]
			m.mountVolume = &vol
			// Switch to launch flow
			m.activeScreen = screenLaunch
			m.launchStep = 0
			m.tierCursor = 0
			m.session = nil
			m.err = nil
			m.sidebarFocus = false
			m.rebuildSidebar()
		}
		return m, nil

	case "vol_delete":
		if len(m.volumes) > 0 {
			m.volumeDeleteConfirm = true
			m.sidebarFocus = false
		}
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	// Build sidebar
	sidebar := m.renderSidebar()

	// Build content panel
	var content string
	switch m.activeScreen {
	case screenDashboard:
		content = m.viewDashboard()
	case screenLaunch:
		if m.err != nil {
			content = m.viewError()
		} else if m.session != nil {
			content = m.viewDone()
		} else {
			content = m.viewLaunchPanel()
		}
	case screenVolumes:
		content = m.viewVolumes()
	case screenResize:
		content = m.viewResize()
	}

	// Wrap content in content style
	contentPanel := contentStyle.Render(content)

	// Join sidebar and content horizontally
	layout := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, contentPanel)

	// Add logo header above the layout
	header := renderLogo() + "\n" + renderDivider(0) + "\n"
	full := header + layout

	// Center in terminal
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			full)
	}
	return full
}

// renderSidebar renders the sidebar with navigation and actions.
func (m model) renderSidebar() string {
	var sb strings.Builder

	for i, item := range m.sidebarItems {
		if item.isSep {
			sep := strings.Repeat("─", sidebarWidth-4)
			sb.WriteString(sidebarSeparatorStyle.Render(sep) + "\n")
			continue
		}
		if item.isHeader {
			sb.WriteString(sidebarSectionStyle.Render(item.label) + "\n")
			continue
		}

		isSelected := m.sidebarFocus && i == m.sidebarCursor
		isActiveScreen := false

		// Highlight the current screen's nav item
		switch item.action {
		case "dashboard":
			isActiveScreen = m.activeScreen == screenDashboard
		case "volumes":
			isActiveScreen = m.activeScreen == screenVolumes
		case "launch":
			isActiveScreen = m.activeScreen == screenLaunch
		}

		if isSelected {
			sb.WriteString(sidebarSelectedStyle.Render("▸ " + item.label) + "\n")
		} else if isActiveScreen {
			activeStyle := lipgloss.NewStyle().
				Foreground(catCyan).
				Bold(true).
				Padding(0, 1)
			sb.WriteString(activeStyle.Render("  " + item.label) + "\n")
		} else {
			sb.WriteString(sidebarItemStyle.Render("  " + item.label) + "\n")
		}
	}

	// Add quit hint at bottom
	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("  q quit"))

	// Apply sidebar style based on focus
	style := sidebarStyle
	if m.sidebarFocus {
		style = sidebarActiveStyle
	}

	return style.Render(sb.String())
}

func (m model) viewDone() string {
	var s string
	s += successStyle.Render("✓ GPU Pod Launched!") + "\n\n"

	if m.session != nil {
		infoBox := boxStyle.Copy().BorderForeground(catGreen)
		info := labelStyle.Render("Pod ID  ") + valueStyle.Render(m.session.PodID) + "\n" +
			labelStyle.Render("Tier    ") + valueStyle.Render(m.session.Tier) + "\n" +
			labelStyle.Render("GPU     ") + valueStyle.Render(m.session.GPUType) + "\n" +
			labelStyle.Render("Status  ") + lipgloss.NewStyle().Foreground(catGreen).Bold(true).Render(m.session.Status)
		if m.session.VolumeGB != nil {
			info += "\n" + labelStyle.Render("Volume  ") + valueStyle.Render(fmt.Sprintf("%d GB", *m.session.VolumeGB))
		}
		if m.session.ContainerDiskGB != nil {
			info += "\n" + labelStyle.Render("Disk    ") + valueStyle.Render(fmt.Sprintf("%d GB", *m.session.ContainerDiskGB))
		}
		s += infoBox.Render(info) + "\n"
	}

	s += "\n" + helpStyle.Render("press enter or q to exit")
	return s
}

func (m model) viewError() string {
	var s string
	s += errorStyle.Render("✗ Error") + "\n\n"
	if m.err != nil {
		errMsg := m.err.Error()
		if len(errMsg) > 60 {
			words := strings.Fields(errMsg)
			var lines []string
			line := ""
			for _, w := range words {
				if len(line)+len(w)+1 > 60 {
					lines = append(lines, line)
					line = w
				} else {
					if line != "" {
						line += " "
					}
					line += w
				}
			}
			if line != "" {
				lines = append(lines, line)
			}
			errMsg = strings.Join(lines, "\n  ")
		}
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(errMsg) + "\n"
	}
	s += "\n" + helpStyle.Render("esc back  q quit")
	return s
}

// RunTUI launches the interactive GPU control center.
func RunTUI(mgr *Manager) error {
	p := tea.NewProgram(initialModel(mgr), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
