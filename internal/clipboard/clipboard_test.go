package clipboard

import (
	"strings"
	"testing"
)

// TestCopyThenPasteRoundTripsThroughTheTerminalSequence prova o caminho inteiro:
// o que o editor copiou é o que o terminal receberia, decodificado de volta.
//
// A sequência é comparada **com o que o terminal entende**, e não com a string que
// o editor guardou: um editor que copia "olá" e manda outra coisa ao terminal
// funciona nos testes e falha no uso, que é a classe de bug que a regra de
// "teste a saída renderizada" descreve.
func TestCopyThenPasteRoundTripsThroughTheTerminalSequence(t *testing.T) {
	const text = "uma linha\ncom acento: ação"

	sequence := Copy(text)
	decoded, ok := Decode(sequence)
	if !ok {
		t.Fatalf("a sequência %q não decodifica", sequence)
	}
	if decoded != text {
		t.Errorf("o terminal receberia %q, esperava %q", decoded, text)
	}
	if got := Paste(); got != text {
		t.Errorf("Paste devolveu %q, esperava %q", got, text)
	}
}

// TestTheSequenceIsWellFormed fixa o formato, porque o terminal é estrito e a
// sequência errada não dá erro: ela simplesmente não cola.
func TestTheSequenceIsWellFormed(t *testing.T) {
	sequence := Sequence("x")

	const prefix = "\x1b]52;c;"
	if !strings.HasPrefix(sequence, prefix) {
		t.Fatalf("a sequência não começa com OSC 52 c: %q", sequence)
	}
	if sequence[len(sequence)-1] != '\a' {
		t.Errorf("a sequência não termina com BEL: %q", sequence)
	}
}

// TestAnEmptyCopyIsAWellFormedSequence: copiar nada ainda tem que devolver o
// terminal ao estado normal, e não deixar a sequência pela metade.
func TestAnEmptyCopyIsAWellFormedSequence(t *testing.T) {
	sequence := Sequence("")

	if _, ok := Decode(sequence); ok {
		t.Error("uma sequência vazia decodificou para texto")
	}
	if sequence != "\x1b]52;c;a\x07" {
		t.Errorf("a sequência vazia é %q", sequence)
	}
}

// TestTheSequenceIsSafeForTextWithControlCharacters é o caso que quebra a
// codificação ingênua: uma sequência de escape dentro do texto copiado fecharia
// o OSC 52 ecommands o terminal.
func TestTheSequenceIsSafeForTextWithControlCharacters(t *testing.T) {
	const text = "antes\x1b]52;c;trap\x07depois"

	decoded, ok := Decode(Sequence(text))
	if !ok {
		t.Fatal("a sequência com escape não decodificou")
	}
	if decoded != text {
		t.Errorf("o texto voltou diferente: %q", decoded)
	}
	if Sequence(text) == Sequence("antes") {
		t.Error("um texto com escape produced a mesma sequência de um texto sem")
	}
}

// TestDecodeRefusesWhatItDidNotProduce evita que a função de teste vire parser de
// entrada do produto.
func TestDecodeRefusesWhatItDidNotProduce(t *testing.T) {
	for _, sequence := range []string{"", "texto solto", "\x1b]52;x;a\x07", "\x1b]52;c;!!!\x07", "\x1b]52;c;"} {
		if _, ok := Decode(sequence); ok {
			t.Errorf("Decode(%q) aceitou o que não é uma sequência nossa", sequence)
		}
	}
}

// TestHasTextFollowsCopy e TestCopyOverwrites provam que a segunda cópia substitui
// a primeira — o defeito clássico de um buffer que só acumula.
func TestHasTextFollowsCopy(t *testing.T) {
	Copy("primeiro")
	if !HasText() {
		t.Fatal("depois de copiar, HasText é falso")
	}

	Copy("segundo")
	if got := Paste(); got != "segundo" {
		t.Errorf("a segunda cópia não substituiu a primeira: %q", got)
	}

	Copy("")
	if HasText() {
		t.Error("copiar vazio esvaziou o buffer, e HasText continua dizendo que há texto")
	}
}
