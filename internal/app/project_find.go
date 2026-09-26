package app

import (
	"fmt"
	"strings"

	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/editor"
	"github.com/ori-team/oride/internal/search"
)

// ProjectFindState is the project search: what the reader typed, what the
// workspace answered, and where the keyboard is.
//
// Os mesmos campos que o oráculo dumpha — `query`, `replace_query`, `file_glob`,
// `selected`, as opções — e na mesma forma. A lista de resultados (`Hits`) é do
// modelo porque é ela que a tecla percorre, e a recomputação acontece a cada
// tecla porque é isso que "busca ao vivo" quer dizer.
type ProjectFindState struct {
	Query        string
	Replace      string
	HasReplace   bool
	Glob         string
	HasGlob      bool
	Options      search.Options
	Hits         []search.Hit
	Selected     int
	FocusField   int
	RecomputeErr string
	LastSummary  string
}

// The fields, in the order `Tab` walks them: 0 é a consulta, 1 a substituição e
// 2 o glob — que é a mesma ordem em que o oráculo os numera.
const (
	ProjectFieldQuery = iota
	ProjectFieldReplace
	ProjectFieldGlob
)

// As opções da busca no projeto vêm do produto e não do `config.toml`: a
// sensibilidade a caixa e o regex são *teclas* (`Alt+C`, `Alt+R`), e uma opção
// que só existe num arquivo de configuração para uma busca interativa é uma
// opção que o leitor nunca muda.
//
// projectFindCommands open the project search, with or without the replace field.
func projectFindCommands() map[action.Action]Handler {
	return map[action.Action]Handler{
		action.ProjectFind: modelHandler(func(a *App, _ *editor.Document) error {
			a.ProjectFind = ProjectFindState{Options: search.Options{}}
			a.Overlay = OverlayProjectFind
			a.Status = "project find · digite · Alt+G glob · Alt+C case · Alt+R regex · Enter abre"
			return nil
		}),

		action.ProjectReplace: modelHandler(func(a *App, _ *editor.Document) error {
			a.ProjectFind = ProjectFindState{Options: search.Options{}, HasReplace: true, FocusField: ProjectFieldReplace}
			a.Overlay = OverlayProjectFind
			a.Status = "project replace · digite · Tab alterna os campos · Enter substitui"
			return nil
		}),
	}
}

// projectQuery is the search the state's fields describe.
func (a *App) projectQuery() search.Query {
	globs := []string{}
	if a.ProjectFind.HasGlob && strings.TrimSpace(a.ProjectFind.Glob) != "" {
		globs = []string{strings.TrimSpace(a.ProjectFind.Glob)}
	}
	return search.Query{
		Text:          a.ProjectFind.Query,
		CaseSensitive: a.ProjectFind.Options.CaseSensitive,
		UseRegex:      a.ProjectFind.Options.UseRegex,
		Globs:         globs,
	}
}

// recomputeProjectFind searches again with the current fields, e é chamada a cada
// tecla que muda a busca — porque "busca ao vivo" é a promessa do produto, e um
// resultado que fica para trás da digitação é uma promessa quebrada.
func (a *App) RecomputeProjectFind() {
	state := &a.ProjectFind
	state.RecomputeErr = ""
	state.Hits = nil
	state.Selected = 0

	if strings.TrimSpace(state.Query) == "" {
		return
	}
	if a.Workspace == "" {
		return
	}

	hits, _, err := search.FindProject(a.Workspace, a.projectQuery())
	if err != nil {
		state.RecomputeErr = err.Error()
		return
	}
	state.Hits = hits
}

// executeProjectReplace substitutes the query across the project and reloads what
// was open, que é o que impede um documento aberto de divergir do disco que ele
// acabou de escrever.
func (a *App) ExecuteProjectReplace() {
	query := strings.TrimSpace(a.ProjectFind.Query)
	if query == "" {
		a.Status = "busca vazia: nada a substituir"
		return
	}

	summary, modified, err := search.ReplaceInProject(a.Workspace, a.projectQuery(), a.ProjectFind.Replace)
	if err != nil {
		a.Status = fmt.Sprintf("erro na substituição: %s", err)
		return
	}
	for _, relative := range modified {
		if id, found := a.Store.DocumentIDForPath(a.Workspace + "/" + relative); found {
			if document, docExists := a.Store.Get(id); docExists {
				_ = document.ReloadFromDisk()
			}
		}
	}
	a.Status = fmt.Sprintf("Substituídas %d ocorrências em %d arquivos", summary.ReplacementsCount, summary.FilesModified)
	a.Overlay = OverlayNone
}

// jumpToProjectHit opens the hit's file and moves the caret to its line.
func (a *App) JumpToProjectHit(hit search.Hit) error {
	id, err := a.Store.OpenPath(a.Workspace + "/" + hit.Path)
	if err != nil {
		return err
	}
	document, _ := a.Store.Get(id)
	if document == nil {
		return fmt.Errorf("documento não abriu: %s", hit.Path)
	}
	line, err := document.Buffer().LineToByte(hit.Line - 1)
	if err != nil {
		return err
	}
	document.JumpToByte(line)
	return nil
}
