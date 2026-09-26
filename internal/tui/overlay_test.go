package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/app"
	"github.com/ori-team/oride/internal/fs"
	"github.com/ori-team/oride/internal/keymap"
	"github.com/ori-team/oride/internal/tui/layout"
	"github.com/ori-team/oride/internal/tui/overlay"
)

// chordFor finds the chord bound to an action, so a test does not hard-code a
// keybinding that a remap would break.
func chordFor(t *testing.T, keys *keymap.Map, id action.Action) string {
	t.Helper()

	for _, binding := range keys.Bindings() {
		if binding.Action == id {
			return binding.Chord.String()
		}
	}
	t.Fatalf("nenhum binding para %s", id)
	return ""
}

// key builds a keystroke the model understands.
func key(text string) tea.KeyPressMsg { return tea.KeyPressMsg(tea.Key{Text: text}) }

// pressChord sends a chord as the runtime would, pelo `Update`.
//
// A versão anterior chamava `openOverlayFor` **direto**, o que pulava o `Update`
// inteiro — e foi exatamente esse atalho que escondeu o defeito que esta rodada
// conserta: a ação punha `Overlay = "OpenFile"` no modelo e a tela nunca olhava,
// porque quem abria a sobreposição era o método privado, não o `Update`. Um teste
// que chama a peça por fora testa a peça; a peça estava certa e o produto errado.
//
// A tecla é **montada** a partir do chord, e o `keymap.Parse` normaliza a caixa —
// então o teste exercita o mesmo caminho que o terminal exercita.
func pressChord(t *testing.T, model Model, chord string) Model {
	t.Helper()

	updated, _ := model.Update(tea.KeyPressMsg(keyEventFor(t, chord)))
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update devolveu %T", updated)
	}
	return next
}

// keyEventFor is the keystroke a terminal would send for a chord.
//
// Os casos que a tabela usa: uma letra com modificadores, ou uma tecla nomeada
// (up, enter, pgdown…). Qualquer outro token é erro do teste, não produto — e
// falhar aqui é melhor do que mandar uma tecla que o terminal nunca manda.
func keyEventFor(t *testing.T, chord string) tea.Key {
	t.Helper()

	parts := strings.Split(chord, "+")
	code := parts[len(parts)-1]
	event := tea.Key{}

	for _, modifier := range parts[:len(parts)-1] {
		switch modifier {
		case "ctrl":
			event.Mod |= tea.ModCtrl
		case "alt":
			event.Mod |= tea.ModAlt
		case "shift":
			event.Mod |= tea.ModShift
		default:
			t.Fatalf("modificador desconhecido em %q", chord)
		}
	}

	if named, found := namedKeyCodes()[code]; found {
		event.Code = named
		return event
	}

	runes := []rune(code)
	if len(runes) != 1 {
		t.Fatalf("chord %q não é uma tecla que o terminal manda", chord)
	}
	event.Code = runes[0]
	return event
}

// namedKeyCodes are the named keys the default keymap uses.
func namedKeyCodes() map[string]rune {
	return map[string]rune{
		"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
		"enter": tea.KeyEnter, "tab": tea.KeyTab, "esc": tea.KeyEscape, "space": tea.KeySpace,
		"backspace": tea.KeyBackspace, "delete": tea.KeyDelete, "home": tea.KeyHome, "end": tea.KeyEnd,
		"pgup": tea.KeyPgUp, "pgdown": tea.KeyPgDown,
		"f1": tea.KeyF1, "f2": tea.KeyF2, "f3": tea.KeyF3, "f4": tea.KeyF4, "f5": tea.KeyF5,
		"f6": tea.KeyF6, "f8": tea.KeyF8, "f12": tea.KeyF12,
	}
}

// TestOverlayCapturesEveryKeystroke is the first rule of the focus graph, and the
// most important test in this file: a key typed into a filter that also edited the
// buffer would be data loss, not a nuisance.
func TestOverlayCapturesEveryKeystroke(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Store.OpenEmpty()

	// Open the palette through the keymap, so the test exercises the real path.
	chord := chordFor(t, model.keys, action.CommandPalette)
	model = pressChord(t, model, chord)
	if !model.overlay.Active() {
		t.Fatalf("a palette não abriu com %q", chord)
	}

	// Type characters that would edit the buffer if the capture rule leaked.
	before := bufferText(t, model)
	for _, r := range "abc" {
		updated, _ := model.Update(key(string(r)))
		model = updated.(Model)
	}

	if got := bufferText(t, model); got != before {
		t.Errorf("o buffer mudou com a palette aberta: %q → %q", before, got)
	}

	// O filtro não é afirmado aqui: quem o guarda é o componente, e um teste meu
	// sobre ele estaria testando a biblioteca em vez deste código. O que este
	// teste guarda é a regra de captura, e ela está acima.
}

// TestClosingTheOverlayReturnsInputToSurfaces is the other direction: the capture
// must not outlive the overlay.
func TestClosingTheOverlayReturnsInputToSurfaces(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Store.OpenEmpty()
	openOverlay(&model, overlay.Palette)

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	model = updated.(Model)
	if model.overlay.Active() {
		t.Fatal("Esc não fechou a sobreposição")
	}

	updated, _ = model.Update(key("z"))
	model = updated.(Model)
	if got := bufferText(t, model); got != "z" {
		t.Errorf("depois de fechar, digitar não alcançou o buffer: %q", got)
	}
}

func bufferText(t *testing.T, model Model) string {
	t.Helper()

	document, err := model.application.Store.Active()
	if err != nil {
		return ""
	}
	return document.Buffer().String()
}

// TestOverlayFramesKeepTheExactSize: an overlay that changed the frame's
// dimensions would leave the previous frame's remnants around it.
func TestOverlayFramesKeepTheExactSize(t *testing.T) {
	for _, kind := range []overlay.Kind{overlay.Palette, overlay.WhichKey, overlay.Help} {
		for _, size := range []struct{ width, height int }{{60, 24}, {80, 30}, {140, 40}} {
			model := sized(t, newModel(t, nil), size.width, size.height)
			openOverlay(&model, kind)

			for index, row := range splitFrame(model.Frame()) {
				if got := layout.Width(row); got != size.width {
					t.Errorf("%v %dx%d: linha %d com %d células", kind, size.width, size.height, index, got)
				}
			}
		}
	}
}

// TestOverlayKeepsTheChromeVisible: the menu bar, tabs and status bar stay
// readable, because the reader keeps needing to know what is open and where the
// cursor is.
func TestOverlayKeepsTheChromeVisible(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	openOverlay(&model, overlay.Palette)

	rows := splitFrame(model.Frame())
	if len(rows) != 30 {
		t.Fatalf("linhas = %d, esperado 30", len(rows))
	}
	if !strings.Contains(rows[0], model.catalog.Menu.File) {
		t.Errorf("o menu bar sumiu: %q", rows[0])
	}
	if !strings.Contains(rows[29], string(model.surface)) {
		t.Errorf("a statusbar sumiu: %q", rows[29])
	}
}

// TestTheFindOverlayCapturesTyping is the capture rule applied to the model's
// overlay rather than the TUI's.
//
// They are two different things: opening the search bar sets the model's overlay,
// and an earlier version checked only the TUI's — so the bar appeared and typing
// went into the document behind it.
func TestTheFindOverlayCapturesTyping(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Store.OpenEmpty()
	document, _ := model.application.Store.Active()
	_ = document.InsertText("alfa beta\n")

	model.application.Overlay = overlayFind
	before := document.Buffer().String()

	for _, r := range "bet" {
		updated, _ := model.Update(key(string(r)))
		model = updated.(Model)
	}

	if got := document.Buffer().String(); got != before {
		t.Errorf("o documento mudou com a barra aberta: %q", got)
	}
	if got := model.application.Find.Query; got != "bet" {
		t.Errorf("consulta = %q, esperado \"bet\"", got)
	}
}

// TestTypingSearchesAsItGoes: the highlight has to follow what is being typed.
func TestTypingSearchesAsItGoes(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Store.OpenEmpty()
	document, _ := model.application.Store.Active()
	_ = document.InsertText("alfa beta alfa\n")

	model.application.Overlay = overlayFind
	for _, r := range "alfa" {
		updated, _ := model.Update(key(string(r)))
		model = updated.(Model)
	}

	if got := len(model.application.Find.Matches); got != 2 {
		t.Errorf("casamentos = %d, esperado 2", got)
	}
}

// TestBackspaceInTheQueryRemovesARuneNotAByte.
func TestBackspaceInTheQueryRemovesARuneNotAByte(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Overlay = overlayFind

	// Digitado, e não atribuído: o campo é do componente `textinput`, e um teste
	// que escreve no modelo sem escrever no campo mediria uma barra que o
	// usuário nunca vê.
	for _, r := range "aç" {
		updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: r, Text: string(r)}))
		model = updated.(Model)
	}
	if got := model.application.Find.Query; got != "aç" {
		t.Fatalf("consulta = %q, esperado \"aç\"", got)
	}

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	model = updated.(Model)

	if got := model.application.Find.Query; got != "a" {
		t.Errorf("consulta = %q, esperado \"a\"", got)
	}
	if got := model.query.Value(); got != "a" {
		t.Errorf("o campo do componente = %q, o modelo diz %q", got, "a")
	}
}

// TestEscapeClosesTheFindBar: closing is the only way the document gets input
// back, so it has to work from the bar.
func TestEscapeClosesTheFindBar(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Store.OpenEmpty()
	model.application.Overlay = overlayFind

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEscape}))
	model = updated.(Model)

	if model.capturesInput() {
		t.Error("Esc não devolveu a entrada às superfícies")
	}

	updated, _ = model.Update(key("z"))
	model = updated.(Model)
	if got := bufferText(t, model); got != "z" {
		t.Errorf("depois de fechar, digitar não alcançou o buffer: %q", got)
	}
}

// TestTabRevealsTheReplaceFieldAndKeepsTheQuery: the two are read together, so
// revealing the replacement must not hide what is being searched.
func TestTabRevealsTheReplaceFieldAndKeepsTheQuery(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Store.OpenEmpty()
	// A busca é aberta **pelo caminho real** — a ação — e não escrevendo no
	// modelo: escrever `Overlay` sem `BottomTab` produz uma busca que o painel não
	// mostra, e o teste mediria um estado que o leitor nunca vê.
	if err := model.application.Apply(action.Find); err != nil {
		t.Fatalf("abrindo a busca: %v", err)
	}
	model.application.Find.Query = "alfa"

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	model = updated.(Model)

	if model.application.BottomTab != "search" {
		t.Errorf("a busca não virou aba do painel: %q", model.application.BottomTab)
	}
	if !model.application.Find.ShowReplace {
		t.Error("Tab não revelou o campo de substituição")
	}
	if got := model.application.Find.Query; got != "alfa" {
		t.Errorf("a consulta foi perdida: %q", got)
	}
	// A altura agora é a do **painel**: o campo de substituição ocupa duas linhas
	// de conteúdo, e a faixa de abas come uma. É a diferença entre "a barra
	// cresceu" e "o painel pediu o tamanho certo".
	if got := model.bottomHeight(); got != 3 {
		t.Errorf("altura do painel = %d, esperado 3 com o campo revelado", got)
	}
}

// TestTypingGoesToTheReplaceFieldOnceRevealed.
func TestTypingGoesToTheReplaceFieldOnceRevealed(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Overlay = overlayFind
	model.application.Find.Query = "alfa"
	model.application.Find.ShowReplace = true

	for _, r := range "ALFA" {
		updated, _ := model.Update(key(string(r)))
		model = updated.(Model)
	}

	if got := model.application.Find.Replace; got != "ALFA" {
		t.Errorf("substituição = %q, esperado \"ALFA\"", got)
	}
	if got := model.application.Find.Query; got != "alfa" {
		t.Errorf("a consulta foi alterada: %q", got)
	}
}

// TestQuitIsReportedToTheRuntime is the defect a reader hits first: the model set
// the quit flag and nothing read it, so the editor could not be exited from
// inside.
//
// The key itself is not synthesised here. Building a Bubble Tea key event means
// reproducing its field encoding, and a test that did that would be checking the
// encoding rather than this code; the path from the keymap to the flag is covered
// in internal/app, where the command lives.
// TestQuitIsReportedToTheRuntime: setting the flag is not enough — the runtime has
// to be told, and that is what a Cmd is for.
func TestQuitIsReportedToTheRuntime(t *testing.T) {
	model := newModel(t, nil)
	model.application.Quit = true

	_, cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	if cmd == nil {
		t.Fatal("o modelo não devolveu Cmd de saída com Quit ligado")
	}

	// Sem o flag, a mesma tecla não pode encerrar nada.
	model.application.Quit = false
	if _, quiet := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab})); quiet != nil {
		t.Error("o modelo pediu para sair sem o flag ligado")
	}
}

// TestTheTreeCanBeWalked is the second thing a reader hits: the panel could be
// looked at and not walked, because the arrows moved the document caret whatever
// the focus was.
func TestTheTreeCanBeWalked(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x\n"), 0o644); err != nil {
			t.Fatalf("escrevendo: %v", err)
		}
	}

	model := sized(t, newModel(t, nil), 100, 30)
	tree, err := fs.Open(root, false)
	if err != nil {
		t.Fatalf("abrindo a árvore: %v", err)
	}
	model.application.Tree = tree
	model.application.ShowTree = true
	model.application.Focus = app.FocusTree

	before := tree.SelectedIndex()
	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(Model)

	if got := model.application.Tree.SelectedIndex(); got == before {
		t.Errorf("a seta não moveu a seleção da árvore: continua em %d", got)
	}
}

// TestTreeKeysDoNotReachTheDocument: when the tree has focus, an arrow moves the
// selection and leaves the caret alone.
func TestTreeKeysDoNotReachTheDocument(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("linha um\nlinha dois\n"), 0o644); err != nil {
		t.Fatalf("escrevendo: %v", err)
	}

	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Store.OpenPath(filepath.Join(root, "a.txt"))
	tree, err := fs.Open(root, false)
	if err != nil {
		t.Fatalf("abrindo a árvore: %v", err)
	}
	model.application.Tree = tree
	model.application.ShowTree = true
	model.application.Focus = app.FocusTree

	document, _ := model.application.Store.Active()
	before, _ := document.Caret()

	updated, _ := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	model = updated.(Model)

	after, _ := model.application.Store.Active()
	now, _ := after.Caret()
	if now != before {
		t.Errorf("a seta mexeu no cursor do documento: %v → %v", before, now)
	}
}
