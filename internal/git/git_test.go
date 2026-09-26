package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repo builds a real git repository, and skips the test where `git` is not
// installed — which is the same rule the conformance harness follows ("a case
// that does not test git must not depend on the binary being installed").
//
// Um repositório de verdade, e não um diretório com arquivos: `porcelain`,
// `diff`, `stage` e `push` precisam de um `.git` para ter o que dizer, e um teste
// que planta `.git/HEAD` à mão testaria `ParsePorcelainZ` contra uma mentira.
func repo(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("sem git neste ambiente")
	}

	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		command := exec.Command("git", args...)
		command.Dir = root
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "t@t.t")
	run("config", "user.name", "t")
	run("config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "base.txt")
	run("commit", "-q", "-m", "base")
	return root
}

// TestStageAndUnstageRoundTrip: o que o painel faz com `s` e `u`, e nada menos.
// Um arquivo rastreado e modificado que entra no índice sai dele, e em cada passo
// o estado-por-caminho diz a verdade.
func TestStageAndUnstageRoundTrip(t *testing.T) {
	root := repo(t)
	mustWrite(t, root, "mudou.txt", "mudou\n")

	if got := StatusMap(root)["mudou.txt"]; got != Untracked {
		t.Fatalf("antes do stage: %q, esperava untracked", got)
	}

	if err := Stage(root, "mudou.txt"); err != nil {
		t.Fatalf("stage: %v", err)
	}
	if got := StatusMap(root)["mudou.txt"]; got != Added {
		t.Errorf("depois do stage: %q, esperava added", got)
	}

	if err := Unstage(root, "mudou.txt"); err != nil {
		t.Fatalf("unstage: %v", err)
	}
	if got := StatusMap(root)["mudou.txt"]; got != Untracked {
		t.Errorf("depois do unstage: %q, esperava untracked", got)
	}
}

// TestUnstageOfAModifiedFileKeepsTheWork é a metade que um `unstage` descuidado
// perde: tirar do índice **não** pode desfazer o trabalho, só o índice.
func TestUnstageOfAModifiedFileKeepsTheWork(t *testing.T) {
	root := repo(t)
	mustWrite(t, root, "trabalho.txt", "v1\n")
	runIn(t, root, "add", "trabalho.txt")
	mustWrite(t, root, "trabalho.txt", "v2\n")
	if err := Stage(root, "trabalho.txt"); err != nil {
		t.Fatalf("stage: %v", err)
	}

	if err := Unstage(root, "trabalho.txt"); err != nil {
		t.Fatalf("unstage: %v", err)
	}
	if got := mustRead(t, root, "trabalho.txt"); got != "v2\n" {
		t.Errorf("o unstage desfez o trabalho: %q", got)
	}
}

// TestDiffIsTheUnifiedDiffOfTheFile é o que `d` mostra no painel: `git diff --`
// do caminho, e nada mais que isso.
func TestDiffIsTheUnifiedDiffOfTheFile(t *testing.T) {
	root := repo(t)
	mustWrite(t, root, "base.txt", "base\nmais uma linha\n")

	diff, err := Diff(root, "base.txt")
	if err != nil {
		t.Fatalf("diff: %v", err)
	}

	for _, want := range []string{"diff --git", "base.txt", "+mais uma linha"} {
		if !stringsContains(diff, want) {
			t.Errorf("o diff não tem %q:\n%s", want, diff)
		}
	}
}

// TestPullAndPushWithoutAnUpstreamSaySo é o caminho que uma `Cmd` devolve: sem
// upstream não há rede, e a mensagem do `git` é o que o leitor lê — não uma
// paráfrase nossa.
func TestPullAndPushWithoutAnUpstreamSaySo(t *testing.T) {
	root := repo(t)

	if _, err := Pull(root); err == nil {
		t.Error("pull sem upstream não falhou")
	} else if !stringsContains(strings.ToLower(err.Error()), "upstream") &&
		!stringsContains(strings.ToLower(err.Error()), "tracking") &&
		!stringsContains(strings.ToLower(err.Error()), "configured") {
		t.Errorf("a mensagem do pull não explica o que falta: %v", err)
	}

	if _, err := Push(root); err == nil {
		t.Error("push sem upstream não falhou")
	}
}

func mustWrite(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
		t.Fatalf("escrevendo: %v", err)
	}
}

func mustRead(t *testing.T, root, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatalf("lendo: %v", err)
	}
	return string(data)
}

func runIn(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func stringsContains(haystack, needle string) bool { return strings.Contains(haystack, needle) }
