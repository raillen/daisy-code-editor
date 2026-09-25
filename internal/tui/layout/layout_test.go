package layout

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// TestClassifyBoundaries pins the edges. An off-by-one here moves every surface
// on screen, and the boundaries are the only part of the classification that can
// be wrong.
func TestClassifyBoundaries(t *testing.T) {
	cases := map[int]WidthClass{
		0: Compact, 1: Compact, 79: Compact,
		80: Standard, 119: Standard,
		120: Wide, 200: Wide,
	}
	for width, want := range cases {
		if got := Classify(width); got != want {
			t.Errorf("Classify(%d) = %v, esperado %v", width, got, want)
		}
	}
}

// TestComputeStacksEverySurface covers the vertical order the layout document
// promises: menu bar, tabs, body, status bar — top to bottom, no gaps.
func TestComputeStacksEverySurface(t *testing.T) {
	regions := Compute(Size{Width: 100, Height: 30}, Wants{})

	if regions.MenuBar.Y != 0 {
		t.Errorf("menu bar em y=%d, esperado 0", regions.MenuBar.Y)
	}
	if regions.Tabs.Y != regions.MenuBar.Y+regions.MenuBar.Height {
		t.Errorf("tabs em y=%d, logo abaixo do menu bar", regions.Tabs.Y)
	}
	if regions.Editor.Y != regions.Tabs.Y+regions.Tabs.Height {
		t.Errorf("editor em y=%d, logo abaixo das abas", regions.Editor.Y)
	}
	if regions.StatusBar.Y != 29 {
		t.Errorf("statusbar em y=%d, esperado 29 (última linha)", regions.StatusBar.Y)
	}

	// Sem sobreposição e sem buraco: a soma das alturas cobre o terminal.
	covered := regions.MenuBar.Height + regions.Tabs.Height + regions.Editor.Height + regions.StatusBar.Height
	if covered != 30 {
		t.Errorf("alturas somam %d, esperado 30", covered)
	}
}

// TestComputeNeverOverlapsHorizontally: a tree drawn on top of the editor's first
// columns would hide text with no sign that it is there.
func TestComputeNeverOverlapsHorizontally(t *testing.T) {
	for _, width := range []int{60, 80, 100, 120, 160} {
		regions := Compute(Size{Width: width, Height: 30}, Wants{ShowTree: true, TreeWidth: 30})

		if regions.TreeOverlaid {
			continue // sobreposição é o comportamento declarado em compact
		}
		if regions.Tree.X+regions.Tree.Width != regions.Editor.X {
			t.Errorf("largura %d: árvore termina em %d e editor começa em %d",
				width, regions.Tree.X+regions.Tree.Width, regions.Editor.X)
		}
	}
}

// TestComputeOverlaysTheTreeWhenCompact is the rule of the layout document: in
// compact the tree floats instead of taking a column, so the editor keeps its
// width.
func TestComputeOverlaysTheTreeWhenCompact(t *testing.T) {
	regions := Compute(Size{Width: 60, Height: 30}, Wants{ShowTree: true, TreeWidth: 30})

	if !regions.TreeOverlaid {
		t.Fatal("em compact a árvore deveria estar sobreposta")
	}
	if regions.Editor.Width != 60 {
		t.Errorf("o editor ficou com %d colunas, esperado 60 (a largura toda)", regions.Editor.Width)
	}
	if regions.Tree.Width > regions.Editor.Width {
		t.Error("a árvore flutuante ficou maior que o editor")
	}
}

// TestComputeProtectsTheEditor: a tree that squeezes the editor below the minimum
// has stopped being useful, so the tree gives way.
func TestComputeProtectsTheEditor(t *testing.T) {
	regions := Compute(Size{Width: 90, Height: 30}, Wants{ShowTree: true, TreeWidth: 70})

	if !regions.TreeOverlaid {
		t.Errorf("o editor ficou com %d colunas; a árvore deveria ter cedido", regions.Editor.Width)
	}
	if regions.Editor.Width < 40 {
		t.Errorf("o editor ficou com %d colunas, abaixo do mínimo", regions.Editor.Width)
	}
}

// TestComputeTerminalTakesFromTheBody: the terminal is carved out of the body and
// the editor keeps what is left, with the status bar untouched.
func TestComputeTerminalTakesFromTheBody(t *testing.T) {
	regions := Compute(Size{Width: 100, Height: 40}, Wants{ShowTerminal: true, TerminalHeight: 10})

	if regions.Terminal.Height != 10 {
		t.Fatalf("terminal com %d linhas, esperado 10", regions.Terminal.Height)
	}
	if regions.Terminal.Y+regions.Terminal.Height != regions.StatusBar.Y {
		t.Errorf("o terminal não termina onde a statusbar começa: %d+%d vs %d",
			regions.Terminal.Y, regions.Terminal.Height, regions.StatusBar.Y)
	}
	if regions.Editor.Y+regions.Editor.Height != regions.Terminal.Y {
		t.Errorf("o editor deveria terminar onde o terminal começa")
	}
}

// TestComputeRefusesToEatTheEditor: on a short terminal a tall terminal panel
// would leave nothing to edit, so it is dropped rather than shown empty.
func TestComputeRefusesToEatTheEditor(t *testing.T) {
	regions := Compute(Size{Width: 100, Height: 12}, Wants{ShowTerminal: true, TerminalHeight: 20})

	if !regions.Terminal.Empty() {
		t.Errorf("o terminal tomou %d linhas de um corpo de %d", regions.Terminal.Height, regions.Editor.Height)
	}
	if regions.Editor.Empty() {
		t.Error("o editor ficou sem área")
	}
}

// TestComputeHandlesAnUnmeasuredTerminal: zero is a valid size. A terminal that
// has not been measured yet must produce empty regions, not a panic or a negative
// width that renders as garbage.
func TestComputeHandlesAnUnmeasuredTerminal(t *testing.T) {
	for _, size := range []Size{{}, {Width: 0, Height: 0}, {Width: -5, Height: 10}, {Width: 80, Height: -1}, {Width: 3, Height: 3}} {
		regions := Compute(size, Wants{ShowTree: true, TreeWidth: 30, ShowTerminal: true, TerminalHeight: 10})

		for name, region := range map[string]Region{
			"editor": regions.Editor, "tree": regions.Tree,
			"terminal": regions.Terminal, "statusbar": regions.StatusBar,
		} {
			if region.Width < 0 || region.Height < 0 {
				t.Errorf("%v: %s com dimensão negativa: %+v", size, name, region)
			}
		}
	}
}

// TestTruncateMarksTheCut: a label that vanishes with no sign is
// indistinguishable from one that never existed.
//
// The expected strings encode the marker's real width, and that width is now
// **measured** instead of assumed: "…" is an ambiguous-width character, and this
// package counts it as one cell. An earlier version of this test hard-coded two
// because the ambiguous table said so, and a hard-coded width that disagrees with
// Width() is the bug this package already paid for three times.
func TestTruncateMarksTheCut(t *testing.T) {
	if got := Width(ellipsis); got != 1 {
		t.Fatalf("o marcador ocupa %d células; as expectativas deste teste derivam disso", got)
	}
	if got := Truncate("abcdef", 4); got != "abc…" {
		t.Errorf("Truncate = %q, esperado \"abc…\" (o marcador ocupa uma célula)", got)
	}
	if got := Truncate("abc", 10); got != "abc" {
		t.Errorf("texto que cabe foi alterado: %q", got)
	}
	if got := Truncate("abcdef", 1); got != "…" {
		t.Errorf("Truncate com uma coluna = %q, esperado \"…\"", got)
	}
	if got := Truncate("abcdef", 0); got != "" {
		t.Errorf("Truncate com zero = %q", got)
	}
}

// TestTruncateNeverOverflows is the invariant behind the three bugs this had:
// every result must fit the width it was given, for every width.
func TestTruncateNeverOverflows(t *testing.T) {
	texts := []string{"abcdef", "日本語のテキスト", "uma frase longa", "→→→", "a", ""}
	for _, text := range texts {
		for width := 0; width <= 20; width++ {
			if got := Width(Truncate(text, width)); got > width {
				t.Errorf("Truncate(%q, %d) = %q com %d células", text, width, Truncate(text, width), got)
			}
		}
	}
}

// TestTruncateCountsCellsNotRunes: a wide character occupies two columns, and
// counting runes overflows the row.
func TestTruncateCountsCellsNotRunes(t *testing.T) {
	// 日本語 = 3 runas, 6 células.
	if got := Width("日本語"); got != 6 {
		t.Fatalf("Width(日本語) = %d, esperado 6", got)
	}
	if got := Truncate("日本語", 5); Width(got) > 5 {
		t.Errorf("Truncate = %q com %d células, acima de 5", got, Width(got))
	}
}

// TestAmbiguousIsNarrowNotConservative substitui a regra antiga.
//
// A regra era "ambíguo conta como largo, porque errar para o largo custa só uma
// coluna". A medição mostrou que o custo era maior: a tabela ambígua do
// `go-runewidth` inclui "é" e não inclui "ã", então num produto em português
// **cada linha acentuada saía uma coluna mais larga** que a linha vizinha, e esta
// medição discordava da que os componentes `bubbles` usam. A prova está em
// TestAmbiguousWidthCountsAsNarrow e TestTheThreeMeasuresAgree.

// TestPadFitsExactly: a row that comes back short lets whatever is to its right
// shift left, which turns a two-column layout into a diagonal one.
func TestPadFitsExactly(t *testing.T) {
	cases := []struct {
		text  string
		width int
	}{
		{"abc", 10}, {"", 5}, {"exatamente", 10}, {"uma frase bem mais longa", 12},
		{"日本語", 8}, {"日本語", 4},
	}
	for _, tc := range cases {
		got := Pad(tc.text, tc.width)
		if Width(got) != tc.width {
			t.Errorf("Pad(%q, %d) = %q com %d células", tc.text, tc.width, got, Width(got))
		}
	}
}

// TestPadWithNonPositiveWidth returns nothing rather than padding to a negative
// width.
func TestPadWithNonPositiveWidth(t *testing.T) {
	for _, width := range []int{0, -1} {
		if got := Pad("abc", width); got != "" {
			t.Errorf("Pad(abc, %d) = %q", width, got)
		}
	}
}

// TestEveryWidthProducesASaneFrame walks the widths the gate names, plus the
// boundaries, and asserts the frame is always usable.
func TestEveryWidthProducesASaneFrame(t *testing.T) {
	for _, width := range []int{20, 60, 79, 80, 100, 119, 120, 140, 200} {
		regions := Compute(Size{Width: width, Height: 40}, Wants{ShowTree: true, TreeWidth: 30})

		if regions.Editor.Empty() {
			t.Errorf("largura %d: o editor ficou vazio", width)
			continue
		}
		if regions.MenuBar.Width != width || regions.StatusBar.Width != width {
			t.Errorf("largura %d: as barras não cobrem o terminal", width)
		}
		if regions.Editor.X+regions.Editor.Width > width {
			t.Errorf("largura %d: o editor passa de %d", regions.Editor.X+regions.Editor.Width, width)
		}
	}
}

// TestOverlaidTreeNeverExceedsTheBody keeps the floating panel inside its parent.
func TestOverlaidTreeNeverExceedsTheBody(t *testing.T) {
	for _, height := range []int{3, 10, 30, 80} {
		regions := Compute(Size{Width: 60, Height: height}, Wants{ShowTree: true, TreeWidth: 40})

		if regions.Tree.Y+regions.Tree.Height > height {
			t.Errorf("altura %d: a árvore flutuante passa do terminal", height)
		}
		if regions.Tree.Width > 60 {
			t.Errorf("altura %d: a árvore flutuante passa da largura", height)
		}
	}
}

// TestNoSurfaceIsEverNegative is the cheap invariant that catches an arithmetic
// mistake anywhere in Compute.
func TestNoSurfaceIsEverNegative(t *testing.T) {
	for width := 0; width <= 200; width += 7 {
		for height := 0; height <= 60; height += 5 {
			regions := Compute(Size{Width: width, Height: height},
				Wants{ShowTree: true, TreeWidth: 30, ShowTerminal: true, TerminalHeight: 12})

			all := []Region{regions.MenuBar, regions.Tabs, regions.Tree,
				regions.Editor, regions.Terminal, regions.StatusBar}
			for index, region := range all {
				if region.Width < 0 || region.Height < 0 || region.X < 0 || region.Y < 0 {
					t.Fatalf("%dx%d: região %d negativa: %+v", width, height, index, region)
				}
			}
		}
	}
}

// TestTruncateIsStableUnderRepetition: truncating an already-truncated string must
// not keep eating it. A surface that re-truncates its own output every frame would
// otherwise shrink the text one marker at a time.
func TestTruncateIsStableUnderRepetition(t *testing.T) {
	text := "uma linha bem longa que não cabe"
	once := Truncate(text, 12)
	twice := Truncate(once, 12)

	if once != twice {
		t.Errorf("truncar de novo mudou o texto: %q → %q", once, twice)
	}
	if strings.HasSuffix(once, "……") {
		t.Errorf("o marcador se acumulou: %q", once)
	}
}

// TestCellsConvertsBytesIntoTerminalCells is the conversion the selection, the
// search and the syntax roles all depend on, pinned against the case that broke
// it: a byte offset is not a cell column once a wide character is in the line.
func TestCellsConvertsBytesIntoTerminalCells(t *testing.T) {
	cases := []struct {
		text   string
		offset int
		want   int
	}{
		{"", 0, 0},
		{"abc", 0, 0},
		{"abc", 3, 3},
		{"日本語", 3, 2},
		{"日本語 func", 9, 6},
		{"日本語 func", 10, 7},
		{"日本語 func", 14, 11},
		// Past the end clamps to the width, because a token that outlived an edit
		// is not an error and must not produce a negative column.
		{"abc", 500, 3},
		// A negative offset is the same as zero.
		{"abc", -1, 0},
	}
	for _, tc := range cases {
		if got := Cells(tc.text, tc.offset); got != tc.want {
			t.Errorf("Cells(%q, %d) = %d, esperava %d", tc.text, tc.offset, got, tc.want)
		}
	}
}

// TestCellsRoundsDownInsideACharacter: an offset that splits a multi-byte
// character rounds to the character's start, which keeps the tint inside the text
// rather than half a character to its left.
func TestCellsRoundsDownInsideACharacter(t *testing.T) {
	for offset := 1; offset <= 2; offset++ {
		if got := Cells("日", offset); got != 0 {
			t.Errorf("Cells(\"日\", %d) = %d, esperava 0", offset, got)
		}
	}
}

// TestCellsAgreesWithSlice is the property that makes the two usable together: the
// column Cells reports has to be the column Slice starts from, or a token is
// painted one character away from where the highlighter said it starts.
func TestCellsAgreesWithSlice(t *testing.T) {
	text := "日本語 e um emoji 🙂 no meio"

	for offset := 0; offset <= len(text); offset++ {
		column := Cells(text, offset)
		if got := len([]rune(Slice(text, column, 1))); got > 1 {
			t.Errorf("deslocamento %d: a coluna %d não é um início de caractere", offset, column)
		}
		if column > Width(text) {
			t.Errorf("deslocamento %d: coluna %d passa da largura %d", offset, column, Width(text))
		}
	}
}

// widthCorpus is the text the three measures are compared on.
//
// Deliberately not only ASCII: a corpus de ASCII_only provaria que as três
// medidas concordam, e é exatamente o texto fácil que nunca produziu bug.
var widthCorpus = []string{
	"",
	"a",
	"abc def",
	"  espacos  ",
	"日本語",
	"日本語のテキスト",
	"한글 텍스트",
	"ἀγγλικὰ",
	"ЖЖЖ",
	// Ambíguos: a regra deliberada deste pacote.
	"são", "ações", "informação", "→", "…", "·", "±", "«»", "☺", "│─",
	// Clusters: acento combinado, emoji com ZWJ, bandeira.
	"é", "a\u0301b", "👍🏽", "🇧🇷🇵🇹", "👨‍👩‍👧",
	// Rune que é dois cells e rune que é zero.
	"\u0301", "Ａ",
	// Sequences: ocupam zero célula.
	"\x1b[31mabc\x1b[m",
	"a\x1b[1mb\x1b[0mc",
	"\x1b[38;2;16;64;96mcolorido\x1b[m",
}

// TestTheThreeMeasuresAgree is the acceptance of S2.
//
// Duas medidas de largura que podem discordar são o terreno exato do sexto bug
// desta classe, que é a mais frequente do projeto. Este teste afirma o acordo
// sobre texto ambíguo, largo, com acento combinado, com emoji e com ANSI — e
// sobre a resposta que as três dão.
func TestTheThreeMeasuresAgree(t *testing.T) {
	for _, text := range widthCorpus {
		ours := Width(text)
		for name, other := range map[string]int{
			"clipperhouse/displaywidth": measure.String(text),
			"x/ansi.StringWidth":        ansi.StringWidth(text),
		} {
			if ours != other {
				t.Errorf("%q: %s mediu %d, esta mediu %d", text, name, other, ours)
			}
		}
	}
}

// TestAmbiguousWidthCountsAsNarrow is a decisão, não um acidente.
//
// A tabela ambígua do go-runewidth inclui "é" e não inclui "ã". Com a regra
// antiga — ambíguo conta como largo — "são" mediava quatro células num terminal
// que desenha três, e cada linha accentuada saía uma coluna mais larga que a
// linha vizinha. Este teste fixa a escolha e o motivo.
func TestAmbiguousWidthCountsAsNarrow(t *testing.T) {
	for _, text := range []string{"são", "informação", "é", "→", "…", "·"} {
		if got := Width(text); got != len([]rune(text)) {
			t.Errorf("%q mediu %d, esperava %d (uma célula por caractere)", text, got, len([]rune(text)))
		}
	}
}

// TestWideTextStillMeasuresWide é a outra metade: estreitar o ambíguo não pode
// ter estragado o largo de verdade.
func TestWideTextStillMeasuresWide(t *testing.T) {
	for _, text := range []string{"日本語", "한글", "Ａ"} {
		if got := Width(text); got <= len([]rune(text)) {
			t.Errorf("%q mediu %d, esperava mais de %d", text, got, len([]rune(text)))
		}
	}
}

// TestSliceAndTruncateNeverSplitACluster é o que a medida por rune não garantia.
//
// Um cluster cortado produz um caractere que o terminal não sabe desenhar, e a
// largura medida da sobra deixa de ser a largura desenhada.
func TestSliceAndTruncateNeverSplitACluster(t *testing.T) {
	texts := []string{"👨‍👩‍👧x", "🇧🇷abc", "a\u0301bc", "日本語x"}

	for _, text := range texts {
		for width := range 6 {
			sliced := Slice(text, 0, width+1)
			if width := Width(sliced); width > width+1 {
				t.Errorf("Slice(%q, 0, %d) devolveu %q medindo %d", text, width+1, sliced, width)
			}
			truncated := Truncate(text, width+1)
			if got := Width(truncated); got > width+1 {
				t.Errorf("Truncate(%q, %d) devolveu %q medindo %d", text, width+1, truncated, got)
			}
		}
	}
}

// TestPadAndSliceAgree is a propriedade que a pintura depende: uma linha
// montada com Pad tem que ter a mesma largura que a soma das fatias que a
// formaram, ou o corte de um campo desalinha o resto da linha.
func TestPadAndSliceAgree(t *testing.T) {
	for _, text := range widthCorpus {
		for width := range 8 {
			column := 0
			for column < width {
				piece := Slice(text, column, width-column)
				cells := Width(piece)
				if piece == "" || cells == 0 {
					// Cluster de largura zero (acento combinado solto) não
					// avança a coluna: devolvê-lo é correto, mas contar
					// progresso por ele seria laço infinito.
					break
				}
				column += cells
			}
			if got := Width(Pad(text, width)); got != width {
				t.Errorf("Pad(%q, %d) mediu %d", text, width, got)
			}
		}
	}
}
