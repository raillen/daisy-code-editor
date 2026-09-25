// Package findbar renders the in-buffer search bar.
package findbar

import (
	"strconv"
	"strings"

	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/theme"
)

// View is what the bar needs to draw.
//
// The two fields arrive **already rendered**, as the exact-width string the
// component `textinput` draws. They are not strings to lay out: quem escreve,
// move o cursor, mostra o placeholder e rola a cauda é o componente, e um segundo
// parágrafo fazendo o mesmo é a duplicação que este pacote existia para ter.
type View struct {
	// Field is the search field, already trimmed to FieldWidth cells.
	Field string
	// ReplaceField is the replacement field, same width. Ignored when
	// ShowReplace is false.
	ReplaceField  string
	ShowReplace   bool
	CaseSensitive bool
	IgnoreAccents bool
	WholeWord     bool
	UseRegex      bool
	// Current is the one-based position of the highlighted match, and Total how
	// many there are. Zero total means no match, which is a state the bar has to
	// say out loud.
	Current int
	Total   int
	// RegexError is set when the pattern does not compile. A half-typed pattern
	// is normal, so it is reported in place rather than refused.
	RegexError string
	Theme      theme.Theme
}

// labels.
const (
	queryLabel   = " buscar: "
	replaceLabel = "substituir: "
	noMatch      = "sem resultados"
)

// FieldWidth is how many cells the text field occupies.
//
// Exported because a caller has to tell the component `textinput` o quanto
// medir, e um valor em dois lugares é um valor que vai divergir — que foi
// exatamente o que aconteceu com o `…` e com o `▶` neste projeto.
const FieldWidth = 28

// Height is how many rows the bar needs.
//
// A method rather than a constant, because revealing the replacement field is
// what makes the bar grow — and the layout has to be told before it divides the
// screen, not after.
func (v View) Height() int {
	if v.ShowReplace {
		return 2
	}
	return 1
}

// Render draws the bar, one string per row, each exactly width cells.
func Render(width int, view View) []string {
	if width <= 0 {
		return nil
	}

	rows := []string{queryRow(width, view)}
	if view.ShowReplace {
		rows = append(rows, replaceRow(width, view))
	}
	return rows
}

// queryRow is the search line: the query, its flags, and the count.
func queryRow(width int, view View) string {
	line := queryLabel + view.Field
	line += "  " + flags(view) + "  " + count(view)

	return style(view, line, width, false)
}

// replaceRow is the replacement line.
func replaceRow(width int, view View) string {
	return style(view, replaceLabel+view.ReplaceField, width, true)
}

// flags spells out the active search options.
//
// Spelled out rather than tinted: the rule is that no state is carried by colour,
// and a flag the reader cannot see is a flag they will misread a result over.
func flags(view View) string {
	active := make([]string, 0, 4)
	if view.CaseSensitive {
		active = append(active, "Aa")
	}
	if view.IgnoreAccents {
		active = append(active, "acentos")
	}
	if view.WholeWord {
		active = append(active, "palavra")
	}
	if view.UseRegex {
		active = append(active, "regex")
	}
	if len(active) == 0 {
		return ""
	}
	return "[" + strings.Join(active, " ") + "]"
}

// count reports the position among the matches, or the reason there are none.
func count(view View) string {
	if view.RegexError != "" {
		// The error is the useful part; the count would be a lie next to it,
		// since a pattern that does not compile matched nothing.
		return "padrão inválido: " + view.RegexError
	}
	if view.Total == 0 {
		return noMatch
	}
	return strconv.Itoa(view.Current) + "/" + strconv.Itoa(view.Total)
}

// style pads to the exact width and applies the bar's own background.
func style(view View, line string, width int, secondary bool) string {
	bar := layout.Pad(line, width)
	if secondary {
		return view.Theme.Gutter().Render(bar)
	}
	return view.Theme.Status().Render(bar)
}
