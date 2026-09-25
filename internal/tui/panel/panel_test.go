package panel

import (
	"strings"
	"testing"

	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/theme"
)

// plainTheme é um tema sem cor, que é o que a maioria destes testes quer: a
// diferença entre texto e sequência de escape é medida em outro teste.
func plainTheme() theme.Theme {
	shipped := config.Default()
	return theme.New(shipped.UI, shipped.Syntax, theme.NoColor)
}

func view(active Tab, content ...string) View {
	return View{
		Tabs:    AllTabs(),
		Active:  active,
		Content: content,
		Theme:   plainTheme(),
	}
}

// TestTheStripNamesEveryTab is a primeira coisa que o leitor procura: o que existe
// e o que está à frente.
func TestTheStripNamesEveryTab(t *testing.T) {
	row := Render(60, 4, view(TabTerminal))[0]

	for _, want := range []string{"Terminal", "Problemas", "Busca"} {
		if !strings.Contains(row, want) {
			t.Errorf("a faixa não nomeia %q: %q", want, row)
		}
	}
	if !strings.Contains(row, "▍Terminal") {
		t.Errorf("a aba ativa não está marcada em texto: %q", row)
	}
	if strings.Contains(row, "▍Problemas") || strings.Contains(row, "▍Busca") {
		t.Errorf("mais de uma aba aparece marcada: %q", row)
	}
}

// TestTheActiveTabIsMarkedInText is a regra do projeto aplicada à faixa: a aba
// ativa é marcada com `▍` e **não** só com cor.
func TestTheActiveTabIsMarkedInText(t *testing.T) {
	row := Render(60, 4, view(TabSearch))[0]

	if !strings.Contains(row, "▍Busca") {
		t.Errorf("a aba de busca não está marcada: %q", row)
	}
	if strings.Contains(row, "▍Terminal") {
		t.Errorf("a aba do terminal continua marcada: %q", row)
	}
}

// TestTheCountAppearsOnlyWhenMeasured é a honestidade da aba de problemas: com o
// LSP desligado, um "0 erros" é uma saúde que ninguém mediu.
func TestTheCountAppearsOnlyWhenMeasured(t *testing.T) {
	silent := view(TabProblems)
	if row := Render(60, 4, silent)[0]; strings.Contains(row, "erro") {
		t.Errorf("a faixa affirmou uma contagem sem cliente: %q", row)
	}

	measured := view(TabProblems)
	measured.HasProblems = true
	measured.ProblemCount = 2
	if row := Render(60, 4, measured)[0]; !strings.Contains(row, "2 erros") {
		t.Errorf("a contagem medida não apareceu: %q", row)
	}
}

// TestEveryRowHasTheExactWidth é o invariante que nenhuma outra regra substitui:
// um painel com uma linha curta move o que está à direita dele.
func TestEveryRowHasTheExactWidth(t *testing.T) {
	full := view(TabProblems, ProblemsEmptyLines(60, false, false)...)

	for width := range []int{12, 40, 100} {
		for index, row := range Render(width, 5, full) {
			if got := layout.Width(row); got != width {
				t.Errorf("largura %d, linha %d com %d células: %q", width, index, got, row)
			}
		}
	}
}

// TestAPanelShorterThanItsStripIsEmpty: um painel de uma linha é a faixa, e a
// faixa sem conteúdo é o que o leitor vê quando a altura pedida é curta demais.
func TestAPanelShorterThanItsStripIsEmpty(t *testing.T) {
	rows := Render(40, 1, view(TabTerminal, "conteúdo"))

	if len(rows) != 1 {
		t.Fatalf("um painel de uma linha devolveu %d", len(rows))
	}
	if !strings.Contains(rows[0], "Terminal") {
		t.Errorf("a faixa sumiu: %q", rows[0])
	}
}

// TestProblemsNamesItsSource is a outra metade da honestidade: a lista vazia **diz
// de onde viria** e o que fazer, em vez de oferecer um "0 problemas".
func TestProblemsNamesItsSource(t *testing.T) {
	off := ProblemsEmpty(80, false, false)
	if !strings.Contains(off, "LSP desligado") || !strings.Contains(off, "lsp") {
		t.Errorf("sem LSP, o texto não diz o que ligar: %q", off)
	}

	enabled := ProblemsEmpty(80, true, false)
	if !strings.Contains(enabled, "cliente") {
		t.Errorf("com LSP ligado e sem cliente, o texto não diz a lacuna real: %q", enabled)
	}

	attached := ProblemsEmpty(80, true, true)
	if !strings.Contains(attached, "nenhum problema") {
		t.Errorf("com cliente anexado, o texto devia dizer que está medido: %q", attached)
	}
}

// TestProblemsRowsAreTheFormatTheReaderCopies fixa a gramática: `arquivo:linha:
// coluna · mensagem`, que é o que o LSP entrega e o que se copia para o editor.
func TestProblemsRowsAreTheFormatTheReaderCopies(t *testing.T) {
	rows := ProblemsRows([]Problem{
		{File: "main.go", Line: 42, Column: 8, Message: "variável não usada"},
		{File: "a/b.rs", Line: 1, Column: 1, Message: "esperado `;`"},
	}, 60, 4)

	joined := strings.Join(rows, "\n")
	for _, want := range []string{"main.go:42:8 · variável não usada", "a/b.rs:1:1 · esperado `;`"} {
		if !strings.Contains(joined, want) {
			t.Errorf("a lista não tem %q:\n%s", want, joined)
		}
	}
}
