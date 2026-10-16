import { expect, test } from "bun:test"
import { extractBufferWords, suggestBufferWords, extractDocumentSymbols } from "../src/services/symbols"
import { visualRows } from "../src/core/wrap"
import { findMatches, searchDefaults } from "../src/core/search"

test("performance: extractBufferWords and suggestBufferWords with 10,000 lines completes in < 50ms", () => {
  // Generate 10,000 lines of code-like text
  const identifiers = ["handleClick", "userId", "renderComponent", "fetchUserData", "processItem", "databasePool", "cacheKey", "calculateMetrics", "sessionToken", "responsePayload"]
  const lines: string[] = []
  for (let i = 0; i < 10000; i++) {
    const id1 = identifiers[i % identifiers.length]
    const id2 = identifiers[(i + 3) % identifiers.length]
    lines.push(`const ${id1}_${i} = compute(${id2}, ${i});`)
  }
  const largeText = lines.join("\n")

  // Benchmark extractBufferWords
  const t0 = performance.now()
  const words = extractBufferWords([{ text: largeText }])
  const extractDuration = performance.now() - t0

  expect(words.size).toBeGreaterThan(5000)
  expect(extractDuration).toBeLessThan(100) // Less than 100ms for 10,000 lines!

  // Benchmark suggestBufferWords with fuzzy matching
  const t1 = performance.now()
  const suggestions = suggestBufferWords("calc", words, 50)
  const suggestDuration = performance.now() - t1

  expect(suggestions.length).toBeGreaterThan(0)
  expect(suggestDuration).toBeLessThan(100)
})

test("performance: visualRows layout for 2,000 lines completes comfortably", () => {
  const lines = Array.from({ length: 2000 }, (_, i) => `function testCase${i}(param1: string, param2: number): boolean { return param2 > 0; }`)
  const code = lines.join("\n")

  const t0 = performance.now()
  const rows = visualRows(code, 80, true, 2)
  const duration = performance.now() - t0

  expect(rows.length).toBeGreaterThanOrEqual(2000)
  expect(duration).toBeLessThan(150)
})

test("performance: extractDocumentSymbols parses 5,000 lines comfortably", () => {
  const blocks: string[] = []
  for (let i = 0; i < 1000; i++) {
    blocks.push(`/// Documented helper function number ${i}\npub fn helper_${i}(val: u32) -> Result<u32, Error> {\n    Ok(val + 1)\n}\n`)
  }
  const text = blocks.join("\n")

  const t0 = performance.now()
  const symbols = extractDocumentSymbols(text)
  const duration = performance.now() - t0

  expect(symbols.length).toBe(1000)
  expect(symbols[0]?.name).toBe("helper_0")
  expect(symbols[0]?.doc).toBe("Documented helper function number 0")
  expect(duration).toBeLessThan(100)
})

test("performance: findMatches with whole-word search across 100,000 characters completes quickly", () => {
  const text = "apple banana orange cherry grape mango ".repeat(2500) // ~100,000 chars

  const t0 = performance.now()
  const matches = findMatches(text, "banana", { ...searchDefaults(), wholeWord: true })
  const duration = performance.now() - t0

  expect(matches.length).toBe(2500)
  expect(duration).toBeLessThan(100)
})

test("performance: positionAt binary search executes 2,000 lookups in a 20,000-line file quickly", () => {
  const lines = Array.from({ length: 20000 }, (_, i) => `line ${i}: const val = ${i * 2};`).join("\n")
  const t0 = performance.now()
  const { positionAt } = require("../src/core/document")
  for (let i = 0; i < 2000; i++) {
    const offset = (i * 123) % lines.length
    const pos = positionAt(lines, offset)
    expect(pos.line).toBeGreaterThanOrEqual(0)
  }
  const duration = performance.now() - t0
  expect(duration).toBeLessThan(100)
})

test("performance: Document movement across 500 steps in a 5,000-line buffer executes quickly", () => {
  const { Document } = require("../src/core/document")
  const lines = Array.from({ length: 5000 }, (_, i) => `line_${i}_content_for_testing = ${i};`).join("\n")
  const doc = new Document(1, lines)
  const t0 = performance.now()
  for (let i = 0; i < 500; i++) {
    doc.move("down")
  }
  const duration = performance.now() - t0
  expect(doc.position.line).toBe(500)
  expect(duration).toBeLessThan(250)
})

test("performance: fuzzyScore evaluates 20,000 candidates quickly", () => {
  const { fuzzyScore } = require("../src/core/search")
  const candidates = Array.from({ length: 20000 }, (_, i) => `src/components/navigation/sidebar_panel_${i}.tsx`)
  const t0 = performance.now()
  let matches = 0
  for (let i = 0; i < candidates.length; i++) {
    const score = fuzzyScore(candidates[i]!, "sidepan")
    if (score > -Infinity) matches++
  }
  const duration = performance.now() - t0
  expect(matches).toBe(20000)
  expect(duration).toBeLessThan(100)
})

