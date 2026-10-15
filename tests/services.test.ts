import { expect, test, afterEach } from "bun:test"
import { mkdtemp, mkdir, symlink, readFile, rm } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join, resolve } from "node:path"
import { atomicWrite, FileTree, readText, saveSession, loadSession } from "../src/services/files"
import { Git, execute, parsePorcelain, parseDiffHunks, parseHunks, hunkSummary } from "../src/services/git"
import { TerminalSession } from "../src/services/terminal"
import { DebugAdapter } from "../src/services/dap"
import { LspClient, lspBackoffDelay, lspRetryAllowed, incrementalEdit } from "../src/services/lsp"
import { highlight, isMarkdownLanguage, parseFenceBlocks, resolvePreviewImage, resolvePreviewImageAsync, clearPreviewImageCache, extractOpenableLink } from "../src/services/languages"
import { fileGlyph, folderGlyph, defaultFile } from "../src/services/icons"
const roots: string[] = []
async function fixture(): Promise<string> { const root = await mkdtemp(join(tmpdir(), "oride-solid-")); roots.push(root); return root }
afterEach(async () => { clearPreviewImageCache(); await Promise.all(roots.splice(0).map(root => rm(root, { recursive: true, force: true }))) })

test("atomic writes preserve symlink targets and refuse binary editing", async () => {
  const root = await fixture(), path = join(root, "a.txt"), link = join(root, "link.txt")
  await Bun.write(path, "before"); await symlink(path, link)
  await atomicWrite(link, "after")
  expect(await readText(path)).toBe("after")
  await Bun.write(join(root, "binary"), new Uint8Array([0, 1, 2]))
  expect(readText(join(root, "binary"))).rejects.toThrow("Binary")
})
test("tree creates exclusively under selection and rejects traversal/symlinks", async () => {
  const root = await fixture(), external = await fixture()
  await mkdir(join(root, "src"))
  const tree = new FileTree(root)
  await tree.refresh(); tree.selected = tree.rows.findIndex(row => row.name === "src")
  expect(await tree.create("main.ts", false)).toBe(join(root, "src", "main.ts"))
  expect(tree.create("../../escape", false)).rejects.toThrow()
  await symlink(external, join(root, "outside"))
  expect(tree.create("../outside/x.txt", false)).rejects.toThrow()
  expect(tree.create("main.ts", false)).rejects.toThrow()
})
test("tree refresh bounds depth and entries and reports truncation", async () => {
  const root = await fixture()
  let deep = root
  const chain: string[] = []
  for (let level = 0; level < 10; level++) { deep = join(deep, `d${level}`); await mkdir(deep); chain.push(deep) }
  for (let file = 0; file < 10; file++) await Bun.write(join(root, `f${file}.txt`), "x")
  const bounded = new FileTree(root, false, { maxDepth: 3, maxEntries: 5 })
  for (const directory of chain) bounded.expanded.add(directory)
  await bounded.refresh()
  expect(bounded.truncated).toBe(true)
  expect(bounded.rows.length).toBeLessThanOrEqual(5)
  expect(bounded.rows.every(row => row.depth <= 3)).toBe(true)
  const full = new FileTree(root)
  for (const directory of chain) full.expanded.add(directory)
  await full.refresh()
  expect(full.truncated).toBe(false)
  expect(full.rows.length).toBeGreaterThan(bounded.rows.length)
})
test("sessions restore active file and layout in compatible TOML", async () => {
  const root = await fixture(); await mkdir(join(root, ".oride"))
  await saveSession({
    workspace: root,
    files: [join(root, "a"), join(root, "b")],
    active_index: 1,
    scroll_y: 42,
    show_tree: false,
    show_scm: true,
    tree_width: 32,
    split: { orientation: "horizontal", secondary_file: join(root, "b"), ratio_percent: 60 }
  })
  const session = await loadSession(root)
  expect(session.files).toEqual([join(root, "a"), join(root, "b")])
  expect(session.show_tree).toBe(false)
  expect(session.scroll_y).toBe(42)
  expect(session.split?.orientation).toBe("horizontal")
  expect(session.split?.secondary_file).toBe(join(root, "b"))
  expect(session.split?.ratio_percent).toBe(60)
  const tomlContent = await readFile(join(root, ".oride", "session.toml"), "utf8")
  expect(tomlContent).toContain("active_index = 1")
  expect(tomlContent).toContain("scroll_y = 42")
})
test("Git stage, unstage, commit and diff operate on a real repository", async () => {
  const root = await fixture(); await execute("git", ["init", "-b", "main"], root)
  await execute("git", ["config", "user.email", "test@example.invalid"], root)
  await execute("git", ["config", "user.name", "Oride Test"], root)
  const git = new Git(root), filename = "a ; file.txt"
  await Bun.write(join(root, filename), "first\n")
  expect((await git.status()).entries[0]?.badge).toBe("?")
  await git.stage(filename); expect((await git.status()).entries[0]?.index).toBe("A")
  await git.commit("initial")
  await Bun.write(join(root, filename), "second\n")
  expect(await git.diff(filename)).toContain("+second")
  const gutter = await git.diffGutter(filename)
  expect(gutter.get(0)).toBe("modified")
  await git.stage(filename); await git.unstage(filename)
  expect((await git.status()).entries[0]?.worktree).toBe("M")
  expect((await git.status()).branch).toBe("main")
  await mkdir(join(root, "src")); await Bun.write(join(root, "src", "nested.ts"), "const x=1")
  const nested = new Git(join(root, "src"))
  expect((await nested.status()).entries[0]?.path).toBe("nested.ts")
  await nested.stage("nested.ts")
  expect((await nested.status()).entries[0]?.index).toBe("A")
})
test("Git porcelain preserves unusual names and rename destination", () => {
  expect(parsePorcelain("R  new name\0old name\0?? a\nb\0").map(entry => entry.path)).toEqual(["new name", "a\nb"])
})
test("Git diff hunks parse additions, modifications and deletions into line gutter markers", () => {
  const diff = `@@ -0,0 +1,3 @@
@@ -10,2 +12,1 @@
@@ -20,1 +25,0 @@`
  const markers = parseDiffHunks(diff)
  expect(markers.get(0)).toBe("added")
  expect(markers.get(1)).toBe("added")
  expect(markers.get(2)).toBe("added")
  expect(markers.get(11)).toBe("modified")
  expect(markers.get(24)).toBe("deleted")
})
test("PTY runs a real shell, renders VT screen and resizes", async () => {
  const root = await fixture()
  const terminal = new TerminalSession("/bin/sh", root, 60, 8, () => undefined)
  try {
    terminal.write("printf '\\033[31mORIDE_PTY_READY\\033[0m\\n'\r")
    for (let attempt = 0; attempt < 100 && !terminal.lines().some(line => line.includes("ORIDE_PTY_READY")); attempt++) await Bun.sleep(10)
    expect(terminal.lines().join("\n")).toContain("ORIDE_PTY_READY")
    terminal.resize(40, 6); expect(terminal.cells()).toHaveLength(6)
    expect(terminal.cells()[0]).toHaveLength(40)
  } finally { await terminal.close() }
})
test("LSP stdio initialization, completion, hover, definitions and formatting", async () => {
  const root = await fixture(), server = new LspClient([process.execPath, resolve("tests/fixtures/lsp-server.ts"), join(root, "did-change.json")], root, 1000, () => undefined)
  try {
    await server.initialize(); await server.sync(join(root, "test.ts"), "typescript", "const foo = 1", 0)
    const response = await server.completion(join(root, "test.ts"), { line: 0, character: 8 })
    expect(Array.isArray(response) ? response[0]?.label : response?.items[0]?.label).toBe("foobar")
    expect((await server.hover(join(root, "test.ts"), { line: 0, character: 8 }))?.contents).toEqual({ kind: "plaintext", value: "hover from fixture" })
    expect(await server.definition(join(root, "test.ts"), { line: 0, character: 8 })).not.toBeNull()
    expect((await server.formatting(join(root, "test.ts"), 4, true))?.[0]?.newText).toBe("formatted")
    expect((await server.references(join(root, "test.ts"), { line: 0, character: 7 }))?.length).toBe(1)
    expect((await server.signatureHelp(join(root, "test.ts"), { line: 0, character: 8 }))?.signatures[0]?.label).toBe("foobar(arg: string): void")
    expect((await server.codeAction(join(root, "test.ts"), { start: { line: 0, character: 0 }, end: { line: 0, character: 5 } }, []))?.[0]?.title).toBe("Fixture fix")
    expect(server.supports("hoverProvider")).toBe(true)
    expect(server.supports("renameProvider")).toBe(false)
    await server.sync(join(root, "test.ts"), "typescript", "const foo = 2", 1)
    let change: { textDocument: { version: number }; contentChanges: Array<{ range: unknown; text: string }> } | undefined
    for (let attempt = 0; attempt < 200 && !change; attempt++) {
      try { change = JSON.parse(await readFile(join(root, "did-change.json"), "utf8")) } catch { await Bun.sleep(10) }
    }
    expect(change?.textDocument.version).toBe(1)
    expect(change?.textDocument.version).toBe(1)
    expect(change?.contentChanges).toHaveLength(1)
    expect(change?.contentChanges[0]?.range).toEqual({ start: { line: 0, character: 12 }, end: { line: 0, character: 13 } })
    expect(change?.contentChanges[0]?.text).toBe("2")
  } finally { await server.close() }
})
test("Git hunk parser summarizes and stages hunks by index", async () => {
  const root = await fixture(); await execute("git", ["init", "-b", "main"], root)
  await execute("git", ["config", "user.email", "test@example.invalid"], root)
  await execute("git", ["config", "user.name", "Hunk Test"], root)
  const file = join(root, "b.txt")
  await Bun.write(file, Array.from({ length: 30 }, (_, i) => `line ${i + 1}`).join("\n") + "\n")
  const git = new Git(root)
  await git.stage("b.txt"); await git.commit("base")
  const modified = (await readFile(file, "utf8")).replace("line 2\n", "LINE 2\n").replace("line 28\n", "LINE 28\n")
  await Bun.write(file, modified)
  const hunks = await git.hunks("b.txt", false)
  expect(hunks.length).toBe(2)
  expect(hunkSummary(hunks[0]!)).toContain("(+1/-1)")
  expect(hunkSummary(hunks[1]!).startsWith("@@ -")).toBe(true)
  expect(hunkSummary(hunks[1]!)).toContain("(+1/-1)")
  expect(parseHunks("no diff here").hunks).toEqual([])
  expect(await git.stageHunkAt("b.txt", 0)).toContain("Staged hunk 1")
  expect((await git.hunks("b.txt", true)).length).toBe(1)
  expect((await git.hunks("b.txt", false)).length).toBe(1)
  expect(await git.unstageHunkAt("b.txt", 0)).toContain("Unstaged hunk 1")
  expect((await git.hunks("b.txt", true)).length).toBe(0)
  expect(git.stageHunkAt("b.txt", 9)).rejects.toThrow("out of range")
})
test("Git rebase onto branch works and validates input", async () => {
  const root = await fixture(); await execute("git", ["init", "-b", "main"], root)
  await execute("git", ["config", "user.email", "test@example.invalid"], root)
  await execute("git", ["config", "user.name", "Rebase Test"], root)
  const git = new Git(root)
  await Bun.write(join(root, "a.txt"), "a\n"); await git.stage("a.txt"); await git.commit("base")
  await execute("git", ["checkout", "-b", "side"], root)
  await Bun.write(join(root, "b.txt"), "b\n"); await git.stage("b.txt"); await git.commit("side work")
  await execute("git", ["checkout", "main"], root)
  await Bun.write(join(root, "c.txt"), "c\n"); await git.stage("c.txt"); await git.commit("main work")
  await execute("git", ["checkout", "side"], root)
  expect(await git.rebase("main")).toBe("")
  expect(await execute("git", ["log", "--oneline"], root)).toContain("main work")
  expect(git.rebase("")).rejects.toThrow("Invalid rebase target")
  expect(git.rebase("../evil")).rejects.toThrow("Invalid rebase target")
  expect(git.rebaseAbort()).rejects.toThrow()
})
test("DAP adapter runs a full session against the fixture", async () => {
  const root = await fixture(), capture = join(root, "dap-requests.json")
  const adapter = new DebugAdapter([process.execPath, resolve("tests/fixtures/dap-adapter.ts"), capture], root, () => undefined)
  try {
    await adapter.initialize()
    expect(await adapter.setBreakpoints("/test.ts", [{ line: 10, condition: "x > 1" }])).toEqual([{ verified: true, line: 10 }])
    expect(JSON.parse(await readFile(capture, "utf8"))).toEqual({ source: { path: "/test.ts", name: "test.ts" }, breakpoints: [{ line: 10, condition: "x > 1" }] })
    await adapter.launch({ program: "/test.ts" })
    expect(await adapter.threads()).toEqual([{ id: 1, name: "main" }])
    expect((await adapter.stackTrace(1))[0]?.name).toBe("main")
    const scopes = await adapter.request<{ scopes?: Array<{ name: string; variablesReference: number }> }>("scopes", { frameId: 11 })
    expect(scopes?.scopes?.[0]?.variablesReference).toBe(7)
    const variables = await adapter.request<{ variables?: Array<{ name: string; value: string }> }>("variables", { variablesReference: 7 })
    expect(variables?.variables?.[0]).toMatchObject({ name: "x", value: "42" })
  } finally { await adapter.disconnect() }
})
test("markdown fences parse and highlight inner languages", async () => {
  expect(parseFenceBlocks("# t\n```ts\nconst x = 1\n```\n")).toHaveLength(1)
  expect(parseFenceBlocks("# t\n```\nplain\n```\n")).toEqual([])
  expect(parseFenceBlocks("# t\n```ts\nunclosed\n")).toEqual([])
  expect(parseFenceBlocks("~~~\nplain\n~~~\n\n```py\nx = 1\n```\n").map(block => block.language)).toEqual(["py"])
  const roles = new Set((await highlight("# Title\n\n```ts\nconst x: number = 1 // hi\n```\n", "markdown", "default")).map(token => token.role))
  for (const role of ["heading", "keyword", "number", "comment"]) expect(roles.has(role)).toBe(true)
})
test("incremental edits cover prefix, suffix and multi-line spans", () => {
  expect(incrementalEdit("abc", "abc")).toBeNull()
  expect(incrementalEdit("abc", "aXc")).toEqual({ start: 1, end: 2, text: "X" })
  expect(incrementalEdit("ab", "abcd")).toEqual({ start: 2, end: 2, text: "cd" })
  expect(incrementalEdit("abcd", "ab")).toEqual({ start: 2, end: 4, text: "" })
  expect(incrementalEdit("a\nb\nc", "a\nB\nc")).toEqual({ start: 2, end: 3, text: "B" })
  expect(incrementalEdit("", "hi")).toEqual({ start: 0, end: 0, text: "hi" })
})
test("LSP retry backoff grows exponentially and caps at 30s", () => {
  expect(lspBackoffDelay(0)).toBe(1000)
  expect(lspBackoffDelay(1)).toBe(2000)
  expect(lspBackoffDelay(3)).toBe(8000)
  expect(lspBackoffDelay(100)).toBe(30000)
  expect(lspBackoffDelay(-5)).toBe(1000)
  expect(lspRetryAllowed(0, 0, 999)).toBe(false)
  expect(lspRetryAllowed(0, 0, 1000)).toBe(true)
  expect(lspRetryAllowed(0, 2, 3999)).toBe(false)
  expect(lspRetryAllowed(0, 2, 4000)).toBe(true)
})
test("syntax engines cover reference languages, including Ori fallback", async () => {
  const typed = await highlight('const x = "hello"; // comment', "typescript", "default")
  expect(typed.length).toBeGreaterThan(3)
  expect(new Set(typed.map(token => token.color)).size).toBeGreaterThan(2)
  expect(typed.find(token => token.start === 0)?.end).toBe(5)
  expect(typed.some(token => token.role === "string")).toBe(true)
  expect(typed.some(token => token.role === "comment")).toBe(true)
  for (const language of ["rust", "c", "bash", "markdown", "html", "css", "typescript", "javascript", "python", "d", "lua", "nim", "ruby", "ori"]) {
    const tokens = await highlight(language === "markdown" ? "# Heading" : 'let x = "hello" // comment', language, "default")
    expect(tokens.length).toBeGreaterThan(0)
  }
  expect(await highlight("uncolored", "text", "default")).toEqual([])
})
test("preview images never download remotes, leave the workspace, or require opt-in", async () => {
  const root = await fixture(), external = await fixture()
  await Bun.write(join(root, "doc.md"), "# doc")
  await Bun.write(join(root, "fig.png"), "fake")
  await Bun.write(join(external, "secret.png"), "fake")
  const escape = `../${external.split("/").at(-1)}/secret.png`
  expect(isMarkdownLanguage("markdown")).toBe(true)
  expect(isMarkdownLanguage("mdx")).toBe(true)
  expect(isMarkdownLanguage("typescript")).toBe(false)
  expect(resolvePreviewImage("https://example.invalid/fig.png", join(root, "doc.md"), root, true)).toBeUndefined()
  expect(resolvePreviewImage("fig.png", join(root, "doc.md"), root, false)).toBeUndefined()
  expect(resolvePreviewImage(escape, join(root, "doc.md"), root, true)).toBeUndefined()
  expect(resolvePreviewImage("missing.png", join(root, "doc.md"), root, true)).toBeUndefined()
  expect(resolvePreviewImage("fig.png", join(root, "doc.md"), root, true)).toBe(join(root, "fig.png"))
  // Cache hit should return identical result without throwing
  expect(resolvePreviewImage("fig.png", join(root, "doc.md"), root, true)).toBe(join(root, "fig.png"))
  expect(await resolvePreviewImageAsync("fig.png", join(root, "doc.md"), root, true)).toBe(join(root, "fig.png"))
  expect(await resolvePreviewImageAsync("missing.png", join(root, "doc.md"), root, true)).toBeUndefined()
  expect(await resolvePreviewImageAsync(escape, join(root, "doc.md"), root, true)).toBeUndefined()
})
test("only http(s) selections open in a browser", () => {
  expect(extractOpenableLink("https://example.com/x")).toBe("https://example.com/x")
  expect(extractOpenableLink("  http://a.b  ")).toBe("http://a.b")
  expect(extractOpenableLink("file:///etc/passwd")).toBeUndefined()
  expect(extractOpenableLink("not a url")).toBeUndefined()
  expect(extractOpenableLink("")).toBeUndefined()
})
test("nerd file icons map names to single-cell glyphs with honest fallback", () => {
  // Codepoints fixados da curadoria nvim-web-devicons; quebrar aqui é drift da tabela.
  expect(fileGlyph("main.rs")).toBe("\u{e68b}")
  expect(fileGlyph("app.ts")).toBe("\u{e628}")
  expect(fileGlyph("NOTES.md")).toBe("\u{f48a}")
  expect(fileGlyph("Dockerfile")).toBe("\u{f0868}")
  expect(fileGlyph(".gitignore")).toBe("\u{e702}")
  expect(fileGlyph("Makefile")).toBe("\u{e779}")
  expect(fileGlyph("archive.tar")).toBe("\u{f410}")
  expect(fileGlyph("fix.orig")).toBe("\u{e728}")
  expect(fileGlyph("noext")).toBe("\u{f15b}")
  expect(fileGlyph("LICENSE")).toBe("\u{f15b}")
  expect(fileGlyph("types.d.ts")).toBe("\u{e628}")
  expect(defaultFile).toBe("\u{f15b}")
  expect(folderGlyph(false)).toBe("\u{f07b}")
  expect(folderGlyph(true)).toBe("\u{f07c}")
  for (const name of ["a.rs", "b.tsx", "c.py", "d.go", "e.lua", "f.md", "Dockerfile", "noext", "x.tar.gz"]) {
    const glyph = fileGlyph(name)
    expect(Bun.stringWidth(glyph)).toBe(1)
    expect([...new Intl.Segmenter(undefined, { granularity: "grapheme" }).segment(glyph)]).toHaveLength(1)
  }
})
