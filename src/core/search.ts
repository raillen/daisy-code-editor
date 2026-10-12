import type { TextEdit } from "./document"
export type SearchOptions = { caseSensitive: boolean; ignoreAccents: boolean; wholeWord: boolean; regex: boolean }
export const searchDefaults = (): SearchOptions => ({ caseSensitive: false, ignoreAccents: false, wholeWord: false, regex: false })
export type Match = { start: number; end: number; groups: string[] }

function fold(text: string, accents: boolean): { text: string; starts: number[]; ends: number[] } {
  const starts: number[] = [], ends: number[] = []
  let result = "", offset = 0
  const segmenter = new Intl.Segmenter(undefined, { granularity: "grapheme" })
  for (const segment of segmenter.segment(text)) {
    const value = accents ? segment.segment.normalize("NFD").replace(/\p{M}/gu, "") : segment.segment
    for (let index = 0; index < value.length; index++) { starts.push(offset); ends.push(offset + segment.segment.length) }
    result += value
    offset += segment.segment.length
  }
  return { text: result, starts, ends }
}

export function findMatches(text: string, query: string, options: SearchOptions): Match[] {
  if (!query) return []
  const matches: Match[] = []
  const isWord = (character: string) => /[\p{L}\p{N}_]/u.test(character)

  if (!options.ignoreAccents) {
    const pattern = options.regex ? query : query.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
    const expression = new RegExp(pattern, "gu" + (options.caseSensitive ? "" : "i"))
    for (const match of text.matchAll(expression)) {
      if (!match[0].length) continue
      const start = match.index
      const end = start + match[0].length
      if (options.wholeWord && (isWord(text.slice(0, start).at(-1) ?? "") || isWord(text.slice(end)[0] ?? ""))) continue
      matches.push({ start, end, groups: [...match] })
      if (matches.length >= 100000) throw new Error("Search exceeds 100,000 matches")
    }
    return matches
  }

  const folded = fold(text, true)
  const normalized = fold(query, true).text
  const pattern = options.regex ? normalized : normalized.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
  const expression = new RegExp(pattern, "gu" + (options.caseSensitive ? "" : "i"))
  for (const match of folded.text.matchAll(expression)) {
    if (!match[0].length) continue
    const start = folded.starts[match.index] ?? match.index
    const end = folded.ends[match.index + match[0].length - 1] ?? start
    if (options.wholeWord && (isWord(text.slice(0, start).at(-1) ?? "") || isWord(text.slice(end)[0] ?? ""))) continue
    matches.push({ start, end, groups: [...match] })
    if (matches.length >= 100000) throw new Error("Search exceeds 100,000 matches")
  }
  return matches
}

export function replacementEdits(matches: readonly Match[], replacement: string, regex: boolean): TextEdit[] {
  return matches.map(match => ({ start: match.start, end: match.end, text: regex ? replacement.replace(/\$\$|\$&|\$(\d{1,2})/g, (token: string, group: string | undefined) => token === "$$" ? "$" : token === "$&" ? match.groups[0] ?? "" : match.groups[Number(group)] ?? "") : replacement }))
}

export function fuzzyScore(value: string, query: string): number {
  if (!query) return 0
  let cursor = 0, score = 0, previous = -2
  const normalized = value.toLowerCase()
  const normalizedQuery = query.toLowerCase()
  for (let i = 0; i < normalizedQuery.length; i++) {
    const character = normalizedQuery[i]!
    const index = normalized.indexOf(character, cursor)
    if (index < 0) return -Infinity
    score += 10 + (index === previous + 1 ? 12 : 0) + (index === 0 || "/_- .".includes(value[index - 1] ?? "") ? 15 : 0) - index * 0.1
    cursor = index + 1
    previous = index
  }
  return score - value.length * 0.01
}
