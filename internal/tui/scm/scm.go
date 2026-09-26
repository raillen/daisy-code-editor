// Package scm renders the source-control panel, on the body's right side.
//
// Flat, with the letter, like the tree's column: an entry is one row, and the
// state is in **text** (`M`, `A`, `?`…) — a panel of coloured names without
// letters is the same defect the tree had before the badge column existed.
package scm

import (
	"github.com/ori-team/oride/internal/tui/component"
	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/theme"
)

// Row is one changed path as the panel sees it.
type Row struct {
	// Path is the workspace-relative path, which é o que o leitor abre com
	// `Enter`.
	Path string
	// Status is the git letter, in the oracle's own spelling.
	Status string
}

// View is what the panel needs to draw.
type View struct {
	// Rows is the panel's content, oldest-change first.
	Rows []Row
	// Selected is the row the keyboard is on.
	Selected int
	// Focused says the panel holds the keyboard.
	Focused bool
	Theme   theme.Theme
}

// Title is the panel's own name, which é a única coisa que ela pode afirmar
// quando ainda não leu nada.
const Title = "SCM"

// Render draws the panel, one string per row, each exactly width cells.
func Render(width, height int, view View) []string {
	if width <= 0 || height <= 0 {
		return nil
	}

	start, end := component.Window(len(view.Rows), height, view.Selected, 0)
	rows := make([]string, 0, height)

	if len(view.Rows) == 0 {
		rows = append(rows, layout.Pad(" (limpo)", width))
	}

	for index := start; index < end; index++ {
		row := view.Rows[index]
		selected := index == view.Selected

		rows = append(rows, component.RowWithBadge(
			row.Path, 0, selected, width,
			view.Theme.FileStatus(row.Status), row.Status))
	}

	for len(rows) < height {
		rows = append(rows, layout.Pad("", width))
	}
	return rows
}
