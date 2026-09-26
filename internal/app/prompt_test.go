package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ori-team/oride/internal/action"
)

// O prompt de criação é a tecla que faz arquivo e pasta nascerem: `tree_new_file`
// e `tree_new_dir` eram dívida nomeada, anunciadas no keymap gerado. O que segue
// prova o caminho inteiro no modelo — abrir, digitar, apagar, criar — com um
// workspace de verdade, porque criar sem disco é um jeito elegante de não testar
// nada.

// promptWired is a model over an empty workspace on the disk.
func promptWired(t *testing.T) *App {
	t.Helper()

	a := buildWired(t, "")
	a.Workspace = t.TempDir()
	return a
}

// TestTreeNewFileOpensThePromptOnTheTree é o que a ação promete: a árvore
// aparece, o foco vai para ela, e o prompt espera um nome vazio.
func TestTreeNewFileOpensThePromptOnTheTree(t *testing.T) {
	a := promptWired(t)
	if err := a.Apply(action.TreeNewFile); err != nil {
		t.Fatalf("abrindo o prompt: %v", err)
	}
	if a.Overlay != OverlayPrompt {
		t.Errorf("overlay = %q, esperava o prompt", a.Overlay)
	}
	if a.Prompt.Kind != PromptNewFile {
		t.Errorf("kind = %q, esperava arquivo", a.Prompt.Kind)
	}
	if a.Prompt.Buffer != "" {
		t.Errorf("o prompt abriu com %q, e resto de criação passada é arquivo com nome errado", a.Prompt.Buffer)
	}
	if !a.ShowTree || a.Focus != FocusTree {
		t.Error("criar sem ver a árvore é criar sem saber onde")
	}
}

// TestPromptEditsAtTheEnd é a regra do oráculo: anexar e apagar no fim, por
// rune — e controle não entra, porque é nome que o disco recusa.
func TestPromptEditsAtTheEnd(t *testing.T) {
	a := promptWired(t)
	if err := a.Apply(action.TreeNewFile); err != nil {
		t.Fatal(err)
	}

	a.PromptAppend("áé")
	a.PromptAppend("\x07")
	if a.Prompt.Buffer != "áé" {
		t.Errorf("buffer = %q, controle entrou ou rune quebrou", a.Prompt.Buffer)
	}
	a.PromptBackspace()
	if a.Prompt.Buffer != "á" {
		t.Errorf("apagar comeu %q, e meio caractere não é nome", a.Prompt.Buffer)
	}
	a.PromptBackspace()
	a.PromptBackspace()
	a.PromptBackspace()
	if a.Prompt.Buffer != "" {
		t.Errorf("apagar no vazio devia parar no vazio: %q", a.Prompt.Buffer)
	}
}

// TestSubmitPromptCreatesAndOpens é o `Enter`: o arquivo nasce vazio, abre no
// editor, o foco vai com ele, e o prompt fecha — com erro ou sem, porque um
// prompt que fica aberto depois de criar mente sobre o que já aconteceu.
func TestSubmitPromptCreatesAndOpens(t *testing.T) {
	a := promptWired(t)
	if err := a.Apply(action.TreeNewFile); err != nil {
		t.Fatal(err)
	}
	a.PromptAppend("novo.txt")
	a.SubmitPrompt()

	if a.Overlay != OverlayNone {
		t.Errorf("o prompt continuou aberto: %q", a.Overlay)
	}
	if _, err := os.Stat(filepath.Join(a.Workspace, "novo.txt")); err != nil {
		t.Errorf("o arquivo não nasceu: %v", err)
	}
	document, err := a.Store.Active()
	if err != nil {
		t.Fatal(err)
	}
	path, ok := document.Path()
	if !ok || !strings.HasSuffix(path, "novo.txt") {
		t.Errorf("criar e não abrir é criar em silêncio: %+v", document)
	}
	if a.Focus != FocusEditor {
		t.Error("o foco ficou na árvore depois de abrir")
	}
}

// TestSubmitPromptDirSaysCreated é o outro tipo: a pasta nasce, e como nada mais
// muda na tela para dizê-lo, o status diz.
func TestSubmitPromptDirSaysCreated(t *testing.T) {
	a := promptWired(t)
	if err := a.Apply(action.TreeNewDir); err != nil {
		t.Fatal(err)
	}
	a.PromptAppend("nova")
	a.SubmitPrompt()

	info, err := os.Stat(filepath.Join(a.Workspace, "nova"))
	if err != nil || !info.IsDir() {
		t.Fatalf("a pasta não nasceu: %v", err)
	}
	if a.Status != "pasta criada" {
		t.Errorf("status = %q, e nada mais na tela diz que criou", a.Status)
	}
}

// TestSubmitPromptEmptyFailsWithAReason é o `Enter` no vazio: fecha, não cria, e
// diz por quê — em vez de um arquivo sem nome ou de um silêncio.
func TestSubmitPromptEmptyFailsWithAReason(t *testing.T) {
	a := promptWired(t)
	before, err := os.ReadDir(a.Workspace)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Apply(action.TreeNewFile); err != nil {
		t.Fatal(err)
	}
	a.SubmitPrompt()

	if a.Overlay != OverlayNone {
		t.Errorf("o prompt continuou aberto: %q", a.Overlay)
	}
	after, err := os.ReadDir(a.Workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Error("o `Enter` no vazio criou alguma coisa")
	}
	if !strings.HasPrefix(a.Status, "criar: ") {
		t.Errorf("status = %q, e falhar sem motivo é a tecla mentindo", a.Status)
	}
}
