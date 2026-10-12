import { readdir } from "node:fs/promises"
import { join, dirname } from "node:path"
import { readToml, record, userConfigDir } from "./config"
import { execute } from "./git"
export type PluginCommand = { id: string; label: string; executable: string; args: string[] }
export type Plugin = { name: string; commands: PluginCommand[]; hooks: { on_open?: PluginCommand; on_save?: PluginCommand } }
export type PluginContext = { file: string; workspace: string }
function parseCommand(raw: unknown, id?: string): PluginCommand {
  const source = record(raw)
  if (typeof source.executable !== "string" || !source.executable.trim()) throw new Error("Plugin command needs executable")
  const args = source.args ?? []
  if (!Array.isArray(args) || !args.every(value => typeof value === "string")) throw new Error("Plugin args must be strings")
  const identifier = id ?? source.id
  if (typeof identifier !== "string" || !identifier.trim()) throw new Error("Plugin command needs id")
  return { id: identifier, label: typeof source.label === "string" ? source.label : identifier, executable: source.executable, args }
}
export function parsePlugin(raw: Record<string, unknown>): Plugin {
  const header = record(raw.plugin)
  if (typeof header.name !== "string" || !header.name.trim()) throw new Error("Plugin requires name")
  if (raw.commands !== undefined && !Array.isArray(raw.commands)) throw new Error("Plugin commands must be an array")
  const commands = (Array.isArray(raw.commands) ? raw.commands : []).map(command => parseCommand(command))
  if (new Set(commands.map(command => command.id)).size !== commands.length) throw new Error("Duplicate plugin command")
  const source = raw.hooks ? record(raw.hooks) : {}
  const hooks: Plugin["hooks"] = {}
  for (const id of ["on_open", "on_save"] as const) if (source[id]) hooks[id] = parseCommand(source[id], id)
  return { name: header.name, commands, hooks }
}
export async function discoverPlugins(workspace: string): Promise<Plugin[]> {
  const found = new Map<string, Plugin>()
  for (const root of [join(userConfigDir, "plugins"), join(workspace, ".daisy", "plugins"), join(workspace, ".daisy"), join(workspace, ".oride", "plugins"), join(workspace, ".oride")]) {
    const directories = await readdir(root, { withFileTypes: true }).catch(error => { if (error.code === "ENOENT") return []; throw error })
    for (const directory of directories.filter(directory => directory.isDirectory())) {
      const source = await readToml(join(root, directory.name, "plugin.toml")).catch(() => undefined)
      if (!source) continue
      try { const plugin = parsePlugin(source); if (!found.has(plugin.name)) found.set(plugin.name, plugin) } catch { /* An invalid extension cannot prevent editing. */ }
    }
  }
  return [...found.values()].sort((left, right) => left.name.localeCompare(right.name))
}
export async function runPlugin(command: PluginCommand, context: PluginContext): Promise<string> {
  const variables: Record<string, string> = { FILE: context.file, DIR: dirname(context.file), WORKSPACE: context.workspace }
  const args = command.args.map(argument => argument.replace(/\$(WORKSPACE|FILE|DIR)\b/g, (_match, key: string) => variables[key] ?? ""))
  return execute(command.executable, args, context.workspace, 10000)
}
