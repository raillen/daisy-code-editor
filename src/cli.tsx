#!/usr/bin/env bun
import { createCliRenderer } from "@opentui/core"
import { render, testRender } from "@opentui/solid"
import { Application } from "./app/application"
import { Workbench } from "./ui/workbench"

const usage = `daisy — Daisy Code Editor (OpenTUI + Solid / Bun)

  daisy [FILE...|DIRECTORY]
  daisy --theme <THEME> [FILE...|DIRECTORY]
  daisy --print-frame [WxH] [FILE|DIRECTORY]
  daisy --version
  daisy --help

Ctrl+P files · Ctrl+Shift+P commands · Alt+F/E/V/G/I/H menus
Ctrl+S save · Ctrl+Q quit · Ctrl+\` terminal · Ctrl+Shift+G Git

Configuration: ~/.config/daisy/config.toml + .daisy/config.toml (compatible with .oride)
`

export async function run(args: string[]): Promise<void> {
  if (args.includes("--version") || args.includes("-V")) { console.log("daisy 0.2.0 (Daisy Code Editor · OpenTUI/Solid)"); return }
  if (args.includes("--help") || args.includes("-h")) { console.log(usage); return }
  const frame = args[0] === "--print-frame"
  let width = 100, height = 30
  if (frame) {
    args = args.slice(1)
    if (args[0]?.match(/^\d+x\d+$/)) {
      const values = args.shift()?.split("x").map(Number)
      width = values?.[0] ?? width; height = values?.[1] ?? height
      if (width < 20 || height < 5 || width > 500 || height > 300) throw new Error("Frame dimensions must be 20..500 by 5..300")
    }
  }
  let customTheme: string | undefined
  const themeIndex = args.indexOf("--theme")
  if (themeIndex !== -1) {
    customTheme = args[themeIndex + 1]
    if (!customTheme || customTheme.startsWith("-")) throw new Error("Flag --theme requires a theme name")
    args.splice(themeIndex, 2)
  }
  if (args.some(arg => arg.startsWith("--"))) throw new Error("Unknown argument; use --help")
  const app = await Application.open(args, !frame)
  if (customTheme) { app.config.theme = customTheme; app.changed() }
  if (frame) {
    app.config.lsp.enabled = false // A deterministic snapshot must never start language-server processes.
    const setup = await testRender(() => <Workbench app={app} />, { width, height })
    try {
      await setup.renderOnce()
      for (let attempt = 0; attempt < 100 && !app.highlights.has(app.activeId); attempt++) await Bun.sleep(10)
      await setup.renderOnce()
      console.log(setup.captureCharFrame())
    } finally { setup.renderer.destroy(); await app.close() }
    return
  }
  let shuttingDown = false
  const shutdown = async () => {
    if (shuttingDown) return
    shuttingDown = true
    try { await app.persist() } finally { await app.close(); process.off("SIGTERM", terminate); process.off("SIGINT", interrupt) }
  }
  let destroy: (() => void) | undefined
  const terminate = () => { void shutdown().finally(() => destroy?.()) }
  const interrupt = () => { void app.dispatch("quit") }
  process.on("SIGTERM", terminate); process.on("SIGINT", interrupt)
  const renderer = await createCliRenderer({ exitOnCtrlC: false, useMouse: app.config.mouse, targetFps: 30, onDestroy: () => { void shutdown() } })
  destroy = () => renderer.destroy()
  await render(() => {
    return <Workbench app={app} live onQuit={async () => {
      try { await shutdown() } catch (error) { app.report(error) }
      destroy?.()
    }} />
  }, renderer)
}

if (import.meta.main) {
  try { await run(process.argv.slice(2)) } catch (error) { console.error(error instanceof Error ? error.message : error); process.exitCode = 1 }
}
