import { BoxRenderable, TextAttributes, RGBA, type OptimizedBuffer, type MouseEvent } from "@opentui/core"
import { createMemo } from "solid-js"
import type { Application, Pane } from "../app/application"
import { lineRanges } from "../core/document"
import { themeOf, resolveColor } from "./theme"
import { selectedTheme } from "../services/preferences"

type Cell = { text: string; start: number; end: number; column: number; width: number }
export type VisualRow = { line: number; start: number; end: number; first: boolean; cells: Cell[]; width: number }
const segmenter = new Intl.Segmenter(undefined, { granularity: "grapheme" })
export function visualRows(text: string, width: number, wrap: boolean, tabSize: number): VisualRow[] {
  const rows: VisualRow[] = []
  for (const [line, range] of lineRanges(text).entries()) {
    const offset = range.start, value = text.slice(range.start, range.end)
    let cells: Cell[] = [], column = 0, start = offset, first = true
    const flush = () => { rows.push({ line, start, end: cells.at(-1)?.end ?? start, first, cells, width: column }); cells = []; column = 0; first = false }
    const hasComplexChars = /[^\x20-\x7e\t]/.test(value)
    if (!hasComplexChars) {
      for (let i = 0; i < value.length; i++) {
        const at = offset + i
        const char = value[i]!
        const glyph = char === "\t" ? " ".repeat(tabSize - (column % tabSize)) : char
        const size = glyph.length
        if (wrap && column > 0 && column + size > width) { flush(); start = at }
        cells.push({ text: glyph, start: at, end: at + 1, column, width: size })
        column += size
      }
    } else {
      for (const segment of segmenter.segment(value)) {
        const at = offset + segment.index
        const glyph = segment.segment === "\t" ? " ".repeat(tabSize - column % tabSize) : /[\x00-\x1f\x7f]/.test(segment.segment) ? "" : segment.segment
        const size = Bun.stringWidth(glyph)
        if (wrap && column > 0 && column + size > width) { flush(); start = at }
        cells.push({ text: glyph, start: at, end: at + segment.segment.length, column, width: size })
        column += size
      }
    }
    flush()
  }
  return rows
}
function caretRow(rows: VisualRow[], offset: number): number {
  const next = rows.findIndex(row => offset >= row.start && offset < row.end)
  return next >= 0 ? next : rows.findLastIndex(row => offset >= row.start && offset <= row.end)
}

export function EditorPane(props: { app: Application; pane: Pane; index: number; revision: () => number }) {
  let box: BoxRenderable | undefined
  let dragAnchor = 0
  let lastClick = { offset: -1, time: 0, count: 0 }
  let lastCaret = -1
  const document = () => props.app.documents.find(document => document.id === props.pane.documentId) ?? props.app.document
  const gutter = () => props.app.config.show_line_numbers ? Math.max(3, String(lineRanges(document().text).length).length) + 2 : 1
  const theme = createMemo(() => { props.revision(); const definition = selectedTheme(props.app.config, props.app.preferences); return themeOf({ ...props.app.config, ui: { ...definition.ui, ...props.app.config.ui }, tokens: { ...definition.tokens, ...props.app.config.tokens } }) })
  const rows = createMemo(() => { props.revision(); return visualRows(document().text, Math.max(1, (box?.width ?? 60) - gutter() - 2), props.app.config.soft_wrap, document().indent?.tabSize ?? props.app.config.editor.tab_size) })

  function paint(buffer: OptimizedBuffer): void {
    if (!box) return
    const app = props.app, doc = document(), definition = selectedTheme(app.config, app.preferences), currentTheme = theme(), visual = rows(), inset = gutter()
    const focused = app.focus === "editor" && app.focusedPane === props.index && !app.overlay && !app.menu
    const caret = doc.primary.head, caretIndex = Math.max(0, caretRow(visual, caret))
    if (caret !== lastCaret) {
      if (caretIndex < props.pane.scroll) props.pane.scroll = caretIndex
      if (caretIndex >= props.pane.scroll + box.height) props.pane.scroll = Math.max(0, caretIndex - box.height + 1)
      lastCaret = caret
    }
    const rowForCaret = visual[caretIndex]
    const caretColumn = rowForCaret?.cells.find(cell => cell.start === caret)?.column ?? rowForCaret?.width ?? 0
    if (!app.config.soft_wrap && focused) {
      const available = Math.max(1, box.width - inset - 2)
      if (caretColumn < props.pane.scrollX) props.pane.scrollX = caretColumn
      if (caretColumn >= props.pane.scrollX + available) props.pane.scrollX = caretColumn - available + 1
    } else props.pane.scrollX = 0
    if (focused) {
      const screenRow = caretIndex - props.pane.scroll
      const screenCol = inset + caretColumn - props.pane.scrollX
      app.cursorScreen = {
        x: box.x + Math.max(0, screenCol),
        y: box.y + screenRow,
        boxWidth: box.width,
        boxHeight: box.height,
        boxX: box.x,
        boxY: box.y,
      }
    }
    const highlighting = app.highlights.get(doc.id)
    const tokens = highlighting?.version === doc.version ? highlighting.tokens : []
    let tokenIndex = 0
    const firstRow = visual[props.pane.scroll]
    const lastRow = visual[Math.min(visual.length - 1, props.pane.scroll + box.height - 1)]
    const visibleStart = firstRow?.start ?? 0
    const visibleEnd = lastRow?.end ?? doc.text.length
    const visibleMatches = (doc.id === app.activeId && app.matches.length > 0)
      ? app.matches.filter(match => match.start < visibleEnd && match.end > visibleStart)
      : []
    const gutterMap = app.gitGutters.get(doc.path ?? "")
    for (let screenRow = 0; screenRow < box.height; screenRow++) {
      const row = visual[props.pane.scroll + screenRow]
      if (!row) continue
      const y = box.y + screenRow
      if (inset > 1) {
        const linePart = row.first ? String(row.line + 1).padStart(inset - 2) : " ".repeat(inset - 2)
        const caretPart = caretIndex === props.pane.scroll + screenRow && focused ? ">" : " "
        const marker = row.first ? gutterMap?.get(row.line) : undefined
        const signPart = marker === "added" ? "+" : marker === "modified" ? "~" : marker === "deleted" ? "-" : " "
        buffer.drawText(linePart, box.x, y, currentTheme.muted, currentTheme.surface)
        buffer.drawText(caretPart, box.x + inset - 2, y, currentTheme.cursor, currentTheme.surface)
        const markerColor = marker === "added" ? RGBA.fromHex("#3fb950") : marker === "modified" ? RGBA.fromHex("#d29922") : RGBA.fromHex("#f85149")
        buffer.drawText(signPart, box.x + inset - 1, y, currentTheme.noColor ? currentTheme.muted : marker ? markerColor : currentTheme.muted, currentTheme.surface)
      }
      for (const cell of row.cells) {
        const x = box.x + inset + cell.column - props.pane.scrollX
        if (x < box.x + inset || x + cell.width > box.x + box.width - 1) continue
        while (tokens[tokenIndex] && (tokens[tokenIndex]?.end ?? 0) <= cell.start) tokenIndex++
        const token = tokens[tokenIndex]
        const syntax = token && token.start <= cell.start && token.end > cell.start ? token : undefined
        const selected = doc.selections.some(selection => cell.start < Math.max(selection.anchor, selection.head) && cell.end > Math.min(selection.anchor, selection.head))
        const cursor = focused && doc.selections.some(selection => selection.head === cell.start)
        const match = visibleMatches.length > 0 && visibleMatches.some(match => match.start < cell.end && match.end > cell.start)
        const override = syntax?.role ? app.config.syntax[syntax.role] ?? definition.syntax[syntax.role] : undefined
        const foreground = currentTheme.noColor ? currentTheme.text : override ? resolveColor(override, syntax?.color ?? "#ffffff") : syntax?.color ? RGBA.fromHex(syntax.color) : currentTheme.text
        const font = syntax?.fontStyle ?? 0
        const attributes = (font & 1 ? TextAttributes.ITALIC : 0) | (font & 2 ? TextAttributes.BOLD : 0) | (font & 4 ? TextAttributes.UNDERLINE : 0)
        buffer.drawText(cell.text, x, y, cursor ? currentTheme.surface : foreground, cursor ? currentTheme.cursor : selected || match ? currentTheme.selection : currentTheme.surface, attributes | (selected && currentTheme.noColor ? TextAttributes.INVERSE : 0))
      }
      if (focused && doc.selections.some(selection => selection.head === row.end) && (!visual[props.pane.scroll + screenRow + 1] || visual[props.pane.scroll + screenRow + 1]?.line !== row.line)) {
        const x = box.x + inset + row.width - props.pane.scrollX
        if (x < box.x + box.width - 1) buffer.drawText(" ", x, y, currentTheme.surface, currentTheme.cursor, currentTheme.noColor ? TextAttributes.INVERSE : 0)
      }
    }
    if (visual.length > box.height && box.height > 0) {
      const scrollbarX = box.x + box.width - 1
      const trackHeight = box.height
      const thumbHeight = Math.max(1, Math.round((trackHeight / visual.length) * trackHeight))
      const maxScroll = Math.max(1, visual.length - trackHeight)
      const currentScroll = Math.max(0, Math.min(maxScroll, props.pane.scroll))
      const thumbY = box.y + Math.min(trackHeight - thumbHeight, Math.round((currentScroll / maxScroll) * (trackHeight - thumbHeight)))
      for (let r = 0; r < trackHeight; r++) {
        const y = box.y + r
        const isThumb = r >= (thumbY - box.y) && r < (thumbY - box.y) + thumbHeight
        const char = isThumb ? "█" : "│"
        const color = isThumb ? (currentTheme.noColor ? currentTheme.text : currentTheme.border) : currentTheme.muted
        buffer.drawText(char, scrollbarX, y, color, currentTheme.surface)
      }
    }
  }
  function offsetFor(event: MouseEvent): number {
    if (!box) return 0
    const row = rows()[Math.max(0, Math.min(rows().length - 1, event.y - box.y + props.pane.scroll))]
    if (!row) return document().text.length
    const column = Math.max(0, event.x - box.x - gutter() + props.pane.scrollX)
    return row.cells.find(cell => column < cell.column + cell.width)?.start ?? row.end
  }
  function down(event: MouseEvent): void {
    if (!props.app.config.mouse || props.app.overlay || props.app.menu) return
    props.app.cancelCompletion()
    props.app.focusPane(props.index)
    if (box && rows().length > box.height && event.x >= box.x + box.width - 1) {
      const relativeY = event.y - box.y
      const ratio = Math.max(0, Math.min(1, relativeY / Math.max(1, box.height - 1)))
      props.pane.scroll = Math.round(ratio * (rows().length - box.height))
      props.app.changed()
      event.preventDefault()
      return
    }
    const offset = offsetFor(event), doc = document()
    const now = performance.now()
    lastClick = { offset, time: now, count: lastClick.offset === offset && now - lastClick.time < 400 ? lastClick.count + 1 : 1 }
    if (lastClick.count === 2) {
      const before = doc.text.slice(0, offset).match(/[\p{L}\p{N}_]+$/u)?.[0] ?? ""
      const after = doc.text.slice(offset).match(/^[\p{L}\p{N}_]+/u)?.[0] ?? ""
      doc.select(offset - before.length, offset + after.length)
    } else if (lastClick.count >= 3) {
      const start = doc.text.lastIndexOf("\n", offset - 1) + 1
      const end = doc.text.indexOf("\n", offset)
      doc.select(start, end < 0 ? doc.text.length : end + 1)
    } else doc.select(event.modifiers.shift ? doc.primary.anchor : offset, offset, event.modifiers.alt)
    dragAnchor = doc.primary.anchor
    props.app.changed()
    event.preventDefault()
  }
  return <box ref={value => { box = value }} id={`editor-${props.index}`} flexGrow={1} minWidth={1} minHeight={1} overflow="hidden"
    backgroundColor={theme().surface} renderAfter={paint}
    onMouseDown={down} onSizeChange={() => props.app.changed()}
    onMouseDrag={event => {
      if (!props.app.overlay && !props.app.menu) {
        if (box && rows().length > box.height && event.x >= box.x + box.width - 1) {
          const relativeY = event.y - box.y
          const ratio = Math.max(0, Math.min(1, relativeY / Math.max(1, box.height - 1)))
          props.pane.scroll = Math.round(ratio * (rows().length - box.height))
          props.app.changed()
          return
        }
        document().select(dragAnchor, offsetFor(event))
        props.app.changed()
      }
    }}
    onMouseScroll={event => { if (props.app.overlay || props.app.menu) return; props.pane.scroll = Math.max(0, Math.min(rows().length - 1, props.pane.scroll + (event.scroll?.direction === "up" ? -3 : 3))); props.app.changed() }} />
}
