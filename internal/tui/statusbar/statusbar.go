// Package statusbar renders the bottom line.
package statusbar

import (
	"strconv"
	"strings"

	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/theme"
)

// View is what the status bar needs to draw.
//
// Every field is a plain value rather than a model reference: the bar is a pure
// function of what it is told, which is what lets it be tested at any width
// without a terminal.
type View struct {
	File      string
	Dirty     bool
	Focus     string
	Message   string
	CaretLine int
	CaretCol  int
	HasCaret  bool
	// Mode is the document's language, e é o campo que responde "o que este
	// arquivo é" sem abrir um seletor.
	Mode string
	// Branch is the workspace's git branch, and Ahead/Behind how far it is from
	// its upstream. HasUpstream diz se existe upstream — sem isso, "↑0 ↓0" é uma
	// afirmação sobre um dado que ninguém leu.
	Branch      string
	Ahead       int
	Behind      int
	HasUpstream bool
	// Diagnostics é a contagem de erros, e HasDiagnostics diz se ela é **real**.
	//
	// A separação é o que impede a mentira: com o LSP desligado, "0 erros" é uma
	// saúde que ninguém mediu, e a barra precisa dizer "LSP off" em vez disso.
	Diagnostics    int
	HasDiagnostics bool
	// LSP é o estado do cliente — "on", "off" — escrito por extenso de propósito:
	// a regra do projeto é que nenhum estado é comunicado só por cor, e um glifo
	// de LSP seria exatamente isso.
	LSP string
	// Theme styles the bar. The zero value renders plain, which is what a
	// no-colour terminal and every test want.
	Theme theme.Theme
}

// markers. The dirty flag is spelled out, not only tinted.
const (
	dirtyMark = "*"
	untitled  = "(sem título)"
	separator = "  "
)

// Render draws the status bar at an exact width.
//
// The caret position is placed at the right edge and everything else is
// truncated to fit what remains, because the position is the field a reader
// looks for at a fixed place — a value that moves is a value that gets missed.
func Render(width int, view View) string {
	if width <= 0 {
		return ""
	}

	right := position(view)
	left := description(view)

	// The right side wins the space it needs: it is short, and it is the field
	// read by position rather than by scanning.
	available := width - layout.Width(right) - 1
	if available < 0 {
		return layout.Pad(layout.Truncate(left, width), width)
	}

	left = layout.Truncate(left, available)
	padding := width - layout.Width(left) - layout.Width(right)
	if padding < 0 {
		padding = 0
	}

	// The style wraps the whole padded line, not just the text: a status bar whose
	// background stopped at the last word would look like a rendering fault.
	bar := layout.Pad(left+spaces(padding)+right, width)
	if view.Dirty {
		return view.Theme.StatusDirty().Render(bar)
	}
	return view.Theme.Status().Render(bar)
}

// description is the left-hand part: what is open, where focus is, and any
// transient message.
// git is the branch with its distance from upstream.
//
// O marcador `●` é texto e precede a branch, e é ele que faz a barra continuar
// legível em terminal sem cor: "main ↑2" é a informação; o ponto é o aviso de que
// há uma.
func git(view View) string {
	if view.Branch == "" {
		return ""
	}
	out := "● " + view.Branch
	if !view.HasUpstream {
		return out
	}
	return out + " ↑" + strconv.Itoa(view.Ahead) + " ↓" + strconv.Itoa(view.Behind)
}

// diagnostics é a contagem de erros, e ela **só aparece** quando alguém mediu.
func diagnostics(view View) string {
	if !view.HasDiagnostics || view.Diagnostics == 0 {
		return ""
	}
	if view.Diagnostics == 1 {
		return "1 erro"
	}
	return strconv.Itoa(view.Diagnostics) + " erros"
}

func description(view View) string {
	file := view.File
	if file == "" {
		file = untitled
	}
	if view.Dirty {
		file += dirtyMark
	}

	parts := []string{" " + file}
	if view.Mode != "" {
		parts = append(parts, view.Mode)
	}
	if view.Focus != "" {
		parts = append(parts, view.Focus)
	}
	if view.Message != "" {
		parts = append(parts, view.Message)
	}
	out := parts[0]
	for _, part := range parts[1:] {
		out += separator + part
	}
	return out
}

// position is the right-hand part: the caret's line and column.
// position is the right-hand block: git, diagnostics, LSP and the caret — com o
// caret **no fim**, porque é o campo que se lê por posição e um valor que anda é
// um valor que passa batido.
//
// A barra continua útil sem documento aberto: sem caret, os outros três campos
// permanecem, e uma barra que esvazia no documento novo é uma barra que mente
// sobre o programa estar vivo.
func position(view View) string {
	parts := make([]string, 0, 4)
	for _, part := range []string{git(view), diagnostics(view), view.LSP} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	if view.HasCaret {
		parts = append(parts, "Ln "+strconv.Itoa(view.CaretLine)+", Col "+strconv.Itoa(view.CaretCol))
	}
	if len(parts) == 0 {
		return " "
	}
	return strings.Join(parts, " · ") + " "
}

// spaces repeats a space.
func spaces(count int) string {
	if count <= 0 {
		return ""
	}
	out := make([]byte, count)
	for index := range out {
		out[index] = ' '
	}
	return string(out)
}
