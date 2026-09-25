package app

import (
	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/editor"
)

// fileCommands are the actions about the session and the documents in it.
func fileCommands() map[action.Action]Handler {
	return map[action.Action]Handler{
		action.Quit: modelHandler(func(a *App, _ *editor.Document) error {
			a.Quit = true
			return nil
		}),

		// Save, SaveAs and SaveAll are one behaviour here: the headless model
		// has no browser, so a document with no path keeps its dirty flag and
		// says so. See markSaved.
		action.Save:    documentHandler(func(a *App, _ *editor.Document) error { return a.markSaved() }),
		action.SaveAs:  documentHandler(func(a *App, _ *editor.Document) error { return a.markSaved() }),
		action.SaveAll: documentHandler(func(a *App, _ *editor.Document) error { return a.markSaved() }),

		// Tabs. The store already had the three operations; what was missing was
		// the dispatch, which is why `ctrl+n` and `ctrl+w` were advertised in a
		// generated table and did nothing.
		action.NextTab: modelHandler(func(a *App, _ *editor.Document) error { return a.cycleTab(1) }),
		action.PrevTab: modelHandler(func(a *App, _ *editor.Document) error { return a.cycleTab(-1) }),

		action.NewTab: modelHandler(func(a *App, _ *editor.Document) error {
			a.Store.OpenEmpty()
			return nil
		}),

		// CloseTab closes the active document and says what happened, because a
		// key that closes the last tab and leaves an empty editor without a word
		// looks like a crash.
		action.CloseTab: modelHandler(func(a *App, _ *editor.Document) error {
			document, err := a.Store.Active()
			if err != nil {
				return err
			}
			if _, err := a.Store.Close(document.ID()); err != nil {
				return err
			}
			if a.Store.Len() == 0 {
				// O último documento fecha e um novo buffer vazio aparece: um editor
				// sem nenhum documento não tem onde placing o caret.
				a.Store.OpenEmpty()
				a.Status = "última aba fechada"
				return nil
			}
			a.Status = "aba fechada"
			return nil
		}),
	}
}

// cycleTab moves the active tab, wrapping around.
//
// Uma aba só não "anda": ciclar de uma em uma é a tecla fazendo nada, e o leitor
// não distingue isso de um programa quebrado.
func (a *App) cycleTab(delta int) error {
	if a.Store.Len() < 2 {
		return nil
	}
	if delta < 0 {
		// "Não moveu" é o caso normal de "esta aba é a última", e não uma falha.
		a.Store.ActivatePrevTab()
		return nil
	}
	a.Store.ActivateNextTab()
	return nil
}
