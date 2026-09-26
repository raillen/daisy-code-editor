package app

import (
	"testing"

	"github.com/ori-team/oride/internal/action"
)

// As teclas de visão mudam o que está na tela sem tocar num documento, e por
// isso funcionam com o store vazio. O que segue prova que elas **mudam estado**
// — um handler que devolve `nil` sem efeito satisfaz a regra de aptidão e
// engana o leitor.

// TestToggleSoftWrapFlipsAndSaysIt é a tecla que o leitor aperta para ler uma
// linha longa: liga, desliga, e diz em qual está — nas palavras do oráculo,
// porque a barra de status é lida, não decorada.
func TestToggleSoftWrapFlipsAndSaysIt(t *testing.T) {
	a := buildWired(t, "uma linha longa o bastante para quebrar\n")

	if !a.Config.SoftWrap {
		t.Fatal("o wrap deveria começar ligado (B19)")
	}
	if err := a.Apply(action.ToggleSoftWrap); err != nil {
		t.Fatalf("toggle soft wrap: %v", err)
	}
	if a.Config.SoftWrap {
		t.Error("a tecla não desligou o wrap")
	}
	if a.Status != "soft wrap: off" {
		t.Errorf("status = %q, esperava as palavras do oráculo", a.Status)
	}

	if err := a.Apply(action.ToggleSoftWrap); err != nil {
		t.Fatalf("toggle soft wrap: %v", err)
	}
	if !a.Config.SoftWrap {
		t.Error("a segunda vez a tecla não religou o wrap")
	}
	if a.Status != "soft wrap: on" {
		t.Errorf("status = %q, esperava as palavras do oráculo", a.Status)
	}
}
