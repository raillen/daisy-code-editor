package app

import (
	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/editor"
	"github.com/ori-team/oride/internal/fs"
)

// panelCommands are the actions about the panels and the focus between them.
//
// São as teclas que o leitor aperta **sem pensar** — toggles e foco — e todas
// estavam na tabela gerada sem tratamento. A regra de aptidão do S4 é que as cobra;
// estas são as que fecham o buraco mais rápido porque a infraestrutura já existe:
// o grafo de foco conhece `Tree`, o layout conhece a região do terminal, e a
// `Tree` sabe se recarregar.
func panelCommands() map[action.Action]Handler {
	return map[action.Action]Handler{
		// Focus alternates between the tree and the editor, and the tree appears
		// if it is hidden: a shortcut whose target is not on screen is a shortcut
		// to nothing.
		action.FocusToggleTreeEditor: modelHandler(func(a *App, _ *editor.Document) error {
			if a.Focus == FocusTree {
				a.Focus = FocusEditor
				a.Status = "foco: editor"
				return nil
			}
			a.ShowTree = true
			if a.Tree == nil {
				tree, err := fs.Open(a.Workspace, a.Config.Tree.ShowHidden)
				if err != nil {
					return err
				}
				a.Tree = tree
			}
			a.Focus = FocusTree
			a.Status = "foco: árvore"
			return nil
		}),

		action.TreeRefresh: modelHandler(func(a *App, _ *editor.Document) error {
			if a.Tree == nil {
				// "Não há árvore" não é erro: a tecla recarrega o que existe, e o
				// que não existe é aberto por `FocusToggleTreeEditor`.
				return nil
			}
			if err := a.Tree.Refresh(); err != nil {
				a.Status = "refresh: " + err.Error()
				return nil
			}
			a.Status = "árvore recarregada"
			return nil
		}),

		// Mouse is a toggle of a capability, and the message says both how to
		// turn it on and why it is off by default — because a reader who just
		// turned off mouse selection will not connect the two without being told.
		action.ToggleMouse: modelHandler(func(a *App, _ *editor.Document) error {
			a.Config.Mouse = !a.Config.Mouse
			if a.Config.Mouse {
				a.Status = "mouse: ON · clique/drag/scroll · desligar: View → Mouse ou mouse=false"
			} else {
				a.Status = "mouse: OFF (padrão) · ligar: View → Enable mouse ou mouse=true no TOML"
			}
			return nil
		}),

		action.ReloadFile: documentHandler(func(a *App, document *editor.Document) error {
			if err := document.ReloadFromDisk(); err != nil {
				// A falha **não** é erro do programa: recarregar um arquivo que
				// sumiu é uma situação normal, e o leitor precisa do motivo na
				// barra em vez de um erro que interrompe a tecla.
				a.Status = "reload: " + err.Error()
				return nil
			}
			a.Status = "arquivo recarregado"
			return nil
		}),
	}
}
