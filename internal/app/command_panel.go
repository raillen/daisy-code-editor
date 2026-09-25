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

		// O painel de terminal. A altura inicial vem de `[terminal].default_height`
		// — a chave que era lida por ninguém desde o port, e que agora tem um
		// leitor de verdade.
		action.ToggleTerminal: modelHandler(func(a *App, _ *editor.Document) error {
			if a.ToggleTerminalPanel() {
				a.Focus = FocusTerminal
				a.Status = "terminal: digite · Esc=editor · Ctrl+C no shell · Ctrl+` fecha"
			} else if a.Focus == FocusTerminal {
				a.Focus = FocusEditor
				a.Status = "terminal oculto"
			}
			return nil
		}),

		action.FocusTerminal: modelHandler(func(a *App, _ *editor.Document) error {
			if !a.ShowTerminal {
				a.ShowTerminal = true
			}
			a.Focus = FocusTerminal
			a.Status = "foco: terminal · digite · Esc=editor"
			return nil
		}),

		action.TerminalGrow: modelHandler(func(a *App, _ *editor.Document) error {
			a.GrowTerminal(2)
			a.Status = "terminal height " + itoa(a.TerminalHeight)
			return nil
		}),

		action.TerminalShrink: modelHandler(func(a *App, _ *editor.Document) error {
			a.ShrinkTerminal(2)
			a.Status = "terminal height " + itoa(a.TerminalHeight)
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

// ToggleTerminalPanel shows or hides the terminal panel, and reports whether it
// is now visible.
//
// A primeira vez usa `[terminal].default_height`, que é o que a configuração
// promete; depois quem manda é o leitor, com grow/shrink. Três linhas é o piso — um
// painel de duas linhas não é um terminal, é um erro de layout.
func (a *App) ToggleTerminalPanel() bool {
	if a.ShowTerminal {
		a.ShowTerminal = false
		return false
	}
	if a.TerminalHeight < minTerminalHeight {
		a.TerminalHeight = a.Config.Terminal.DefaultHeight
	}
	if a.TerminalHeight < minTerminalHeight {
		a.TerminalHeight = minTerminalHeight
	}
	a.ShowTerminal = true
	return true
}

// GrowTerminal makes the panel taller, following the limits the reference uses.
func (a *App) GrowTerminal(delta int) {
	a.TerminalHeight = min(maxTerminalHeight, max(minTerminalHeight, a.TerminalHeight+delta))
	a.ShowTerminal = true
}

// ShrinkTerminal makes the panel shorter, down to the floor.
func (a *App) ShrinkTerminal(delta int) {
	a.TerminalHeight = max(minTerminalHeight, a.TerminalHeight-delta)
}

// The panel's size limits, the same as the reference terminal (3 to 40 rows).
const (
	minTerminalHeight = 3
	maxTerminalHeight = 40
)
