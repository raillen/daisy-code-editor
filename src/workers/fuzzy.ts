import { fuzzyScore } from "../core/search"

export type FuzzyWorkerRequest = {
  id: number
  type: "fuzzy"
  query: string
  entries: { index: number; target: string }[]
  limit: number
}

export type FuzzyWorkerResponse = {
  id: number
  type: "fuzzy:result"
  items: { index: number; score: number }[]
}

self.onmessage = (event: MessageEvent<FuzzyWorkerRequest>) => {
  const data = event.data
  if (!data || data.type !== "fuzzy") return
  const { id, query, entries, limit } = data
  if (!query) {
    const items = entries.slice(0, limit > 0 ? limit : entries.length).map(e => ({ index: e.index, score: 0 }))
    self.postMessage({ id, type: "fuzzy:result", items } satisfies FuzzyWorkerResponse)
    return
  }
  const scored: { index: number; score: number }[] = []
  for (let i = 0; i < entries.length; i++) {
    const entry = entries[i]
    if (!entry) continue
    const score = fuzzyScore(entry.target, query)
    if (score > -Infinity) {
      scored.push({ index: entry.index, score })
    }
  }
  scored.sort((left, right) => right.score - left.score)
  const items = limit > 0 ? scored.slice(0, limit) : scored
  self.postMessage({ id, type: "fuzzy:result", items } satisfies FuzzyWorkerResponse)
}
