package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ori-team/oride/internal/action"
)

// O seletor de buffers é a resposta para "qual era aquela aba?": as abas
// abertas, com a suja marcada, filtradas ao digitar. Era uma tecla anunciada no
// keymap gerado sem tratamento — `Ctrl+Shift+O` — e estes testes abrem, marcam,
// ativam e cancelam, com documentos de verdade, porque um seletor sem sessão é
// um jeito elegante de não testar nada.

// bufferSession is a model with three open files, the middle one dirty.
func bufferSession(t *testing.T) Model {
	t.Helper()

	model := wired(t)
	root := model.application.Workspace
	names := []string{"alfa.txt", "beta.txt", "gamma.txt"}
	for _, name := range names {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte("conteúdo de "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := model.application.Store.OpenPath(path); err != nil {
			t.Fatalf("abrindo %s: %v", name, err)
		}
	}
	// A do meio fica suja: digitar no editor é o caminho que o leitor usa, e é
	// o que marca.
	model = typeText(t, model, "x")
	return model
}

// TestBufferPickerListsOpenTabsWithTheDirtyOneMarked é o que o seletor promete:
// as abas abertas, em ordem, com a suja marcada em texto.
func TestBufferPickerListsOpenTabsWithTheDirtyOneMarked(t *testing.T) {
	model := bufferSession(t)
	if err := model.application.Apply(action.BufferPicker); err != nil {
		t.Fatalf("abrindo o seletor: %v", err)
	}
	if model.application.Overlay != "BufferPicker" {
		t.Fatalf("overlay = %q, esperava o seletor de buffers", model.application.Overlay)
	}

	rows := model.bufferItems()
	if len(rows) != 4 {
		t.Fatalf("o seletor lista %d linhas, e a sessão tem 4 abas", len(rows))
	}
	marked := 0
	for _, row := range rows {
		if strings.HasPrefix(row.Label, "● ") {
			marked++
		}
	}
	if marked != 1 {
		t.Errorf("o seletor marca %d abas como sujas, e só uma está suja", marked)
	}
	for _, name := range []string{"alfa.txt", "beta.txt", "gamma.txt"} {
		found := false
		for _, row := range rows {
			if strings.HasSuffix(row.Label, name) {
				found = true
			}
		}
		if !found {
			t.Errorf("o seletor não lista %s", name)
		}
	}
}

// TestBufferPickerEnterActivates é o que a tecla promete: a aba ativa troca, o
// teclado vai para o editor, e o seletor fecha.
func TestBufferPickerEnterActivates(t *testing.T) {
	model := bufferSession(t)
	if err := model.application.Apply(action.BufferPicker); err != nil {
		t.Fatalf("abrindo o seletor: %v", err)
	}
	// O `Apply` é no modelo; a projeção na tela acontece no `Update` seguinte, e
	// aqui ela é explícita porque o teste dirigiu o modelo direto.
	model.syncOverlay()

	before, err := model.application.Store.Active()
	if err != nil {
		t.Fatal(err)
	}
	beforePath, _ := before.Path()

	// A primeira linha é a aba vazia inicial; a segunda é o alfa.
	model = press(t, model, tea.Key{Code: tea.KeyDown})
	model = press(t, model, tea.Key{Code: tea.KeyEnter})

	if model.application.Overlay != "none" {
		t.Errorf("Enter não fechou o seletor: %q", model.application.Overlay)
	}
	after, err := model.application.Store.Active()
	if err != nil {
		t.Fatal(err)
	}
	afterPath, ok := after.Path()
	if !ok || !strings.HasSuffix(afterPath, "alfa.txt") {
		t.Errorf("Enter não ativou o alfa: %q (era %q)", afterPath, beforePath)
	}
	_ = beforePath
}

// TestEscapeClosesTheBufferPicker sem tocar em nada: cancelar não troca de aba.
func TestEscapeClosesTheBufferPicker(t *testing.T) {
	model := bufferSession(t)
	active, err := model.application.Store.Active()
	if err != nil {
		t.Fatal(err)
	}
	activeID := active.ID()

	if err := model.application.Apply(action.BufferPicker); err != nil {
		t.Fatalf("abrindo o seletor: %v", err)
	}
	model.syncOverlay()
	model = press(t, model, tea.Key{Code: tea.KeyDown})
	model = press(t, model, tea.Key{Code: tea.KeyEscape})

	if model.application.Overlay != "none" {
		t.Errorf("Esc não fechou o seletor: %q", model.application.Overlay)
	}
	current, err := model.application.Store.Active()
	if err != nil {
		t.Fatal(err)
	}
	if current.ID() != activeID {
		t.Error("Esc trocou de aba")
	}
}
