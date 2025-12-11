package utils

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/maxthom/mir/internal/ui/tui/components/menu"
)

// ViewportDimensions holds standard viewport sizing configuration
type ViewportDimensions struct {
	WidthOffset  int
	HeightOffset int
	MinWidth     int
	MinHeight    int
}

// DefaultViewportDimensions returns standard dimensions for viewports
func DefaultViewportDimensions() ViewportDimensions {
	return ViewportDimensions{
		WidthOffset:  ViewportWidthOffset,
		HeightOffset: ViewportHeightOffset,
		MinWidth:     MinViewportWidth,
		MinHeight:    MinViewportHeight,
	}
}

// UpdateViewportSize updates viewport dimensions based on window size.
// It applies offsets and enforces minimum dimensions.
func UpdateViewportSize(vp *viewport.Model, screenWidth, screenHeight int, dims ViewportDimensions) {
	vp.Width = screenWidth - dims.WidthOffset
	if vp.Width < dims.MinWidth {
		vp.Width = dims.MinWidth
	}

	vp.Height = screenHeight - dims.HeightOffset
	if vp.Height < dims.MinHeight {
		vp.Height = dims.MinHeight
	}
}

// SyncViewportWithMenu synchronizes viewport scrolling with menu cursor position.
// It scrolls the viewport to keep the selected menu item visible.
func SyncViewportWithMenu(vp *viewport.Model, choice menu.Option, position int, total int, direction int) {
	lineCount := len(strings.Split(choice.Description, "\n"))

	if position == 0 {
		vp.GotoTop()
	} else if position == total-1 {
		vp.GotoBottom()
	} else {
		vp.ScrollDown(direction * lineCount)
	}
}
