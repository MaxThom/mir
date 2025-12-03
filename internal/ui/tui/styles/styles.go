package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// TODO research on how to create a
// proper palette of colors with
// accents and other things

var (
	Mir       = lipgloss.NewStyle().Foreground(lipgloss.Color("#C26BFF"))
	Error     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0500"))
	Sucess    = lipgloss.NewStyle().Foreground(lipgloss.Color("#80ff00"))
	Info      = lipgloss.NewStyle().Foreground(lipgloss.Color("#007fff"))
	Primary   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF75B7"))
	FormLabel = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#B2B2B2",
		Dark:  "#B2B2B2",
	})
	Help = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#B2B2B2",
		Dark:  "#4A4A4A",
	})
	CursorUnderline = lipgloss.NewStyle().Underline(true)
	Selection       = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF75B7"))
	Subtext         = lipgloss.NewStyle().Foreground(lipgloss.Color("#605F63"))
	SidePanel       = lipgloss.NewStyle().
			Align(lipgloss.Left, lipgloss.Top).
			Border(lipgloss.RoundedBorder(), true, true, true, true).
			Foreground(Help.GetForeground())
)
