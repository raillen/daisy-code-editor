package app

import (
	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/editor"
)

// viewCommands change what is on screen and where the focus is.
//
// They do not touch a document, so they keep working with an empty store — which
// is what lets someone toggle the tree or quit before opening a file.
func viewCommands() map[action.Action]Handler {
	return map[action.Action]Handler{
		action.ToggleTree: modelHandler(func(a *App, _ *editor.Document) error {
			a.ShowTree = !a.ShowTree
			return nil
		}),
		action.FocusTree: modelHandler(func(a *App, _ *editor.Document) error {
			a.Focus = FocusTree
			return nil
		}),
		action.FocusEditor: modelHandler(func(a *App, _ *editor.Document) error {
			a.Focus = FocusEditor
			return nil
		}),

		// Os dois seletores nomeiam a sobreposição e **sayem como usá-la**, que é
		// a mesma frase do oráculo (crates/oride-app/src/app/actions.rs): um
		// seletor que não diz o que Enter faz parece um menu decorativo.
		action.SelectTheme: modelHandler(func(a *App, _ *editor.Document) error {
			a.Overlay = OverlayThemePicker
			a.Status = "seletor de tema: ↑↓ navega/preview · Enter aplica · Esc cancela"
			return nil
		}),
		action.SelectLocale: modelHandler(func(a *App, _ *editor.Document) error {
			a.Overlay = OverlayLocalePicker
			a.Status = "seletor de idioma: ↑↓ seleciona · Enter aplica · Esc cancela"
			return nil
		}),
	}
}

// The names the model uses for the overlay in the dump.
//
// São os mesmos nomes que o oráculo usa no seu `Overlay` — o dump compara a
// chave, e um nome diferente aqui seria uma divergência que o nome da ferramenta
// não explica. **Exportados** porque a TUI precisa abri-los: e a TUI deriva os
// dela destes, em vez de declarar as mesmas strings em dois pacotes — que foi
// exatamente a confusão que o ledger já registrou uma vez com o overlay da busca.
const (
	OverlayNone         = "none"
	OverlayFind         = "find"
	OverlayThemePicker  = "ThemePicker"
	OverlayLocalePicker = "LocalePicker"
)
