import { offsetAt, lineRanges } from "../core/document"
import type { Application } from "./application"
import type { KeyEvent } from "@opentui/core"

export function receiveVimKey(app: Application, key: KeyEvent): boolean {
  if (key.ctrl && key.name === "r" && app.vim !== "insert") { app.document.redo(); app.afterEdit(); return true }
  if (key.ctrl || key.meta) return false
  if (app.vim === "insert") return false
  const name = key.sequence || key.name, document = app.document
  const visual = app.vim === "visual" || app.vim === "visual-line"
  const move = (direction: Parameters<typeof document.move>[0]) => {
    document.move(direction, visual)
    if (app.vim === "visual-line") selectLines(app)
  }
  const motions: Record<string, Parameters<typeof document.move>[0]> = { h: "left", left: "left", l: "right", right: "right", j: "down", down: "down", k: "up", up: "up", "0": "line_start", "$": "line_end", G: "doc_end" }
  if (app.vimPending) {
    const operator = app.vimPending; app.vimPending = ""
    if (operator === "g" && name === "g") move("doc_start")
    else if (operator === "d" && name === "d") { selectLines(app); copySelection(app); document.remove(false) }
    else if (operator === "y" && name === "y") { const head = document.primary.head; selectLines(app); copySelection(app); document.select(head) }
    app.afterEdit(); return true
  }
  if (motions[name] || motions[key.name]) move(motions[name] ?? motions[key.name] ?? "left")
  else if (["w", "b", "e"].includes(name)) {
    const head = document.primary.head
    const words = [...document.text.matchAll(/[\p{L}\p{N}_]+|[^\s\p{L}\p{N}_]+/gu)]
    const target = name === "b" ? words.findLast(word => word.index < head)?.index ?? 0
      : name === "e" ? (words.find(word => word.index + word[0].length - 1 > head)?.index ?? document.text.length) + (words.find(word => word.index + word[0].length - 1 > head)?.[0].length ?? 1) - 1
      : words.find(word => word.index > head)?.index ?? document.text.length
    document.select(visual ? document.primary.anchor : target, target)
  } else if (["i", "a", "I", "A", "o", "O"].includes(name)) {
    if (name === "a") document.move("right")
    if (name === "I") document.move("line_start")
    if (name === "A") document.move("line_end")
    if (name === "o") { document.move("line_end"); document.insert("\n", false) }
    if (name === "O") { document.move("line_start"); const head = document.primary.head; document.insert("\n", false); document.select(head) }
    app.vim = "insert"
  } else if (name === "v" || name === "V") {
    app.vim = visual ? "normal" : name === "V" ? "visual-line" : "visual"
    app.vimLineAnchor = document.position.line
    if (app.vim === "visual-line") selectLines(app)
    else document.select(document.primary.head)
  } else if (name === "x" || (visual && name === "d")) { if (visual) copySelection(app); document.remove(false); app.vim = "normal" }
  else if (visual && name === "y") { copySelection(app); document.select(document.primary.head); app.vim = "normal" }
  else if (["d", "y", "g"].includes(name)) app.vimPending = name
  else if (name === "p" || name === "P") {
    if (app.clipboard.endsWith("\n")) { document.move(name === "p" ? "line_end" : "line_start"); if (name === "p") document.insert("\n" + app.clipboard.replace(/\n$/, ""), false); else document.insert(app.clipboard, false) }
    else { if (name === "p") document.move("right"); document.insert(app.clipboard, false) }
  } else if (name === "u") document.undo()
  else if (name === ":") app.prompt("Vim command", "", async value => {
    const [command, ...args] = value.trim().split(/\s+/)
    if (command === "w") { if (args.length) await app.save(document, args.join(" ")); else await app.dispatch("save") }
    else if (command === "wq") { await app.dispatch("save"); if (!app.document.dirty) await app.dispatch("quit") }
    else if (command === "q") await app.dispatch("quit")
    else if (command === "q!") { app.quit = true; app.changed() }
    else if (command === "tasks") await app.dispatch("run_tasks")
    else if (command === "health") await app.dispatch("health_check")
    else if (command === "normal") app.vim = "normal"
    else if (command === "insert") app.vim = "insert"
    else throw new Error(`Unsupported Vim command: ${value}`)
  })
  else if (name === "/") app.openSearch(false)
  else if (name === "n") app.findNext(1)
  else if (name === "N") app.findNext(-1)
  // Normal mode must consume unknown printable keys, never insert them.
  app.afterEdit(); return true
}
function selectLines(app: Application): void {
  const document = app.document
  const current = document.position.line
  const anchor = app.vim === "visual-line" ? app.vimLineAnchor : current
  const start = offsetAt(document.text, Math.min(anchor, current), 0)
  const last = Math.max(anchor, current)
  const end = last + 1 < lineRanges(document.text).length ? offsetAt(document.text, last + 1, 0) : document.text.length
  document.select(start, end)
}
function copySelection(app: Application): void {
  app.clipboard = app.document.selectedText
  app.clipboardWrite = `\x1b]52;c;${Buffer.from(app.clipboard).toString("base64")}\x07`
}
