package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/editor"
	"github.com/ori-team/oride/internal/keymap"
)

// As teclas de painel e foco são as que o leitor aperta sem pensar, e nenhuma
// delas fazia nada. O que segue prova que elas **mudam estado** — porque um
// handler que devolve `nil` sem efeito satisfaz a regra de aptidão e engana o
// leitor, que é o defeito que a auditoria encontrou.

// withFile writes a workspace with one file and opens it, returning the model and
// the file's path.
func withFile(t *testing.T, name, text string) (*App, string) {
	t.Helper()

	root := t.TempDir()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("escrevendo %s: %v", path, err)
	}

	keys, err := keymap.FromBindings(config.Default().Keys)
	if err != nil {
		t.Fatalf("keymap: %v", err)
	}
	application := New(editor.NewStore(), config.Default(), keys)
	application.Workspace = root
	if _, err := application.Store.OpenPath(path); err != nil {
		t.Fatalf("abrindo: %v", err)
	}
	return application, path
}

// TestFocusToggleWalksBetweenTreeAndEditor fecha o caminho que o atalho promete:
// a árvore aparece, o foco vai para ela, e a mesma tecla volta.
func TestFocusToggleWalksBetweenTreeAndEditor(t *testing.T) {
	a, _ := withFile(t, "a.txt", "x\n")

	if a.Focus != FocusEditor {
		t.Fatalf("a sessão começou com foco em %q", a.Focus)
	}
	if err := a.Apply(action.FocusToggleTreeEditor); err != nil {
		t.Fatalf("foco: %v", err)
	}
	if a.Focus != FocusTree {
		t.Errorf("foco = %q, esperava a árvore", a.Focus)
	}
	if !a.ShowTree || a.Tree == nil {
		t.Error("o atalho mandou o foco para uma árvore que não apareceu")
	}
	if !strings.Contains(a.Status, "árvore") {
		t.Errorf("status = %q, o leitor precisa saber onde o foco foi", a.Status)
	}

	if err := a.Apply(action.FocusToggleTreeEditor); err != nil {
		t.Fatalf("foco: %v", err)
	}
	if a.Focus != FocusEditor {
		t.Errorf("a segunda vez o foco foi para %q, esperava o editor", a.Focus)
	}
}

// TestFocusToggleShowsAHiddenTree é o caso em que o alvo do atalho está escondido:
// um atalho para uma superfície invisível é um atalho para nada.
func TestFocusToggleShowsAHiddenTree(t *testing.T) {
	a, _ := withFile(t, "a.txt", "x\n")
	a.ShowTree = false
	a.Tree = nil

	if err := a.Apply(action.FocusToggleTreeEditor); err != nil {
		t.Fatalf("foco: %v", err)
	}
	if !a.ShowTree || a.Tree == nil {
		t.Error("a árvore não apareceu para receber o foco")
	}
}

// TestTreeRefreshReadsTheDiskAgain prova que a tecla recarrega, e não que muda um
// texto de status: um arquivo novo na árvore tem de aparecer.
func TestTreeRefreshReadsTheDiskAgain(t *testing.T) {
	a, path := withFile(t, "a.txt", "x\n")
	if err := a.Apply(action.FocusToggleTreeEditor); err != nil {
		t.Fatalf("foco: %v", err)
	}

	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "b.txt"), []byte("y\n"), 0o644); err != nil {
		t.Fatalf("escrevendo: %v", err)
	}
	before := len(a.Tree.FlatRows())
	if err := a.Apply(action.TreeRefresh); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	after := a.Tree.FlatRows()
	if len(after) <= before {
		t.Errorf("a árvore tinha %d linhas e ficou com %d: o disco não foi lido", before, len(after))
	}
	found := false
	for _, row := range after {
		if row.Name == "b.txt" {
			found = true
		}
	}
	if !found {
		t.Error("o arquivo criado depois não apareceu na árvore")
	}
}

// TestTreeRefreshWithNoTreeIsNotAnError: não haver árvore não é falha, e a tecla
// recarrega o que existe.
func TestTreeRefreshWithNoTreeIsNotAnError(t *testing.T) {
	a := buildWired(t, "x")

	if err := a.Apply(action.TreeRefresh); err != nil {
		t.Errorf("refresh sem árvore foi erro: %v", err)
	}
}

// TestToggleMouseFlipsTheConfigurationAndSaysWhy é a tecla que muda a
// capacidade do terminal, e a mensagem precisa explicar a desgraça: quem acabou de
// desligar a seleção de texto do terminal não liga as duas coisas sozinho.
func TestToggleMouseFlipsTheConfigurationAndSaysWhy(t *testing.T) {
	a := buildWired(t, "x")

	if a.Config.Mouse {
		t.Fatal("o mouse começa ligado, e o padrão documentado é desligado")
	}
	if err := a.Apply(action.ToggleMouse); err != nil {
		t.Fatalf("toggle mouse: %v", err)
	}
	if !a.Config.Mouse {
		t.Error("a tecla não ligou o mouse")
	}
	if !strings.Contains(a.Status, "ON") {
		t.Errorf("status = %q", a.Status)
	}

	if err := a.Apply(action.ToggleMouse); err != nil {
		t.Fatalf("toggle mouse: %v", err)
	}
	if a.Config.Mouse {
		t.Error("a segunda vez a tecla não desligou o mouse")
	}
	if !strings.Contains(a.Status, "padrão") {
		t.Errorf("status = %q, desligar precisa dizer por que o padrão é desligado", a.Status)
	}
}

// TestReloadReadsTheFileAndKeepsTheCaret is a tecla que o leitor usa depois de
// editar fora do editor — e recarregar descartando o cursor seria devolver o
// leitor ao começo do arquivo.
func TestReloadReadsTheFileAndKeepsTheCaret(t *testing.T) {
	a, path := withFile(t, "a.txt", "primeira\nsegunda\n")
	document, _ := a.Store.Active()
	document.JumpToByte(9) // começo da segunda linha

	if err := os.WriteFile(path, []byte("primeira\nSEGUNDA\n"), 0o644); err != nil {
		t.Fatalf("escrevendo: %v", err)
	}
	if err := a.Apply(action.ReloadFile); err != nil {
		t.Fatalf("reload: %v", err)
	}

	if got := document.Buffer().String(); !strings.Contains(got, "SEGUNDA") {
		t.Errorf("o arquivo em disco não chegou ao documento: %q", got)
	}
	if document.IsDirty() {
		t.Error("o documento recarregado continua sujo")
	}
	caret, _ := document.Caret()
	if caret.Line != 1 {
		t.Errorf("o caret foi para a linha %d, esperava a 1", caret.Line)
	}
	if !strings.Contains(a.Status, "recarregado") {
		t.Errorf("status = %q", a.Status)
	}
}

// TestReloadClampsACaretPastTheNewEnd é o defeito de um arquivo que encolheu: um
// offset além do fim escreveria no lugar errado no próximo `insert`.
func TestReloadClampsACaretPastTheNewEnd(t *testing.T) {
	a, path := withFile(t, "a.txt", "um\ndois\ntrês\n")
	document, _ := a.Store.Active()
	document.JumpToByte(12)

	if err := os.WriteFile(path, []byte("um\n"), 0o644); err != nil {
		t.Fatalf("escrevendo: %v", err)
	}
	if err := a.Apply(action.ReloadFile); err != nil {
		t.Fatalf("reload: %v", err)
	}

	// O invariante não é "qual linha" — o fim do arquivo é uma linha válida — e
	// sim que o caret **resolve** e que o próximo texto cai no fim, e não no
	// lugar de onde o caret veio.
	if _, err := document.Caret(); err != nil {
		t.Fatalf("após recarregar um arquivo menor, o caret não resolve: %v", err)
	}
	if err := document.InsertText("X"); err != nil {
		t.Fatalf("inserindo: %v", err)
	}
	if got := document.Buffer().String(); got != "um\nX" {
		t.Errorf("o texto caiu em %q; um offset além do fim escreveria no lugar errado", got)
	}
}

// TestReloadOfAFileThatVanishedSaysSo: recarregar um arquivo que sumiu é
// situação normal, e o leitor precisa do motivo na barra — não um erro que
// interrompe a tecla.
func TestReloadOfAFileThatVanishedSaysSo(t *testing.T) {
	a, path := withFile(t, "a.txt", "conteúdo\n")
	if err := os.Remove(path); err != nil {
		t.Fatalf("removendo: %v", err)
	}

	if err := a.Apply(action.ReloadFile); err != nil {
		t.Errorf("reload de arquivo sumido foi erro: %v", err)
	}
	if !strings.Contains(a.Status, "reload:") {
		t.Errorf("status = %q, o leitor precisa do motivo", a.Status)
	}
}

// TestReloadOfADocumentWithNoPathIsRefused: um buffer novo não tem de onde ler.
func TestReloadOfADocumentWithNoPathIsRefused(t *testing.T) {
	a := buildWired(t, "sem caminho")

	if err := a.Apply(action.ReloadFile); err != nil {
		t.Errorf("reload sem caminho foi erro: %v", err)
	}
	if !strings.Contains(a.Status, "reload:") {
		t.Errorf("status = %q", a.Status)
	}
}

// TestThePanelTabIsReachableByKeyboard é a correção de uma omissão minha: as três
// abas existiam, e **não havia tecla para trocar entre elas**. Um painel cujas abas
// não são alcançáveis é a mesma classe de UI que não funciona que o resto deste
// trabalho pegou — a tecla que o leitor vê e a tecla que ele tem não batem.
func TestThePanelTabIsReachableByKeyboard(t *testing.T) {
	a := buildWired(t, "x")
	keys, err := keymap.FromBindings(config.Default().Keys)
	if err != nil {
		t.Fatalf("keymap: %v", err)
	}

	// O oráculo publica `focus_next_pane` em `ctrl+alt+left` e `ctrl+alt+right`.
	// Uma tabela nova seria divergência de paridade, então a tecla é a que existe.
	for _, chord := range []string{"ctrl+alt+right", "ctrl+alt+left"} {
		parsed, parseErr := keymap.Parse(chord)
		if parseErr != nil {
			t.Fatalf("Parse(%q): %v", chord, parseErr)
		}
		if bound, found := keys.Resolve(parsed); !found || bound != action.FocusNextPane {
			t.Fatalf("%q aponta para %q, esperava focus_next_pane", chord, bound)
		}
	}

	// Fechado: o comando abre o painel na primeira aba.
	if err := a.ApplyKey("ctrl+alt+right"); err != nil {
		t.Fatalf("abrindo o painel: %v", err)
	}
	if !a.ShowBottom || a.BottomTab != BottomTabTerminal {
		t.Fatalf("o painel não abriu: %v / %q", a.ShowBottom, a.BottomTab)
	}

	// Aberto: o comando avança, e o ciclo fecha.
	for _, want := range []string{BottomTabProblems, BottomTabSearch, BottomTabTerminal} {
		if err := a.ApplyKey("ctrl+alt+right"); err != nil {
			t.Fatalf("trocando de aba: %v", err)
		}
		if a.BottomTab != want {
			t.Fatalf("aba = %q, esperava %q", a.BottomTab, want)
		}
	}
}

// TestEveryTabOfThePanelHasAWayToReachIt é a forma mais honesta de dizer "a aba
// existe" — cada uma tem que aparecer na sequência de comando.
func TestEveryTabOfThePanelHasAWayToReachIt(t *testing.T) {
	a := buildWired(t, "x")
	if err := a.ApplyKey("ctrl+alt+right"); err != nil {
		t.Fatalf("abrindo: %v", err)
	}

	seen := map[string]bool{a.BottomTab: true}
	for range 3 {
		if err := a.ApplyKey("ctrl+alt+right"); err != nil {
			t.Fatalf("trocando: %v", err)
		}
		seen[a.BottomTab] = true
	}
	for _, tab := range []string{BottomTabTerminal, BottomTabProblems, BottomTabSearch} {
		if !seen[tab] {
			t.Errorf("a aba %q não é alcançável pelo teclado", tab)
		}
	}
}
