package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ori-team/oride/internal/app"
	"github.com/ori-team/oride/internal/tui/focus"
)

// As três funções abaixo são o vocabulário do modelo em teste, e existir é o que
// evita que cada teste escreva a string do foco — que é o mesmo valor que o
// dump do oráculo compara.
func appFocusEditor() app.Focus { return app.FocusEditor }
func appFocusTree() app.Focus   { return app.FocusTree }

// O mouse é metade da navegação, e até esta rodada ele não existia: a TUI
// **declarava** a capacidade no `tea.View` e não tratava nenhuma mensagem. Estes
// testes dirigem coordenadas de clique e de roda como o terminal faria.

// click sends a left click at a cell, pelo `Update`.
func click(t *testing.T, model Model, x, y int) Model {
	t.Helper()
	updated, _ := model.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	return updated.(Model)
}

// wheel sends a wheel event at a cell.
func wheel(t *testing.T, model Model, y int, button tea.MouseButton) Model {
	t.Helper()
	updated, _ := model.Update(tea.MouseWheelMsg{X: 20, Y: y, Button: button})
	return updated.(Model)
}

// TestAClickOnTheTreeSelectsAndTheSecondOpens is a regra de lista que evita o pior
// defeito de um clique: abrir o arquivo errado por um clique que ninguém revisou.
func TestAClickOnTheTreeSelectsAndTheSecondOpens(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Tree = stubTree(t)
	model.application.ShowTree = true
	regions := model.regions()

	// A primeira linha do arquivo é a segunda linha da árvore (a raiz é a
	// primeira), e o clique é na coluna do texto.
	row := regions.Tree.Y + 1
	model = click(t, model, regions.Tree.X+2, row)
	if model.application.Tree.SelectedIndex() == 0 {
		t.Error("o clique na linha não moveu a seleção")
	}

	if _, has := model.application.Store.ActiveID(); has {
		t.Fatal("o arquivo já estava aberto antes do segundo clique")
	}

	model = click(t, model, regions.Tree.X+2, row)
	if _, has := model.application.Store.ActiveID(); !has {
		t.Fatal("o segundo clique na mesma linha não abriu o arquivo")
	}
	if model.application.Focus != appFocusEditor() {
		t.Errorf("abrir o arquivo não devolveu o foco ao editor (foco = %q)", model.application.Focus)
	}
}

// TestAClickOnTheTreeGivesItTheKeyboard: a árvore é clicável, e o clique precisa
// trazer o teclado junto — senão as setas continuam indo para o documento, que é
// o bug de foco que a auditoria já registrou uma vez.
func TestAClickOnTheTreeGivesItTheKeyboard(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Tree = stubTree(t)
	model.application.ShowTree = true
	regions := model.regions()

	model = click(t, model, regions.Tree.X+2, regions.Tree.Y+1)

	if model.application.Focus != appFocusTree() {
		t.Errorf("o clique na árvore não deu o foco a ela (foco = %q)", model.application.Focus)
	}
}

// TestAClickOutsideThePanelsReturnsToTheEditor: clicar no texto devolve o teclado
// para onde o texto é escrito.
func TestAClickOutsideThePanelsReturnsToTheEditor(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Tree = stubTree(t)
	model.application.ShowTree = true
	model.application.Focus = appFocusTree()
	regions := model.regions()

	model = click(t, model, regions.Editor.X+5, regions.Editor.Y+2)

	if model.application.Focus != appFocusEditor() {
		t.Errorf("o clique no editor não devolveu o foco (foco = %q)", model.application.Focus)
	}
}

// TestTheWheelScrolls: a roda move a visão, e sem estado de rolagem próprio o que
// se move é o caret.
func TestTheWheelScrolls(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Store.OpenEmpty()
	document, _ := model.application.Store.Active()
	_ = document.InsertText(strings.Repeat("linha\n", 60))

	regions := model.regions()
	model = wheel(t, model, regions.Editor.Y+3, tea.MouseWheelDown)
	down, _ := document.Caret()

	model = wheel(t, model, regions.Editor.Y+3, tea.MouseWheelUp)
	up, _ := document.Caret()

	if down.Line == 0 {
		t.Error("a roda para baixo não moveu o caret")
	}
	if up.Line >= down.Line {
		t.Errorf("a roda para cima não voltou: %d → %d", down.Line, up.Line)
	}
}

// TestTheWheelDoesNothingWhenTheOverlayIsOpen: sobreposição captura tudo, e um
// scroll por baixo dela é o leitor rolando uma lista que não está vendo.
func TestTheWheelDoesNothingWhenTheOverlayIsOpen(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Store.OpenEmpty()
	chord := chordFor(t, model.keys, "command_palette")
	model = pressChord(t, model, chord)
	if !model.overlay.Active() {
		t.Skip("a paleta não abriu, e o teste não tem o que medir")
	}
	document, _ := model.application.Store.Active()
	_ = document.InsertText(strings.Repeat("linha\n", 30))

	before, _ := document.Caret()
	wheel(t, model, 10, tea.MouseWheelDown)

	if after, _ := document.Caret(); after.Line != before.Line {
		t.Errorf("a roda rolou o documento por baixo da sobreposição: %d → %d", before.Line, after.Line)
	}
}

// TestOneTabReachesTheTree is o caminho mais usado de um editor de código, e ele
// exigia **quatro** pressões: editor → menu → abas → árvore. A ordem do `Tab` põe
// os painéis antes do chrome por isso, e este teste é o que impede a ordem de
// voltar atrás sem ninguém notar.
func TestOneTabReachesTheTree(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Tree = stubTree(t)
	model.application.ShowTree = true

	model = press(t, model, tea.Key{Code: tea.KeyTab})

	if model.surface != focusTree() {
		t.Errorf("um Tab foi para %q, e a árvore é a uma pressão", model.surface)
	}
	if model.application.Focus != appFocusTree() {
		t.Errorf("o modelo ficou em %q", model.application.Focus)
	}
}

// TestTabFromTheTreeReturnsToTheEditor is a outra ponta do ciclo: da árvore, um
// `Tab` devolve o teclado para onde o texto é escrito, e não para a barra de menu.
func TestTabFromTheTreeReturnsToTheEditor(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Tree = stubTree(t)
	model.application.ShowTree = true

	model = press(t, model, tea.Key{Code: tea.KeyTab})
	model = press(t, model, tea.Key{Code: tea.KeyTab})

	if model.surface != focusEditor() {
		t.Errorf("o segundo Tab foi para %q, e o editor é o destino de volta", model.surface)
	}
}

// focusTree and focusEditor are the presentation's vocabulary, written once here
// because two tests use them and the string is um contrato com o modelo.
func focusTree() focus.Surface   { return focus.Tree }
func focusEditor() focus.Surface { return focus.Editor }
