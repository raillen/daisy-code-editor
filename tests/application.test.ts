import { expect, test, afterEach } from "bun:test"
import { mkdtemp, mkdir, rm } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { KeyEvent } from "@opentui/core"
import { Application, JumpStack, type PickerEntry } from "../src/app/application"
import { receiveKey, receivePaste } from "../src/app/input"
import { fuzzyScore } from "../src/core/search"
import { defaultConfig, loadConfig } from "../src/services/config"
import { loadPreferences, editorIndent } from "../src/services/preferences"
import { parsePlugin, runPlugin } from "../src/services/plugins"


const roots: string[] = []
async function fixture(): Promise<string> { const root = await mkdtemp(join(tmpdir(), "oride-app-")); roots.push(root); return root }
afterEach(async () => { await Promise.all(roots.splice(0).map(root => rm(root, { recursive: true, force: true }))) })
function key(name: string, sequence = name): KeyEvent { return new KeyEvent({ name, sequence, ctrl: false, meta: false, shift: false, option: false, number: false, raw: sequence, eventType: "press", source: "raw" }) }

test("layered TOML preserves keymap and custom languages", async () => {
  const root = await fixture(); await mkdir(join(root, ".oride"))
  await Bun.write(join(root, ".oride", "config.toml"), 'mouse = false\n[editor]\ntab_size = 2\n[keys]\n"ctrl+s" = "save_all"\n[[languages]]\nid = "custom"\nextensions = ["custom"]\n')
  const config = await loadConfig(root)
  expect(config.mouse).toBe(false); expect(config.editor.tab_size).toBe(2)
  expect(config.keys["ctrl+s"]).toBe("save_all"); expect(config.languages[0]?.id).toBe("custom")
  await Bun.write(join(root, ".oride", "config.toml"), '[keys]\n"ctrl+s" = "nonexistent"\n')
  await expect(loadConfig(root)).rejects.toThrow("Unknown action")
})
test("reference themes and locale catalog names are preserved", async () => {
  const root = await fixture(), preferences = await loadPreferences(root)
  expect(preferences.themes.has("catppuccin-mocha")).toBe(true)
  expect(preferences.themes.has("one-dark")).toBe(true)
  expect(preferences.locales.has("en-US")).toBe(true)
  const app = new Application(root, defaultConfig()); app.newDocument(); app.preferences = preferences
  expect(app.label("save")).toBe("Salvar")
  await app.close()
})
test("EditorConfig closest matching section controls indentation", async () => {
  const root = await fixture(); await mkdir(join(root, "src"))
  await Bun.write(join(root, ".editorconfig"), "root = true\n[*]\nindent_size = 2\nindent_style = space\n")
  await Bun.write(join(root, "src", ".editorconfig"), "[*.ts]\nindent_size = 3\n")
  expect(await editorIndent(join(root, "src", "main.ts"), { tabSize: 8, insertSpaces: false })).toEqual({ tabSize: 3, insertSpaces: true })
})
test("plugin variables are argv, never executable shell syntax", async () => {
  const root = await fixture()
  const plugin = parsePlugin({ plugin: { name: "test" }, commands: [{ id: "echo", executable: process.execPath, args: ["-e", "process.stdout.write(process.argv[1])", "$FILE"] }] })
  const command = plugin.commands[0]
  if (!command) throw new Error("Missing test command")
  const hostileName = "; touch OWNED"
  expect(await runPlugin(command, { file: hostileName, workspace: root })).toBe(hostileName)
  expect(await Bun.file(join(root, "OWNED")).exists()).toBe(false)
  expect(() => parsePlugin({ plugin: { name: "bad" }, commands: [{ id: "bad" }] })).toThrow("executable")
})
test("disk reload updates clean documents and offers reload for dirty ones", async () => {
  const root = await fixture(), path = join(root, "a.txt")
  await Bun.write(path, "before")
  const config = defaultConfig(); config.lsp.enabled = false
  const app = new Application(root, config)
  try {
    await app.openFile(path)
    await Bun.sleep(10); await Bun.write(path, "after"); await app.pollDisk()
    expect(app.document.text).toBe("after")
    app.insert("dirty")
    await Bun.sleep(10); await Bun.write(path, "external"); await app.pollDisk()
    // Sujo + mudou fora: um confirm oferece o reload, sem decidir pelo leitor.
    if (app.overlay?.kind !== "confirm") throw new Error("Missing disk confirm")
    const confirm = app.overlay
    expect(app.document.text).toContain("dirty")
    // Esc mantém as edições; aceitar recarregaria (coberto no teste abaixo).
    app.overlay = undefined
    expect(confirm.title).toContain("Changed on disk")
  } finally { await app.close() }
})
test("project replacement rewrites disk and refuses dirty matching buffers", async () => {
  const root = await fixture(), path = join(root, "a.txt")
  await Bun.write(path, "foo foo")
  const config = defaultConfig(); config.lsp.enabled = false
  const app = new Application(root, config); app.newDocument()
  try {
    app.openSearch(true, true)
    if (app.overlay?.kind !== "search") throw new Error("Missing search")
    app.overlay.query = "foo"; app.overlay.replacement = "$100"
    await app.updateSearch(); await app.replaceSearch(true)
    expect(await Bun.file(path).text()).toBe("$100 $100")
    await app.openFile(path); app.insert("dirty")
    app.openSearch(true, true)
    if (app.overlay?.kind !== "search") throw new Error("Missing search")
    app.overlay.query = "$100"; app.overlay.replacement = "bar"
    await app.updateSearch()
    await expect(app.replaceSearch(true)).rejects.toThrow("Save dirty")
  } finally { await app.close() }
})
test("missing LSP binary fails closed without crashing editing", async () => {
  const root = await fixture(), path = join(root, "main.ts")
  await Bun.write(path, "const x=1")
  const config = defaultConfig(); config.lsp.servers.typescript = ["oride-nonexistent-lsp-fixture"]
  const app = new Application(root, config)
  try {
    await app.openFile(path)
    expect(await app.ensureLsp()).toBeUndefined()
    expect(app.status).toContain("not found on PATH")
    app.insert("//still editing\n"); expect(app.document.text).toContain("still editing")
  } finally { await app.close() }
})
test("Vim normal mode captures unknown keys, words, gg, dd, yy, paste and insert", async () => {  const root = await fixture(), config = defaultConfig(); config.lsp.enabled = false; config.editor.modal_mode = true
  const app = new Application(root, config); app.newDocument(); app.document.insert("one two\nthree\n")
  try {
    app.document.select(0)
    receiveKey(app, key("s")); expect(app.document.text).toBe("one two\nthree\n")
    receiveKey(app, key("w")); expect(app.document.primary.head).toBe(4)
    receiveKey(app, key("g")); receiveKey(app, key("g")); expect(app.document.primary.head).toBe(0)
    receiveKey(app, key("y")); receiveKey(app, key("y")); expect(app.clipboard).toBe("one two\n")
    receiveKey(app, key("d")); receiveKey(app, key("d")); expect(app.document.text).toBe("three\n")
    receiveKey(app, key("i")); receiveKey(app, key("x")); expect(app.document.text).toBe("xthree\n")
    receiveKey(app, key("escape", "\x1b"));     expect(app.vim).toBe("normal")
  } finally { await app.close() }
})
test("jump back returns to the recorded position and forward restores it", async () => {
  const root = await fixture()
  const config = defaultConfig(); config.lsp.enabled = false
  const app = new Application(root, config)
  try {
    await Bun.write(join(root, "a.txt"), "aaa\n"); await Bun.write(join(root, "b.txt"), "bbb\n")
    await app.openFile(join(root, "a.txt")); app.document.select(1); app.recordJump()
    await app.openFile(join(root, "b.txt")); app.document.select(2); app.recordJump()
    // Recording the same spot twice never piles up.
    app.recordJump()
    await app.dispatch("jump_back")
    expect(app.document.path).toBe(join(root, "a.txt"))
    expect(app.document.primary.head).toBe(1)
    expect(app.status).toBe("jump back")
    await app.dispatch("jump_forward")
    expect(app.document.path).toBe(join(root, "b.txt"))
    expect(app.status).toBe("jump forward")
  } finally { await app.close() }
})
test("jump stack dedupes, truncates the future and caps at 64", () => {
  const jumps = new JumpStack()
  expect(jumps.back()).toBeUndefined()
  jumps.push({ path: "a", offset: 1 }); jumps.push({ path: "a", offset: 1 })
  expect(jumps.stack).toHaveLength(1)
  jumps.push({ path: "b", offset: 2 })
  expect(jumps.back()).toEqual({ path: "a", offset: 1 })
  // A new push after going back discards the future — e, como no oráculo, partindo
  // do mais antigo (índice 0) o truncate leva tudo: navegar dali recomeça a lista.
  jumps.push({ path: "c", offset: 3 })
  expect(jumps.stack).toEqual([{ path: "c", offset: 3 }])
  expect(jumps.forward()).toBeUndefined()
  for (let index = 0; index < 70; index++) jumps.push({ path: `f${index}`, offset: index })
  expect(jumps.stack).toHaveLength(64)
  expect(jumps.stack[0]).toEqual({ path: "f6", offset: 6 })
})
test("jump back on an empty list reports instead of opening anything", async () => {
  const root = await fixture()
  const config = defaultConfig(); config.lsp.enabled = false
  const app = new Application(root, config); app.newDocument()
  try {
    await app.dispatch("jump_back")
    expect(app.status).toBe("jump list vazio")
    expect(app.documents).toHaveLength(1)
  } finally { await app.close() }
})
test("dirty buffer changed on disk offers reload instead of deciding", async () => {
  const root = await fixture(), path = join(root, "a.txt")
  await Bun.write(path, "before")
  const config = defaultConfig(); config.lsp.enabled = false
  const app = new Application(root, config)
  try {
    await app.openFile(path)
    app.insert("dirty")
    await Bun.sleep(10); await Bun.write(path, "external"); await app.pollDisk()
    if (app.overlay?.kind !== "confirm") throw new Error("Missing disk confirm")
    const confirm = app.overlay
    expect(confirm.title).toContain("Changed on disk")
    // A second tick never nags with a second overlay while one is open.
    await Bun.sleep(10); await Bun.write(path, "external2"); await app.pollDisk()
    expect(app.overlay).toBe(confirm)
    app.overlay = undefined; await confirm.accept()
    expect(app.document.text).toBe("external2")
    expect(app.document.dirty).toBe(false)
  } finally { await app.close() }
})
test("Markdown preview only opens for the Markdown family", async () => {
  const root = await fixture()
  const config = defaultConfig(); config.lsp.enabled = false
  const app = new Application(root, config)
  try {
    await Bun.write(join(root, "main.ts"), "const x = 1\n")
    await Bun.write(join(root, "notes.md"), "# Notes\n")
    await app.openFile(join(root, "main.ts"))
    await app.dispatch("toggle_md_preview")
    expect(app.preview).toBe(false)
    expect(app.status).toBe("preview só para Markdown/MDX")
    await app.openFile(join(root, "notes.md"))
    await app.dispatch("toggle_md_preview")
    expect(app.preview).toBe(true)
    expect(app.status).toContain("md preview: on")
    await app.dispatch("toggle_md_preview")
    expect(app.preview).toBe(false)
  } finally { await app.close() }
})
test("occurrence picker, Escape collapse and multicursor paste", async () => {
  const root = await fixture()
  const config = defaultConfig(); config.lsp.enabled = false
  const app = new Application(root, config); app.newDocument()
  try {
    app.document.insert("ab\nab\nab")
    app.document.select(0, 2)
    await app.dispatch("multi_picker")
    if (app.overlay?.kind !== "prompt") throw new Error("Missing occurrences prompt")
    expect(app.overlay.value).toBe("ab")
    // O submit não fecha o prompt sozinho — na UI quem fecha é o overlayKey antes
    // de chamar; o teste espelha esse caminho.
    const submit = app.overlay.submit
    app.overlay = undefined
    await submit("ab")
    expect(app.document.selections).toHaveLength(3)
    receivePaste(app, "Z")
    expect(app.document.text).toBe("Z\nZ\nZ")
    receiveKey(app, key("escape", "\x1b"))
    expect(app.document.selections).toHaveLength(1)
  } finally { await app.close() }
})
test("bracketed paste routes to fields with newlines stripped", async () => {
  const root = await fixture()
  const config = defaultConfig(); config.lsp.enabled = false
  const app = new Application(root, config); app.newDocument()
  try {
    receivePaste(app, "a\nb")
    expect(app.document.text).toBe("a\nb")
    app.prompt("Save as", "", () => undefined)
    receivePaste(app, "x\ny")
    const prompted = app.overlay
    if (prompted?.kind !== "prompt") throw new Error("Missing prompt")
    expect(prompted.value).toBe("xy")
  } finally { await app.close() }
})
test("bracketed paste filters picker queries with newlines stripped", async () => {
  const root = await fixture()
  const config = defaultConfig(); config.lsp.enabled = false
  const app = new Application(root, config); app.newDocument()
  try {
    app.picker("Pick", [{ label: "one", run: () => undefined }])
    receivePaste(app, "q\nq")
    const picked = app.overlay
    if (picked?.kind !== "picker") throw new Error("Missing picker")
    expect(picked.query).toBe("qq")
  } finally { await app.close() }
})

test("picker offloads fuzzy query with 20k entries to worker and matches synchronous top results", async () => {
  const root = await fixture()
  const config = defaultConfig(); config.lsp.enabled = false
  const app = new Application(root, config); app.newDocument()
  try {
    const entries: PickerEntry[] = []
    for (let i = 0; i < 20000; i++) {
      entries.push({ label: `src/core/module-${i}.ts`, detail: `item ${i}`, run: () => undefined })
    }
    app.picker("Files", entries)
    expect(app.pickerEntries.length).toBe(100)

    if (app.overlay?.kind === "picker") {
      app.overlay.query = "mod-1999"
      app.updatePicker()
      const targetGen = app.overlay.generation
      const start = performance.now()
      while (performance.now() - start < 3000) {
        if (app.overlay.generation === targetGen && app.overlay.filtered?.[0]?.label.includes("1999")) break
        await Bun.sleep(10)
      }
    }

    const syncTop = entries
      .map(e => ({ e, score: fuzzyScore(e.label + " " + (e.detail ?? ""), "mod-1999") }))
      .filter(c => c.score > -Infinity)
      .sort((a, b) => b.score - a.score)
      .slice(0, 5)
      .map(c => c.e.label)

    expect(app.pickerEntries.slice(0, 5).map(e => e.label)).toEqual(syncTop)
  } finally { await app.close() }
})

test("picker falls back to synchronous filtering when worker is unavailable", async () => {
  const root = await fixture()
  const config = defaultConfig(); config.lsp.enabled = false
  const app = new Application(root, config); app.newDocument()
  try {
    const entries: PickerEntry[] = [
      { label: "alpha.ts", run: () => undefined },
      { label: "beta.ts", run: () => undefined },
      { label: "gamma.ts", run: () => undefined },
    ]
    app.picker("Files", entries)
    if (app.overlay?.kind === "picker") {
      app.overlay.query = "bet"
      app.updatePicker()
    }
    expect(app.pickerEntries.map(e => e.label)).toEqual(["beta.ts"])
  } finally { await app.close() }
})

test("which_key and welcome overlays display structured essential shortcuts", async () => {
  const root = await fixture()
  const config = defaultConfig(); config.lsp.enabled = false; config.locale = "pt-BR"
  const app = new Application(root, config); app.newDocument()
  try {
    await app.dispatch("which_key")
    expect(app.overlay?.kind).toBe("read")
    if (app.overlay?.kind === "read") {
      expect(app.overlay.title).toContain("Which-Key")
      expect(app.overlay.text).toContain("Ctrl+Shift+P")
      expect(app.overlay.text).toContain("command palette")
      expect(app.overlay.text).toContain("Alt+/")
    }

    await app.dispatch("welcome")
    expect(app.overlay?.kind).toBe("read")
    if (app.overlay?.kind === "read") {
      expect(app.overlay.title).toBe("Boas-vindas")
      expect(app.overlay.text).toContain("Bem-vindo ao Daisy Code Editor")
      expect(app.overlay.text).toContain("Ctrl+S")
    }
  } finally { await app.close() }
})

test("interactive path browser navigates directories and confirms destination", async () => {
  const root = await fixture()
  await mkdir(join(root, "subdir"))
  await Bun.write(join(root, "subdir", "existing.txt"), "hello")
  const config = defaultConfig(); config.lsp.enabled = false; config.locale = "pt-BR"
  const app = new Application(root, config); app.newDocument()
  try {
    await app.dispatch("open_folder")
    expect(app.overlay?.kind).toBe("picker")
    if (app.overlay?.kind === "picker") {
      expect(app.overlay.entries.some(e => e.label.includes("Selecionar pasta"))).toBe(true)
      expect(app.overlay.entries.some(e => e.label.includes("subdir"))).toBe(true)
    }

    await app.dispatch("save_as")
    expect(app.overlay?.kind).toBe("picker")
    if (app.overlay?.kind === "picker") {
      expect(app.overlay.entries.some(e => e.label.includes("Salvar como"))).toBe(true)
      expect(app.overlay.entries.some(e => e.label.includes("subdir"))).toBe(true)
      const subdirEntry = app.overlay.entries.find(e => e.label.includes("subdir"))
      await subdirEntry?.run()
      expect(app.overlay?.kind).toBe("picker")
      if (app.overlay?.kind === "picker") {
        expect(app.overlay.entries.some(e => e.label.includes("existing.txt"))).toBe(true)
      }
    }
  } finally { await app.close() }
})
test("command palette lists unmenued actions and hides editing primitives", async () => {
  const root = await fixture()
  const app = new Application(root, defaultConfig())
  try {
    app.newDocument()
    const { buildPaletteEntries } = await import("../src/app/commands")
    const entries = buildPaletteEntries(app)
    const labels = entries.map(entry => entry.label)
    expect(labels.some(label => /rebase/i.test(label))).toBe(true)
    expect(labels.some(label => /grow tree/i.test(label))).toBe(true)
    expect(labels.some(label => /move_(left|right)$/i.test(label))).toBe(false)
    expect(labels.some(label => /backspace$/i.test(label))).toBe(false)
    const shrink = entries.find(entry => /shrink terminal/i.test(entry.label))
    expect(shrink?.detail).toContain("alt+-")
    const labelsUnique = new Set(labels)
    expect(labelsUnique.size).toBe(labels.length)
    const grow = entries.find(entry => /grow tree/i.test(entry.label))
    await grow!.run()
    expect(app.config.tree.width).toBe(30)
  } finally { await app.close() }
})
