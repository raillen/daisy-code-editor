// Quebra visual de texto em linhas de largura fixa.
//
// Nasceu no editor (soft wrap) e é pura: sem renderer, sem estado. O painel
// de agentes usa a mesma função para saber quantas linhas um transcript ocupa
// — o app precisa do total para prender o scroll, e o desenho usa as mesmas
// linhas. Uma função, dois consumidores, nenhuma conta duplicada.
import { lineRanges } from "./document"

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

/**
 * Linhas já quebradas, prontas para desenhar (sem as coordenadas de origem).
 *
 * Quebra **por palavra**: o transcript de um agente é prosa, e cortar "called"
 * em "c/alled" custa a leitura. Palavra maior que a largura (URL, identificador
 * longo) cai na quebra por caractere. A base é o `visualRows` sem wrap: ele já
 * expande tabs e remove controles, então a medida segue correta para
 * caracteres largos.
 */
export function wrapText(text: string, width: number, tabSize = 4): string[] {
  const limit = Math.max(1, width)
  const rows: string[] = []
  for (const source of text.split("\n")) {
    const expanded = visualRows(source, limit, false, tabSize).map(row => row.cells.map(cell => cell.text).join("")).join("")
    let row = ""
    for (const word of expanded.split(" ")) {
      const candidate = row ? `${row} ${word}` : word
      if (Bun.stringWidth(candidate) <= limit) { row = candidate; continue }
      if (row) rows.push(row)
      row = ""
      for (const char of word) {
        if (row && Bun.stringWidth(row + char) > limit) { rows.push(row); row = "" }
        row += char
      }
    }
    rows.push(row)
  }
  return rows
}
