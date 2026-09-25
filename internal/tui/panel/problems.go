package panel

import (
	"strconv"

	"github.com/ori-team/oride/internal/tui/layout"
)

// ProblemsEmptyLines é o que a aba de problemas mostra quando não há lista, já
// com a largura da região.
//
// E o texto é a metade do trabalho: uma lista vazia sem legenda é um programa
// quebrado, e um "0 problemas" com o LSP desligado é uma saúde que ninguém mediu.
// Por isso a aba **nomeia a fonte** e diz o que fazer.
func ProblemsEmptyLines(width int, lspEnabled, lspAttached bool) []string {
	return []string{layout.Pad(ProblemsEmpty(width, lspEnabled, lspAttached), width)}
}

// ProblemsEmpty é o texto que a aba de problemas mostra quando não há lista.
func ProblemsEmpty(width int, lspEnabled, lspAttached bool) string {
	switch {
	case lspAttached:
		return " nenhum problema"
	case lspEnabled:
		// A lacuna real: ligado no `config.toml` e sem cliente. Dizer na tela
		// resolve a mentira, mas repetir um texto fixo em cada frame vira ruído —
		// por isso o texto é curto e a dívida é do estado e do ledger.
		return " LSP desligado · cliente não iniciado"
	default:
		return " LSP desligado · ligue em [lsp].enabled"
	}
}

// ProblemsRows desenha a lista de problemas, uma por linha.
//
// A lista é `arquivo:linha: coluna · mensagem`, que é o formato que o reader
// copia para o editor — e o mesmo que o LSP entrega, para não haver duas
// gramáticas do mesmo dado.
func ProblemsRows(entries []Problem, width, height int) []string {
	rows := make([]string, 0, max(height, 1))
	if len(entries) > 0 {
		rows = append(rows, layout.Pad(problemsIntro, width))
	}
	for index, entry := range entries {
		if index >= height {
			break
		}
		line := entry.File + ":" + strconv.Itoa(entry.Line) + ":" + strconv.Itoa(entry.Column) + " · " + entry.Message
		rows = append(rows, layout.Pad(line, width))
	}
	return rows
}

// Problem is one diagnostic.
type Problem struct {
	File    string
	Line    int
	Column  int
	Message string
}

// problemsIntro é a legenda da lista, e ela também é a que diz a fonte do dado.
const problemsIntro = " arquivo:linha:coluna · mensagem"
