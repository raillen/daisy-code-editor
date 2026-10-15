import { spawn } from "node:child_process"
import { unlink, writeFile } from "node:fs/promises"
import { join } from "node:path"
import { tmpdir } from "node:os"
export type GitEntry = { path: string; index: string; worktree: string; badge: string }
export type GitStatus = { branch: string; ahead: number; behind: number; entries: GitEntry[] }

export async function execute(program: string, args: readonly string[], cwd: string, timeout = 10000): Promise<string> {
  return new Promise((resolve, reject) => {
    const child = spawn(program, [...args], { cwd, stdio: ["ignore", "pipe", "pipe"], env: { ...process.env, GIT_TERMINAL_PROMPT: "0" } })
    let output = "", errors = "", settled = false
    const fail = (error: Error) => { if (!settled) { settled = true; clearTimeout(timer); reject(error) } }
    const timer = setTimeout(() => { child.kill("SIGKILL"); fail(new Error(`${program} timed out`)) }, timeout)
    child.stdout.on("data", (chunk: Buffer) => {
      output += chunk.toString()
      if (output.length > 8 * 1024 * 1024) { child.kill("SIGKILL"); fail(new Error("Process output exceeds 8 MiB")) }
    })
    child.stderr.on("data", (chunk: Buffer) => { errors = (errors + chunk.toString()).slice(-65536) })
    child.on("error", fail)
    child.on("close", code => {
      clearTimeout(timer)
      if (settled) return
      settled = true
      if (code !== 0) reject(new Error(errors.trim() || `${program} exited ${code}`))
      else resolve(output)
    })
  })
}

export function parsePorcelain(output: string): GitEntry[] {
  const fields = output.split("\0"), entries: GitEntry[] = []
  for (let index = 0; index < fields.length; index++) {
    const field = fields[index]
    if (!field || field.length < 4) continue
    const staged = field[0] ?? " ", worktree = field[1] ?? " "
    const path = field.slice(3)
    if (staged === "R" || staged === "C" || worktree === "R" || worktree === "C") index++
    const conflict = staged === "U" || worktree === "U" || (staged === worktree && (staged === "A" || staged === "D"))
    entries.push({ path, index: staged, worktree, badge: conflict ? "U" : worktree !== " " ? worktree : staged })
  }
  return entries
}

export class Git {
  constructor(readonly root: string) {}
  async status(): Promise<GitStatus> {
    if (!await execute("git", ["rev-parse", "--is-inside-work-tree"], this.root).catch(() => "")) return { branch: "", ahead: 0, behind: 0, entries: [] }
    const [branch, porcelain, prefix] = await Promise.all([
      execute("git", ["branch", "--show-current"], this.root),
      execute("git", ["status", "--porcelain=v1", "-z", "--untracked-files=all", "--", "."], this.root),
      execute("git", ["rev-parse", "--show-prefix"], this.root),
    ])
    const distances = await execute("git", ["rev-list", "--left-right", "--count", "HEAD...@{upstream}"], this.root).catch(() => "0\t0")
    const [ahead = 0, behind = 0] = distances.trim().split(/\s+/).map(Number)
    return { branch: branch.trim() || "detached", ahead, behind, entries: parsePorcelain(porcelain).map(entry => ({ ...entry, path: entry.path.startsWith(prefix.trim()) ? entry.path.slice(prefix.trim().length) : entry.path })) }
  }
  stage(path: string): Promise<string> { return execute("git", ["add", "--", path], this.root) }
  unstage(path: string): Promise<string> { return execute("git", ["restore", "--staged", "--", path], this.root) }
  commit(message: string): Promise<string> {
    if (!message.trim()) throw new Error("Commit message is empty")
    return execute("git", ["commit", "-m", message], this.root)
  }
  async diff(path?: string): Promise<string> {
    const args = path ? ["--", path] : []
    const unstaged = await execute("git", ["diff", "--no-ext-diff", ...args], this.root)
    const staged = await execute("git", ["diff", "--cached", "--no-ext-diff", ...args], this.root)
    return unstaged + staged || "No tracked changes"
  }
  sync(operation: "pull" | "push"): Promise<string> { return execute("git", [operation], this.root, 60000) }
  async blame(path: string, line: number): Promise<string> {
    try {
      const output = await execute("git", ["blame", "-L", `${line},${line}`, "--", path], this.root)
      return output.trim()
    } catch (error) {
      return `Not committed: ${error instanceof Error ? error.message : String(error)}`
    }
  }
  stash(message = "Daisy stash"): Promise<string> { return execute("git", ["stash", "push", "-m", message], this.root, 60000) }
  stashPop(): Promise<string> { return execute("git", ["stash", "pop"], this.root, 60000) }
  async diffUnstaged(path: string): Promise<string> {
    try { return await execute("git", ["diff", "--no-color", "-U3", "--no-ext-diff", "--", path], this.root) } catch { return "" }
  }
  async diffStaged(path: string): Promise<string> {
    try { return await execute("git", ["diff", "--cached", "--no-color", "-U3", "--no-ext-diff", "--", path], this.root) } catch { return "" }
  }
  private async applyPatch(patch: string, reverse = false): Promise<string> {
    const patchFile = join(tmpdir(), `daisy-hunk-${process.pid}.patch`)
    try {
      await writeFile(patchFile, `${patch.endsWith("\n") ? patch : `${patch}\n`}`, "utf8")
      return await execute("git", ["apply", reverse ? "-R" : undefined, "--cached", "--whitespace=nowarn", patchFile].filter(Boolean) as string[], this.root, 30000)
    } finally {
      await unlink(patchFile).catch(() => undefined)
    }
  }
  async stageHunk(path: string, cursorLine: number): Promise<string> {
    const diff = path === null ? "" : await this.diffUnstaged(path)
    const patch = selectHunkPatch(diff, cursorLine)
    const output = await this.applyPatch(patch)
    return output.trim() || "Staged hunk"
  }
  async stageHunkAt(path: string, index: number): Promise<string> {
    const diff = await this.diffUnstaged(path)
    const { meta, hunks } = parseHunks(diff)
    const hunk = hunks[index]
    if (!hunk) throw new Error("Hunk index out of range")
    const output = await this.applyPatch(hunkPatch(meta, hunk))
    return output.trim() || `Staged hunk ${index + 1}`
  }
  async unstageHunkAt(path: string, index: number): Promise<string> {
    const diff = await this.diffStaged(path)
    const { meta, hunks } = parseHunks(diff)
    const hunk = hunks[index]
    if (!hunk) throw new Error("Hunk index out of range")
    const output = await this.applyPatch(hunkPatch(meta, hunk), true)
    return output.trim() || `Unstaged hunk ${index + 1}`
  }
  async hunks(path: string, staged: boolean): Promise<GitHunk[]> {
    const diff = staged ? await this.diffStaged(path) : await this.diffUnstaged(path)
    return parseHunks(diff).hunks
  }
  async unstageHunk(path: string, cursorLine: number): Promise<string> {
    const diff = await this.diffStaged(path)
    const patch = selectHunkPatch(diff, cursorLine)
    const output = await this.applyPatch(patch, true)
    return output.trim() || "Unstaged hunk"
  }
  stashList(): Promise<string> { return execute("git", ["stash", "list"], this.root, 10000) }
  async diffGutter(path: string): Promise<Map<number, GutterKind>> {
    try {
      const output = await execute("git", ["diff", "HEAD", "-U0", "--no-ext-diff", "--", path], this.root)
      return parseDiffHunks(output)
    } catch {
      try {
        const output = await execute("git", ["diff", "-U0", "--no-ext-diff", "--", path], this.root)
        return parseDiffHunks(output)
      } catch {
        return new Map()
      }
    }
  }
}

export type GitHunk = { header: string; body: string[]; newStart: number; newCount: number }

export function parseHunks(diffText: string): { meta: string[]; hunks: GitHunk[] } {
  const lines = diffText.split("\n")
  const meta: string[] = []
  const hunks: GitHunk[] = []
  let current: string[] | null = null
  let header = ""
  let newStart = 0, newCount = 0
  for (const line of lines) {
    if (line.startsWith("@@ ")) {
      if (current !== null) hunks.push({ header, body: current, newStart, newCount })
      const match = line.match(/^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@/)
      newStart = match ? Number(match[1]) : 0
      newCount = match && match[2] !== undefined ? Number(match[2]) : 1
      header = line
      current = []
    } else if (current !== null) current.push(line)
    else meta.push(line)
  }
  if (current !== null) hunks.push({ header, body: current, newStart, newCount })
  return { meta, hunks }
}

export function hunkPatch(meta: string[], hunk: GitHunk): string {
  const body = hunk.body.filter((line, index) => !(index === hunk.body.length - 1 && line.trim() === ""))
  return [...meta, hunk.header, ...body].join("\n")
}

export function hunkSummary(hunk: GitHunk): string {
  const added = hunk.body.filter(line => line.startsWith("+") && !line.startsWith("+++")).length
  const removed = hunk.body.filter(line => line.startsWith("-") && !line.startsWith("---")).length
  const preview = hunk.body.find(line => (line.startsWith("+") || line.startsWith("-")) && !line.startsWith("+++") && !line.startsWith("---"))?.slice(1, 81) ?? ""
  return `${hunk.header} (+${added}/-${removed})${preview ? ` ${preview}` : ""}`
}

export function selectHunkPatch(diffText: string, cursorLine: number): string {
  const { meta, hunks } = parseHunks(diffText)
  if (!hunks.length) throw new Error("No changed hunks")
  let best = hunks[0]!, bestDistance = Infinity
  for (const hunk of hunks) {
    const start = hunk.newStart
    const end = start + Math.max(0, hunk.newCount - 1)
    let distance = 0
    if (cursorLine < start) distance = start - cursorLine
    else if (cursorLine > end) distance = cursorLine - end
    else distance = 0
    if (distance < bestDistance) { best = hunk; bestDistance = distance }
  }
  return hunkPatch(meta, best)
}

export type GutterKind = "added" | "modified" | "deleted"

export function parseDiffHunks(diffText: string): Map<number, GutterKind> {
  const map = new Map<number, GutterKind>()
  const regex = /^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/gm
  let match: RegExpExecArray | null
  while ((match = regex.exec(diffText)) !== null) {
    const oldCount = match[2] !== undefined ? parseInt(match[2], 10) : 1
    const newStart = parseInt(match[3]!, 10)
    const newCount = match[4] !== undefined ? parseInt(match[4], 10) : 1

    if (oldCount === 0 && newCount > 0) {
      for (let i = 0; i < newCount; i++) map.set(newStart + i - 1, "added")
    } else if (newCount === 0) {
      map.set(Math.max(0, newStart - 1), "deleted")
    } else {
      for (let i = 0; i < newCount; i++) map.set(newStart + i - 1, "modified")
    }
  }
  return map
}
