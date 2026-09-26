package search

import (
	"os"
	"path/filepath"
	"testing"
)

// Os testes do `replace` são os quatro do oráculo, traduzidos — e a razão é
// medida, não deferência: cada um fixa uma decisão que um port descuidado perde.
//
// fixture is one directory with the named files, and the test owns it.
func fixture(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	for name, content := range files {
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

// TestReplaceInFileIsLiteralByDefault é a decisão que o port perde fácil: sem
// regex, `$100` é **texto**.
func TestReplaceInFileIsLiteralByDefault(t *testing.T) {
	root := fixture(t, map[string]string{"price.txt": "cost is PRICE USD"})

	count, err := ReplaceInFile(filepath.Join(root, "price.txt"),
		Query{Text: "PRICE", CaseSensitive: true}, "$100")
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if count != 1 {
		t.Fatalf("substituições = %d, esperava 1", count)
	}
	if got := readFile(t, root, "price.txt"); got != "cost is $100 USD" {
		t.Errorf("conteúdo = %q", got)
	}
}

// TestReplaceInProjectWritesTheSameFilesItSearches é a propriedade que liga os
// dois: a mesma caminhada, os mesmos filtros, os mesmos ignorados.
func TestReplaceInProjectWritesTheSameFilesItSearches(t *testing.T) {
	root := fixture(t, map[string]string{
		"a.txt":        "target_word in first file",
		"b.rs":         "target_word and target_word again",
		"target/c.txt": "target_word in target folder",
	})

	summary, modified, err := ReplaceInProject(root,
		Query{Text: "target_word", CaseSensitive: true}, "replacement_word")
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if summary.FilesModified != 2 || summary.ReplacementsCount != 3 || len(modified) != 2 {
		t.Errorf("resumo = %+v, modificados = %v", summary, modified)
	}
	if got := readFile(t, root, "a.txt"); got != "replacement_word in first file" {
		t.Errorf("a.txt = %q", got)
	}
	if got := readFile(t, root, "b.rs"); got != "replacement_word and replacement_word again" {
		t.Errorf("b.rs = %q", got)
	}
	if got := readFile(t, root, "target/c.txt"); got != "target_word in target folder" {
		t.Errorf("o diretório ignorado foi tocado: %q", got)
	}
}

// TestReplaceInProjectHonoursTheGlobFilter é o `Alt+G` do leitor: `*.rs` toca só
// o `.rs`, e o `.txt` continua como está.
func TestReplaceInProjectHonoursTheGlobFilter(t *testing.T) {
	root := fixture(t, map[string]string{
		"doc.txt": "token_match in txt",
		"lib.rs":  "token_match in rs",
	})

	summary, _, err := ReplaceInProject(root,
		Query{Text: "token_match", Globs: []string{"*.rs"}}, "token_replaced")
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if summary.FilesModified != 1 {
		t.Errorf("arquivos = %d, esperava 1", summary.FilesModified)
	}
	if got := readFile(t, root, "doc.txt"); got != "token_match in txt" {
		t.Errorf("o txt foi tocado: %q", got)
	}
	if got := readFile(t, root, "lib.rs"); got != "token_replaced in rs" {
		t.Errorf("o rs não foi tocado: %q", got)
	}
}

// TestReplaceRefusesAnEmptyQuery é o que impede o replace de reescrever o
// repositório: um padrão que casa com tudo não é uma busca, é um acidente.
func TestReplaceRefusesAnEmptyQuery(t *testing.T) {
	root := fixture(t, map[string]string{"a.txt": "x"})

	if _, _, err := ReplaceInProject(root, Query{Text: "  "}, "y"); err == nil {
		t.Error("um padrão vazio foi aceito para o projeto inteiro")
	}
	if _, err := ReplaceInFile(filepath.Join(root, "a.txt"), Query{Text: ""}, "y"); err == nil {
		t.Error("um padrão vazio foi aceito para um arquivo")
	}
	if got := readFile(t, root, "a.txt"); got != "x" {
		t.Errorf("o arquivo foi tocado: %q", got)
	}
}

// TestRegexReplacementExpandsGroups é o outro lado da decisão literal: **com**
// regex, `$1` expande — que é `replace_all` contra `NoExpand` do outro lado.
func TestRegexReplacementExpandsGroups(t *testing.T) {
	root := fixture(t, map[string]string{"a.txt": "um dois tres"})

	count, err := ReplaceInFile(filepath.Join(root, "a.txt"),
		Query{Text: "(\\w+) (\\w+)", UseRegex: true}, "$2-$1")
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	if count != 1 {
		t.Fatalf("substituições = %d, esperava 1", count)
	}
	if got := readFile(t, root, "a.txt"); got != "dois-um tres" {
		t.Errorf("conteúdo = %q", got)
	}
}

func readFile(t *testing.T, root, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatalf("lendo: %v", err)
	}
	return string(data)
}
