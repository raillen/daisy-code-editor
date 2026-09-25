package app

import (
	"strconv"

	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/buffer"
	"github.com/ori-team/oride/internal/clipboard"
	"github.com/ori-team/oride/internal/editor"
)

// clipboardCommands are copy, paste and cut.
//
// As três têm a mesma forma, que é a do oráculo
// (crates/oride-app/src/app/actions.rs): **a seleção, ou a linha inteira quando
// não há seleção**. A regra existe porque é a que o leitor espera — "copiar" sem
// seleção num editor de linha copia a linha, e não um caractere aleatório.
func clipboardCommands() map[action.Action]Handler {
	return map[action.Action]Handler{
		action.Copy: documentHandler(func(a *App, document *editor.Document) error {
			a.copySelectionOrLine(document)
			return nil
		}),

		action.Cut: documentHandler(func(a *App, document *editor.Document) error {
			return a.cutSelectionOrLine(document)
		}),

		action.Paste: documentHandler(func(a *App, document *editor.Document) error {
			return a.pasteInto(document)
		}),
	}
}

// copySelectionOrLine puts the selection, or the current line, on the clipboard.
func (a *App) copySelectionOrLine(document *editor.Document) {
	text := selectionOrLine(document)
	if text == "" {
		a.Status = "nada para copiar"
		return
	}

	a.ClipboardWrite = clipboard.Copy(text)
	a.Status = "copiado " + itoa(len(text)) + " bytes"
}

// cutSelectionOrLine copies and then deletes what it copied.
//
// A ordem é obrigatória: cortar antes de copiar apaga o texto de quem vai colar
// nos dois buffers, e um `ctrl+x` que não cola é um `ctrl+x` que perdeu dados.
func (a *App) cutSelectionOrLine(document *editor.Document) error {
	text := selectionOrLine(document)
	if text == "" {
		a.Status = "nada para cortar"
		return nil
	}
	a.ClipboardWrite = clipboard.Copy(text)

	if !document.Selection().IsEmpty() {
		if err := document.DeleteSelection(); err != nil {
			return err
		}
		a.Status = "cortado " + itoa(len(text)) + " bytes"
		return nil
	}
	return a.deleteCurrentLine(document, text)
}

// pasteInto inserts the session's clipboard at the caret.
func (a *App) pasteInto(document *editor.Document) error {
	text := clipboard.Paste()
	if text == "" {
		// A mensagem diz de onde a colagem vem, porque um `ctrl+v` que não cola
		// sem explicação é indistinguível de um terminal que engoliu a tecla.
		a.Status = "clipboard vazio (a cópia da sessão é a fonte)"
		return nil
	}
	if err := document.InsertText(text); err != nil {
		return err
	}
	a.Status = "colado " + itoa(len(text)) + " bytes"
	return nil
}

// deleteCurrentLine removes the caret's line, the line break included.
//
// O fim de linha **completo** some junto: cortar uma linha e deixar o `\n` produz
// uma linha em branco a cada corte, e o documento cresce enquanto o leitor
// acredita estar deletando. O teste do oráculo
// (`cut_line_removes_complete_crlf_ending`) fixa exatamente isto.
func (a *App) deleteCurrentLine(document *editor.Document, copied string) error {
	caret, err := document.Caret()
	if err != nil {
		return err
	}
	start, err := document.Buffer().LineToByte(caret.Line)
	if err != nil {
		return err
	}

	end := buffer.Offset(len(copied))
	if next := caret.Line + 1; next < document.Buffer().LineCount() {
		nextStart, err := document.Buffer().LineToByte(next)
		if err != nil {
			return err
		}
		end = nextStart
	}

	document.SelectByteRange(start, end)
	if err := document.DeleteSelection(); err != nil {
		return err
	}
	document.CollapseSelection()
	a.Status = "cortado " + itoa(len(copied)) + " bytes"
	return nil
}

// selectionOrLine is what a copy verb acts on: the selection, or the caret's line.
func selectionOrLine(document *editor.Document) string {
	if text := document.SelectedText(); text != "" {
		return text
	}
	caret, err := document.Caret()
	if err != nil {
		return ""
	}
	line, err := document.Buffer().LineText(caret.Line)
	if err != nil {
		return ""
	}
	return line
}

// itoa is strconv.Itoa, and the messages say **bytes** because that is the unit the
// oracle reports in.
func itoa(value int) string { return strconv.Itoa(value) }
