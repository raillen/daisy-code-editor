// Package terminal renders the embedded terminal panel.
//
// The panel is not an emulator and does not pretend to be one: it draws the
// last lines the shell wrote, which is the same thing the reference does. The
// difference that matters is the empty state — a panel with no shell yet and a
// panel whose shell died look identical otherwise, and the reader cannot tell
// "starting" from "crashed".
package terminal

import (
	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/theme"
)

// View is what the panel needs to draw.
type View struct {
	// Rows are the shell's last lines, oldest first. Already without escape
	// sequences: the panel paints text, and a `Term` is a promise of VT that
	// neither implementation keeps (ledger B5).
	Rows []string
	// Focused says the panel holds the keyboard. The marker is **text**, never
	// colour: the rule of this project is that no state is carried by colour
	// alone, and a panel that only shows focus by tint is a panel a reader sem
	// cor não sabe se está digitando.
	Focused bool
	// Attached is whether a shell is running. False with no rows is "starting",
	// and that has to be said.
	Attached bool
	// LastError is the shell's own complaint, if it made one.
	LastError string
	Theme     theme.Theme
}

// The states the panel can be in, all of them readable as text.
const (
	emptyMessage = "(terminal: iniciando)"
	deadMessage  = "(terminal: encerrado)"
)

// Render draws the panel, one string per line, each exactly width cells.
func Render(width, height int, view View) []string {
	if width <= 0 || height <= 0 {
		return nil
	}

	rows := visible(view.Rows, height)
	out := make([]string, 0, height)

	if len(rows) == 0 {
		out = append(out, style(view, layout.Pad(" "+message(view), width)))
		rows = nil
	}

	// The last rows win: the newest output is what the reader is looking at, and
	// a shell that has printed more than fits pushes the old lines off.
	used := 0
	for index := len(rows) - 1; index >= 0 && used < height; index-- {
		out = append([]string{style(view, layout.Pad(rows[index], width))}, out...)
		used++
	}

	for len(out) < height {
		out = append(out, style(view, layout.Pad("", width)))
	}
	return out[:height]
}

// message is what the panel says when it has no rows to show.
func message(view View) string {
	if view.LastError != "" {
		return deadMessage + " · " + view.LastError
	}
	if !view.Attached {
		return emptyMessage
	}
	return deadMessage
}

// visible is the tail of the rows that fits in height.
func visible(rows []string, height int) []string {
	if len(rows) <= height {
		return rows
	}
	return rows[len(rows)-height:]
}

// style pads to the exact width and applies the panel's own background, with the
// focus marker in front of the first row.
func style(view View, line string) string {
	if !view.Focused {
		return view.Theme.Gutter().Render(line)
	}
	marked := caretMarker + line[len(caretMarker):]
	return view.Theme.Gutter().Render(marked)
}

// caretMarker sits in the first column while the panel has the keyboard.
//
// Uma coluna, e medida: o marcador tem a largura que `layout` diz que tem, e não
// a que este arquivo imagina. Foi um bug nesta família — o `▶` do componente
// `component` reservava três células e a função de medida contava duas.
const caretMarker = ">"
