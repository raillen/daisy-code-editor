package config

// Tokens are the **named intents** a theme speaks.
//
// A theme que escreve `#6272a4` na gutter está dizendo *como* e não *o quê*: a
// cor fica na paleta, e um tema novo precisa repetir a decisão em vez de herdar
// dela. Zed e Lapce resolvem isso com uma camada de tokens — `text-muted` é uma
// intenção, e a cor é da paleta.
//
// A segunda metade do motivo é mais dura: **sem token, `no-color` não existe**.
// Um perfil que desliga a cor resolve tudo para "nada", e é o que temos hoje; um
// **tema** `no-color` é uma paleta que afirma não ter cor nenhuma, e a afirmação
// só é verificável se cada token resolver para `none` — e se uma paleta que
// promete `none` e entrega cor for **erro**, não aviso.
//
// Os tokens são strings de cor, e não nomes de primitivo: a forma da cor que o
// runtime entende (`#rrggbb`, `red`, `reset`) é a mesma que `[ui]` já usa, e uma
// segunda gramática seria uma segunda dialecto para a mesma coisa.
type Tokens struct {
	// Surface is the editor's background.
	Surface string
	// Panel is the background of a surface that is not the editor: status bar,
	// find bar, terminal.
	Panel string
	// PanelText is the text **on** a panel. Separate from `text` because a panel
	// usually reads at a different weight than the editor, e o token é o lugar
	// onde essa diferença mora — não uma cor repetida em cada superfície.
	PanelText string
	// BorderSubtle separates surfaces without competing with the text.
	BorderSubtle string
	// Text is the body text.
	Text string
	// TextMuted is secondary text: gutter, line numbers, hints.
	TextMuted string
	// Accent marks what is active or selected: the current tab, the caret row.
	Accent string
	// Success is a state that is fine: a clean working tree, a finished task.
	Success string
	// Warning is a state that needs attention without being an error.
	Warning string
	// Error is a diagnostic, and the one state that must never be the only
	// signal — the rule of this projeto é que nenhum estado vive só na cor.
	Error string
	// Focus marks the surface that has the keyboard.
	Focus string
	// LineHighlight is the background of the line the caret is on.
	//
	// Deliberadamente **vazio** por padrão: um realce de linha atual é uma
	// decoração, e uma decoração que muda o quadro de todos os temas gerados do
	// oráculo seria uma mudança de produto feita por conta própria. O token
	// existe para o tema que quiser — e `TestLineHighlightIsOptionalByDefault`
	// segura essa escolha.
	LineHighlight string
}

// The token names, as they appear in `[tokens]`.
//
// Uma lista e não um mapa de constantes soltas: o nome é parte do contrato com o
// arquivo de configuração do usuário, e um nome escrito em dois lugares é um nome
// que diverge.
const (
	TokenSurface       = "surface"
	TokenPanel         = "panel"
	TokenPanelText     = "panel_text"
	TokenBorderSubtle  = "border_subtle"
	TokenText          = "text"
	TokenTextMuted     = "text_muted"
	TokenAccent        = "accent"
	TokenSuccess       = "success"
	TokenWarning       = "warning"
	TokenError         = "error"
	TokenFocus         = "focus"
	TokenLineHighlight = "line_highlight"
)

// defaultTokens is the token layer every theme starts from.
//
// O fallback vai para as cores diretas de `[ui]`, e não para valores novos: um
// tema gerado do oráculo não declara tokens, e o quadro dele **não pode mudar**.
// A prova de que a camada é inerte por padrão são os dez goldens, que passam sem
// regerar.
// DefaultTokens is the layer a theme inherits when it declares none.
func DefaultTokens(ui UIConfig) Tokens { return defaultTokens(ui) }

// TokenDefaultFields names the `[ui]` fields that exist to **seed** the token
// layer.
//
// A regra de aptidão (R8) conta um campo como lido quando um seletor o usa fora do
// `config` — e essas cores deixaram de ser lidas pelas superfícies quando elas
// passaram a resolver por token. A verdade é que elas continuam sendo lidas: como
// **omissão** de um tema que não declara a camada.
//
// A lista mora ao lado da função que faz a leitura, então um campo renomeado
// quebra a compilação em vez de virar um nome morto. `TestTokenDefaultFieldsMatchWhatIsRead`
// a compara com o que `defaultTokens` realmente toca, para que ela não envelheça.
func TokenDefaultFields() []string {
	return []string{
		"Background", "Foreground", "LineNumber", "StatusBG", "StatusFG",
		"StatusDirty", "CursorBG", "CursorFG",
	}
}

func defaultTokens(ui UIConfig) Tokens {
	return Tokens{
		Surface:      ui.Background,
		Panel:        ui.StatusBG,
		PanelText:    ui.StatusFG,
		BorderSubtle: ui.LineNumber,
		Text:         ui.Foreground,
		TextMuted:    ui.LineNumber,
		Accent:       ui.CursorFG,
		Success:      ui.CursorFG,
		Warning:      ui.StatusDirty,
		Error:        ui.StatusDirty,
		Focus:        ui.CursorBG,
	}
}

// NoColourTokens is the `no-color` theme's token layer: every intent resolves to
// nothing.
//
// É um **tema**, não um perfil, e a diferença é quem afirma. O perfil diz "este
// terminal não sabe fazer cor"; o tema diz "esta paleta não tem cor nenhuma", e
// uma paleta que afirma isso e entrega cor é uma mentira — que
// `TestNoColourThemeReallyPaintsNothing` faz o build rejeitar.
func NoColourTokens() Tokens { return Tokens{} }

// A token that resolves to nothing carries the literal `none`, and so does the
// empty string: um tema escrito à mão deixa o campo em branco, e obrigar o autor
// a escrever `none` seria um erro de digitação esperando acontecer.
const noColour = "none"

// IsNoColour reports whether a colour value means "no colour at all".
func IsNoColour(value string) bool {
	return value == "" || value == noColour
}

// NoColourTheme is the palette that says it has no colour at all.
//
// Registrado **fora** de `themes.go`, que é arquivo gerado do oráculo Rust: um
// tema que só existe no Go não pode morar num arquivo que o `--check` reescreve
// a partir do outro lado da migração. A mão que escreve aqui é a mesma que
// escreve as regras de aptidão: o que o produto ganhou e o oráculo não tem.
//
// Ele é um **tema**, não um perfil: o perfil desliga a cor porque o terminal
// não sabe fazer; este afirma que a paleta não tem cor nenhuma. A afirmação é
// verificável, e `TestNoColourThemeReallyPaintsNothing` é quem verifica.
func NoColourTheme() ThemeDefinition {
	tokens := NoColourTokens()
	// O nome é **"No Color"**, e a grafia americana é a que o resto do produto usa:
	// o id derivado é `no-color`, que é como o plano do M6 escreve, e como o
	// usuário digita. "No Colour" normalizaria para `no-colour`, e o tema existiria
	// sob um nome que ninguém escreveria.
	return ThemeDefinition{Name: "No Color", IsDark: true, Tokens: &tokens}
}
