package app

import (
	"testing"

	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/config"
)

// boundActions is every action the product advertises a key for.
//
// An action nobody can press is not a missing feature, it is a **false promise**:
// the keymap table is generated into docs/ui-ux/keymap.md and printed in the help,
// so a binding for an action with no handler is documentation that lies. The audit
// found 54 of them, and the reason they survived every gate is the reason this
// test exists: nothing asked.
func boundActions(t *testing.T) map[action.Action]bool {
	t.Helper()

	bound := map[action.Action]bool{}
	for _, id := range config.Default().Keys {
		parsed, err := action.Parse(id)
		if err != nil {
			t.Fatalf("binding com id de ação inválido %q: %v", id, err)
		}
		bound[parsed] = true
	}
	return bound
}

var withoutHandler = map[action.Action]string{
	"tree_new_file":       "M4 — criação de arquivo",
	"tree_new_dir":        "M4 — idem",
	"toggle_soft_wrap":    "M4 — soft wrap no editorview",
	"toggle_comment":      "M4 — comentário de linha",
	"toggle_md_preview":   "M4 — preview Markdown",
	"lsp_complete":        "M4 — cliente LSP sobe no M4",
	"lsp_hover":           "M4 — idem",
	"lsp_goto_definition": "M4 — idem",
	"lsp_format":          "M4 — idem",
	"toggle_diagnostics":  "M4 — idem",
	"split_vertical":      "M4 — splits",
	"split_horizontal":    "M4 — idem",
	"close_pane":          "M4 — idem",
	"multi_picker":        "M4 — multi-cursor",
	"jump_back":           "M4 — jump list",
	"jump_forward":        "M4 — idem",
	"undo_tree":           "M4 — histórico",
	"surround":            "M4 — edição",
}

var reachableFromThePaletteOnly = map[action.Action]string{
	"tree_grow":          "M4 — resizing de painel",
	"tree_shrink":        "M4 — idem",
	"resize_pane_grow":   "M4 — idem",
	"resize_pane_shrink": "M4 — idem",
	"toggle_modal":       "M4 — motor modal Vim",
	"run_tasks":          "M4 — task runner",
}

// interceptedBeforeTheKeymap names the actions the composition root handles on
// its own — e hoje está **vazia**. Ela existiu para as quatro sobreposições
// (paleta, which-key, ajuda, boas-vindas), que eram interceptadas antes do keymap;
// quando elas ganharam tratamento **no modelo**, migraram para a tabela e saíram
// daqui. A estrutura fica: a próxima ação que precisar de interceptação tem onde
// morar, sem ninguém precisar inventar o mecanismo.
var interceptedBeforeTheKeymap = map[action.Action]string{}

// TestEveryBoundActionHasAHandlerOrIsNamed is the fitness rule, applied to keys.
//
// A configuração sem consumidor é um defeito; uma **tecla** sem consumidor é o
// mesmo defeito com o leitor na frente, porque ela é impressa na ajuda e gerada
// para o documento. Este teste é a contraparte do que a auditoria pediu para a
// configuração: a regra falha o build quando uma ação anunciada não tem
// tratamento, e o mapa acima é a dívida nomeada.
func TestEveryBoundActionHasAHandlerOrIsNamed(t *testing.T) {
	bound := boundActions(t)

	for id := range bound {
		if _, handled := commands[id]; handled {
			if reason, excused := withoutHandler[id]; excused {
				t.Errorf("ação %q tem tratamento e continua na lista de dívida (%s)", id, reason)
			}
			continue
		}
		if reason, intercepted := interceptedBeforeTheKeymap[id]; intercepted {
			_ = reason
			continue
		}
		if _, excused := withoutHandler[id]; !excused {
			t.Errorf("ação %q tem tecla e nenhum tratamento: acrescente o handler ou "+
				"um motivo em withoutHandler", id)
		}
	}

	// O inverso também conta: uma entrada que já não tem tecla é dívida que
	// ninguém mais está pagando.
	for id := range withoutHandler {
		if !bound[id] {
			t.Errorf("dívida %q não tem tecla: mova para reachableFromThePaletteOnly", id)
		}
	}
	for id := range reachableFromThePaletteOnly {
		if bound[id] {
			t.Errorf("%q tem tecla e está na lista de sem-tecla", id)
		}
	}
}

// TestTheExceptionMapCoversEveryUnwiredAction is a count, not a judgement.
//
// It exists so the debt is a number someone can watch go down: the gate of the
// milestone asks for "no open exception", and a list that can only grow silently
// is not a gate.
func TestTheExceptionMapCoversEveryUnwiredAction(t *testing.T) {
	bound := boundActions(t)
	wired, intercepted, dead := 0, 0, 0
	for id := range bound {
		switch {
		case func() bool { _, ok := commands[id]; return ok }():
			wired++
		case func() bool { _, ok := interceptedBeforeTheKeymap[id]; return ok }():
			intercepted++
		default:
			dead++
		}
	}
	if t.Failed() {
		return
	}
	t.Logf("ações com tecla: %d · tratadas: %d · interceptadas pela TUI: %d · mortas: %d",
		len(bound), wired, intercepted, dead)
}
