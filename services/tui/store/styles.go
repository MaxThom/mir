package store

import "github.com/charmbracelet/lipgloss"

var Styles = map[string]lipgloss.Style{
	"mir":     lipgloss.NewStyle().Foreground(lipgloss.Color("#C26BFF")),
	"error":   lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0500")),
	"success": lipgloss.NewStyle().Foreground(lipgloss.Color("#80ff00")),
	"info":    lipgloss.NewStyle().Foreground(lipgloss.Color("#007fff")),
	"primary": lipgloss.NewStyle().Foreground(lipgloss.Color("#FF75B7")),
	"help": lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "#B2B2B2",
		Dark:  "#4A4A4A",
	}),
}
