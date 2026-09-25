// Package tree renders the project tree panel.
package tree

import (
	"github.com/ori-team/oride/internal/tui/component"
	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/theme"
)

// Row is one entry as the panel sees it.
type Row struct {
	Depth    int
	Name     string
	IsDir    bool
	Expanded bool
	// Path is the entry's full path, which é o que casa com o status do git.
	//
	// O nome sozinho não serve: o status vem por caminho, e casar por nome seria
	// casar `a.go` de um diretório com o `a.go` de outro.
	Path string
	// Status é o estado do git, em **letra** (`M`, `A`, `?`…) — a mesma
	// simplificação que `git.Status.Badge` já fazia para o painel SCM.
	Status string
}

// View is what the panel needs to draw.
//
// **Não há `Scroll`.** Havia, e ninguém escrevia nele: a composição monta a
// visão e a rolagem seguia a seleção, porque `component.Window` é a função pura
// que garante "a seleção está sempre visível e a lista não salta". Um campo que
// ninguém preenche é uma promessa de um recurso que não existe — e é também a
// prova medida de que não há rolagem manual para trocar por `viewport`.
type View struct {
	Rows     []Row
	Selected int
	Focused  bool
	// Theme paints the git badge. O valor zero desenha plain — que é o que um
	// terminal sem cor e todo teste querem.
	Theme theme.Theme
}

// Affordances. A directory says what it is and whether it is open, in text: an
// arrow would be an ambiguous-width character, and its presence would also be the
// only signal that a row can be expanded.
const (
	expandedDir  = "v "
	collapsedDir = "> "
	plainFile    = "  "
)

// emptyMessage is shown when the tree has no rows, so an empty panel is
// distinguishable from a panel that failed to load.
const emptyMessage = "(vazio)"

// Render draws the panel, one string per line, each exactly width cells.
func Render(width, height int, view View) []string {
	if width <= 0 || height <= 0 {
		return nil
	}

	start, end := component.Window(len(view.Rows), height, view.Selected, 0)
	rows := make([]string, 0, height)

	if len(view.Rows) == 0 {
		rows = append(rows, layout.Pad(" "+emptyMessage, width))
	}

	for index := start; index < end; index++ {
		row := view.Rows[index]
		selected := index == view.Selected

		// O badge é desenhado pelo `component.Row`, e não colado aqui: a coluna da
		// direita é geométrica — o nome precisa caber no que sobra, e um nome
		// colado com um espaço antes do badge desalinha a coluna entre linhas de
		// larguras diferentes.
		rows = append(rows, component.RowWithBadge(
			affordance(row)+row.Name, row.Depth, selected, width,
			view.Theme.FileStatus(row.Status), row.Status))
	}

	for len(rows) < height {
		rows = append(rows, layout.Pad("", width))
	}
	return rows
}

// affordance is the marker a row starts with.
//
// A directory is marked even when collapsed and even when it is the selected row,
// because the marker is what tells a reader the entry can be opened.
func affordance(row Row) string {
	if !row.IsDir {
		return plainFile
	}
	if row.Expanded {
		return expandedDir
	}
	return collapsedDir
}
