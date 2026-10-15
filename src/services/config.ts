import { homedir } from "node:os"
import { join } from "node:path"
import { parse } from "smol-toml"
import { actionIds, defaultBindings, type ActionId } from "../core/keymap"

export type Language = { id: string; extensions: string[]; filenames: string[]; line_comment: string; lsp_command: string[]; completion_words: string[] }
export type Config = {
  theme: string; locale: string; mouse: boolean; soft_wrap: boolean; show_line_numbers: boolean; icons: boolean
  editor: { tab_size: number; insert_spaces: boolean; format_on_save: boolean; use_editorconfig: boolean; modal_mode: boolean; completion_auto: boolean; completion_min_chars: number; auto_pairs: boolean }
  tree: { width: number; show_hidden: boolean; git_status: boolean }
  terminal: { shell: string; default_height: number }
  markdown: { terminal_images: boolean }
  lsp: { enabled: boolean; timeout_ms: number; servers: Record<string, string[]> }
  debug: { enabled: boolean; adapters: Record<string, string[]>; launches: Record<string, { adapter?: string[]; language: string; program: string; args: string[]; cwd?: string }> }
  ui: Record<string, string>; syntax: Record<string, string>; tokens: Record<string, string>
  keys: Record<string, ActionId>; languages: Language[]
}
export const userConfigDir = join(process.env.XDG_CONFIG_HOME ?? join(homedir(), ".config"), "daisy")
export const legacyUserConfigDir = join(process.env.XDG_CONFIG_HOME ?? join(homedir(), ".config"), "oride")
export const defaultConfig = (): Config => ({
  theme: "default", locale: "pt-BR", mouse: true, soft_wrap: true, show_line_numbers: true, icons: false,
  editor: { tab_size: 4, insert_spaces: true, format_on_save: false, use_editorconfig: true, modal_mode: false, completion_auto: true, completion_min_chars: 2, auto_pairs: true },
  markdown: { terminal_images: false }, tree: { width: 28, show_hidden: false, git_status: true }, terminal: { shell: process.env.SHELL ?? "/bin/sh", default_height: 10 },
  debug: { enabled: true, adapters: { rust: ["rust-analyzer", "--dap"], c: ["clangd", "--dap"], python: ["debugpy-adapter"], typescript: ["typescript-debug-adapter"], go: ["dlv", "dap", "--listen=:4711"] }, launches: {} },
  lsp: { enabled: true, timeout_ms: 10000, servers: { rust: ["rust-analyzer"], c: ["clangd"], cpp: ["clangd"], bash: ["bash-language-server", "start"], ori: ["ori-lsp"], python: ["pylsp"], typescript: ["typescript-language-server", "--stdio"], javascript: ["typescript-language-server", "--stdio"], go: ["gopls"] } },
  ui: {}, syntax: {}, tokens: {}, keys: { ...defaultBindings }, languages: [],
})
export function record(value: unknown): Record<string, unknown> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) throw new Error("Expected TOML table")
  return value as Record<string, unknown>
}
function merge(target: Record<string, unknown>, source: Record<string, unknown>): void {
  for (const [key, value] of Object.entries(source)) {
    if (key === "__proto__" || key === "constructor" || key === "prototype") throw new Error("Invalid configuration key")
    const previous = target[key]
    if (typeof previous === "object" && previous !== null && !Array.isArray(previous) && typeof value === "object" && value !== null && !Array.isArray(value)) merge(record(previous), record(value))
    else if (previous !== undefined && typeof previous !== typeof value) throw new Error(`Configuration type mismatch: ${key}`)
    else target[key] = value
  }
}
export async function readToml(path: string): Promise<Record<string, unknown> | undefined> {
  const file = Bun.file(path)
  if (!await file.exists()) return undefined
  return record(parse(await file.text()))
}
export async function loadConfig(workspace: string): Promise<Config> {
  const config = defaultConfig()
  for (const path of [
    join(legacyUserConfigDir, "config.toml"),
    join(userConfigDir, "config.toml"),
    join(workspace, ".oride", "config.toml"),
    join(workspace, ".daisy", "config.toml"),
  ]) {
    const source = await readToml(path)
    if (source) merge(config as unknown as Record<string, unknown>, source)
  }
  if (process.env.DAISY_THEME) config.theme = process.env.DAISY_THEME
  for (const [chord, action] of Object.entries(config.keys)) if (!actionIds.includes(action)) throw new Error(`Unknown action ${action} for ${chord}`)
  for (const command of Object.values(config.lsp.servers)) if (!Array.isArray(command) || !command.every(argument => typeof argument === "string")) throw new Error("LSP commands must be arrays of strings")
  if (typeof config.debug !== "object" || config.debug === null) throw new Error("Debug configuration must be a table")
  if (typeof config.debug.enabled !== "boolean") throw new Error("Debug enabled flag must be boolean")
  for (const command of Object.values(config.debug.adapters ?? {})) if (!Array.isArray(command) || !command.every(argument => typeof argument === "string")) throw new Error("Debug adapters must be arrays of strings")
if (typeof config.debug.launches !== "object" || config.debug.launches === null) throw new Error("Debug launches must be a table")
  for (const launch of Object.values(config.debug.launches ?? {})) {
    const entry = record(launch)
    if (typeof entry.language !== "string") throw new Error("Launch language is required")
    if (typeof entry.program !== "string" || !entry.program.trim()) throw new Error("Launch program is required")
    if (entry.args !== undefined && (!Array.isArray(entry.args) || !entry.args.every(argument => typeof argument === "string"))) throw new Error("Launch args must be string arrays")
    if (entry.cwd !== undefined && typeof entry.cwd !== "string") throw new Error("Launch cwd must be string")
    if (entry.adapter !== undefined && (!Array.isArray(entry.adapter) || !entry.adapter.every(argument => typeof argument === "string"))) throw new Error("Launch adapter must be string array")
  }
  config.languages = config.languages.map(raw => {
    const language = record(raw)
    if (typeof language.id !== "string") throw new Error("Language requires id")
    const strings = (key: string): string[] => {
      const values = language[key] ?? []
      if (!Array.isArray(values) || !values.every(value => typeof value === "string")) throw new Error(`Invalid language ${key}`)
      return values
    }
    return { id: language.id, extensions: strings("extensions"), filenames: strings("filenames"), line_comment: typeof language.line_comment === "string" ? language.line_comment : "//", lsp_command: strings("lsp_command"), completion_words: strings("completion_words") }
  })
  config.editor.tab_size = Math.max(1, Math.min(16, config.editor.tab_size))
  config.tree.width = Math.max(8, config.tree.width)
  config.terminal.default_height = Math.max(3, config.terminal.default_height)
  config.lsp.timeout_ms = Math.max(500, config.lsp.timeout_ms)
  return config
}
