package component

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ori-team/oride/internal/tui/layout"
)

// Mark is the cursor drawn beside the selected row.
//
// A visible mark rather than colour alone: the rule is that no state is carried
// by colour, and a selection tinted but unmarked is invisible to anyone who
// cannot see the tint — and to anyone in a no-colour terminal.
const Mark = "▶"

// indentUnit is the spaces one level of depth adds.
const indentUnit = 2

// cursorWidth is the column reserved for the cursor, measured rather than
// assumed.
//
// The mark is an ambiguous-width character and this package counts those as wide,
// so writing `Mark + " "` reserved three cells while an unselected row reserved
// two — and every row shifted sideways the moment it was selected. Deriving the
// width from the same measurement that fills it removes the disagreement.
var cursorWidth = layout.Width(Mark) + 1

// Row renders one list entry at an exact width.
//
// The cursor column is reserved whether or not the row is selected, so selecting
// a row does not shift the text sideways — a list that jumps on every arrow key
// is exhausting to read.
func Row(name string, depth int, selected bool, width int) string {
	return row(name, depth, selected, width, lipgloss.NewStyle(), "")
}

// RowWithBadge renders a list entry with a **right-aligned** badge, the state
// column the tree shows for git.
//
// A coluna é medida, não colada: o nome recebe `largura − cursor − badge`, e o
// badge fica na direita. Colar " nome M" funciona até a primeira linha com nome
// mais curto — e a partir dali a coluna de estado anda, que é o defeito que a
// medição resolve e que a lista de arquivos sempre teve.
//
// O badge é **letra e texto** por dois motivos: a letra é o que o oráculo e o
// painel SCM já usam, e a cor do estilo é a segunda pista — nunca a única. Um
// terminal sem cor continua mostrando o estado.
func RowWithBadge(name string, depth int, selected bool, width int, style lipgloss.Style, badge string) string {
	return row(name, depth, selected, width, style, badge)
}

// minNameRoom é quantas células o nome precisa antes de a coluna de estado
// aparecer. É a regra da degradação, e ela é simétrica com o `minNameRoom` do
// resto do projeto: quando as duas colunas não cabem, fica a que é conteúdo.
const minNameRoom = 4

func row(name string, depth int, selected bool, width int, style lipgloss.Style, badge string) string {
	cursor := layout.Pad("", cursorWidth)
	if selected {
		cursor = layout.Pad(Mark, cursorWidth)
	}

	// A largura do cursor é **medida**, e o `…` é a mesma medição: um marcador
	// ambíguo que ninguém mediu é a linha que transborda na primeira letra longa.
	prefix := cursor + strings.Repeat(" ", depth*indentUnit)
	if badge == "" {
		return layout.Pad(prefix+name, width)
	}

	badgeWidth := layout.Width(badge)
	room := width - layout.Width(prefix) - badgeWidth - 1
	if room < minNameRoom {
		// Duas colunas não cabem, e some a menos importante: o nome é o
		// conteúdo da lista, a letra é informação sobre ele. Numa coluna de
		// quatro, "M" e um "…" não informam nada.
		return layout.Pad(prefix+layout.Truncate(name, max(width-layout.Width(prefix), 1)), width)
	}

	// O nome é preenchido até a borda do badge, e o badge é o **último** caractere
	// da linha: é a coluna, e uma coluna que anda não é coluna.
	shown := layout.Truncate(name, room)
	return layout.Pad(prefix+shown, width-badgeWidth-1) + " " + style.Render(layout.Pad(badge, badgeWidth))
}

// O nome é preenchido até `room` e o badge vem logo depois com um espaço: a soma
// `largura(cursor) + profundidade + nome + 1 + badge` dá exatamente a largura, e
// nenhuma das três medidas aqui é uma constante presumida — a mesma regra que a
// largura do `Mark` e do `…` cobrem.
