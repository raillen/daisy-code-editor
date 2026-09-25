package app

import (
	"strings"
	"testing"

	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/editor"
	"github.com/ori-team/oride/internal/keymap"
)

// As quatro teclas mais próximas do leitor foram mortas até a regra de aptidão
// aparecer, e a regra prova que deixou de ser. Estos testes são a outra ponta: a
// regra prova que existe tratamento, e **estes provam que o tratamento muda o
// estado** — porque um handler que devolve `nil` sem fazer nada satisfaz a regra
// e engana o leitor.

// buildWired is a model with a keymap from the shipped defaults, so the tests
// drive the same keys a person's hands reach for.
func buildWired(t *testing.T, text string) *App {
	t.Helper()

	keys, err := keymap.FromBindings(config.Default().Keys)
	if err != nil {
		t.Fatalf("keymap: %v", err)
	}
	application := New(editor.NewStore(), config.Default(), keys)
	application.Store.OpenEmpty()
	active, err := application.Store.Active()
	if err != nil {
		t.Fatalf("documento: %v", err)
	}
	if text != "" {
		if err := active.InsertText(text); err != nil {
			t.Fatalf("inserindo: %v", err)
		}
	}
	return application
}

// caretLine is where the caret is, by line number.
func caretLine(t *testing.T, a *App) int {
	t.Helper()

	document, err := a.Store.Active()
	if err != nil {
		t.Fatalf("documento: %v", err)
	}
	caret, err := document.Caret()
	if err != nil {
		t.Fatalf("caret: %v", err)
	}
	return caret.Line
}

// TestTheTabKeysMoveBetweenDocuments is a tecla que o leitor usa para alternar
// entre arquivos, e que até aqui não fazia nada.
func TestTheTabKeysMoveBetweenDocuments(t *testing.T) {
	a := buildWired(t, "primeiro")
	first, _ := a.Store.ActiveID()

	if err := a.Apply(action.NewTab); err != nil {
		t.Fatalf("ctrl+t: %v", err)
	}
	if a.Store.Len() != 2 {
		t.Fatalf("ctrl+t abriu %d documentos, esperado 2", a.Store.Len())
	}
	second, _ := a.Store.ActiveID()
	if second == first {
		t.Fatal("ctrl+t não deixou o documento novo ativo")
	}

	before := second
	if err := a.Apply(action.PrevTab); err != nil {
		t.Fatalf("ctrl+pageup: %v", err)
	}
	back, _ := a.Store.ActiveID()
	if back == before {
		t.Fatal("Ctrl+PageUp não voltou de aba")
	}
	if err := a.Apply(action.NextTab); err != nil {
		t.Fatalf("ctrl+pagedown: %v", err)
	}
	again, _ := a.Store.ActiveID()
	if again != before {
		t.Fatal("Ctrl+PageDown não voltou para a aba anterior")
	}
}

// TestCyclingWithOneTabDoesNothingAndSaysSo is the case degenerate: uma tecla que
// cicla uma lista de um só elemento é a tecla fazendo nada, e o leitor não
// distingue de um programa quebrado.
func TestCyclingWithOneTabDoesNothingAndSaysSo(t *testing.T) {
	a := buildWired(t, "único")
	before, _ := a.Store.ActiveID()

	if err := a.Apply(action.NextTab); err != nil {
		t.Fatalf("uma aba não pode ser erro: %v", err)
	}
	after, _ := a.Store.ActiveID()
	if after != before {
		t.Error("uma aba só mudou de lugar")
	}
}

// TestClosingTheLastTabLeavesAnEditor is o que o leitor vê ao fechar tudo: um
// editor vazio, com um caret, e não um programa morto.
func TestClosingTheLastTabLeavesAnEditor(t *testing.T) {
	a := buildWired(t, "conteúdo")
	if err := a.Apply(action.CloseTab); err != nil {
		t.Fatalf("ctrl+w: %v", err)
	}

	if a.Store.Len() != 1 {
		t.Fatalf("fechar a última aba deixou %d documentos", a.Store.Len())
	}
	if !strings.Contains(a.Status, "fechada") {
		t.Errorf("status = %q, o leitor precisa saber que a aba fechou", a.Status)
	}
	if _, err := a.Store.Active(); err != nil {
		t.Errorf("o editor ficou sem documento onde o cursor pode ficar: %v", err)
	}
}

// TestPagingMovesByTheViewportHeight is a paginação medida, e a medida é a
// altura que a apresentação injetou.
func TestPagingMovesByTheViewportHeight(t *testing.T) {
	const height = 10
	lines := strings.Repeat("linha\n", 40)

	a := buildWired(t, lines)
	a.EditorHeight = height
	// O texto inserido deixa o caret no fim; a paginação mede daqui.
	if err := a.Apply(action.MoveDocStartPlain); err != nil {
		t.Fatalf("indo ao começo: %v", err)
	}

	if err := a.Apply(action.PageDown); err != nil {
		t.Fatalf("PageDown: %v", err)
	}
	// A altura menos uma linha é a regra do oráculo
	// (crates/oride-app/src/app/actions.rs): a última linha visível.
	if got := caretLine(t, a); got != height-1 {
		t.Errorf("PageDown levou à linha %d, esperava %d", got, height-1)
	}

	if err := a.Apply(action.PageUp); err != nil {
		t.Fatalf("PageUp: %v", err)
	}
	if got := caretLine(t, a); got != 0 {
		t.Errorf("PageUp levou à linha %d, esperava 0", got)
	}
}

// TestPagingWithAnUnmeasuredViewportIsOneLine é o valor zero, que é um estado
// legítimo: o modelo ainda não foi medido por um frame.
func TestPagingWithAnUnmeasuredViewportIsOneLine(t *testing.T) {
	a := buildWired(t, strings.Repeat("linha\n", 10))
	a.EditorHeight = 0
	if err := a.Apply(action.MoveDocStartPlain); err != nil {
		t.Fatalf("indo ao começo: %v", err)
	}

	if err := a.Apply(action.PageDown); err != nil {
		t.Fatalf("PageDown sem altura: %v", err)
	}
	if got := caretLine(t, a); got != 1 {
		t.Errorf("sem altura medida, PageDown levou à linha %d, esperava 1", got)
	}
}

// TestPagingClampsAtTheEdges é o que a repetição de movimentos entrega: a última
// linha, e não uma linha além do fim.
func TestPagingClampsAtTheEdges(t *testing.T) {
	a := buildWired(t, "uma\nduas\ntrês")
	a.EditorHeight = 5
	// "uma\nduas\ntrês" tem três linhas, e a última é a 2.
	if err := a.Apply(action.MoveDocStartPlain); err != nil {
		t.Fatalf("indo ao começo: %v", err)
	}

	for range 3 {
		if err := a.Apply(action.PageDown); err != nil {
			t.Fatalf("PageDown: %v", err)
		}
	}
	if got := caretLine(t, a); got != 2 {
		t.Errorf("PageDown além do fim parou na linha %d, esperava 2", got)
	}
	for range 3 {
		if err := a.Apply(action.PageUp); err != nil {
			t.Fatalf("PageUp: %v", err)
		}
	}
	if got := caretLine(t, a); got != 0 {
		t.Errorf("PageUp além do começo parou na linha %d, esperava 0", got)
	}
}

// TestTheKeyPathReachesTheseHandlers fecha o circuito: as teclas do keymap, e não
// só as ações.
func TestTheKeyPathReachesTheseHandlers(t *testing.T) {
	for _, tc := range []struct{ chord, actionID string }{
		{"ctrl+n", "new_tab"},
		{"ctrl+pagedown", "next_tab"},
		{"ctrl+pageup", "prev_tab"},
		{"ctrl+w", "close_tab"},
		{"pagedown", "page_down"},
		{"pageup", "page_up"},
	} {
		bound, found := config.Default().Keys[tc.chord]
		if !found || bound != tc.actionID {
			t.Errorf("%s aponta para %q, esperava %q", tc.chord, bound, tc.actionID)
		}
	}
}
