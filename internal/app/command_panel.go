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
			if a.ShowBottom && a.BottomTab == BottomTabTerminal {
				a.ShowBottom = false
				if a.Focus == FocusTerminal {
					a.Focus = FocusEditor
				}
				a.Status = "painel oculto"
				return nil
			}
			a.EnsureTerminalHeight()
			a.ShowBottom = true
			a.BottomTab = BottomTabTerminal
			a.Focus = FocusTerminal
			a.Status = "terminal: digite · Esc=editor · Ctrl+C no shell · Ctrl+` fecha"
			return nil
		}),

		action.FocusTerminal: modelHandler(func(a *App, _ *editor.Document) error {
			a.ShowBottom = true
			a.BottomTab = BottomTabTerminal
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
			a.EnsureTerminalHeight()
			a.ShrinkTerminal(2)
			a.Status = "terminal height " + itoa(a.TerminalHeight)
			return nil
		}),

		// Trocar de aba é "focar o próximo painel", que é o nome que a ação tem no
		// oráculo e que o keymap já publica em `ctrl+alt+left` e `ctrl+alt+right`.
		// Nenhuma tabela nova: `internal/action` e `internal/config/keybindings.go`
		// são **gerados** do Rust, e um id ou um binding inventados seriam uma
		// divergência de paridade — a mesma razão que fez o wrap virar B19.
		action.FocusNextPane: modelHandler(func(a *App, _ *editor.Document) error {
			if !a.ShowBottom {
				a.ShowBottom = true
				a.BottomTab = BottomTabTerminal
				return nil
			}
			a.BottomTab = nextBottomTab(a.BottomTab)
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

// nextBottomTab é a próxima aba do painel, em ciclo.
//
// Só para a frente, e a razão está no keymap: o oráculo publica
// `focus_next_pane` — e não um `focus_prev_pane` —, e as tabelas de ação e de
// bindings são **geradas** dele. Um "aba anterior" seria um id novo, e um id novo
// é divergência de paridade: a mesma razão que fez o wrap virar B19. Três abas
// com um ciclo à frente é o que o produto oferece sem mentir sobre a referência.
func nextBottomTab(current string) string {
	order := []string{BottomTabTerminal, BottomTabProblems, BottomTabSearch}
	for index, tab := range order {
		if tab == current {
			return order[(index+1)%len(order)]
		}
	}
	return order[0]
}

// The bottom panel's tabs, and a default that nenhum caminho pode deixar vazio.
const (
	BottomTabTerminal = "terminal"
	BottomTabProblems = "problems"
	BottomTabSearch   = "search"
)

// EnsureTerminalHeight dá ao painel a altura que a configuração pediu, na
// primeira vez em que o terminal aparece.
func (a *App) EnsureTerminalHeight() {
	if a.TerminalHeight < minTerminalHeight {
		a.TerminalHeight = a.Config.Terminal.DefaultHeight
	}
	if a.TerminalHeight < minTerminalHeight {
		a.TerminalHeight = minTerminalHeight
	}
}

// GrowTerminal makes the panel taller, following the limits the reference uses.
func (a *App) GrowTerminal(delta int) {
	a.EnsureTerminalHeight()
	a.TerminalHeight = min(maxTerminalHeight, max(minTerminalHeight, a.TerminalHeight+delta))
	a.ShowBottom = true
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
