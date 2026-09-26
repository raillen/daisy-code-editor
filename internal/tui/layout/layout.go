// Package layout computes where each surface goes, and how text fits inside it.
//
// It is pure geometry: no terminal, no Bubble Tea, no model. The model says what
// it wants shown; this package answers where it lands. Keeping it separate means
// the layout rules in docs/ui-ux/layout.md are testable at every width without
// rendering a frame.
package layout

import (
	"strings"

	"github.com/clipperhouse/displaywidth"
)

// WidthClass is how much room the terminal offers.
type WidthClass int

// The classes. Boundaries are inclusive at the bottom: 80 is Standard, 120 is
// Wide. Stated once, because an off-by-one here moves every surface on screen.
const (
	Compact WidthClass = iota
	Standard
	Wide
)

// CompactMax and StandardMax are the last widths of their class.
const (
	CompactMax  = 79
	StandardMax = 119
)

// Classify reports the class of a terminal width.
func Classify(width int) WidthClass {
	switch {
	case width <= CompactMax:
		return Compact
	case width <= StandardMax:
		return Standard
	default:
		return Wide
	}
}

// Size is a terminal size, in cells.
type Size struct {
	Width  int
	Height int
}

// Region is a rectangle, in cells.
type Region struct {
	X      int
	Y      int
	Width  int
	Height int
}

// Empty reports a region with no area.
func (r Region) Empty() bool { return r.Width <= 0 || r.Height <= 0 }

// Contains reports whether a cell is inside the region, e é o que decide o que um
// clique acertou.
//
// A comparação é em **células da tela**: um clique numa linha em branco acima da
// última linha real ainda é na região, e dizer que não é faria o clique no vazio
// não fazer nada — que é o que se espera de um clique no vazio.
func (r Region) Contains(x, y int) bool {
	return !r.Empty() && x >= r.X && x < r.X+r.Width && y >= r.Y && y < r.Y+r.Height
}

// Wants is what the model asks to show.
//
// A value rather than a set of booleans threaded through the call: four
// parameters of which three are booleans is the arity the contract calls a
// smell, and this keeps the layout input readable at the call site.
type Wants struct {
	ShowTree  bool
	TreeWidth int
	// ShowBottom asks for the bottom panel — Terminal, Problemas e Busca em um
	// lugar, que é o modelo do Lapce e do Zed.
	//
	// Uma região só, com **abas**, em vez de duas superfícies (o terminal embaixo e
	// a barra de busca acima) disputando as mesmas linhas: o leitor abre a busca e
	// o terminal some, ou vice-versa, e nenhuma das duas é o que ele pediu.
	ShowBottom bool
	// BottomHeight is how many rows the panel takes, **aba inclusive**: a faixa de
	// abas come uma linha, e é ela que diz o que o painel está mostrando.
	BottomHeight int
}

// Regions is where every surface landed for one frame.
type Regions struct {
	Class   WidthClass
	MenuBar Region
	Tabs    Region
	Tree    Region
	Editor  Region
	// Bottom is the tabbed panel: Terminal │ Problemas │ Busca.
	Bottom    Region
	StatusBar Region
	// TreeOverlaid says the tree floats over the editor instead of sitting
	// beside it, which is what the compact class does.
	TreeOverlaid bool
}

// minEditorWidth is the narrowest the editor may become before a side panel is
// dropped. A tree that pushes the editor below this has stopped being useful.
const minEditorWidth = 40

// Compute places every surface for a terminal of the given size.
//
// The stack is fixed — menu bar, tabs, body, status bar — and the body splits
// horizontally. A zero or negative size is not an error: a terminal that has not
// been measured yet must produce empty regions rather than a panic or a negative
// width that renders as garbage.
func Compute(size Size, wants Wants) Regions {
	regions := Regions{Class: Classify(size.Width)}
	if size.Width <= 0 || size.Height <= 0 {
		return regions
	}

	y := 0
	regions.MenuBar, y = row(y, size.Width, size.Height)
	regions.Tabs, y = row(y, size.Width, size.Height)

	// The status bar is reserved before the body, so the body never has to know
	// whether it is the last surface.
	regions.StatusBar = Region{X: 0, Y: size.Height - 1, Width: size.Width, Height: 1}

	bodyHeight := size.Height - y - regions.StatusBar.Height
	if bodyHeight < 0 {
		bodyHeight = 0
	}
	body := Region{X: 0, Y: y, Width: size.Width, Height: bodyHeight}

	// O painel inferior fica entre o editor e a barra de status, e **cresce para
	// cima**: abrir a busca tira linhas do editor em vez de cobrir o texto que
	// está sendo procurado.
	if wants.ShowBottom && wants.BottomHeight > 0 && body.Height > wants.BottomHeight {
		regions.Bottom = Region{
			X:      0,
			Y:      body.Y + body.Height - wants.BottomHeight,
			Width:  body.Width,
			Height: wants.BottomHeight,
		}
		body.Height -= wants.BottomHeight
	}

	regions.Tree, regions.Editor, regions.TreeOverlaid = splitBody(body, wants)
	return regions
}

// row reserves one line at the given offset, clamped to the terminal.
func row(y, width, height int) (Region, int) {
	if y >= height {
		return Region{}, y
	}
	return Region{X: 0, Y: y, Width: width, Height: 1}, y + 1
}

// splitBody divides the body between the tree and the editor.
//
// The tree gives way before the editor does: below the compact boundary it
// overlays instead of taking a column, and in any class it is dropped if the
// editor would fall under the minimum. An editor squeezed to a dozen columns is
// not a smaller editor, it is a broken one.
func splitBody(body Region, wants Wants) (tree, editor Region, overlaid bool) {
	editor = body

	if !wants.ShowTree || wants.TreeWidth <= 0 {
		return Region{}, editor, false
	}

	if Classify(body.Width) == Compact || body.Width-wants.TreeWidth < minEditorWidth {
		// Overlaid: the tree floats over the editor's top-left corner, so the
		// editor keeps its full width underneath.
		return Region{
			X:      body.X,
			Y:      body.Y,
			Width:  min(wants.TreeWidth, body.Width),
			Height: min(body.Height, overlaidTreeHeight),
		}, editor, true
	}

	tree = Region{X: body.X, Y: body.Y, Width: wants.TreeWidth, Height: body.Height}
	editor = Region{
		X:      body.X + wants.TreeWidth,
		Y:      body.Y,
		Width:  body.Width - wants.TreeWidth,
		Height: body.Height,
	}
	return tree, editor, false
}

// overlaidTreeHeight bounds the floating tree so it cannot cover the whole
// editor on a short terminal.
const overlaidTreeHeight = 12

// measure is the single authority for how wide a string is on screen.
//
// It is the engine the `bubbles` components measure with — `clipperhouse/
// displaywidth` — configurado uma vez, e não uma segunda implementação: três
// medidas convivendo é o terreno exato do sexto bug de largura, que é a classe de
// bug mais frequente deste projeto. As duas decisões são o contrário do que este
// pacote fazia antes, e as duas são deliberadas:
//
//   - **Ambíguo conta como estreito.** A regra antiga era o contrário, na
//     intenção de errar para o lado largo. A medição mostrou o custo: a tabela
//     ambígua do `runewidth` inclui `à á è é ê ì í ò ó ù ú ü` e **não** inclui
//     `ã õ â ç` — ou seja, num produto em português, "são" mediava quatro células
//     num terminal que desenha três. Errar para o largo não custava só uma
//     coluna: desincronizava esta medição da que os componentes usam, e o
//     resultado era a linha do editor mais larga que a da árvore ao lado.
//   - **Sequência de escape não ocupa célula.** O próprio `displaywidth` sabe
//     disso quandoconfigured; sem isso, estilizar uma linha mudaria a largura
//     medida e tudo à direita deslizaria.
//
// A menor unidade de exibição é o **cluster de grafema**, não o rune: um acento
// combinado, um emoji com ZWJ e uma bandeira são um caractere na tela e varios
// runes no texto. Por isso `Slice`, `Truncate` e `Cells` também percorrem
// clusters — medir por rune é a forma curta de errar.
var measure = displaywidth.Options{
	EastAsianWidth:       false,
	ControlSequences:     true,
	ControlSequences8Bit: true,
}

// Width measures a string in terminal cells.
//
// O caminho rápido é ASCII puro, e é por isso que ele existe: toda linha de todo
// frame é medida, e a maioria não tem nada não-ASCII.
func Width(text string) int {
	if isPrintableASCII(text) {
		return len(text)
	}
	return measure.String(text)
}

// isPrintableASCII is the fast path's whole condition: a string of bytes that each
// print as exactly one cell needs no cluster walk and no escape scan.
func isPrintableASCII(text string) bool {
	for index := range len(text) {
		if text[index] < 0x20 || text[index] > 0x7e {
			return false
		}
	}
	return true
}

// Cells converts a byte offset into a terminal cell column.
//
// The document counts scalars and bytes; the layout counts cells, and the two
// are the same distance only while every character is one cell wide. This is the
// one conversion between them, and it exists because a selection, a match or a
// syntax role that began after a wide character was painted a column short —
// silently, and only on the lines that had a wide character before it.
//
// Clusters, não runes: um offset que cai dentro de um cluster arredonda para o
// começo dele, porque a outra escolha pintaria meia caractere à esquerda. Um
// offset além do fim satura na largura do texto.
//
// The text must be plain. A byte offset into a string carrying ANSI is an offset
// into the escape sequence, which is a different thing from a position.
func Cells(text string, byteOffset int) int {
	if byteOffset <= 0 {
		return 0
	}
	if byteOffset > len(text) {
		byteOffset = len(text)
	}

	column := 0
	for _, piece := range clusters(text) {
		if piece.end > byteOffset {
			// O offset caiu dentro do cluster: a coluna é a do começo dele, e a
			// escolha de arredondar para baixo é o que mantém o realce dentro do
			// texto em vez de meio caractere à esquerda.
			break
		}
		column += piece.cells
	}
	return column
}

// Offset converts a scalar column into a byte offset in the same text.
//
// A inversa de `Cells`, e existe porque as duas conversões usam índices
// diferentes — byte e escalar — e uma linha quebrada precisa dos dois. Uma função
// só para cada direção impede que alguém reintroduza a conta dentro da
// superfície, que é onde ela estava antes de o wrap existir.
func Offset(column int, text string) int {
	if column <= 0 {
		return 0
	}
	seen := 0
	for offset := range text {
		if seen == column {
			return offset
		}
		seen++
	}
	return len(text)
}

// cluster is one grapheme cluster, with where it is and how wide it draws.
type cluster struct {
	text  string
	cells int
	start int
	end   int
}

// clusters walks the grapheme clusters of a string.
//
// A lista é materializada porque três das funções acima precisam de dois passes
// (medir e depois cortar) e because medir o texto inteiro para descobrir que ele
// cabe é o caso comum.
func clusters(text string) []cluster {
	graphemes := measure.StringGraphemes(text)
	out := make([]cluster, 0, len(text))

	offset := 0
	for graphemes.Next() {
		value := graphemes.Value()
		out = append(out, cluster{
			text:  value,
			cells: graphemes.Width(),
			start: offset,
			end:   offset + len(value),
		})
		offset += len(value)
	}
	return out
}

// Truncate cuts text to fit width cells, marking that it was cut.
//
// O marcador não é decoração: um rótulo que desaparece sem aviso é indistinguível
// de um rótulo que nunca existiu, e o leitor não tem como distinguir um valor
// curto de um valor cortado.
func Truncate(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if Width(text) <= width {
		return text
	}

	// A largura do marcador é medida, nunca assumida: um "…" contado como duas
	// células transbordava a linha em uma coluna, e foi um bug real aqui.
	marker := ellipsis
	limit := width - Width(marker)
	return prefixCells(text, limit) + marker
}

// ellipsis is the cut marker. It is one cell wide, which is why it fits a
// one-column field — the fallback that existed when this package counted it as
// two has no condition left to fire.
const ellipsis = "…"

// Slice extracts width cells from text, starting at a cell offset.
//
// Cell arithmetic, not byte arithmetic: the layout measured in cells, and taking
// a substring by byte is how a row with a wide character in it ends up one column
// short. It lives here because three callers needed it — the list overlay, the
// composition root's overlay painting, and the editor's selection.
//
// Por cluster, e nunca por rune: cortar no meio de um cluster produz um caractere
// que o terminal não sabe desenhar, e um emoji com ZWJ cortado ocupa uma célula
// que não existe.
func Slice(text string, from, width int) string {
	if width <= 0 {
		return ""
	}

	// Aritmética em **células**, do começo ao fim: `from` e `width` são colunas,
	// e comparar uma coluna com o deslocamento de byte de um cluster é exatamente
	// a troca de unidade que produz o bug.
	skipped, taken := 0, 0
	var out strings.Builder
	for _, piece := range clusters(text) {
		switch {
		case skipped+piece.cells <= from:
			skipped += piece.cells
		case skipped < from:
			// O cluster começa antes da coluna pedida e termina depois dela: a
			// coluna está dentro do cluster, e o que se devolve é o cluster
			// inteiro — a metade dele não é um caractere.
			return piece.text
		case taken+piece.cells > width:
			return out.String()
		default:
			out.WriteString(piece.text)
			taken += piece.cells
		}
	}
	return out.String()
}

// prefixCells is the longest prefix of text that draws at most width cells, cut at
// a cluster boundary.
func prefixCells(text string, width int) string {
	if width <= 0 {
		return ""
	}

	used := 0
	var out strings.Builder
	for _, piece := range clusters(text) {
		if used+piece.cells > width {
			break
		}
		out.WriteString(piece.text)
		used += piece.cells
	}
	return out.String()
}

// Pad fits text into exactly width cells, truncating when needed.
//
// Exactly, not at most: a row that comes back short lets whatever is to its
// right shift left, which is how a two-column layout turns into a diagonal one.
func Pad(text string, width int) string {
	if width <= 0 {
		return ""
	}

	fitted := Truncate(text, width)
	if padding := width - Width(fitted); padding > 0 {
		return fitted + strings.Repeat(" ", padding)
	}
	return fitted
}
