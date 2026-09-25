// Package tui is the presentation layer, on Bubble Tea v2.
//
// It turns the state of internal/app into a frame and input into messages, and
// does nothing else. No editing rule, no text-layout rule and no protocol lives
// here: a decision that has to hold when there is no terminal belongs in the
// model.
//
// model.go is the composition root, and the only file in this package allowed to
// import internal/app — rule R3 in internal/architecture fails the build
// otherwise. Surfaces receive what they need as data, which is what makes one
// deletable by removing a folder and the line that calls it.
package tui

import (
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/app"
	"github.com/ori-team/oride/internal/buffer"
	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/editor"
	"github.com/ori-team/oride/internal/git"
	"github.com/ori-team/oride/internal/i18n"
	"github.com/ori-team/oride/internal/keymap"
	"github.com/ori-team/oride/internal/search"
	"github.com/ori-team/oride/internal/syntax"
	"github.com/ori-team/oride/internal/term"
	"github.com/ori-team/oride/internal/tui/component"
	"github.com/ori-team/oride/internal/tui/editorview"
	"github.com/ori-team/oride/internal/tui/findbar"
	"github.com/ori-team/oride/internal/tui/focus"
	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/menubar"
	"github.com/ori-team/oride/internal/tui/overlay"
	"github.com/ori-team/oride/internal/tui/panel"
	"github.com/ori-team/oride/internal/tui/statusbar"
	"github.com/ori-team/oride/internal/tui/tabs"
	"github.com/ori-team/oride/internal/tui/terminal"
	"github.com/ori-team/oride/internal/tui/theme"
	"github.com/ori-team/oride/internal/tui/tree"
)

// windowTitle is what the terminal title bar shows.
const windowTitle = "oride"

// Model is the Bubble Tea model.
//
// It owns what is presentation: the measured size, which surface holds focus, and
// the graph that decides where focus goes next. Everything else lives in app,
// which is the single implementation of the behaviour.
type Model struct {
	application *app.App
	keys        *keymap.Map
	graph       focus.Graph

	// catalog supplies the menu labels, so no surface holds hard-coded text.
	catalog i18n.Catalog
	// theme is resolved once, from configuration and a profile, and handed to
	// every surface. Resolving it per frame would mean rebuilding the same styles
	// on every keystroke.
	theme theme.Theme

	size    layout.Size
	surface focus.Surface

	// The overlay takes every keystroke while it is open. Its filter and
	// selection live here rather than inside a surface because they are state,
	// and state belongs where the dump can see it.
	overlay overlay.Model

	// highlight holds the syntax roles of the active document. They are computed
	// in a Cmd, not while rendering, because a keystroke must not pay for the
	// whole file.
	highlight highlightCache

	// pty is the shell behind the terminal panel, or nil when none was started.
	//
	// A handle, e não um processo criado aqui: o modelo não cria processo
	// nenhum, porque o harness de conformidade roda o mesmo modelo sem terminal e
	// sem shell. Quem cria é a raiz de composição, numa `Cmd`, e o resultado volta
	// por mensagem — o mesmo caminho do realce.
	pty *term.Terminal

	// query and replace are the search bar's text fields, from the component
	// `textinput`. As strings canônicas continuam no modelo — é de lá que o dump
	// do oráculo lê — e o componente é quem escreve, apaga e rola a cauda.
	query   textinput.Model
	replace textinput.Model
}

// highlightCache is the roles of one version of one document.
//
// Keyed by version because a tokenise that finishes after the next keystroke
// describes a document that no longer exists: painting it would put the colours
// of the old text on the new. The path is part of the key because renaming a
// document changes its language without changing a single byte of its version.
type highlightCache struct {
	document editor.ID
	version  uint64
	path     string
	lines    []syntax.Line
	filled   bool
}

// highlightedMsg is the answer of a tokenise, coming back from its Cmd.
type highlightedMsg struct {
	document editor.ID
	version  uint64
	path     string
	lines    []syntax.Line
}

// New builds the model over an application.
func New(application *app.App, keys *keymap.Map) Model {
	return Model{
		application: application,
		keys:        keys,
		graph:       focus.New(),
		catalog:     i18n.LoadRegistry(application.Workspace).Get(i18n.ParseID(application.Config.Locale)),
		theme:       resolveTheme(application),
		surface:     focus.Editor,
		query:       newFindField(),
		replace:     newFindField(),
	}
}

// newFindField builds one field of the search bar.
//
// **O cursor virtual fica desligado**, e a razão é o golden: o cursor do
// `textinput` pisca, e uma saída que depende do relógio é uma saída que um golden
// fixaria por acidente numa máquina e reprovaria noutra. Desligado, o campo
// desenha texto e nada mais — que é o que a barra já fazia, porque o cursor do
// terminal é do runtime.
//
// A consequência é uma dívida dita: **a barra de busca não tem cursor visível**.
// Arrumar isso é posicionar o cursor real do runtime sobre o campo, e é trabalho
// de M4 — não um detalhe que se resolve trocando o componente.
func newFindField() textinput.Model {
	field := textinput.New()
	field.Prompt = ""
	field.Placeholder = "digite…"
	// Um a menos, porque o componente reserva uma célula para o cursor — que aqui
	// está desligado. Pedir 28 e receber 29 deslocaria as flags da busca em
	// relação a tudo que a barra já desenhava, e a barra não é a dona do campo.
	field.SetWidth(findbar.FieldWidth - 1)
	field.SetVirtualCursor(false)
	// Os estilos do componente são **zerados**, e não os do tema. O campo é
	// desenhado dentro de uma linha que a barra já pinta com o fundo dela; um
	// estilo dentro do estilo é cor dentro de cor, e o pior caso é o perfil sem
	// cor, onde o componente emittingia sequências de escape que o perfil promete
	// não emitir. `TestNoColourFramesCarryNoEscapeSequences` é o que mede.
	field.SetStyles(textinput.Styles{})
	// O componente só aceita teclas quando está com foco. Sem isto, a barra
	// ignora a digitação inteira.
	field.Focus()
	return field
}

// WithProfile rebuilds the theme for a colour profile.
//
// A method rather than a constructor argument because the profile comes from the
// terminal the program is actually talking to, which is known after the model
// exists — and injecting it keeps a test off the process environment.
func (m Model) WithProfile(profile theme.Profile) Model {
	m.theme = theme.New(m.palette().UI, m.palette().Syntax, profile)
	return m
}

// palette resolves the colours the theme is built from.
//
// The named theme supplies them; Config.UI and Config.Syntax are the user's
// overrides on top. Reading Config.UI alone — which an earlier version did — gives
// an empty palette, because the shipped configuration names a theme instead of
// spelling out its colours, and the frame comes out monochrome with every test
// still passing.
func (m Model) palette() config.ThemeDefinition {
	return paletteOf(m.application)
}

// resolveTheme builds the theme a fresh model starts with.
func resolveTheme(application *app.App) theme.Theme {
	definition := paletteOf(application)
	return theme.New(definition.UI, definition.Syntax, theme.TrueColor)
}

// paletteOf resolves the effective theme definition for an application.
func paletteOf(application *app.App) config.ThemeDefinition {
	registry := config.LoadThemeRegistry(application.Workspace)
	if definition, found := registry.Get(config.NormalizeThemeName(application.Config.Theme)); found {
		return definition
	}
	// A theme name that resolves to nothing falls back to the configuration's own
	// colours rather than to an empty palette.
	return config.ThemeDefinition{UI: application.Config.UI, Syntax: application.Config.Syntax}
}

// Resize sets the measured size.
//
// Exported for callers that render without a runtime — the frame-printing
// command and the golden tests — so they do not have to fabricate a message to
// tell the model how big the terminal is.
func (m Model) Resize(width, height int) Model {
	m.size = layout.Size{Width: width, Height: height}
	return m
}

// Init performs no work.
//
// Init has no error path, so anything that can fail belongs in a Cmd that reports
// the failure as a message.
func (m Model) Init() tea.Cmd { return nil }

// Update handles one message.
//
// It never blocks: anything slow returns a Cmd. The switch is over message types
// rather than a chain of conditions, so an unhandled message is visibly unhandled.
//
// The highlighting is refreshed on the way out rather than inside a case: it
// depends on the document's version, not on the message that arrived, so a Cmd
// scheduled by any message — and by a message that only moved the window — has to
// be able to ask for it.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	// O foco do modelo e a superfície da apresentação são **a mesma** informação
	// em dois lugares, e uma ação move o do modelo: `Ctrl+`` leva para o
	// terminal, e sem esta linha a apresentação continuaria no editor — o painel
	// seguraria o teclado do shell enquanto a barra de status promete outra
	// coisa. É o caminho inverso do que `moveFocus` faz, e ele precisa existir
	// tanto quanto o outro.
	next.syncSurface()
	return next, tea.Batch(cmd, next.clipboardWrite(), next.highlightRoles(),
		next.terminalCmd(), next.gitCmd())
}

// syncSurface follows the model's focus into the presentation.
//
// Um método, e não uma linha no `handleKey`: qualquer caminho que mude o foco
// sem passar pela tecla — a paleta, o painel, uma ação disparada por outra ação —
// precisa que a apresentação acompanhe, e o ponto único depois do despacho é o
// lugar onde isso é garantido.
func (m *Model) syncSurface() {
	switch m.application.Focus {
	case app.FocusTree:
		m.surface = focus.Tree
	case app.FocusTerminal:
		m.surface = focus.Terminal
	case app.FocusSCM:
		m.surface = focus.SCM
	case app.FocusEditor:
		// O editor do modelo cobre também as três superfícies de **chrome** —
		// menu, abas e barra de status — porque o modelo não as distingue.
		// Escrever "editor" aqui apagaria o passo do Tab entre elas, e o
		// perímetro de foco encolheria sem ninguém notar.
		switch m.surface {
		case focus.Tree, focus.Terminal, focus.SCM:
			m.surface = focus.Editor
		}
	}
}

// update is the message switch, without the highlighting that follows every one.
//
// It returns the model rather than the interface, so the caller does not have to
// assert its way back to the type it started from.
func (m Model) update(msg tea.Msg) (Model, tea.Cmd) {
	switch message := msg.(type) {
	case tea.WindowSizeMsg:
		m.size = layout.Size{Width: message.Width, Height: message.Height}
		return m, nil

	case tea.KeyPressMsg:
		m.handleKey(message)
		if m.application.Quit {
			// The model sets the flag and the runtime ends the program. Nothing
			// read the flag before, so the editor could not be closed at all.
			return m, tea.Quit
		}
		return m, nil

	case highlightedMsg:
		m.keepHighlight(message)
		return m, nil

	case gitStateMsg:
		m.application.GitBranch = message.branch
		m.application.GitAhead = message.ahead
		m.application.GitBehind = message.behind
		m.application.HasUpstream = message.tracked
		return m, nil

	case terminalReadyMsg:
		m.pty = message.pty
		m.application.TerminalAttached = message.pty != nil
		if message.err != nil {
			// Falha fechada: sem shell, o editor continua. Um painel que impede
			// abrir porque o shell não subiu seria um editor quebrado por um
			// detalhe do ambiente.
			m.application.Status = "terminal indisponível: " + message.err.Error()
		}
		return m, nil
	}
	return m, nil
}

// terminalReadyMsg is a shell coming up, or the reason it did not.
type terminalReadyMsg struct {
	pty *term.Terminal
	err error
}

// terminalPollMsg é o "houve leitura" do painel.
//
// A mensagem existe porque um PTY **notifica**: o shell escreve por um lado que o
// runtime não vê, e sem um sinal o painel ficaria com a saída do comando anterior
// até a próxima tecla. As linhas **não** viajam na mensagem: quem as tem é o
// handle, e copiá-las seria um segundo lugar com o mesmo scrollback.
type terminalPollMsg struct{}

// gitStateMsg is a branch read coming back from a `Cmd`.
//
// Sem campo de erro de propósito: `git` fora de um repositório é o caso comum —
// um arquivo novo, um workspace sem `.git` — e a resposta é uma barra sem branch,
// que é a verdade. Um erro aqui viraria status na tela por um `git` que ninguém
// pediu.
type gitStateMsg struct {
	branch  string
	ahead   int
	behind  int
	tracked bool
}

// gitCmd re-reads the workspace's branch, on a slow tick.
//
// **Lento e deliberado.** `git` é um processo, e a regra do projeto é que
// capacidade se injeta e não se lê do ambiente: um `git` por tecla seria um
// `fork` por tecla. Dois segundos é o que o GitHub Desktop e o VS Code usam, e é
// imperceptível para quem está lendo a branch.
//
// A leitura acontece na `Cmd`, nunca no caminho do frame: o modelo recebe o
// resultado por mensagem, e um `git` que falha **não** derruba o editor — a
// barra simplesmente não mostra a branch.
func (m Model) gitCmd() tea.Cmd {
	workspace := m.application.Workspace
	if workspace == "" {
		return nil
	}
	return tea.Tick(gitPollInterval, func(time.Time) tea.Msg {
		branch := git.Branch(workspace)
		message := gitStateMsg{branch: branch}
		if branch != "" {
			if ahead, behind, ok := git.AheadBehind(workspace); ok {
				message.ahead, message.behind, message.tracked = ahead, behind, true
			}
		}
		return message
	})
}

// gitPollInterval is how often the branch is re-read.
const gitPollInterval = 2 * time.Second

// terminalCmd schedules the shell when the panel needs one, or a refresh of the
// rows when it already has it.
//
// Uma `Cmd` só, com os dois casos: pedir a criação e pedir a leitura ao mesmo tempo
// seria dois comandos disputando o mesmo campo, e o que chegasse por último
// decidiria se existe terminal.
func (m Model) terminalCmd() tea.Cmd {
	// A PTY só existe quando a aba **do terminal** está à frente: pedir um
	// processo para uma aba de busca é abrir um shell que ninguém vê.
	if !m.application.ShowBottom || m.application.BottomTab != app.BottomTabTerminal {
		return nil
	}

	shell := m.application.Config.Terminal.Shell
	workspace := m.application.Workspace
	height := m.application.TerminalHeight

	if m.pty == nil {
		return func() tea.Msg {
			// A PTY nasce aqui, e não no modelo: um processo criado no caminho do
			// Updateabriria um shell a cada tecla num harness headless.
			pty, err := term.Spawn(workspace, terminalColumns, uint16(height), shell)
			return terminalReadyMsg{pty: pty, err: err}
		}
	}

	pty := m.pty
	return func() tea.Msg {
		// A espera é do componente, não nossa: `Poll` esvazia o canal do leitor, e
		// perguntar antes de haver leitura seria um laço que nunca para. O atraso
		// curto é o que impede que o painel fique colado no processador com o
		// painel aberto.
		pty.Poll()
		time.Sleep(terminalPollInterval)
		return terminalPollMsg{}
	}
}

// The panel's shell geometry and how often it is read.
const (
	terminalColumns      = 80
	terminalPollInterval = 40 * time.Millisecond
)

// keepHighlight stores the roles that still describe the document on screen.
//
// An answer for a version that has moved on is dropped rather than painted: it
// is not stale by a little, it describes text the user has already changed.
func (m *Model) keepHighlight(message highlightedMsg) {
	document, err := m.application.Store.Active()
	if err != nil {
		return
	}
	if document.ID() != message.document || document.Version() != message.version {
		return
	}
	m.highlight = highlightCache{
		document: message.document,
		version:  message.version,
		path:     message.path,
		lines:    message.lines,
		filled:   true,
	}
}

// clipboardWrite turns a copy into a command the terminal can act on.
//
// The model holds the OSC 52 sequence and the composition root writes it, because
// the model does not own stdout: writing an escape from there would put escapes in
// the conformance dump, and the harness compares that dump byte for byte.
func (m Model) clipboardWrite() tea.Cmd {
	sequence := m.application.ClipboardWrite
	if sequence == "" {
		return nil
	}
	m.application.ClipboardWrite = ""
	return tea.Println(sequence)
}

// highlightRoles schedules the tokenise of the active document, if it is needed.
//
// The text is read on this side and lexed on the other: reading the buffer from
// the Cmd's goroutine would race the keystroke that is being handled right now,
// and a data race is not a price worth paying for a lexer. What crosses the
// boundary is a copy of the document — one pass over its bytes per version, not
// per frame.
func (m Model) highlightRoles() tea.Cmd {
	document, err := m.application.Store.Active()
	if err != nil {
		return nil
	}
	id, version := document.ID(), document.Version()
	path, _ := document.Path()
	if m.highlight.filled && m.highlight.document == id &&
		m.highlight.version == version && m.highlight.path == path {
		return nil
	}

	highlighter := syntax.For(path, m.application.Config.Languages)
	lines := linesOf(document)
	return func() tea.Msg {
		return highlightedMsg{
			document: id,
			version:  version,
			path:     path,
			lines:    highlighter.Highlight(lines),
		}
	}
}

// linesOf is the whole document, one string per line, without line breaks.
func linesOf(document *editor.Document) []string {
	source := document.Buffer()
	lines := make([]string, 0, source.LineCount())
	for line := range source.LineCount() {
		text, err := source.LineText(line)
		if err != nil {
			break
		}
		lines = append(lines, strings.TrimRight(text, "\n"))
	}
	return lines
}

// handleKey routes one keystroke.
//
// Focus traversal is handled before the keymap, because Tab is navigation in
// every surface and must not be remappable into an editing action. Everything
// else goes through the keymap, and a keystroke the keymap does not resolve is
// text — which is what makes typing work at all.
func (m *Model) handleKey(key tea.KeyPressMsg) {
	// The first rule of the focus graph: an active overlay captures every
	// keystroke. Nothing below runs while one is open — not the focus traversal,
	// not the keymap, not typing — because a key typed into a filter that also
	// edited the buffer behind it would be a data-loss bug, not a nuisance.
	if m.capturesInput() {
		m.handleOverlayKey(key)
		return
	}

	// O runtime e a tabela falam línguas diferentes em alguns nomes — `pgdown` e
	// `pagedown` — e a tradução é feita por `keymap.Parse`, dentro de `ApplyKey`.
	// Ninguém traduz aqui: duas traduções são duas verdades, e a segunda erra sem
	// aviso.
	chord := key.Keystroke()

	// Esc devolve o foco ao editor **com ou sem shell**: o painel promises
	// "Esc=editor" na barra de status, e um Esc que não faz nada quando a PTY
	// ainda não subiu seria a mesma teimosia de um painel que não fecha.
	if m.surface == focus.Terminal && (chord == "escape" || chord == "esc") {
		m.surface = focus.Editor
		m.application.Focus = app.FocusEditor
		return
	}

	// O terminal tem o teclado antes de qualquer outra coisa: com o foco nele, a
	// tecla é **do shell** — é o que o painel promete na barra de status. Sem
	// esta ordem, digitar no painel editava o documento por trás.
	if m.surface == focus.Terminal && m.pty != nil {
		if m.writeToShell(key, chord) {
			return
		}
	}

	// The focused surface gets the key first. Before this, the arrows moved the
	// document caret whatever the focus was, and the tree could be looked at but
	// not walked.
	if m.handleFocusedSurfaceKey(chord) {
		return
	}

	switch chord {
	case "tab":
		m.moveFocus(false)
		return
	case "shift+tab":
		m.moveFocus(true)
		return
	}

	if chord != "" {
		if m.openOverlayFor(chord) {
			return
		}
		if err := m.application.ApplyKey(chord); err == nil {
			return
		}
	}

	// The key carries its own text. Keystroke() is a display form and comes back
	// empty for a key that only has Text set, which is exactly how a plain letter
	// arrives — so resolving the character from Keystroke() alone would type
	// nothing at all.
	text := key.Text
	if text == "" && isTypable(chord) {
		text = chord
	}
	if text != "" {
		_ = m.application.ApplyText(text)
	}
}

// writeToShell sends one keystroke to the shell, and reports whether it did.
//
// Esc volta para o editor — é a única tecla que **não** vai para o shell, e é o
// que a barra de status promete ("Esc=editor"). Ctrl+C também não: no shell, ele
// interrompe o que estiver rodando, e roubar a tecla para o editor transformaria
// um comando longo em algo que não se interrompe.
func (m *Model) writeToShell(key tea.KeyPressMsg, chord string) bool {
	switch chord {
	case "escape", "esc":
		m.surface = focus.Editor
		m.application.Focus = app.FocusEditor
		return true
	case "ctrl+c":
		return true
	}

	switch chord {
	case "enter":
		return m.writeShell("\r") == nil
	case "backspace":
		return m.writeShell("\x7f") == nil
	case "tab":
		return m.writeShell("\t") == nil
	case "up", "down", "left", "right", "home", "end", "delete", "pgup", "pgdown", "pagedown", "pageup":
		// Uma tecla de navegação é enviada como o byte que o shell espera; as
		// sequências CSI são o trabalho do terminal, que aqui é o próprio shell.
		if sequence, ok := shellSequences[chord]; ok {
			return m.writeShell(sequence) == nil
		}
		return false
	}

	if key.Text != "" && isTypable(chord) {
		return m.writeShell(key.Text) == nil
	}
	return false
}

// writeShell writes bytes to the shell, reporting the failure in the status bar.
func (m *Model) writeShell(data string) error {
	writeErr := m.pty.WriteString(data)
	if writeErr != nil {
		m.application.Status = "terminal: " + writeErr.Error()
	}
	return writeErr
}

// shellSequences is what a navigation key sends to the shell, as the bytes a
// line editor reads: o shell que está do outro lado é quem interpreta.
var shellSequences = map[string]string{
	"up":       "\x1b[A",
	"down":     "\x1b[B",
	"right":    "\x1b[C",
	"left":     "\x1b[D",
	"home":     "\x1b[H",
	"end":      "\x1b[F",
	"delete":   "\x1b[3~",
	"pgup":     "\x1b[5~",
	"pgdown":   "\x1b[6~",
	"pageup":   "\x1b[5~",
	"pagedown": "\x1b[6~",
}

// moveFocus walks the graph and records the result in the model.
func (m *Model) moveFocus(backwards bool) {
	next := m.graph.Next(m.surface, m.surfaceVisible, backwards)
	m.surface = next

	// The graph speaks the presentation vocabulary; the model speaks its own.
	// Syncing here is what keeps a single source of truth for the dump while the
	// graph stays independent of it.
	m.application.Focus = focusToModel(next)
}

// surfaceVisible reports whether a surface may hold focus.
//
// The editor and the chrome are always drawn. The panels are not, and focus must
// never rest where nothing is painted.
func (m Model) surfaceVisible(surface focus.Surface) bool {
	switch surface {
	case focus.Editor, focus.MenuBar, focus.Tabs, focus.StatusBar:
		return true
	case focus.Tree:
		return m.application.ShowTree
	case focus.SCM:
		return m.application.SCM
	case focus.Terminal:
		// O painel existe quando o modelo diz que existe, **e** quando a aba
		// mostrada é a do terminal: com a aba de busca à frente, o teclado do shell
		// iria para uma superfície invisível — que é o bug de foco que a regra
		// "foco nunca descansa onde nada é pintado" existe para impedir.
		return m.application.ShowBottom && m.application.BottomTab == app.BottomTabTerminal
	}
	return false
}

// focusToModel maps a surface onto the model's focus vocabulary.
func focusToModel(surface focus.Surface) app.Focus {
	switch surface {
	case focus.Tree:
		return app.FocusTree
	case focus.Terminal:
		return app.FocusTerminal
	case focus.SCM:
		return app.FocusSCM
	default:
		return app.FocusEditor
	}
}

// isTypable reports whether a keystroke is a single printable character.
//
// A named key that the keymap did not resolve — F5, Escape, a bare modifier — must
// not be inserted as text, or an unbound key would type its own name into the
// buffer.
func isTypable(chord string) bool {
	runes := []rune(chord)
	return len(runes) == 1 && runes[0] >= 0x20 && runes[0] != 0x7f
}

// View returns the declarative view.
//
// Terminal capability is declared, not toggled per frame: the runtime reads this
// and reconciles. Toggling imperatively is how a terminal is left in the alternate
// screen after a crash.
func (m Model) View() tea.View {
	view := tea.NewView(m.Frame())
	view.AltScreen = true
	view.WindowTitle = windowTitle
	view.MouseMode = m.mouseMode()
	return view
}

// mouseMode derives the mode from configuration.
//
// Off by default, and off means off: an editor that captures the mouse without
// being asked takes text selection away from the terminal.
func (m Model) mouseMode() tea.MouseMode {
	if !m.application.Config.Mouse {
		return tea.MouseModeNone
	}
	return tea.MouseModeCellMotion
}

// Frame renders one screen.
//
// Exported so tests and golden frames can call it without a running tea.Program:
// the frame is a pure function of the model, and a golden file should exercise the
// function rather than the runtime.
//
// The frame comes back exactly as tall and as wide as the terminal. A frame one
// row short leaves the previous one's remnants on screen, and a row one cell wide
// shifts everything to its right.
func (m Model) Frame() string {
	regions := layout.Compute(m.size, m.wants())
	if regions.Editor.Empty() {
		return ""
	}

	// A paginação precisa da altura do editor, e só a apresentação a conhece. É
	// injeção, não leitura de ambiente: o mesmo caminho serve para o runtime e
	// para um teste que quer uma página de dez linhas.
	m.application.EditorHeight = regions.Editor.Height

	rows := make([]string, 0, m.size.Height)
	rows = append(rows, menubar.Render(m.size.Width, m.menuView()))
	rows = append(rows, tabs.Render(m.size.Width, m.tabsView()))
	rows = append(rows, m.bodyRows(regions)...)
	rows = append(rows, statusbar.Render(m.size.Width, m.statusView()))

	for len(rows) < m.size.Height {
		rows = append(rows, strings.Repeat(" ", m.size.Width))
	}
	if len(rows) > m.size.Height {
		rows = rows[:m.size.Height]
	}

	if m.overlay.Active() {
		rows = m.paintOverlay(rows, regions)
	}
	return strings.Join(rows, "\n")
}

// bodyRows composes the tree and the editor, side by side or overlaid.
func (m Model) bodyRows(regions layout.Regions) []string {
	editorRows := editorview.Render(regions.Editor.Width, regions.Editor.Height, m.editorView(regions))

	if !regions.Tree.Empty() {
		treeRows := tree.Render(regions.Tree.Width, regions.Tree.Height, m.treeView())

		if regions.TreeOverlaid {
			for index, treeRow := range treeRows {
				if index >= len(editorRows) {
					break
				}
				editorRows[index] = overlayCells(editorRows[index], treeRow, regions.Tree.X)
			}
		} else {
			for index := range editorRows {
				treeRow := ""
				if index < len(treeRows) {
					treeRow = treeRows[index]
				}
				editorRows[index] = layout.Pad(treeRow, regions.Tree.Width) + editorRows[index]
			}
		}
	}

	// O painel inferior entra por último e em **largura cheia**: ele fica abaixo da
	// árvore, não ao lado, e por isso não leva o prefixo dela — que é exatamente
	// o defeito que apareceu quando o terminal entrou antes da composição.
	if !regions.Bottom.Empty() {
		for _, row := range m.panelRows(regions.Bottom) {
			editorRows = append(editorRows, layout.Pad(row, regions.Bottom.Width))
		}
	}
	return editorRows
}

// menuView builds the menu bar's view from the locale catalog.
func (m Model) menuView() menubar.View {
	return menubar.View{
		Labels: menubar.LabelsFor(
			m.catalog.Menu.File, m.catalog.Menu.Edit, m.catalog.Menu.View,
			m.catalog.Menu.Go, m.catalog.Menu.Git, m.catalog.Menu.Help,
		),
		Open: -1,
	}
}

// tabsView builds the tab bar's view from the open documents.
func (m Model) tabsView() tabs.View {
	summaries := m.application.Store.TabSummaries()
	active, hasActive := m.application.Store.ActiveID()

	list := make([]tabs.Tab, 0, len(summaries))
	for _, summary := range summaries {
		list = append(list, tabs.Tab{
			Title:  summary.Title,
			Dirty:  summary.Dirty,
			Active: hasActive && summary.ID == active,
		})
	}
	return tabs.View{Tabs: list, Focused: m.surface == focus.Tabs, Theme: m.theme}
}

// treeView builds the panel's view from the project tree.
func (m Model) treeView() tree.View {
	if m.application.Tree == nil {
		return tree.View{Focused: m.surface == focus.Tree}
	}

	rows := m.application.Tree.FlatRows()
	list := make([]tree.Row, 0, len(rows))
	for _, row := range rows {
		list = append(list, tree.Row{
			Depth: row.Depth, Name: row.Name, IsDir: row.IsDir, Expanded: row.Expanded,
		})
	}
	return tree.View{
		Rows:     list,
		Selected: m.application.Tree.SelectedIndex(),
		Focused:  m.surface == focus.Tree,
	}
}

// panelRows draws the bottom panel: a faixa de abas e o conteúdo da aba ativa.
//
// A aba decide o que há dentro, e o painel decide o **tamanho** — a faixa come
// uma linha em todas, e é ela que diz o que está à vista.
func (m Model) panelRows(region layout.Region) []string {
	view := panel.View{
		Tabs:         panel.AllTabs(),
		Active:       panel.Tab(m.application.BottomTab),
		Focused:      m.surface == focus.Terminal,
		HasProblems:  m.application.LSPAttached,
		ProblemCount: m.application.Diagnostics,
		Theme:        m.theme,
	}

	width := region.Width
	height := max(region.Height-panel.TabStripHeight, 0)
	switch m.application.BottomTab {
	case app.BottomTabSearch:
		view.Content = findbar.Render(width, m.findView())
	case app.BottomTabProblems:
		view.Content = panel.ProblemsEmptyLines(width,
			m.application.Config.LSP.Enabled, m.application.LSPAttached)
	default:
		view.Content = terminal.Render(width, height, m.terminalView())
	}
	return panel.Render(width, region.Height, view)
}

// terminalView is the panel's view, with the rows the shell has written.
//
// The rows come from the handle the composition root owns; the model never
// touches the PTY, so this is the same shape every other surface receives — data,
// not a process.
func (m Model) terminalView() terminal.View {
	view := terminal.View{
		Focused:  m.surface == focus.Terminal,
		Attached: m.application.TerminalAttached,
		Theme:    m.theme,
	}
	if m.pty != nil {
		view.Rows = m.pty.VisibleLines(m.application.TerminalHeight)
		view.LastError = m.pty.LastError()
	}
	return view
}

// editorView builds the viewport's view, extracting only the visible lines.
func (m Model) editorView(regions layout.Regions) editorview.View {
	view := editorview.View{
		Gutter:  m.application.Config.ShowLineNumbers,
		Focused: m.surface == focus.Editor,
		// O wrap é o padrão do produto, como no Zed: linha longa é o caso comum
		// de código, e cortar o fim dela sem caminho para ele é o pior defeito de
		// leitura que um editor pode ter. `soft_wrap = false` desliga.
		Wrap:  m.application.Config.SoftWrap,
		Theme: m.theme,
	}

	document, err := m.application.Store.Active()
	if err != nil {
		return view
	}
	if caret, caretErr := document.Caret(); caretErr == nil {
		view.Caret = editorview.Position{Line: caret.Line, Column: caret.Column}
		view.HasCaret = true
	}
	view.Selection = selectionOf(document)
	view.Matches, view.CurrentMatch = matchesOf(document, m.application.Find)
	view.Lines, view.Offset = m.visibleLines(document, regions.Editor.Height, view.Caret.Line)
	return view
}

// selectionOf converts the document's byte-range selection into the line and
// column coordinates the viewport paints in.
//
// The document keeps offsets in bytes, because that is the canonical index; the
// viewport paints cells, so the conversion happens here, once, at the boundary
// between the two — and a selection that cannot be resolved paints nothing rather
// than failing the frame.
func selectionOf(document *editor.Document) editorview.Selection {
	selection := document.Selection()
	if selection.IsEmpty() {
		return editorview.Selection{Empty: true}
	}

	start, startErr := document.Buffer().ByteToCaret(selection.Start())
	end, endErr := document.Buffer().ByteToCaret(selection.End())
	if startErr != nil || endErr != nil {
		return editorview.Selection{Empty: true}
	}

	return editorview.Selection{
		Start: editorview.Position{Line: start.Line, Column: start.Column},
		End:   editorview.Position{Line: end.Line, Column: end.Column},
	}
}

// matchesOf converts the search results into the coordinates the viewport paints
// in.
//
// The search keeps byte offsets because that is the canonical index; the viewport
// paints cells. The conversion happens here, once, at the boundary — and a match
// that cannot be resolved is dropped rather than failing the frame, because a
// stale offset is a normal consequence of editing while a search is open.
func matchesOf(document *editor.Document, found search.State) ([]editorview.Match, int) {
	if len(found.Matches) == 0 {
		return nil, -1
	}

	matches := make([]editorview.Match, 0, len(found.Matches))
	for _, match := range found.Matches {
		start, startErr := document.Buffer().ByteToCaret(buffer.Offset(match.Start))
		end, endErr := document.Buffer().ByteToCaret(buffer.Offset(match.End))
		if startErr != nil || endErr != nil {
			continue
		}
		matches = append(matches, editorview.Match{
			Start: editorview.Position{Line: start.Line, Column: start.Column},
			End:   editorview.Position{Line: end.Line, Column: end.Column},
		})
	}

	// The index has to be remapped: dropping a match shifts everything after it,
	// and marking the wrong result as current is worse than marking none.
	current := -1
	if found.Current >= 0 && found.Current < len(found.Matches) {
		current = found.Current
		if len(matches) != len(found.Matches) {
			current = -1
		}
	}
	return matches, current
}

// visibleLines extracts the rows the viewport can show, with the roles of their
// bytes when they are known.
//
// The model owns the window because it is state, and because extracting every line
// of the document on each frame would be O(file) per keystroke for a screen that
// shows forty rows.
//
// A line outside the highlighted range is plain. That happens for one frame after
// a document opens — the roles are still being computed — and it is the same
// trade every editor makes rather than blocking the first paint on a lexer.
func (m Model) visibleLines(document *editor.Document, height, caretLine int) ([]editorview.Line, int) {
	buffer := document.Buffer()
	start, end := component.Window(buffer.LineCount(), height, caretLine, 0)

	lines := make([]editorview.Line, 0, end-start)
	for line := start; line < end; line++ {
		text, err := buffer.LineText(line)
		if err != nil {
			break
		}
		visible := editorview.Line{Text: strings.TrimRight(text, "\n")}
		if line < len(m.highlight.lines) {
			visible.Tokens = m.highlight.lines[line].Tokens
		}
		lines = append(lines, visible)
	}
	return lines, start
}

// statusView builds the status bar's view.
func (m Model) statusView() statusbar.View {
	view := statusbar.View{
		Focus:       string(m.surface),
		Message:     m.application.Status,
		Branch:      m.application.GitBranch,
		Ahead:       m.application.GitAhead,
		Behind:      m.application.GitBehind,
		HasUpstream: m.application.HasUpstream,
		Theme:       m.theme,
	}
	// O estado do LSP é escrito por extenso, e a barra só fala dele quando o
	// usuário o **desligou**: um "LSP —" permanente na barra seria ruído que faz o
	// leitor parar de ler a barra inteira, e o que ele/configurou é o que a barra
	// deve relatar. A contagem de erros aparece só com cliente anexado — sem
	// servidor, "0 erros" é ausência de medição, não saúde.
	//
	// A lacuna real — ligado no config e sem cliente — é uma dívida de produto
	// (M4, sobe o cliente LSP), e ela é do PROJECT_STATE e do ledger, não de
	// cada frame.
	if !m.application.Config.LSP.Enabled {
		view.LSP = "LSP off"
	}

	document, err := m.application.Store.Active()
	if err != nil {
		return view
	}
	// O modo vem do mesmo resolvedor que faz o realce: uma barra que escreve "Rust"
	// ao lado de um arquivo realçado como Go é uma barra em que o leitor não
	// acredita.
	if path, hasPath := document.Path(); hasPath {
		view.Mode = syntax.LanguageOf(path, m.application.Config.Languages)
	}
	view.Diagnostics = m.application.Diagnostics
	view.HasDiagnostics = m.application.LSPAttached
	view.File = document.TabTitle()
	view.Dirty = document.IsDirty()
	if caret, caretErr := document.Caret(); caretErr == nil {
		// One-based for display: a reader counts lines from one, and a status bar
		// that says Ln 0 is a status bar that looks broken.
		view.CaretLine = caret.Line + 1
		view.CaretCol = caret.Column + 1
		view.HasCaret = true
	}
	return view
}

// capturesInput reports whether an overlay owns the keyboard.
//
// Two overlays exist and they are not the same thing: the TUI's own (the palette,
// which-key and help) and the model's, which the oracle dumps and which the search
// uses. Checking only the first was a real defect — opening the search bar set the
// model's overlay, the TUI saw none, and typing went into the document behind the
// bar.
func (m Model) capturesInput() bool {
	return m.overlay.Active() || m.application.Overlay != overlayNone
}

// handleFocusedSurfaceKey gives the focused surface its own keys.
//
// It reports whether the key was consumed. The tree is the first surface with a
// local keyboard, and the shape is the one the focus graph documents: local keys
// resolve before the global keymap, and an unconsumed key falls through to it.
func (m *Model) handleFocusedSurfaceKey(chord string) bool {
	if m.application.Focus != app.FocusTree || !m.application.ShowTree {
		return false
	}

	tree := m.application.Tree
	if tree == nil {
		return false
	}

	switch chord {
	case "up":
		tree.MoveSelection(-1)
		return true
	case "down":
		tree.MoveSelection(1)
		return true
	case "right":
		_ = tree.ExpandSelected()
		return true
	case "left":
		_ = tree.CollapseOrParent()
		return true
	case "enter":
		// Activating opens a file or toggles a directory, and the store takes
		// the path — the panel does not know about documents.
		if path, opened, err := tree.Activate(); err == nil && opened {
			_, _ = m.application.Store.OpenPath(path)
		}
		return true
	}
	return false
}

// handleFindKey routes a keystroke inside the search bar.
//
// The bar's own keys stay ours — close, next, previous, reveal the replacement:
// they are the product's semantics, not text editing. Everything else goes to the
// `textinput` component, which is where insertion, deletion by rune, the
// placeholder and the tail-scrolling live. A second implementation of those in
// this method is exactly what the S3 item exists to remove.
func (m *Model) handleFindKey(key tea.KeyPressMsg) {
	chord := key.Keystroke()

	switch chord {
	case "escape", "esc", "ctrl+c":
		m.closeFind()
		return
	case "enter":
		_ = m.application.Apply(action.FindNext)
		return
	case "shift+enter":
		_ = m.application.Apply(action.FindPrev)
		return
	case "tab":
		// The replacement field is revealed, not switched to: the query stays
		// visible while the replacement is typed, because the two are read
		// together.
		m.application.Find.ShowReplace = !m.application.Find.ShowReplace
		return
	}

	m.editFindField(key)
}

// closeFind fecha a busca e recolhe o painel **se** a busca era o que estava à
// frente.
//
// A busca é uma aba: fechar é tirar a aba da frente, e o painel some quando não
// sobra nenhuma. Um painel que fica vazio depois de Esc é um painel que mente
// sobre o que o programa está fazendo.
func (m *Model) closeFind() {
	m.application.Overlay = app.OverlayNone
	if m.application.BottomTab == app.BottomTabSearch {
		m.application.ShowBottom = false
	}
}

// editFindField hands the key to the active field and copies the result back into
// the model.
//
// The copy is one-directional and immediate: the model is the source of truth
// because the oracle's dump reads it, and the component is the editor. Two sources
// of truth that drift are the same class of bug as two places measuring width.
func (m *Model) editFindField(key tea.KeyPressMsg) {
	if m.application.Find.ShowReplace {
		updated, _ := m.replace.Update(key)
		m.replace = updated
		m.setFindReplace(m.replace.Value())
		return
	}

	updated, _ := m.query.Update(key)
	m.query = updated
	if m.application.Find.Query == m.query.Value() {
		// Uma tecla de navegação dentro do campo não muda a busca; recomputar
		// aqui gastaria uma varredura do documento por tecla.
		return
	}
	m.setFindQuery(m.query.Value())
}

// setFindQuery is the one place the query changes: the component's field and the
// model's search state, together.
//
// Two writers would be two truths, and a test (or a future action) that writes
// one of them would leave a bar whose text and whose results disagree. A method
// rather than two assignments because the invariant is the point.
func (m *Model) setFindQuery(query string) {
	m.query.SetValue(query)
	m.application.Find.Query = query
	m.recomputeFind()
}

// setFindReplace is the counterpart for the replacement field, which does not
// search: it is only typed and read.
func (m *Model) setFindReplace(replace string) {
	m.replace.SetValue(replace)
	m.application.Find.Replace = replace
}

// recomputeFind re-runs the search against the document.
//
// A missing document leaves the previous matches alone rather than clearing them:
// an empty buffer would otherwise look like a query that stopped matching.
func (m *Model) recomputeFind() {
	document, err := m.application.Store.Active()
	if err != nil {
		return
	}
	m.application.Find.Recompute(document.Buffer().String())
}

// overlayNone is the model's value for no overlay open.
//
// Derivado do pacote do modelo, e não declarado aqui: o valor é um contrato
// comparado com o Rust, e duas cópias da mesma string em dois pacotes é a
// divergência esperando o dia.
const (
	overlayNone = app.OverlayNone
	overlayFind = app.OverlayFind
)

// wants is what the model asks the layout to show.
func (m Model) wants() layout.Wants {
	return layout.Wants{
		ShowTree:     m.application.ShowTree,
		TreeWidth:    treeWidth,
		ShowBottom:   m.application.ShowBottom,
		BottomHeight: m.bottomHeight(),
	}
}

// findHeight is how many rows the search bar wants.
//
// The overlay name is the model's, and the extra row for the replacement field is
// the search state's: the bar grows when the field is revealed, and the layout is
// told before it divides the screen.
// bottomHeight is how many rows the panel asks for, **aba inclusive**.
//
// Cada aba pede o que precisa, e a faixa de abas come uma linha em todas: são
// três linhas para o terminal (o piso do layout) e três para a busca com o campo
// de substituição. O mínimo é o **piso do conteúdo mais a faixa**, e não o piso do
// conteúdo — pedir duas linhas para uma faixa de uma devolve um painel sem
// nenhuma linha de conteúdo, e o leitor vê abas e nada mais.
func (m Model) bottomHeight() int {
	if !m.application.ShowBottom {
		return 0
	}

	content := 0
	switch m.application.BottomTab {
	case app.BottomTabSearch:
		content = 1
		if m.application.Find.ShowReplace {
			content = 2
		}
	case app.BottomTabProblems:
		content = 1
	default:
		// O terminal tem o piso de três linhas do layout, e mais a faixa.
		height := m.application.TerminalHeight
		if height < minPanelHeight {
			height = m.application.Config.Terminal.DefaultHeight
		}
		content = max(height, minPanelHeight)
	}
	return content + panel.TabStripHeight
}

// minPanelHeight is the floor for the terminal's content: um painel de duas
// linhas não é um terminal, é um erro de layout.
const minPanelHeight = 3

// overlayFind is the value the model and the oracle use for the search overlay.
//
// The pickers' names come from the model too, and for the same reason.
const (
	overlayThemePicker  = app.OverlayThemePicker
	overlayLocalePicker = app.OverlayLocalePicker
)

// findView builds the search bar's view from the search state.
func (m Model) findView() findbar.View {
	found := m.application.Find

	return findbar.View{
		Field:         m.query.View(),
		ReplaceField:  m.replace.View(),
		ShowReplace:   found.ShowReplace,
		CaseSensitive: found.Options.CaseSensitive,
		IgnoreAccents: found.Options.IgnoreAccents,
		WholeWord:     found.Options.WholeWord,
		UseRegex:      found.Options.UseRegex,
		Current:       found.Current + 1,
		Total:         len(found.Matches),
		RegexError:    found.RegexError,
		Theme:         m.theme,
	}
}

// treeWidth is the default tree column until the width becomes configurable.
const treeWidth = 30

// overlayCells writes text over a row starting at a cell offset, replacing as many
// cells as the text occupies.
//
// It builds on layout.Slice rather than carrying its own walk: the copy here was
// written before that helper had three callers, and two implementations of cell
// arithmetic is one more place for a wide character to come out a column short.
func overlayCells(row, text string, at int) string {
	total := layout.Width(row)
	if at >= total || total <= 0 {
		return row
	}

	width := min(layout.Width(text), total-at)
	before := layout.Slice(row, 0, at)
	after := layout.Slice(row, at+width, total-at-width)

	return layout.Pad(before, at) + layout.Pad(text, width) + layout.Pad(after, total-at-width)
}

// paintOverlay draws the open overlay over the body.
//
// It goes over the body rather than over the whole frame, so the menu bar, the tabs
// and the status bar stay visible: the reader keeps knowing what is open and where
// the cursor is while a list is in front of them.
func (m Model) paintOverlay(rows []string, regions layout.Regions) []string {
	rendered := m.overlay.View()
	if rendered == "" {
		return rows
	}

	width, height := m.overlaySize()
	region := centerRegion(regions.Editor, width, height)
	if region.Empty() {
		return rows
	}

	for index, line := range strings.Split(rendered, "\n") {
		row := region.Y + index
		if row < 0 || row >= len(rows) || index >= region.Height {
			break
		}
		rows[row] = overlayCells(rows[row], layout.Pad(line, region.Width), region.X)
	}
	return rows
}

// centerRegion places a box in the middle of an area.
func centerRegion(area layout.Region, width, height int) layout.Region {
	width = min(width, area.Width)
	height = min(height, area.Height)
	if width <= 0 || height <= 0 {
		return layout.Region{}
	}
	return layout.Region{
		X:      area.X + (area.Width-width)/2,
		Y:      area.Y + (area.Height-height)/2,
		Width:  width,
		Height: height,
	}
}

// overlaySize is how much of the screen an overlay takes.
//
// Capped rather than proportional: a palette filling a 200-column terminal would
// be a wall of text with a filter at the bottom, and the surface behind it stops
// being visible — which is what makes it an overlay rather than another screen.
func (m Model) overlaySize() (int, int) {
	width := min(max(m.size.Width-8, 24), 76)
	height := min(max(m.size.Height-8, 6), 20)
	return width, height
}

// overlayContent maps an action onto the overlay it opens.
//
// Driven by the keymap rather than by hard-coded chords, so remapping the palette
// key still opens the palette.
func (m Model) overlayContent(bound action.Action) (overlay.Kind, []overlay.Item, string, bool) {
	keys := m.keys
	switch bound {
	case action.CommandPalette:
		return overlay.Palette, m.paletteItems(), m.catalog.Palette.PaletteHint, true
	case action.WhichKey:
		return overlay.WhichKey, bindingItems(keys), "Atalhos", true
	case action.Help, action.Welcome:
		return overlay.Help, bindingItems(keys), "Ajuda", true
	}
	return overlay.None, nil, "", false
}

// paletteItems is the command palette's rows, labelled in the session's locale.
//
// O rótulo vem do catálogo e o **id** viaja no `Value`: a lista precisa saber o que
// executar, e o catálogo sabe o que **chamar** a coisa. Uma paleta que mostra
// `toggle_tree` e outra que mostra "Alternar árvore" executam a mesma tecla — e a
// segunda é a que o leitor consegue ler.
func (m Model) paletteItems() []overlay.Item {
	commands := action.Palette()
	items := make([]overlay.Item, 0, len(commands))
	for _, command := range commands {
		items = append(items, overlay.Item{
			Label: m.catalog.Label(command.String()),
			Value: string(command),
		})
	}
	return items
}

// bindingItems lists every binding, sorted by chord by the keymap itself.
func bindingItems(keys *keymap.Map) []overlay.Item {
	// A nil keymap resolves nothing, and listing an empty table would look like a
	// keymap that lost its bindings.
	if keys == nil {
		return nil
	}

	bindings := keys.Bindings()
	items := make([]overlay.Item, 0, len(bindings))
	for _, binding := range bindings {
		items = append(items, overlay.Item{
			Label:  binding.Chord.String(),
			Detail: binding.Action.String(),
		})
	}
	return items
}

// handleOverlayKey routes a keystroke inside an open overlay.
//
// The list component owns the filter, the scrolling and the selection, so the key
// goes to it rather than being interpreted here — which is the whole reason for
// using it.
func (m *Model) handleOverlayKey(key tea.KeyPressMsg) {
	if m.application.Overlay == overlayFind {
		m.handleFindKey(key)
		return
	}

	// Escape has two owners and they must not both act: while the filter is open it
	// belongs to the filter, and closing on the first Escape would discard a
	// half-typed filter the reader meant to correct.
	chord := key.Keystroke()
	if chord == "escape" || chord == "esc" || chord == "ctrl+c" {
		if !m.overlay.Filtering() {
			m.overlay.Close()
			return
		}
	}

	// Uma tecla digitável **abre** o filtro. O componente só entra em modo filtro
	// pela tecla `/`, que é a convenção de lista; numa paleta de comandos o leitor
	// digita o que quer e Enter executa, e esperar `/` antes de cada busca seria uma
	// tecla a mais entre lembrar o comando e executá-lo.
	if !m.overlay.FilteringNow() && isTypable(key.Keystroke()) && key.Text != "" {
		m.overlay.StartFilter(key.Text)
		return
	}

	// Enter aplica a linha destacada — o caminho que faltava. A paleta listava
	// comandos e não rodava nenhum: um filtro que mostra o que se pode fazer e não
	// faz nada é a mesma mentira de um atalho sem tratamento.
	//
	// Enter aplica **com o filtro aberto também**, e a interceptação é antes da
	// lista: o componente gasta o Enter em "aceitar o filtro" (AcceptWhileFiltering),
	// e num seletor isso é uma tecla jogada fora entre digitar e escolher. Quem
	// filtra e aperta Enter quer o item, não o estado do filtro.
	if key.Keystroke() == "enter" {
		m.applyOverlaySelection()
		return
	}

	updated, _ := m.overlay.Update(key)
	m.overlay = updated
}

// applyOverlaySelection acts on the highlighted row and closes the overlay.
//
// Quem decide o que a linha significa é a raiz, e não a superfície: a lista devolve
// a escolha e o modelo sabe o que um id de ação, um nome de tema ou um locale
// provocam. Uma linha sem valor — a lista de ajuda — fecha sem executar, e fingir
// que executou seria a tecla fazendo nada com um clique.
func (m *Model) applyOverlaySelection() {
	kind := m.overlay.Kind()
	selected, ok := m.overlay.Selected()
	m.overlay.Close()
	if !ok || selected.Value == "" {
		return
	}

	switch {
	case kind == overlay.Theme:
		m.applyTheme(selected.Value)
		return
	case kind == overlay.Locale:
		m.applyLocale(selected.Value)
		return
	}

	parsed, err := action.Parse(selected.Value)
	if err != nil {
		// Um id que o registro não conhece é um bug de quem montou a lista, e
		// engolir seria o mesmo defeito das teclas sem tratamento.
		m.application.Status = "comando desconhecido: " + selected.Value
		return
	}
	_ = m.application.Apply(parsed)

	// O comando pode ter aberto outra sobreposição — o `which_key`, por exemplo.
	// Abrir a próxima **depois** de fechar a atual é o que impede a lista de
	// apagar a que acabou de nascer.
	if m.application.Overlay == overlayNone {
		return
	}
	width, height := m.overlaySize()
	m.overlay.Open(m.kindFor(m.application.Overlay), m.titleFor(m.application.Overlay),
		m.itemsFor(m.application.Overlay), width, height)
}

// kindFor, titleFor and itemsFor resolve one overlay by the model's name.
//
// Um lugar só para a tradução, porque três funções que traduzem o mesmo nome
// podem discordar — e o sintoma seria uma sobreposição que abre com o título de
// outra, que é o tipo de bug que o que o golden não pega.
func (m Model) kindFor(name string) overlay.Kind {
	switch name {
	case overlayThemePicker:
		return overlay.Theme
	case overlayLocalePicker:
		return overlay.Locale
	}
	// A barra de busca não é uma lista, e nenhuma outra sobreposição é nomeada pelo
	// modelo: as demais nascem de uma ação, e é `openOverlayFor` que as abre.
	return overlay.None
}

func (m Model) titleFor(name string) string {
	switch name {
	case overlayThemePicker:
		return "Temas"
	case overlayLocalePicker:
		return "Idioma"
	}
	return ""
}

func (m Model) itemsFor(name string) []overlay.Item {
	switch name {
	case overlayThemePicker:
		return m.themeItems()
	case overlayLocalePicker:
		return m.localeItems()
	}
	return nil
}

// themeItems is the picker's rows: every theme the registry knows, sorted, with
// the current one marked.
//
// A marca é **texto** e não cor: a regra do projeto é que nenhum estado é
// comunicado só por cor, e "qual tema está ativo" é exatamente o estado que um
// tema ruim torna invisível.
func (m Model) themeItems() []overlay.Item {
	registry := config.LoadThemeRegistry(m.application.Workspace)
	names := registry.ListNames()
	sort.Strings(names)

	// O rótulo é o **nome de exibição** ("Tokyo Night") e o valor é o nome
	// normalizado ("tokyo-night"), porque é o que `Config.Theme` guarda e o que o
	// registro resolve. Aplicar o nome de exibição resolve um tema que não existe,
	// e o sintoma é um seletor que "funciona" e não muda nada.
	current := config.NormalizeThemeName(m.application.Config.Theme)
	items := make([]overlay.Item, 0, len(names))
	for _, name := range names {
		detail := ""
		id := config.NormalizeThemeName(name)
		if id == current {
			detail = "ativo"
		}
		items = append(items, overlay.Item{Label: name, Detail: detail, Value: id})
	}
	return items
}

// localeItems is the locale picker's rows, each with the menu label the reader
// would recognise rather than the id.
func (m Model) localeItems() []overlay.Item {
	registry := i18n.LoadRegistry(m.application.Workspace)
	locales := registry.Available()
	sort.Slice(locales, func(i, j int) bool { return locales[i] < locales[j] })

	current := i18n.ParseID(m.application.Config.Locale)
	items := make([]overlay.Item, 0, len(locales))
	for _, locale := range locales {
		detail := ""
		if locale == current {
			detail = "ativo"
		}
		items = append(items, overlay.Item{Label: string(locale), Detail: detail, Value: string(locale)})
	}
	return items
}

// applyTheme adopts a theme and rebuilds the styles.
func (m *Model) applyTheme(name string) {
	m.application.Config.Theme = name
	m.theme = theme.New(m.palette().UI, m.palette().Syntax, m.theme.Profile())
	m.application.Status = "tema: " + name
}

// applyLocale adopts a locale and reloads the catalog.
func (m *Model) applyLocale(id string) {
	m.application.Config.Locale = id
	m.catalog = i18n.LoadRegistry(m.application.Workspace).Get(i18n.ParseID(id))
	m.application.Status = "idioma: " + id
}

// openOverlayFor opens the overlay a chord resolves to, if any.
func (m *Model) openOverlayFor(chord string) bool {
	parsed, err := keymap.Parse(chord)
	if err != nil {
		return false
	}
	bound, ok := m.keys.Resolve(parsed)
	if !ok {
		return false
	}

	// Um comando que abre uma sobreposição é aplicado pelo modelo, que é quem sabe
	// o nome dela; a paleta em si é aberta por aqui, porque é a única que não vem de
	// um comando e sim de um acorde.
	kind, items, title, opens := m.overlayContent(bound)
	if !opens {
		return false
	}

	width, height := m.overlaySize()
	m.overlay.Open(kind, title, items, width, height)
	return true
}
