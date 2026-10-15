import { stat, readdir, mkdir } from "node:fs/promises"
import { dirname, join, resolve, basename } from "node:path"
import { fileURLToPath, pathToFileURL } from "node:url"
import { Document, offsetAt, positionAt } from "../core/document"
import { type ActionId } from "../core/keymap"
import { fuzzyScore, findMatches, searchDefaults, replacementEdits, type SearchOptions, type Match } from "../core/search"
import { loadConfig, userConfigDir, type Config } from "../services/config"
import { FileTree, atomicWrite, discoverFiles, readText, loadSession, saveSession, type SplitSession } from "../services/files"
import { Git, hunkSummary, type GitStatus, type GutterKind, type GitHunk } from "../services/git"
import { highlight, languageOf, type HighlightToken } from "../services/languages"
import { TerminalSession } from "../services/terminal"
import { LspClient, lspRetryAllowed } from "../services/lsp"
import { DebugAdapter } from "../services/dap"
import { discoverPlugins, runPlugin, type Plugin, type PluginCommand } from "../services/plugins"
import { loadUserSnippets } from "../services/snippets"
import { commands, type Command } from "./commands"
import { builtInThemes, builtInLocales, t } from "../services/catalogs"
import { loadPreferences, labelOf, editorIndent, type Preferences } from "../services/preferences"
import { extractBufferWords, suggestBufferWords, wordAtOffset, extractDocumentSymbols, findSymbol } from "../services/symbols"
import { getLanguageCatalog, lookupCatalogItem } from "../services/languageCatalog"
import type { FuzzyWorkerRequest, FuzzyWorkerResponse } from "../workers/fuzzy"

export type Focus = "editor" | "tree" | "terminal" | "scm"
export type PickerEntry = { label: string; detail?: string; run: () => void | Promise<void> }
export type Overlay =
  | { kind: "picker"; title: string; query: string; selected: number; entries: PickerEntry[]; onSelect?: (entry: PickerEntry, index: number) => void; onCancel?: () => void; filtered?: PickerEntry[]; generation?: number }
  | { kind: "prompt"; title: string; value: string; submit: (value: string) => void | Promise<void> }
  | { kind: "read"; title: string; text: string; scroll: number }
  | { kind: "confirm"; title: string; accept: () => void | Promise<void> }
  | { kind: "search"; project: boolean; field: number; query: string; replacement: string; glob: string; replace: boolean; selected: number; hits: ProjectHit[]; error: string; generation: number }
export type ProjectHit = { path: string; line: number; column: number; text: string; match: Match }
export type Pane = { documentId: number; scroll: number; scrollX: number }
export type Jump = { path: string; offset: number }

export type CompletionItem = {
  label: string
  detail?: string
  kind?: string
  doc?: string
  insertText?: string
  run?: () => void
}

export type CompletionState = {
  word: string
  offset: number
  selected: number
  items: CompletionItem[]
}

export type CursorScreenPos = {
  x: number
  y: number
  boxWidth: number
  boxHeight: number
  boxX: number
  boxY: number
}

// Pilha de saltos com índice (para forward), espelhando
// `crates/oride-app/src/jump_list.rs`: push descarta repetição consecutiva,
// trunca o futuro e limita a 64; o primeiro back pula a posição atual que o
// jump_back acabou de registrar.
export class JumpStack {
  stack: Jump[] = []
  index = 0
  push(entry: Jump): void {
    const last = this.stack[this.index - 1]
    if (last?.path === entry.path && last.offset === entry.offset) return
    if (this.index < this.stack.length) this.stack.length = this.index
    this.stack.push(entry)
    if (this.stack.length > 64) this.stack.shift()
    this.index = this.stack.length
  }
  back(): Jump | undefined {
    if (this.index === 0 || !this.stack.length) return undefined
    if (this.index >= this.stack.length) this.index = this.stack.length - 1
    if (this.index === 0) return this.stack[0]
    this.index -= 1
    return this.stack[this.index]
  }
  forward(): Jump | undefined {
    if (this.index + 1 >= this.stack.length) return undefined
    this.index += 1
    return this.stack[this.index]
  }
}

export function flattenLspSymbols(app: import("./application").Application, symbols: Array<import("../services/lsp").DocumentSymbol | import("../services/lsp").SymbolInformation>, fallbackPath: string): Array<{ label: string; detail?: string; run: () => void | Promise<void> }> {
  const entries: PickerEntry[] = []
  const visit = (symbol: any, depth = 0) => {
    const name = typeof symbol?.name === "string" ? symbol.name : "symbol"
    const range = symbol?.selectionRange ?? symbol?.range
    const uri = typeof symbol?.location?.uri === "string" ? symbol.location.uri : fallbackPath
    const left = typeof uri === "string" && uri.startsWith("file://") ? fileURLToPath(uri) : String(uri)
    if (range) entries.push({
      label: `${"  ".repeat(depth)}${name}`,
      detail: String(symbol.kind ?? symbol.kind ?? "symbol"),
      run: () => { void (async () => { await app.openFile(left); app.document.select(offsetAt(app.document.text, range.start.line, range.start.character)); app.recordJump(); app.changed() })() },
    })
    for (const child of Array.isArray(symbol?.children) ? symbol.children : []) visit(child, depth + 1)
  }
  for (const symbol of symbols) visit(symbol)
  return entries.length > 200 ? entries.slice(0, 200) : entries
}

export class Application {
  config: Config
  preferences: Preferences = { themes: new Map(Object.entries(builtInThemes)), locales: new Map(Object.entries(builtInLocales)) }
  workspace: string
  tree: FileTree
  git: Git
  gitStatus: GitStatus = { branch: "", ahead: 0, behind: 0, entries: [] }
  gitGutters = new Map<string, Map<number, GutterKind>>()
  scmSelected = 0
  showTree = true
  showScm = false
  bottom: "terminal" | "problems" | undefined
  terminal: TerminalSession | undefined
  terminalHeight: number
  debugBreakpoints = new Map<string, Array<{ line: number; condition?: string }>>()
  debug: DebugAdapter | undefined
  debugStatus: string = "idle"
  focus: Focus = "editor"
  documents: Document[] = []
  activeId = 0
  panes: Pane[] = []
  focusedPane = 0
  orientation: "vertical" | "horizontal" = "vertical"
  splitRatio = 50
  preview = false
  previewScroll = 0
  overlay: Overlay | undefined
  completion: CompletionState | undefined
  cursorScreen: CursorScreenPos | undefined
  menu: { column: number; selected: number } | undefined
  altActive = false
  status = ""
  quit = false
  clipboard = ""
  clipboardWrite = ""
  readClipboard: (() => Promise<string | undefined>) | undefined
  findQuery = ""
  findReplacement = ""
  searchOptions: SearchOptions = searchDefaults()
  matches: Match[] = []
  matchIndex = 0
  editorHeight = 20
  width = 100
  height = 30
  vim: "normal" | "insert" | "visual" | "visual-line" = "insert"
  vimPending = ""
  vimLineAnchor = 0
  highlights = new Map<number, { version: number; path?: string; theme: string; tokens: HighlightToken[] }>()
  readonly servers = new Map<string, LspClient>()
  readonly serverFailures = new Map<string, string>()
  readonly lspFailures = new Map<string, number>()
  readonly lspRetryAt = new Map<string, number>()
  private starting = new Map<string, Promise<LspClient>>()
  private listeners = new Set<() => void>()
  private nextId = 1
  private lspTimer: ReturnType<typeof setTimeout> | undefined
  private completionTimer: ReturnType<typeof setTimeout> | undefined
  private fuzzyWorker?: Worker
  private fuzzyTimer?: ReturnType<typeof setTimeout>
  private closed = false
  private jumpStack = new JumpStack()
  private gitReading = false
  private pollingDisk = false
  private diskVersions = new Map<string, number>()
  private diskConfirmPath: string | undefined
  plugins: Plugin[] = []
  private trustedPlugins = new Set<string>()
  private filesPromise: Promise<string[]> | undefined

  constructor(workspace: string, config: Config) {
    this.workspace = workspace; this.config = config
    this.tree = new FileTree(workspace, config.tree.show_hidden)
    this.git = new Git(workspace)
    this.terminalHeight = config.terminal.default_height
    this.vim = config.editor.modal_mode ? "normal" : "insert"
  }

  static async open(paths: string[], restore = true): Promise<Application> {
    const first = resolve(paths[0] ?? process.cwd())
    const directory = await stat(first).then(info => info.isDirectory()).catch(() => false)
    const workspace = paths.length ? directory ? first : dirname(first) : process.cwd()
    const app = new Application(workspace, await loadConfig(workspace))
    app.preferences = await loadPreferences(workspace)
    app.plugins = await discoverPlugins(workspace)
    await app.tree.refresh().catch(error => { app.status = String(error) })
    const previous = restore && !paths.length ? await loadSession(workspace) : {}
    const files = paths.length ? paths.filter(path => resolve(path) !== workspace) : previous.files ?? []
    for (const path of files) {
      try { await app.openFile(resolve(path)) } catch (error) { if (paths.length) throw error; app.status = String(error) }
    }
    if (!app.documents.length) app.newDocument()
    if (previous.active_index !== undefined) app.activate(app.documents[previous.active_index]?.id ?? app.activeId)
    if (previous.scroll_y !== undefined && app.panes[0]) app.panes[0].scroll = previous.scroll_y
    if (previous.split) {
      app.orientation = previous.split.orientation
      app.splitRatio = previous.split.ratio_percent
      if (previous.split.secondary_file) {
        const secPath = resolve(previous.split.secondary_file)
        let secDoc = app.documents.find(d => d.path === secPath)
        if (!secDoc) {
          try {
            await app.openFile(secPath)
            secDoc = app.documents.find(d => d.path === secPath)
          } catch {}
        }
        if (secDoc) {
          app.panes = [
            { documentId: app.activeId, scroll: previous.scroll_y ?? 0, scrollX: 0 },
            { documentId: secDoc.id, scroll: 0, scrollX: 0 }
          ]
          app.focusedPane = 0
        }
      } else if (app.documents.length > 0) {
        app.split(previous.split.orientation)
        app.focusedPane = 0
      }
    }
    app.showTree = previous.show_tree ?? true
    app.showScm = previous.show_scm ?? false
    app.config.tree.width = Math.max(8, previous.tree_width ?? app.config.tree.width)
    await app.refreshGit()
    return app
  }

  label(action: ActionId): string { return labelOf(action, commands[action].label, this.config, this.preferences) }
  menuLabel(action: ActionId): string {
    const base = this.label(action)
    switch (action) {
      case "toggle_soft_wrap":
        return `[${this.config.soft_wrap ? "✓" : " "}] ${base}`
      case "toggle_auto_pairs":
        return `[${this.config.editor.auto_pairs ? "✓" : " "}] ${base}`
      case "toggle_line_numbers":
        return `[${this.config.show_line_numbers ? "✓" : " "}] ${base}`
      case "toggle_modal":
        return `[${this.config.editor.modal_mode ? "✓" : " "}] ${base}`
      case "toggle_icons":
        return `[${this.config.icons ? "✓" : " "}] ${base}`
      case "toggle_mouse":
        return `[${this.config.mouse ? "✓" : " "}] ${base}`
      default:
        return base
    }
  }
  t(section: "dialogs" | "prompts" | "messages" | "status" | "palette", key: string, params?: Record<string, string>): string {
    return t(this.config.locale, section, key, params)
  }
  get document(): Document {
    const document = this.documents.find(document => document.id === this.activeId)
    if (!document) throw new Error("No active document")
    return document
  }
  subscribe(listener: () => void): () => void { this.listeners.add(listener); return () => this.listeners.delete(listener) }
  changed(): void { if (!this.closed) for (const listener of this.listeners) listener() }
  report(error: unknown): void { this.status = error instanceof Error ? error.message : String(error); this.changed() }
  async guard(operation: () => void | Promise<void>): Promise<void> { try { await operation() } catch (error) { this.report(error) } }

  newDocument(): void {
    const document = new Document(this.nextId++)
    this.documents.push(document); this.activate(document.id)
  }
  activate(id: number): void {
    if (!this.documents.some(document => document.id === id)) return
    this.activeId = id
    if (!this.panes.length) this.panes.push({ documentId: id, scroll: 0, scrollX: 0 })
    else { const pane = this.panes[this.focusedPane]; if (pane) pane.documentId = id }
    this.focus = "editor"; this.afterEdit(); void this.refreshGitGutter()
  }
  async openFile(path: string, offset?: number): Promise<void> {
    path = resolve(path)
    let document = this.documents.find(document => document.path === path)
    if (!document) {
      document = new Document(this.nextId++, await readText(path), path)
      if (this.config.editor.use_editorconfig) document.indent = await editorIndent(path, { tabSize: this.config.editor.tab_size, insertSpaces: this.config.editor.insert_spaces })
      this.documents.push(document)
      this.diskVersions.set(path, (await stat(path)).mtimeMs)
      this.offerHooks("on_open", path)
    }
    this.activate(document.id)
    if (offset !== undefined) document.select(offset)
    this.overlay = undefined
    this.afterEdit()
  }
  async save(document = this.document, path = document.path): Promise<void> {
    if (!path) { this.prompt(this.t("prompts", "save_as"), "", async value => this.save(document, resolve(this.workspace, value))); return }
    if (document.path !== path && await Bun.file(path).exists()) {
      this.overlay = { kind: "confirm", title: `Overwrite ${path}? Enter confirms · Esc cancels`, accept: async () => this.writeDocument(document, path) }
      this.changed(); return
    }
    if (this.config.editor.format_on_save) {
      const active = this.activeId
      try { this.activeId = document.id; await this.format() } finally { this.activeId = active }
    }
    await this.writeDocument(document, path)
  }
  private async writeDocument(document: Document, path: string): Promise<void> {
    const text = document.text
    await atomicWrite(path, text)
    document.path = path; document.savedText = text; document.commit()
    if (this.diskConfirmPath === path) this.diskConfirmPath = undefined
    this.diskVersions.set(path, (await stat(path)).mtimeMs)
    this.offerHooks("on_save", path)
    const server = this.servers.get(languageOf(path, this.config))
    if (server?.running) await server.saved(path)
    this.status = this.t("messages", "saved", { path })
    if (path.endsWith("config.toml")) {
      await this.reloadConfig()
    }
    await this.tree.refresh(); this.invalidateFiles(); this.afterEdit(); await this.refreshGit()
  }
  closeTab(targetId = this.activeId): void {
    const document = this.documents.find(candidate => candidate.id === targetId) ?? this.document
    const close = async () => {
      if (document.path) await this.servers.get(languageOf(document.path, this.config))?.closeDocument(document.path)
      this.documents = this.documents.filter(candidate => candidate !== document)
      this.highlights.delete(document.id)
      if (!this.documents.length) this.newDocument()
      else if (this.activeId === document.id) this.activate(this.documents.at(-1)?.id ?? 0)
      for (const pane of this.panes) if (pane.documentId === document.id) pane.documentId = this.activeId
      this.changed()
    }
    if (document.dirty) this.overlay = { kind: "confirm", title: this.t("dialogs", "discard_buffer_dirty", { title: document.title }), accept: close }
    else void this.guard(close)
  }
  cycleTab(delta: number): void {
    const index = this.documents.findIndex(document => document.id === this.activeId)
    this.activate(this.documents[(index + delta + this.documents.length) % this.documents.length]?.id ?? this.activeId)
  }
  split(orientation: "vertical" | "horizontal"): void {
    this.orientation = orientation
    // No máximo 2 panes, como o oráculo: dividir de novo reorienta, sem empilhar.
    if (this.panes.length < 2) this.panes.push({ documentId: this.activeId, scroll: 0, scrollX: 0 })
    this.focusedPane = this.panes.length - 1
  }
  focusPane(index: number): void {
    this.focusedPane = index % this.panes.length
    this.activeId = this.panes[this.focusedPane]?.documentId ?? this.activeId
    this.focus = "editor"; this.changed()
  }
  closePane(): void {
    if (this.panes.length > 1) this.panes.splice(this.focusedPane, 1)
    this.focusPane(Math.min(this.focusedPane, this.panes.length - 1))
  }
  centerCursor(): void {
    const pane = this.panes[this.focusedPane]
    if (!pane) return
    const line = this.document.position.line
    pane.scroll = Math.max(0, line - Math.floor(this.editorHeight / 2))
    this.changed()
  }
  scrollPreview(delta: number): void {
    this.previewScroll = Math.max(0, this.previewScroll + delta)
    this.changed()
  }
  scrollPreviewTo(y: number): void {
    this.previewScroll = Math.max(0, y)
    this.changed()
  }

  async dispatch(action: ActionId): Promise<void> {
    await this.guard(async () => {
      const command: Command = commands[action]
      if (!command) throw new Error(`Unknown action: ${action}`)
      this.menu = undefined
      this.altActive = false
      await command.run(this)
      this.afterEdit()
    })
  }
  insert(text: string): void { this.document.insert(text); this.afterEdit() }

  typeChar(char: string): void {
    if (!this.config.editor.auto_pairs) {
      this.insert(char)
      return
    }
    const PAIRS: Record<string, string> = {
      "(": ")",
      "[": "]",
      "{": "}",
      '"': '"',
      "'": "'",
      "`": "`",
    }
    const CLOSING = new Set([")", "]", "}", '"', "'", "`"])

    // 1. Envolver seleções ativas com os delimitadores
    const hasSelection = this.document.selections.some(s => s.anchor !== s.head)
    if (hasSelection && char in PAIRS) {
      const close = PAIRS[char]!
      const edits = this.document.selections.map(s => {
        const start = Math.min(s.anchor, s.head)
        const end = Math.max(s.anchor, s.head)
        return {
          start,
          end,
          text: char + this.document.text.slice(start, end) + close
        }
      })
      this.document.applyEdits(edits, "surround")
      this.afterEdit()
      return
    }

    // 2. Pular caractere de fechamento se o próximo caractere for idêntico
    if (CLOSING.has(char)) {
      const head = this.document.primary.head
      const nextChar = this.document.text[head]
      if (nextChar === char) {
        this.document.move("right")
        this.afterEdit()
        return
      }
    }

    // 3. Inserir par automático com cursor posicionado no centro
    if (char in PAIRS) {
      const close = PAIRS[char]!
      const head = this.document.primary.head
      const prevChar = head > 0 ? this.document.text[head - 1] : ""
      const nextChar = head < this.document.text.length ? this.document.text[head] : ""

      // Para aspas: não duplica se precedido ou seguido por caractere de palavra
      if (char === '"' || char === "'" || char === "`") {
        if (prevChar && /[\p{L}\p{N}_]/u.test(prevChar)) {
          this.insert(char)
          return
        }
        if (nextChar && /[\p{L}\p{N}_]/u.test(nextChar)) {
          this.insert(char)
          return
        }
      } else {
        // Para parênteses/chaves/colchetes: não duplica se seguido por alfanumérico
        if (nextChar && /[\p{L}\p{N}]/u.test(nextChar)) {
          this.insert(char)
          return
        }
      }

      this.document.insert(char + close)
      this.document.move("left")
      this.afterEdit()
      return
    }

    this.insert(char)
  }

  backspace(): void {
    if (this.config.editor.auto_pairs) {
      const canDeletePairs = this.document.selections.length > 0 && this.document.selections.every(s => {
        if (s.anchor !== s.head) return false
        const head = s.head
        if (head <= 0 || head >= this.document.text.length) return false
        const prev = this.document.text[head - 1]
        const next = this.document.text[head]
        return (
          (prev === "(" && next === ")") ||
          (prev === "[" && next === "]") ||
          (prev === "{" && next === "}") ||
          (prev === '"' && next === '"') ||
          (prev === "'" && next === "'") ||
          (prev === "`" && next === "`")
        )
      })
      if (canDeletePairs) {
        this.document.applyEdits(
          this.document.selections.map(s => ({
            start: s.head - 1,
            end: s.head + 1,
            text: ""
          })),
          "backspace"
        )
        this.afterEdit()
        if (this.completion) {
          const prefix = this.document.text.slice(0, this.document.primary.head).match(/[\p{L}\p{N}_]+$/u)?.[0] ?? ""
          if (prefix.length >= this.config.editor.completion_min_chars) void this.complete(true)
          else this.cancelCompletion()
        }
        return
      }
    }
    this.document.remove(true)
    this.afterEdit()
    if (this.completion) {
      const prefix = this.document.text.slice(0, this.document.primary.head).match(/[\p{L}\p{N}_]+$/u)?.[0] ?? ""
      if (prefix.length >= this.config.editor.completion_min_chars) void this.complete(true)
      else this.cancelCompletion()
    }
  }

  async reloadConfig(): Promise<void> {
    this.config = await loadConfig(this.workspace)
    this.preferences = await loadPreferences(this.workspace)
    this.status = this.t("messages", "config_reloaded")
    this.highlights.clear()
    await this.refreshAllHighlights()
    this.changed()
  }

  async openUserConfig(): Promise<void> {
    await mkdir(userConfigDir, { recursive: true })
    const configPath = join(userConfigDir, "config.toml")
    const file = Bun.file(configPath)
    if (!await file.exists()) {
      const template = [
        "# Daisy Code Editor — Configuração do Usuário",
        "# Modificações têm efeito imediato ao salvar este arquivo.",
        "",
        'theme = "default"',
        'locale = "pt-BR"',
        "mouse = true",
        "soft_wrap = true",
        "show_line_numbers = true",
        "icons = false",
        "",
        "[editor]",
        "tab_size = 4",
        "insert_spaces = true",
        "auto_pairs = true",
        "modal_mode = false",
        "completion_auto = true",
        "completion_min_chars = 2",
        "",
        "[tree]",
        "width = 28",
        "show_hidden = false",
        "",
        "[terminal]",
        "default_height = 10",
        "",
        "# Atalhos personalizados:",
        "# [keys]",
        '# "alt+z" = "toggle_soft_wrap"',
        ""
      ].join("\n")
      await Bun.write(configPath, template)
    }
    await this.openFile(configPath)
    this.status = this.t("messages", "user_config_opened")
    this.changed()
  }

  async openWorkspaceConfig(): Promise<void> {
    const legacyPath = join(this.workspace, ".oride", "config.toml")
    const dir = await Bun.file(legacyPath).exists() ? join(this.workspace, ".oride") : join(this.workspace, ".daisy")
    await mkdir(dir, { recursive: true })
    const configPath = join(dir, "config.toml")
    const file = Bun.file(configPath)
    if (!await file.exists()) {
      const template = [
        "# Daisy Code Editor — Configuração do Workspace (.daisy/config.toml)",
        "# Sobrescreve as configurações globais para este repositório.",
        "",
        '# theme = "catppuccin-mocha"',
        "# soft_wrap = true",
        "# icons = true",
        "",
        "# [editor]",
        "# tab_size = 2",
        "# insert_spaces = true",
        "# auto_pairs = true",
        ""
      ].join("\n")
      await Bun.write(configPath, template)
    }
    await this.openFile(configPath)
    this.status = this.t("messages", "workspace_config_opened")
    this.changed()
  }
  afterEdit(): void {
    if (this.closed) return
    if (!this.documents.length) {
      this.changed()
      return
    }
    this.recomputeFind()
    void this.refreshHighlight()
    clearTimeout(this.lspTimer); clearTimeout(this.completionTimer)
    if (this.config.lsp.enabled && this.document.path) this.lspTimer = setTimeout(() => { void this.guard(async () => { await this.ensureLsp() }) }, 200)
    this.changed()
  }
  private async refreshHighlight(): Promise<void> {
    if (!this.documents.length) return
    const document = this.document
    const { id, version, path, text } = document, theme = this.config.theme
    const previous = this.highlights.get(id)
    if (previous?.version === version && previous.path === path && previous.theme === theme) return
    try {
      const tokens = await highlight(text, languageOf(path, this.config), theme, this.config.highlighting.engine)
      if (document.version === version && document.path === path && this.config.theme === theme && !this.closed) {
        this.highlights.set(id, { version, path, theme, tokens }); this.changed()
      }
    } catch (error) { this.report(error) }
  }

  async refreshAllHighlights(): Promise<void> {
    for (const doc of this.documents) {
      try {
        const tokens = await highlight(doc.text, languageOf(doc.path, this.config), this.config.theme, this.config.highlighting.engine)
        this.highlights.set(doc.id, { version: doc.version, path: doc.path, theme: this.config.theme, tokens })
      } catch (error) { this.report(error) }
    }
    this.changed()
  }

  prompt(title: string, value: string, submit: (value: string) => void | Promise<void>): void {
    this.cancelCompletion()
    this.overlay = { kind: "prompt", title, value, submit }; this.changed()
  }
  private getFuzzyWorker(): Worker | undefined {
    if (this.fuzzyWorker) return this.fuzzyWorker
    try {
      this.fuzzyWorker = new Worker(new URL("../workers/fuzzy.ts", import.meta.url))
      this.fuzzyWorker.onmessage = (event: MessageEvent<FuzzyWorkerResponse>) => {
        const data = event.data
        if (!data || data.type !== "fuzzy:result") return
        const overlay = this.overlay
        if (overlay?.kind === "picker" && overlay.generation === data.id) {
          overlay.filtered = data.items.map(item => overlay.entries[item.index]).filter((e): e is PickerEntry => e !== undefined)
          if (overlay.selected >= overlay.filtered.length) {
            overlay.selected = Math.max(0, overlay.filtered.length - 1)
          }
          if (overlay.onSelect && overlay.filtered[overlay.selected]) {
            overlay.onSelect(overlay.filtered[overlay.selected]!, overlay.selected)
          }
          this.changed()
        }
      }
      this.fuzzyWorker.onerror = () => {
        try { this.fuzzyWorker?.terminate() } catch {}
        this.fuzzyWorker = undefined
      }
    } catch {
      this.fuzzyWorker = undefined
    }
    return this.fuzzyWorker
  }

  picker(title: string, entries: PickerEntry[], onSelect?: (entry: PickerEntry, index: number) => void, onCancel?: () => void): void {
    this.cancelCompletion()
    const filtered = entries.length > 2000 ? entries.slice(0, 100) : undefined
    this.overlay = { kind: "picker", title, entries, query: "", selected: 0, generation: 0, filtered, onSelect, onCancel }
    this.changed()
  }

  updatePicker(): void {
    const overlay = this.overlay
    if (overlay?.kind !== "picker") return
    clearTimeout(this.fuzzyTimer)
    const generation = (overlay.generation ?? 0) + 1
    overlay.generation = generation

    if (overlay.entries.length <= 2000) {
      overlay.filtered = undefined
      this.changed()
      return
    }

    const worker = this.getFuzzyWorker()
    if (!worker) {
      overlay.filtered = undefined
      this.changed()
      return
    }

    this.fuzzyTimer = setTimeout(() => {
      if (this.overlay !== overlay || overlay.generation !== generation) return
      const targets = overlay.entries.map((entry, index) => ({
        index,
        target: entry.label + " " + (entry.detail ?? ""),
      }))
      worker.postMessage({
        id: generation,
        type: "fuzzy",
        query: overlay.query,
        entries: targets,
        limit: 100,
      } satisfies FuzzyWorkerRequest)
    }, 50)
  }

  get pickerEntries(): PickerEntry[] {
    const overlay = this.overlay
    if (overlay?.kind !== "picker") return []
    if (overlay.filtered) return overlay.filtered
    if (!overlay.query) return overlay.entries
    return overlay.entries.map(entry => ({ entry, score: fuzzyScore(entry.label + " " + (entry.detail ?? ""), overlay.query) })).filter(candidate => candidate.score > -Infinity).sort((left, right) => right.score - left.score).map(candidate => candidate.entry)
  }
  invalidateFiles(): void { this.filesPromise = undefined }
  files(): Promise<string[]> { return this.filesPromise ??= discoverFiles(this.workspace, this.config.tree.show_hidden).catch(error => { this.filesPromise = undefined; throw error }) }
  async finder(): Promise<void> {
    const files = await this.files()
    this.picker("Open file", files.map(path => ({ label: path, run: () => this.openFile(join(this.workspace, path)) })))
  }

  async changeWorkspace(path: string): Promise<void> {
    const change = async () => {
      await this.terminal?.close(); this.terminal = undefined
      await this.debug?.disconnect().catch(() => undefined); this.debug = undefined
      await Promise.all([...this.servers.values()].map(server => server.close())); this.servers.clear(); this.serverFailures.clear()
      this.workspace = path; this.config = await loadConfig(path); this.preferences = await loadPreferences(path); this.tree = new FileTree(path, this.config.tree.show_hidden)
      const { Git } = await import("../services/git"); this.git = new Git(path)
      this.documents = []; this.panes = []; this.focusedPane = 0; this.newDocument(); this.invalidateFiles()
      await this.tree.refresh(); await this.refreshGit(); this.changed()
    }
    if (this.documents.some(document => document.dirty)) {
      this.overlay = { kind: "confirm", title: this.t("dialogs", "change_workspace_dirty"), accept: change }
    } else {
      await change()
    }
  }

  async openPathBrowser(mode: "folder" | "save_as" | "file", startDir = this.workspace, defaultName = ""): Promise<void> {
    let currentDir = resolve(startDir)
    let currentName = defaultName
    const refresh = async () => {
      const isPt = this.config.locale === "pt-BR"
      const entries: PickerEntry[] = []
      if (mode === "folder") {
        entries.push({
          label: isPt ? `📁 [Selecionar pasta: ${basename(currentDir) || currentDir}]` : `📁 [Select folder: ${basename(currentDir) || currentDir}]`,
          detail: currentDir,
          run: () => void this.changeWorkspace(currentDir),
        })
      } else if (mode === "save_as") {
        const targetPath = join(currentDir, currentName || "untitled.txt")
        entries.push({
          label: isPt ? `💾 [Salvar como: ${currentName || "untitled.txt"}]` : `💾 [Save as: ${currentName || "untitled.txt"}]`,
          detail: targetPath,
          run: () => void this.save(this.document, targetPath),
        })
        entries.push({
          label: isPt ? `✏️ [Digitar nome do arquivo…]` : `✏️ [Type filename…]`,
          detail: currentName,
          run: () => {
            this.prompt(this.t("prompts", "save_as"), currentName, value => {
              currentName = value.trim()
              void refresh()
            })
          },
        })
      }

      const parent = dirname(currentDir)
      if (parent !== currentDir) {
        entries.push({
          label: "📁 ..",
          detail: isPt ? "subir um nível" : "go up one level",
          run: () => { currentDir = parent; return refresh() },
        })
      }

      try {
        const dirEntries = await readdir(currentDir, { withFileTypes: true })
        const sorted = dirEntries
          .filter(e => !e.name.startsWith(".") && e.name !== "node_modules" && e.name !== "target")
          .sort((a, b) => {
            if (a.isDirectory() && !b.isDirectory()) return -1
            if (!a.isDirectory() && b.isDirectory()) return 1
            return a.name.localeCompare(b.name)
          })

        for (const entry of sorted) {
          if (entry.isDirectory()) {
            const subPath = join(currentDir, entry.name)
            entries.push({
              label: `📁 ${entry.name}/`,
              detail: subPath,
              run: () => { currentDir = subPath; return refresh() },
            })
          } else if (mode === "file") {
            const filePath = join(currentDir, entry.name)
            entries.push({
              label: `📄 ${entry.name}`,
              detail: filePath,
              run: () => void this.openFile(filePath),
            })
          } else if (mode === "save_as") {
            entries.push({
              label: `📄 ${entry.name}`,
              detail: isPt ? "substituir este arquivo" : "overwrite this file",
              run: () => {
                currentName = entry.name
                return refresh()
              },
            })
          }
        }
      } catch (error) {
        this.report(error)
      }

      const title = mode === "folder"
        ? `${this.t("prompts", "workspace")}: ${currentDir}`
        : `${this.t("prompts", "save_as")}: ${currentDir}`
      this.picker(title, entries)
    }
    await refresh()
  }

  openSearch(project: boolean, replace = false): void {
    this.cancelCompletion()
    this.overlay = { kind: "search", project, replace, field: 0, query: this.document.selectedText || this.findQuery, replacement: this.findReplacement, glob: "", selected: 0, hits: [], error: "", generation: 0 }
    void this.updateSearch(); this.changed()
  }
  recomputeFind(): void {
    try { this.matches = findMatches(this.document.text, this.findQuery, this.searchOptions) } catch (error) { this.matches = []; if (this.overlay?.kind === "search") this.overlay.error = String(error) }
  }
  async updateSearch(): Promise<void> {
    const overlay = this.overlay
    if (overlay?.kind !== "search") return
    const generation = ++overlay.generation
    overlay.error = ""
    try {
      if (!overlay.project) { this.findQuery = overlay.query; this.findReplacement = overlay.replacement; this.recomputeFind(); this.changed(); return }
      const hits: ProjectHit[] = []
      const files = await this.files()
      const globs = overlay.glob.split(",").map(value => value.trim()).filter(Boolean).map(pattern => new Bun.Glob(pattern))
      for (const path of files) {
        if (this.overlay !== overlay || overlay.generation !== generation) return
        if (globs.length && !globs.some(glob => glob.match(path) || glob.match(path.split("/").at(-1) ?? path))) continue
        const text = await readText(join(this.workspace, path)).catch(() => undefined)
        if (text === undefined) continue
        for (const match of findMatches(text, overlay.query, this.searchOptions)) {
          const prefix = text.slice(0, match.start), line = prefix.split("\n").length - 1
          hits.push({ path, line, column: match.start - prefix.lastIndexOf("\n") - 1, text: text.split("\n")[line] ?? "", match })
          if (hits.length >= 10000) break
        }
        if (hits.length >= 10000) break
      }
      if (this.overlay === overlay && overlay.generation === generation) { overlay.hits = hits; overlay.selected = Math.max(0, Math.min(overlay.selected, hits.length - 1)); this.changed() }
    } catch (error) { if (this.overlay === overlay) { overlay.error = String(error); this.changed() } }
  }
  findNext(delta: number): void {
    if (!this.matches.length) return
    this.matchIndex = (this.matchIndex + delta + this.matches.length) % this.matches.length
    const match = this.matches[this.matchIndex]
    if (match) this.document.select(match.start, match.end)
    this.changed()
  }
  async replaceSearch(all: boolean): Promise<void> {
    const overlay = this.overlay
    if (overlay?.kind !== "search") return
    if (!overlay.project) {
      const matches = all ? this.matches : this.matches.slice(this.matchIndex, this.matchIndex + 1)
      this.document.applyEdits(replacementEdits(matches, overlay.replacement, this.searchOptions.regex))
      this.afterEdit(); await this.updateSearch(); return
    }
    if (!all) { await this.openSearchHit(); return }
    // Recheck disk and refuse to overwrite unsaved buffers.
    const paths = [...new Set(overlay.hits.map(hit => hit.path))]
    if (this.documents.some(document => document.dirty && document.path && paths.includes(document.path.slice(this.workspace.length + 1)))) throw new Error("Save dirty matching buffers before project replacement")
    let count = 0
    for (const path of paths) {
      const absolute = join(this.workspace, path), text = await readText(absolute)
      const matches = findMatches(text, overlay.query, this.searchOptions)
      const document = new Document(0, text)
      document.applyEdits(replacementEdits(matches, overlay.replacement, this.searchOptions.regex))
      await atomicWrite(absolute, document.text)
      const opened = this.documents.find(document => document.path === absolute)
      if (opened) { opened.applyEdits([{ start: 0, end: opened.text.length, text: document.text }]); opened.savedText = opened.text }
      count += matches.length
    }
    this.status = `Replaced ${count} occurrences in ${paths.length} files`
    await this.updateSearch(); this.afterEdit()
  }
  async openSearchHit(): Promise<void> {
    const overlay = this.overlay
    if (overlay?.kind !== "search") return
    if (!overlay.project) { this.findNext(1); return }
    const hit = overlay.hits[overlay.selected]
    if (hit) { this.recordJump(); await this.openFile(join(this.workspace, hit.path)); this.document.select(offsetAt(this.document.text, hit.line, hit.column)); this.changed() }
  }

  async gitStash(): Promise<void> {
    this.status = await this.git.stash(); this.changed(); await this.refreshGit()
  }
  async gitStashPop(): Promise<void> {
    this.status = await this.git.stashPop(); this.changed(); await this.refreshGit()
  }
  async gitStageHunk(): Promise<void> {
    const path = this.document.path
    if (!path) { this.status = this.config.locale === "pt-BR" ? "Sem arquivo ativo" : "No active file"; this.changed(); return }
    const cursorLine = positionAt(this.document.text, this.document.selections[0]?.anchor ?? 0).line + 1
    try {
      const output = await this.git.stageHunk(path, cursorLine)
      this.status = output || (this.config.locale === "pt-BR" ? "Hunk staged" : "Staged hunk")
      await this.refreshGit()
    } catch (error) {
      this.report(error)
    } finally { this.changed() }
  }
  async gitUnstageHunk(): Promise<void> {
    const path = this.document.path
    if (!path) { this.status = this.config.locale === "pt-BR" ? "Sem arquivo ativo" : "No active file"; this.changed(); return }
    const cursorLine = positionAt(this.document.text, this.document.selections[0]?.anchor ?? 0).line + 1
    try {
      const output = await this.git.unstageHunk(path, cursorLine)
      this.status = output || (this.config.locale === "pt-BR" ? "Hunk unstaged" : "Unstaged hunk")
      await this.refreshGit()
    } catch (error) {
      this.report(error)
    } finally { this.changed() }
  }
  async showHunks(staged: boolean): Promise<void> {
    const path = this.document.path
    if (!path) { this.status = this.config.locale === "pt-BR" ? "Sem arquivo ativo" : "No active file"; this.changed(); return }
    let hunks: GitHunk[]
    try {
      hunks = await this.git.hunks(path, staged)
    } catch (error) { this.report(error); return }
    if (!hunks.length) {
      this.status = this.config.locale === "pt-BR" ? (staged ? "Sem hunks staged" : "Sem hunks para stage") : (staged ? "No staged hunks" : "No hunks to stage")
      this.changed(); return
    }
    const action = staged
      ? (this.config.locale === "pt-BR" ? "Remover da stage" : "Unstage")
      : (this.config.locale === "pt-BR" ? "Adicionar à stage" : "Stage")
    this.picker(`${action}: ${basename(path)}`, hunks.map((hunk, index) => ({
      label: `#${index + 1} ${hunkSummary(hunk)}`,
      run: () => { void this.guard(async () => {
        this.status = staged ? await this.git.unstageHunkAt(path, index) : await this.git.stageHunkAt(path, index)
        await this.refreshGit(); this.changed()
      }) },
    })))
  }
  async refreshGit(): Promise<void> {
    if (this.gitReading || this.closed) return
    this.gitReading = true
    try {
      this.gitStatus = await this.git.status()
      this.scmSelected = Math.min(this.scmSelected, Math.max(0, this.gitStatus.entries.length - 1))
      await this.refreshGitGutter()
      this.changed()
    }
    catch (error) { this.report(error) }
    finally { this.gitReading = false }
  }
  async refreshGitGutter(path = this.documents.find(doc => doc.id === this.activeId)?.path): Promise<void> {
    if (!path) return
    const gutter = await this.git.diffGutter(path).catch(() => new Map<number, GutterKind>())
    this.gitGutters.set(path, gutter)
    this.changed()
  }
  startTerminal(): void {
    if (!this.terminal) this.terminal = new TerminalSession(this.config.terminal.shell, this.workspace, this.width, Math.max(3, this.terminalHeight - 1), () => this.changed())
    this.bottom = "terminal"; this.focus = "terminal"; this.changed()
  }
  async ensureLsp(): Promise<LspClient | undefined> {
    const document = this.document
    if (!document.path || !this.config.lsp.enabled) return undefined
    const language = languageOf(document.path, this.config)
    let server = this.servers.get(language)
    if (!server?.running) {
      if (this.serverFailures.has(language)) {
        if (!lspRetryAllowed(this.lspRetryAt.get(language) ?? 0, this.lspFailures.get(language) ?? 1)) return undefined
        this.serverFailures.delete(language)
      }
      let starting = this.starting.get(language)
      if (!starting) {
        const custom = this.config.languages.find(entry => entry.id === language)?.lsp_command
        const command = custom?.length ? custom : this.config.lsp.servers[language]
        if (!command?.length) return undefined
        starting = (async () => {
          const candidate = new LspClient(command, this.workspace, this.config.lsp.timeout_ms, () => this.changed())
          try { await candidate.initialize(); if (this.closed) throw new Error("Application closed"); this.servers.set(language, candidate); return candidate }
          catch (error) { await candidate.close(); throw error }
        })()
        this.starting.set(language, starting)
      }
      try { server = await starting } catch (error) {
        this.lspFailures.set(language, (this.lspFailures.get(language) ?? 0) + 1)
        this.lspRetryAt.set(language, Date.now())
        this.serverFailures.set(language, String(error)); this.status = `LSP ${language}: ${String(error)}`; this.changed(); return undefined
      }
      finally { this.starting.delete(language) }
    }
    this.lspFailures.delete(language); this.lspRetryAt.delete(language)
    if (document.text.length > this.maxLspBytes()) {
      this.status = this.config.locale === "pt-BR"
        ? `LSP desabilitado para este arquivo (${Math.ceil(document.text.length / 1024)} KiB > 8192 KiB)`
        : `LSP skipped for this file (${Math.ceil(document.text.length / 1024)} KiB > 8192 KiB)`
      this.changed()
      return undefined
    }
    await server.sync(document.path, language, document.text, document.version)
    return server
  }
  private currentLanguage(): string | undefined {
    return this.document.path ? languageOf(this.document.path, this.config) : undefined
  }
  private lspSupported(server: LspClient, provider: keyof import("../services/lsp").ServerCapabilities, feature: string): boolean {
    if (server.supports(provider)) return true
    this.status = this.config.locale === "pt-BR" ? `LSP não suporta ${feature}` : `LSP does not support ${feature}`
    this.changed()
    return false
  }
  private maxLspBytes(): number { return 8 * 1024 * 1024 }
  async nextDiagnostic(delta = 1): Promise<void> {
    const items = [...this.diagnostics].sort((left, right) => {
      const leftPath = fileURLToPath(left.uri), rightPath = fileURLToPath(right.uri)
      if (leftPath !== rightPath) return leftPath.localeCompare(rightPath)
      const c = left.diagnostic.range.start.line - right.diagnostic.range.start.line
      return c !== 0 ? c : left.diagnostic.range.start.character - right.diagnostic.range.start.character
    })
    if (!items.length) { this.status = this.config.locale === "pt-BR" ? "Nenhum problema" : "No problems"; this.changed(); return }
    const activePath = this.document.path
    const pos = this.document.position
    const isForward = delta >= 0
    let index = items.findIndex(item => {
      const itemPath = fileURLToPath(item.uri)
      if (itemPath !== activePath) return itemPath > (activePath ?? "")
      const l = item.diagnostic.range.start.line, c = item.diagnostic.range.start.character
      return isForward ? (l > pos.line || (l === pos.line && c >= pos.character)) : (l < pos.line || (l === pos.line && c <= pos.character))
    })
    if (index < 0) index = isForward ? 0 : items.length - 1
    const item = items[index]!
    const targetPath = fileURLToPath(item.uri)
    this.recordJump()
    await this.openFile(targetPath)
    this.document.select(offsetAt(this.document.text, item.diagnostic.range.start.line, item.diagnostic.range.start.character))
    this.recordJump()
    const sev = item.diagnostic.severity === 1 ? (this.config.locale === "pt-BR" ? "erro" : "error") : (this.config.locale === "pt-BR" ? "aviso" : "warning")
    this.status = `${sev.toUpperCase()}: ${item.diagnostic.message}`
    this.changed()
  }

  lspHealth(): void {
    const servers = [...this.servers.entries()].map(([language, server]) => `${language}:${server.running ? "running" : "failed"}`)
    this.status = this.config.locale === "pt-BR"
      ? `Status LSP · ${this.servers.size} server(s) · ${servers.join(", ") || "none"} · failures: ${this.serverFailures.size}`
      : `LSP health · ${this.servers.size} server(s) · ${servers.join(", ") || "none"} · failures: ${this.serverFailures.size}`
    this.changed()
  }
  async restartLsp(): Promise<void> {
    const language = this.currentLanguage()
    for (const server of this.servers.values()) await server.close().catch(() => undefined)
    this.servers.clear(); this.serverFailures.clear()
    this.lspFailures.clear(); this.lspRetryAt.clear()
    this.starting.clear()
    if (language) await this.ensureLsp().catch(error => { this.status = String(error) })
    this.status = this.config.locale === "pt-BR" ? "Servidor LSP reiniciado" : "LSP restarted"
    this.afterEdit()
  }

  async format(): Promise<void> {
    const document = this.document, server = await this.ensureLsp()
    if (!server || !document.path) throw new Error("No running LSP for this document")
    if (!this.lspSupported(server, "documentFormattingProvider", "formatting")) return
    const version = document.version
    const edits = await server.formatting(document.path, this.config.editor.tab_size, this.config.editor.insert_spaces)
    if (document.version !== version) throw new Error("Document changed while formatting")
    document.applyEdits((edits ?? []).map(edit => ({ start: offsetAt(document.text, edit.range.start.line, edit.range.start.character), end: offsetAt(document.text, edit.range.end.line, edit.range.end.character), text: edit.newText })), "format")
    this.afterEdit()
  }
  async rename(newName?: string): Promise<void> {
    const document = this.document
    const currentWord = wordAtOffset(document.text, document.primary.head) || document.selectedText
    if (!newName) {
      const promptLabel = this.config.locale === "pt-BR"
        ? (currentWord ? `Renomear '${currentWord}' para:` : "Novo nome do símbolo:")
        : (currentWord ? `Rename '${currentWord}' to:` : "New symbol name:")
      this.prompt(promptLabel, currentWord, async value => {
        const trimmed = value.trim()
        if (!trimmed || trimmed === currentWord) return
        await this.rename(trimmed)
      })
      return
    }

    let lspSuccess = false
    if (document.path && this.config.lsp.enabled) {
      try {
        const server = await this.ensureLsp()
        if (server && this.lspSupported(server, "renameProvider", "rename")) {
          const workspaceEdit = await server.rename(document.path, document.position, newName)
          if (workspaceEdit) {
            if (workspaceEdit.changes) {
              for (const [uri, edits] of Object.entries(workspaceEdit.changes)) {
                const filePath = fileURLToPath(uri)
                const targetDoc = this.documents.find(doc => doc.path === filePath)
                if (targetDoc && edits.length > 0) {
                  const mappedEdits = edits.map(edit => ({
                    start: offsetAt(targetDoc.text, edit.range.start.line, edit.range.start.character),
                    end: offsetAt(targetDoc.text, edit.range.end.line, edit.range.end.character),
                    text: edit.newText,
                  }))
                  targetDoc.applyEdits(mappedEdits, "rename")
                }
              }
              lspSuccess = true
            }
          }
        }
      } catch {
        lspSuccess = false
      }
    }

    if (!lspSuccess) {
      const wordToReplace = currentWord || document.selectedText
      if (!wordToReplace) throw new Error("No symbol at cursor to rename")
      const matches = findMatches(document.text, wordToReplace, {
        caseSensitive: true,
        ignoreAccents: false,
        wholeWord: true,
        regex: false,
      })
      if (!matches.length) throw new Error(`Symbol '${wordToReplace}' not found`)
      const edits = replacementEdits(matches, newName, false)
      document.applyEdits(edits, "rename")
      this.status = this.config.locale === "pt-BR"
        ? `Renomeadas ${edits.length} ocorrência(s) de '${wordToReplace}' para '${newName}' (buffer)`
        : `Renamed ${edits.length} occurrence(s) of '${wordToReplace}' to '${newName}' (buffer)`
    } else {
      this.status = this.config.locale === "pt-BR"
        ? `Símbolo renomeado para '${newName}' via LSP`
        : `Symbol renamed to '${newName}' via LSP`
    }
    this.afterEdit()
  }
  scheduleCompletion(): void {
    clearTimeout(this.completionTimer)
    const document = this.document, version = document.version
    this.completionTimer = setTimeout(() => {
      if (!this.closed && !this.overlay && this.focus === "editor" && this.document === document && document.version === version) void this.guard(() => this.complete(true))
    }, 400)
  }
  acceptCompletion(index = this.completion?.selected ?? 0): void {
    const comp = this.completion
    if (!comp || !comp.items.length) return
    const item = comp.items[index]
    if (!item) return
    this.completion = undefined
    if (item.run) {
      item.run()
    } else {
      const document = this.document
      const word = comp.word
      const start = document.primary.head - word.length
      document.select(start, document.primary.head)
      const rawText = item.insertText ?? item.label
      if (rawText.includes("$0")) {
        const cursorOffset = rawText.indexOf("$0")
        const textToInsert = rawText.replace("$0", "")
        document.insert(textToInsert, false)
        document.select(start + cursorOffset)
      } else {
        document.insert(rawText, false)
      }
      this.afterEdit()
    }
    this.changed()
  }

  cancelCompletion(): void {
    if (!this.completion) return
    this.completion = undefined
    this.changed()
  }

  nextCompletion(step: number): void {
    const comp = this.completion
    if (!comp || !comp.items.length) return
    const count = comp.items.length
    comp.selected = (comp.selected + step % count + count) % count
    const item = comp.items[comp.selected]
    if (item?.doc) {
      this.status = `${item.label}: ${item.doc}`
    }
    this.changed()
  }

  async complete(automatic = false): Promise<void> {
    const document = this.document
    const word = document.text.slice(0, document.primary.head).match(/[\p{L}\p{N}_!]+$/u)?.[0] ?? ""
    if (automatic && word.length < this.config.editor.completion_min_chars) {
      this.cancelCompletion()
      return
    }
    const version = document.version
    const server = await this.ensureLsp().catch(() => undefined)
    const response = server && document.path && server.supports("completionProvider") ? await server.completion(document.path, document.position).catch(() => null) : null
    if (document.version !== version || this.closed || (automatic && this.overlay)) return

    const items: CompletionItem[] = []
    const seenLabels = new Set<string>()
    const resolvables: Array<{ item: CompletionItem; raw: import("../services/lsp").LspCompletionItem }> = []

    // 1. LSP completion items (se disponível)
    const suggestions = (Array.isArray(response) ? response : response?.items ?? [])
      .slice(0, this.config.lsp.enabled ? 100 : 100)
    for (const suggestion of suggestions) {
      if (!seenLabels.has(suggestion.label)) {
        seenLabels.add(suggestion.label)
        const item: CompletionItem = {
          label: suggestion.label,
          detail: suggestion.detail,
          kind: suggestion.kind ? String(suggestion.kind) : "lsp",
          run: () => {
            if (document.version !== version) throw new Error("Completion is stale")
            const edit = suggestion.textEdit
            if (edit) {
              const range = "range" in edit ? edit.range : edit.replace
              document.applyEdits([{ start: offsetAt(document.text, range.start.line, range.start.character), end: offsetAt(document.text, range.end.line, range.end.character), text: edit.newText }], "completion")
            } else {
              document.select(document.primary.head - word.length, document.primary.head)
              document.insert(suggestion.insertText ?? suggestion.label, false)
            }
            this.afterEdit()
          }
        }
        items.push(item)
        resolvables.push({ item, raw: suggestion })
      }
    }

    // 2. Extrator léxico de símbolos declarados (funções, classes, tipos)
    const localSymbols = extractDocumentSymbols(document.text)
    for (const sym of localSymbols) {
      const matches = !word || sym.name.toLowerCase().startsWith(word.toLowerCase()) || (word.length >= 3 && fuzzyScore(sym.name, word) > -Infinity)
      if (matches && !seenLabels.has(sym.name)) {
        seenLabels.add(sym.name)
        items.push({
          label: sym.name,
          detail: `${sym.signature} (${sym.kind})`,
          kind: sym.kind,
          doc: sym.doc,
          run: () => {
            document.select(document.primary.head - word.length, document.primary.head)
            document.insert(sym.name, false)
            this.afterEdit()
          }
        })
      }
    }

    // 3. Catálogo estático da linguagem (keywords, built-ins, standard library)
    const lang = languageOf(document.path, this.config)
    const catalog = getLanguageCatalog(lang)
    if (catalog) {
      for (const [key, item] of Object.entries(catalog)) {
        const matches = !word || key.toLowerCase().startsWith(word.toLowerCase()) || (word.length >= 3 && fuzzyScore(key, word) > -Infinity)
        if (matches && !seenLabels.has(key)) {
          seenLabels.add(key)
          items.push({
            label: key,
            detail: item.snippet ? `${item.detail} · snippet` : `${item.detail} · ${item.kind}`,
            kind: item.kind,
            doc: item.doc,
            insertText: item.snippet ?? key,
            run: () => {
              const start = document.primary.head - word.length
              document.select(start, document.primary.head)
              const insertText = item.snippet ?? key
              if (insertText.includes("$0")) {
                const cursorOffset = insertText.indexOf("$0")
                const textToInsert = insertText.replace("$0", "")
                document.insert(textToInsert, false)
                document.select(start + cursorOffset)
              } else {
                document.insert(insertText, false)
              }
              this.afterEdit()
            }
          })
        }
      }
    }

    // 3.5 Snippets definidos pelo usuário em .daisy/snippets
    const userSnippets = await loadUserSnippets(this.workspace, lang)
    for (const snippet of userSnippets) {
      const matches = !word || snippet.trigger.toLowerCase().startsWith(word.toLowerCase()) || (word.length >= 3 && fuzzyScore(snippet.trigger, word) > -Infinity)
      if (matches && !seenLabels.has(snippet.trigger)) {
        seenLabels.add(snippet.trigger)
        items.push({
          label: snippet.trigger,
          detail: `${snippet.description ?? "user snippet"} · snippet`,
          kind: "snippet",
          doc: snippet.body,
          insertText: snippet.body,
          run: () => {
            const start = document.primary.head - word.length
            document.select(start, document.primary.head)
            const insertText = snippet.body
            if (insertText.includes("$0")) {
              const cursorOffset = insertText.indexOf("$0")
              const textToInsert = insertText.replace("$0", "")
              document.insert(textToInsert, false)
              document.select(start + cursorOffset)
            } else {
              document.insert(insertText, false)
            }
            this.afterEdit()
          }
        })
      }
    }

    // 4. Palavras configuradas nas preferências da linguagem
    const languageWords = this.config.languages.find(entry => entry.id === lang)?.completion_words ?? []
    for (const langWord of languageWords) {
      const matches = !word || langWord.toLowerCase().startsWith(word.toLowerCase()) || (word.length >= 3 && fuzzyScore(langWord, word) > -Infinity)
      if (matches && !seenLabels.has(langWord)) {
        seenLabels.add(langWord)
        items.push({
          label: langWord,
          detail: "language",
          kind: "keyword",
          run: () => {
            document.select(document.primary.head - word.length, document.primary.head)
            document.insert(langWord, false)
            this.afterEdit()
          }
        })
      }
    }

    // 5. Palavras do buffer (Buffer-Word Completion de todos os buffers abertos)
    const allWords = extractBufferWords(this.documents)
    const bufferWords = suggestBufferWords(word, allWords, 100)
    for (const bWord of bufferWords) {
      if (!seenLabels.has(bWord)) {
        seenLabels.add(bWord)
        items.push({
          label: bWord,
          detail: "buffer",
          kind: "word",
          run: () => {
            document.select(document.primary.head - word.length, document.primary.head)
            document.insert(bWord, false)
            this.afterEdit()
          }
        })
      }
    }

    if (!items.length) {
      this.cancelCompletion()
      if (!automatic) {
        this.status = this.config.locale === "pt-BR" ? "Nenhuma sugestão" : "No completion suggestions"
        this.changed()
      }
      return
    }

    if (word) {
      const lowerWord = word.toLowerCase()
      items.sort((a, b) => {
        const aExact = a.label.toLowerCase() === lowerWord ? 1 : 0
        const bExact = b.label.toLowerCase() === lowerWord ? 1 : 0
        if (aExact !== bExact) return bExact - aExact

        const aPrefix = a.label.toLowerCase().startsWith(lowerWord) ? 1 : 0
        const bPrefix = b.label.toLowerCase().startsWith(lowerWord) ? 1 : 0
        if (aPrefix !== bPrefix) return bPrefix - aPrefix

        const kindRank = (k?: string) => k === "keyword" ? 3 : k === "function" || k === "type" ? 2 : k === "lsp" ? 2 : 1
        const aRank = kindRank(a.kind)
        const bRank = kindRank(b.kind)
        if (aRank !== bRank) return bRank - aRank

        return a.label.length - b.label.length
      })
    }

    this.completion = {
      word,
      offset: document.primary.head,
      selected: 0,
      items
    }
    // Manual completion enriches the top LSP items via completionItem/resolve.
    // Automatic completion skips this to stay latency-free while typing.
    if (!automatic && server) {
      const pending = resolvables.slice(0, 3)
      const enriched = await Promise.all(pending.map(entry => server.resolveCompletion(entry.raw).catch(() => null)))
      if (document.version !== version || this.closed) return
      for (const [index, resolved] of enriched.entries()) {
        const target = pending[index]?.item
        if (!target || !resolved) continue
        target.detail ??= resolved.detail
        const doc = typeof resolved.documentation === "string" ? resolved.documentation : resolved.documentation?.value
        if (doc && !target.doc) target.doc = doc
      }
    }
    const firstItem = items[0]
    if (firstItem?.doc) {
      this.status = `${firstItem.label}: ${firstItem.doc}`
    }
    this.changed()
  }
  async hover(): Promise<void> {
    const document = this.document
    const word = wordAtOffset(document.text, document.primary.head)
    if (!word) {
      this.status = this.config.locale === "pt-BR" ? "Nenhuma palavra sob o cursor" : "No word under cursor"
      this.changed()
      return
    }

    let hoverText: string | undefined
    let hoverTitle = `Hover: ${word}`

    // 1. Tenta LSP se disponível
    const server = await this.ensureLsp().catch(() => undefined)
    if (server && document.path && server.supports("hoverProvider")) {
      try {
        const response = await server.hover(document.path, document.position)
        const contents = response?.contents
        if (contents) {
          hoverText = typeof contents === "string"
            ? contents
            : Array.isArray(contents)
            ? contents.map(v => typeof v === "string" ? v : v.value).join("\n\n")
            : contents.value
          if (hoverText && hoverText.length > 8000) hoverText = `${hoverText.slice(0, 8000)}\n… [hover truncated]`
          hoverTitle = `LSP: ${word}`
        }
      } catch {
        // Fallback do LSP
      }
    }

    // 2. Tenta símbolos declarados no documento ou buffers
    if (!hoverText) {
      const symbol = findSymbol(word, this.documents)
      if (symbol) {
        hoverTitle = `${symbol.name} (${symbol.kind})`
        hoverText = `${symbol.signature}${symbol.doc ? `\n\n${symbol.doc}` : ""}`
      }
    }

    // 3. Tenta catálogo estático da linguagem
    if (!hoverText) {
      const lang = languageOf(document.path, this.config)
      const catalogItem = lookupCatalogItem(lang, word)
      if (catalogItem) {
        hoverTitle = `${word} (${catalogItem.kind})`
        hoverText = `${catalogItem.detail}\n\n${catalogItem.doc}`
      }
    }

    if (hoverText) {
      this.overlay = { kind: "read", title: hoverTitle, text: hoverText, scroll: 0 }
      this.changed()
    } else {
      this.status = this.config.locale === "pt-BR"
        ? `Sem informações de hover para '${word}'`
        : `No hover information for '${word}'`
      this.changed()
    }
  }
  async references(): Promise<void> {
    const document = this.document, server = await this.ensureLsp()
    if (!server || !document.path) throw new Error("No running LSP for this document")
    if (!this.lspSupported(server, "referencesProvider", "references")) return
    const response = await server.references(document.path, document.position, false)
    const locations = Array.isArray(response) ? response : response ? [response] : []
    if (!locations.length) { this.status = this.config.locale === "pt-BR" ? "Sem referências" : "No references"; this.changed(); return }
    const normalized = locations.map(location => {
      const anyLocation = location as any
      const uri = "uri" in location ? location.uri : anyLocation.targetUri
      const range = "range" in location ? location.range : (anyLocation.targetSelectionRange ?? anyLocation.range)
      return { uri, range, label: `${basename(uri)}:${range.start.line + 1}:${range.start.character + 1}` }
    })
    if (normalized.length === 1) {
      const location = normalized[0]!
      this.recordJump()
      await this.openFile(fileURLToPath(location.uri)); this.document.select(offsetAt(this.document.text, location.range.start.line, location.range.start.character)); this.recordJump(); this.changed(); return
    }
    this.picker(this.config.locale === "pt-BR" ? "Referências LSP" : "LSP references", normalized.map(location => ({
      label: location.label,
      run: () => { void this.openFile(fileURLToPath(location.uri)).then(() => { this.document.select(offsetAt(this.document.text, location.range.start.line, location.range.start.character)); this.changed() }) },
    })))
  }
  async signatureHelp(): Promise<void> {
    const document = this.document, server = await this.ensureLsp()
    if (!server || !document.path) throw new Error("No running LSP for this document")
    if (!this.lspSupported(server, "signatureHelpProvider", "signature help")) return
    const help = await server.signatureHelp(document.path, document.position)
    const signatures = help?.signatures ?? []
    if (!signatures.length) {
      this.status = this.config.locale === "pt-BR" ? "Sem ajuda de assinatura" : "No signature help"
      this.changed(); return
    }
    const active = help?.activeSignature ?? 0
    const lines = signatures.map((signature, index) => {
      const parameter = signature.parameters?.[help?.activeParameter ?? 0]
      const detail = typeof signature.documentation === "string" ? signature.documentation : signature.documentation?.value
      return `${index === active ? "▶ " : "  "}${signature.label}${parameter ? `  ← ${String(parameter.label)}` : ""}${detail ? `\n    ${detail}` : ""}`
    })
    let text = lines.join("\n")
    if (text.length > 8000) text = `${text.slice(0, 8000)}\n… [signature help truncated]`
    this.overlay = { kind: "read", title: "Signature help", text, scroll: 0 }
    this.changed()
  }
  async codeActions(): Promise<void> {
    const document = this.document, server = await this.ensureLsp()
    if (!server || !document.path) throw new Error("No running LSP for this document")
    if (!this.lspSupported(server, "codeActionProvider", "code actions")) return
    const { anchor, head } = document.primary
    const range = { start: positionAt(document.text, Math.min(anchor, head)), end: positionAt(document.text, Math.max(anchor, head)) }
    const uri = pathToFileURL(document.path).href
    const actions = await server.codeAction(document.path, range, server.diagnostics.get(uri) ?? [])
    if (!actions?.length) {
      this.status = this.config.locale === "pt-BR" ? "Sem ações de código" : "No code actions"
      this.changed(); return
    }
    this.picker(this.config.locale === "pt-BR" ? "Ações de código (LSP)" : "Code actions (LSP)", actions.map(action => ({
      label: action.title,
      detail: "kind" in action && typeof action.kind === "string" ? action.kind : undefined,
      run: () => { void this.applyCodeAction(action) },
    })))
  }
  private async applyCodeAction(action: import("../services/lsp").CodeAction | import("../services/lsp").LspCommand): Promise<void> {
    const document = this.document
    if (!document.path) return
    if (!("edit" in action) || !action.edit) { this.status = `Code action '${action.title}' has no local edit`; this.changed(); return }
    const uri = pathToFileURL(document.path).href
    const edits = Object.entries(action.edit?.changes ?? {}).filter(([changeUri]) => changeUri === uri).flatMap(([, changeEdits]) => changeEdits)
    if (!edits.length) { this.status = `Code action has no edits for this file`; this.changed(); return }
    const version = document.version
    document.applyEdits(edits.map(edit => ({ start: offsetAt(document.text, edit.range.start.line, edit.range.start.character), end: offsetAt(document.text, edit.range.end.line, edit.range.end.character), text: edit.newText })), "code action")
    if (document.version === version) { this.status = "Code action produced no changes"; this.changed(); return }
    this.afterEdit()
  }
  async definition(): Promise<void> {
    const document = this.document, server = await this.ensureLsp()
    if (!server || !document.path) throw new Error("No running LSP for this document")
    if (!this.lspSupported(server, "definitionProvider", "go to definition")) return
    const response = await server.definition(document.path, document.position)
    const locations = Array.isArray(response) ? response : response ? [response] : []
    const location = locations[0]
    if (!location) { this.status = "No definition"; return }
    this.recordJump()
    const uri = "uri" in location ? location.uri : location.targetUri
    const range = "range" in location ? location.range : location.targetSelectionRange
    await this.openFile(fileURLToPath(uri)); this.document.select(offsetAt(this.document.text, range.start.line, range.start.character)); this.recordJump(); this.changed()
  }
  recordJump(): void {
    if (!this.document.path) return
    this.jumpStack.push({ path: this.document.path, offset: this.document.primary.head })
  }
  private async gotoJump(entry: { path: string; offset: number }): Promise<void> {
    await this.openFile(entry.path, entry.offset)
  }
  async jumpBack(): Promise<void> {
    this.recordJump()
    const jump = this.jumpStack.back()
    if (!jump) { this.status = "jump list vazio"; this.changed(); return }
    await this.gotoJump(jump)
    this.status = "jump back"; this.changed()
  }
  showUndoHistory(): void {
    const labels = this.document.historyLabels()
    if (!labels.length) {
      this.status = this.config.locale === "pt-BR" ? "Sem histórico de undo" : "No undo history"
      this.changed(); return
    }
    const window = labels.map((label, index) => ({ label, index })).slice(-200)
    this.picker(this.config.locale === "pt-BR" ? "Histórico de undo" : "Undo history", window.map(({ label, index }) => ({
      label: `↩ ${labels.length - index} · ${label || "edit"}`,
      run: () => { if (this.document.restoreHistory(index)) this.afterEdit() },
    })))
  }
  async jumpForward(): Promise<void> {
    const jump = this.jumpStack.forward()
    if (!jump) { this.status = "sem jump forward"; this.changed(); return }
    await this.gotoJump(jump)
    this.status = "jump forward"; this.changed()
  }
  get diagnostics() {
    return [...this.servers.values()].flatMap(server => [...server.diagnostics.entries()].flatMap(([uri, entries]) => entries.map(diagnostic => ({ uri, diagnostic }))))
  }
  async pollDisk(): Promise<void> {
    if (this.pollingDisk || this.closed) return
    this.pollingDisk = true
    try {
      for (const document of this.documents) {
        if (!document.path) continue
        const path = document.path
        const modified = await stat(path).then(info => info.mtimeMs).catch(() => undefined)
        if (modified === undefined || modified === this.diskVersions.get(path)) continue
        this.diskVersions.set(path, modified)
        if (document.dirty) {
          // Como o oráculo: sujo + mudou fora → oferecer reload em vez de
          // decidir pelo leitor. Sem sobreposição aberta, um confirm; com uma
          // (o leitor está digitando um filtro, p.ex.), só o status — trocar o
          // que ele está fazendo pelo nosso aviso perderia o texto digitado.
          // Uma oferta por mudança: repetir a cada tique seria um nag.
          if (!this.overlay && this.diskConfirmPath !== path) {
            this.diskConfirmPath = path
            this.overlay = { kind: "confirm", title: `Changed on disk: ${document.title} — Enter reloads and discards my edits · Esc keeps them`, accept: async () => {
              this.diskConfirmPath = undefined
              const text = await readText(path)
              document.applyEdits([{ start: 0, end: document.text.length, text }], "disk reload")
              document.savedText = text
              this.diskVersions.set(path, (await stat(path)).mtimeMs)
              this.status = `Reloaded ${document.title} from disk`; this.afterEdit()
            } }
          } else this.status = `Changed on disk: ${document.title}; unsaved edits preserved`
          this.changed(); continue
        }
        const text = await readText(path)
        if (text === document.text || document.dirty) continue
        const head = document.primary.head
        document.applyEdits([{ start: 0, end: document.text.length, text }], "disk reload")
        document.savedText = text; document.select(head)
        this.status = `Reloaded ${document.title} from disk`; this.afterEdit()
      }
    } finally { this.pollingDisk = false }
  }
  pluginEntries(): PickerEntry[] {
    return this.plugins.flatMap(plugin => plugin.commands.map(command => ({ label: `${plugin.name}: ${command.label}`, detail: command.id, run: () => this.offerPlugin(plugin, command, this.document.path ?? "") })))
  }
  private offerHooks(hook: "on_open" | "on_save", path: string): void {
    for (const plugin of this.plugins) {
      const command = plugin.hooks[hook]
      if (command && this.trustedPlugins.has(plugin.name)) void this.guard(async () => { this.status = await runPlugin(command, { file: path, workspace: this.workspace }); this.changed() })
    }
  }
  private offerPlugin(plugin: Plugin, command: PluginCommand, path: string): void {
    this.overlay = { kind: "confirm", title: `Run ${plugin.name}: ${command.label}? Plugin receives filesystem access. Enter confirms for this session`, accept: async () => {
      this.trustedPlugins.add(plugin.name)
      this.status = await runPlugin(command, { file: path, workspace: this.workspace }); this.changed()
    } }
    this.changed()
  }
  async persist(): Promise<void> {
    const files = this.documents.flatMap(document => document.path ? [document.path] : [])
    const secondaryFile = this.panes.length > 1 ? this.documents.find(d => d.id === this.panes[1]?.documentId)?.path : undefined
    const split: SplitSession | undefined = this.panes.length > 1 ? {
      orientation: this.orientation,
      secondary_file: secondaryFile,
      ratio_percent: this.splitRatio,
    } : undefined
    await saveSession({
      workspace: this.workspace,
      files,
      active_index: Math.max(0, files.indexOf(this.document.path ?? "")),
      scroll_y: this.panes[0]?.scroll ?? 0,
      show_tree: this.showTree,
      show_scm: this.showScm,
      tree_width: this.config.tree.width,
      split,
    })
  }
  debugBreakpointLines(): number[] { return (this.debugBreakpoints.get(this.document.path ?? "") ?? []).map(entry => entry.line) }

  toggleBreakpoint(): void {
    const path = this.document.path
    if (!path) { this.status = this.config.locale === "pt-BR" ? "Salve o arquivo antes de usar breakpoint" : "Save the file before using breakpoints"; this.changed(); return }
    const line = this.document.position.line + 1
    const entries = this.debugBreakpoints.get(path) ?? []
    const index = entries.findIndex(entry => entry.line === line)
    if (index >= 0) entries.splice(index, 1); else entries.push({ line })
    this.debugBreakpoints.set(path, entries.sort((a, b) => a.line - b.line))
    this.status = this.config.locale === "pt-BR" ? `Breakpoint ${index >= 0 ? "removido" : "adicionado"} da linha ${line}` : `Breakpoint ${index >= 0 ? "removed" : "added"} at line ${line}`
    this.changed()
  }
  breakpointCondition(): void {
    const path = this.document.path
    if (!path) { this.status = this.config.locale === "pt-BR" ? "Salve o arquivo antes de usar breakpoint" : "Save the file before using breakpoints"; this.changed(); return }
    const line = this.document.position.line + 1
    const current = (this.debugBreakpoints.get(path) ?? []).find(entry => entry.line === line)?.condition ?? ""
    const title = this.config.locale === "pt-BR" ? `Condição do breakpoint na linha ${line}` : `Breakpoint condition at line ${line}`
    this.prompt(title, current, async value => {
      const entries = this.debugBreakpoints.get(path) ?? []
      const existing = entries.find(entry => entry.line === line)
      const condition = value.trim() || undefined
      if (existing) existing.condition = condition
      else entries.push({ line, condition })
      this.debugBreakpoints.set(path, entries.sort((a, b) => a.line - b.line))
      this.status = this.config.locale === "pt-BR" ? `Breakpoint da linha ${line} atualizado` : `Breakpoint at line ${line} updated`
      this.changed()
    })
  }

  private debugAdapterCommand(language: string): string[] | undefined {
    if (!this.config.debug.enabled) return undefined
    return this.config.debug.adapters[language] ?? undefined
  }

  async startDebug(name?: string): Promise<void> {
    const document = this.document
    const path = document.path
    const requestedLaunches = Object.keys(this.config.debug.launches)
    if (!name && requestedLaunches.length > 1) {
      this.picker(this.config.locale === "pt-BR" ? "Config de debug" : "Debug launch configs", requestedLaunches.map(launch => ({
        label: launch,
        detail: this.config.debug.launches[launch]?.language ?? "launch",
        run: () => { void this.startDebug(launch) },
      })))
      return
    }
    const launch = name ? this.config.debug.launches[name] : requestedLaunches.length === 1 ? this.config.debug.launches[requestedLaunches[0]!] : undefined
    if (launch && !path && !launch.program) throw new Error("Debug requires a saved file")
    const language = launch?.language ?? languageOf(path ?? launch?.program, this.config)
    const command = launch?.adapter?.length ? launch.adapter : this.debugAdapterCommand(language)
    if (!command?.length) throw new Error(`No debug adapter configured for ${language}`)
    if (this.debug && (this.debugStatus === "running" || this.debugStatus === "stopped")) {
      this.status = this.config.locale === "pt-BR" ? "Debug já em execução" : "Debug already running"
      this.changed(); return
    }
    if (this.debug) { await this.debug?.disconnect().catch(() => undefined); this.debug = undefined }
    const program = launch?.program ?? path
    if (!program) throw new Error("Debug requires a program")
    const args = launch?.args ?? []
    const cwd = launch?.cwd ? resolve(this.workspace, launch.cwd) : dirname(program)
    const debugBreakpointsPath = path ?? program
    const adapter = new DebugAdapter(command, this.workspace, (status, message) => { this.debugStatus = status; this.status = message ? `Debug ${status}: ${message}` : `Debug ${status}`; this.changed() })
    try {
      await adapter.initialize()
      const breakpoints = this.debugBreakpoints.get(debugBreakpointsPath) ?? []
      await adapter.setBreakpoints(debugBreakpointsPath, breakpoints)
      await adapter.launch({ cwd, program, args })
      this.debug = adapter
      this.debugStatus = "running"
      this.status = this.config.locale === "pt-BR" ? "Debug iniciado" : "Debug started"
    } catch (error) {
      await adapter.disconnect().catch(() => undefined)
      this.debug = undefined
      this.debugStatus = "failed"
      throw error
    }
    this.changed()
  }

  async showDebugThreads(): Promise<void> {
    if (!this.debug) {
      this.status = this.config.locale === "pt-BR" ? "Sem debug ativo" : "No debug session"
      this.changed(); return
    }
    try {
      const threads = await this.debug.threads()
      const thread = threads[0]?.id ?? 1
      const frames = await this.debug.stackTrace(thread)
      const lines = [
        "Debug session:",
        `status: ${this.debugStatus}`,
        `thread: ${threads[0]?.name ?? thread}`,
        "stack frames:",
        ...frames.map((frame, index) => `  #${index + 1} ${frame.name} @ ${frame.source?.name ?? frame.source?.path ?? "?"}:${frame.line ?? 0}`),
      ]
      this.overlay = { kind: "read", title: "Debug threads", text: lines.join("\n"), scroll: 0 }
    } catch (error) {
      this.status = String(error)
    }
    this.changed()
  }

  async showDebugVariables(): Promise<void> {
    if (!this.debug) {
      this.status = this.config.locale === "pt-BR" ? "Sem debug ativo" : "No debug session"
      this.changed(); return
    }
    try {
      const threadId = (await this.debug.threads())[0]?.id ?? 1
      const frames = await this.debug.stackTrace(threadId)
      const frameId = frames[0]?.id
      let variables: Array<{ name: string; value: string; type?: string }> = []
      if (frameId !== undefined) {
        const scopes = await this.debug.request<{ scopes?: Array<{ variablesReference: number }> }>("scopes", { frameId })
        const variablesReference = scopes?.scopes?.[0]?.variablesReference
        if (variablesReference) {
          const response = await this.debug.request<{ variables?: Array<{ name: string; value: string; type?: string }> }>("variables", { variablesReference })
          variables = response?.variables ?? []
        }
      }
      const lines = [
        "Debug variables:",
        ...variables.map(v => `  ${v.name}: ${v.value}${v.type ? ` (${v.type})` : ""}`),
      ]
      this.overlay = { kind: "read", title: "Debug variables", text: lines.join("\n"), scroll: 0 }
    } catch (error) {
      this.status = String(error)
    }
    this.changed()
  }

  async stopDebug(): Promise<void> {
    await this.debug?.disconnect()
    this.debug = undefined
    this.debugStatus = "idle"
    this.status = this.config.locale === "pt-BR" ? "Debug parado" : "Debug stopped"
    this.changed()
  }

  async debugContinue(): Promise<void> {
    if (!this.debug) return
    const serverThreads = await (this.debug.request("threads").catch(() => ({ threads: [] })) as Promise<{ threads?: Array<{ id: number }> }>)
    const thread = serverThreads.threads?.[0]?.id ?? 1
    await this.debug.continue(thread)
  }

  async debugStepOver(): Promise<void> {
    if (!this.debug) return
    const serverThreads = await (this.debug.request("threads").catch(() => ({ threads: [] })) as Promise<{ threads?: Array<{ id: number }> }>)
    const thread = serverThreads.threads?.[0]?.id ?? 1
    await this.debug.request("next", { threadId: thread })
    this.debugStatus = "stopped"
    this.changed()
  }

  async lspSymbols(): Promise<void> {
    const document = this.document
    const path = document.path
    if (!path) { this.status = this.config.locale === "pt-BR" ? "Salve o arquivo antes de símbolos" : "Save file before symbols"; this.changed(); return }
    const server = await this.ensureLsp().catch(() => undefined)
    if (!server) { this.status = this.config.locale === "pt-BR" ? "Sem LSP ativo" : "No LSP attached"; this.changed(); return }
    if (!this.lspSupported(server, "documentSymbolProvider", "document symbols")) return
    try {
      const symbols = await server.documentSymbols(path)
      const entries: PickerEntry[] = symbols ? flattenLspSymbols(this, symbols, path) : []
      if (!entries.length) {
        this.status = this.config.locale === "pt-BR" ? "No LSP symbols (local?)" : "No LSP symbols"
        this.changed(); return
      }
      this.picker(this.config.locale === "pt-BR" ? "Símbodo LSP" : "Symbols (LSP)", entries)
    } catch (error) { this.status = String(error); this.changed() }
  }

  async close(): Promise<void> {
    this.closed = true; clearTimeout(this.lspTimer); clearTimeout(this.completionTimer); clearTimeout(this.fuzzyTimer)
    try { this.fuzzyWorker?.terminate() } catch {}
    this.fuzzyWorker = undefined
    await this.terminal?.close()
    await this.debug?.disconnect().catch(() => undefined)
    this.debug = undefined
    await Promise.all([...this.servers.values()].map(server => server.close?.()))
    await Promise.allSettled(this.starting.values())
    this.listeners.clear()
  }
}
