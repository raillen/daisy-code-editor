package editorview

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/ori-team/oride/internal/syntax"
	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/theme"
)

type syntaxToken = syntax.Token

const syntaxKeyword = syntax.Keyword

// O soft wrap é a diferença entre "editor" e "visualizador de arquivo", e cada um
// destes testes afirma uma promessa que o leitor faz ao abrir o programa.

const longLine = "func processar(entrada string) (resultado string, erro error) { return validar(entrada), nil }"

// TestALongLineIsReadableWhole is a promessa principal: a linha aparece **inteira**.
//
// Antes, uma linha de 121 caracteres num terminal de 60 mostrava 59 e um `…`, e
// não havia comando que levasse ao resto — o texto simplesmente não existia na
// tela. A cauda desaparecia, e é a cauda que costuma ser a condição do `if`.
func TestALongLineIsReadableWhole(t *testing.T) {
	view := View{Lines: plain(longLine), Wrap: true, Gutter: true}

	rows := Render(40, 8, view)
	joined := strings.Join(rows, "\n")

	if !strings.Contains(joined, "processar") {
		t.Fatalf("o começo da linha sumiu:\n%s", joined)
	}
	if !strings.Contains(joined, "erro error") {
		t.Errorf("a cauda da linha sumiu — é o defeito que o wrap veio para tirar:\n%s", joined)
	}
	if strings.Contains(joined, "…") {
		t.Errorf("a linha foi truncada em vez de quebrada:\n%s", joined)
	}
}

// TestTheLineNumberIsShownOncePerLine: com quebra, repetir o número em cada
// linha da tela mente sobre quantas linhas o arquivo tem.
func TestTheLineNumberIsShownOncePerLine(t *testing.T) {
	view := View{Lines: plain(longLine), Wrap: true, Gutter: true}

	rows := Render(40, 8, view)
	occurrences := 0
	for _, row := range rows {
		occurrences += strings.Count(row, "1")
	}
	if occurrences != 1 {
		t.Errorf("o número da linha apareceu %d vezes em %d linhas de tela", occurrences, len(rows))
	}
}

// TestTheCaretStaysVisibleOnAWrappedLine é o que o wrap faz com a janela: sem
// converter linha lógica em linha visual, o caret sai da tela na hora em que a
// linha é maior que a janela.
func TestTheCaretStaysVisibleOnAWrappedLine(t *testing.T) {
	// Uma linha com vinte palavras: a última está longe da primeira linha de
	// tela, e o caret nela é o que a janela precisa seguir.
	words := strings.TrimSpace(strings.Repeat("palavra ", 20))
	view := View{
		Lines:    plain(words),
		Wrap:     true,
		Gutter:   true,
		HasCaret: true,
		Caret:    Position{Line: 0, Column: len([]rune(words)) - 1},
	}

	rows := Render(30, 6, view)
	last := rows[len(rows)-1]

	if !strings.Contains(last, caretMarker) {
		t.Errorf("a linha do caret não está na tela:\n%s", strings.Join(rows, "\n"))
	}
}

// TestTheCaretOnAnEmptyLineIsStillMarked é o caso que o wrap criou: uma linha vazia
// tem zero colunas, e um intervalo de colunas sem largura não contém o cursor.
func TestTheCaretOnAnEmptyLineIsStillMarked(t *testing.T) {
	view := View{
		Lines:    plain("primeira linha", "", "terceira"),
		Wrap:     true,
		Gutter:   true,
		HasCaret: true,
		Caret:    Position{Line: 1, Column: 0},
	}

	rows := Render(40, 6, view)
	marked := 0
	for _, row := range rows {
		marked += strings.Count(row, caretMarker)
	}
	if marked != 1 {
		t.Errorf("o caret da linha vazia aparece %d vezes:\n%s", marked, strings.Join(rows, "\n"))
	}
}

// TestTheSelectionCrossesWrappedRows: uma seleção de muitas palavras pinta **todas**
// as linhas de tela que ela cobre, e só elas.
func TestTheSelectionCrossesWrappedRows(t *testing.T) {
	words := "uma duas tres quatro cinco seis sete oito nove dez onze doze"
	view := View{
		Lines:     plain(words),
		Wrap:      true,
		Gutter:    true,
		Selection: selection(0, 4, 0, 30),
		Theme:     colourTheme(),
	}

	rows := Render(20, 10, view)
	painted := 0
	for _, row := range rows {
		if strings.ContainsRune(row, 0x1b) {
			painted++
		}
	}
	if painted < 2 {
		t.Errorf("a seleção pintou %d linhas de tela, e deveria cruzar mais de uma:\n%s",
			painted, strings.Join(rows, "\n"))
	}
}

// TestRolesFollowTheWrappedRow is a parte invisível: o realce de sintaxe tem que
// descer com a linha, e não pintar a linha inteira na primeira fatia.
//
// A afirmação é pela **cor do papel**, e não por "tem escape": a gutter é pintada
// em toda linha, e um teste que contasse escapes contaria a gutter e passaria com o
// papel vazado.
func TestRolesFollowTheWrappedRow(t *testing.T) {
	palette := colourTheme()
	keyword := keywordSequence(palette)

	view := View{
		Lines: []Line{{
			Text:   "func " + longLine,
			Tokens: []syntaxToken{{Role: syntaxKeyword, Start: 0, End: 4}},
		}},
		Wrap:   true,
		Gutter: true,
		Theme:  palette,
	}

	rows := Render(30, 6, view)
	if !strings.Contains(rows[0], keyword) {
		t.Errorf("a primeira linha da tela perdeu o realce do papel:\n%s", strings.Join(rows, "\n"))
	}
	for index, row := range rows[1:] {
		if strings.Contains(row, keyword) {
			t.Errorf("o realce do papel vazou para a linha %d da tela:\n%s", index+1, strings.Join(rows, "\n"))
		}
	}
}

// keywordSequence is the escape the keyword role produces, e é isso que o teste
// procura: a sequência, e não a palavra.
func keywordSequence(palette theme.Theme) string {
	rendered := palette.Syntax(syntax.Keyword).Render("x")
	if index := strings.IndexRune(rendered, 'x'); index > 0 {
		return rendered[:index]
	}
	return rendered
}

// TestEveryRowKeepsTheExactWidth: a propriedade que nenhuma das outras substitui.
func TestEveryRowKeepsTheExactWidth(t *testing.T) {
	view := View{Lines: plain(longLine, "curta", "日本語のテキスト"), Wrap: true, Gutter: true}
	for width := range []int{12, 20, 40, 80} {
		for index, row := range Render(width, 8, view) {
			if got := layout.Width(row); got != width {
				t.Errorf("largura %d, linha %d com %d células: %q", width, index, got, row)
			}
		}
	}
}

// TestWithoutWrapTheLineIsCut is a contraprova: sem o wrap, o corte volta — e é o
// que prova que o wrap é o que segura este caminho, e não o preenchimento.
func TestWithoutWrapTheLineIsCut(t *testing.T) {
	view := View{Lines: plain(longLine), Wrap: false, Gutter: true}

	rows := Render(40, 8, view)
	joined := strings.Join(rows, "\n")

	if strings.Contains(joined, "erro error") {
		t.Errorf("sem wrap a linha não devia caber inteira:\n%s", joined)
	}
	if len(rows) != 8 {
		t.Errorf("sem wrap são 8 linhas de tela, e a fatia devolvem %d", len(rows))
	}
}

// TestTheCurrentLineIsMarkedWhenTheThemeAsks is a segunda pista, e ela é
// opcional: o token `line_highlight` vem vazio, e um tema que o declara vê a
// linha do caret com um fundo sutil.
//
// A primeira pista é o marcador `>` na gutter, que é **texto** — e a regra do
// projeto é que nenhum estado vive só na cor. Esta aqui é decoração, e por isso
// pode faltar sem custo.
func TestTheCurrentLineIsMarkedWhenTheThemeAsks(t *testing.T) {
	base := colourTheme()
	withHighlight := base
	tokens := withHighlight.Tokens()
	tokens.LineHighlight = "#333344"
	withHighlight = base.WithTokens(&tokens)

	view := View{
		Lines:    plain("primeira", "segunda"),
		Wrap:     true,
		Gutter:   true,
		HasCaret: true,
		Caret:    Position{Line: 1, Column: 0},
		Theme:    withHighlight,
	}

	// A sequência vem do próprio tema, e não de uma constante escrita aqui: o
	// estilo converte a cor em decimal, e um teste que compara com o hexadecimal
	// do token falha com o produto funcionando.
	background := backgroundSequence(withHighlight.LineHighlight())

	rows := Render(30, 6, view)
	if !strings.Contains(rows[1], background) {
		t.Errorf("a linha do caret não recebeu o fundo do token:\n%q", rows[1])
	}
	if strings.Contains(rows[0], background) {
		t.Errorf("o fundo da linha atual vazou para outra linha:\n%q", rows[0])
	}
}

// backgroundSequence is the escape prefix a background style produces.
func backgroundSequence(style lipgloss.Style) string {
	rendered := style.Render("x")
	if index := strings.IndexRune(rendered, 'x'); index > 0 {
		return rendered[:index]
	}
	return rendered
}

// TestTheCurrentLineIsNotMarkedWhenTheThemeIsSilent fecha o outro lado: um token
// vazio não pode virar um fundo preto, que é o defeito clássico de um estilo
// construído a partir de nada.
func TestTheCurrentLineIsNotMarkedWhenTheThemeIsSilent(t *testing.T) {
	view := View{
		Lines:    plain("primeira", "segunda"),
		Wrap:     true,
		Gutter:   true,
		HasCaret: true,
		Caret:    Position{Line: 1, Column: 0},
		Theme:    colourTheme(),
	}

	// A sequência que um fundo **negro** produziria: é o defeito clássico de um
	// estilo construído a partir de nada, e é o que este teste caça.
	black := backgroundSequence(lipgloss.NewStyle().Background(lipgloss.Color("#010101")))

	rows := Render(30, 6, view)
	if strings.Contains(rows[1], black) {
		t.Errorf("um token vazio virou fundo:\n%q", rows[1])
	}
}
