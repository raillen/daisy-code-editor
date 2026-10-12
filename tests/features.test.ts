import { expect, test, afterEach } from "bun:test"
import { mkdtemp, rm, writeFile } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { Application } from "../src/app/application"
import { defaultConfig } from "../src/services/config"
import { execute } from "../src/services/git"

const roots: string[] = []
async function fixture(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), "daisy-features-"))
  roots.push(root)
  return root
}

afterEach(async () => {
  await Promise.all(roots.splice(0).map(root => rm(root, { recursive: true, force: true })))
})

test("goto_line jumps to line and column, records jump, and centers cursor", async () => {
  const root = await fixture()
  const filePath = join(root, "sample.txt")
  const lines = Array.from({ length: 50 }, (_, i) => `line ${i + 1}`).join("\n")
  await writeFile(filePath, lines)
  const app = new Application(root, defaultConfig())
  app.editorHeight = 10
  await app.openFile(filePath)
  app.document.select(app.document.text.length)
  expect(app.document.position.line).toBe(49)

  // 1. Dispatch goto_line opens prompt
  await app.dispatch("goto_line")
  expect(app.overlay?.kind).toBe("prompt")

  // 2. Submit line 20:5
  if (app.overlay?.kind === "prompt") {
    await app.overlay.submit("20:5")
  }
  expect(app.document.position.line).toBe(19) // 0-indexed line 20 is index 19
  expect(app.document.position.character).toBe(4) // 0-indexed column 5 is index 4
  expect(app.panes[0]?.scroll).toBe(14) // 19 - Math.floor(10 / 2) = 14

  // 3. Jump back restores previous position
  await app.dispatch("jump_back")
  expect(app.document.position.line).toBe(49)

  await app.close()
})

test("closeTab closes specific document by id on middle-click or programmatic call", async () => {
  const root = await fixture()
  const app = new Application(root, defaultConfig())
  app.newDocument()
  const doc1Id = app.document.id

  app.newDocument()
  const doc2Id = app.document.id

  app.newDocument()
  const doc3Id = app.document.id

  expect(app.documents.length).toBe(3)
  expect(app.activeId).toBe(doc3Id)

  // Close middle tab (doc2) without changing active tab away from doc3
  app.closeTab(doc2Id)
  expect(app.documents.map(d => d.id)).toEqual([doc1Id, doc3Id])
  expect(app.activeId).toBe(doc3Id)

  // Close active tab (doc3)
  app.closeTab(doc3Id)
  expect(app.documents.map(d => d.id)).toEqual([doc1Id])
  expect(app.activeId).toBe(doc1Id)

  await app.close()
})

test("snippets expand $0 placeholder and position cursor inside template", async () => {
  const root = await fixture()
  const tsFile = join(root, "sample.ts")
  await writeFile(tsFile, "")
  const app = new Application(root, defaultConfig())
  await app.openFile(tsFile)

  // 1. Type "clg" snippet trigger
  app.insert("clg")
  await app.complete(true)
  expect(app.completion).toBeDefined()
  const clgItem = app.completion?.items.find(item => item.label === "clg")
  expect(clgItem).toBeDefined()
  expect(clgItem?.kind).toBe("snippet")

  // Accept completion
  const clgIndex = app.completion!.items.indexOf(clgItem!)
  app.acceptCompletion(clgIndex)
  expect(app.document.text).toBe("console.log()")
  // Cursor should be at offset 12: inside console.log()
  expect(app.document.primary.head).toBe(12)

  // 2. Test Rust snippet with $0
  const rsFile = join(root, "sample.rs")
  await writeFile(rsFile, "")
  await app.openFile(rsFile)
  app.insert("pfn")
  await app.complete(true)
  const pfnItem = app.completion?.items.find(item => item.label === "pfn")
  expect(pfnItem).toBeDefined()
  const pfnIndex = app.completion!.items.indexOf(pfnItem!)
  app.acceptCompletion(pfnIndex)
  expect(app.document.text).toBe("pub fn () {\n    \n}")
  expect(app.document.primary.head).toBe(7) // offset after "pub fn "

  // 3. Test Go snippet with $0
  const goFile = join(root, "sample.go")
  await writeFile(goFile, "")
  await app.openFile(goFile)
  app.insert("ife")
  await app.complete(true)
  const ifeItem = app.completion?.items.find(item => item.label === "ife")
  expect(ifeItem).toBeDefined()
  const ifeIndex = app.completion!.items.indexOf(ifeItem!)
  app.acceptCompletion(ifeIndex)
  expect(app.document.text).toBe("if err != nil {\n\treturn \n}")
  expect(app.document.primary.head).toBe(24) // offset after "return "

  await app.close()
})

test("local buffer rename replaces whole-word occurrences and ignores substrings", async () => {
  const root = await fixture()
  const app = new Application(root, defaultConfig())
  app.newDocument()
  app.insert("const count = 10;\nconst totalCount = count * 2;\nfunction getCount() { return count; }")

  // Position cursor on "count" (offset 6)
  app.document.select(6)
  await app.rename("itemCount")

  // "count" should be replaced by "itemCount", but "totalCount" and "getCount" must NOT be touched
  expect(app.document.text).toBe("const itemCount = 10;\nconst totalCount = itemCount * 2;\nfunction getCount() { return itemCount; }")
  expect(app.status).toContain("3 ocorrência(s)")

  await app.close()
})

test("git_blame displays commit info in read overlay for tracked line", async () => {
  const root = await fixture()
  await execute("git", ["init"], root)
  await execute("git", ["config", "user.name", "Tester"], root)
  await execute("git", ["config", "user.email", "tester@example.com"], root)

  const testFile = join(root, "blame_test.txt")
  await writeFile(testFile, "hello world\nsecond line\n")
  await execute("git", ["add", "blame_test.txt"], root)
  await execute("git", ["commit", "-m", "Initial commit for blame"], root)

  const app = new Application(root, defaultConfig())
  await app.openFile(testFile)
  app.document.select(0)

  // Dispatch git_blame
  await app.dispatch("git_blame")

  expect(app.overlay).toBeDefined()
  expect(app.overlay?.kind).toBe("read")
  if (app.overlay?.kind === "read") {
    expect(app.overlay.title).toContain("Git Blame: blame_test.txt:1")
    expect(app.overlay.text).toContain("Tester")
    expect(app.overlay.text).toContain("hello world")
  }

  await app.close()
})
