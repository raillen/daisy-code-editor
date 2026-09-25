// Package clipboard is the editor's copy buffer, and the escape that hands it to
// the terminal.
//
// Three layers, in the order they are tried, and the order is the design:
//
//  1. **Internal buffer** — always there, always authoritative for *this*
//     session. A terminal program cannot ask the user what is in the clipboard
//     without asking the terminal to tell it, and that answer is asynchronous.
//  2. **OSC 52** — the write side is free and synchronous: a terminal that accepts
//     it puts the text on the *system* clipboard, so copy survives the exit and
//     works over SSH, where no X11 clipboard exists.
//  3. **Nothing else.** `atotto/clipboard` would shell out to `xclip`/`wl-copy` per
//     copy, and the plan lists it as a non-candidate; a TUI that spawns a process
//     per keystroke is a TUI that stutters.
//
// The write side returns a string, not a side effect: this package is pure, and the
// terminal is the composition root's business. The same split keeps the escape
// sequence testable without a terminal.
package clipboard

import (
	"encoding/base64"
	"strings"
	"sync"
)

// internal is the session's own copy buffer.
//
// A package-level value with a mutex, and the alternative — threading a buffer
// through the model, the store, every document and the TUI — is four constructors
// changed to save one variable. The mutex is the honest part: two goroutines
// copiando não colidem, e a corrida aqui seria um texto truncado sem aviso.
var internal = struct {
	sync.RWMutex
	text string
}{}

// Copy stores the text in the session buffer and returns the OSC 52 sequence that
// also offers it to the terminal.
//
// A sequência volta mesmo com o buffer vazio: um `\e]52;c;a` vazio é inofensivo
// e devolve o terminal ao estado normal depois, o que importa numa TUI que acabou
// de desenhar.
func Copy(text string) string {
	internal.Lock()
	internal.text = text
	internal.Unlock()

	return Sequence(text)
}

// Paste returns the text this session copied.
//
// A sessão, e não o sistema: perguntar ao terminal o que há no clipboard exige uma
// resposta assíncrona que o runtime ainda não tem, e um editor que finge ter
// clipboard do sistema e cola o buffer da última cópia é pior do que um que
// declara de onde lê. Quando a sequência de escrita falhou no terminal, isto ainda
// funciona — que é o motivo de a ordem ser buffer primeiro.
func Paste() string {
	internal.RLock()
	defer internal.RUnlock()
	return internal.text
}

// HasText reports whether there is anything to paste.
//
// Uma consulta e não um Paste seguido de comparação: `Paste` devolve uma cópia da
// string, e comparar duas strings para saber se há conteúdo é trabalho de quem tem
// medo de `len("") == 0`.
func HasText() bool {
	internal.RLock()
	defer internal.RUnlock()
	return internal.text != ""
}

// Sequence encodes text as an OSC 52 clipboard write.
//
// # <esc> ] 52 ; c ; <base64> <bel>
//
// `c` é o alvo padrão (a seleção do sistema), `a` seria a seleção primária. O
// BEL e não o ST porque é o que os terminais aceitam sem negociação, e um
// terminais que prefere ST também aceita BEL.
func Sequence(text string) string {
	if text == "" {
		return "\x1b]52;c;a\x07"
	}
	return "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
}

// Decode is the inverse of Sequence, and it exists to be tested against it.
//
// Uma função de decodificar em produção seria código morto; em teste, é o que
// prova que a sequência que o editor manda é a sequência que o terminal entende.
func Decode(sequence string) (string, bool) {
	const prefix = "\x1b]52;c;"
	if !strings.HasPrefix(sequence, prefix) {
		return "", false
	}
	payload := strings.TrimSuffix(strings.TrimPrefix(sequence, prefix), "\x07")
	if payload == "" {
		return "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", false
	}
	return string(decoded), true
}
