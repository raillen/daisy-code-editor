package focus

import "testing"

// allVisible is the graph with every surface shown.
func allVisible(Surface) bool { return true }

// onlyEditor matches a model with no side panel open.
func onlyEditor(s Surface) bool { return s == Editor }

// TestOrderMatchesTheDocumentedGraph pins the walk order against
// docs/ui-ux/focus-graph.md. A reordering here changes where Tab goes for every
// user, so the order is asserted rather than assumed.
func TestOrderMatchesTheDocumentedGraph(t *testing.T) {
	// A ordem é a tabela de `docs/ui-ux/focus-graph.md`, e o motivo de ela existir
	// como tabela no documento é este teste: uma pressão de `Tab` a mais até a
	// árvore é o tipo de coisa que se nota no uso e não no código.
	want := []Surface{Editor, Tree, SCM, Terminal, MenuBar, Tabs, StatusBar}

	got := New().Order()
	if len(got) != len(want) {
		t.Fatalf("ordem = %v, esperado %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("posição %d = %q, esperado %q", index, got[index], want[index])
		}
	}
}

// TestTabCyclesForwardAndWraps follows the first rule of the graph: Tab walks the
// visible surfaces in order and comes back around.
func TestTabCyclesForwardAndWraps(t *testing.T) {
	graph := New()

	// Só os painéis entram no ciclo: menu, abas e status são alvo do mouse, e o `Tab`
	// que parava num rótulo era quatro pressões de distância do editor.
	walk := []Surface{Tree, SCM, Terminal, Editor}
	from := Editor
	for _, want := range walk {
		from = graph.Next(from, allVisible, false)
		if from != want {
			t.Fatalf("de %v Tab foi para %v, esperado %v", want, from, want)
		}
	}
}

func TestShiftTabCyclesBackwards(t *testing.T) {
	graph := New()

	// Ao contrário: terminal → SCM → árvore → editor. A volta é o mesmo ciclo no
	// sentido inverso, e o chrome não participa dos dois.
	walk := []Surface{Terminal, SCM, Tree, Editor}
	from := Editor
	for _, want := range walk {
		from = graph.Next(from, allVisible, true)
		if from != want {
			t.Fatalf("Shift+Tab foi para %v, esperado %v", from, want)
		}
	}
}

// TestHiddenSurfacesAreSkipped is the rule that keeps the focus off something the
// user cannot see: the next keystroke would otherwise go somewhere invisible.
func TestHiddenSurfacesAreSkipped(t *testing.T) {
	graph := New()
	visible := func(s Surface) bool { return s == Editor || s == Tree }

	if got := graph.Next(Editor, visible, false); got != Tree {
		t.Errorf("Tab foi para %v, esperado Tree", got)
	}
	if got := graph.Next(Tree, visible, false); got != Editor {
		t.Errorf("Tab de Tree foi para %v, esperado Editor", got)
	}
	if got := graph.Next(Editor, visible, true); got != Tree {
		t.Errorf("Shift+Tab foi para %v, esperado Tree", got)
	}
}

// TestFocusStaysPutWhenNothingElseIsVisible: with only the editor shown, Tab must
// not be a way to lose the cursor.
func TestFocusStaysPutWhenNothingElseIsVisible(t *testing.T) {
	graph := New()

	for _, backwards := range []bool{false, true} {
		if got := graph.Next(Editor, onlyEditor, backwards); got != Editor {
			t.Errorf("Tab/backwards=%v foi para %v, esperado Editor", backwards, got)
		}
	}
}

// TestAnUnknownSurfaceFallsBackToTheEditor: a surface that left the graph — a
// panel removed in a later release — must not leave focus dangling.
func TestAnUnknownSurfaceFallsBackToTheEditor(t *testing.T) {
	graph := New()

	if got := graph.Next(Surface("inexistente"), allVisible, false); got != Editor {
		t.Errorf("de uma superfície desconhecida foi para %v, esperado Editor", got)
	}
}

func TestHolds(t *testing.T) {
	graph := New()

	if !graph.Holds(Tree) {
		t.Error("Tree deveria pertencer ao grafo")
	}
	if graph.Holds(Surface("inexistente")) {
		t.Error("uma superfície desconhecida não deveria pertencer ao grafo")
	}
}

// TestOrderIsACopy: handing out the internal slice would let a caller reorder the
// graph from the outside, which is exactly the hidden coupling this package was
// extracted to remove.
func TestOrderIsACopy(t *testing.T) {
	graph := New()

	order := graph.Order()
	order[0] = Surface("adulterado")

	if graph.Order()[0] != Editor {
		t.Error("a ordem devolvida altera o grafo")
	}
}

// TestEveryPanelIsReachableByTab is a distinção que vale a pena fixar: o `Tab`
// alcança **painéis**, e o chrome é alvo do clique.
//
// Focar menu, abas e barra de status pelo teclado só custava atalhos no caminho
// mais usado do editor — e o custo era medido: quatro `Tab`s para a árvore, e
// outras quatro de volta.
func TestEveryPanelIsReachableByTab(t *testing.T) {
	graph := New()

	seen := map[Surface]bool{Editor: true}
	from := Editor
	for range graph.Order() {
		from = graph.Next(from, allVisible, false)
		seen[from] = true
	}

	for _, surface := range graph.Order() {
		if chrome[surface] {
			if seen[surface] {
				t.Errorf("%v não deveria entrar no ciclo do Tab", surface)
			}
			continue
		}
		if !seen[surface] {
			t.Errorf("o painel %v não é alcançável por Tab", surface)
		}
	}
}

// TestTheChromeIsStillAFocusTarget is a outra face: sair do ciclo do `Tab` **não**
// tira a superfície do grafo. Um clique na barra de abas continua focusing o
// editor, e o `Surface` continua existindo para isso.
func TestTheChromeIsStillAFocusTarget(t *testing.T) {
	graph := New()

	for _, surface := range graph.Order() {
		if !graph.Holds(surface) {
			t.Errorf("%v saiu do grafo: sair do Tab não é sair do foco", surface)
		}
	}
}
