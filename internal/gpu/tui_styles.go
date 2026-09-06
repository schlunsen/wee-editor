package gpu

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Colors ──────────────────────────────────────────────────────────

var (
	// Brand colors
	catPurple = lipgloss.Color("99")
	catPink   = lipgloss.Color("205")
	catCyan   = lipgloss.Color("51")
	catGreen  = lipgloss.Color("42")
	catYellow = lipgloss.Color("220")
	catDim    = lipgloss.Color("240")
	catWhite  = lipgloss.Color("255")
	catRed    = lipgloss.Color("196")
)

// ── Base Styles ─────────────────────────────────────────────────────

var (
	logoStyle = lipgloss.NewStyle().
			Foreground(catPurple).
			Bold(true)

	logoTextStyle = lipgloss.NewStyle().
			Foreground(catPink).
			Bold(true)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(catCyan).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(catDim).
			Italic(true)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(catWhite).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Padding(0, 1)

	dimStyle = lipgloss.NewStyle().
			Foreground(catDim)

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(catGreen)

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(catRed)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			MarginTop(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("57")).
			Padding(0, 2)

	tierBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			Width(46)

	labelStyle = lipgloss.NewStyle().
			Foreground(catDim)

	valueStyle = lipgloss.NewStyle().
			Foreground(catWhite).
			Bold(true)

	costStyle = lipgloss.NewStyle().
			Foreground(catYellow).
			Bold(true)

	spinnerDots = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
)

// ── Sidebar Styles ──────────────────────────────────────────────────

var (
	sidebarWidth = 22

	sidebarStyle = lipgloss.NewStyle().
			Width(sidebarWidth).
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(catDim).
			Padding(1, 1)

	sidebarActiveStyle = lipgloss.NewStyle().
				Width(sidebarWidth).
				BorderRight(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(catCyan).
				Padding(1, 1)

	sidebarItemStyle = lipgloss.NewStyle().
				Foreground(catWhite).
				Padding(0, 1)

	sidebarSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(catWhite).
				Background(lipgloss.Color("57")).
				Padding(0, 1).
				Width(sidebarWidth - 2)

	sidebarSectionStyle = lipgloss.NewStyle().
				Foreground(catDim).
				Bold(true).
				Padding(0, 1).
				MarginTop(1)

	sidebarSeparatorStyle = lipgloss.NewStyle().
				Foreground(catDim)
)

// ── Content Panel Styles ────────────────────────────────────────────

var (
	contentStyle = lipgloss.NewStyle().
			PaddingLeft(2)
)

// ── Status Dot Styles ───────────────────────────────────────────────

var (
	statusRunning  = lipgloss.NewStyle().Foreground(catGreen).Render("●")
	statusCreating = lipgloss.NewStyle().Foreground(catYellow).Render("●")
	statusStopped  = lipgloss.NewStyle().Foreground(catRed).Render("●")
	statusError    = lipgloss.NewStyle().Foreground(catRed).Bold(true).Render("✕")
)

// ── Helper Functions ────────────────────────────────────────────────

func renderLogo() string {
	catArt := "   ∧ ∧\n  ≡>_≡\n  ╱| |╲"
	cat := logoStyle.Render(catArt)
	titleLine := logoTextStyle.Render(" wee") + dimStyle.Render(".cat") + logoTextStyle.Render(" gpu")
	tagline := dimStyle.Render(" cloud GPUs, meow")
	titleBlock := titleLine + "\n" + tagline
	return lipgloss.JoinHorizontal(lipgloss.Center, cat, "  ", titleBlock)
}

func renderDivider(width int) string {
	if width <= 0 {
		width = 50
	}
	if width > 60 {
		width = 60
	}
	// Gradient-ish divider with dots
	left := strings.Repeat("━", width/3)
	mid := strings.Repeat("─", width/3)
	right := strings.Repeat("╌", width-2*(width/3))
	return lipgloss.NewStyle().Foreground(lipgloss.Color("57")).Render(left) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(mid) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render(right)
}
