import { basename, dirname, join, relative } from "node:path"
import { rename, unlink, rmdir, stat } from "node:fs/promises"
import { actionIds, type ActionId } from "../core/keymap"
import { offsetAt, lineRanges, type Direction } from "../core/document"
import { findMatches } from "../core/search"
import { languageOf, commentDelimiters } from "../services/languages"
import { readText } from "../services/files"
import { readToml, record, userConfigDir } from "../services/config"
import type { Application, PickerEntry } from "./application"
import { savePreference } from "../services/preferences"
import { menus } from "./menus"

export type Command = { label: string; run: (app: Application) => void | Promise<void> }
const command = (label: string, run: Command["run"]): Command => ({ label, run })
const movement = (direction: Direction, extend = false) => command(`Move ${direction}${extend ? " (select)" : ""}`, app => app.document.move(direction, extend))
const toggle = (label: string, run: (app: Application) => void) => command(label, run)

export const commands: Record<ActionId, Command> = {
  quit: command("Quit", app => {
    if (app.documents.some(document => document.dirty)) app.overlay = { kind: "confirm", title: app.t("dialogs", "quit_dirty"), accept: () => { app.quit = true; app.changed() } }
    else app.quit = true
  }),
  save: command("Save", app => app.save()),
  save_as: command("Save as…", app => app.openPathBrowser("save_as", app.document.path ? dirname(app.document.path) : app.workspace, app.document.path ? basename(app.document.path) : "untitled.txt")),
  save_all: command("Save all", async app => { for (const document of app.documents) if (document.dirty) { if (!document.path) throw new Error("Save untitled buffers with Save as first"); await app.save(document) } }),
  undo: command("Undo", app => app.document.undo()), redo: command("Redo", app => app.document.redo()),
  insert_newline: command("Insert newline", app => app.document.insert("\n")),
  insert_tab: command("Indent", app => app.document.insert((app.document.indent?.insertSpaces ?? app.config.editor.insert_spaces) ? " ".repeat(app.document.indent?.tabSize ?? app.config.editor.tab_size) : "\t")),
  backspace: command("Backspace", app => app.backspace()), delete: command("Delete", app => app.document.remove(false)),
  move_left: movement("left"), move_right: movement("right"), move_up: movement("up"), move_down: movement("down"),
  move_line_start: movement("line_start"), move_line_end: movement("line_end"), move_doc_start: movement("doc_start"), move_doc_end: movement("doc_end"),
  move_left_extend: movement("left", true), move_right_extend: movement("right", true), move_up_extend: movement("up", true), move_down_extend: movement("down", true),
  move_line_start_extend: movement("line_start", true), move_line_end_extend: movement("line_end", true), move_doc_start_extend: movement("doc_start", true), move_doc_end_extend: movement("doc_end", true),
  select_all: command("Select all", app => app.document.select(0, app.document.text.length)),
  page_up: command("Page up", app => app.document.move("up", false, Math.max(1, app.editorHeight - 1))),
  page_down: command("Page down", app => app.document.move("down", false, Math.max(1, app.editorHeight - 1))),
  toggle_tree: toggle("Toggle file tree", app => { app.showTree = !app.showTree; if (!app.showTree && app.focus === "tree") app.focus = "editor" }),
  toggle_terminal: toggle("Toggle terminal", app => { if (app.bottom === "terminal") { app.bottom = undefined; app.focus = "editor" } else app.startTerminal() }),
  terminal_grow: toggle("Grow terminal", app => { app.terminalHeight = Math.min(app.height - 5, app.terminalHeight + 2) }),
  terminal_shrink: toggle("Shrink terminal", app => { app.terminalHeight = Math.max(3, app.terminalHeight - 2) }),
  tree_grow: toggle("Grow tree", app => { app.config.tree.width = Math.min(app.width - 20, app.config.tree.width + 2) }),
  tree_shrink: toggle("Shrink tree", app => { app.config.tree.width = Math.max(8, app.config.tree.width - 2) }),
  focus_tree: toggle("Focus tree", app => { app.showTree = true; app.focus = "tree" }),
  focus_editor: toggle("Focus editor", app => { app.focus = "editor" }),
  focus_terminal: toggle("Focus terminal", app => app.startTerminal()),
  next_tab: command("Next tab", app => app.cycleTab(1)), prev_tab: command("Previous tab", app => app.cycleTab(-1)),
  close_tab: command("Close tab", app => app.closeTab()), new_tab: command("New tab", app => app.newDocument()),
  command_palette: command("Command palette", app => app.picker("Commands", buildPaletteEntries(app))),
  open_file_fuzzy: command("Open file…", app => app.finder()),
  goto_line: command("Go to line…", app => {
    const totalLines = lineRanges(app.document.text).length
    const currentLine = app.document.position.line + 1
    app.prompt(
      app.config.locale === "pt-BR" ? `Ir para a linha (1-${totalLines}):` : `Go to line (1-${totalLines}):`,
      String(currentLine),
      input => {
        const trimmed = input.trim()
        if (!trimmed) return
        const parts = trimmed.split(":")
        const targetLine = parseInt(parts[0]!, 10)
        const targetCol = parts[1] ? parseInt(parts[1], 10) : 1
        if (Number.isNaN(targetLine)) return
        const line = Math.max(0, Math.min(totalLines - 1, targetLine - 1))
        const col = Math.max(0, (Number.isNaN(targetCol) ? 1 : targetCol) - 1)
        const offset = offsetAt(app.document.text, line, col)
        app.recordJump()
        app.document.select(offset)
        app.centerCursor()
        app.focus = "editor"
      }
    )
  }),
  tree_new_file: command("New file", app => { app.showTree = true; app.focus = "tree"; app.prompt(app.t("prompts", "new_file"), "", async name => { await app.openFile(await app.tree.create(name, false)); app.invalidateFiles() }) }),
  tree_new_dir: command("New directory", app => { app.showTree = true; app.focus = "tree"; app.prompt(app.t("prompts", "new_dir"), "", async name => { await app.tree.create(name, true); app.invalidateFiles() }) }),
  tree_refresh: command("Refresh files and Git", async app => { await app.tree.refresh(); app.invalidateFiles(); await app.refreshGit() }),
  open_folder: command("Open workspace…", app => app.openPathBrowser("folder", app.workspace)),
  focus_toggle_tree_editor: toggle("Toggle tree/editor focus", app => { app.focus = app.focus === "tree" ? "editor" : "tree"; app.showTree = true }),
  toggle_soft_wrap: toggle("Toggle soft wrap", app => { app.config.soft_wrap = !app.config.soft_wrap; app.status = app.t("messages", app.config.soft_wrap ? "soft_wrap_on" : "soft_wrap_off") }),
  toggle_comment: command("Toggle comment", app => {
    const delimiters = commentDelimiters(languageOf(app.document.path, app.config), app.config)
    if (delimiters) app.document.toggleComment(delimiters)
  }),
  toggle_md_preview: toggle("Markdown preview", app => {
    // Como o oráculo: preview é da família Markdown; fora dela, dizer — não abrir.
    if (!app.preview && !["markdown", "mdx"].includes(languageOf(app.document.path, app.config))) { app.status = app.t("messages", "preview_only_md"); return }
    app.preview = !app.preview
    app.previewScroll = 0
    app.status = app.t("messages", app.preview ? "preview_on" : "preview_off")
  }),
  help: command("All keybindings", app => app.picker("Keybindings", Object.entries(app.config.keys).map(([chord, id]) => ({ label: `${chord.padEnd(24)} ${app.label(id)}`, run: () => app.dispatch(id) })))),
  find: command("Find", app => app.openSearch(false)), replace: command("Replace", app => app.openSearch(false, true)),
  find_next: command("Next match", app => app.findNext(1)), find_prev: command("Previous match", app => app.findNext(-1)),
  project_find: command("Find in project", app => app.openSearch(true)), project_replace: command("Replace in project", app => app.openSearch(true, true)),
  copy: command("Copy", app => copy(app)), cut: command("Cut", app => { copy(app); app.document.remove(false) }),
  paste: command("Paste", async app => {
    const document = app.document
    const text = await app.readClipboard?.() ?? app.clipboard
    if (text) document.insert(text, false)
  }),
  reload_file: command("Reload file", async app => {
    const document = app.document
    if (!document.path) throw new Error("Untitled document")
    const reload = async () => {
      const text = await readText(document.path ?? "")
      const offset = document.primary.head
      document.applyEdits([{ start: 0, end: document.text.length, text }], "reload"); document.savedText = text; document.select(offset); app.afterEdit()
    }
    if (document.dirty) app.overlay = { kind: "confirm", title: app.t("dialogs", "reload_dirty"), accept: reload }
    else await reload()
  }),
  lsp_complete: command("Completion (LSP/local)", app => app.complete()), lsp_hover: command("LSP hover", app => app.hover()),
  lsp_goto_definition: command("Go to definition", app => app.definition()), lsp_format: command("Format document", app => app.format()),
  lsp_rename: command("Rename symbol…", async app => { await app.rename() }),
  lsp_symbols: command("Document symbols (LSP)", async app => { await app.lspSymbols() }),
  lsp_references: command("Find references (LSP)", async app => { await app.references() }),
  lsp_signature_help: command("Signature help (LSP)", async app => { await app.signatureHelp() }),
  lsp_code_action: command("Code actions (LSP)", async app => { await app.codeActions() }),
  lsp_next_problem: command("Next problem", app => { void app.nextDiagnostic(1) }),
  lsp_prev_problem: command("Previous problem", app => { void app.nextDiagnostic(-1) }),
  lsp_health: command("LSP health", app => app.lspHealth()),
  lsp_restart: command("Restart LSP", async app => { await app.restartLsp() }),
  debug_toggle_breakpoint: command("Toggle breakpoint", app => app.toggleBreakpoint()),
  debug_launch: command("Start debug", app => app.startDebug()),
  debug_stop: command("Stop debug", app => { void app.stopDebug() }),
  debug_continue: command("Continue debug", app => { void app.debugContinue() }),
  debug_step_over: command("Debug step over", app => { void app.debugStepOver() }),
  debug_show_threads: command("Show debug threads", app => { void app.showDebugThreads() }),
  debug_variables: command("Show debug variables", app => { void app.showDebugVariables() }),
  debug_breakpoint_condition: command("Breakpoint condition…", app => app.breakpointCondition()),
  toggle_diagnostics: toggle("Problems", app => { app.bottom = app.bottom === "problems" ? undefined : "problems"; app.focus = "editor" }),
  split_vertical: command("Split vertically", app => app.split("vertical")), split_horizontal: command("Split horizontally", app => app.split("horizontal")),
  focus_next_pane: command("Next editor pane", app => app.focusPane(app.focusedPane + 1)), close_pane: command("Close editor pane", app => app.closePane()),
  resize_pane_grow: toggle("Grow focused pane", app => { app.splitRatio = Math.min(85, app.splitRatio + 5) }),
  resize_pane_shrink: toggle("Shrink focused pane", app => { app.splitRatio = Math.max(15, app.splitRatio - 5) }),
  add_cursor_above: command("Add cursor above", app => app.document.addCursor(-1)), add_cursor_below: command("Add cursor below", app => app.document.addCursor(1)),
  clear_extra_cursors: command("Clear extra cursors", app => app.document.clearExtraCursors()),
  toggle_scm: toggle("Source control", app => { app.showScm = !app.showScm; app.focus = app.showScm ? "scm" : "editor"; void app.refreshGit() }),
  focus_scm: toggle("Focus source control", app => { app.showScm = true; app.focus = "scm" }),
  git_pull: command("Git pull", app => confirmGit(app, "pull")), git_push: command("Git push", app => confirmGit(app, "push")),
  git_stash: command("Git stash", async app => { await app.gitStash(); app.status = "Stashed" }),
  git_stash_pop: command("Git stash pop", async app => { await app.gitStashPop(); app.status = "Stash applied" }),
  git_stage_hunk: command("Git stage current hunk", async app => { await app.gitStageHunk() }),
  git_unstage_hunk: command("Git unstage current hunk", async app => { await app.gitUnstageHunk() }),
  git_blame: command("Git blame line", async app => {
    if (!app.document.path) throw new Error("Untitled document")
    const line = app.document.position.line + 1
    const info = await app.git.blame(app.document.path, line)
    app.status = info
    app.overlay = { kind: "read", title: `Git Blame: ${basename(app.document.path)}:${line}`, text: info, scroll: 0 }
  }),
  buffer_picker: command("Open buffers", app => app.picker("Buffers", app.documents.map(document => ({ label: `${document.dirty ? "● " : ""}${document.title}`, detail: document.path, run: () => { app.recordJump(); app.activate(document.id) } })))),
  jump_back: command("Jump back", app => app.jumpBack()), jump_forward: command("Jump forward", app => app.jumpForward()),
  which_key: command("Essential shortcuts", app => {
    const isPt = app.config.locale === "pt-BR"
    const rows: [string, string][] = [
      ["F1", isPt ? "todos os atalhos (help)" : "all keybindings (help)"],
      ["Ctrl+Shift+P", isPt ? "command palette" : "command palette"],
      ["Ctrl+P", isPt ? "abrir arquivo (fuzzy finder)" : "open file (fuzzy finder)"],
      ["Ctrl+S", isPt ? "salvar arquivo ativo" : "save active buffer"],
      ["Ctrl+F", isPt ? "localizar no buffer" : "find in buffer"],
      ["Ctrl+Shift+F", isPt ? "localizar no projeto" : "find in project"],
      ["Ctrl+`", isPt ? "alternar terminal embutido" : "toggle embedded terminal"],
      ["Ctrl+Shift+G", isPt ? "painel Git / SCM" : "Git / SCM panel"],
      ["Ctrl+Shift+O", isPt ? "seletor de buffers abertos" : "open buffers picker"],
      ["Ctrl+B", isPt ? "alternar / focar árvore de arquivos" : "toggle / focus file tree"],
      ["Ctrl+E", isPt ? "focar editor principal" : "focus main editor"],
      ["Alt+P", isPt ? "alternar preview Markdown" : "toggle Markdown preview"],
      ["Alt+F/E/V/G/I/H", isPt ? "menus da menubar" : "menubar dropdowns"],
      ["Alt+/", isPt ? "este which-key" : "this which-key"],
    ]
    const text = rows.map(([chord, desc]) => `  ${chord.padEnd(20)} ${desc}`).join("\n")
    app.overlay = { kind: "read", title: isPt ? "Atalhos Essenciais (Which-Key)" : "Essential Shortcuts (Which-Key)", text, scroll: 0 }
  }),
  welcome: command("Welcome / shortcuts", app => {
    const isPt = app.config.locale === "pt-BR"
    const text = isPt ? [
      "Bem-vindo ao Daisy Code Editor — Terminal IDE",
      "",
      "  F1                todos os atalhos",
      "  Ctrl+Shift+P      command palette",
      "  Ctrl+P            abrir arquivo (fuzzy)",
      "  Ctrl+S            salvar",
      "  Ctrl+`            terminal interativo",
      "  Ctrl+Shift+G      painel Git (SCM)",
      "  Alt+P             preview Markdown",
      "  Alt+F             menu Arquivo",
      "  Alt+/             which-key",
      "",
      "  Pressione Esc para fechar este diálogo e começar a editar."
    ].join("\n") : [
      "Welcome to Daisy Code Editor — Terminal IDE",
      "",
      "  F1                all keybindings",
      "  Ctrl+Shift+P      command palette",
      "  Ctrl+P            open file (fuzzy)",
      "  Ctrl+S            save buffer",
      "  Ctrl+`            interactive terminal",
      "  Ctrl+Shift+G      Git (SCM) panel",
      "  Alt+P             Markdown preview",
      "  Alt+F             File menu",
      "  Alt+/             which-key",
      "",
      "  Press Esc to close this dialog and start editing."
    ].join("\n")
    app.overlay = { kind: "read", title: isPt ? "Boas-vindas" : "Welcome", text, scroll: 0 }
  }),
  show_diff: command("Diff active file", async app => { app.overlay = { kind: "read", title: "Git diff", text: await app.git.diff(app.document.path), scroll: 0 } }),
  surround: command("Surround selection", app => app.prompt(app.t("prompts", "surround"), "()", value => {
    const characters = [...value]
    if (characters.length !== 2) throw new Error("Enter exactly two delimiter characters")
    app.document.applyEdits(app.document.selections.map(selection => ({ start: Math.min(selection.anchor, selection.head), end: Math.max(selection.anchor, selection.head), text: characters[0] + app.document.text.slice(Math.min(selection.anchor, selection.head), Math.max(selection.anchor, selection.head)) + characters[1] })), "surround")
  })),
  multi_picker: command("Select all occurrences / multicursor", app => app.prompt(app.t("prompts", "select_occurrences"), app.document.selectedText, query => {
    const matches = findMatches(app.document.text, query, app.searchOptions)
    for (const [index, match] of matches.entries()) app.document.select(match.start, match.end, index !== 0)
  })),
  undo_tree: command("Undo history", app => app.picker("Undo history (linear)", app.document.history.toReversed().map((entry, index) => ({ label: `${index + 1}. ${entry.label}`, run: () => { for (let count = 0; count <= index; count++) app.document.undo() } })))),
  toggle_mouse: toggle("Toggle mouse", app => { app.config.mouse = !app.config.mouse; app.status = app.t("status", app.config.mouse ? "mouse_on" : "mouse_off") }),
  select_theme: command("Choose theme", app => {
    const originalTheme = app.config.theme
    const themes = [...app.preferences.themes.keys()]
    app.picker(
      app.t("palette", "theme_picker_title"),
      themes.map(theme => ({
        label: theme,
        run: async () => {
          app.config.theme = theme
          await savePreference(app.workspace, "theme", theme)
          app.highlights.clear()
          await app.refreshAllHighlights()
          app.status = app.t("messages", "theme_status", { theme })
        }
      })),
      entry => {
        app.config.theme = entry.label
        app.highlights.clear()
        void app.refreshAllHighlights()
      },
      () => {
        app.config.theme = originalTheme
        app.highlights.clear()
        void app.refreshAllHighlights()
      }
    )
  }),
  select_locale: command("Display language", app => app.picker(app.t("palette", "locale_picker_title"), [...app.preferences.locales.keys()].map(locale => ({ label: locale, run: async () => { app.config.locale = locale; await savePreference(app.workspace, "locale", locale); app.changed() } })))),
  health_check: command("Environment health", app => {
    const text = [`Bun ${Bun.version}`, `Workspace: ${app.workspace}`, `Git: ${Bun.which("git") ?? "missing"}`, ...Object.entries(app.config.lsp.servers).map(([language, command]) => `${language}: ${command[0] ? Bun.which(command[0]) ?? "missing" : "unconfigured"}`), ...app.serverFailures.entries()].join("\n")
    app.overlay = { kind: "read", title: "Health", text, scroll: 0 }
  }),
  toggle_modal: toggle("Toggle Vim mode", app => { app.config.editor.modal_mode = !app.config.editor.modal_mode; app.vim = app.config.editor.modal_mode ? "normal" : "insert" }),
  toggle_auto_pairs: toggle("Auto-close pairs ({[\"')", app => {
    app.config.editor.auto_pairs = !app.config.editor.auto_pairs
    app.status = app.t("messages", app.config.editor.auto_pairs ? "auto_pairs_on" : "auto_pairs_off")
  }),
  toggle_line_numbers: toggle("Toggle line numbers", app => {
    app.config.show_line_numbers = !app.config.show_line_numbers
    app.status = app.t("messages", app.config.show_line_numbers ? "line_numbers_on" : "line_numbers_off")
  }),
  toggle_icons: toggle("Toggle file icons", app => {
    app.config.icons = !app.config.icons
    app.status = app.t("messages", app.config.icons ? "icons_on" : "icons_off")
  }),
  open_user_config: command("Open user settings (TOML)…", async app => {
    await app.openUserConfig()
  }),
  open_workspace_config: command("Open workspace settings (TOML)…", async app => {
    await app.openWorkspaceConfig()
  }),
  reload_config: command("Reload configuration", async app => {
    await app.reloadConfig()
  }),
  run_tasks: command("Run task", async app => {
    const tasks: { name: string; command: string }[] = []
    for (const path of [join(userConfigDir, "tasks.toml"), join(app.workspace, ".daisy", "tasks.toml"), join(app.workspace, ".oride", "tasks.toml")]) {
      const file = await readToml(path)
      if (!Array.isArray(file?.tasks)) continue
      for (const raw of file.tasks) { const task = record(raw); if (typeof task.name === "string" && typeof task.command === "string") tasks.push({ name: task.name, command: task.command }) }
    }
    if (!tasks.length) throw new Error(app.t("messages", "no_tasks"))
    app.picker("Tasks (project commands execute in your shell)", tasks.map(task => ({ label: task.name, detail: task.command, run: () => {
      app.overlay = { kind: "confirm", title: app.t("dialogs", "run_task_confirm", { name: task.name }), accept: () => {
        app.startTerminal()
        const path = app.document.path ?? ""
        const quote = (value: string) => "'" + value.replaceAll("'", "'\\''") + "'"
        const variables: Record<string, string> = { FILE_NAME: basename(path), FILE_STEM: basename(path).replace(/\.[^.]+$/, ""), FILE_DIR: dirname(path), FILE: path, WORKSPACE: app.workspace, LINE: String(app.document.position.line + 1), COL: String(app.document.position.character + 1) }
        const command = task.command.replace(/\$(FILE_NAME|FILE_STEM|FILE_DIR|WORKSPACE|FILE|LINE|COL)\b/g, (_match, key: string) => quote(variables[key] ?? ""))
        app.terminal?.write(command + "\r")
      } }
    } })))
  }),
}

function copy(app: Application): void {
  const document = app.document
  if (document.selections.every(selection => selection.anchor === selection.head)) {
    const line = document.position.line
    const end = line + 1 < lineRanges(document.text).length ? offsetAt(document.text, line + 1, 0) : document.text.length
    document.select(offsetAt(document.text, line, 0), end)
    if (!document.selectedText) document.select(offsetAt(document.text, line, 0), document.text.length)
  }
  app.clipboard = document.selectedText
  app.clipboardWrite = `\x1b]52;c;${Buffer.from(app.clipboard).toString("base64")}\x07`
}
function confirmGit(app: Application, operation: "pull" | "push"): void {
  app.overlay = { kind: "confirm", title: app.t("dialogs", operation === "pull" ? "git_pull_confirm" : "git_push_confirm"), accept: async () => { app.status = await app.git.sync(operation); await app.refreshGit() } }
}
export async function renameTreeSelection(app: Application, name: string): Promise<void> {
  const row = app.tree.rows[app.tree.selected]
  if (!row || row.path === app.workspace || !name.trim() || name.includes("/") || name === "." || name === "..") throw new Error("Invalid rename")
  const target = join(dirname(row.path), name)
  if (await stat(target).then(() => true).catch(() => false)) throw new Error("Target exists")
  await rename(row.path, target)
  for (const document of app.documents) if (document.path === row.path || document.path?.startsWith(row.path + "/")) document.path = target + document.path.slice(row.path.length)
  await app.tree.refresh(); app.invalidateFiles(); app.changed()
}
export function deleteTreeSelection(app: Application): void {
  const row = app.tree.rows[app.tree.selected]
  if (!row || row.path === app.workspace) return
  app.overlay = { kind: "confirm", title: `Delete ${relative(app.workspace, row.path)}? Enter confirms (directories must be empty)`, accept: async () => {
    if (row.directory) await rmdir(row.path); else await unlink(row.path)
    await app.tree.refresh(); app.invalidateFiles(); app.changed()
  } }
}

function actionSynonyms(actionId: string): string {
  switch (actionId) {
    case "select_theme":
      return "tema temas theme color cores visual live preview paleta"
    case "select_locale":
      return "idioma language locale tradução pt-br en-us"
    case "open_user_config":
      return "configuracao usuario toml settings preferences config"
    case "open_workspace_config":
      return "configuracao projeto workspace toml settings config"
    case "reload_config":
      return "recarregar reload config toml"
    case "toggle_tree":
      return "arvore tree explorer arquivos pasta file"
    case "toggle_terminal":
      return "terminal pty shell bash zsh console linha de comando"
    case "toggle_scm":
      return "git scm controle versao diff commit stage"
    case "toggle_md_preview":
      return "markdown preview visualizar render doc"
    case "toggle_modal":
      return "vim modal normal insert visual vi"
    case "toggle_soft_wrap":
      return "quebra linha wrap word wrap auto wrap"
    case "toggle_auto_pairs":
      return "pares auto close delimitadores brackets parenteses aspas"
    case "toggle_line_numbers":
      return "numeros linha line numbers"
    case "toggle_icons":
      return "icones nerd font glyphs simbolos"
    case "toggle_mouse":
      return "mouse clique cursor scroll"
    case "goto_line":
      return "ir linha go to line jump saltar"
    case "find":
      return "localizar buscar search find"
    case "replace":
      return "substituir replace trocar"
    case "project_find":
      return "buscar projeto find project grep procurar"
    case "project_replace":
      return "substituir projeto replace project"
    case "lsp_format":
      return "formatar codigo format prettier lsp"
    case "lsp_rename":
      return "renomear simbolo rename refactor lsp"
    case "lsp_goto_definition":
      return "definicao definition jump go to lsp"
    case "lsp_hover":
      return "hover documentacao info doc lsp"
    case "lsp_complete":
      return "completar autocompletar intellisense suggest"
    case "toggle_diagnostics":
      return "problemas erros warnings diagnostics lsp"
    case "git_blame":
      return "blame autor quem escreveu linha commit"
    case "git_pull":
      return "pull atualizar baixar sincronizar"
    case "git_push":
      return "push enviar subir remoto"
    case "show_diff":
      return "diff alteracoes mudancas git"
    case "help":
      return "ajuda atalhos keybindings help"
    case "which_key":
      return "which key atalhos essenciais guia cheatsheet"
    case "health_check":
      return "saude diagnostico health check lsp bun"
    case "run_tasks":
      return "tarefas tasks scripts run build test"
    case "undo_tree":
      return "historico undo history alteracoes"
    default:
      return ""
  }
}

export function buildPaletteEntries(app: Application): PickerEntry[] {
  const isPt = app.config.locale === "pt-BR"
  const entries: PickerEntry[] = []
  const seenActions = new Set<string>()

  // 1. Actions from menus with category prefixes
  for (const menu of menus) {
    const category = isPt ? menu.pt : menu.en
    for (const actionId of menu.actions) {
      if (seenActions.has(actionId)) continue
      seenActions.add(actionId)

      const baseLabel = app.label(actionId)
      const label = `${category}: ${baseLabel}`
      const chord = Object.entries(app.config.keys).find(([, action]) => action === actionId)?.[0]
      const synonyms = actionSynonyms(actionId)
      const detail = [chord, synonyms].filter(Boolean).join("  ·  ")

      entries.push({
        label,
        detail,
        run: () => app.dispatch(actionId),
      })
    }
  }

  // 2. Direct theme quick-pickers
  const allThemes = [...app.preferences.themes.keys()]
  for (const themeName of allThemes) {
    const isCurrent = app.config.theme.toLowerCase().replaceAll(" ", "-") === themeName.toLowerCase().replaceAll(" ", "-")
    const label = isPt ? `Tema: ${themeName}` : `Theme: ${themeName}`
    const detail = [
      isCurrent ? (isPt ? "[ativo]" : "[active]") : undefined,
      "tema temas theme color cores visual"
    ].filter(Boolean).join(" ")
    entries.push({
      label,
      detail,
      run: async () => {
        app.config.theme = themeName
        await savePreference(app.workspace, "theme", themeName)
        app.highlights.clear()
        await app.refreshAllHighlights()
        app.status = app.t("messages", "theme_status", { theme: themeName })
      },
    })
  }

  // 3. Direct locale quick-pickers
  const allLocales = [...app.preferences.locales.keys()]
  for (const loc of allLocales) {
    const isCurrent = app.config.locale === loc
    const label = isPt ? `Idioma: ${loc}` : `Language: ${loc}`
    const detail = [
      isCurrent ? (isPt ? "[ativo]" : "[active]") : undefined,
      "idioma language locale tradução"
    ].filter(Boolean).join(" ")
    entries.push({
      label,
      detail,
      run: async () => {
        app.config.locale = loc
        await savePreference(app.workspace, "locale", loc)
        app.changed()
      },
    })
  }

  // 4. Any other non-menu actionIds
  for (const id of actionIds) {
    if (seenActions.has(id) || id.startsWith("move_") || ["backspace", "delete", "insert_newline", "insert_tab"].includes(id)) continue
    seenActions.add(id)
    const label = app.label(id)
    const chord = Object.entries(app.config.keys).find(([, action]) => action === id)?.[0]
    const synonyms = actionSynonyms(id)
    const detail = [chord, synonyms].filter(Boolean).join("  ·  ")
    entries.push({
      label,
      detail,
      run: () => app.dispatch(id),
    })
  }

  // 5. Plugin commands
  entries.push(...app.pluginEntries())

  return entries
}
