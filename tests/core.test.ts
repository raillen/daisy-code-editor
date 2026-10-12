import { expect, test } from "bun:test"
import { Document } from "../src/core/document"
import { findMatches, replacementEdits, searchDefaults, fuzzyScore } from "../src/core/search"
import { actionIds, defaultBindings } from "../src/core/keymap"
import { commands } from "../src/app/commands"
import { sessionName } from "../src/services/files"

test("multicursor insert, backspace, undo and redo are atomic", () => {
  const document = new Document(1, "ab\nab\nab")
  document.select(1); document.addCursor(1); document.addCursor(1)
  document.insert("X")
  expect(document.text).toBe("aXb\naXb\naXb")
  expect(document.selections.map(selection => selection.head)).toEqual([2, 6, 10])
  document.undo(); expect(document.text).toBe("ab\nab\nab")
  expect(document.selections).toHaveLength(3)
  document.redo(); expect(document.text).toBe("aXb\naXb\naXb")
  document.remove(true); expect(document.text).toBe("ab\nab\nab")
})
test("Unicode movement and deletion never split graphemes", () => {
  const document = new Document(1, "a👩‍💻é界")
  document.move("doc_end"); document.remove(true)
  expect(document.text).toBe("a👩‍💻é")
  document.remove(true); expect(document.text).toBe("a👩‍💻")
  document.move("left"); expect(document.primary.head).toBe(1)
})
test("overlapping ranges merge before editing", () => {
  const document = new Document(1, "abcdef")
  document.select(1, 4); document.select(3, 5, true); document.insert("X")
  expect(document.text).toBe("aXf")
})
test("literal and regex replacement have different dollar semantics", () => {
  const options = searchDefaults()
  const matches = findMatches("foo FOO", "foo", options)
  expect(matches).toHaveLength(2)
  expect(replacementEdits(matches, "$100", false)[0]?.text).toBe("$100")
  options.regex = true
  expect(replacementEdits(findMatches("foo", "(foo)", options), "$1!", true)[0]?.text).toBe("foo!")
})
test("accent search retains original text offsets", () => {
  const options = { ...searchDefaults(), ignoreAccents: true }
  expect(findMatches("🙂 café café", "cafe", options).map(match => [match.start, match.end])).toEqual([[3, 7], [8, 13]])
  expect(() => findMatches("x", "[", { ...options, regex: true })).toThrow()
})
test("all reference actions and bindings have handlers", () => {
  expect(Object.keys(commands).sort()).toEqual([...actionIds].sort())
  expect(Object.keys(defaultBindings)).toHaveLength(99)
  for (const action of Object.values(defaultBindings)) expect(commands[action].run).toBeFunction()
})
test("fuzzy finder scores subsequences and rejects nonmatches", () => {
  expect(fuzzyScore("src/main.ts", "smt")).toBeGreaterThan(-Infinity)
  expect(fuzzyScore("main.ts", "xyz")).toBe(-Infinity)
})
test("session digest stays deterministic", () => {
  expect(sessionName("/workspace")).toMatch(/^[a-f0-9]{16}\.toml$/)
  expect(sessionName("/workspace")).toBe(sessionName("/workspace"))
})
test("toggle comment toggles line comments and block comments for web formats", () => {
  const tsDoc = new Document(1, "  const answer = 42\n")
  tsDoc.toggleComment("//")
  expect(tsDoc.text).toBe("  // const answer = 42\n")
  tsDoc.toggleComment("//")
  expect(tsDoc.text).toBe("  const answer = 42\n")

  const htmlDoc = new Document(2, "  <div>hello</div>\n")
  htmlDoc.toggleComment({ open: "<!--", close: "-->" })
  expect(htmlDoc.text).toBe("  <!-- <div>hello</div> -->\n")
  htmlDoc.toggleComment({ open: "<!--", close: "-->" })
  expect(htmlDoc.text).toBe("  <div>hello</div>\n")

  const cssDoc = new Document(3, "  body { margin: 0; }\n")
  cssDoc.toggleComment({ open: "/*", close: "*/" })
  expect(cssDoc.text).toBe("  /* body { margin: 0; } */\n")
  cssDoc.toggleComment({ open: "/*", close: "*/" })
  expect(cssDoc.text).toBe("  body { margin: 0; }\n")
})
