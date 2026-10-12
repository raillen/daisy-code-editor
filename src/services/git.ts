import { spawn } from "node:child_process"
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
