package main

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ori-team/oride/internal/app"
	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/editor"
	"github.com/ori-team/oride/internal/keymap"
	"github.com/ori-team/oride/internal/tui"
	"github.com/ori-team/oride/internal/tui/theme"
)

// overlayNone is the model's own value for "no overlay open". The composition root
// does not export its vocabulary, and duplicá-lo aqui seria um segundo lugar para
// as duas constantes divergirem — que é o defeito que o ledger já registrou uma
// vez (o overlay da TUI e o do modelo sendo confundidos).
//
// O valor vem do dump do oráculo, que é a fonte: sem overlay aberto, o dump
// escreve isto.
const overlayNone = "none"

// O teste de fumaça do S5.
//
// Nove checks verdes conviveram com uma TUI em que não se fechava o programa, não
// se navegava a árvore e os menus não faziam nada. Frame, teste e paridade não
// perguntam se uma tecla faz alguma coisa — e cada tecla **isolada** já tinha
// teste. O que não tinha era a sessão: a sequência inteira, na ordem em que um
// usuário a executa. É aí que "cada parte funciona" e "o programa funciona"
// deixam de ser a mesma afirmação.
//
// Este teste dirige o modelo **como o binário o constrói** — a mesma configuração
// mesclada, o mesmo keymap, a mesma árvore, a mesma restauração de sessão — e
// afirma sobre **estado**, nunca sobre frame. Um frame muda com o tema e com a
// largura; um caminho de usuário não.

// smoke é a sessão dirigida por um usuário, com o estado que ela produz.
//
// O nome não é "session" porque o pacote internal/session já é importado aqui, e
// dois nomes para coisas diferentes numa linha só é o tipo de colisão que um
// arquivo de teste longo esconde.
type smoke struct {
	t      *testing.T
	model  tui.Model
	app    *app.App
	width  int
	height int
}

// start builds the editor the way openEditor does, on a real workspace.
func start(t *testing.T, root string) *smoke {
	t.Helper()

	configuration, err := config.LoadMerged(root)
	if err != nil {
		t.Fatalf("configuração: %v", err)
	}
	keys, err := keymap.FromBindings(configuration.Keys)
	if err != nil {
		t.Fatalf("keymap: %v", err)
	}

	application := app.New(editor.NewStore(), configuration, keys)
	application.Workspace = root
	application.Tree = openTree(root, configuration.Tree.ShowHidden)
	if err := restore(application, nil, root); err != nil {
		t.Fatalf("restauração: %v", err)
	}

	model := tui.New(application, keys).WithProfile(theme.TrueColor).Resize(100, 30)
	return &smoke{t: t, model: model, app: application, width: 100, height: 30}
}

// press sends one keystroke, the way the runtime would.
func (s *smoke) press(key tea.Key) *smoke {
	s.t.Helper()

	updated, _ := s.model.Update(tea.KeyPressMsg(key))
	next, ok := updated.(tui.Model)
	if !ok {
		s.t.Fatalf("Update devolveu %T", updated)
	}
	s.model = next
	return s
}

// typeText sends text one character at a time.
//
// Um evento por caractere, como o terminal entrega: um teste que mandasse a
// string inteira estaría exercitando um caminho que o runtime não usa.
func (s *smoke) typeText(text string) *smoke {
	for _, r := range text {
		s.press(tea.Key{Code: r, Text: string(r)})
	}
	return s
}

// tab moves focus forward until the named surface holds it.
func (s *smoke) focus(appFocus app.Focus, times int) *smoke {
	s.t.Helper()

	for range times {
		if s.app.Focus == appFocus {
			return s
		}
		s.press(tea.Key{Code: tea.KeyTab})
	}
	if s.app.Focus != appFocus {
		s.t.Fatalf("o foco não chegou a %v em %d tabulações", appFocus, times)
	}
	return s
}

// openInTree walks the tree with the keys a user presses, and opens the named
// file: down to move, right to expand a directory, enter on the file.
func (s *smoke) openInTree(name string) *smoke {
	s.t.Helper()

	// Acima do número de linhas visíveis, qualquer coisa está errada; o limite
	// existe para o teste falhar em vez de ficar andando.
	for round := range s.app.Tree.CountVisibleRows() * 4 {
		row, ok := s.app.Tree.SelectedRow()
		if !ok {
			s.t.Fatalf("a seleção saiu das linhas visíveis depois de %d teclas", round)
		}
		switch {
		case !row.IsDir && row.Name == name:
			s.press(tea.Key{Code: tea.KeyEnter})
			return s
		case row.IsDir && !row.Expanded:
			s.press(tea.Key{Code: tea.KeyRight})
		default:
			s.press(tea.Key{Code: tea.KeyDown})
		}
	}
	s.t.Fatalf("a árvore não mostrou %q em %d passadas", name, s.app.Tree.CountVisibleRows()*4)
	return s
}

// buffer is the active document's text.
func (s *smoke) buffer() string {
	s.t.Helper()

	document, err := s.app.Store.Active()
	if err != nil {
		s.t.Fatalf("documento ativo: %v", err)
	}
	return document.Buffer().String()
}

// path is the active document's path.
func (s *smoke) path() string {
	s.t.Helper()

	document, err := s.app.Store.Active()
	if err != nil {
		s.t.Fatalf("documento ativo: %v", err)
	}
	path, ok := document.Path()
	if !ok {
		return ""
	}
	return path
}

// treeWorkspace is a workspace with a nested file and a loose one, so the tree has
// a directory to expand and two files to open.
//
// It reuses the workspace helper that session_test.go already builds rather than
// writing a second one: two helpers that make the same temporary tree are the
// duplication S3 exists to remove, and a test file is not an exception.
func treeWorkspace(t *testing.T) string {
	t.Helper()

	return workspace(t, map[string]string{
		"raiz.txt":      "conteúdo da raiz\n",
		"docs/notas.md": "# notas\nalfa beta\n",
		"docs/guia.txt": "guia\n",
	})
}

// TestAUserSessionEndsWithTheEditorClosed é o teste que o gate do M3 não tinha.
//
// Abre, anda na árvore, abre um arquivo, digita, busca, fecha. Cada passo afirma
// o estado que o passo produz; o passo final afirma que o programa pediu para
// sair, que é a única coisa que importa quando o reader aperta Ctrl+Q.
func TestAUserSessionEndsWithTheEditorClosed(t *testing.T) {
	root := treeWorkspace(t)
	s := start(t, root)

	// 1. Abrir: um documento ativo e uma árvore, sem ninguém apertar nada.
	if s.app.Tree == nil {
		t.Fatal("a sessão começou sem árvore, e a árvore é o que se navega")
	}
	if len(s.app.Store.OpenPaths()) == 0 && s.buffer() != "" {
		t.Fatalf("a sessão começou com um documento que não foi aberto: %q", s.buffer())
	}

	// 2. Olhar a árvore: ela está visível e tem linhas.
	s.app.ShowTree = true
	if rows := s.app.Tree.FlatRows(); len(rows) == 0 {
		t.Fatal("a árvore não tem linhas para navegar")
	}

	// 3. Andar até ela com Tab, e percorrer a árvore até o arquivo.
	s.focus(app.FocusTree, 8)
	before := s.app.Tree.SelectedIndex()
	s.press(tea.Key{Code: tea.KeyDown})
	if got := s.app.Tree.SelectedIndex(); got == before {
		t.Fatalf("a seta não andou na árvore: continua em %d", got)
	}

	// 4. Expandir diretórios e abrir o arquivo, como um usuário faz: desce até a
	// linha, expande o que é pasta, e entra no que é arquivo.
	//
	// A navegação é por nome e não por índice porque a ordem das linhas é da
	// árvore, não do teste: um índice fixo aqui seria um teste que passa no
	// workspace dele e falha no do reader.
	s.openInTree("notas.md")
	if got := filepath.Base(s.path()); got != "notas.md" {
		t.Fatalf("a árvore abriu %q, esperado notas.md", got)
	}

	// 5. Voltar ao editor e digitar: o texto vai para o documento, não para o
	// painel.
	s.focus(app.FocusEditor, 8)
	s.press(tea.Key{Code: tea.KeyHome})
	s.typeText("olá")
	if got := s.buffer(); !strings.HasPrefix(got, "olá") {
		t.Fatalf("o que foi digitado não chegou ao documento: %q", got)
	}
	if !strings.Contains(s.buffer(), "alfa beta") {
		t.Errorf("digitar destruiu o conteúdo do arquivo: %q", s.buffer())
	}

	// 6. Buscar: Ctrl+F abre a barra, digitar filtra ao vivo, Esc fecha.
	s.press(tea.Key{Code: 'f', Mod: tea.ModCtrl})
	if s.app.Overlay == overlayNone {
		t.Fatal("Ctrl+F não abriu a busca")
	}
	s.typeText("alfa")
	if len(s.app.Find.Matches) == 0 {
		t.Fatalf("a busca não encontrou \"alfa\" no documento: %q", s.buffer())
	}
	s.press(tea.Key{Code: tea.KeyEscape})
	if s.app.Overlay != overlayNone {
		t.Error("Esc não fechou a busca")
	}

	// 7. Fechar: é o passo que o leitor nota primeiro, e o que o gate nunca
	// perguntou.
	if s.app.Quit {
		t.Fatal("a sessão já estava fechando antes do Ctrl+Q")
	}
	// A afirmação é o flag do modelo, e não o Cmd devolvido: desde o realce, todo
	// Update devolve um Cmd (o do tokenize), e um teste que affirmasse "cmd != nil"
	// estaria medindo a existência do realce e não a da saída.
	s.press(tea.Key{Code: 'q', Mod: tea.ModCtrl})
	if !s.app.Quit {
		t.Error("Ctrl+Q não ligou o flag de saída: o programa não fecha")
	}
}

// TestTheSmokeTestCanFail é a contraprova.
//
// Um teste de fumaça que nunca falhou não mediu nada. Aqui a mesma sessão roda
// **sem o binding de saída**, e o programa não fecha: se a contraprova passasse
// com o binding de volta, o teste estaria medindo o próprio teste.
func TestTheSmokeTestCanFail(t *testing.T) {
	root := treeWorkspace(t)

	configuration, err := config.LoadMerged(root)
	if err != nil {
		t.Fatalf("configuração: %v", err)
	}
	delete(configuration.Keys, "ctrl+q")
	keys, err := keymap.FromBindings(configuration.Keys)
	if err != nil {
		t.Fatalf("keymap sem ctrl+q: %v", err)
	}

	application := app.New(editor.NewStore(), configuration, keys)
	application.Store.OpenEmpty()

	tui.New(application, keys).Resize(100, 30).
		Update(tea.KeyPressMsg(tea.Key{Code: 'q', Mod: tea.ModCtrl}))

	if application.Quit {
		t.Error("a contraprova falhou: sem binding, o modelo ligou o flag de saída")
	}

	// E a tecla não fez nada, que é o que um acorde sem binding deve fazer: um
	// acorde não é um caractere, e "ctrl+q" no buffer seria pior que ignorá-lo.
	// (Uma tecla **sem** modificador vira texto — é o que o terceiro teste mede.)
	document, err := application.Store.Active()
	if err != nil {
		t.Fatalf("documento ativo: %v", err)
	}
	if got := document.Buffer().String(); got != "" {
		t.Errorf("acorde sem binding não pode virar texto; o buffer é %q", got)
	}
}

// TestAUserPathThatEndsNowhereIsNotAUserPath mede a outra ponta: um caminho que
// digita em vez de sair não pode ser confundido com um caminho que fecha.
func TestAUserPathThatEndsNowhereIsNotAUserPath(t *testing.T) {
	application := app.New(editor.NewStore(), config.Default(), keymap.New())
	application.Store.OpenEmpty()

	tui.New(application, keymap.New()).Resize(100, 30).
		Update(tea.KeyPressMsg(tea.Key{Code: 'q'}))

	document, err := application.Store.Active()
	if err != nil {
		t.Fatalf("documento ativo: %v", err)
	}
	if got := document.Buffer().String(); got != "q" {
		t.Errorf("a tecla sem modificador devia ser texto, e o buffer é %q", got)
	}
}
