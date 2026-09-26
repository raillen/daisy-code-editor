package app

import (
	"github.com/ori-team/oride/internal/action"
	"github.com/ori-team/oride/internal/i18n"
)

// MenuItem is one entry of one menu: what the reader reads, what it costs to
// press, and what it does.
type MenuItem struct {
	// Label is what the reader reads, in the session's locale when the catalog
	// has one — and in the reference's own English when it does not. Mixing is
	// deliberate and documented: the reference mixes the same way.
	Label string
	// Shortcut is how the entry reads in the key table, for the reader who
	// presses keys instead of menus. Empty when the entry has no binding.
	Shortcut string
	// Action is the action id the entry runs.
	Action action.Action
}

// MenuColumn is one dropdown of the menu bar.
type MenuColumn struct {
	// Title is the bar label, localized.
	Title string
	// Hotkey is the letter that opens it with Alt.
	Hotkey rune
	Items  []MenuItem
}

// menuHotkeys are the letters that open each column with Alt, in bar order.
//
// Os mesmos do oráculo (crates/oride-app/src/menus.rs) — a referência escolheu as
// letras, e paridade aqui não é sobre língua, é sobre a mão lembrar onde fica.
var menuHotkeys = []rune{'f', 'e', 'v', 'g', 'i', 'h'}

// Menus builds the six columns from a catalog.
//
// A estrutura é do produto e os textos são do catálogo, com os literais do oráculo
// onde o catálogo não nomeia. Essa mistura é **o que a referência faz**, e
// escondê-la seria pior: um leitor em pt-BR lendo "Git pull" em inglês sabe que
// aquilo é uma label real, não um id que vazou para a tela.
func Menus(catalog i18n.Catalog) []MenuColumn {
	file, edit, view := catalog.File, catalog.Edit, catalog.View
	return []MenuColumn{
		{
			Title:  catalog.Menu.File,
			Hotkey: 'f',
			Items: []MenuItem{
				{Label: file.NewTab, Shortcut: "ctrl+n", Action: action.NewTab},
				{Label: file.OpenFile, Shortcut: "ctrl+p", Action: action.OpenFileFuzzy},
				{Label: file.OpenFolder, Shortcut: "ctrl+o", Action: action.OpenFolder},
				{Label: file.Save, Shortcut: "ctrl+s", Action: action.Save},
				{Label: file.SaveAs, Shortcut: "f12", Action: action.SaveAs},
				{Label: file.SaveAll, Shortcut: "ctrl+alt+s", Action: action.SaveAll},
				{Label: file.ReloadFile, Shortcut: "ctrl+r", Action: action.ReloadFile},
				{Label: file.Quit, Shortcut: "ctrl+q", Action: action.Quit},
			},
		},
		{
			Title:  catalog.Menu.Edit,
			Hotkey: 'e',
			Items: []MenuItem{
				{Label: edit.Undo, Shortcut: "ctrl+z", Action: action.Undo},
				{Label: edit.Redo, Shortcut: "ctrl+y", Action: action.Redo},
				{Label: edit.Cut, Shortcut: "ctrl+x", Action: action.Cut},
				{Label: edit.Copy, Shortcut: "ctrl+c", Action: action.Copy},
				{Label: edit.Paste, Shortcut: "ctrl+v", Action: action.Paste},
				{Label: edit.SelectAll, Shortcut: "ctrl+a", Action: action.SelectAll},
				{Label: edit.Find, Shortcut: "ctrl+f", Action: action.Find},
				{Label: edit.FindInProject, Shortcut: "ctrl+shift+f", Action: action.ProjectFind},
				{Label: edit.Replace, Shortcut: "ctrl+h", Action: action.Replace},
				{Label: edit.ToggleComment, Shortcut: "ctrl+/", Action: action.ToggleComment},
			},
		},
		{
			Title:  catalog.Menu.View,
			Hotkey: 'v',
			Items: []MenuItem{
				{Label: view.CommandPalette, Shortcut: "ctrl+shift+p", Action: action.CommandPalette},
				{Label: view.ToggleTree, Shortcut: "ctrl+shift+b", Action: action.ToggleTree},
				{Label: view.ToggleTerminal, Shortcut: "ctrl+`", Action: action.ToggleTerminal},
				{Label: view.ToggleSCM, Shortcut: "ctrl+shift+g", Action: action.ToggleScm},
				{Label: "MD preview", Shortcut: "alt+p", Action: action.ToggleMdPreview},
				{Label: "Soft wrap", Shortcut: "alt+z", Action: action.ToggleSoftWrap},
				{Label: "Split vertical", Shortcut: "ctrl+alt+v", Action: action.SplitVertical},
				{Label: "Split horizontal", Shortcut: "ctrl+alt+h", Action: action.SplitHorizontal},
				{Label: "Which-key", Shortcut: "alt+/", Action: action.WhichKey},
				{Label: view.ColorTheme, Shortcut: "palette", Action: action.SelectTheme},
				{Label: view.DisplayLanguage, Shortcut: "palette", Action: action.SelectLocale},
				{Label: view.ToggleMouse, Shortcut: "palette", Action: action.ToggleMouse},
			},
		},
		{
			Title:  catalog.Menu.Go,
			Hotkey: 'g',
			Items: []MenuItem{
				{Label: "Buffer picker…", Shortcut: "ctrl+shift+o", Action: action.BufferPicker},
				{Label: "Jump back", Shortcut: "ctrl+alt+o", Action: action.JumpBack},
				{Label: "Jump forward", Shortcut: "ctrl+alt+i", Action: action.JumpForward},
				{Label: "LSP: definition", Shortcut: "f4", Action: action.LspGotoDefinition},
				{Label: "LSP: hover", Shortcut: "ctrl+k", Action: action.LspHover},
				{Label: "Diagnostics", Shortcut: "ctrl+shift+m", Action: action.ToggleDiagnostics},
				{Label: "Next tab", Shortcut: "ctrl+pagedown", Action: action.NextTab},
				{Label: "Prev tab", Shortcut: "ctrl+pageup", Action: action.PrevTab},
			},
		},
		{
			Title:  catalog.Menu.Git,
			Hotkey: 'i',
			Items: []MenuItem{
				{Label: view.ToggleSCM, Shortcut: "ctrl+shift+g", Action: action.ToggleScm},
				{Label: "Git pull", Shortcut: "", Action: action.GitPull},
				{Label: "Git push", Shortcut: "", Action: action.GitPush},
				{Label: "Diff active file", Shortcut: "f2", Action: action.ShowDiff},
				{Label: "Refresh git", Shortcut: "f5", Action: action.TreeRefresh},
			},
		},
		{
			Title:  catalog.Menu.Help,
			Hotkey: 'h',
			Items: []MenuItem{
				{Label: "All keybindings…", Shortcut: "f1", Action: action.Help},
				{Label: "Essential shortcuts", Shortcut: "alt+shift+/", Action: action.Welcome},
				{Label: "Which-key", Shortcut: "alt+/", Action: action.WhichKey},
				{Label: view.CommandPalette, Shortcut: "ctrl+shift+p", Action: action.CommandPalette},
				{Label: "Multi picker…", Shortcut: "ctrl+shift+t", Action: action.MultiPicker},
				{Label: "Surround selection…", Shortcut: "f8", Action: action.Surround},
				{Label: "Undo history…", Shortcut: "ctrl+shift+u", Action: action.UndoTree},
				{Label: view.DisplayLanguage, Shortcut: "palette", Action: action.SelectLocale},
			},
		},
	}
}

// MenuHotkeys is the hotkey of each column, in bar order, for the capture that
// opens them.
func MenuHotkeys() []rune { return append([]rune(nil), menuHotkeys...) }
