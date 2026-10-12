import { readdir, stat, lstat, realpath, rename, unlink, mkdir, readFile, open } from "node:fs/promises"
import { basename, dirname, join, relative, resolve, isAbsolute } from "node:path"
import { homedir } from "node:os"
import { stringify } from "smol-toml"
import { readToml } from "./config"

export async function atomicWrite(path: string, text: string): Promise<void> {
  const target = await realpath(path).catch(() => resolve(path))
  const temporary = join(dirname(target), `.${basename(target)}.daisy-${crypto.randomUUID()}`)
  const mode = await stat(target).then(info => info.mode & 0o777).catch(() => 0o644)
  try {
    const file = await open(temporary, "wx", mode)
    try { await file.writeFile(text); await file.sync() } finally { await file.close() }
    await rename(temporary, target)
  } finally { await unlink(temporary).catch(error => { if (error.code !== "ENOENT") throw error }) }
}

export async function readText(path: string): Promise<string> {
  const info = await stat(path)
  if (!info.isFile()) throw new Error("Not a regular file")
  if (info.size > 32 * 1024 * 1024) throw new Error("File exceeds the current 32 MiB editing limit")
  const bytes = await readFile(path)
  if (bytes.includes(0)) throw new Error("Binary files cannot be edited as text")
  return new TextDecoder("utf-8", { fatal: true }).decode(bytes)
}

export type TreeRow = { path: string; name: string; directory: boolean; depth: number; expanded: boolean }
export class FileTree {
  rows: TreeRow[] = []
  selected = 0
  expanded = new Set<string>()
  constructor(public root: string, public showHidden = false) { this.expanded.add(root) }

  async refresh(): Promise<void> {
    const selectedPath = this.rows[this.selected]?.path
    const rows: TreeRow[] = [{ path: this.root, name: basename(this.root), directory: true, depth: 0, expanded: true }]
    const visit = async (directory: string, depth: number): Promise<void> => {
      const entries = await readdir(directory, { withFileTypes: true })
      entries.sort((left, right) => Number(right.isDirectory()) - Number(left.isDirectory()) || left.name.localeCompare(right.name))
      for (const entry of entries) {
        if ([".git", "node_modules", "target", "dist"].includes(entry.name) || (!this.showHidden && entry.name.startsWith("."))) continue
        const path = join(directory, entry.name)
        const expanded = this.expanded.has(path)
        rows.push({ path, name: entry.name, directory: entry.isDirectory(), depth, expanded })
        if (entry.isDirectory() && expanded) await visit(path, depth + 1)
      }
    }
    await visit(this.root, 1)
    this.rows = rows
    const index = rows.findIndex(row => row.path === selectedPath)
    this.selected = Math.max(0, Math.min(index >= 0 ? index : this.selected, rows.length - 1))
  }

  async toggle(row: TreeRow): Promise<void> {
    if (this.expanded.has(row.path)) this.expanded.delete(row.path)
    else this.expanded.add(row.path)
    await this.refresh()
  }

  async create(name: string, directory: boolean): Promise<string> {
    if (!name.trim() || name.includes("\0") || isAbsolute(name)) throw new Error("Invalid relative name")
    const selected = this.rows[this.selected]
    const parent = selected?.directory ? selected.path : dirname(selected?.path ?? this.root)
    const target = resolve(parent, name)
    const rel = relative(this.root, target)
    if (!rel || rel === ".." || rel.startsWith("../")) throw new Error("Name escapes the workspace")
    const resolvedParent = await realpath(parent)
    const canonicalRoot = await realpath(this.root)
    const parentRel = relative(canonicalRoot, resolvedParent)
    if (parentRel === ".." || parentRel.startsWith("../")) throw new Error("Parent symlink escapes workspace")
    // Only existing parents are accepted, so an intermediate symlink cannot redirect creation.
    const targetParent = await realpath(dirname(target))
    const targetRel = relative(canonicalRoot, targetParent)
    if (targetRel === ".." || targetRel.startsWith("../")) throw new Error("Parent escapes workspace")
    if (directory) await mkdir(target)
    else { const file = await open(target, "wx"); await file.close() }
    this.expanded.add(parent)
    await this.refresh()
    this.selected = Math.max(0, this.rows.findIndex(row => row.path === target))
    return target
  }
}

export async function discoverFiles(root: string, showHidden = false): Promise<string[]> {
  const ignored = new Set([".git", "node_modules", "target", "dist"])
  const files: string[] = []
  const visit = async (directory: string): Promise<void> => {
    for (const entry of await readdir(directory, { withFileTypes: true })) {
      if (ignored.has(entry.name) || (!showHidden && entry.name.startsWith("."))) continue
      const path = join(directory, entry.name)
      if (entry.isDirectory()) await visit(path)
      else if (entry.isFile()) files.push(relative(root, path))
      if (files.length >= 100000) throw new Error("Finder exceeds 100,000 files")
    }
  }
  await visit(root)
  return files.sort()
}

export function sessionName(workspace: string): string {
  let hash = 0xcbf29ce484222325n
  for (const byte of new TextEncoder().encode(workspace)) hash = BigInt.asUintN(64, (hash ^ BigInt(byte)) * 0x100000001b3n)
  return hash.toString(16).padStart(16, "0") + ".toml"
}
async function sessionPath(workspace: string): Promise<string> {
  const canonical = await realpath(workspace)
  if (await lstat(join(canonical, ".daisy")).then(info => info.isDirectory()).catch(() => false)) return join(canonical, ".daisy", "session.toml")
  if (await lstat(join(canonical, ".oride")).then(info => info.isDirectory()).catch(() => false)) return join(canonical, ".oride", "session.toml")
  return join(process.env.XDG_CACHE_HOME ?? join(homedir(), ".cache"), "daisy", "sessions", sessionName(canonical))
}
export type SplitSession = {
  orientation: "vertical" | "horizontal"
  secondary_file?: string
  ratio_percent: number
}

export type Session = {
  workspace: string
  files: string[]
  active_index: number
  scroll_y?: number
  show_tree: boolean
  show_scm: boolean
  tree_width: number
  split?: SplitSession
}

export async function loadSession(workspace: string): Promise<Partial<Session>> {
  const source = await readToml(await sessionPath(workspace)).catch(() => undefined)
  if (!source) return {}
  const rawSplit = typeof source.split === "object" && source.split !== null ? source.split as Record<string, unknown> : undefined
  const split: SplitSession | undefined = rawSplit ? {
    orientation: rawSplit.orientation === "horizontal" ? "horizontal" : "vertical",
    secondary_file: typeof rawSplit.secondary_file === "string" ? rawSplit.secondary_file : undefined,
    ratio_percent: typeof rawSplit.ratio_percent === "number" ? Math.min(85, Math.max(15, rawSplit.ratio_percent)) : 50,
  } : undefined
  return {
    files: Array.isArray(source.files) ? source.files.filter((file): file is string => typeof file === "string") : [],
    active_index: typeof source.active_index === "number" ? source.active_index : 0,
    scroll_y: typeof source.scroll_y === "number" ? Math.max(0, source.scroll_y) : 0,
    show_tree: typeof source.show_tree === "boolean" ? source.show_tree : true,
    show_scm: typeof source.show_scm === "boolean" ? source.show_scm : false,
    tree_width: typeof source.tree_width === "number" ? source.tree_width : 28,
    split,
  }
}
export async function saveSession(session: Session): Promise<void> {
  const path = await sessionPath(session.workspace)
  await mkdir(dirname(path), { recursive: true })
  await atomicWrite(path, stringify(session))
}
