// Package theme resolves the configured colours into styles.
//
// It is the only place that knows a colour exists. Surfaces ask for a role —
// selection, gutter, status — and get a style, so a palette change does not reach
// into the code that draws.
package theme

import (
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/syntax"
)

// Profile is what the terminal can render.
//
// Ordered from least to most capable so a comparison answers "does this terminal
// support at least that", which is how every colour decision is made.
type Profile int

// The profiles, matching docs/ui-ux/degradation.md.
const (
	// NoColor renders nothing: every non-primitive token resolves to no styling
	// at all. A theme that claims to suppress colour and paints one is an error.
	NoColor Profile = iota
	ANSI16
	ANSI256
	TrueColor
)

// Theme resolves colours into styles for one profile.
//
// Duas camadas, e a ordem importa: uma **superfície** pede um **token**, e o
// token resolve para uma cor. A superfície nunca nomeia cor — é o que permite
// que o mesmo código pinte a gutter de `text_muted` num tema e noutro, sem que
// alguém precise saber qual valor usar.
type Theme struct {
	profile Profile
	ui      config.UIConfig
	syntax  config.SyntaxColors
	tokens  config.Tokens
}

// New builds a theme.
//
// The profile is a parameter rather than read from the environment: a test must be
// able to ask for NoColor without writing to the process environment, which is
// shared state.
func New(ui config.UIConfig, syntax config.SyntaxColors, profile Profile) Theme {
	return NewWithTokens(ui, syntax, nil, profile)
}

// NewWithTokens builds a theme with its semantic layer.
//
// `tokens` nulo herda das cores diretas, e é o caminho de todo tema que não
// declara a camada — os dez gerados do oráculo entre eles. A opcionalidade não é
// conveniência: é o que mantém o quadro deles byte a byte igual.
func NewWithTokens(ui config.UIConfig, syntax config.SyntaxColors, tokens *config.Tokens, profile Profile) Theme {
	layer := config.DefaultTokens(ui)
	if tokens != nil {
		layer = *tokens
	}
	return Theme{profile: profile, ui: ui, syntax: syntax, tokens: layer}
}

// Profile reports what this theme renders for.
func (t Theme) Profile() Profile { return t.profile }

// The semantic tokens, as the surfaces ask for them.
//
// Um acessor por token, e não um `Token(name string)`: o nome é string e uma
// string errada é um token que não pinta, silenciosamente. Com o acessor, o
// compilador guarda.

// Surface is the editor's background.
func (t Theme) Surface() lipgloss.Style { return t.token(t.tokens.Surface) }

// Panel is a surface that is not the editor.
func (t Theme) Panel() lipgloss.Style { return t.token(t.tokens.Panel) }

// PanelText is the text on a panel, which reads at a different weight than the
// editor's body text — e é por isso que é um token e não `Text` de novo.
func (t Theme) PanelText() lipgloss.Style { return t.token(t.tokens.PanelText) }

// Text is body text.
func (t Theme) Text() lipgloss.Style { return t.token(t.tokens.Text) }

// Muted is secondary text: gutter, line numbers, hints.
func (t Theme) Muted() lipgloss.Style { return t.token(t.tokens.TextMuted) }

// Accent marks what is active or selected.
func (t Theme) Accent() lipgloss.Style { return t.token(t.tokens.Accent) }

// Success is a state that is fine.
func (t Theme) Success() lipgloss.Style { return t.token(t.tokens.Success) }

// Warning is a state that needs attention without being an error.
func (t Theme) Warning() lipgloss.Style { return t.token(t.tokens.Warning) }

// Error is a diagnostic.
//
// E o token que **nunca** pode ser a única pista: a regra do projeto é que nenhum
// estado vive só na cor, e um token de erro que é só cor é um estado que some
// num terminal sem cor. As superfícies que o usam escrevem o texto junto.
func (t Theme) Error() lipgloss.Style { return t.token(t.tokens.Error) }

// Focus marks the surface that has the keyboard.
func (t Theme) Focus() lipgloss.Style { return t.token(t.tokens.Focus) }

// LineHighlight is the background of the caret's line, and **nothing at all** when
// the token is empty.
//
// Decoração, e não estado: a linha atual já é marcada em texto, na gutter, com o
// marcador do caret. Esta é a segunda pista, e a segunda pista some sem custo
// nenhum num terminal sem cor.
//
// `HasLineHighlight` é separado do estilo porque `GetBackground` nunca devolve
// nulo na biblioteca — um estilo sem fundo tem um fundo, e é o fundo do terminal.
// A pergunta "este tema quer a linha atual marcada?" é do **token**, e não do
// estilo.
func (t Theme) LineHighlight() lipgloss.Style {
	background := parse(t.tokens.LineHighlight)
	if background == nil {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Background(background)
}

// HasLineHighlight reports whether the theme asks for the caret's line to be
// marked, which é o que decide se a superfície pinta o fundo.
func (t Theme) HasLineHighlight() bool {
	return t.profile != NoColor && !config.IsNoColour(t.tokens.LineHighlight)
}

// token resolves one token into a style that paints nothing when the token says
// "no colour".
func (t Theme) token(value string) lipgloss.Style {
	if config.IsNoColour(value) {
		// Um token `none` é o estilo zero, e não um estilo sem cor: ele não
		// desenha fundo nem muda a frente, o que é o que "não tenho cor" quer
		// dizer.
		return lipgloss.NewStyle()
	}
	return t.style("", "", value, false)
}

// Tokens exposes the resolved layer, for the surfaces that compose a style out of
// two tokens (a borda de um painel, por exemplo).
func (t Theme) Tokens() config.Tokens { return t.tokens }

// WithTokens returns the theme with another token layer, keeping the profile.
//
// Um método e não um construtor: o perfil vem do terminal, e um tema novo muda a
// paleta sem mudar a capacidade — que são duas coisas independentes.
func (t Theme) WithTokens(tokens *config.Tokens) Theme {
	layer := t.tokens
	if tokens != nil {
		layer = *tokens
	}
	t.tokens = layer
	return t
}

// Selection marks selected text.
func (t Theme) Selection() lipgloss.Style {
	return t.style(t.ui.Background, t.ui.CursorBG, t.ui.CursorFG, false)
}

// Gutter dims the line numbers.
//
// Fundo pela **intenção** (`surface`) e tinta pela **intenção** (`text_muted`), e
// não pelas cores diretas: é isso que faz a camada ser carregada em vez de
// existir. Um tema que declara `text_muted = "#ff0000"` muda a gutter sem tocar
// em `[ui]`, que é o ponto de ter uma camada.
func (t Theme) Gutter() lipgloss.Style {
	return t.style(t.tokens.Surface, "", t.tokens.TextMuted, false)
}

// Status is the bottom bar.
func (t Theme) Status() lipgloss.Style {
	return t.style(t.tokens.Panel, "", t.tokens.PanelText, false)
}

// StatusDirty is the bottom bar when the document has unsaved edits.
//
// A tinta vem do token `warning`: "há algo que precisa de atenção" é uma
// **intenção**, e o mesmo token serve para um arquivo modificado, um LSP
// carregando e uma tarefa que falhou. E continua **em negrito**, porque nenhum
// estado deste projeto vive só na cor.
func (t Theme) StatusDirty() lipgloss.Style {
	return t.style(t.tokens.Panel, "", t.tokens.Warning, true)
}

// Match marks a search result.
//
// The configuration has no search colours yet, so this borrows the cursor pair
// rather than inventing a token: a provisional choice that is visible in one place
// and will be replaced when the palette gains the role.
func (t Theme) Match() lipgloss.Style {
	return t.style(t.ui.CursorBG, "", t.ui.LineNumber, false)
}

// CurrentMatch marks the result the search is on.
//
// Distinguished from the others by weight and colour rather than by tint alone,
// because "which one of these am I on" is the question the highlight exists to
// answer.
func (t Theme) CurrentMatch() lipgloss.Style {
	return t.style(t.ui.CursorBG, "", t.ui.StatusDirty, true)
}

// Syntax is the style of a syntax role.
//
// One accessor for the twenty roles rather than twenty accessors: the roles are
// data, the field each one reads is data, and a switch spelled out here would be
// a second place for a role to be missing. A role with no colour resolves to the
// zero style, which paints nothing — the same promise NoColor makes.
func (t Theme) Syntax(role syntax.Role) lipgloss.Style {
	return t.style(t.ui.Background, "", syntax.Colour(t.syntax, role), false)
}

// style builds a style, resolving nothing when colour is suppressed.
func (t Theme) style(background, foreground, text string, bold bool) lipgloss.Style {
	style := lipgloss.NewStyle()
	if t.profile == NoColor {
		// The zero style applies no attribute, so the text renders as-is. This is
		// the whole no-colour guarantee, and it lives in one place.
		return style
	}

	if c := parse(background); c != nil {
		style = style.Background(c)
	}
	if c := parse(foreground); c != nil {
		style = style.Foreground(c)
	}
	if c := parse(text); c != nil {
		style = style.Foreground(c)
	}
	if bold {
		style = style.Bold(true)
	}
	return style
}

// ansiNames maps the colour names a configuration may use onto ANSI indices.
//
// The style library resolves hex and ANSI numbers, and silently ignores a name it
// does not know. That silence is what made an earlier version of this package ship
// a monochrome editor while every test passed: the tests used hex, the shipped
// configuration uses names, and an unresolved name is a no-op rather than an
// error.
var ansiNames = map[string]int{
	"black": 0, "red": 1, "green": 2, "yellow": 3,
	"blue": 4, "magenta": 5, "cyan": 6, "white": 7,
	"brightblack": 8, "darkgray": 8, "darkgrey": 8, "gray": 8, "grey": 8,
	"brightred": 9, "brightgreen": 10, "brightyellow": 11,
	"brightblue": 12, "brightmagenta": 13, "brightcyan": 14, "brightwhite": 15,
}

// parse reads a colour from configuration.
//
// An empty value and the literal "reset" both mean "no colour", and both are
// normal in the shipped themes. A recognised name becomes its ANSI index, so the
// value the style library receives is one it can actually render.
func parse(value string) color.Color {
	switch value {
	case "", "reset", "none":
		return nil
	}
	if index, known := ansiNames[strings.ToLower(strings.TrimSpace(value))]; known {
		return lipgloss.Color(strconv.Itoa(index))
	}
	return lipgloss.Color(value)
}
