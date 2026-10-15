import type { KeyEvent } from "@opentui/core"
import { join } from "node:path"
import { deleteTreeSelection, renameTreeSelection } from "./commands"
import type { Application } from "./application"
import { menus } from "./menus"
import { receiveVimKey } from "./vim"

export function chordOf(key: Pick<KeyEvent, "name" | "ctrl" | "meta" | "shift">): string {
  const aliases: Record<string, string> = { return: "enter", linefeed: "enter", escape: "esc", prior: "pageup", next: "pagedown" }
  const name = aliases[key.name] ?? key.name.toLowerCase()
  return [key.ctrl ? "ctrl" : "", key.meta ? "alt" : "", key.shift ? "shift" : "", name].filter(Boolean).join("+")
}
function printable(key: KeyEvent): string | undefined {
  if (key.ctrl || key.meta || key.super || key.hyper) return undefined
  if (key.name === "space") return " "
  return key.sequence && !/[\x00-\x1f\x7f]/.test(key.sequence) ? key.sequence : undefined
}

const menuColumnByKey: Record<string, number> = {}
for (const [colIndex, menu] of menus.entries()) {
  for (const k of menu.altKeys) {
    menuColumnByKey[k.toLowerCase()] = colIndex
  }
}

export function receiveKey(app: Application, key: KeyEvent): void {
  if (key.eventType === "release") return
  key.preventDefault(); key.stopPropagation()
  const chord = chordOf(key)
  if (app.overlay) { void app.guard(() => overlayKey(app, key, chord)); return }

  // Alt solo toggle or F10 menubar focus
  const isAltSolo = (key.name === "alt" || key.name === "left_alt" || key.name === "right_alt" || key.name === "meta") && !key.ctrl && !key.shift
  if (isAltSolo) {
    app.altActive = !app.altActive
    if (!app.altActive && app.menu) app.menu = undefined
    app.changed()
    return
  }
  if (key.name === "f10") {
    if (app.menu) {
      app.menu = undefined
      app.altActive = false
    } else {
      app.menu = { column: 0, selected: 0 }
      app.altActive = true
    }
    app.changed()
    return
  }

  // Active open menu navigation
  if (app.menu) { menuKey(app, key); return }

  // Alt+Key combo (e.g. Alt+F, Alt+A, Alt+E, Alt+V, Alt+X, etc.)
  if (key.meta && !key.ctrl && !key.shift) {
    const column = menuColumnByKey[key.name.toLowerCase()]
    if (column !== undefined) {
      app.menu = { column, selected: 0 }
      app.altActive = true
      app.changed()
      return
    }
  }

  // If altActive is set (user tapped Alt), pressing a menu mnemonic letter directly opens that menu
  if (app.altActive) {
    if (key.name === "escape") {
      app.altActive = false
      app.changed()
      return
    }
    if (!key.ctrl && !key.meta && !key.shift) {
      const column = menuColumnByKey[key.name.toLowerCase()]
      if (column !== undefined) {
        app.menu = { column, selected: 0 }
        app.altActive = true
        app.changed()
        return
      }
    }
    app.altActive = false
    app.changed()
  }
  if (app.focus === "terminal" && app.bottom === "terminal") {
    if (key.name === "escape") { app.focus = "editor"; app.changed() }
    else if (chord === "ctrl+`" || chord === "ctrl+'" || chord === "ctrl+\"") void app.dispatch("toggle_terminal")
    else if (chord === "alt+=" || chord === "alt+-") void app.dispatch(chord === "alt+=" ? "terminal_grow" : "terminal_shrink")
    else {
      const sequences: Record<string, string> = { return: "\r", enter: "\r", backspace: "\x7f", delete: "\x1b[3~", up: "\x1b[A", down: "\x1b[B", right: "\x1b[C", left: "\x1b[D", home: "\x1b[H", end: "\x1b[F", pageup: "\x1b[5~", pagedown: "\x1b[6~", tab: "\t" }
      void app.guard(() => app.terminal?.write(key.ctrl && key.name.length === 1 ? String.fromCharCode(key.name.toUpperCase().charCodeAt(0) & 31) : sequences[key.name] ?? key.raw ?? key.sequence))
    }
    return
  }
  if (app.preview && app.focus === "editor") { previewKey(app, key, chord); return }
  if (app.completion && app.focus === "editor") {
    if (key.name === "escape") { app.cancelCompletion(); return }
    if (key.name === "tab" || chord === "enter" || key.name === "return") { app.acceptCompletion(); return }
    if (key.name === "down" || chord === "ctrl+n") { app.nextCompletion(1); return }
    if (key.name === "up" || chord === "ctrl+p") { app.nextCompletion(-1); return }
    if (key.name === "pagedown") { app.nextCompletion(5); return }
    if (key.name === "pageup") { app.nextCompletion(-5); return }
    if (key.name === "left" || key.name === "right" || key.name === "home" || key.name === "end") {
      app.cancelCompletion()
    }
  }
  if (key.name === "escape") {
    if (app.focus !== "editor") app.focus = "editor"
    else if (app.config.editor.modal_mode) { app.vim = "normal"; app.vimPending = ""; app.document.select(app.document.primary.head) }
    else if (app.document.selections.length > 1 || app.document.selectedText) app.document.select(app.document.primary.head)
    else app.status = "Ctrl+Q quits · Ctrl+Shift+P commands"
    app.changed(); return
  }
  if (key.name === "tab" && app.focus !== "editor") { app.focus = "editor"; app.changed(); return }
  if (key.name === "tab" && key.shift) { app.showTree = true; app.focus = "tree"; app.changed(); return }
  if (app.focus === "tree" && !key.ctrl && !key.meta) { void app.guard(() => treeKey(app, key)); return }
  if (app.config.editor.modal_mode && receiveVimKey(app, key)) return
  const action = app.config.keys[chord]
  if (action) { void app.dispatch(action); return }
  const text = printable(key)
  if (text && app.focus === "editor") {
    app.typeChar(text)
    if (app.config.editor.completion_auto) {
      const prefix = app.document.text.slice(0, app.document.primary.head).match(/[\p{L}\p{N}_]+$/u)?.[0] ?? ""
      if (prefix.length >= app.config.editor.completion_min_chars) {
        if (app.completion) void app.complete(true)
        else app.scheduleCompletion()
      } else {
        app.cancelCompletion()
      }
    } else if (app.completion) {
      const prefix = app.document.text.slice(0, app.document.primary.head).match(/[\p{L}\p{N}_]+$/u)?.[0] ?? ""
      if (prefix) void app.complete(true)
      else app.cancelCompletion()
    }
  }
}

export function receivePaste(app: Application, text: string): void {
  const overlay = app.overlay
  if (overlay?.kind === "prompt") overlay.value += text.replace(/[\r\n]/g, "")
  else if (overlay?.kind === "picker") { overlay.query += text.replace(/[\r\n]/g, ""); overlay.selected = 0; app.updatePicker() }
  else if (overlay?.kind === "search") {
    const field = overlay.field === 0 ? "query" : overlay.field === 1 ? "replacement" : "glob"
    overlay[field] += text.replace(/[\r\n]/g, ""); void app.updateSearch()
  } else if (app.focus === "terminal") app.terminal?.write(text)
  else if (!overlay && !app.preview) app.insert(text)
  app.changed()
}

function previewKey(app: Application, key: KeyEvent, chord: string): void {
  if (key.name === "escape" || chord === "q") {
    app.preview = false
    app.status = "md preview: off"
    app.changed()
    return
  }
  if (chord === "alt+p" || chord === "ctrl+shift+v") {
    void app.dispatch("toggle_md_preview")
    return
  }
  if (chord === "alt+enter") return
  if (key.name === "down" || chord === "j" || chord === "alt+down") {
    app.scrollPreview(3)
    return
  }
  if (key.name === "up" || chord === "k" || chord === "alt+up") {
    app.scrollPreview(-3)
    return
  }
  if (key.name === "pagedown" || chord === "ctrl+d" || chord === "alt+pagedown" || key.name === "space") {
    app.scrollPreview(Math.max(1, app.editorHeight - 2))
    return
  }
  if (key.name === "pageup" || chord === "ctrl+u" || chord === "alt+pageup" || chord === "shift+space") {
    app.scrollPreview(-Math.max(1, app.editorHeight - 2))
    return
  }
  if (key.name === "home" || chord === "g") {
    app.scrollPreviewTo(0)
    return
  }
  if (key.name === "end" || chord === "shift+g") {
    app.scrollPreviewTo(Number.MAX_SAFE_INTEGER)
    return
  }
  const action = app.config.keys[chord]
  if (action) {
    const isEditAction = action.startsWith("move_") || [
      "insert_newline", "insert_tab", "backspace", "delete", "undo", "redo",
      "select_all", "cut", "paste", "toggle_comment", "lsp_complete", "lsp_hover",
      "lsp_goto_definition", "lsp_format", "add_cursor_above", "add_cursor_below",
      "clear_extra_cursors", "surround", "multi_picker"
    ].includes(action)
    if (!isEditAction) void app.dispatch(action)
  }
}

async function overlayKey(app: Application, key: KeyEvent, chord: string): Promise<void> {
  const overlay = app.overlay
  if (!overlay) return
  if (key.name === "escape") {
    if (overlay.kind === "picker" && overlay.onCancel) overlay.onCancel()
    app.overlay = undefined; app.changed(); return
  }
  const enter = key.name === "return" || key.name === "enter"
  const text = printable(key)
  const erase = (value: string) => [...new Intl.Segmenter(undefined, { granularity: "grapheme" }).segment(value)].slice(0, -1).map(segment => segment.segment).join("")
  if (overlay.kind === "confirm") {
    if (enter) { app.overlay = undefined; await overlay.accept(); app.changed() }
    return
  }
  if (overlay.kind === "read") {
    if (enter) app.overlay = undefined
    if (key.name === "down" || key.name === "pagedown") overlay.scroll = Math.min(Math.max(0, overlay.text.split("\n").length - 1), overlay.scroll + (key.name === "down" ? 1 : 10))
    if (key.name === "up" || key.name === "pageup") overlay.scroll = Math.max(0, overlay.scroll - (key.name === "up" ? 1 : 10))
  }
  if (overlay.kind === "scm") {
    const entries = app.gitStatus.entries
    const entry = entries[overlay.selected]
    if (enter && entry) { app.overlay = undefined; app.recordJump(); await app.openFile(join(app.workspace, entry.path)) }
    else if (key.name === "down" || key.name === "j") overlay.selected = Math.min(entries.length - 1, overlay.selected + 1)
    else if (key.name === "up" || key.name === "k") overlay.selected = Math.max(0, overlay.selected - 1)
    else if (key.name === "s" && entry) { await app.git.stage(entry.path); await app.refreshGit() }
    else if (key.name === "u" && entry) { await app.git.unstage(entry.path); await app.refreshGit() }
    else if (key.name === "d" && entry) app.overlay = { kind: "read", title: "Git diff", text: await app.git.diff(entry.path), scroll: 0 }
    else if (key.name === "c") app.prompt("Git commit message", "", async message => { app.status = await app.git.commit(message); await app.refreshGit() })
    else if (key.name === "r") await app.refreshGit()
    else if (key.name === "P") await app.dispatch("git_push")
    else if (key.name === "p") await app.dispatch("git_pull")
    app.changed(); return
  }
  if (overlay.kind === "prompt") {
    if (enter) { app.overlay = undefined; await overlay.submit(overlay.value); app.afterEdit() }
    else if (key.name === "backspace") overlay.value = erase(overlay.value)
    else if (chord === "ctrl+u") overlay.value = ""
    else if (text) overlay.value += text
  }
  if (overlay.kind === "picker") {
    const prevSelected = overlay.selected
    if (enter) {
      const entry = app.pickerEntries[overlay.selected]
      if (entry) { app.overlay = undefined; await entry.run(); app.afterEdit() }
    } else if (key.name === "down") overlay.selected = Math.min(app.pickerEntries.length - 1, overlay.selected + 1)
    else if (key.name === "up") overlay.selected = Math.max(0, overlay.selected - 1)
    else if (key.name === "backspace") { overlay.query = erase(overlay.query); overlay.selected = 0; app.updatePicker() }
    else if (chord === "ctrl+u") { overlay.query = ""; overlay.selected = 0; app.updatePicker() }
    else if (text) { overlay.query += text; overlay.selected = 0; app.updatePicker() }
    if (overlay.selected !== prevSelected || text || key.name === "backspace" || chord === "ctrl+u") {
      const current = app.pickerEntries[overlay.selected]
      if (current && overlay.onSelect) overlay.onSelect(current, overlay.selected)
    }
  }
  if (overlay.kind === "search") {
    if (chord === "alt+c") app.searchOptions.caseSensitive = !app.searchOptions.caseSensitive
    else if (chord === "alt+a") app.searchOptions.ignoreAccents = !app.searchOptions.ignoreAccents
    else if (chord === "alt+w") app.searchOptions.wholeWord = !app.searchOptions.wholeWord
    else if (chord === "alt+r") app.searchOptions.regex = !app.searchOptions.regex
    else if (chord === "alt+g") overlay.field = 2
    else if (key.name === "tab") {
      overlay.replace = true
      overlay.field = (overlay.field + 1) % (overlay.project ? 3 : 2)
    } else if (enter && key.meta) {
      if (overlay.project && key.ctrl) app.overlay = { kind: "confirm", title: `Replace ${overlay.hits.length} matches on disk? Enter confirms`, accept: async () => { app.overlay = overlay; await app.replaceSearch(true) } }
      else await app.replaceSearch(key.ctrl)
      app.changed(); return
    } else if (enter) { await app.openSearchHit(); return }
    else if (key.name === "down") overlay.selected = Math.min(overlay.hits.length - 1, overlay.selected + 1)
    else if (key.name === "up") overlay.selected = Math.max(0, overlay.selected - 1)
    else {
      const field = overlay.field === 0 ? "query" : overlay.field === 1 ? "replacement" : "glob"
      if (key.name === "backspace") overlay[field] = erase(overlay[field])
      else if (text) overlay[field] += text
    }
    await app.updateSearch()
  }
  app.changed()
}
function menuKey(app: Application, key: KeyEvent): void {
  const menu = app.menu
  if (!menu) return
  if (key.name === "escape") {
    app.menu = undefined
    app.altActive = false
  } else if (key.name === "left") {
    menu.column = (menu.column + menus.length - 1) % menus.length
    menu.selected = 0
  } else if (key.name === "right") {
    menu.column = (menu.column + 1) % menus.length
    menu.selected = 0
  } else if (key.name === "up") {
    menu.selected = Math.max(0, menu.selected - 1)
  } else if (key.name === "down") {
    menu.selected = Math.min((menus[menu.column]?.actions.length ?? 1) - 1, menu.selected + 1)
  } else if (key.name === "return" || key.name === "enter") {
    const action = menus[menu.column]?.actions[menu.selected]
    app.menu = undefined
    app.altActive = false
    if (action) void app.dispatch(action)
  } else if (!key.ctrl && !key.shift) {
    const col = menuColumnByKey[key.name.toLowerCase()]
    if (col !== undefined) {
      menu.column = col
      menu.selected = 0
    }
  }
  app.changed()
}
async function treeKey(app: Application, key: KeyEvent): Promise<void> {
  const row = app.tree.rows[app.tree.selected]
  if (["up", "k"].includes(key.name)) app.tree.selected = Math.max(0, app.tree.selected - 1)
  else if (["down", "j"].includes(key.name)) app.tree.selected = Math.min(app.tree.rows.length - 1, app.tree.selected + 1)
  else if (row && ["return", "enter", "right"].includes(key.name)) { if (row.directory) await app.tree.toggle(row); else await app.openFile(row.path) }
  else if (row && key.name === "left") { if (row.directory && row.expanded) await app.tree.toggle(row); else { const parent = app.tree.rows.findLastIndex(candidate => candidate.directory && candidate.depth < row.depth && app.tree.rows.indexOf(candidate) < app.tree.selected); app.tree.selected = Math.max(0, parent) } }
  else if (key.name === "r" && row) app.prompt("Rename", row.name, value => renameTreeSelection(app, value))
  else if (key.name === "d") deleteTreeSelection(app)
  else if (key.name === "n") await app.dispatch("tree_new_file")
  else if (["y", "c"].includes(key.name) && row) { app.clipboard = row.path; app.clipboardWrite = `\x1b]52;c;${Buffer.from(row.path).toString("base64")}\x07` }
  app.changed()
}
