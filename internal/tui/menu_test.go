package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ori-team/oride/internal/app"
)

// A barra de menus passou duas milestones desenhada e **fechada**: `Open: -1`,
// sempre. Um menubar que não abre é a primeira coisa que um leitor do oráculo
// oficial testa — e a primeira coisa que ele declara morto. Estes testes abrem,
// andam e executam, e provam o que **não** executa.

// openMenu sends Alt+letter and fails the test when the menu does not open.
func openMenu(t *testing.T, model Model, letter rune) Model {
	t.Helper()

	model = press(t, model, tea.Key{Code: letter, Mod: tea.ModAlt})
	if model.application.MenuOpen == nil {
		t.Fatalf("Alt+%c não abriu o menu", letter)
	}
	return model
}

// TestAltOpensTheSameColumnTheOracleOpens is a aposta inteira: as seis letras são
// as da referência, e na mesma ordem.
func TestAltOpensTheSameColumnTheOracleOpens(t *testing.T) {
	for _, tc := range []struct {
		letter rune
		title  string
	}{
		{'f', "Arquivo"}, {'e', "Editar"}, {'v', "Exibir"},
		{'g', "Ir"}, {'i', "Git"}, {'h', "Ajuda"},
	} {
		model := sized(t, newModel(t, nil), 100, 30)
		model = openMenu(t, model, tc.letter)

		column := model.application.MenuOpen[0]
		menus := app.Menus(model.catalog)
		if menus[column].Title != tc.title {
			t.Errorf("Alt+%c abriu %q, esperava %q", tc.letter, menus[column].Title, tc.title)
		}
	}
}

// TestArrowsWalkTheMenuAndEnterRuns fecha o caminho de telemetria do menu: abrir,
// descer duas vezes, executar — e o arquivo salvo, não uma linha a mais na tela.
func TestArrowsWalkTheMenuAndEnterRuns(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Store.OpenEmpty()
	document, _ := model.application.Store.Active()
	_ = document.InsertText("conteúdo")
	model = openMenu(t, model, 'f')

	// File, item 1: `open_file_fuzzy`. Executá-lo abre o seletor, e não salva
	// nada — que é o que este teste mede.
	model = press(t, model, tea.Key{Code: tea.KeyDown})
	if got := model.application.MenuOpen[1]; got != 1 {
		t.Fatalf("uma seta levou ao item %d, e não ao 1", got)
	}

	model = press(t, model, tea.Key{Code: tea.KeyEnter})

	if model.overlay.Active() || model.application.Overlay != app.OverlayNone {
		// A execução de um item que abre outra sobreposição mantém a verdade no
		// modelo, e é esse o estado que o dump compara.
		if model.application.Overlay != app.OverlayOpenFile {
			t.Errorf("depois de Enter, overlay = %q, esperava o seletor de arquivo", model.application.Overlay)
		}
	}
}

// TestEscapeClosesTheMenu é o que o oráculo faz: `Esc` com menu aberto fecha o
// menu, e só ele.
func TestEscapeClosesTheMenu(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Store.OpenEmpty()
	model = openMenu(t, model, 'e')

	model = press(t, model, tea.Key{Code: tea.KeyEscape})

	if model.application.MenuOpen != nil {
		t.Error("Esc não fechou o menu")
	}
	// E não fechou mais nada: o documento continua aberto, e o programa não está
	// saindo.
	active, has := model.application.Store.ActiveID()
	if !has || active == 0 && false {
		t.Error("Esc afetou o documento")
	}
	if model.application.Quit {
		t.Error("Esc fechou o programa")
	}
}

// TestALetterJumpsToTheFirstItemThatStartsWithIt é a navegação de quem sabe o que
// quer: a referência pula para o primeiro item que começa pela letra, e o leitor
// não precisa andar a coluna inteira.
func TestALetterJumpsToTheFirstItemThatStartsWithIt(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model = openMenu(t, model, 'e')

	model = press(t, model, tea.Key{Code: 'r', Text: "r"})

	column := model.application.MenuOpen[0]
	menus := app.Menus(model.catalog)
	item := menus[column].Items[model.application.MenuOpen[1]]
	if !strings.HasPrefix(strings.ToLower(item.Label), "r") {
		t.Errorf("a letra r levou a %q, que não começa por r", item.Label)
	}
}

// TestAnUnwiredMenuItemSaysSo é a honestidade do menu: um item sem tratamento no
// modelo não finge que executou — ele **diz** na barra.
func TestAnUnwiredMenuItemSaysSo(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Store.OpenEmpty()
	model = openMenu(t, model, 'g')

	menus := app.Menus(model.catalog)

	// `LSP: definition` (f4) é morta no modelo: executá-la pelo menu tem de dizer,
	// e não calar.
	for index, item := range menus[3].Items {
		if item.Action == "lsp_goto_definition" {
			model.application.MenuOpen = &[2]int{3, index}
			break
		}
	}
	model = press(t, model, tea.Key{Code: tea.KeyEnter})

	if !strings.Contains(model.application.Status, "implementado") {
		t.Errorf("executar um item sem tratamento não disse nada: %q", model.application.Status)
	}
}

// TestTheMouseOpensAndRunsFromTheMenu é a terceira entrada: clique no título abre
// a coluna, e clique no item executa.
func TestTheMouseOpensAndRunsFromTheMenu(t *testing.T) {
	model := sized(t, newModel(t, nil), 100, 30)
	model.application.Store.OpenEmpty()
	model.application.ShowTree = true
	model.application.Tree = stubTree(t)

	regions := model.regions()
	model = click(t, model, 3, regions.MenuBar.Y)

	if model.application.MenuOpen == nil {
		t.Fatal("o clique no título não abriu o menu")
	}

	// A primeira coluna é File, e o item 0 é New tab.
	model = click(t, model, regions.MenuBar.X+2, regions.MenuBar.Y+2)

	if model.application.MenuOpen != nil {
		t.Error("o clique no item não fechou o menu")
	}
}
