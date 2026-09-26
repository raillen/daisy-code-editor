// Package focus holds the focus graph: where input can land, in what order, and
// what outranks what.
//
// The graph is data. The reference implementation resolved focus through the
// order of `if` statements in a two-thousand-line function — the order existed,
// but nothing could assert it, so an `if` inserted at the top could silently
// steal a key from a modal. Here the order is a value, and the rules in
// docs/ui-ux/focus-graph.md are testable directly.
//
// It knows nothing about internal/app. The composition root maps a Surface onto
// the model's own focus vocabulary, which is what keeps this package reusable and
// a surface disposable.
package focus

// Surface names a place input can land.
type Surface string

// The surfaces, in the order Tab walks them.
const (
	Editor    Surface = "editor"
	MenuBar   Surface = "menubar"
	Tabs      Surface = "tabs"
	Tree      Surface = "tree"
	SCM       Surface = "scm"
	Terminal  Surface = "terminal"
	StatusBar Surface = "statusbar"
)

// declaration is the canonical order, and the single source of it.
//
// Editor first: it is the root, and it is where focus goes back to whenever the
// current target stops being visible.
//
// **Os painéis vêm antes do chrome.** A ordem anterior era editor → menu → abas →
// árvore, e o preço era concreto: **quatro `Tab`s** para alcançar a árvore de um
// projeto, ou seja, o caminho mais usado de um editor de código.
var declaration = []Surface{Editor, Tree, SCM, Terminal, MenuBar, Tabs, StatusBar}

// chrome é a faixa que **não** entra no ciclo do `Tab`: menu, abas e barra de
// status.
//
// A separação é medida, e não é gosto: com o chrome no ciclo, ir da árvore ao
// editor levava outras quatro pressões — SCM e terminal ocultos são pulados, e o
// passo seguinte é o rótulo do menu. Menu e abas são **clicáveis**, e o teclado
// chega a tudo o que eles oferecem pela paleta: um leitor de teclado não precisa
// focar um rótulo para usar a barra.
var chrome = map[Surface]bool{
	MenuBar:   true,
	Tabs:      true,
	StatusBar: true,
}

// Graph is the declared focus order.
type Graph struct {
	order []Surface
}

// New returns the graph as documented.
func New() Graph {
	return Graph{order: append([]Surface(nil), declaration...)}
}

// Order lists the surfaces in walk order.
func (g Graph) Order() []Surface {
	return append([]Surface(nil), g.order...)
}

// Holds reports whether a surface belongs to the graph.
func (g Graph) Holds(surface Surface) bool {
	for _, candidate := range g.order {
		if candidate == surface {
			return true
		}
	}
	return false
}

// Visible reports whether a surface may hold focus right now.
type Visible func(Surface) bool

// Next returns the surface focus moves to from `from`.
//
// Hidden surfaces are skipped rather than landed on: focus must never rest where
// nothing is drawn, because the next keystroke would then go somewhere invisible.
// When only the editor is visible it stays put, so Tab is never a way to lose the
// cursor.
func (g Graph) Next(from Surface, visible Visible, backwards bool) Surface {
	if g.onlyChromeVisible(visible) {
		// Sem nenhum painel visível, ficar no editor é a resposta: o `Tab` nunca é
		// uma forma de perder o cursor para um rótulo.
		return Editor
	}
	start := g.indexOf(from)
	if start < 0 {
		return g.fallback(visible)
	}

	step := 1
	if backwards {
		step = len(g.order) - 1
	}

	// One full lap: if nothing else is visible, the editor is still the answer.
	for offset := 1; offset <= len(g.order); offset++ {
		candidate := g.order[(start+offset*step)%len(g.order)]
		if chrome[candidate] {
			continue
		}
		if visible(candidate) {
			return candidate
		}
	}
	return g.fallback(visible)
}

// fallback is where focus goes when the current surface is unknown.
func (g Graph) fallback(visible Visible) Surface {
	if visible(Editor) {
		return Editor
	}
	// A model that has hidden the editor has hidden every surface; returning the
	// editor anyway is better than returning nothing, and the caller is told
	// which surface it asked about.
	return Editor
}

// indexOf finds a surface's position, or -1.
// onlyChromeVisible reports whether no panel is on screen, which is the case where
// the walk has nowhere to go.
func (g Graph) onlyChromeVisible(visible Visible) bool {
	for _, surface := range g.order {
		if !chrome[surface] && visible(surface) {
			return false
		}
	}
	return true
}

func (g Graph) indexOf(surface Surface) int {
	for index, candidate := range g.order {
		if candidate == surface {
			return index
		}
	}
	return -1
}
