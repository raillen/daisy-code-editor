package app

import (
	"unicode"

	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/editor"
	"github.com/ori-team/oride/internal/fs"
)

// PromptState is the tree's create prompt: what to make and the name so far.
//
// Os mesmos campos que o oráculo dumpha — `kind` e `buffer`. A edição é no fim
// (anexar/apagar), como no oráculo, então não há cursor para guardar: o que o
// leitor digitou é o que está no buffer, e o que está no buffer é o que a tela
// mostra.
type PromptState struct {
	Kind   string
	Buffer string
}

// The prompt kinds, in the oracle's vocabulary: the dump compares them.
const (
	PromptNewFile = "NewFile"
	PromptNewDir  = "NewDir"

	// OverlayPrompt is the create prompt, named like the oracle's `Prompt`: the
	// dump compares the string, and an invented name would be a silent
	// divergence.
	OverlayPrompt = "Prompt"
)

// promptCommands open the tree's create prompt, for a file or a directory.
func promptCommands() map[action.Action]Handler {
	return map[action.Action]Handler{
		action.TreeNewFile: modelHandler(func(a *App, _ *editor.Document) error {
			return a.OpenPrompt(PromptNewFile)
		}),
		action.TreeNewDir: modelHandler(func(a *App, _ *editor.Document) error {
			return a.OpenPrompt(PromptNewDir)
		}),
	}
}

// OpenPrompt shows the tree and asks for a name: quem cria quer ver onde cria,
// e um atalho para uma superfície invisível é um atalho para nada — a mesma
// regra que o `focus_toggle` já aplica.
func (a *App) OpenPrompt(kind string) error {
	if a.Tree == nil {
		tree, err := fs.Open(a.Workspace, a.Config.Tree.ShowHidden)
		if err != nil {
			return err
		}
		a.Tree = tree
	}
	a.ShowTree = true
	a.Focus = FocusTree
	a.Prompt = PromptState{Kind: kind}
	a.Overlay = OverlayPrompt
	if kind == PromptNewDir {
		a.Status = "nova pasta: digite o nome · Enter cria · Esc cancela"
	} else {
		a.Status = "novo arquivo: digite o nome · Enter cria e abre · Esc cancela"
	}
	return nil
}

// PromptAppend types into the prompt, ignoring controls: um controle no nome de
// um arquivo é um nome que o disco recusa, e recusar na digitação é recusar com
// o motivo visível — o leitor vê que a tecla não entrou.
func (a *App) PromptAppend(text string) {
	for _, r := range text {
		if unicode.IsControl(r) {
			continue
		}
		a.Prompt.Buffer += string(r)
	}
}

// PromptBackspace erases the last character — the last **rune**, porque apagar
// um byte de um caractere de dois é deixar metade de um caractere no nome.
func (a *App) PromptBackspace() {
	runes := []rune(a.Prompt.Buffer)
	if len(runes) == 0 {
		return
	}
	a.Prompt.Buffer = string(runes[:len(runes)-1])
}

// SubmitPrompt creates what the prompt asked for and closes it: o `Enter`
// fecha com sucesso ou com erro, como no oráculo — um prompt que fica aberto
// depois de criar é um prompt que mente sobre o que já aconteceu.
//
// Um arquivo criado e aberto não ganha status: o arquivo aberto é a resposta.
// Uma pasta ganha, porque nada mais muda na tela para dizê-lo.
func (a *App) SubmitPrompt() {
	kind := fs.CreateFile
	if a.Prompt.Kind == PromptNewDir {
		kind = fs.CreateDirectory
	}
	a.Overlay = OverlayNone
	if a.Tree == nil {
		return
	}
	target, err := a.Tree.CreateUnderSelection(kind, a.Prompt.Buffer)
	if err != nil {
		a.Status = "criar: " + err.Error()
		return
	}
	if kind == fs.CreateDirectory {
		a.Status = "pasta criada"
		return
	}
	if _, err := a.Store.OpenPath(target); err != nil {
		a.Status = "abrir: " + err.Error()
		return
	}
	a.Focus = FocusEditor
}
