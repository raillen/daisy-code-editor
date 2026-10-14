import { basename } from "node:path"

export type Selection = { anchor: number; head: number }
export type TextEdit = { start: number; end: number; text: string }
type Snapshot = { text: string; selections: Selection[]; label: string }
export type Direction = "left" | "right" | "up" | "down" | "line_start" | "line_end" | "doc_start" | "doc_end"
const segmenter = new Intl.Segmenter(undefined, { granularity: "grapheme" })

let lastBoundaryText: string | undefined
let lastBoundaries: number[] = []

export function boundaries(text: string): number[] {
  if (text === lastBoundaryText) return lastBoundaries
  const points = [...segmenter.segment(text)].map(segment => segment.index).concat(text.length)
  lastBoundaryText = text
  lastBoundaries = points
  return points
}

let lastRangeText: string | undefined
let lastRanges: { start: number; end: number; next: number }[] = []

export function lineRanges(text: string): { start: number; end: number; next: number }[] {
  if (text === lastRangeText) return lastRanges
  const lines: { start: number; end: number; next: number }[] = []
  let start = 0
  for (const match of text.matchAll(/\r\n|[\n\r\u0085\u2028\u2029]/g)) {
    const next = match.index + match[0].length
    lines.push({ start, end: match.index, next }); start = next
  }
  lines.push({ start, end: text.length, next: text.length })
  lastRangeText = text
  lastRanges = lines
  return lines
}

export function binarySearchIndex(array: readonly number[], target: number): number {
  let low = 0, high = array.length - 1
  while (low <= high) {
    const mid = (low + high) >> 1
    const val = array[mid]!
    if (val === target) return mid
    if (val < target) low = mid + 1
    else high = mid - 1
  }
  return -1
}

export function binarySearchFloor(array: readonly number[], target: number): number {
  let low = 0, high = array.length - 1, best = 0
  while (low <= high) {
    const mid = (low + high) >> 1
    if (array[mid]! <= target) {
      best = array[mid]!
      low = mid + 1
    } else {
      high = mid - 1
    }
  }
  return best
}

export function positionAt(text: string, offset: number): { line: number; character: number } {
  const ranges = lineRanges(text)
  if (!ranges.length) return { line: 0, character: 0 }
  let low = 0, high = ranges.length - 1, line = 0
  while (low <= high) {
    const mid = (low + high) >> 1
    if (ranges[mid]!.start <= offset) {
      line = mid
      low = mid + 1
    } else {
      high = mid - 1
    }
  }
  const range = ranges[line]
  return { line, character: Math.min(offset, range?.end ?? offset) - (range?.start ?? 0) }
}

export function offsetAt(text: string, line: number, character: number): number {
  const lines = lineRanges(text)
  const row = Math.max(0, Math.min(line, lines.length - 1))
  const range = lines[row] ?? { start: 0, end: 0 }
  return range.start + Math.max(0, Math.min(character, range.end - range.start))
}

export class Document {
  text: string
  path: string | undefined
  selections: Selection[] = [{ anchor: 0, head: 0 }]
  version = 0
  indent: { tabSize: number; insertSpaces: boolean } | undefined
  savedText: string
  readonly history: Snapshot[] = []
  private future: Snapshot[] = []
  private historyBytes = 0
  private grouping = false
  private preferredColumn: number | undefined

  constructor(readonly id: number, text = "", path?: string) {
    this.text = this.savedText = text
    this.path = path
  }

  get dirty(): boolean { return this.text !== this.savedText }
  get title(): string { return this.path ? basename(this.path) : "untitled" }
  get primary(): Selection { return this.selections[0] ?? { anchor: 0, head: 0 } }
  get position(): { line: number; character: number } { return positionAt(this.text, this.primary.head) }
  get selectedText(): string {
    return this.selections.map(({ anchor, head }) => this.text.slice(Math.min(anchor, head), Math.max(anchor, head))).join("\n")
  }

  snapshot(label: string): Snapshot {
    return { text: this.text, selections: this.selections.map(selection => ({ ...selection })), label }
  }

  private remember(label: string, group: boolean): void {
    if (!group || !this.grouping) {
      const snapshot = this.snapshot(label)
      this.history.push(snapshot)
      this.historyBytes += snapshot.text.length * 2
    }
    // Bound history by bytes, not only entry count: large files must not exhaust memory.
    while (this.history.length > 1 && this.historyBytes > 32 * 1024 * 1024) {
      const removed = this.history.shift()
      if (removed) this.historyBytes = Math.max(0, this.historyBytes - removed.text.length * 2)
    }
    this.future = []
    this.grouping = group
    this.preferredColumn = undefined
  }

  commit(): void { this.grouping = false }

  select(anchor: number, head = anchor, add = false): void {
    const points = boundaries(this.text)
    const clamp = (value: number) => binarySearchFloor(points, Math.max(0, Math.min(value, this.text.length)))
    const selection = { anchor: clamp(anchor), head: clamp(head) }
    this.selections = add ? [...this.selections, selection] : [selection]
    this.normalizeSelections()
    this.preferredColumn = undefined
    this.commit()
  }

  private normalizeSelections(): void {
    const sorted = this.selections.map(selection => ({
      start: Math.min(selection.anchor, selection.head), end: Math.max(selection.anchor, selection.head), selection,
    })).sort((left, right) => left.start - right.start)
    const merged: Selection[] = []
    for (const range of sorted) {
      const previous = merged.at(-1)
      if (previous && range.start <= Math.max(previous.anchor, previous.head)) {
        previous.anchor = Math.min(previous.anchor, previous.head, range.start)
        previous.head = Math.max(previous.head, range.end)
      } else merged.push({ ...range.selection })
    }
    this.selections = merged
  }

  insert(text: string, group = true): void {
    if (!text) return
    this.remember("insert", group)
    let shift = 0
    let result = this.text
    const selections: Selection[] = []
    for (const { anchor, head } of this.selections) {
      const start = Math.min(anchor, head) + shift
      const end = Math.max(anchor, head) + shift
      result = result.slice(0, start) + text + result.slice(end)
      const caret = start + text.length
      selections.push({ anchor: caret, head: caret })
      shift += text.length - (end - start)
    }
    this.text = result
    this.selections = selections
    this.version++
    if (/[\n\r\u0085\u2028\u2029]/.test(text)) this.commit()
  }

  remove(backward: boolean): void {
    const points = boundaries(this.text)
    const originalSelections = this.selections.map(selection => ({ ...selection }))
    this.selections = this.selections.map(({ anchor, head }) => {
      if (anchor !== head) return { anchor, head }
      const index = binarySearchIndex(points, head)
      return backward ? { anchor: points[Math.max(0, index - 1)] ?? 0, head }
        : { anchor: head, head: points[Math.min(points.length - 1, index + 1)] ?? head }
    })
    this.normalizeSelections()
    if (this.selections.every(selection => selection.anchor === selection.head)) return
    const deletionSelections = this.selections
    this.selections = originalSelections
    this.remember(backward ? "backspace" : "delete", true)
    this.selections = deletionSelections
    let shift = 0
    this.selections = this.selections.map(({ anchor, head }) => {
      const start = Math.min(anchor, head) - shift
      const end = Math.max(anchor, head) - shift
      this.text = this.text.slice(0, start) + this.text.slice(end)
      shift += end - start
      return { anchor: start, head: start }
    })
    this.version++
  }

  applyEdits(edits: readonly TextEdit[], label = "replace"): void {
    const sorted = [...edits].sort((left, right) => left.start - right.start)
    let previousEnd = 0
    for (const edit of sorted) {
      if (edit.start < previousEnd || edit.start < 0 || edit.end < edit.start || edit.end > this.text.length) throw new Error("Invalid or overlapping text edits")
      previousEnd = edit.end
    }
    if (!sorted.length) return
    this.remember(label, false)

    // Preserva e mapeia seleções através das edições aplicadas
    const newSelections = this.selections.map(selection => {
      let anchor = selection.anchor
      let head = selection.head
      for (const edit of sorted) {
        const delta = edit.text.length - (edit.end - edit.start)
        if (selection.anchor >= edit.end) anchor += delta
        else if (selection.anchor > edit.start) anchor = edit.start + edit.text.length
        if (selection.head >= edit.end) head += delta
        else if (selection.head > edit.start) head = edit.start + edit.text.length
      }
      return { anchor, head }
    })

    for (const edit of sorted.toReversed()) this.text = this.text.slice(0, edit.start) + edit.text + this.text.slice(edit.end)
    this.selections = newSelections.length ? newSelections : [{ anchor: 0, head: 0 }]
    this.version++
  }

  undo(): void {
    this.commit()
    const previous = this.history.pop()
    if (!previous) return
    this.historyBytes = Math.max(0, this.historyBytes - previous.text.length * 2)
    this.future.push(this.snapshot(previous.label))
    this.restore(previous)
  }

  redo(): void {
    const next = this.future.pop()
    if (!next) return
    const current = this.snapshot(next.label)
    this.history.push(current)
    this.historyBytes += current.text.length * 2
    this.restore(next)
  }

  private restore(snapshot: Snapshot): void {
    this.text = snapshot.text
    this.selections = snapshot.selections.map(selection => ({ ...selection }))
    this.version++
    this.commit()
  }

  move(direction: Direction, extend = false, count = 1): void {
    const points = boundaries(this.text)
    const vertical = direction === "up" || direction === "down"
    const targetColumn = this.preferredColumn ?? this.position.character
    this.selections = this.selections.map(({ anchor, head }) => {
      const position = positionAt(this.text, head)
      let target = head
      if (direction === "left") {
        target = (head > 0 && this.text.charCodeAt(head - 1) < 128 && (head >= this.text.length || this.text.charCodeAt(head) < 128))
          ? Math.max(0, head - count)
          : (points[Math.max(0, binarySearchIndex(points, head) - count)] ?? 0)
      } else if (direction === "right") {
        target = (head < this.text.length && this.text.charCodeAt(head) < 128 && (head + 1 >= this.text.length || this.text.charCodeAt(head + 1) < 128))
          ? Math.min(this.text.length, head + count)
          : (points[Math.min(points.length - 1, binarySearchIndex(points, head) + count)] ?? head)
      } else if (vertical) {
        target = offsetAt(this.text, position.line + (direction === "up" ? -count : count), targetColumn)
      } else if (direction === "line_start") {
        target = offsetAt(this.text, position.line, 0)
      } else if (direction === "line_end") {
        target = offsetAt(this.text, position.line, Number.MAX_SAFE_INTEGER)
      } else if (direction === "doc_start") {
        target = 0
      } else if (direction === "doc_end") {
        target = this.text.length
      }
      if (target < this.text.length && this.text.charCodeAt(target) >= 128) {
        target = binarySearchFloor(points, target)
      }
      return { anchor: extend ? anchor : target, head: target }
    })
    this.preferredColumn = vertical ? targetColumn : undefined
    this.normalizeSelections()
    this.commit()
  }

  addCursor(delta: number): void {
    const last = this.selections.at(delta > 0 ? -1 : 0) ?? this.primary
    const position = positionAt(this.text, last.head)
    this.select(offsetAt(this.text, position.line + delta, position.character), undefined, true)
  }

  clearExtraCursors(): void { this.selections = [this.primary]; this.commit() }

  toggleComment(marker: string | { open: string; close?: string }): void {
    const delimiters = typeof marker === "string" ? { open: marker } : marker
    const open = delimiters.open.trim()
    const close = delimiters.close?.trim() ?? ""
    const lines = lineRanges(this.text).map(range => this.text.slice(range.start, range.end))
    const start = positionAt(this.text, Math.min(this.primary.anchor, this.primary.head)).line
    const end = positionAt(this.text, Math.max(this.primary.anchor, this.primary.head)).line

    if (!close) {
      const uncomment = lines.slice(start, end + 1).every(line => !line.trim() || line.trimStart().startsWith(open))
      const edits: TextEdit[] = []
      for (let row = start; row <= end; row++) {
        const line = lines[row] ?? ""
        const indent = line.length - line.trimStart().length
        const at = offsetAt(this.text, row, indent)
        if (uncomment && line.trimStart().startsWith(open)) edits.push({ start: at, end: at + open.length + (line[indent + open.length] === " " ? 1 : 0), text: "" })
        else if (!uncomment) edits.push({ start: at, end: at, text: open + " " })
      }
      this.applyEdits(edits, "comment")
    } else {
      const nonEmpties = lines.slice(start, end + 1).filter(line => line.trim().length > 0)
      const allCommented = nonEmpties.length > 0 && nonEmpties.every(line => {
        const trimmed = line.trim()
        return trimmed.startsWith(open) && trimmed.endsWith(close)
      })
      const edits: TextEdit[] = []
      for (let row = start; row <= end; row++) {
        const line = lines[row] ?? ""
        if (!line.trim()) continue
        const indent = line.length - line.trimStart().length
        const trimmed = line.trim()
        const lineStart = offsetAt(this.text, row, 0)
        const lineEnd = offsetAt(this.text, row, line.length)
        if (allCommented) {
          let inner = trimmed
          if (inner.startsWith(open)) inner = inner.slice(open.length)
          if (inner.startsWith(" ")) inner = inner.slice(1)
          if (inner.endsWith(close)) inner = inner.slice(0, -close.length)
          if (inner.endsWith(" ")) inner = inner.slice(0, -1)
          edits.push({ start: lineStart, end: lineEnd, text: " ".repeat(indent) + inner })
        } else {
          edits.push({ start: lineStart, end: lineEnd, text: " ".repeat(indent) + `${open} ${trimmed} ${close}` })
        }
      }
      if (edits.length) this.applyEdits(edits, "comment")
    }
  }
}
