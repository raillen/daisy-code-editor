package scm

import (
	"strings"
	"testing"

	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/theme"
)

func themed(t *testing.T) theme.Theme {
	t.Helper()
	shipped := config.Default()
	return theme.New(shipped.UI, shipped.Syntax, theme.TrueColor)
}

func rows() []Row {
	return []Row{
		{Path: "a.go", Status: "M"},
		{Path: "novo.go", Status: "A"},
		{Path: "velho.go", Status: "D"},
	}
}

// TestTheScmPanelShowsTheLetterOfEachChange é a promessa da superfície: o que o
// disco diz, na letra.
func TestTheScmPanelShowsTheLetterOfEachChange(t *testing.T) {
	for _, width := range []int{20, 30, 60} {
		painted := Render(width, 5, View{Rows: rows(), Theme: themed(t)})
		joined := strings.Join(painted, "\n")
		for _, want := range []string{"a.go", "novo.go", "velho.go", "M", "A", "D"} {
			if !strings.Contains(joined, want) {
				t.Errorf("largura %d: o painel não mostra %q:\n%s", width, want, joined)
			}
		}
		for index, line := range painted {
			if got := layout.Width(line); got != width {
				t.Errorf("largura %d, linha %d com %d células: %q", width, index, got, line)
			}
		}
	}
}

// TestACleanRepositorySaysSo é o que distingue um repositório limpo de um painel
// que não carregou: a primeira é informação, e a segunda é falha disfarçada.
func TestACleanRepositorySaysSo(t *testing.T) {
	painted := Render(30, 3, View{Rows: nil, Theme: themed(t)})

	if !strings.Contains(strings.Join(painted, "\n"), "limpo") {
		t.Errorf("um repositório limpo não disse que está limpo:\n%s", strings.Join(painted, "\n"))
	}
}
