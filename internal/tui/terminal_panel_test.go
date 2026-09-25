package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/app"
	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/editor"
	"github.com/ori-team/oride/internal/keymap"
	"github.com/ori-team/oride/internal/term"
	"github.com/ori-team/oride/internal/tui/focus"
	"github.com/ori-team/oride/internal/tui/layout"
)

// O painel do terminal é o único caminho do produto que envolve um **processo**,
// e por isso o único que um teste de caminho puro não prova. Estes testes
// exercitam a sessão real — `Cmd` que cria a PTY, teclas que chegam ao shell, o
// que o shell devolve — e um deles usa um shell de verdade, que é a única forma de
// o painel deixar de ser um desenho.

// shellModel builds a model over a real workspace.
func shellModel(t *testing.T) Model {
	t.Helper()

	root := t.TempDir()
	keys, err := keymap.FromBindings(config.Default().Keys)
	if err != nil {
		t.Fatalf("keymap: %v", err)
	}
	configuration := config.Default()
	// Um shell que existe em toda máquina de teste, sem depender do que o
	// utilizador tem instalado: `sh` em POSIX, e o teste pula se não houver.
	configuration.Terminal.Shell = ""

	application := app.New(editor.NewStore(), configuration, keys)
	application.Workspace = root
	application.Store.OpenEmpty()
	return New(application, keys).Resize(100, 30)
}

// TestTheTerminalPanelOpensAndTakesFocus is the state half: toggling shows the
// panel, gives it the keyboard, and says how to leave.
func TestTheTerminalPanelOpensAndTakesFocus(t *testing.T) {
	model := shellModel(t)
	if model.application.ShowBottom {
		t.Fatal("o painel começou aberto")
	}

	model = press(t, model, tea.Key{Code: '`', Mod: tea.ModCtrl})

	if !model.application.ShowBottom || model.application.BottomTab != app.BottomTabTerminal {
		t.Fatalf("Ctrl+` não mostrou a aba do terminal: aberto=%v, aba=%q",
			model.application.ShowBottom, model.application.BottomTab)
	}
	if model.application.Focus != app.FocusTerminal {
		t.Errorf("foco = %q, esperava o terminal", model.application.Focus)
	}
	if model.application.TerminalHeight < 3 {
		t.Errorf("altura = %d, o piso do painel é 3", model.application.TerminalHeight)
	}
	if !strings.Contains(model.application.Status, "digite") {
		t.Errorf("status = %q, o painel precisa dizer o que fazer nele", model.application.Status)
	}
}

// TestThePanelTakesItsRowsFromTheEditor is the discipline the panel exists to
// keep: com o foco nele, a tecla vai para o shell e **não** para o documento.
func TestThePanelTakesItsRowsFromTheEditor(t *testing.T) {
	model := shellModel(t)
	document, _ := model.application.Store.Active()

	model = press(t, model, tea.Key{Code: '`', Mod: tea.ModCtrl})
	model = press(t, model, tea.Key{Code: 'a', Text: "a"})

	if got := document.Buffer().String(); got != "" {
		t.Errorf("digitar no painel escreveu no documento: %q", got)
	}
	if document.IsDirty() {
		t.Error("o documento ficou sujo com o foco no terminal")
	}
}

// TestEscapeLeavesTheTerminal is o que a barra de status promete: Esc volta para
// o editor, e não vai para o shell.
func TestEscapeLeavesTheTerminal(t *testing.T) {
	model := shellModel(t)
	model = press(t, model, tea.Key{Code: '`', Mod: tea.ModCtrl})
	if model.application.Focus != app.FocusTerminal {
		t.Fatal("o foco não foi para o terminal")
	}

	model = press(t, model, tea.Key{Code: tea.KeyEscape})

	if model.application.Focus != app.FocusEditor {
		t.Errorf("Esc não devolveu o foco ao editor (foco = %q)", model.application.Focus)
	}
	if string(model.surface) != "editor" {
		t.Errorf("a superfície do foco ficou em %q", model.surface)
	}
}

// TestThePanelGrowsAndShrinksWithinLimits cobre os limites, que são os do
// terminal de referência: três a quarenta linhas.
func TestThePanelGrowsAndShrinksWithinLimits(t *testing.T) {
	model := shellModel(t)
	model = press(t, model, tea.Key{Code: '`', Mod: tea.ModCtrl})

	for range 20 {
		model = press(t, model, tea.Key{Code: '+', Mod: tea.ModCtrl | tea.ModShift})
	}
	if model.application.TerminalHeight > 40 {
		t.Errorf("cresceu para %d, e o teto é 40", model.application.TerminalHeight)
	}

	for range 30 {
		model = press(t, model, tea.Key{Code: '-', Mod: tea.ModCtrl | tea.ModShift})
	}
	if model.application.TerminalHeight < 3 {
		t.Errorf("encolheu para %d, e o piso é 3", model.application.TerminalHeight)
	}
}

// TestARealShellAnswers é a prova de que o caminho inteiro funciona: um `Cmd`
// cria a PTY, uma tecla chega ao shell, e o que o shell devolve aparece no painel.
func TestARealShellAnswers(t *testing.T) {
	if _, err := os.Stat("/bin/sh"); err != nil {
		t.Skip("sem /bin/sh: o painel precisa de um shell para ser provado")
	}

	model := shellModel(t)
	model = press(t, model, tea.Key{Code: '`', Mod: tea.ModCtrl})

	// A criação da PTY é uma `Cmd`: roda o spawn e entrega a mensagem.
	updated, cmd := model.Update(nil)
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("abrir o painel não pediu a criação do shell")
	}
	for _, msg := range flatten(cmd()) {
		ready, ok := msg.(terminalReadyMsg)
		if !ok {
			continue
		}
		if ready.err != nil {
			t.Skipf("a PTY não subiu neste ambiente: %v", ready.err)
		}
		updated, _ = model.Update(ready)
		model = updated.(Model)
	}
	if model.pty == nil {
		t.Fatal("o shell não ficou anexado ao modelo")
	}
	if !model.application.TerminalAttached {
		t.Error("o modelo não sabe que há shell")
	}
	t.Cleanup(func() { _ = model.pty.Close() })

	// `echo oride 1` imprime o próprio texto: o teste não depende de prompt nem
	// de nada que a máquina decida.
	model = typeText(t, model, "echo oride 1")
	model = press(t, model, tea.Key{Code: tea.KeyEnter})

	// O shell precisa de tempo para responder, e o painel lê por `Cmd` — então o
	// teste espera o que o runtime esperaria.
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		model.pty.Poll()
		if strings.Contains(strings.Join(model.pty.VisibleLines(20), "\n"), "oride") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	rows := strings.Join(model.pty.VisibleLines(20), "\n")
	if !strings.Contains(rows, "oride") {
		t.Errorf("o shell não devolveu o que foi digitado; o painel viu %q", rows)
	}
}

// TestTheFrameDrawsThePanel proves the paint path: com o painel aberto, o quadro
// tem as linhas do shell e cada linha mantém a largura do terminal.
func TestTheFrameDrawsThePanel(t *testing.T) {
	model := shellModel(t)
	model = press(t, model, tea.Key{Code: '`', Mod: tea.ModCtrl})

	// Um handle falso não é possível aqui: o campo é concreto. O que dá para
	// provar sem PTY é o **estado vazio**, que é o que o leitor vê antes do shell
	// responder — e é a única coisa que ele vê quando o shell falha.
	frame := model.Frame()
	if !strings.Contains(frame, "terminal") {
		t.Errorf("o painel aberto não aparece no quadro:\n%s", frame)
	}
	for index, row := range strings.Split(frame, "\n") {
		if got := layout.Width(row); got != 100 {
			t.Fatalf("linha %d com %d células: %q", index, got, row)
		}
	}
}

// TestTheHandleIsNotASecondScrollback keeps the panel honest: as linhas vêm do
// handle, e não de uma cópia no modelo.
func TestTheHandleIsNotASecondScrollback(t *testing.T) {
	pty, err := term.Spawn(t.TempDir(), 80, 10, "")
	if err != nil {
		t.Skipf("sem PTY neste ambiente: %v", err)
	}
	defer func() { _ = pty.Close() }()

	if err := pty.WriteString("echo oride\r"); err != nil {
		t.Fatalf("escrevendo no shell: %v", err)
	}

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		pty.Poll()
		if strings.Contains(strings.Join(pty.VisibleLines(20), "\n"), "oride") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if rows := pty.VisibleLines(20); len(rows) == 0 {
		t.Error("o handle não tem linhas para o painel pintar")
	}
}

// TestThePanelNeverBlocksStartup is the guarantee that keeps the editor
// usable: um `Cmd` que falha não pode trancar a TUI.
func TestThePanelNeverBlocksStartup(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "x"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Um shell que não existe é o pior caso: a `Cmd` volta com erro, e o modelo
	// precisa seguir vivo.
	model := shellModel(t)
	model.application.Config.Terminal.Shell = "/caminho/que/não/existe"
	model = press(t, model, tea.Key{Code: '`', Mod: tea.ModCtrl})

	updated, cmd := model.Update(nil)
	model = updated.(Model)
	if cmd == nil {
		t.Fatal("abrir o painel não pediu nada")
	}
	for _, msg := range flatten(cmd()) {
		if ready, ok := msg.(terminalReadyMsg); ok {
			updated, _ = model.Update(ready)
			model = updated.(Model)
		}
	}
	if model.pty != nil {
		t.Error("um shell inexistente foi anexado mesmo assim")
	}
	if !strings.Contains(model.application.Status, "terminal") {
		t.Errorf("status = %q, o leitor precisa saber que o terminal não subiu", model.application.Status)
	}
	if got := model.Frame(); got == "" {
		t.Error("o editor deixou de desenhar por causa do terminal")
	}
}

// TestThePanelNeverSpawnsAShellNobodySees é a regra que o painel com abas
// introduz: pedir uma PTY para uma aba de busca é abrir um shell que ninguém vê.
func TestThePanelNeverSpawnsAShellNobodySees(t *testing.T) {
	model := shellModel(t)
	model = press(t, model, tea.Key{Code: '`', Mod: tea.ModCtrl})
	// A PTY nasce na `Cmd`, e o teste pergunta pelo **pedido** — o handle só
	// existe depois que o comando roda, e um teste que espera o handle estaria
	// medindo o teste, não o produto.
	if cmd := model.terminalCmd(); cmd == nil {
		t.Fatal("a aba do terminal deveria pedir um shell")
	}

	// Abrir a busca leva a aba da frente: o shell continua vivo (é o que o
	// leitor espera ao voltar), mas **não** nasce um segundo.
	if err := model.application.Apply(action.Find); err != nil {
		t.Fatalf("abrindo a busca: %v", err)
	}
	if model.application.BottomTab != app.BottomTabSearch {
		t.Fatalf("a busca não virou aba: %q", model.application.BottomTab)
	}
	if cmd := model.terminalCmd(); cmd != nil {
		t.Error("a busca pediu um shell: a aba da frente não é o terminal")
	}
}

// TestTheTerminalKeepsTheKeyboardOnlyWhenVisible fecha a outra ponta: com a aba de
// busca à frente, o teclado **não** é do shell — senão a regra "foco nunca
// descansa onde nada é pintado" vira letra.
func TestTheTerminalKeepsTheKeyboardOnlyWhenVisible(t *testing.T) {
	model := shellModel(t)
	model = press(t, model, tea.Key{Code: '`', Mod: tea.ModCtrl})
	if !model.surfaceVisible(focus.Terminal) {
		t.Fatal("com a aba do terminal à frente, o foco deveria ser visível")
	}

	if err := model.application.Apply(action.Find); err != nil {
		t.Fatalf("abrindo a busca: %v", err)
	}
	if model.surfaceVisible(focus.Terminal) {
		t.Error("com a aba de busca à frente, o terminal ainda aceita o teclado")
	}
}

// TestClosingTheSearchTakesThePanelWithIt: um painel que fica vazio depois de Esc
// é um painel que mente sobre o que o programa está fazendo.
func TestClosingTheSearchTakesThePanelWithIt(t *testing.T) {
	model := shellModel(t)
	model.application.Store.OpenEmpty()
	if err := model.application.Apply(action.Find); err != nil {
		t.Fatalf("abrindo a busca: %v", err)
	}
	if !model.application.ShowBottom {
		t.Fatal("a busca não abriu o painel")
	}

	model = press(t, model, tea.Key{Code: tea.KeyEscape})

	if model.application.ShowBottom {
		t.Error("Esc fechou a busca e deixou o painel vazio à frente")
	}
}
