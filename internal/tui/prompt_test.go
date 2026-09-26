package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ori-team/oride/internal/action"
)

// O prompt de criação mora na sobreposição e captura tudo: estes testes abrem,
// digitam vendo o nome, criam, e cancelam — com um workspace de verdade, porque
// o que está em jogo é arquivo no disco.

// TestTreeNewFileTypesAndCreates é o caminho que o leitor faz: abre, digita o
// nome **vendo** o que digita, `Enter`, e o arquivo nasce aberto no editor.
func TestTreeNewFileTypesAndCreates(t *testing.T) {
	model := wired(t)
	if err := model.application.Apply(action.TreeNewFile); err != nil {
		t.Fatalf("abrindo o prompt: %v", err)
	}
	model.syncOverlay()
	model = typeText(t, model, "novo.txt")

	rows := model.promptItems()
	if len(rows) != 1 || rows[0].Label != "novo.txt" {
		t.Fatalf("o prompt mostra %v, e o leitor precisa ver o que digita", rows)
	}
	model = press(t, model, tea.Key{Code: tea.KeyEnter})

	if model.application.Overlay != "none" {
		t.Errorf("o prompt continuou aberto: %q", model.application.Overlay)
	}
	target := filepath.Join(model.application.Workspace, "novo.txt")
	if _, err := os.Stat(target); err != nil {
		t.Errorf("o arquivo não nasceu: %v", err)
	}
	document, err := model.application.Store.Active()
	if err != nil {
		t.Fatal(err)
	}
	path, ok := document.Path()
	if !ok || path != target {
		t.Errorf("criar e não abrir é criar em silêncio: %q", path)
	}
}

// TestPromptBackspaceErases prova que o apagar chega ao buffer: digitar errado
// e não poder corrigir é um prompt que só aceita nome certo de primeira.
func TestPromptBackspaceErases(t *testing.T) {
	model := wired(t)
	if err := model.application.Apply(action.TreeNewFile); err != nil {
		t.Fatal(err)
	}
	model.syncOverlay()
	model = typeText(t, model, "novx")
	model = press(t, model, tea.Key{Code: tea.KeyBackspace})
	model = typeText(t, model, "o.txt")

	if got := model.application.Prompt.Buffer; got != "novo.txt" {
		t.Errorf("buffer = %q, o apagar não chegou", got)
	}
}

// TestPromptEscapeCancels sem tocar no disco: cancelar não cria, não abre e não
// troca.
func TestPromptEscapeCancels(t *testing.T) {
	model := wired(t)
	paths := model.application.Store.OpenPaths()
	if err := model.application.Apply(action.TreeNewDir); err != nil {
		t.Fatalf("abrindo o prompt: %v", err)
	}
	model.syncOverlay()
	model = typeText(t, model, "nova")
	model = press(t, model, tea.Key{Code: tea.KeyEscape})

	if model.application.Overlay != "none" {
		t.Errorf("Esc não fechou o prompt: %q", model.application.Overlay)
	}
	if _, err := os.Stat(filepath.Join(model.application.Workspace, "nova")); !os.IsNotExist(err) {
		t.Error("Esc criou a pasta")
	}
	if got := model.application.Store.OpenPaths(); !sameStrings(got, paths) {
		t.Errorf("Esc abriu documento: %v → %v", paths, got)
	}
}

// TestTreeNewDirSaysCreated é o outro tipo pelo teclado: a pasta nasce e o
// status diz, porque nada mais na tela diz.
func TestTreeNewDirSaysCreated(t *testing.T) {
	model := wired(t)
	if err := model.application.Apply(action.TreeNewDir); err != nil {
		t.Fatal(err)
	}
	model.syncOverlay()

	rows := model.promptItems()
	if len(rows) != 1 || !strings.Contains(rows[0].Label, "pasta") {
		t.Errorf("o prompt de pasta mostra %v, e o leitor precisa saber o que vai nascer", rows)
	}
	model = typeText(t, model, "nova")
	model = press(t, model, tea.Key{Code: tea.KeyEnter})

	info, err := os.Stat(filepath.Join(model.application.Workspace, "nova"))
	if err != nil || !info.IsDir() {
		t.Fatalf("a pasta não nasceu: %v", err)
	}
	if model.application.Status != "pasta criada" {
		t.Errorf("status = %q", model.application.Status)
	}
}
