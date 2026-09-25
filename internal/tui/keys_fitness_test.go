package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/keymap"
)

// namedKeys is the runtime's name for every **named** key the keymap vocabulary
// knows.
//
// Uma tabela, e não uma introspecção: a propriedade que este arquivo mede é "a
// tecla que o terminal manda é uma tecla que a tabela nomeia", e prová-la exige
// dizer qual tecla do runtime produz cada nome. Os nomes que o terminal **não**
// produz (`insert` é uma tecla real e não tem token) são justamente o que o
// teste abaixo accuses.
var namedKeys = map[string]rune{
	"esc":       tea.KeyEscape,
	"enter":     tea.KeyEnter,
	"backspace": tea.KeyBackspace,
	"delete":    tea.KeyDelete,
	"tab":       tea.KeyTab,
	"left":      tea.KeyLeft,
	"right":     tea.KeyRight,
	"up":        tea.KeyUp,
	"down":      tea.KeyDown,
	"home":      tea.KeyHome,
	"end":       tea.KeyEnd,
	"pageup":    tea.KeyPgUp,
	"pagedown":  tea.KeyPgDown,
	"space":     tea.KeySpace,
	"f1":        tea.KeyF1,
	"f2":        tea.KeyF2,
	"f3":        tea.KeyF3,
	"f4":        tea.KeyF4,
	"f5":        tea.KeyF5,
	"f6":        tea.KeyF6,
	"f7":        tea.KeyF7,
	"f8":        tea.KeyF8,
	"f9":        tea.KeyF9,
	"f10":       tea.KeyF10,
	"f11":       tea.KeyF11,
	"f12":       tea.KeyF12,
}

// runtimeKeyFor builds the keystroke the terminal would deliver for a canonical
// chord.
//
// A síntese é o ponto: uma binding nova não ganha uma linha nesta tabela — o
// teste monta a tecla a partir do próprio acorde. Só os códigos nomeados precisam
// ser escritos, e são poucos porque o runtime nomeia poucas teclas.
func runtimeKeyFor(chord string) (tea.Key, bool) {
	parts := strings.Split(strings.ToLower(chord), "+")
	code := parts[len(parts)-1]

	key := tea.Key{}
	for _, part := range parts[:len(parts)-1] {
		switch part {
		case "ctrl":
			key.Mod |= tea.ModCtrl
		case "alt":
			key.Mod |= tea.ModAlt
		case "shift":
			key.Mod |= tea.ModShift
		default:
			return tea.Key{}, false
		}
	}

	if named, known := namedKeys[code]; known {
		key.Code = named
		return key, true
	}

	runes := []rune(code)
	if len(runes) != 1 {
		// Código com mais de um caractere que não é nomeado não é tecla: é um
		// acorde malformado, e fingir que é mediria o teste.
		return tea.Key{}, false
	}
	key.Code = runes[0]
	return key, true
}

// TestEveryAdvertisedKeyIsAKeyTheTerminalSends fecha o circuito binding ↔ runtime.
//
// A tabela de teclas é gerada para `docs/ui-ux/keymap.md` e impressa na ajuda. Uma
// binding cujo nome o terminal nunca produz é uma **promessa falsa**: aparece na
// documentação e não é alcançável por tecla nenhuma. Foi exatamente o que
// aconteceu com uma correção anterior, que trocou o canônico de `pagedown` para
// `pgdown` e quebrou a paridade do dump sem melhorar tecla nenhuma — a tradução
// já morava em `keymap.Parse`, e a segunda tradução só criou uma segunda
// verdade.
//
// O teste vai pelo caminho do produto: sintetiza a tecla, tira a grafia que o
// runtime entregaria e pergunta ao keymap a que ação ela resolve.
func TestEveryAdvertisedKeyIsAKeyTheTerminalSends(t *testing.T) {
	keys, err := keymap.FromBindings(config.Default().Keys)
	if err != nil {
		t.Fatalf("keymap: %v", err)
	}

	for _, binding := range keys.Bindings() {
		chord := binding.Chord.String()
		key, producible := runtimeKeyFor(chord)
		if !producible {
			t.Errorf("a binding %q não corresponde a nenhuma tecla do runtime", chord)
			continue
		}

		delivered := key.Keystroke()
		parsed, parseErr := keymap.Parse(delivered)
		if parseErr != nil {
			t.Errorf("o runtime entregaria %q para %q, e o keymap não entende: %v",
				delivered, chord, parseErr)
			continue
		}
		resolved, found := keys.Resolve(parsed)
		if !found {
			t.Errorf("a tecla %q (entregue como %q) não resolve binding nenhuma; %q aponta para %q",
				chord, delivered, chord, binding.Action)
			continue
		}
		if resolved != binding.Action {
			t.Errorf("%q resolve para %q, e a tabela diz %q", chord, resolved, binding.Action)
		}
	}
}

// TestTheRuntimeSpeaksInAliasesForSomeKeys é a documentação da costura, em teste.
//
// PageDown chega como `pgdown` e a tabela diz `pagedown`; a tradução está nos
// apelidos de `keymap.Parse`. Se alguém "limpar" os apelidos achando que são
// ruído, estas quatro bindings morrem — e o teste anterior continuaria verde,
// porque ele também passa por `Parse`. Por isso a afirmação é sobre os apelidos.
func TestTheRuntimeSpeaksInAliasesForSomeKeys(t *testing.T) {
	for runtimeSpelling, canonical := range map[string]string{
		"pgup":      "pageup",
		"pgdown":    "pagedown",
		"pgdn":      "pagedown",
		"esc":       "esc",
		"escape":    "esc",
		"return":    "enter",
		"bs":        "backspace",
		"del":       "delete",
		"grave":     "`",
		"backslash": "\\",
	} {
		parsed, err := keymap.Parse(runtimeSpelling)
		if err != nil {
			t.Errorf("Parse(%q): %v", runtimeSpelling, err)
			continue
		}
		if parsed.String() != canonical {
			t.Errorf("%q normaliza para %q, esperava %q", runtimeSpelling, parsed, canonical)
		}
	}
}

// TestAnUnknownKeyNameIsRefusedNotGuessed fecha a outra ponta: um nome que o
// keymap não conhece tem de ser recusado, porque um palpite vira uma binding que
// aponta para outra tecla.
func TestAnUnknownKeyNameIsRefusedNotGuessed(t *testing.T) {
	for _, name := range []string{"insert", "pageupp", "f13", "wibble"} {
		if _, err := keymap.Parse(name); err == nil {
			t.Errorf("Parse(%q) aceitou um nome que não é tecla do keymap", name)
		}
	}
}
