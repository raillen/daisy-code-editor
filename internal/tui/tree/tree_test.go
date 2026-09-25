package tree

import (
	"strings"
	"testing"

	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/theme"
)

func sample() View {
	return View{
		Rows:     []Row{{Name: "src", IsDir: true, Expanded: true}, {Name: "a.go", Depth: 1}},
		Selected: 1,
	}
}

func TestExactWidthAndHeight(t *testing.T) {
	for _, width := range []int{10, 30, 80} {
		rows := Render(width, 5, sample())
		if len(rows) != 5 {
			t.Fatalf("largura %d: %d linhas", width, len(rows))
		}
		for index, row := range rows {
			if got := layout.Width(row); got != width {
				t.Errorf("largura %d linha %d: %d células", width, index, got)
			}
		}
	}
}

// TestMarksDirectoriesInText: the affordance is what tells a reader the entry can
// be opened, so it cannot be colour or an ambiguous-width glyph.
func TestMarksDirectoriesInText(t *testing.T) {
	view := View{Rows: []Row{
		{Name: "aberto", IsDir: true, Expanded: true},
		{Name: "fechado", IsDir: true},
		{Name: "arquivo.go"},
	}}

	rows := Render(30, 3, view)
	if !strings.Contains(rows[0], expandedDir) {
		t.Errorf("diretório aberto sem marcador: %q", rows[0])
	}
	if !strings.Contains(rows[1], collapsedDir) {
		t.Errorf("diretório fechado sem marcador: %q", rows[1])
	}
	if strings.Contains(rows[2], expandedDir) || strings.Contains(rows[2], collapsedDir) {
		t.Errorf("arquivo marcado como diretório: %q", rows[2])
	}
}

// TestSaysEmptyInsteadOfBlank: a blank rectangle does not distinguish an empty
// panel from one that failed to load.
func TestSaysEmptyInsteadOfBlank(t *testing.T) {
	if rows := Render(30, 3, View{}); !strings.Contains(rows[0], emptyMessage) {
		t.Errorf("painel vazio sem aviso: %q", rows[0])
	}
}

// TestTheSelectionStaysVisible: the window follows the selection, which is what
// the panel is for.
func TestTheSelectionStaysVisible(t *testing.T) {
	rows := make([]Row, 40)
	for index := range rows {
		rows[index] = Row{Name: "arquivo"}
	}
	view := View{Rows: rows, Selected: 35}

	rendered := Render(30, 10, view)
	marked := -1
	for index, row := range rendered {
		if strings.Contains(row, "▶") {
			marked = index
		}
	}
	if marked < 0 {
		t.Fatal("a linha selecionada não apareceu")
	}
}

func TestZeroSizeRendersNothing(t *testing.T) {
	if rows := Render(0, 10, sample()); rows != nil {
		t.Errorf("largura zero devolveu %d linhas", len(rows))
	}
	if rows := Render(10, 0, sample()); rows != nil {
		t.Errorf("altura zero devolveu %d linhas", len(rows))
	}
}

// O badge de git é a **coluna de estado** da árvore: a letra que o `git` já
// simplificava, alinhada à direita e pintada por **intenção** (`warning` para
// modificado, `success` para novo, `error` para apagado).

// TestTheBadgeIsRightAligned é a propriedade que a coluna tem: o nome recebe o que
// sobra e o badge cai na borda. Uma coluna que anda é o defeito que a medição
// resolve — e que a lista de arquivos teve enquanto o badge era colado com um
// espaço.
func TestTheBadgeIsRightAligned(t *testing.T) {
	rows := []Row{
		{Path: "a.go", Name: "a.go", Status: "M"},
		{Path: "b.go", Name: "arquivo-de-nome-bem-mais-longo.go", Status: "A"},
	}

	for _, width := range []int{20, 30, 40} {
		painted := Render(width, 3, View{Rows: rows, Theme: sampleTheme()})
		for index, line := range painted[:2] {
			if got := layout.Width(line); got != width {
				t.Fatalf("largura %d, linha %d com %d células: %q", width, index, got, line)
			}
		}
		// O badge é a **última célula** das duas linhas, e não "a letra que
		// aparece": um badge colado com um espaço no fim passaria na comparação por
		// substring e falharia aqui, que é o defeito que a coluna medida elimina.
		for index, want := range []string{"M", "A"} {
			if got := lastCells(painted[index], 1); got != want {
				t.Errorf("linha %d termina em %q, esperava o badge %q", index, got, want)
			}
		}
	}
}

// TestTheBadgeIsALetterAndTheColourIsTheSecondPista: a regra do projeto é que
// nenhum estado é comunicado só por cor, e o badge é a letra que sobrevive num
// terminal sem cor.
func TestTheBadgeIsALetterAndTheColourIsTheSecondPista(t *testing.T) {
	view := View{Rows: []Row{{Path: "a.go", Name: "a.go", Status: "M"}}, Theme: sampleTheme()}

	coloured := Render(20, 1, view)[0]
	plain := Render(20, 1, View{Rows: view.Rows, Theme: plainTheme()})[0]

	if !strings.Contains(plain, "M") {
		t.Errorf("sem cor, o estado sumiu: %q", plain)
	}
	if !strings.Contains(coloured, "M") {
		t.Errorf("o estado sumiu: %q", coloured)
	}
}

// TestANameNeverDisappearsForABadge: a coluna de estado não pode comer o nome
// inteiro — um nome que some é pior do que um nome sem badge.
func TestANameNeverDisappearsForABadge(t *testing.T) {
	rows := []Row{{Path: "x.go", Name: "arquivo-com-nome-absurdamente-longo-para-nao-caber.go", Status: "M"}}

	// Estreito: duas colunas não cabem, e **some a letra** — que é informação
	// sobre o arquivo, não o arquivo. O nome fica, truncado.
	for _, width := range []int{4, 6, 10, 12} {
		line := Render(width, 1, View{Rows: rows, Theme: sampleTheme()})[0]
		if layout.Width(line) != width {
			t.Errorf("largura %d: linha com %d células: %q", width, layout.Width(line), line)
		}
		hasBadge := strings.Contains(line, "M")
		hasName := strings.Contains(stripEscapes(line), "…")
		if hasBadge && !hasName {
			t.Errorf("largura %d: o badge ficou e o nome sumiu: %q", width, stripEscapes(line))
		}
	}

	// Largo: as duas colunas coexistem, com o nome inteiro.
	line := Render(60, 1, View{Rows: rows, Theme: sampleTheme()})[0]
	if !strings.Contains(line, "arquivo-com-nome-absurdamente-longo-para-nao-caber.go") {
		t.Errorf("o nome sumiu em 60 colunas: %q", line)
	}
	if !strings.Contains(line, "M") {
		t.Errorf("o badge sumiu em 60 colunas: %q", line)
	}
}

// lastCells is the row's last n characters, with the escape sequences removed, e é
// por isso que a coluna se compara em texto.
func lastCells(line string, n int) string {
	plain := stripEscapes(line)
	runes := []rune(strings.TrimRight(plain, " "))
	if len(runes) < n {
		return string(runes)
	}
	return string(runes[len(runes)-n:])
}

// sampleTheme é a paleta padrão do produto: a coluna de estado precisa de uma cor
// real para o teste da segunda pista fazer sentido.
func sampleTheme() theme.Theme {
	shipped := config.Default()
	return theme.New(shipped.UI, shipped.Syntax, theme.TrueColor)
}

// plainTheme é o perfil sem cor, que é o que prova que a letra sobrevive sem ela.
func plainTheme() theme.Theme {
	shipped := config.Default()
	return theme.New(shipped.UI, shipped.Syntax, theme.NoColor)
}

// stripEscapes remove as sequências, para comparar **texto** e não codificação.
func stripEscapes(text string) string {
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
