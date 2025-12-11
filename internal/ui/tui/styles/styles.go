package styles

import (
	"github.com/charmbracelet/lipgloss"
)

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

// StatusColors defines standard colors for different statuses
var StatusColors = struct {
	Success   lipgloss.Color
	Error     lipgloss.Color
	Pending   lipgloss.Color
	Warning   lipgloss.Color
	Validated lipgloss.Color
}{
	Success:   lipgloss.Color("34"),
	Error:     lipgloss.Color("160"),
	Pending:   lipgloss.Color("214"),
	Warning:   lipgloss.Color("214"),
	Validated: lipgloss.Color("214"),
}

// RenderStatusBadge creates a colored status badge for the given text and color
func RenderStatusBadge(text string, color lipgloss.Color) string {
	return lipgloss.NewStyle().Foreground(color).Render(text)
}
