package findbar

import (
	"strings"
	"testing"

	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/theme"
)

// sampleTheme is the shipped palette: a View with no theme renders plain, and a
// test do layout não deve estar medindo a ausência de cor.
func sampleTheme() theme.Theme {
	shipped := config.Default()
	return theme.New(shipped.UI, shipped.Syntax, theme.TrueColor)
}

func sample() View {
	return View{
		Theme:         sampleTheme(),
		Field:         layout.Pad("alfa", FieldWidth),
		ReplaceField:  layout.Pad("", FieldWidth),
		IgnoreAccents: true,
		Current:       2,
		Total:         5,
	}
}

// TestEveryRowHasTheExactWidth keeps the bar from shifting the frame.
func TestEveryRowHasTheExactWidth(t *testing.T) {
	for _, showReplace := range []bool{false, true} {
		view := sample()
		view.ShowReplace = showReplace

		for _, width := range []int{20, 40, 79, 80, 200} {
			rows := Render(width, view)
			if len(rows) != view.Height() {
				t.Errorf("revelar=%v: %d linhas, esperado %d", showReplace, len(rows), view.Height())
			}
			for index, row := range rows {
				if got := layout.Width(row); got != width {
					t.Errorf("revelar=%v largura=%d: linha %d com %d células", showReplace, width, index, got)
				}
			}
		}
	}
}

// TestHeightGrowsOnlyForTheReplaceField: the layout is told before it divides the
// screen, so the height has to be right before anything is drawn.
func TestHeightGrowsOnlyForTheReplaceField(t *testing.T) {
	view := sample()
	if got := view.Height(); got != 1 {
		t.Errorf("altura = %d, esperado 1", got)
	}
	view.ShowReplace = true
	if got := view.Height(); got != 2 {
		t.Errorf("altura = %d, esperado 2", got)
	}
}

// TestFlagsAreSpelledOut: a flag carried only by colour is a flag the reader
// misreads a result over.
func TestFlagsAreSpelledOut(t *testing.T) {
	view := sample()
	view.CaseSensitive = true
	view.UseRegex = true

	row := Render(120, view)[0]
	for _, want := range []string{"Aa", "acentos", "regex"} {
		if !strings.Contains(row, want) {
			t.Errorf("o modificador %q não está escrito: %q", want, row)
		}
	}
}

// TestNoMatchSaysSo: a bar showing "0/0" leaves the reader unsure whether the
// search ran.
func TestNoMatchSaysSo(t *testing.T) {
	view := sample()
	view.Total = 0

	if row := Render(120, view)[0]; !strings.Contains(row, noMatch) {
		t.Errorf("sem casamentos sem aviso: %q", row)
	}
}

// TestARegularExpressionErrorReplacesTheCount: a pattern that does not compile
// matched nothing, so a count beside the error would be a useful lie.
func TestARegularExpressionErrorReplacesTheCount(t *testing.T) {
	view := sample()
	view.RegexError = "error parsing regexp: missing closing ]"
	view.Total = 0

	row := Render(200, view)[0]
	if !strings.Contains(row, "padrão inválido") {
		t.Errorf("o erro não apareceu: %q", row)
	}
	if strings.Contains(row, noMatch) {
		t.Errorf("a contagem apareceu junto do erro: %q", row)
	}
}

// TestALongQueryShowsItsTail: someone typing needs to see where they are typing.
// TestTheBarDrawsTheFieldItIsGiven is what is ours after the S3 swap: the bar
// lays out a field that the component `textinput` already drew, and does not
// second-guess it.
//
// The two tests that used to live here — the tail of a long query and the
// placeholder of an empty one — were deleted rather than adapted: they measured
// the component's behaviour, and the library tests it better. Keeping them would
// be a copy of somebody else's suite, que diverge no primeiro bug corrigido lá.
func TestTheBarDrawsTheFieldItIsGiven(t *testing.T) {
	view := View{
		Field:         layout.Pad("alfa", FieldWidth),
		ReplaceField:  layout.Pad("digite…", FieldWidth),
		ShowReplace:   true,
		CaseSensitive: false,
		Total:         0,
		Theme:         sample().Theme,
	}

	rows := Render(60, view)

	if !strings.Contains(rows[0], "alfa") {
		t.Errorf("o campo da consulta não está na linha: %q", rows[0])
	}
	if !strings.Contains(rows[1], "digite…") {
		t.Errorf("o campo da substituição não está na linha: %q", rows[1])
	}
	for index, row := range rows {
		if got := layout.Width(row); got != 60 {
			t.Errorf("linha %d com %d células", index, got)
		}
	}
}
