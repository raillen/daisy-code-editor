// Package panel renders the tabbed bottom panel: Terminal │ Problems │ Search.
//
// One region, three contents, one row of tabs saying which. The alternative —
// the terminal below and the find bar above, each in its own region — is how the
// product was, and it has a failure mode: opening the search makes the terminal
// disappear, and the reader who wanted the terminal lost it to the keystroke he
// did not mean as a switch.
package panel

import (
	"strconv"
	"strings"

	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/theme"
)

// Tab is one of the panel's contents.
type Tab string

// The tabs, na ordem em que aparecem — que é a ordem em que o leitor as procura.
const (
	TabTerminal Tab = "terminal"
	TabProblems Tab = "problems"
	TabSearch   Tab = "search"
)

// AllTabs is the strip's order, and a method on the panel rather than a package
// list: a lista fora de ordem é uma lista que alguém vai reordenar sem querer.
func AllTabs() []Tab { return []Tab{TabTerminal, TabProblems, TabSearch} }

// Label is the tab's name as the reader reads it.
func (t Tab) Label() string {
	switch t {
	case TabTerminal:
		return "Terminal"
	case TabProblems:
		return "Problemas"
	case TabSearch:
		return "Busca"
	}
	return string(t)
}

// View is what the panel needs to draw.
//
// Os conteúdos chegam **prontos**, cada um com a altura que a região tem menos a
// faixa de abas: o painel conhece a faixa e o desenho, e não o conteúdo de cada
// aba — que é a superfície de cada uma, com os testes dela.
type View struct {
	Tabs    []Tab
	Active  Tab
	Focused bool
	// Content is the active tab's rows, already measured to the content width.
	Content []string
	// ProblemCount é quantos problemas a aba tem, e **zero significa que ninguém
	// mediu**: o desenho diz "LSP desligado" em vez de "0 problemas".
	ProblemCount int
	// HasProblems é o que separa "medido zero" de "sem medição".
	HasProblems bool
	Theme       theme.Theme
}

// TabStripHeight is how many rows the tabs take.
const TabStripHeight = 1

// Render draws the panel: a row of tabs, then the active tab's content.
func Render(width, height int, view View) []string {
	if width <= 0 || height <= 0 {
		return nil
	}

	rows := make([]string, 0, height)
	rows = append(rows, style(view, layout.Pad(strip(width, view), width)))

	for _, row := range view.Content {
		rows = append(rows, style(view, layout.Pad(row, width)))
	}
	for len(rows) < height {
		rows = append(rows, style(view, layout.Pad("", width)))
	}
	return rows[:height]
}

// strip é a linha de abas, com a ativa marcada **em texto**.
//
// A marca é o nome em negrito mais um `▍` na frente, e não só a cor: a regra do
// projeto é que nenhum estado é comunicado apenas por cor, e a faixa de abas é o
// lugar em que o estado é mais lido.
func strip(width int, view View) string {
	var out strings.Builder
	for _, tab := range view.Tabs {
		label := tab.Label()
		if tab == view.Active {
			label = "▍" + label
		}
		out.WriteString(" " + label)
	}

	line := out.String()
	if count := problemBadge(view); count != "" {
		line += "  " + count
	}
	return line
}

// problemBadge é a contagem da aba de problemas, e ela só aparece quando alguém
// mediu.
func problemBadge(view View) string {
	if !view.HasProblems || view.ProblemCount == 0 {
		return ""
	}
	if view.ProblemCount == 1 {
		return "1 erro"
	}
	return strconv.Itoa(view.ProblemCount) + " erros"
}

// style paints the panel: a cor do chrome da aplicação, e não a da região — um
// painel com fundo de editor se confundiria com o texto que ele contém.
func style(view View, line string) string {
	return view.Theme.Gutter().Render(line)
}
