package app

import (
	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/editor"
)

// editCommands change the document's text or its history.
func editCommands() map[action.Action]Handler {
	return map[action.Action]Handler{
		// A falha do undo **não** é erro do programa: desfazer sem histórico é
		// uma situação normal, e a palavra é a mesma do oráculo
		// (crates/oride-app/src/app/actions.rs) porque o texto da barra é
		// comparado byte a byte pela paridade.
		action.Undo: documentHandler(func(a *App, document *editor.Document) error {
			undone, err := document.Undo()
			if err != nil {
				return err
			}
			if !undone {
				a.Status = "nothing to undo"
			}
			return nil
		}),
		action.Redo: documentHandler(func(a *App, document *editor.Document) error {
			redone, err := document.Redo()
			if err != nil {
				return err
			}
			if !redone {
				a.Status = "nothing to redo"
			}
			return nil
		}),

		action.InsertNewline: documentHandler(func(_ *App, document *editor.Document) error {
			return document.InsertText("\n")
		}),
		action.InsertTab: documentHandler(func(a *App, document *editor.Document) error {
			return a.indent(document)
		}),

		action.Backspace: documentHandler(func(_ *App, document *editor.Document) error {
			return document.Backspace()
		}),
		action.Delete: documentHandler(func(_ *App, document *editor.Document) error {
			return document.DeleteForward()
		}),
	}
}
