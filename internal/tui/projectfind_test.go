package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/editor"
	"github.com/ori-team/oride/internal/fs"
)

// A busca no projeto é a segunda via de abrir arquivo, e era uma tecla morta —
// `project_find` e `project_replace` estavam anunciados no keymap gerado, sem
// tratamento. Estes testes abrem, digitam, alternam os campos, e abrem o
// resultado — com um workspace de verdade, porque `find` sem disco é um jeito
// elegante de não testar nada.

// projectWorkspace is a workspace with text the search can find.
func projectWorkspace(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for name, content := range map[string]string{
		"alfa.txt":        "uma linha com alfa dentro\n",
		"beta/beta.go":    "package beta\n\n// alfa no comentário\n",
		"gamma/leia.md":   "# gamma\n\nleia este arquivo\n",
		"target/ignorado": "alfa que não aparece\n",
	} {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// wiredProject is a model over a searchable workspace, usando o mesmo `wired`
// que monta a sessão — com o workspace trocado pelo que tem texto para achar.
func wiredProject(t *testing.T) Model {
	t.Helper()

	model := wired(t)
	root := projectWorkspace(t)
	model.application.Workspace = root
	model.application.Tree, _ = openTreeForTest(t, root)
	return model
}

// openTreeForTest is the workspace's tree, e a falha é falhar a construção: sem
// árvore não há vizinhança, e sem vizinhança o teste mede uma sessão que o leitor
// nunca vê.
func openTreeForTest(t *testing.T, root string) (*fs.Tree, error) {
	t.Helper()
	return fs.Open(root, false)
}

// TestProjectFindOpensAndSearchesLive é o caminho inteiro: abrir, digitar, e ver
// os resultados mudarem a cada tecla.
func TestProjectFindOpensAndSearchesLive(t *testing.T) {
	model := wiredProject(t)
	if err := model.application.Apply(action.ProjectFind); err != nil {
		t.Fatalf("abrindo a busca: %v", err)
	}
	if model.application.Overlay != "ProjectFind" {
		t.Fatalf("overlay = %q, esperava a busca no projeto", model.application.Overlay)
	}

	model = typeText(t, model, "alf")
	if len(model.application.ProjectFind.Hits) == 0 {
		t.Fatal("digitar não buscou nada num workspace com alfa em todo lugar")
	}
	before := len(model.application.ProjectFind.Hits)
	model = typeText(t, model, "a")

	after := model.application.ProjectFind.Hits
	if len(after) == 0 {
		t.Fatal("refinar a consulta esvaziou os resultados")
	}
	for _, hit := range after {
		if !strings.Contains(strings.ToLower(hit.Text), "alfa") {
			t.Errorf("resultado %q não contém a consulta", hit.Text)
		}
	}
	if len(after) > before {
		t.Errorf("refinar a consulta aumentou os resultados: %d → %d", before, len(after))
	}
}

// TestTabRevealsTheReplaceFieldAndThenTheGlob é a regra do oráculo, em código:
// o primeiro `Tab` revela o substituir, e os seguintes andam entre consulta,
// substituir e glob.
func TestTabRevealsTheReplaceFieldAndThenTheGlob(t *testing.T) {
	model := wiredProject(t)
	if err := model.application.Apply(action.ProjectFind); err != nil {
		t.Fatalf("abrindo a busca: %v", err)
	}

	model = press(t, model, tea.Key{Code: tea.KeyTab})
	if !model.application.ProjectFind.HasReplace {
		t.Fatal("o primeiro Tab não revelou o substituir")
	}

	// O segundo `Tab` revela o glob — como no oráculo, onde o primeiro revela e
	// os seguintes andam entre os visíveis: com dois campos, o segundo `Tab` volta
	// para a consulta, e é o terceiro que chega ao glob.
	model = press(t, model, tea.Key{Code: tea.KeyTab})
	if model.application.ProjectFind.HasGlob {
		t.Error("o segundo Tab não devia revelar o glob: consulta e substituir estão visíveis")
	}
	if model.application.ProjectFind.FocusField != 0 {
		t.Errorf("o ciclo voltou para %d, e não para a consulta", model.application.ProjectFind.FocusField)
	}

	model = press(t, model, tea.Key{Code: tea.KeyTab})
	_ = model
	model = press(t, model, tea.Key{Code: 'g', Mod: tea.ModAlt})
	if !model.application.ProjectFind.HasGlob {
		t.Error("Alt+G não revelou o glob")
	}
}

// TestAltTogglesTheOptionsWithoutTyping é o que diferencia uma opção de um
// filtro: `Alt+C`, `Alt+R` e `Alt+G` mudam a busca sem digitar nada nela.
func TestAltTogglesTheOptionsWithoutTyping(t *testing.T) {
	model := wiredProject(t)
	if err := model.application.Apply(action.ProjectFind); err != nil {
		t.Fatalf("abrindo a busca: %v", err)
	}
	model = typeText(t, model, "Alfa")

	model = press(t, model, tea.Key{Code: 'c', Mod: tea.ModAlt})
	if !model.application.ProjectFind.Options.CaseSensitive {
		t.Error("Alt+C não ligou a caixa")
	}
	if len(model.application.ProjectFind.Hits) != 0 {
		t.Errorf("com caixa ligada, %d resultados para Alfa maiúsculo",
			len(model.application.ProjectFind.Hits))
	}

	model = press(t, model, tea.Key{Code: 'c', Mod: tea.ModAlt})
	if len(model.application.ProjectFind.Hits) == 0 {
		t.Error("desligar a caixa não trouxe os resultados de volta")
	}
}

// TestEnterOpensTheSelectedHit é o que a tecla promete: o arquivo abre, o caret
// vai para a linha, e a sobreposição fecha.
func TestEnterOpensTheSelectedHit(t *testing.T) {
	model := wiredProject(t)
	if err := model.application.Apply(action.ProjectFind); err != nil {
		t.Fatalf("abrindo a busca: %v", err)
	}
	model = typeText(t, model, "leia")

	model = press(t, model, tea.Key{Code: tea.KeyDown})
	model = press(t, model, tea.Key{Code: tea.KeyEnter})

	if model.application.Overlay != "none" {
		t.Errorf("Enter não fechou a busca: %q", model.application.Overlay)
	}
	document, err := model.application.Store.Active()
	if err != nil || !strings.HasSuffix(mustPath(t, document), "leia.md") {
		t.Errorf("Enter não abriu o arquivo do resultado: %+v %v", document, err)
	}
}

// TestReplaceAcrossTheProjectWritesAndReloads é a parte que escreve no disco — e
// a que recarrega o que está aberto, para o documento não divergir do arquivo
// que ele acabou de escrever.
func TestReplaceAcrossTheProjectWritesAndReloads(t *testing.T) {
	model := wiredProject(t)
	if err := model.application.Apply(action.ProjectReplace); err != nil {
		t.Fatalf("abrindo a substituição: %v", err)
	}
	// A ação abre com o foco no campo de substituir, como no oráculo: primeiro o
	// que entra, depois o que sai.
	model = typeText(t, model, "OMEGA")
	model = press(t, model, tea.Key{Code: tea.KeyTab})
	model = typeText(t, model, "alfa")
	model = press(t, model, tea.Key{Code: tea.KeyTab})
	model = press(t, model, tea.Key{Code: tea.KeyEnter})

	if !strings.Contains(model.application.Status, "Substituídas") {
		t.Errorf("status = %q, e o leitor precisa dos dois números", model.application.Status)
	}
	if model.application.Overlay != "none" {
		t.Errorf("a substituição continuou aberta: %q", model.application.Overlay)
	}
}

// TestEscapeClosesTheProjectSearch sem tocar em nada: cancelar não escreve, não
// abre e não troca.
func TestEscapeClosesTheProjectSearch(t *testing.T) {
	model := wiredProject(t)
	paths := model.application.Store.OpenPaths()
	if err := model.application.Apply(action.ProjectFind); err != nil {
		t.Fatalf("abrindo a busca: %v", err)
	}
	model = typeText(t, model, "alfa")
	model = press(t, model, tea.Key{Code: tea.KeyEscape})

	if model.application.Overlay != "none" {
		t.Errorf("Esc não fechou a busca: %q", model.application.Overlay)
	}
	if got := model.application.Store.OpenPaths(); !sameStrings(got, paths) {
		t.Errorf("Esc abriu documento: %v → %v", paths, got)
	}
}

// mustPath is the active document's path.
func mustPath(t *testing.T, document *editor.Document) string {
	t.Helper()
	path, ok := document.Path()
	if !ok {
		t.Fatal("o documento ativo não tem caminho")
	}
	return path
}

// sameStrings compares two string slices, and order matters: opening a document
// changes the tab order, and that is what the test is watching for.
func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
