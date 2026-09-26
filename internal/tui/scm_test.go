package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ori-team/oride/internal/app"
	"github.com/ori-team/oride/internal/fs"
)

// O painel SCM é o último dos três painéis a ganhar conteúdo de verdade, e estes
// testes fecham o caminho: tecla, painel, stage e diff — tudo com um repositório
// de verdade, porque `porcelain` sem `.git` é um jeito elegante de não testar
// nada.

func scmRepo(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("sem git neste ambiente")
	}

	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		command := exec.Command("git", args...)
		command.Dir = root
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "t@t.t")
	run("config", "user.name", "t")
	run("config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "base.txt")
	run("commit", "-q", "-m", "base")
	if err := os.WriteFile(filepath.Join(root, "trabalho.txt"), []byte("trabalho\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// ctrlShiftG is the `Ctrl+Shift+G` that toggles the SCM panel, montado como o
// terminal montaria: código mais modificador, e não uma string que alguém
// interpreta.
func ctrlShiftG() tea.Key { return tea.Key{Code: 'G', Mod: tea.ModCtrl | tea.ModShift} }

// teaKeyEscape is `Esc` como o terminal manda.
func teaKeyEscape() tea.Key { return tea.Key{Code: tea.KeyEscape} }

// scmKey is a surface key of the panel, que são letras sem modificador: o painel
// tem o teclado.
func scmKey(t *testing.T, letter string) tea.Key {
	t.Helper()
	runes := []rune(letter)
	if len(runes) != 1 {
		t.Fatalf("a tecla do painel tem de ser uma letra: %q", letter)
	}
	return tea.Key{Code: runes[0], Text: letter}
}

// modelWorkspace is the workspace the model points at, e o teste o constrói —
// que é o que impede o helper de ler `os.Getwd` do processo.
func modelWorkspace(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// wiredScmTracked is a model over a git repository with a **tracked, modified**
// file: é o que `git diff` tem o que dizer, e é por isso que o teste do diff usa
// este e não o outro.
func wiredScmTracked(t *testing.T) Model {
	t.Helper()

	root := scmRepo(t)
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("base\nmais uma linha\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	model := sized(t, newModel(t, nil), 120, 30)
	model.application.Workspace = root
	model.application.GitEnabled = true
	model.application.Tree, _ = fs.Open(root, false)
	return model
}

// wiredScm is a model over a git repository, com a árvore aberta.
func wiredScm(t *testing.T) Model {
	t.Helper()

	model := sized(t, newModel(t, nil), 120, 30)
	model.application.Workspace = scmRepo(t)
	model.application.GitEnabled = true
	model.application.Tree = mustOpenTree(t)
	return model
}

// mustOpenTree is the workspace's tree, para o painel ter vizinhança.
func mustOpenTree(t *testing.T) *fs.Tree {
	t.Helper()
	tree, err := fs.Open(modelWorkspace(t), false)
	if err != nil {
		t.Fatalf("abrindo a árvore: %v", err)
	}
	return tree
}

// TestToggleScmOpensThePanelAndFocusesIt é o que `Ctrl+Shift+G` promete, com um
// repositório de verdade atrás.
func TestToggleScmOpensThePanelAndFocusesIt(t *testing.T) {
	model := wiredScm(t)
	if model.application.SCM {
		t.Fatal("o painel começou aberto")
	}

	model = press(t, model, ctrlShiftG())

	if !model.application.SCM {
		t.Fatal("Ctrl+Shift+G não abriu o painel")
	}
	if model.application.Focus != app.FocusSCM {
		t.Errorf("foco = %q, esperava o scm", model.application.Focus)
	}
	if !strings.Contains(model.application.Status, "SCM") {
		t.Errorf("status = %q, a barra precisa dizer o que a tecla fez", model.application.Status)
	}

	// Esc com foco no SCM devolve o **foco** ao editor e deixa o painel aberto:
	// é o que a referência faz, e é o que a barra de status promete ("Esc
	// editor"). Fechar é `Ctrl+Shift+G` de novo — o painel é mobília, e Esc é
	// para teclas transitórias.
	model = press(t, model, teaKeyEscape())
	if model.application.Focus != app.FocusEditor {
		t.Errorf("Esc não devolveu o foco ao editor (foco = %q)", model.application.Focus)
	}
	if !model.application.SCM {
		t.Error("Esc fechou o painel em vez de devolver o foco")
	}

	model = press(t, model, ctrlShiftG())
	if model.application.SCM {
		t.Error("a segunda vez não fechou o painel")
	}
	if model.application.Focus != app.FocusEditor {
		t.Errorf("fechar não devolveu o foco ao editor (foco = %q)", model.application.Focus)
	}
}

// TestTheScmKeysActOnTheSelectedLine: `s` adiciona ao índice, `u` tira, `r`
// relê, `d` mostra o diff e `Enter` abre o arquivo — tudo com um repositório de
// verdade, porque um teste de `stage` sem índice é uma afirmação sobre nada.
func TestTheScmKeysActOnTheSelectedLine(t *testing.T) {
	model := wiredScm(t)
	model = press(t, model, ctrlShiftG())

	entries := model.application.SCMEntries
	if len(entries) != 1 || entries[0].Path != "trabalho.txt" {
		t.Fatalf("o painel não viu o arquivo não rastreado: %+v", entries)
	}

	// `s` adiciona ao índice.
	model = press(t, model, scmKey(t, "s"))
	if badge := badgeOf(t, model, "trabalho.txt"); badge != "A" {
		t.Fatalf("depois de `s`, o badge é %q, e não A", badge)
	}

	// `d` mostra o diff.
	model = press(t, model, scmKey(t, "d"))
	if model.application.Overlay != app.OverlayDiff {
		t.Fatalf("depois de `d`, overlay = %q, e não o diff", model.application.Overlay)
	}
	if len(model.diffLines) == 0 {
		t.Error("o diff veio vazio para um arquivo com conteúdo")
	}
	model = press(t, model, teaKeyEscape())
	if model.application.Overlay != app.OverlayNone {
		t.Error("Esc não fechou o diff")
	}

	// `u` tira do índice, e o trabalho continua lá.
	model = press(t, model, scmKey(t, "u"))
	if badge := badgeOf(t, model, "trabalho.txt"); badge != "?" {
		t.Fatalf("depois de `u`, o badge é %q, e não ?", badge)
	}
}

// TestTheDiffOverlayIsReadable fecha o caminho do `d`: as linhas do diff são
// texto, e o quadro as mostra.
//
// O arquivo é **rastreado e modificado**: `git diff` de arquivo não rastreado é
// vazio, e um teste que esperasse diff de um arquivo que o git nunca viu estaria
// afirmando contra o próprio `git`.
func TestTheDiffOverlayIsReadable(t *testing.T) {
	model := wiredScmTracked(t)
	model = press(t, model, ctrlShiftG())
	model = press(t, model, scmKey(t, "d"))

	// A entrada selecionada é a primeira (`base.txt`, modificada): o diff fala
	// dela, e não da segunda da lista.
	joined := strings.Join(model.diffLines, "\n")
	if !strings.Contains(joined, "base.txt") {
		t.Errorf("o diff não fala do arquivo selecionado:\n%s", joined)
	}

	frame := model.Frame()
	if !strings.Contains(frame, "base.txt") {
		t.Errorf("o diff não aparece no quadro")
	}
}

// badgeOf is the badge the panel shows for a path, lido do estado do modelo —
// que é o que o dump compara, e não a tela.
func badgeOf(t *testing.T, model Model, path string) string {
	t.Helper()
	for _, entry := range model.application.SCMEntries {
		if entry.Path == path {
			return entry.Badge
		}
	}
	return ""
}
