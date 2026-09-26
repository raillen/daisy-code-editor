package app

import (
	"strings"
	"testing"

	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/buffer"
	"github.com/ori-team/oride/internal/clipboard"
	"github.com/ori-team/oride/internal/config"
)

// O ciclo copy/paste/cut é o caminho que o leitor mais exercita de qualquer editor,
// e era um dos verbos que o keymap anunciava sem tratamento. Estes testes fixam a
// **regra** que o oráculo usa, que é a parte que se perde fácil: a seleção, ou a
// linha inteira quando não há seleção.
//
// **O clipboard é estado do processo**, não da aplicação: um teste que presume
// buffer vazio falha quando alguém acrescenta um teste antes dele, e a falha
// aparece no teste errado. Por isso todo teste que cola começa esvaziando o
// buffer explicitamente — a mesma razão pela qual o oráculo usa um lock de teste
// (`CLIPBOARD_TEST_LOCK` em crates/oride-app/src/app/tests.rs).

// selectAll marks the whole document, the way Shift+Ctrl+A does.
func selectAll(t *testing.T, a *App) {
	t.Helper()

	if err := a.Apply(action.SelectAll); err != nil {
		t.Fatalf("select all: %v", err)
	}
}

// TestCopyPasteRoundTrip is the plain path: copy a selection, paste it elsewhere.
func TestCopyPasteRoundTrip(t *testing.T) {
	clipboard.Copy("")
	a := buildWired(t, "abcdef")
	selectAll(t, a)

	if err := a.Apply(action.Copy); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if !strings.Contains(a.Status, "copied 6 bytes") {
		t.Errorf("status = %q, o leitor precisa do tamanho em bytes — e a frase é a do oráculo, porque o dump a compara", a.Status)
	}
	if a.ClipboardWrite == "" {
		t.Error("a cópia não produziu sequência OSC 52 para o terminal")
	}

	document, _ := a.Store.Active()
	document.JumpToByte(buffer.Offset(len("abcdef")))
	if err := a.Apply(action.Paste); err != nil {
		t.Fatalf("paste: %v", err)
	}
	if got := document.Buffer().String(); got != "abcdefabcdef" {
		t.Errorf("o buffer ficou %q, esperava o texto colado", got)
	}
}

// TestCopyWithNoSelectionCopiesTheLine is the rule que o leitor espera e que um
// handler descuidado perde: sem seleção, copia a **linha**, não um caractere.
func TestCopyWithNoSelectionCopiesTheLine(t *testing.T) {
	clipboard.Copy("")
	a := buildWired(t, "primeira\nsegunda")
	if err := a.Apply(action.MoveDocStartPlain); err != nil {
		t.Fatalf("indo ao começo: %v", err)
	}
	if err := a.Apply(action.Copy); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if !strings.Contains(a.Status, "copied 8 bytes") {
		t.Errorf("status = %q, esperava a linha inteira (8 bytes)", a.Status)
	}

	if err := a.Apply(action.Paste); err != nil {
		t.Fatalf("paste: %v", err)
	}
	document, _ := a.Store.Active()
	// A linha copiada é o **conteúdo** da linha, sem o fim de linha: é o que o
	// oráculo copia (`line_text`), e é o que o leitor quer colar de volta.
	//
	// A assimetria com o corte é deliberada e vem do oráculo: cortar remove a
	// linha com o fim, porque deixar o `\n` produziria uma linha em branco a cada
	// corte; copiar devolve o texto, para o que for colado virar uma linha só.
	if got := document.Buffer().String(); !strings.HasPrefix(got, "primeiraprimeira\n") {
		t.Errorf("a colagem não trouxe a linha: %q", got)
	}
}

// TestCutWithNoSelectionRemovesTheLineAndItsBreak é o defeito que o teste do
// oráculo fixou: cortar a linha e deixar o fim de linha produz uma linha em branco
// a cada corte, e o documento cresce enquanto o leitor acredita estar deletando.
func TestCutWithNoSelectionRemovesTheLineAndItsBreak(t *testing.T) {
	clipboard.Copy("")
	a := buildWired(t, "one\r\ntwo")
	if err := a.Apply(action.MoveDocStartPlain); err != nil {
		t.Fatalf("indo ao começo: %v", err)
	}

	if err := a.Apply(action.Cut); err != nil {
		t.Fatalf("cut: %v", err)
	}
	document, _ := a.Store.Active()
	if got := document.Buffer().String(); got != "two" {
		t.Errorf("o buffer ficou %q, esperava \"two\"", got)
	}
}

// TestCutACopyOrderPastaFirst é a ordem que não pode ser invertida: cortar antes
// de copiar apaga o texto de quem vai colar, e um ctrl+x que não cola é um ctrl+x
// que perdeu dados.
func TestCutCopiesBeforeItDeletes(t *testing.T) {
	clipboard.Copy("")
	a := buildWired(t, "abcdef")
	selectAll(t, a)

	if err := a.Apply(action.Cut); err != nil {
		t.Fatalf("cut: %v", err)
	}
	document, _ := a.Store.Active()
	if got := document.Buffer().String(); got != "" {
		t.Fatalf("o corte não removeu a seleção: %q", got)
	}

	if err := a.Apply(action.Paste); err != nil {
		t.Fatalf("paste: %v", err)
	}
	if got := document.Buffer().String(); got != "abcdef" {
		t.Errorf("depois de cortar e colar o buffer ficou %q, esperava o texto de volta", got)
	}
}

// TestPastingAnEmptyClipboardSaysWhereItReads: um ctrl+v que não cola sem dizer
// de onde lê é indistinguível de um terminal que engoliu a tecla.
func TestPastingAnEmptyClipboardSaysWhereItReads(t *testing.T) {
	clipboard.Copy("")
	a := buildWired(t, "conteúdo")

	if err := a.Apply(action.Paste); err != nil {
		t.Fatalf("paste sem clipboard não pode ser erro: %v", err)
	}
	if !strings.Contains(a.Status, "vazio") {
		t.Errorf("status = %q, o leitor precisa saber que não havia nada", a.Status)
	}
	document, _ := a.Store.Active()
	if got := document.Buffer().String(); got != "conteúdo" {
		t.Errorf("a colagem vazia alterou o buffer: %q", got)
	}
}

// TestCopyingNothingDoesNotClaimSuccess fecha o lado honesto do contador: 0 bytes
// copiados não é "copiado 0 bytes".
func TestCopyingNothingDoesNotClaimSuccess(t *testing.T) {
	a := buildWired(t, "")
	if err := a.Apply(action.Copy); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if strings.Contains(a.Status, "copiado") {
		t.Errorf("status = %q, copiar nada não é copiar", a.Status)
	}
}

// TestTheKeyboardPathReachesThem fecha o circuito com o keymap: as três teclas
// existem na tabela, e é por isso que a regra de aptidão as cobra.
func TestTheKeyboardPathReachesThem(t *testing.T) {
	for _, tc := range []struct{ chord, actionID string }{
		{"ctrl+c", "copy"},
		{"ctrl+v", "paste"},
		{"ctrl+x", "cut"},
	} {
		bound, found := defaultKeys()[tc.chord]
		if !found || bound != tc.actionID {
			t.Errorf("%s aponta para %q, esperava %q", tc.chord, bound, tc.actionID)
		}
	}
}

// defaultKeys is the shipped keymap, read once.
func defaultKeys() map[string]string { return config.Default().Keys }
