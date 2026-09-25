package statusbar

import (
	"strconv"
	"strings"
	"testing"

	"github.com/ori-team/oride/internal/tui/layout"
)

func exact(t *testing.T, name, row string, width int) {
	t.Helper()
	if got := layout.Width(row); got != width {
		t.Errorf("%s: %d células, esperado %d em %q", name, got, width, row)
	}
}

func TestExactWidth(t *testing.T) {
	views := []View{
		{},
		{File: "a.go", Dirty: true, Focus: "editor", CaretLine: 12, CaretCol: 4, HasCaret: true},
		{File: strings.Repeat("pasta/", 20) + "a.go", Message: strings.Repeat("mensagem ", 10), HasCaret: true},
	}
	for index, view := range views {
		for _, width := range []int{10, 40, 80, 200} {
			exact(t, "caso "+strconv.Itoa(index), Render(width, view), width)
		}
	}
}

// TestKeepsTheCaretPosition: the position is the field read by location, so it
// survives truncation of everything else.
func TestKeepsTheCaretPosition(t *testing.T) {
	view := View{
		File:    strings.Repeat("muito/longo/", 20) + "arquivo.go",
		Message: strings.Repeat("aviso ", 20), Focus: "editor",
		CaretLine: 7, CaretCol: 3, HasCaret: true,
	}

	if row := Render(60, view); !strings.Contains(row, "Ln 7, Col 3") {
		t.Errorf("a posição do cursor não sobreviveu: %q", row)
	}
}

// TestMarksDirtyInText: no state is carried by colour alone.
func TestMarksDirtyInText(t *testing.T) {
	dirty := Render(40, View{File: "a.go", Dirty: true})
	clean := Render(40, View{File: "a.go"})

	if !strings.Contains(dirty, dirtyMark) || strings.Contains(clean, dirtyMark) {
		t.Errorf("sujo=%q limpo=%q", dirty, clean)
	}
}

// TestNoDocumentSaysUntitled.
func TestNoDocumentSaysUntitled(t *testing.T) {
	if row := Render(40, View{}); !strings.Contains(row, untitled) {
		t.Errorf("sem documento sem aviso: %q", row)
	}
}

// TestNoCaretOmitsThePosition: a bar claiming "Ln 1, Col 1" with nothing open
// would be describing a cursor that does not exist.
func TestNoCaretOmitsThePosition(t *testing.T) {
	if row := Render(60, View{File: "a.go"}); strings.Contains(row, "Ln") {
		t.Errorf("posição exibida sem cursor: %q", row)
	}
}

// A barra de status como **superfície de informação** é o item 3 da direção, e
// estes testes fixam as duas metades: o que ela mostra e — o mais importante — o
// que ela **se recusa** a mostrar.

// TestTheBarShowsWhatTheReaderAsksFor cobre os quatro campos que o Zed e o Lapce
// mostram e que o Oride tinha guardados: modo da linguagem, branch, distância do
// upstream e estado do LSP.
func TestTheBarShowsWhatTheReaderAsksFor(t *testing.T) {
	view := View{
		File:        "main.go",
		Mode:        "Go",
		Branch:      "main",
		Ahead:       2,
		Behind:      1,
		HasUpstream: true,
		LSP:         "LSP off",
		HasCaret:    true,
		CaretLine:   4,
		CaretCol:    8,
	}

	bar := Render(100, view)

	for _, want := range []string{"main.go", "Go", "● main", "↑2", "↓1", "LSP off", "Ln 4, Col 8"} {
		if !strings.Contains(bar, want) {
			t.Errorf("a barra não mostra %q: %q", want, bar)
		}
	}
}

// TestTheBarNeverClaimsAHealthNobodyMeasured é a metade que importa: com o LSP
// desligado, um "0 erros" é ausência de medição apresentada como saúde.
func TestTheBarNeverClaimsAHealthNobodyMeasured(t *testing.T) {
	view := View{File: "a.go", HasCaret: true, Diagnostics: 0, HasDiagnostics: false}

	if bar := Render(80, view); strings.Contains(bar, "erro") {
		t.Errorf("a barra affirmou um número de erros sem cliente: %q", bar)
	}
}

// TestTheErrorCountAppearsOnlyWhenItWasMeasured é a outra face do mesmo: com
// cliente, o número é informação, e escondê-lo seria perder o dado que a barra
// existe para dar.
func TestTheErrorCountAppearsOnlyWhenItWasMeasured(t *testing.T) {
	one := Render(80, View{File: "a.go", HasDiagnostics: true, Diagnostics: 1})
	if !strings.Contains(one, "1 erro") {
		t.Errorf("um erro medido não apareceu: %q", one)
	}

	many := Render(80, View{File: "a.go", HasDiagnostics: true, Diagnostics: 3})
	if !strings.Contains(many, "3 erros") {
		t.Errorf("três erros medidos não apareceram: %q", many)
	}
}

// TestTheBranchIsVisibleWithoutACaret: um documento novo não pode esvaziar a
// barra — a branch e o LSP continuam sendo o que o leitor procura.
func TestTheBranchIsVisibleWithoutACaret(t *testing.T) {
	bar := Render(80, View{File: "a.go", Branch: "feature", HasUpstream: false})

	if !strings.Contains(bar, "feature") {
		t.Errorf("a branch sumiu sem caret: %q", bar)
	}
	if strings.Contains(bar, "↑") || strings.Contains(bar, "↓") {
		t.Errorf("a barra inventou uma distância de upstream que ninguém leu: %q", bar)
	}
}

// TestTheCaretKeepsItsPlaceAtTheRightEdge é a razão de ele ser o último: é o
// campo lido por posição, e um valor que anda é um valor que passa batido.
func TestTheCaretKeepsItsPlaceAtTheRightEdge(t *testing.T) {
	view := View{File: "a.go", Branch: "main", HasUpstream: true, HasCaret: true, CaretLine: 1, CaretCol: 1}
	bar := Render(100, view)

	if !strings.HasSuffix(strings.TrimRight(stripANSI(bar), " "), "Ln 1, Col 1") {
		t.Errorf("o caret não ficou na borda: %q", bar)
	}
}

// stripANSI removes escape sequences, para comparar o **texto** da barra e não a
// sua codificação: um teste que compara bytes com a cor no meio passa com a
// produto quebrado e falha com a paleta trocada.
func stripANSI(text string) string {
	var out strings.Builder
	for index := 0; index < len(text); index++ {
		if text[index] != 0x1b {
			out.WriteByte(text[index])
			continue
		}
		for index < len(text) && text[index] != 'm' {
			index++
		}
	}
	return out.String()
}
