// Package editorview renders the text viewport.
package editorview

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ori-team/oride/internal/syntax"
	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/theme"
)

// View is what the viewport needs to draw.
//
// Lines arrive already extracted rather than as a buffer: the surface must not
// reach into the document, and a viewport does not need the whole file to draw
// one screen.
type View struct {
	Lines        []Line
	Offset       int
	Caret        Position
	HasCaret     bool
	Selection    Selection
	Matches      []Match
	CurrentMatch int
	Gutter       bool
	Focused      bool
	// Theme dims the gutter. The zero value renders plain.
	Theme theme.Theme
}

// Line is one row of text and the roles of its bytes.
//
// The roles travel with the text rather than in a second array beside it: two
// index-aligned arrays are a coupling nothing checks, and a line that lost its
// tokens would still be a line, painted plain, with nothing saying why.
type Line struct {
	Text string
	// Tokens carry byte offsets into Text, which is the index the buffer and the
	// highlighter both use. A line with no tokens is plain text, not a line that
	// failed to be highlighted.
	Tokens []syntax.Token
}

// TotalLines is the document's length, which the gutter is sized from.
func (v View) TotalLines() int { return v.Offset + len(v.Lines) }

// Match is one search result, in document coordinates.
type Match struct {
	Start Position
	End   Position
}

// Position is a caret location, zero-based, in document coordinates.
//
// The column counts scalars, because that is what the buffer counts and every
// coordinate the model hands over arrives through ByteToCaret. The painting
// measures it into cells — the conversion lives in layout.Cells, and the reason
// it cannot be skipped is in that function's contract.
type Position struct {
	Line   int
	Column int
}

// Selection is the selected range, in document coordinates.
type Selection struct {
	Start Position
	End   Position
	Empty bool
}

// isCollapsed reports whether the range covers nothing.
//
// A zero-value Selection has Empty false and Start equal to End, so a surface that
// trusted the flag alone would paint a collapsed range — and, worse, would take the
// painting path for a line that needed the plain one, losing the truncation marker
// on every long line. The surface decides from the range, not from the caller.
func (s Selection) isCollapsed() bool {
	return s.Start.Line == s.End.Line && s.Start.Column == s.End.Column
}

// caretMarker sits in the gutter on the line the caret is on.
//
// The caret is marked in the gutter rather than painted over a character: a
// terminal cursor is a real thing the runtime draws, and a surface that also
// drew its own would show two. And the marker is text, so it survives a no-colour
// terminal — where a tinted caret would be invisible and the position unknowable.
const caretMarker = ">"

// gutterPadding is the space between the line numbers and the text.
const gutterPadding = 1

// Render draws the viewport, one string per line, each exactly width cells.
func Render(width, height int, view View) []string {
	if width <= 0 || height <= 0 {
		return nil
	}

	gutterWidth := 0
	if view.Gutter {
		gutterWidth = gutterSize(view.TotalLines()) + gutterPadding
	}
	if gutterWidth >= width {
		// The gutter would leave no room for text, so it is what gives way.
		gutterWidth = 0
	}

	rows := make([]string, 0, height)
	for index := range view.Lines {
		if len(rows) == height {
			break
		}
		rows = append(rows, renderLine(width, gutterWidth, index, view))
	}

	// Padding rows keep the frame exactly as tall as asked. A viewport that came
	// back short would let whatever is below it move up.
	for len(rows) < height {
		blank := ""
		if gutterWidth > 0 {
			blank = layout.Pad("", gutterWidth)
		}
		rows = append(rows, layout.Pad(blank, width))
	}
	return rows
}

// renderLine draws one line with its gutter.
//
// index is a position in the visible slice; the gutter shows the document line
// number, which is what a reader expects to see beside the text.
func renderLine(width, gutterWidth, index int, view View) string {
	line := view.Lines[index]
	number := view.Offset + index

	if gutterWidth == 0 {
		// Without a gutter there is no line number, but there is still a
		// selection and there are still roles: returning the padded text here
		// skipped the painting entirely, and every line came out plain.
		return paintLine(line, number, width, view)
	}

	marker := " "
	if view.HasCaret && number == view.Caret.Line {
		marker = caretMarker
	}
	gutter := marker + layout.Pad(strconv.Itoa(number+1), gutterWidth-1)

	// The gutter is styled and the text is not, so the line stays selectable by
	// copy: a background over the text would be captured by a terminal copy.
	body := paintLine(line, number, width-gutterWidth, view)
	return layout.Pad(view.Theme.Gutter().Render(gutter), gutterWidth) + body
}

// cell is what a column of the line is, before any styling.
//
// A cell array rather than a chain of conditions per rune: selection, matches
// and syntax overlap, and deciding the winner once per cell is what makes the
// precedence explicit instead of an accident of the order the styles were
// applied.
type cell struct {
	kind cellKind
	role syntax.Role
}

// cellKind is what a column is, ignoring which syntax role paints it.
type cellKind int

const (
	cellPlain cellKind = iota
	cellToken
	cellMatch
	cellCurrentMatch
	cellSelection
)

// paintLine draws the line with its roles, matches and selection tinted.
//
// Precedence is decided per cell: the selection wins over a match, because a
// reader who has selected text is asking about the selection, and a match wins
// over a role, because a highlight the reader asked for outranks a colour the
// theme chose.
func paintLine(line Line, number, width int, view View) string {
	kinds := classify(line, number, width, view)
	if !hasStyled(kinds) {
		return layout.Pad(line.Text, width)
	}

	var out strings.Builder
	position := 0
	for position < width {
		kind := kinds[position]
		run := 0
		for position+run < width && kinds[position+run] == kind {
			run++
		}

		segment := layout.Pad(layout.Slice(line.Text, position, run), run)
		out.WriteString(styleFor(kind, view.Theme).Render(segment))
		position += run
	}
	return out.String()
}

// classify decides what each column of the line is.
func classify(line Line, number, width int, view View) []cell {
	kinds := make([]cell, width)
	paintTokens(kinds, line, width)

	for _, match := range matchesOn(number, view.Matches) {
		mark(kinds, line.Text, match.Start, match.End, cell{kind: cellMatch})
	}
	if view.CurrentMatch >= 0 && view.CurrentMatch < len(view.Matches) {
		current := view.Matches[view.CurrentMatch]
		if current.Start.Line <= number && number <= current.End.Line {
			mark(kinds, line.Text, current.Start, current.End, cell{kind: cellCurrentMatch})
		}
	}

	if !view.Selection.Empty && !view.Selection.isCollapsed() && lineIsSelected(number, view.Selection) {
		start, end := selectedColumns(number, line.Text, view.Selection)
		for column := start; column < end && column < width; column++ {
			kinds[column] = cell{kind: cellSelection}
		}
	}
	return kinds
}

// paintTokens tints the columns the highlighter gave a role.
//
// Offsets are bytes and the array is cells, so the conversion happens per token
// boundary rather than per byte: a token is painted as a run, and the two
// measurements cannot disagree about where it starts.
func paintTokens(kinds []cell, line Line, width int) {
	for _, token := range line.Tokens {
		from := layout.Cells(line.Text, token.Start)
		to := layout.Cells(line.Text, token.End)
		for column := from; column < to && column < len(kinds); column++ {
			if kinds[column].kind == cellPlain {
				kinds[column] = cell{kind: cellToken, role: token.Role}
			}
		}
	}
}

// matchesOn lists the matches touching a line.
func matchesOn(line int, matches []Match) []Match {
	out := make([]Match, 0, 2)
	for _, match := range matches {
		if match.Start.Line <= line && line <= match.End.Line {
			out = append(out, match)
		}
	}
	return out
}

// mark paints the columns of one range.
func mark(kinds []cell, text string, start, end Position, kind cell) {
	line := start.Line
	from, to := selectedColumns(line, text, Selection{Start: start, End: end})
	for column := from; column < to && column < len(kinds); column++ {
		kinds[column] = kind
	}
}

// hasStyled reports whether anything on the line needs styling.
//
// The plain path stays the common one for a document with no lexer: most lines
// of most frames have no role, no match and no selection, and building styled
// output for them would cost escape sequences on every row.
func hasStyled(kinds []cell) bool {
	for _, kind := range kinds {
		if kind.kind != cellPlain {
			return true
		}
	}
	return false
}

// styleFor maps a cell onto the style that draws it.
func styleFor(kind cell, theme theme.Theme) lipgloss.Style {
	switch kind.kind {
	case cellSelection:
		return theme.Selection()
	case cellCurrentMatch:
		return theme.CurrentMatch()
	case cellMatch:
		return theme.Match()
	case cellToken:
		return theme.Syntax(kind.role)
	}
	return lipgloss.NewStyle()
}

// lineIsSelected reports whether the selection touches a line at all.
func lineIsSelected(line int, selection Selection) bool {
	return line >= selection.Start.Line && line <= selection.End.Line
}

// selectedColumns is the selected range on one line, in cells.
//
// The columns arrive as scalars, because the buffer counts scalars, and they are
// measured into cells here. The conversion is this function's job and nobody
// else's: the buffer's column and the layout's cell are two units for the same
// distance, and a selection after a wide character was a column short every time
// it was compared without measuring.
//
// The first and last lines of a range are partial; every line between them is
// covered whole. A column past the end of the line would make the padding
// arithmetic produce a negative width, so it is clamped.
func selectedColumns(line int, text string, selection Selection) (start, end int) {
	lineWidth := layout.Width(text)

	switch {
	case line == selection.Start.Line && line == selection.End.Line:
		start, end = layout.Cells(text, selection.Start.Column), layout.Cells(text, selection.End.Column)
	case line == selection.Start.Line:
		start, end = layout.Cells(text, selection.Start.Column), lineWidth
	case line == selection.End.Line:
		start, end = 0, layout.Cells(text, selection.End.Column)
	default:
		start, end = 0, lineWidth
	}

	start = min(max(0, start), lineWidth)
	end = min(max(start, end), lineWidth)
	return start, end
}

// gutterSize is the width of the widest line number the file will reach.
//
// Derived from the document's length rather than from the visible slice: numbering
// that re-widened as the reader scrolled would shift every line of text sideways.
func gutterSize(totalLines int) int {
	if totalLines <= 1 {
		return 1
	}
	return len(strconv.Itoa(totalLines))
}
