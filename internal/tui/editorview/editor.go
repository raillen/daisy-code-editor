// Package editorview renders the text viewport.
package editorview

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"github.com/ori-team/oride/internal/syntax"
	"github.com/ori-team/oride/internal/tui/component"
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
	// Wrap breaks a long line into as many rows as it takes instead of cutting
	// it: a 121-character line in a 60-column terminal has to be readable whole.
	//
	// O default é **ligado**, como no Zed, e `soft_wrap = false` desliga. Um
	// recurso que só existe atrás de uma chave é um recurso que ninguém usa.
	Wrap bool
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
//
// A janela é calculada em **linhas visuais**, e não em linhas do documento: com
// quebra, uma linha do documento ocupa N linhas da tela, e uma janela contada em
// linhas do documento mostraria um pedaço e esconderia o resto — que é exatamente
// o defeito que o wrap veio para tirar. A expansão acontece **uma vez**, aqui, e a
// mesma lista alimenta o cálculo da janela e a pintura, para que as duas não
// possam discordar de onde uma linha termina.
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
	textWidth := width - gutterWidth

	rows := view.visualRows(textWidth)
	start, end := component.Window(len(rows), height, view.caretRow(rows), 0)
	painted := make([]string, 0, height)

	for index := start; index < end; index++ {
		painted = append(painted, renderRow(gutterWidth, textWidth, rows[index], view))
	}

	// Padding rows keep the frame exactly as tall as asked. A viewport that came
	// back short would let whatever is below it move up.
	for len(painted) < height {
		blank := ""
		if gutterWidth > 0 {
			blank = layout.Pad("", gutterWidth)
		}
		painted = append(painted, layout.Pad(blank, width))
	}
	return painted
}

// visualRow is one row of the screen, with where it came from.
type visualRow struct {
	// Line is the document line this row belongs to.
	Line int
	// Text is this row's own slice of the line.
	Text string
	// Column is the scalar column where the row starts.
	Column int
	// IsFirst is whether the row opens its line: só a linha que começa na tela
	// mostra o número, que é o que evita um número repetido quatro vezes.
	IsFirst bool
	// IsCaret says the caret is in this row, e é o que marca a linha atual.
	IsCaret bool
}

// visualRows expands the document lines into the rows the screen shows.
//
// A lista inteira, e não só a fatia visível: a janela precisa saber quantas linhas
// visuais existem para decidir o que cabe, e o caret precisa achar a sua em meio a
// elas. Expandir só o visível faria a decisão depender do que já foi decidido.
func (v View) visualRows(width int) []visualRow {
	rows := make([]visualRow, 0, len(v.Lines))
	for index, line := range v.Lines {
		number := v.Offset + index

		pieces := []layout.WrappedRow{{Text: line.Text, Start: 0, End: len(line.Text)}}
		if v.Wrap {
			pieces = layout.Wrap(line.Text, width)
		}

		for piece, wrapped := range pieces {
			row := visualRow{
				Line:    number,
				Text:    wrapped.Text,
				Column:  utf8.RuneCountInString(line.Text[:wrapped.Start]),
				IsFirst: piece == 0,
			}
			row.IsCaret = v.caretInside(number, row.Column, wrapped.End, piece == len(pieces)-1)
			rows = append(rows, row)
		}
	}
	return rows
}

// caretInside reports whether the caret's column falls in this wrapped row.
//
// Dois acréscimos de um escalar, e os dois resolvem um caso em que o cursor
// desaparece — o pior defeito que uma tela pode ter:
//
//   - **linha vazia** tem zero colunas, e `[0, 0)` não contém nada;
//   - **caret no fim de uma linha quebrada** pertence à última linha dela, e não
//     à próxima: é onde o terminal o desenha, e sem isso o cursor pula de linha
//     toda vez que a seleção chega ao fim do texto.
func (v View) caretInside(line, rowStartColumn, rowEndByte int, lastRowOfLine bool) bool {
	if !v.HasCaret || v.Caret.Line != line {
		return false
	}
	index := line - v.Offset
	if index < 0 || index >= len(v.Lines) {
		return false
	}
	source := v.Lines[index].Text
	rowStartByte := layout.Offset(rowStartColumn, source)
	rowEnd := rowStartColumn + utf8.RuneCountInString(source[rowStartByte:rowEndByte])
	switch {
	case lastRowOfLine:
		rowEnd++
	case rowEnd == rowStartColumn:
		// Linha vazia: um intervalo de zero colunas não contém o cursor.
		rowEnd++
	}

	return v.Caret.Column >= rowStartColumn && v.Caret.Column < rowEnd
}

// caretRow is the visual row the caret is on, and it is o que a janela segue.
func (v View) caretRow(rows []visualRow) int {
	for index, row := range rows {
		if row.IsCaret {
			return index
		}
	}
	return 0
}

// renderRow draws one visual row with its gutter.
func renderRow(gutterWidth, textWidth int, row visualRow, view View) string {
	body := paintRow(row, textWidth, view)
	if gutterWidth == 0 {
		// Sem gutter não há número de linha, mas há seleção e há papéis: voltar
		// aqui com o texto só preenchido foi um defeito que existiu e fez toda
		// linha sair sem cor.
		return body
	}

	// O marcador acompanha o caret mesmo em linha quebrada: sem ele, uma linha
	// partida em quatro perde a única marca que diz onde o cursor está.
	marker := " "
	if row.IsCaret {
		marker = caretMarker
	}
	number := ""
	if row.IsFirst {
		number = strconv.Itoa(row.Line + 1)
	}
	gutter := marker + layout.Pad(number, gutterWidth-1)

	// The gutter is styled and the text is not, so the line stays selectable by
	// copy: a background over the text would be captured by a terminal copy.
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

// paintRow draws one visual row: its syntax roles, the matches and the selection
// that **cross it**.
//
// Tudo é recortado para a linha visual antes de pintar. A seleção e o match vivem em
// coordenadas do documento (linha e coluna de escalar), e a linha da tela é um
// recorte: sem o recorte, uma seleção de três linhas pintaria a linha inteira
// quatro vezes, e o realce da palavra mostraria a linha toda.
func paintRow(row visualRow, width int, view View) string {
	line := rowText(row, view)
	kinds := classify(line, row, width, view)
	highlight := currentLine(row, view)

	if !hasStyled(kinds) && highlight == nil {
		return layout.Pad(row.Text, width)
	}

	var out strings.Builder
	position := 0
	for position < width {
		kind := kinds[position]
		run := 0
		for position+run < width && kinds[position+run] == kind {
			run++
		}

		segment := layout.Pad(layout.Slice(row.Text, position, run), run)
		out.WriteString(styleFor(kind, view.Theme).Render(segment))
		position += run
	}

	if highlight == nil {
		return out.String()
	}
	// O realce de linha atual é um **fundo** por baixo do que já foi pintado, e
	// não uma cor de frente: um fundo Some atrás de um realce de sintaxe, que é
	// exatamente o que se quer de uma linha atual.
	return highlight.Render(out.String())
}

// currentLine is the style of the caret's line, or nil when the theme does not
// ask for one.
//
// A segunda pista da linha atual: a primeira é o marcador `>` na gutter, que é
// texto e portanto sobrevive a um terminal sem cor.
func currentLine(row visualRow, view View) *lipgloss.Style {
	if !row.IsCaret || !view.Theme.HasLineHighlight() {
		return nil
	}
	style := view.Theme.LineHighlight()
	return &style
}

// rowText is the row's own slice of the document line, with the roles that fall
// inside it.
func rowText(row visualRow, view View) Line {
	index := row.Line - view.Offset
	if index < 0 || index >= len(view.Lines) {
		return Line{Text: row.Text}
	}

	source := view.Lines[index]
	if row.Column == 0 && len(row.Text) == len(source.Text) {
		return source
	}
	return Line{Text: row.Text, Tokens: tokensInside(source.Tokens, row, view)}
}

// tokensInside clips the document line's tokens to the part this row shows.
//
// O recorte é por **offset de byte** relativo à linha, e não por coluna de
// escalar: a pintura trabalha em células, e converter duas vezes é a forma
// garantida de a coluna sair errada na primeira linha larga e certa na segunda.
func tokensInside(tokens []syntax.Token, row visualRow, view View) []syntax.Token {
	index := row.Line - view.Offset
	if index < 0 || index >= len(view.Lines) {
		return nil
	}
	line := view.Lines[index].Text
	from := layout.Offset(row.Column, line)
	to := from + len(row.Text)

	var out []syntax.Token
	for _, token := range tokens {
		start, end := max(token.Start, from), min(token.End, to)
		if start < end {
			out = append(out, syntax.Token{Role: token.Role, Start: start - from, End: end - from})
		}
	}
	return out
}

// paintTokens tints the columns the highlighter gave a role.
//
// Os offsets são de byte e o array é de células, e a conversão é feita nas
// fronteiras do token — um token pintado como corrida, e as duas medições não
// podem discordar de onde ele começa.
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

// classify decides what each column of the visual row is.
//
// A precedência é por célula: seleção > resultado da busca > papel de sintaxe.
// O que o leitor pediu vence o que o tema escolheu, e é decidir por célula — e não
// pela ordem em que os estilos são aplicados — que torna a precedência explícita.
func classify(line Line, row visualRow, width int, view View) []cell {
	kinds := make([]cell, width)
	paintTokens(kinds, line, width)

	for _, match := range matchesOn(row.Line, view.Matches) {
		mark(kinds, line, row, match.Start, match.End, cell{kind: cellMatch})
	}
	if view.CurrentMatch >= 0 && view.CurrentMatch < len(view.Matches) {
		current := view.Matches[view.CurrentMatch]
		if current.Start.Line <= row.Line && row.Line <= current.End.Line {
			mark(kinds, line, row, current.Start, current.End, cell{kind: cellCurrentMatch})
		}
	}

	if !view.Selection.Empty && !view.Selection.isCollapsed() && lineIsSelected(row.Line, view.Selection) {
		start, end := selectedColumns(row.Line, row, view.Selection)
		for column := start; column < end && column < width; column++ {
			kinds[column] = cell{kind: cellSelection}
		}
	}
	return kinds
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

// mark paints the columns of one document range that fall inside this row.
func mark(kinds []cell, line Line, row visualRow, start, end Position, kind cell) {
	from, to := selectedColumns(row.Line, row, Selection{Start: start, End: end})
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

// selectedColumns is the selected range inside one **visual row**, in cells.
//
// Quatro conversões encadeadas, e cada uma é medida: a seleção vem em coluna de
// escalar do documento, a linha visual ocupa um intervalo de escalares, e a
// pintura acontece em células. A interseção é feita em **escalares** — a única
// unidade em que as duas pontas são comparáveis — e só o resultado é convertido
// para célula.
//
// A primeira e a última linha da seleção são parciais; as do meio são cobertas
// inteiras. Uma coluna além do fim faria a conta do preenchimento dar largura
// negativa, e por isso a interseção satura.
func selectedColumns(lineNumber int, row visualRow, selection Selection) (start, end int) {
	const wholeLine = 1 << 30

	from, to := 0, wholeLine
	switch {
	case lineNumber < selection.Start.Line || lineNumber > selection.End.Line:
		return 0, 0
	case lineNumber == selection.Start.Line && lineNumber == selection.End.Line:
		from, to = selection.Start.Column, selection.End.Column
	case lineNumber == selection.Start.Line:
		from = selection.Start.Column
	case lineNumber == selection.End.Line:
		to = selection.End.Column
	}

	rowStart := row.Column
	rowEnd := rowStart + utf8.RuneCountInString(row.Text)
	from = min(max(from, rowStart), rowEnd)
	to = min(max(to, rowStart), rowEnd)

	return layout.Cells(row.Text, layout.Offset(from-rowStart, row.Text)),
		layout.Cells(row.Text, layout.Offset(to-rowStart, row.Text))
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
