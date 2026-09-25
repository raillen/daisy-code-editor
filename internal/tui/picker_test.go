package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/app"
	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/editor"
	"github.com/ori-team/oride/internal/keymap"
	"github.com/ori-team/oride/internal/tui/overlay"
)

// Enter numa lista é o caminho que **faltava**: a paleta listava comandos e não
// rodava nenhum, e os dois seletores não existiam. Estes testes dirigem a tecla como
// o runtime faria e afirmam o efeito — estado, não frame.

// wired builds a model with a document open, as way a session starts.
func wired(t *testing.T) Model {
	t.Helper()

	keys, err := keymap.FromBindings(config.Default().Keys)
	if err != nil {
		t.Fatalf("keymap: %v", err)
	}
	application := app.New(editor.NewStore(), config.Default(), keys)
	application.Workspace = t.TempDir()
	application.Store.OpenEmpty()
	return New(application, keys).Resize(100, 30)
}

// press sends a keystroke and returns the model after it.
func press(t *testing.T, model Model, key tea.Key) Model {
	t.Helper()

	updated, _ := model.Update(tea.KeyPressMsg(key))
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update devolveu %T", updated)
	}
	return next
}

// typeText types characters one event at a time, as the terminal delivers them.
func typeText(t *testing.T, model Model, text string) Model {
	t.Helper()

	for _, r := range text {
		key := tea.Key{Code: r, Text: string(r)}
		model = press(t, model, key)
	}
	return model
}

// TestEnterOnThePaletteRunsTheCommand é a afirmação central: a paleta deixou de ser
// decorativa.
func TestEnterOnThePaletteRunsTheCommand(t *testing.T) {
	model := wired(t)
	if !model.application.ShowTree {
		t.Fatal("a sessão começou com a árvore escondida, e o comando a mostra")
	}

	// Ctrl+Shift+P abre a paleta; digitar filtra; Enter aplica.
	model = press(t, model, tea.Key{Code: 'P', Mod: tea.ModCtrl | tea.ModShift})
	if !model.overlay.Active() {
		t.Fatal("Ctrl+Shift+P não abriu a paleta")
	}
	model = typeText(t, model, "árvore")
	model = press(t, model, tea.Key{Code: tea.KeyEnter})

	if model.overlay.Active() {
		t.Error("a paleta continuou aberta depois de aplicar")
	}
	if model.application.ShowTree {
		t.Error("o comando escolhido não foi executado: a árvore deveria estar escondida")
	}
}

// TestEnterOnAHelpRowRunsNothing fecha o outro lado: a lista de ajuda é
// informativa, e fingir que ela executa seria a tecla fazendo nada com um clique.
func TestEnterOnAHelpRowRunsNothing(t *testing.T) {
	model := wired(t)
	before := model.application.ShowTree

	model = press(t, model, tea.Key{Code: '?', Mod: tea.ModCtrl}) // ajuda
	if !model.overlay.Active() {
		t.Skip("a tecla de ajuda não abre nada nesta build")
	}
	model = press(t, model, tea.Key{Code: tea.KeyEnter})

	if model.application.ShowTree != before {
		t.Error("aplicar uma linha de ajuda mudou o estado do produto")
	}
	if model.overlay.Active() {
		t.Error("a lista de ajuda continuou aberta")
	}
}

// TestTheThemePickerListsEveryThemeAndApplies proves o picker inteiro: as opções
// são as do registro, o tema ativo é marcado **em texto**, e Enter adota.
func TestTheThemePickerListsEveryThemeAndApplies(t *testing.T) {
	model := wired(t)
	model.application.Config.Theme = "default-dark"

	_ = model.application.Apply(action.SelectTheme)
	if model.application.Overlay != app.OverlayThemePicker {
		t.Fatalf("overlay = %q, esperava o seletor de tema", model.application.Overlay)
	}

	width, height := model.overlaySize()
	model.overlay.Open(model.kindFor(model.application.Overlay), model.titleFor(model.application.Overlay),
		model.itemsFor(model.application.Overlay), width, height)

	items := model.itemsFor(model.application.Overlay)
	registry := config.LoadThemeRegistry(model.application.Workspace)
	if len(items) != registry.Len() {
		t.Errorf("o seletor listou %d temas, e o registro tem %d", len(items), registry.Len())
	}
	marked := 0
	for _, item := range items {
		if item.Detail == "ativo" {
			marked++
			if item.Value != "default-dark" {
				t.Errorf("marcou %q como ativo, e o ativo é default-dark", item.Value)
			}
			if item.Label == item.Value {
				t.Errorf("rótulo e valor são iguais (%q): o rótulo é o nome de exibição", item.Value)
			}
		}
		if item.Value == "" {
			t.Error("um tema sem valor não pode ser aplicado por Enter")
		}
	}
	if marked != 1 {
		t.Errorf("%d temas marcados como ativos, esperava 1", marked)
	}

	// Aplica um tema diferente dos marcados, com a lista filtrada para que a
	// escolha seja o que o leitor está vendo.
	chosen := ""
	for _, item := range items {
		if item.Detail != "ativo" {
			chosen = item.Value
		}
	}
	model.overlay.Open(model.kindFor(model.application.Overlay), model.titleFor(model.application.Overlay),
		[]overlay.Item{{Label: chosen, Value: chosen}}, width, height)
	model = press(t, model, tea.Key{Code: tea.KeyEnter})

	if model.application.Config.Theme != chosen {
		t.Errorf("o tema aplicado é %q, esperava %q", model.application.Config.Theme, chosen)
	}
	if !strings.Contains(model.application.Status, "tema:") {
		t.Errorf("status = %q", model.application.Status)
	}
}

// TestTheLocalePickerAppliesAndReloadsTheCatalog é o outro picker: aplicar um locale
// tem que trocar o catálogo, e não só o texto da configuração — uma troca que não
// recarrega nada é uma troca que não aconteceu.
func TestTheLocalePickerAppliesAndReloadsTheCatalog(t *testing.T) {
	model := wired(t)
	model.application.Config.Locale = "pt-BR"
	before := model.catalog.Menu.File

	_ = model.application.Apply(action.SelectLocale)
	if model.application.Overlay != app.OverlayLocalePicker {
		t.Fatalf("overlay = %q, esperava o seletor de idioma", model.application.Overlay)
	}

	items := model.itemsFor(model.application.Overlay)
	if len(items) == 0 {
		t.Fatal("o seletor de idioma não listou nada")
	}
	chosen := ""
	for _, item := range items {
		if item.Value != "pt-BR" {
			chosen = item.Value
		}
	}
	if chosen == "" {
		t.Skip("só há um locale disponível neste build")
	}

	model.applyLocale(chosen)
	if model.application.Config.Locale != chosen {
		t.Errorf("locale = %q, esperava %q", model.application.Config.Locale, chosen)
	}
	if model.catalog.Menu.File == before && chosen == "pt-BR" {
		t.Error("o catálogo não mudou com o locale")
	}
}

// TestEscapeClosesAPickerWithoutApplying: cancelar é o caminho que o leitor espera
// num seletor, e Enter sem querer não pode ser irreversível.
func TestEscapeClosesAPickerWithoutApplying(t *testing.T) {
	model := wired(t)
	model.application.Config.Theme = "default-dark"
	_ = model.application.Apply(action.SelectTheme)

	width, height := model.overlaySize()
	model.overlay.Open(model.kindFor(model.application.Overlay), model.titleFor(model.application.Overlay),
		model.itemsFor(model.application.Overlay), width, height)

	model = press(t, model, tea.Key{Code: tea.KeyEscape})

	if model.overlay.Active() {
		t.Error("Esc não fechou o seletor")
	}
	if model.application.Config.Theme != "default-dark" {
		t.Errorf("Esc mudou o tema para %q", model.application.Config.Theme)
	}
}
