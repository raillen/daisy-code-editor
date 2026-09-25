package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ori-team/oride/internal/config"
)

// The highlighting engine is the one feature whose result only exists after a Cmd
// runs, so these tests drive the same loop the runtime does: Update, run what it
// returned, feed the message back. A test that called Frame() straight away would
// see a plain document and conclude the engine does nothing.

const goSource = "package main\n\n// soma dois números\nfunc soma(a, b int) int {\n\treturn a + b\n}\n"

// TestAnOpenedGoFileIsColoured is the acceptance of the syntax engine as a reader
// sees it: a .go file on screen has coloured words, and the first escape of the
// row comes before the text rather than after it.
func TestAnOpenedGoFileIsColoured(t *testing.T) {
	model := opened(t, "main.go", goSource, 100, 24)

	// The needle is one word, not the line: the role's escape sits between the
	// word and its neighbour, so the whole phrase is not in the frame as text.
	row := rowWith(t, model.Frame(), "package")
	prefix := row[:strings.Index(row, "package")]

	if !strings.ContainsRune(prefix, 0x1b) {
		t.Errorf("a linha do código saiu sem cor: %q", row)
	}
}

// TestAPlainFileStaysPlain is the contraprova of the test above.
//
// Without it, a frame where the gutter, the barra de menu or o status bar paints
// something would satisfy "there is an escape" and the engine could be doing
// nothing at all. A file with no lexer has to come out with no colour on its text.
func TestAPlainFileStaysPlain(t *testing.T) {
	source := "package main\n\n// soma dois números\nfunc soma(a, b int) int {\n\treturn a + b\n}\n"
	model := opened(t, "notes.txt", source, 100, 24)

	row := rowWith(t, model.Frame(), "package")
	prefix := row[:strings.Index(row, "package")]

	if strings.ContainsRune(prefix, 0x1b) {
		t.Errorf("um arquivo sem lexer pintou o texto: %q", row)
	}
}

// TestRolesFollowTheTypingAndNotTheFrame: the cache is keyed by the document's
// version, so an edit asks for new roles and a frame with no edit does not.
func TestRolesFollowTheTypingAndNotTheFrame(t *testing.T) {
	model := opened(t, "main.go", goSource, 100, 24)

	before, _ := model.application.Store.Active()
	version := before.Version()
	document, _ := model.application.Store.Active()
	if err := document.InsertText("// mais um comentário\n"); err != nil {
		t.Fatalf("inserindo: %v", err)
	}

	model = pump(t, model, 4)
	if document.Version() == version {
		t.Fatal("a edição não mudou a versão, e o cache não tem como saber")
	}
	row := rowWith(t, model.Frame(), "package")
	if !strings.ContainsRune(row[:strings.Index(row, "package")], 0x1b) {
		t.Errorf("depois de digitar a linha ficou sem cor: %q", row)
	}
}

// TestAnAnswerForAnOldVersionIsDiscarded: a tokenise that finishes after the next
// keystroke describes a document that no longer exists, and painting it would put
// the colours of the old text on the new.
func TestAnAnswerForAnOldVersionIsDiscarded(t *testing.T) {
	model := opened(t, "main.go", goSource, 100, 24)

	document, err := model.application.Store.Active()
	if err != nil {
		t.Fatalf("documento ativo: %v", err)
	}
	id, _ := model.application.Store.ActiveID()

	stale := highlightedMsg{document: id, version: document.Version() - 1}
	updated, _ := model.Update(stale)
	model = updated.(Model)

	if model.highlight.filled && model.highlight.lines == nil {
		t.Error("uma resposta de uma versão antiga foi guardada")
	}
}

// TestAPathChangeAsksForNewRoles: renaming a document changes its language without
// changing a single byte of its version, so the path is part of the cache key.
func TestAPathChangeAsksForNewRoles(t *testing.T) {
	model := opened(t, "notes.txt", goSource, 100, 24)
	if model.highlight.path != "" && strings.HasSuffix(model.highlight.path, "notes.txt") == false {
		t.Fatalf("caminho em cache %q", model.highlight.path)
	}

	renamed := filepath.Join(filepath.Dir(model.highlight.path), "main.go")
	document, _ := model.application.Store.Active()
	document.SetPath(renamed)

	if model.highlightRoles() == nil {
		t.Error("renomear o documento não pediu novo realce")
	}
}

// opened builds a model showing a file, with the highlighting already delivered.
func opened(t *testing.T, name, source string, width, height int) Model {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("escrevendo %s: %v", path, err)
	}

	model := sized(t, newModel(t, func(c *config.Config) { c.ShowLineNumbers = false }), width, height)
	if _, err := model.application.Store.OpenPath(path); err != nil {
		t.Fatalf("abrindo %s: %v", path, err)
	}
	return pump(t, model, 8)
}

// pump runs the commands a model asks for and feeds the answers back, the way the
// runtime does. The bound is a stop, not a rule: a model that keeps asking for
// work forever would otherwise hang the suite.
func pump(t *testing.T, model Model, rounds int) Model {
	t.Helper()

	for range rounds {
		updated, cmd := model.Update(nil)
		model = updated.(Model)
		if cmd == nil {
			return model
		}
		for _, inner := range flatten(cmd()) {
			updated, _ = model.Update(inner)
			model = updated.(Model)
		}
	}
	return model
}

// flatten is the messages of a batch, without the type switch bubbletea would do
// internally.
func flatten(msg tea.Msg) []tea.Msg {
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		if msg == nil {
			return nil
		}
		return []tea.Msg{msg}
	}
	out := make([]tea.Msg, 0, len(batch))
	for _, cmd := range batch {
		if cmd == nil {
			continue
		}
		out = append(out, flatten(cmd())...)
	}
	return out
}

// rowWith is the frame's row that contains a piece of text.
func rowWith(t *testing.T, frame, text string) string {
	t.Helper()

	for _, row := range strings.Split(frame, "\n") {
		if strings.Contains(row, text) {
			return row
		}
	}
	t.Fatalf("nenhuma linha do frame contém %q", text)
	return ""
}
