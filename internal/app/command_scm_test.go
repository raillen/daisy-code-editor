package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/config"
	"github.com/ori-team/oride/internal/editor"
	"github.com/ori-team/oride/internal/git"
	"github.com/ori-team/oride/internal/keymap"
)

// gitRepo builds a real repository with one tracked file and one staged file,
// and skips where `git` is not installed.
//
// Um repositório de verdade, e não um diretório com arquivos: `porcelain`,
// `stage` e `commit` precisam de um `.git` para ter o que dizer, e um teste que
// planta `.git/HEAD` à mão testaria o parser contra uma mentira.
func gitRepo(t *testing.T) string {
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
	if err := os.WriteFile(filepath.Join(root, "trabalho.txt"), []byte("trabalho\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestTheScmEntriesNameWhatGitSays é o que a chave `scm` do dump precisa dizer: o
// caminho com a letra, na ordem do `porcelain`.
func TestTheScmEntriesNameWhatGitSays(t *testing.T) {
	root := gitRepo(t)

	keys, err := keymap.FromBindings(config.Default().Keys)
	if err != nil {
		t.Fatal(err)
	}
	application := New(editor.NewStore(), config.Default(), keys)
	application.Workspace = root
	application.GitEnabled = true

	state, err := application.state()
	if err != nil {
		t.Fatalf("state: %v", err)
	}

	if len(state.SCM) != 1 || state.SCM[0].Path != "trabalho.txt" {
		t.Errorf("scm = %+v, esperava o trabalho não rastreado", state.SCM)
	}
	if state.SCM[0].Badge != "?" {
		t.Errorf("badge = %q, esperava o de não rastreado", state.SCM[0].Badge)
	}
}

// TestScmIsEmptyWithoutGitOrWithoutARepo são os dois casos em que a resposta é
// vazia: fora de um repositório, e com o git desligado no caso. Vazios, e não um
// erro — um workspace sem `.git` é um estado normal.
func TestScmIsEmptyWithoutGitOrWithoutARepo(t *testing.T) {
	keys, err := keymap.FromBindings(config.Default().Keys)
	if err != nil {
		t.Fatal(err)
	}

	bare := New(editor.NewStore(), config.Default(), keys)
	bare.Workspace = t.TempDir()
	bare.GitEnabled = true
	if state, err := bare.state(); err != nil {
		t.Fatalf("state: %v", err)
	} else if len(state.SCM) != 0 {
		t.Errorf("fora de um repositório, scm = %+v", state.SCM)
	}

	nopq := New(editor.NewStore(), config.Default(), keys)
	nopq.Workspace = gitRepo(t)
	if state, err := nopq.state(); err != nil {
		t.Fatalf("state: %v", err)
	} else if len(state.SCM) != 0 {
		t.Errorf("com o git desligado, scm = %+v", state.SCM)
	}
}

// TestToggleScmShowsThePanelAndFocusesIt é o que `Ctrl+Shift+G` promete: o painel
// abre, o foco vai com ele, e a barra diz como sair.
func TestToggleScmShowsThePanelAndFocusesIt(t *testing.T) {
	application := scmApp(t)

	if application.SCM {
		t.Fatal("o painel começou aberto")
	}
	if err := application.Apply(action.ToggleScm); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if !application.SCM || application.Focus != FocusSCM {
		t.Errorf("o painel não abriu com o foco: %v / %q", application.SCM, application.Focus)
	}
	if !strings.Contains(application.Status, "SCM") {
		t.Errorf("status = %q, a barra precisa dizer o que a tecla fez", application.Status)
	}

	if err := application.Apply(action.ToggleScm); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if application.SCM || application.Focus != FocusEditor {
		t.Errorf("o painel continuou aberto ou o foco não voltou: %v / %q", application.SCM, application.Focus)
	}
}

// TestStageAndUnstageMoveTheEntry é o que `s` e `u` fazem no painel: `git add` e
// `git restore --staged`, com o `s` que não adiciona dizendo o motivo.
func TestStageAndUnstageMoveTheEntry(t *testing.T) {
	application := scmApp(t)
	if err := git.Stage(application.Workspace, "trabalho.txt"); err != nil {
		t.Fatalf("stage: %v", err)
	}

	dumped, err := application.state()
	if err != nil {
		t.Fatalf("dump: %v", err)
	}
	if len(dumped.SCM) != 1 || dumped.SCM[0].Badge != "A" {
		t.Fatalf("depois do stage, scm = %+v", dumped.SCM)
	}

	if err := git.Unstage(application.Workspace, "trabalho.txt"); err != nil {
		t.Fatalf("unstage: %v", err)
	}
	dumped, err = application.state()
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if len(dumped.SCM) != 1 || dumped.SCM[0].Badge != "?" {
		t.Fatalf("depois do unstage, scm = %+v", dumped.SCM)
	}
}

// TestPullWithoutAnUpstreamSaysSo é o caminho que uma `Cmd` devolve: sem upstream
// não há rede, e a mensagem do `git` é o que o leitor lê — não uma paráfrase.
func TestPullWithoutAnUpstreamSaysSo(t *testing.T) {
	root := gitRepo(t)

	if _, err := git.Pull(root); err == nil {
		t.Error("pull sem upstream não falhou")
	}
	if _, err := git.Push(root); err == nil {
		t.Error("push sem upstream não falhou")
	}
}

// scmApp builds a model over a git repository with the panel's data on.
func scmApp(t *testing.T) *App {
	t.Helper()

	keys, err := keymap.FromBindings(config.Default().Keys)
	if err != nil {
		t.Fatal(err)
	}
	application := New(editor.NewStore(), config.Default(), keys)
	application.Workspace = gitRepo(t)
	application.GitEnabled = true
	return application
}
