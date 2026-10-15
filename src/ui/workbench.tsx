import { BoxRenderable, ImageRenderable, ScrollBoxRenderable, createHostClipboard, createClipboard, createRendererClipboardAdapter, SyntaxStyle, RGBA, TextAttributes, type OptimizedBuffer, type MarkdownOptions } from "@opentui/core"
import { useKeyboard, usePaste, useRenderer, useTerminalDimensions } from "@opentui/solid"
import { createSignal, createMemo, createEffect, onCleanup, onMount, For, Show } from "solid-js"
import { fileURLToPath } from "node:url"
import { join } from "node:path"
import { execute } from "../services/git"
import { selectedTheme } from "../services/preferences"
import type { Application } from "../app/application"

import { receiveKey, receivePaste } from "../app/input"
import { languageOf, resolvePreviewImage, extractOpenableLink } from "../services/languages"
import { offsetAt } from "../core/document"
import { EditorPane } from "./editor"
import { themeOf, resolveColor, type Theme } from "./theme"
import { clampTreeWidth, clampScmWidth, clampSplitRatio, clampTerminalHeight } from "../core/layout"
import { fileGlyph, folderGlyph, gitBranch, lspGlyph, diagnosticError, diagnosticWarning, searchGlyph } from "../services/icons"
import { menus } from "./menus"

export function Workbench(props: { app: Application; live?: boolean; onQuit?: () => void | Promise<void> }) {
  const app = props.app, renderer = useRenderer(), dimensions = useTerminalDimensions()
  const [revision, setRevision] = createSignal(0)
  const hostClipboard = props.live ? createHostClipboard({ timeoutMs: 1000, maxReadBytes: 1024 * 1024, maxWriteBytes: 1024 * 1024 }) : undefined
  const clipboard = hostClipboard ? createClipboard({ host: hostClipboard, terminal: createRendererClipboardAdapter(renderer) }) : undefined
  if (clipboard) app.readClipboard = async () => {
    const result = await clipboard.read({ preferredTypes: ["text/plain;charset=utf-8", "text/plain"] })
    if (result.status === "failed") app.report(result.error)
    return result.status === "read" ? new TextDecoder().decode(result.representation.bytes) : undefined
  }
  const unsubscribe = app.subscribe(() => setRevision(value => value + 1))
  const theme = createMemo(() => { revision(); return themeOf({ ...app.config, ui: { ...selectedTheme(app.config, app.preferences).ui, ...app.config.ui }, tokens: { ...selectedTheme(app.config, app.preferences).tokens, ...app.config.tokens } }) })
  const state = createMemo(() => { revision(); return { tree: app.showTree, scm: app.showScm, bottom: app.bottom, overlay: app.overlay, menu: app.menu, preview: app.preview && ["markdown", "mdx"].includes(languageOf(app.document.path, app.config)), documents: [...app.documents], panes: [...app.panes], completion: app.completion } })
  const columns = () => dimensions().width
  const height = () => dimensions().height
  const treeWidth = () => state().tree && columns() >= 45 ? Math.min(app.config.tree.width, Math.floor(columns() * 0.35)) : 0
  const scmWidth = () => state().scm && columns() >= 70 ? Math.min(app.config.scm.width, Math.floor(columns() * 0.3)) : 0
  const bottomHeight = () => state().bottom ? Math.max(3, Math.min(app.terminalHeight, Math.floor(height() / 2))) : 0
  const contentHeight = () => Math.max(1, height() - 3 - bottomHeight())
  const treeStart = () => Math.max(0, app.tree.selected - contentHeight() + 2)
  const scmStart = () => Math.max(0, app.scmSelected - contentHeight() + 3)
  const style = SyntaxStyle.fromStyles({
    default: { fg: RGBA.fromHex("#e6edf3") }, keyword: { fg: RGBA.fromHex("#ff7b72") }, string: { fg: RGBA.fromHex("#a5d6ff") },
    comment: { fg: RGBA.fromHex("#8b949e"), italic: true }, "markup.heading": { fg: RGBA.fromHex("#79c0ff"), bold: true },
    "markup.link": { fg: RGBA.fromHex("#79c0ff"), underline: true }, "markup.strong": { bold: true }, "markup.italic": { italic: true },
  }, renderer.nativeScene)
  const renderMarkdownNode: MarkdownOptions["renderNode"] = token => {
    if (token.type !== "paragraph" || !("tokens" in token) || !Array.isArray(token.tokens) || token.tokens.length !== 1) return undefined
    const image = token.tokens[0]
    if (image?.type !== "image") return undefined
    const path = resolvePreviewImage(image.href, app.document.path, app.workspace, app.config.markdown.terminal_images && app.config.theme !== "no-color")
    if (!path) return undefined
    return new ImageRenderable(renderer, { source: path, width: "100%", height: 12, fit: "fit", protocol: "auto", onError: error => app.report(error) })
  }
  const openLink = () => {
    const url = extractOpenableLink(renderer.getSelection()?.getSelectedText() ?? "")
    if (!url) { app.status = app.t("messages", "select_url_hint"); app.changed(); return }
    app.overlay = { kind: "confirm", title: app.t("dialogs", "open_browser_link", { url }), accept: async () => {
      const program = process.platform === "darwin" ? "open" : "xdg-open"
      await execute(program, [url], app.workspace)
    } }; app.changed()
  }
  let poll: ReturnType<typeof setInterval> | undefined
  let quitting = false
  let treeDivider: BoxRenderable | undefined
  let scmDivider: BoxRenderable | undefined
  let bottomDivider: BoxRenderable | undefined
  let splitDivider: BoxRenderable | undefined
  let terminalBox: BoxRenderable | undefined
  let mdScrollbox: ScrollBoxRenderable | undefined
  useKeyboard(key => {
    if (app.preview && !app.overlay && key.meta && (key.name === "return" || key.name === "enter")) { key.preventDefault(); key.stopPropagation(); openLink() }
    else receiveKey(app, key)
  })
  usePaste(event => { event.preventDefault(); event.stopPropagation(); receivePaste(app, new TextDecoder().decode(event.bytes)) })
  createEffect(() => {
    revision()
    app.width = columns(); app.height = height(); app.editorHeight = contentHeight()
    const palette = theme()
    const definitions = selectedTheme(app.config, app.preferences).syntax
    const syntax = (role: string, fallback: RGBA) => palette.noColor ? palette.text : app.config.syntax[role] || definitions[role] ? resolveColor(app.config.syntax[role] ?? definitions[role], "#e6edf3") : fallback
    style.registerStyle("default", { fg: palette.text })
    style.registerStyle("keyword", { fg: syntax("keyword", palette.accent) })
    style.registerStyle("string", { fg: syntax("string", palette.text) })
    style.registerStyle("comment", { fg: syntax("comment", palette.muted), italic: true })
    style.registerStyle("markup.heading", { fg: syntax("heading", palette.accent), bold: true })
    style.registerStyle("markup.link", { fg: syntax("link", palette.accent), underline: true })
    renderer.useMouse = app.config.mouse
    app.terminal?.resize(columns(), Math.max(1, bottomHeight() - 1))
    if (state().preview) {
      app.scrollPreview = (delta: number) => {
        if (mdScrollbox) {
          mdScrollbox.scrollBy(delta)
          app.previewScroll = mdScrollbox.scrollTop
        } else {
          app.previewScroll = Math.max(0, app.previewScroll + delta)
        }
        app.changed()
      }
      app.scrollPreviewTo = (y: number) => {
        if (mdScrollbox) {
          mdScrollbox.scrollTo(y)
          app.previewScroll = mdScrollbox.scrollTop
        } else {
          app.previewScroll = Math.max(0, y)
        }
        app.changed()
      }
    } else {
      app.scrollPreview = (delta: number) => { app.previewScroll = Math.max(0, app.previewScroll + delta); app.changed() }
      app.scrollPreviewTo = (y: number) => { app.previewScroll = Math.max(0, y); app.changed() }
    }
    if (props.live && app.clipboardWrite) { const text = app.clipboard; app.clipboardWrite = ""
      if (clipboard) void clipboard.writeText(text, { destination: "best-available" }).catch(error => app.report(error))
      else renderer.copyToClipboardOSC52(text) }
    if (app.quit && !quitting) { quitting = true; if (props.onQuit) void props.onQuit(); else renderer.destroy() }
  })
  onMount(() => { if (props.live) poll = setInterval(() => { void app.refreshGit(); void app.guard(() => app.pollDisk()) }, 3000) })
  onCleanup(() => {
    unsubscribe(); clearInterval(poll); style.destroy(); app.readClipboard = undefined
    app.scrollPreview = delta => { app.previewScroll = Math.max(0, app.previewScroll + delta); app.changed() }
    app.scrollPreviewTo = y => { app.previewScroll = Math.max(0, y); app.changed() }
    void clipboard?.dispose(); void hostClipboard?.dispose()
  })

  function terminalPaint(buffer: OptimizedBuffer): void {
    if (!terminalBox || !app.terminal) return
    const palette = theme()
    const cells = app.terminal.cells()
    const color = (value: number | undefined, indexed: boolean, fallback: RGBA) => value === undefined ? fallback : indexed ? RGBA.fromIndex(value) : RGBA.fromInts(value >> 16 & 255, value >> 8 & 255, value & 255)
    for (const [row, line] of cells.entries()) {
      for (const [column, cell] of line.entries()) {
        if (cell.width === 0 || row >= terminalBox.height || column >= terminalBox.width) continue
        const fg = palette.noColor ? palette.text : color(cell.foreground, cell.foregroundPalette, palette.text)
        const bg = palette.noColor ? palette.panel : color(cell.background, cell.backgroundPalette, palette.panel)
        buffer.drawText(cell.text, terminalBox.x + column, terminalBox.y + row, cell.inverse ? bg : fg, cell.inverse ? fg : bg,
          (cell.bold ? TextAttributes.BOLD : 0) | (cell.italic ? TextAttributes.ITALIC : 0) | (cell.underline ? TextAttributes.UNDERLINE : 0))
      }
    }
    const cursor = app.terminal.cursor
    if (app.focus === "terminal" && !app.overlay && !app.menu && app.terminal.scroll === 0 && cursor.row < terminalBox.height && cursor.column < terminalBox.width) {
      const glyph = cells[cursor.row]?.[cursor.column]?.text ?? " "
      buffer.drawText(glyph, terminalBox.x + cursor.column, terminalBox.y + cursor.row, palette.panel, palette.cursor, palette.noColor ? TextAttributes.INVERSE : 0)
    }
  }

  return <box id="workbench" width="100%" height="100%" flexDirection="column" backgroundColor={theme().surface} overflow="hidden" renderBefore={buffer => {
    if (theme().noColor) for (let row = 0; row < height(); row++) buffer.drawText(" ".repeat(columns()), 0, row, RGBA.defaultForeground(), RGBA.defaultBackground())
  }}>
    <box id="menubar" height={1} flexShrink={0} flexDirection="row" backgroundColor={theme().panel}>
      <Show when={theme().isLain}>
        <text content=" ✳ DAISY_ " fg={theme().accent} selectable={false} />
      </Show>
      <For each={menus}>{(menu, index) => {
        const isOpen = () => state().menu?.column === index()
        const showUnderline = () => app.altActive || state().menu !== undefined
        const isPt = () => app.config.locale === "pt-BR"
        const raw = () => {
          let text = isPt() ? menu.pt : menu.en
          if (theme().isLain && text.toLowerCase().startsWith("config")) text = "Config"
          return text
        }
        const text = () => theme().isLain ? raw().toUpperCase() : raw()
        const lead = () => theme().isLain ? ` 0${index() + 1} ` : " "
        const mnemonic = () => (isPt() ? menu.mnemonicPt : menu.mnemonicEn).toLowerCase()
        const charIdx = () => text().toLowerCase().indexOf(mnemonic())
        const before = () => charIdx() >= 0 ? text().slice(0, charIdx()) : text()
        const m = () => charIdx() >= 0 ? text().charAt(charIdx()) : ""
        const after = () => charIdx() >= 0 ? text().slice(charIdx() + 1) : ""

        return <text id={`menu-${index()}`}
          fg={isOpen() ? (theme().isLain ? theme().surface : theme().accent) : theme().text}
          bg={isOpen() && theme().isLain ? theme().accent : undefined}
          selectable={false}
          onMouseDown={() => {
            app.altActive = !isOpen()
            app.menu = isOpen() ? undefined : { column: index(), selected: 0 }
            app.changed()
          }}>
          {showUnderline() && charIdx() >= 0
            ? <>{lead()}{before()}<u><b>{m()}</b></u>{after()} </>
            : `${lead()}${text()} `}
        </text>
      }}</For>
      <text flexGrow={1} textAlign="right" fg={theme().isLain ? theme().accent : theme().muted}
        content={theme().isLain ? (columns() >= 80 ? "● CONNECTED / PORT: 98 " : "● WIRED: 98 ") : "Daisy / Solid "}
        selectable={false} />
    </box>
    <box id="tabs-bar" height={1} flexShrink={0} flexDirection="row" backgroundColor={theme().panel} overflow="hidden">
      <For each={state().documents}>{(document, index) => {
        const isActive = () => document.id === app.activeId
        const tabContent = () => {
          if (theme().isLain) {
            return ` [0${index() + 1}] ${document.dirty ? "● " : ""}${app.config.icons ? fileGlyph(document.title) + " " : ""}${document.title}${isActive() ? " ×" : " "} `
          }
          return ` ${isActive() ? "▍" : " "}${document.dirty ? "● " : ""}${app.config.icons ? fileGlyph(document.title) + " " : ""}${document.title}${isActive() ? " ×" : " "} `
        }
        return <box flexDirection="row" backgroundColor={isActive() ? (theme().isLain ? theme().selection : theme().surface) : theme().panel}>
          <text id={`tab-${document.id}`} content={tabContent()}
            fg={isActive() ? theme().accent : theme().muted} selectable={false}
            onMouseDown={event => {
              if (event.button === 1) {
                app.closeTab(document.id)
              } else {
                app.activate(document.id)
              }
            }} />
        </box>
      }}</For>
      <text flexGrow={1} textAlign="right" fg={theme().muted}
        content={theme().isLain
          ? ` WIRED://${app.workspace.split("/").at(-1) || "root"}${app.document.path ? `/${app.document.title}` : ""} `
          : ` ${app.workspace.split("/").at(-1) || "workspace"}${app.document.path ? ` › ${app.document.title}` : ""} `}
        selectable={false} />
    </box>
    <box id="main-region" flexGrow={1} minHeight={1} flexDirection="row" overflow="hidden">
      <Show when={treeWidth() > 0}>
        <box id="file-tree" width={treeWidth()} minWidth={1} flexShrink={0} flexDirection="column" overflow="hidden" backgroundColor={theme().panel}>
          <text content={(theme().isLain ? (app.focus === "tree" ? "▣ PROJECT // [N:NEW R:REN D:DEL]" : "□ PROJECT // [N:NEW R:REN D:DEL]") : (app.focus === "tree" ? "▍PROJECT  n:new r:ren d:del" : " PROJECT  n:new r:ren d:del")) + (app.tree.truncated ? ` ${app.tree.rows.length}+` : "")} fg={app.focus === "tree" ? theme().accent : theme().muted} height={1} selectable={false} />
          <For each={(revision(), app.tree.rows.slice(treeStart(), treeStart() + contentHeight() - 1))}>{(row, index) => {
            const isSelected = () => app.tree.selected === treeStart() + index()
            const gitBadge = () => {
              const entry = (app.gitStatus.entries ?? []).find(entry => join(app.workspace, entry.path) === row.path)
              return entry?.badge || (entry ? (entry.worktree === "M" || entry.index === "M" ? "M" : entry.index === "A" ? "A" : "?") : "")
            }
            return <text
              id={`tree-row-${treeStart() + index()}`} height={1} wrapMode="none" selectable={false}
              fg={isSelected() && app.focus === "tree" ? theme().accent : theme().text}
              content={`${isSelected() ? (app.focus === "tree" ? "▎" : ">") : " "}${"  ".repeat(row.depth)}${row.directory ? (row.expanded ? "▾ " : "▸ ") : "  "}${app.config.icons ? (row.directory ? folderGlyph(row.expanded) : fileGlyph(row.name)) + " " : ""}${row.name}${gitBadge() ? " " + gitBadge() : ""}`}
              onMouseDown={() => {
                if (app.overlay || app.menu) return
                const wasSelected = app.focus === "tree" && app.tree.selected === treeStart() + index()
                app.focus = "tree"; app.tree.selected = treeStart() + index()
                if (row.directory) void app.guard(() => app.tree.toggle(row))
                else if (wasSelected) void app.guard(() => app.openFile(row.path))
                app.changed()
              }} />
          }}</For>
        </box>
        <box ref={value => { treeDivider = value }} width={1} flexShrink={0} backgroundColor={theme().border}
          onMouseDrag={event => { if (!app.overlay && !app.menu && treeDivider) { app.config.tree.width = clampTreeWidth(event.x, columns()); app.changed() } }} />
      </Show>
      <box id="editor-area" flexGrow={1} minWidth={1} minHeight={1} overflow="hidden" flexDirection={app.orientation === "vertical" ? "row" : "column"} paddingLeft={1} paddingRight={1}>
        <Show when={!state().preview} fallback={<scrollbox ref={value => { mdScrollbox = value; if (value && app.previewScroll > 0) value.scrollTo(app.previewScroll) }} id="markdown-preview" flexGrow={1} minWidth={1} minHeight={1}
          onMouseScroll={event => {
            if (mdScrollbox && !app.overlay && !app.menu) {
              const delta = event.scroll?.direction === "up" ? -3 : 3
              mdScrollbox.scrollBy(delta)
              app.previewScroll = mdScrollbox.scrollTop
              app.changed()
            }
          }}><markdown content={(revision(), app.document.text)} syntaxStyle={style} fg={theme().text} renderNode={renderMarkdownNode} streaming internalBlockMode="top-level" /></scrollbox>}>
          <For each={state().panes}>{(pane, index) => <>
            <Show when={index() > 0}><box ref={value => { splitDivider = value }} width={app.orientation === "vertical" ? 1 : "100%"} height={app.orientation === "horizontal" ? 1 : "100%"} flexShrink={0} backgroundColor={theme().panel}
              onMouseDrag={event => { if (splitDivider && !app.overlay) { app.splitRatio = app.orientation === "vertical" ? clampSplitRatio(event.x - treeWidth(), columns() - treeWidth() - scmWidth()) : clampSplitRatio(event.y - 2, contentHeight()); app.changed() } }} /></Show>
            <box flexGrow={index() === 0 && state().panes.length === 2 ? app.splitRatio : state().panes.length === 2 ? 100 - app.splitRatio : 1} flexBasis={0} minWidth={1} minHeight={1}>
              <EditorPane app={app} pane={pane} index={index()} revision={revision} />
            </box>
          </>}</For>
        </Show>
      </box>
      <Show when={scmWidth() > 0}>
        <box ref={value => { scmDivider = value }} width={1} flexShrink={0} backgroundColor={theme().border}
          onMouseDrag={event => { if (!app.overlay && !app.menu && scmDivider) { app.config.scm.width = clampScmWidth(event.x, columns()); app.changed() } }} />
        <box id="source-control" width={scmWidth()} flexShrink={0} minWidth={1} flexDirection="column" backgroundColor={theme().panel} overflow="hidden">
          <text height={1} fg={theme().accent} content={theme().isLain ? (app.focus === "scm" ? "▣ SCM // [S:STAGE U:UNSTAGE C:COMMIT]" : "□ SCM") : (app.focus === "scm" ? "▍Git  s:stage u:unstage c:commit" : " Git")} />
          <Show when={(app.gitStatus.entries?.length ?? 0) > 0} fallback={<text fg={theme().muted} content={app.gitStatus.branch ? "Working tree clean" : "Not a Git repository"} />}>
            <For each={(revision(), (app.gitStatus.entries ?? []).slice(scmStart(), scmStart() + contentHeight() - 2))}>{(entry, index) => <text height={1} wrapMode="none" selectable={false}
              content={`${app.scmSelected === scmStart() + index() ? ">" : " "}${entry.index}${entry.worktree} ${entry.path}`} fg={app.scmSelected === scmStart() + index() && app.focus === "scm" ? theme().accent : theme().text}
              onMouseDown={() => { if (!app.overlay && !app.menu) { app.focus = "scm"; app.scmSelected = scmStart() + index(); app.changed() } }} />}</For>
          </Show>
        </box>
      </Show>
    </box>
    <Show when={state().bottom}>
      <box ref={value => { bottomDivider = value }} id="panel-tabs" height={1} flexShrink={0} flexDirection="row" backgroundColor={theme().panel}
        onMouseDrag={event => { if (bottomDivider && !app.overlay) { app.terminalHeight = clampTerminalHeight(event.y, height()); app.changed() } }}>
        <text content={theme().isLain ? (app.bottom === "terminal" ? "[▶ TERMINAL] " : "[ TERMINAL ] ") : `${app.bottom === "terminal" ? "▍" : " "}Terminal `} fg={theme().accent} selectable={false} onMouseDown={() => { if (!app.overlay) void app.guard(() => app.startTerminal()) }} />
        <text content={theme().isLain ? (app.bottom === "problems" ? "[▶ PROBLEMS] " : "[ PROBLEMS ] ") : `${app.bottom === "problems" ? "▍" : " "}Problems `} fg={theme().text} selectable={false} onMouseDown={() => { if (!app.overlay) { app.bottom = "problems"; app.focus = "editor"; app.changed() } }} />
      </box>
      <Show when={state().bottom === "terminal"} fallback={<box height={Math.max(1, bottomHeight() - 1)} flexShrink={0} flexDirection="column" overflow="hidden" backgroundColor={theme().panel}>
        <Show when={app.diagnostics.length} fallback={<text content={!app.config.lsp.enabled ? "LSP disabled" : app.servers.size ? "No diagnostics reported" : "No LSP attached · command palette → Environment health"} fg={theme().muted} />}>
          <For each={(revision(), app.diagnostics)}>{entry => <text content={`${entry.diagnostic.severity === 1 ? "E" : "W"} ${fileURLToPath(entry.uri)}:${entry.diagnostic.range.start.line + 1} ${entry.diagnostic.message}`}
            fg={entry.diagnostic.severity === 1 ? theme().error : theme().warning} selectable={false} onMouseDown={() => { void app.guard(async () => { await app.openFile(fileURLToPath(entry.uri)); app.document.select(offsetAt(app.document.text, entry.diagnostic.range.start.line, entry.diagnostic.range.start.character)); app.changed() }) }} />}</For>
        </Show>
      </box>}>
        <box ref={value => { terminalBox = value }} id="terminal" height={Math.max(1, bottomHeight() - 1)} flexShrink={0} backgroundColor={theme().panel} renderAfter={terminalPaint}
          onMouseDown={() => { if (!app.overlay && !app.menu) { app.focus = "terminal"; app.changed() } }}
          onMouseScroll={event => { if (app.terminal && !app.overlay) { app.terminal.scroll = Math.max(0, Math.min(2000, app.terminal.scroll + (event.scroll?.direction === "up" ? 3 : -3))); app.changed() } }} />
      </Show>
    </Show>
    <StatusBar app={app} revision={revision} theme={theme} width={columns} />
    <Show when={state().menu && !state().overlay}><MenuDropdown app={app} revision={revision} theme={theme} /></Show>
    <Show when={state().overlay}><OverlayView app={app} revision={revision} theme={theme} width={columns} height={height} /></Show>
    <Show when={state().completion && !state().overlay && !state().menu && app.focus === "editor"}>
      <CompletionDropdown app={app} revision={revision} theme={theme} />
    </Show>
  </box>
}

function StatusBar(props: { app: Application; revision: () => number; theme: () => Theme; width: () => number }) {
  const app = props.app
  const theme = props.theme
  const isModal = () => { props.revision(); return app.config.editor.modal_mode }
  const vimMode = () => { props.revision(); return (app.vim ?? "normal").toUpperCase() }
  const isDirty = () => { props.revision(); return app.document.dirty }
  const focus = () => { props.revision(); return app.focus }
  const lang = () => { props.revision(); return languageOf(app.document.path, app.config) }
  const width = () => props.width()
  const diagnostics = () => { props.revision(); return app.diagnostics }
  const errors = () => diagnostics().filter(d => d.diagnostic.severity === 1).length
  const warnings = () => diagnostics().filter(d => d.diagnostic.severity === 2).length
  const status = () => { props.revision(); return app.status }
  const pos = () => { props.revision(); return app.document.position }
  const selectionsCount = () => { props.revision(); return app.document.selections.length }
  const git = () => { props.revision(); return app.gitStatus }
  const indent = () => {
    props.revision()
    const insertSpaces = app.document.indent?.insertSpaces ?? app.config.editor.insert_spaces
    const tabSize = app.document.indent?.tabSize ?? app.config.editor.tab_size
    return insertSpaces ? `Spaces: ${tabSize}` : `Tabs: ${tabSize}`
  }

  const modeBg = () => {
    if (theme().noColor) return theme().panel
    if (!isModal()) return theme().accent
    if (vimMode() === "INSERT") return theme().selection
    if (vimMode() === "VISUAL") return theme().warning
    return theme().accent
  }
  const modeFg = () => {
    if (theme().noColor) return theme().text
    if (!isModal()) return theme().surface
    if (vimMode() === "INSERT") return theme().text
    if (vimMode() === "VISUAL") return theme().surface
    return theme().surface
  }
  const modeLabel = () => {
    if (theme().isLain) {
      return isModal() ? ` [ 001 ${vimMode()} ] ` : " [ ✳ DAISY ] "
    }
    return isModal() ? ` ${vimMode()} ` : " DAISY "
  }

  return (
    <box id="statusbar" height={1} flexShrink={0} flexDirection="row" justifyContent="space-between" backgroundColor={theme().panel} overflow="hidden">
      {/* Left zone: Mode, Dirty indicator, Focus, Git */}
      <box flexDirection="row" flexShrink={0} alignItems="center">
        <text content={modeLabel()} fg={modeFg()} bg={modeBg()} selectable={false}
          onMouseDown={() => { void app.dispatch("toggle_modal") }} />
        <Show when={isDirty()}>
          <text content={theme().isLain ? " [● REC]" : " ●"} fg={theme().isLain ? theme().error : theme().accent} selectable={false}
            onMouseDown={() => { void app.dispatch("save") }} />
        </Show>
        <text content={theme().isLain ? ` //${focus().toUpperCase()}` : ` ${focus()}`} fg={theme().text} selectable={false}
          onMouseDown={() => { app.focus = "editor"; app.changed() }} />
        <Show when={git().branch && width() >= 55}>
          <text content=" │ " fg={theme().muted} selectable={false} />
          <text content={theme().isLain ? `//GIT:${git().branch}${git().ahead > 0 ? ` ↑${git().ahead}` : ""}${git().behind > 0 ? ` ↓${git().behind}` : ""}` : `${app.config.icons ? gitBranch + " " : ""}${git().branch}${git().ahead > 0 ? ` ↑${git().ahead}` : ""}${git().behind > 0 ? ` ↓${git().behind}` : ""}`}
            fg={theme().muted} selectable={false}
            onMouseDown={() => { void app.dispatch("toggle_scm") }} />
        </Show>
      </box>

      {/* Center zone: Status messages or essential shortcuts hint */}
      <box flexGrow={1} flexShrink={1} overflow="hidden" paddingLeft={1} paddingRight={1} alignItems="center">
        <Show when={status()} fallback={
          <Show when={width() >= 65}>
            <text content={theme().isLain ? (width() >= 85 ? "SIGNAL_STABLE ▰▰▰▰▰▱▱▱" : "SIGNAL ▰▰▰▰▰") : "Ctrl+P files · Ctrl+Shift+P commands"} fg={theme().muted} wrapMode="none" selectable={false} />
          </Show>
        }>
          <text content={theme().isLain ? `▶ ${status()}` : `● ${status()}`} fg={theme().accent} wrapMode="none" selectable={false}
            onMouseDown={() => { app.status = ""; app.changed() }} />
        </Show>
      </box>

      {/* Right zone: LSP/Diagnostics, Language, Indent, Cursor position */}
      <box flexDirection="row" flexShrink={0} alignItems="center">
        {/* LSP / Diagnostics */}
        <Show when={width() >= 65}>
          <Show when={!app.config.lsp.enabled} fallback={
            <Show when={app.servers.size} fallback={<text content={theme().isLain ? "[ LSP: — ] │ " : "LSP — │ "} fg={theme().muted} selectable={false} />}>
              <Show when={diagnostics().length > 0} fallback={
                <text content={theme().isLain ? "[ LSP: OK ] │ " : `${app.config.icons ? lspGlyph + " " : ""}LSP clean │ `} fg={theme().muted} selectable={false}
                  onMouseDown={() => { void app.dispatch("toggle_diagnostics") }} />
              }>
                <box flexDirection="row" onMouseDown={() => { void app.dispatch("toggle_diagnostics") }}>
                  <Show when={errors() > 0}>
                    <text content={`${app.config.icons ? diagnosticError + " " : "E:"}${errors()} `} fg={theme().error} selectable={false} />
                  </Show>
                  <Show when={warnings() > 0}>
                    <text content={`${app.config.icons ? diagnosticWarning + " " : "W:"}${warnings()} `} fg={theme().warning} selectable={false} />
                  </Show>
                  <text content="│ " fg={theme().muted} selectable={false} />
                </box>
              </Show>
            </Show>
          }>
            <text content={theme().isLain ? "[ LSP: OFF ] │ " : "LSP off │ "} fg={theme().muted} selectable={false} />
          </Show>
        </Show>

        {/* Encoding (Lain only) */}
        <Show when={theme().isLain && width() >= 105}>
          <text content="[ UTF-8 ] │ " fg={theme().muted} selectable={false} />
        </Show>

        {/* Language */}
        <Show when={width() >= 55}>
          <text content={theme().isLain ? `[ ${lang().toUpperCase()} ] │ ` : `${app.config.icons ? fileGlyph(app.document.path) + " " : ""}${lang()} │ `} fg={theme().muted} selectable={false}
            onMouseDown={() => { void app.dispatch("command_palette") }} />
        </Show>

        {/* Indent */}
        <Show when={width() >= (theme().isLain ? 115 : 85)}>
          <text content={theme().isLain ? `[ ${indent()} ] │ ` : `${indent()} │ `} fg={theme().muted} selectable={false} />
        </Show>

        {/* Position pill: Ln X, Col Y */}
        <box backgroundColor={theme().noColor ? theme().panel : (theme().isLain ? theme().selection : theme().surface)} paddingLeft={1} paddingRight={1}
          onMouseDown={() => { void app.dispatch("goto_line") }}>
          <text content={theme().isLain ? `[ Ln ${pos().line + 1}, Col ${pos().character + 1}${selectionsCount() > 1 ? ` (${selectionsCount()})` : ""} ]` : `Ln ${pos().line + 1}, Col ${pos().character + 1}${selectionsCount() > 1 ? ` [${selectionsCount()}]` : ""}`}
            fg={theme().isLain ? theme().accent : theme().text} selectable={false} />
        </box>
        <Show when={theme().isLain && width() >= 120}>
          <text content=" [ ✳ FX: ON ]" fg={theme().accent} selectable={false} />
        </Show>
      </box>
    </box>
  )
}

function MenuDropdown(props: { app: Application; revision: () => number; theme: () => Theme }) {
  const menu = () => { props.revision(); return props.app.menu }
  const entries = () => menus[menu()?.column ?? 0]?.actions ?? []
  return <box id="menu-dropdown" position="absolute" top={1} left={Math.min((menu()?.column ?? 0) * 8, Math.max(0, props.app.width - 48))} width={Math.min(48, props.app.width)} height={entries().length + 2} zIndex={50}
    border borderStyle={props.theme().isLain ? "single" : "rounded"} borderColor={props.theme().accent} backgroundColor={props.theme().panel} flexDirection="column">
    <For each={entries()}>{(action, index) => {
      const isSelected = () => menu()?.selected === index()
      return <text height={1} selectable={false}
        fg={isSelected() ? (props.theme().isLain ? props.theme().surface : props.theme().accent) : props.theme().text}
        bg={isSelected() && props.theme().isLain ? props.theme().accent : undefined}
        content={`${isSelected() ? (props.theme().isLain ? "▶ " : "▎") : " "}${props.app.menuLabel(action)} ${Object.entries(props.app.config.keys).find(([, id]) => id === action)?.[0] ?? ""}`}
        onMouseDown={() => { void props.app.dispatch(action) }} />
    }}</For>
  </box>
}

function OverlayView(props: { app: Application; revision: () => number; theme: () => Theme; width: () => number; height: () => number }) {
  const overlay = () => { props.revision(); return props.app.overlay }
  const title = () => { const value = overlay(); return value?.kind === "search" ? `${value.project ? "Project" : "Buffer"} ${value.replace ? "find & replace" : "find"}` : value?.title ?? "" }
  const rows = () => Math.max(1, props.height() - 9)
  const entries = () => { props.revision(); const value = overlay(); return value?.kind === "picker" ? props.app.pickerEntries : [] }
  const start = () => { const value = overlay(); return Math.max(0, (value?.kind === "picker" || value?.kind === "search" ? value.selected : 0) - rows() + 1) }
  return <box id="overlay-scrim" position="absolute" top={0} left={0} width="100%" height="100%" zIndex={100} alignItems="center" justifyContent="center"
    onMouseDown={event => event.stopPropagation()} onMouseScroll={event => event.stopPropagation()}>
    <box id="overlay" width={Math.min(90, Math.max(20, props.width() - 4))} maxHeight={Math.max(3, props.height() - 2)} flexDirection="column" border borderStyle={props.theme().isLain ? "single" : "rounded"} borderColor={props.theme().accent} backgroundColor={props.theme().panel} paddingX={1} overflow="hidden">
      <text height={1} fg={props.theme().accent} content={props.theme().isLain ? `▣ DAISY://${title().toUpperCase()} _ □ ×` : `◈ ${title()}`} />
      <Show when={overlay()?.kind === "confirm"}><text content="Enter: confirm   Esc: cancel" fg={props.theme().warning} /></Show>
      <Show when={overlay()?.kind === "prompt"}><text content={(() => { const value = overlay(); return value?.kind === "prompt" ? `${props.theme().isLain ? "root@daisy:~# " : "> "}${value.value}▏` : "" })()} fg={props.theme().text} /><text fg={props.theme().muted} content="Enter: accept · Esc: cancel · Ctrl+U: clear" /></Show>
      <Show when={overlay()?.kind === "picker"}>
        <text height={1} content={(() => { const value = overlay(); return value?.kind === "picker" ? `${props.theme().isLain ? "QUERY:// " : (props.app.config.icons ? searchGlyph + " " : "> ")}${value.query}▏` : "" })()} fg={props.theme().text} />
        <Show when={entries().length} fallback={<text content="No matching entries" fg={props.theme().muted} />}>
          <For each={entries().slice(start(), start() + rows())}>{(entry, index) => {
            const isSelected = () => overlay()?.kind === "picker" && (overlay() as any).selected === start() + index()
            const entryIcon = () => {
              if (!props.app.config.icons) return ""
              const t = title().toLowerCase()
              if (t.includes("file") || t.includes("buffer") || t.includes("arquivo")) return fileGlyph(entry.label) + " "
              if (t.includes("theme") || t.includes("tema")) return " "
              if (t.includes("command") || t.includes("comando")) return "󰘳 "
              return ""
            }
            return <text height={1} wrapMode="none" selectable={false}
              fg={isSelected() ? (props.theme().isLain ? props.theme().surface : props.theme().accent) : props.theme().text}
              bg={isSelected() && props.theme().isLain ? props.theme().accent : undefined}
              content={`${isSelected() ? (props.theme().isLain ? "▶ " : "▎") : " "}${entryIcon()}${entry.label}${entry.detail ? "  [" + entry.detail + "]" : ""}`}
              onMouseDown={() => { props.app.overlay = undefined; void props.app.guard(async () => { await entry.run(); props.app.afterEdit() }) }} />
          }}</For>
        </Show>
        <text height={1} content={props.theme().isLain ? "↑↓ NAVIGATE · ENTER EXECUTE · ESC ABORT" : "↑↓ preview/select · Enter open · Esc close"} fg={props.theme().muted} />
      </Show>
      <Show when={overlay()?.kind === "read"}><text content={(() => { const value = overlay(); return value?.kind === "read" ? value.text.split("\n").slice(value.scroll, value.scroll + rows()).join("\n") : "" })()} fg={props.theme().text} /><text content="↑↓ scroll · Esc close" fg={props.theme().muted} /></Show>
      <Show when={overlay()?.kind === "search"}>
        <text height={1} content={(() => { const value = overlay(); return value?.kind === "search" ? `${value.field === 0 ? "▎" : " "} Find: ${value.query}▏` : "" })()} fg={props.theme().text} />
        <Show when={(() => { const value = overlay(); return value?.kind === "search" && value.replace })()}><text height={1} content={(() => { const value = overlay(); return value?.kind === "search" ? `${value.field === 1 ? "▎" : " "} Replace: ${value.replacement}▏` : "" })()} fg={props.theme().text} /></Show>
        <Show when={(() => { const value = overlay(); return value?.kind === "search" && value.project })()}><text height={1} content={(() => { const value = overlay(); return value?.kind === "search" ? `${value.field === 2 ? "▎" : " "} Glob: ${value.glob || "*"}` : "" })()} fg={props.theme().text} /></Show>
        <text height={1} fg={props.theme().muted} content={(props.revision(), `Alt+C case:${props.app.searchOptions.caseSensitive ? "on" : "off"} Alt+A accents:${props.app.searchOptions.ignoreAccents ? "off" : "on"} Alt+W word:${props.app.searchOptions.wholeWord ? "on" : "off"} Alt+R regex:${props.app.searchOptions.regex ? "on" : "off"}`)} />
        <text height={1} fg={props.theme().accent} content={(() => { const value = overlay(); return value?.kind === "search" ? value.error || `${value.project ? value.hits.length : props.app.matches.length} matches` : "" })()} />
        <For each={(() => { const value = overlay(); return value?.kind === "search" ? value.hits.slice(start(), start() + Math.max(1, rows() - 4)) : [] })()}>{(hit, index) => <text height={1} wrapMode="none" selectable={false} fg={props.theme().text} content={`${(() => { const value = overlay(); return value?.kind === "search" && value.selected === start() + index() ? "▎" : " " })()} ${hit.path}:${hit.line + 1} ${hit.text}`}
          onMouseDown={() => { const value = overlay(); if (value?.kind === "search") { value.selected = start() + index(); void props.app.guard(() => props.app.openSearchHit()) } }} />}</For>
        <text height={1} fg={props.theme().muted} content="Tab fields · Enter next/open · Alt+Enter replace · Ctrl+Alt+Enter all" />
      </Show>
    </box>
  </box>
}

function CompletionDropdown(props: { app: Application; revision: () => number; theme: () => Theme }) {
  const comp = () => { props.revision(); return props.app.completion }
  const items = () => comp()?.items ?? []
  const count = () => items().length
  const selected = () => comp()?.selected ?? 0
  const maxVisible = () => Math.min(8, Math.max(3, props.app.height - 6))
  const visibleCount = () => Math.min(maxVisible(), count())

  const scrollStart = () => {
    const sel = selected()
    const max = maxVisible()
    if (sel < max) return 0
    return Math.min(sel - max + 1, count() - max)
  }

  const visibleItems = () => items().slice(scrollStart(), scrollStart() + maxVisible())

  const cursorX = () => {
    if (props.app.cursorScreen) return props.app.cursorScreen.x
    const treeW = props.app.showTree && props.app.width >= 45 ? Math.min(props.app.config.tree.width, Math.floor(props.app.width * 0.35)) : 0
    return treeW + 5 + props.app.document.position.character
  }
  const cursorY = () => {
    if (props.app.cursorScreen) return props.app.cursorScreen.y
    return 2 + props.app.document.position.line
  }

  const dropdownWidth = () => Math.min(50, Math.max(30, props.app.width - 4))
  const dropdownHeight = () => visibleCount() + 2

  const top = () => {
    const y = cursorY()
    const h = dropdownHeight()
    if (y + 1 + h <= props.app.height - 1) {
      return y + 1
    }
    return Math.max(2, y - h)
  }

  const left = () => {
    const x = cursorX()
    const w = dropdownWidth()
    return Math.max(0, Math.min(x, props.app.width - w - 1))
  }

  const kindBadge = (kind?: string) => {
    switch (kind) {
      case "keyword": return "kw"
      case "function": return "fn"
      case "type": case "interface": case "class": return "typ"
      case "macro": return "m!"
      case "word": return "buf"
      case "lsp": return "lsp"
      default: return kind ? kind.slice(0, 3) : "id"
    }
  }

  return (
    <box
      id="completion-dropdown"
      position="absolute"
      top={top()}
      left={left()}
      width={dropdownWidth()}
      height={dropdownHeight()}
      zIndex={60}
      border
      borderStyle="rounded"
      borderColor={props.theme().accent}
      backgroundColor={props.theme().panel}
      flexDirection="column"
      overflow="hidden"
    >
      <For each={visibleItems()}>
        {(item, index) => {
          const actualIndex = () => scrollStart() + index()
          const isSelected = () => selected() === actualIndex()
          const badge = () => kindBadge(item.kind)
          return (
            <text
              id={`completion-item-${actualIndex()}`}
              height={1}
              wrapMode="none"
              selectable={false}
              fg={isSelected() ? props.theme().accent : props.theme().text}
              content={`${isSelected() ? "▎" : " "}[${badge()}] ${item.label}${item.detail ? "  " + item.detail : ""}`}
              onMouseDown={event => {
                event.stopPropagation()
                props.app.acceptCompletion(actualIndex())
              }}
            />
          )
        }}
      </For>
    </box>
  )
}

