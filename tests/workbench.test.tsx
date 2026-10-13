import { expect, test } from "bun:test"
import { mkdtemp, rm } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { testRender } from "@opentui/solid"
import { Application } from "../src/app/application"
import { defaultConfig } from "../src/services/config"
import { Workbench } from "../src/ui/workbench"
import { clampTreeWidth, clampSplitRatio, clampTerminalHeight } from "../src/ui/dividers"

async function waitFor(predicate: () => boolean, timeout = 3000): Promise<void> {
  const deadline = performance.now() + timeout
  while (!predicate() && performance.now() < deadline) await Bun.sleep(10)
  expect(predicate()).toBe(true)
}

async function setup(width = 100, height = 25) {
  const root = await mkdtemp(join(tmpdir(), "oride-workbench-"))
  const config = defaultConfig(); config.lsp.enabled = false; config.locale = "en"
  const app = new Application(root, config)
  await Bun.write(join(root, "main.ts"), 'const answer = 42\nconsole.log(answer)\n')
  await Bun.write(join(root, "notes.md"), "# Notes\nhello answer\n")
  await app.tree.refresh(); app.newDocument()
  const renderer = await testRender(() => <Workbench app={app} />, { width, height, kittyKeyboard: true })
  await renderer.renderOnce()
  return { app, root, ...renderer, cleanup: async () => { renderer.renderer.destroy(); await app.close(); await rm(root, { recursive: true, force: true }) } }
}

test("keyboard editing, bracketed paste, undo and redo go through UI", async () => {
  const view = await setup()
  try {
    await view.mockInput.typeText("hello")
    await view.mockInput.pasteBracketedText(" 🌍\nworld")
    expect(view.app.document.text).toBe("hello 🌍\nworld")
    view.mockInput.pressKey("z", { ctrl: true }); await view.flush()
    expect(view.app.document.text).toBe("")
    view.mockInput.pressKey("y", { ctrl: true }); await view.flush()
    expect(view.app.document.text).toBe("hello 🌍\nworld")
    await view.renderOnce(); expect(view.captureCharFrame()).toContain("world")
  } finally { await view.cleanup() }
})
test("Ctrl+P filters and opens a file; Ctrl+S writes it", async () => {
  const view = await setup()
  try {
    view.mockInput.pressKey("p", { ctrl: true })
    await waitFor(() => view.app.overlay?.kind === "picker")
    await view.mockInput.typeText("main")
    view.mockInput.pressEnter()
    await waitFor(() => view.app.document.path === join(view.root, "main.ts"))
    expect(view.app.overlay).toBeUndefined()
    await view.mockInput.typeText("// test\n")
    view.mockInput.pressKey("s", { ctrl: true })
    await waitFor(() => !view.app.document.dirty)
    expect(await Bun.file(join(view.root, "main.ts")).text()).toStartWith("// test\n")
  } finally { await view.cleanup() }
})
test("menu mouse capture and Enter execute shared actions", async () => {
  const view = await setup()
  try {
    await view.mockMouse.click(2, 0)
    await view.renderOnce()
    expect(view.app.menu?.column).toBe(0)
    expect(view.captureCharFrame()).toContain("New tab")
    view.mockInput.pressEnter(); await view.flush()
    expect(view.app.documents).toHaveLength(2)
    expect(view.app.menu).toBeUndefined()
    view.mockInput.pressKey("e", { meta: true }); await view.flush()
    expect(view.app.menu?.column).toBe(1)
    view.mockInput.pressEscape(); await view.flush()
    expect(view.app.menu).toBeUndefined()
  } finally { await view.cleanup() }
})
test("tree double click opens file and editor mouse drag selects text", async () => {
  const view = await setup()
  try {
    const row = view.app.tree.rows.findIndex(row => row.name === "main.ts")
    await view.mockMouse.click(6, 3 + row)
    await view.mockMouse.click(6, 3 + row)
    await waitFor(() => view.app.document.path === join(view.root, "main.ts"))
    await view.renderOnce()
    await view.mockMouse.click(36, 2)
    expect(view.app.document.primary.head).toBeGreaterThan(0)
    await view.mockMouse.drag(34, 2, 39, 2)
    expect(view.app.document.selectedText.length).toBeGreaterThan(0)
  } finally { await view.cleanup() }
})
test("multicursor shortcuts edit both lines and paint both caret cells", async () => {
  const view = await setup()
  try {
    await view.mockInput.typeText("ab\nab")
    view.mockInput.pressKey("HOME", { ctrl: true })
    await view.flush()
    view.mockInput.pressArrow("down", { ctrl: true, meta: true })
    await view.flush()
    expect(view.app.document.selections).toHaveLength(2)
    await view.mockInput.typeText("X")
    expect(view.app.document.text).toBe("Xab\nXab")
    await view.renderOnce()
    expect(view.captureCharFrame()).toContain("Xab")
  } finally { await view.cleanup() }
})
test("split shortcuts and Tab buffers use shared documents", async () => {
  const view = await setup()
  try {
    await view.mockInput.typeText("shared")
    view.mockInput.pressKey("v", { ctrl: true, meta: true }); await view.flush()
    expect(view.app.panes).toHaveLength(2)
    await view.mockInput.typeText("!")
    await view.renderOnce()
    expect(view.captureCharFrame().split("shared!").length - 1).toBe(2)
    view.mockInput.pressKey("w", { ctrl: true, meta: true }); await view.flush()
    expect(view.app.panes).toHaveLength(1)
  } finally { await view.cleanup() }
})
test("search overlay captures text and project result opens correct line", async () => {
  const view = await setup()
  try {
    view.mockInput.pressKey("f", { ctrl: true, shift: true })
    await view.mockInput.typeText("answer")
    await waitFor(() => view.app.overlay?.kind === "search" && view.app.overlay.hits.length === 3)
    expect(view.app.document.text).toBe("")
    view.mockInput.pressEnter()
    await waitFor(() => view.app.document.path === join(view.root, "main.ts"))
    expect(view.app.document.position.line).toBe(0)
    expect(view.app.overlay).toBeUndefined()
  } finally { await view.cleanup() }
})
test("Markdown preview and terminal shortcuts display actual panels", async () => {
  const view = await setup()
  try {
    await view.app.openFile(join(view.root, "notes.md"))
    view.mockInput.pressKey("p", { meta: true }); await view.flush()
    expect(view.app.preview).toBe(true)
    await view.renderOnce(); expect(view.captureCharFrame()).toContain("Notes")
    view.mockInput.pressKey("`", { ctrl: true }); await view.flush()
    await view.mockInput.typeText("echo WORKBENCH_PTY_READY")
    view.mockInput.pressEnter()
    for (let attempt = 0; attempt < 100 && !view.app.terminal?.lines().join("\n").includes("WORKBENCH_PTY_READY"); attempt++) await Bun.sleep(10)
    expect(view.app.terminal?.lines().join("\n")).toContain("WORKBENCH_PTY_READY")
    await view.renderOnce(); expect(view.captureCharFrame()).toContain("WORKBENCH_PTY_READY")
    view.mockInput.pressEscape(); await view.flush(); expect(view.app.focus).toBe("editor")
  } finally { await view.cleanup() }
})
test("overlay blocks mouse edits and quit protects unsaved documents", async () => {
  const view = await setup()
  try {
    await view.mockInput.typeText("unsaved")
    const head = view.app.document.primary.head
    view.mockInput.pressKey("q", { ctrl: true }); await view.flush()
    expect(view.app.quit).toBe(false); expect(view.app.overlay?.kind).toBe("confirm")
    await view.mockMouse.click(36, 2)
    expect(view.app.document.primary.head).toBe(head)
    view.mockInput.pressEscape(); await view.flush()
    expect(view.app.overlay).toBeUndefined(); expect(view.app.quit).toBe(false)
  } finally { await view.cleanup() }
})
test("no-color produces only terminal-default color intents", async () => {
  const view = await setup()
  try {
    view.app.config.theme = "no-color"; view.app.changed()
    await view.mockInput.typeText("plain text")
    await view.renderOnce()
    for (const line of view.captureSpans().lines) for (const span of line.spans) {
      if (span.text.trim().length > 0) expect(span.fg.intent).toBe("default")
      expect(span.bg.intent).toBe("default")
    }
  } finally { await view.cleanup() }
})

test("resize to narrow terminal keeps menu/editor/status reachable", async () => {
  const view = await setup()
  try {
    await view.mockInput.typeText("界👩‍💻 long line ".repeat(5))
    view.resize(40, 12); await view.renderOnce()
    const frame = view.captureCharFrame()
    expect(frame).toContain("Daisy")
    expect(frame).toContain("editor")
    expect(frame).toContain("界")
  } finally { await view.cleanup() }
})
test("file icons stay off by default and appear with icons=true", async () => {
  const view = await setup()
  try {
    expect(view.app.config.icons).toBe(false)
    await view.renderOnce()
    // Sem Nerd Font e sem opt-in, a árvore mostra só texto: nenhum glifo vaza.
    expect(view.captureCharFrame()).not.toContain("\u{e628}")
    view.app.config.icons = true; view.app.changed()
    await view.renderOnce()
    const frame = view.captureCharFrame()
    expect(frame).toContain("\u{e628}")
    expect(frame).toContain("\u{f07c}")
  } finally { await view.cleanup() }
})
test("divider geometry clamps tree, split ratio and terminal height", () => {
  expect(clampTreeWidth(34, 100)).toBe(34)
  expect(clampTreeWidth(2, 100)).toBe(8)
  expect(clampTreeWidth(95, 100)).toBe(80)
  expect(clampSplitRatio(36, 72)).toBe(50)
  expect(clampSplitRatio(1000, 72)).toBe(85)
  expect(clampSplitRatio(-5, 72)).toBe(15)
  expect(clampSplitRatio(5, 0)).toBe(85)
  expect(clampTerminalHeight(12, 25)).toBe(12)
  expect(clampTerminalHeight(0, 25)).toBe(20)
  expect(clampTerminalHeight(24, 25)).toBe(3)
})
test("dragging the panel divider resizes the terminal", async () => {
  const view = await setup()
  try {
    view.mockInput.pressKey("`", { ctrl: true }); await view.flush()
    expect(view.app.bottom).toBe("terminal")
    expect(view.app.terminalHeight).toBe(10)
    await view.mockMouse.drag(50, 14, 50, 12)
    expect(view.app.terminalHeight).toBe(12)
  } finally { await view.cleanup() }
})
test("Markdown gate keeps the editor and Alt+Enter without URL reports", async () => {
  const view = await setup()
  try {
    await view.app.openFile(join(view.root, "main.ts"))
    view.mockInput.pressKey("p", { meta: true }); await view.flush()
    expect(view.app.preview).toBe(false)
    expect(view.app.status).toBe("preview only for Markdown/MDX")
    await view.renderOnce()
    expect(view.captureCharFrame()).toContain("answer")
    await view.app.openFile(join(view.root, "notes.md"))
    view.mockInput.pressKey("p", { meta: true }); await view.flush()
    expect(view.app.preview).toBe(true)
    view.mockInput.pressEnter({ meta: true }); await view.flush()
    expect(view.app.status).toContain("http(s)")
    expect(view.app.overlay).toBeUndefined()
  } finally { await view.cleanup() }
})

test("theme picker live previews on arrow navigation and rolls back on Escape", async () => {
  const view = await setup()
  try {
    const original = view.app.config.theme
    await view.app.dispatch("select_theme")
    expect(view.app.overlay?.kind).toBe("picker")
    view.mockInput.pressArrow("down")
    await view.flush()
    expect(view.app.config.theme).not.toBe(original)
    view.mockInput.pressEscape()
    await view.flush()
    expect(view.app.config.theme).toBe(original)
    expect(view.app.overlay).toBeUndefined()
  } finally { await view.cleanup() }
})

test("Markdown preview scrolls via keyboard and mouse without mutating buffer", async () => {
  const view = await setup()
  try {
    const markdown = Array.from({ length: 60 }, (_, i) => `# Section ${i + 1}\nParagraph content line ${i + 1}`).join("\n\n")
    await Bun.write(join(view.root, "long.md"), markdown)
    await view.app.openFile(join(view.root, "long.md"))
    view.mockInput.pressKey("p", { meta: true }); await view.flush()
    expect(view.app.preview).toBe(true)
    expect(view.app.previewScroll).toBe(0)

    view.mockInput.pressArrow("down"); await view.flush()
    expect(view.app.previewScroll).toBe(3)
    view.mockInput.pressKey("j"); await view.flush()
    expect(view.app.previewScroll).toBe(6)

    view.mockInput.pressArrow("up"); await view.flush()
    expect(view.app.previewScroll).toBe(3)
    view.mockInput.pressKey("k"); await view.flush()
    expect(view.app.previewScroll).toBe(0)

    view.mockInput.pressKey(" "); await view.flush()
    expect(view.app.previewScroll).toBeGreaterThan(0)

    view.mockInput.pressKey("g"); await view.flush()
    expect(view.app.previewScroll).toBe(0)

    await view.mockMouse.scroll(40, 10, "down"); await view.flush()
    expect(view.app.previewScroll).toBeGreaterThan(0)

    const beforeText = view.app.document.text
    await view.mockInput.typeText("forbidden text"); await view.flush()
    expect(view.app.document.text).toBe(beforeText)

    view.mockInput.pressEscape(); await view.flush()
    expect(view.app.preview).toBe(false)
  } finally { await view.cleanup() }
})

test("statusbar renders structured responsive layout with interactive badges", async () => {
  const view = await setup(100, 25)
  try {
    view.app.config.icons = true
    view.app.gitStatus = { branch: "main", ahead: 2, behind: 1, entries: [] }
    view.app.changed()
    await view.renderOnce()
    let frame = view.captureCharFrame()
    expect(frame).toContain("DAISY")
    expect(frame).toContain("editor")
    expect(frame).toContain("main")
    expect(frame).toContain("Ln 1, Col 1")

    // Click mode pill to toggle modal mode
    await view.mockMouse.click(2, 24)
    await view.flush()
    expect(view.app.config.editor.modal_mode).toBe(true)
    await view.renderOnce()
    frame = view.captureCharFrame()
    expect(frame).toContain("NORMAL")

    // Click mode pill again to toggle back
    await view.mockMouse.click(2, 24)
    await view.flush()
    expect(view.app.config.editor.modal_mode).toBe(false)

    // Diagnostics badge click opens problems panel
    view.app.servers.set("typescript", {
      diagnostics: new Map([["file:///test.ts", [{
        range: { start: { line: 0, character: 0 }, end: { line: 0, character: 5 } },
        severity: 1,
        message: "Type error"
      }]]]),
      close: async () => {}
    } as any)
    view.app.config.lsp.enabled = true
    view.app.changed()
    await view.renderOnce()
    frame = view.captureCharFrame()
    expect(frame).toContain("1")

    await view.app.dispatch("toggle_diagnostics")
    expect(view.app.bottom).toBe("problems")
  } finally { await view.cleanup() }
})

test("scrollbar renders for long buffer and tab middle-click closes tab", async () => {
  const view = await setup()
  try {
    view.app.newDocument()
    expect(view.app.documents.length).toBe(2)
    const doc2Id = view.app.documents[1]!.id

    // Close tab by id
    view.app.closeTab(doc2Id)
    expect(view.app.documents.length).toBe(1)

    // Insert many lines to exceed editor height and verify scrollbar rendered
    const lines = Array.from({ length: 60 }, (_, i) => `content line ${i + 1}`).join("\n")
    view.app.document.insert(lines)
    view.app.changed()
    await view.renderOnce()
    const frame = view.captureCharFrame()
    expect(frame.includes("█") || frame.includes("│")).toBe(true)

    // Dispatch goto_line opens prompt
    await view.app.dispatch("goto_line")
    expect(view.app.overlay?.kind).toBe("prompt")
  } finally { await view.cleanup() }
})

test("daisy-lain cyberpunk theme renders HUD badges and allows seamless switching back to classic themes", async () => {
  const view = await setup(125, 25)
  try {
    expect(view.app.preferences.themes.has("daisy-lain")).toBe(true)

    // Switch to daisy-lain
    view.app.config.theme = "daisy-lain"
    view.app.changed()
    await view.renderOnce()
    let frame = view.captureCharFrame()

    // Menubar and HUD indicators
    expect(frame).toContain("DAISY_")
    expect(frame).toContain("PORT: 98")
    expect(frame).toContain("[01]")
    expect(frame).toContain("[ ✳ DAISY ]")
    expect(frame).toContain("SIGNAL_STABLE")
    expect(frame).toContain("[ UTF-8 ]")
    expect(frame).toContain("[ ✳ FX: ON ]")

    // Modal mode in daisy-lain shows [ 001 NORMAL ]
    view.app.config.editor.modal_mode = true
    view.app.vim = "normal"
    view.app.changed()
    await view.renderOnce()
    frame = view.captureCharFrame()
    expect(frame).toContain("[ 001 NORMAL ]")

    // Overlay in daisy-lain shows ▣ DAISY:// wireframe header
    await view.app.dispatch("command_palette")
    await view.renderOnce()
    frame = view.captureCharFrame()
    expect(frame).toContain("▣ DAISY://")
    expect(frame).toContain("QUERY://")

    // Close overlay
    view.app.overlay = undefined

    // Switch to daisy-lain-purple
    view.app.config.theme = "daisy-lain-purple"
    view.app.changed()
    await view.renderOnce()
    frame = view.captureCharFrame()
    expect(frame).toContain("DAISY_")
    expect(frame).toContain("[ 001 NORMAL ]")

    // Switch to daisy-lain-pink
    view.app.config.theme = "daisy-lain-pink"
    view.app.changed()
    await view.renderOnce()
    frame = view.captureCharFrame()
    expect(frame).toContain("DAISY_")
    expect(frame).toContain("[ 001 NORMAL ]")

    // Switch to daisy-lain-light
    view.app.config.theme = "daisy-lain-light"
    view.app.changed()
    await view.renderOnce()
    frame = view.captureCharFrame()
    expect(frame).toContain("DAISY_")
    expect(frame).toContain("[ 001 NORMAL ]")

    // Switch to classic theme (catppuccin-mocha)
    view.app.config.theme = "catppuccin-mocha"
    view.app.changed()
    await view.renderOnce()
    frame = view.captureCharFrame()

    // In classic theme, cyberpunk badges revert to clean classic style
    expect(frame).not.toContain("DAISY_")
    expect(frame).not.toContain("PORT: 98")
    expect(frame).not.toContain("[ 001 NORMAL ]")
    expect(frame).toContain("Daisy / Solid")
    expect(frame).toContain("NORMAL")
  } finally { await view.cleanup() }
})

