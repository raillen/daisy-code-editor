package app

import (
	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/editor"
)

// movement is one cursor direction, in both of its variants.
//
// Plain and extend are the same movement with the selection flag set, which is
// the domain truth rather than a coincidence: declaring them as a pair keeps the
// two from drifting apart, and stops eight directions becoming sixteen entries.
type movement struct {
	plain  action.Action
	extend action.Action
	move   func(*editor.Document, bool) error
}

// movements pairs every direction with the document method it calls.
var movements = []movement{
	{action.MoveLeftPlain, action.MoveLeftExtend, (*editor.Document).MoveLeft},
	{action.MoveRightPlain, action.MoveRightExtend, (*editor.Document).MoveRight},
	{action.MoveUpPlain, action.MoveUpExtend, (*editor.Document).MoveUp},
	{action.MoveDownPlain, action.MoveDownExtend, (*editor.Document).MoveDown},
	{action.MoveLineStartPlain, action.MoveLineStartExtend, (*editor.Document).MoveLineStart},
	{action.MoveLineEndPlain, action.MoveLineEndExtend, (*editor.Document).MoveLineEnd},
	{action.MoveDocStartPlain, action.MoveDocStartExtend, (*editor.Document).MoveBufferStart},
	{action.MoveDocEndPlain, action.MoveDocEndExtend, (*editor.Document).MoveBufferEnd},
}

// movementCommands derives both variants of every direction.
func movementCommands() map[action.Action]Handler {
	table := make(map[action.Action]Handler, len(movements)*2)

	for _, direction := range movements {
		table[direction.plain] = documentHandler(func(_ *App, document *editor.Document) error {
			return direction.move(document, false)
		})
		table[direction.extend] = documentHandler(func(_ *App, document *editor.Document) error {
			return direction.move(document, true)
		})
	}

	// Page up and page down are N movements of one line, with N being what fits on
	// the screen. N is a property of the viewport, so it arrives from the
	// presentation — and the value is the same rule the oracle uses
	// (crates/oride-app/src/app/actions.rs): altura menos uma linha, no mínimo
	// uma.
	table[action.PageUp] = documentHandler(func(a *App, document *editor.Document) error {
		return a.page(document, (*editor.Document).MoveUp, -1)
	})
	table[action.PageDown] = documentHandler(func(a *App, document *editor.Document) error {
		return a.page(document, (*editor.Document).MoveDown, 1)
	})
	return table
}

// page moves the caret by one screenful, without extending the selection.
//
// Uma linha por vez, e não um salto: o movimento de linha é o que sabe colar no
// início e no fim da linha, e um salto próprio saltaria essa regra — que é
// exatamente o que o oráculo faz.
func (a *App) page(document *editor.Document, move func(*editor.Document, bool) error, times int) error {
	steps := max(a.EditorHeight-1, 1)
	for range steps {
		if err := move(document, false); err != nil {
			return err
		}
	}
	return nil
}
