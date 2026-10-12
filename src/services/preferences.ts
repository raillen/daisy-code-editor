import { readdir, mkdir } from "node:fs/promises"
import { join, basename, dirname, relative } from "node:path"
import { stringify } from "smol-toml"
import { builtInThemes, builtInLocales } from "./catalogs"
import { readToml, record, userConfigDir, type Config } from "./config"
import { atomicWrite } from "./files"
import type { ActionId } from "../core/keymap"

export type ThemeDefinition = { name: string; ui: Record<string, string>; syntax: Record<string, string>; tokens?: Record<string, string> }
export type LocaleDefinition = Record<string, unknown>
export type Preferences = { themes: Map<string, ThemeDefinition>; locales: Map<string, LocaleDefinition> }
const normalize = (name: string) => name.toLowerCase().replaceAll(" ", "-")
const colorTable = (source: unknown): Record<string, string> => Object.fromEntries(Object.entries(source ? record(source) : {}).filter((pair): pair is [string, string] => typeof pair[1] === "string"))
export async function loadPreferences(workspace: string): Promise<Preferences> {
  const themes = new Map<string, ThemeDefinition>(Object.entries(builtInThemes))
  themes.set("default", themes.get("default-dark") ?? { name: "Default Dark", ui: {}, syntax: {} })
  themes.set("no-color", { name: "No Color", ui: {}, syntax: {} })
  const locales = new Map<string, LocaleDefinition>(Object.entries(builtInLocales))
  for (const root of [userConfigDir, join(workspace, ".daisy"), join(workspace, ".oride")]) {
    for (const kind of ["themes", "locales"] as const) {
      const directory = join(root, kind)
      const names = await readdir(directory).catch(error => { if (error.code === "ENOENT") return []; throw error })
      for (const name of names.filter(name => name.endsWith(".toml"))) {
        const source = await readToml(join(directory, name))
        if (!source) continue
        if (kind === "themes" && typeof source.name === "string") themes.set(normalize(source.name), { name: source.name, ui: colorTable(source.ui), syntax: colorTable(source.syntax), tokens: colorTable(source.tokens) })
        else if (kind === "locales" && typeof source.id === "string") locales.set(source.id, source)
      }
    }
  }
  return { themes, locales }
}
export function selectedTheme(config: Config, preferences: Preferences): ThemeDefinition {
  return preferences.themes.get(normalize(config.theme)) ?? preferences.themes.get("default") ?? { name: "Default Dark", ui: {}, syntax: {} }
}
export function labelOf(action: ActionId, fallback: string, config: Config, preferences: Preferences): string {
  const aliases: Partial<Record<ActionId, string>> = { open_file_fuzzy: "open_file", project_find: "find_in_project", select_theme: "color_theme", select_locale: "display_language" }
  const key = aliases[action] ?? action
  const locale: LocaleDefinition = preferences.locales.get(config.locale) ?? builtInLocales["en-US"]
  for (const section of ["file_menu", "edit_menu", "view_menu", "settings_menu"]) {
    const table = locale[section]
    if (table && typeof table === "object" && !Array.isArray(table)) {
      const label = record(table)[key]
      if (typeof label === "string" && label) return label
    }
  }
  return fallback
}
export async function savePreference(workspace: string, key: "theme" | "locale", value: string): Promise<void> {
  const legacyDir = join(workspace, ".oride")
  const directory = await Bun.file(join(legacyDir, "config.toml")).exists() ? legacyDir : join(workspace, ".daisy")
  await mkdir(directory, { recursive: true })
  const path = join(directory, "config.toml")
  const source = await readToml(path) ?? {}
  source[key] = value
  await atomicWrite(path, stringify(source as Parameters<typeof stringify>[0]))
}
export async function editorIndent(path: string, fallback: { tabSize: number; insertSpaces: boolean }): Promise<{ tabSize: number; insertSpaces: boolean }> {
  const layers: { directory: string; text: string }[] = []
  let directory = dirname(path)
  for (let depth = 0; depth < 32; depth++) {
    const file = Bun.file(join(directory, ".editorconfig"))
    if (await file.exists()) {
      const text = await file.text()
      layers.push({ directory, text })
      if (/^\s*root\s*=\s*true\s*$/im.test(text)) break
    }
    const parent = dirname(directory)
    if (parent === directory) break
    directory = parent
  }
  const result = { ...fallback }
  for (const layer of layers.toReversed()) {
    let matches = false
    for (const raw of layer.text.split(/\r?\n/)) {
      const line = raw.trim()
      if (!line || /^[#;]/.test(line)) continue
      if (line.startsWith("[") && line.endsWith("]")) {
        const pattern = line.slice(1, -1)
        matches = new Bun.Glob(pattern).match(pattern.includes("/") ? relative(layer.directory, path) : basename(path))
      } else if (matches) {
        const [key, value] = line.split("=").map(part => part.trim())
        if (key === "indent_style" && (value === "space" || value === "tab")) result.insertSpaces = value === "space"
        else if ((key === "indent_size" || key === "tab_width" && !result.insertSpaces) && value && /^\d+$/.test(value)) result.tabSize = Math.max(1, Math.min(16, Number(value)))
      }
    }
  }
  return result
}
