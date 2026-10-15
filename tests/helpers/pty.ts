// Harness de smoke para o binário real: spawna o CLI dentro de um PTY de
// verdade e emula a saída com @xterm/headless (o mesmo emulador do painel de
// terminal). O testRender exercita o app in-process; aqui o caminho é o
// completo — stdin real, parser de teclas do renderer, alt-screen, sinais de
// tty — que é onde falhas de ambiente (tcgetpgrp, resize) aparecem.
//
// Limites deliberados: sem asserção de SIGWINCH. No Bun 1.4.2 o
// Bun.Terminal.resize() não entrega o sinal ao filho (verificado com um
// processo sonda), então redimensionar o emulador menor que o app só gera
// frame corrompido. Teclas com ESC (escape) precisam de folga nas chamadas
// seguintes: ESC + byte colados viram alt+tecla no parser.
import xterm from "@xterm/headless"
import { fileURLToPath } from "node:url"
import { join } from "node:path"

// O processo filho roda com cwd no repo (resolução de tsconfig/JSX do Bun) e
// recebe o workspace como argumento — os testes mantêm o workspace em /tmp.
const repoRoot = fileURLToPath(new URL("../..", import.meta.url))
const cli = join(repoRoot, "src", "cli.tsx")

export type KeyPress = {
  name: "enter" | "escape" | "tab" | "backspace" | "space" | "up" | "down" | "left" | "right" | "home" | "end" | "pageup" | "pagedown" | (string & {})
  ctrl?: boolean
  alt?: boolean
  shift?: boolean
}

function sequence(press: KeyPress): string {
  if (press.ctrl) {
    if (press.name === "space") return "\x00"
    if (press.name === "[") return "\x1b"
    if (press.name === "\\") return "\x1c"
    if (press.name === "]") return "\x1d"
    if (press.name === "^") return "\x1e"
    if (press.name === "_") return "\x1f"
    if (press.name.length === 1) {
      const code = press.name.toLowerCase().charCodeAt(0)
      if (code >= 97 && code <= 122) return String.fromCharCode(code - 96)
    }
    throw new Error(`Unsupported ctrl chord: ctrl+${press.name}`)
  }
  const alt = press.alt ? "\x1b" : ""
  switch (press.name) {
    case "enter": return "\r"
    case "escape": return "\x1b"
    case "tab": return press.shift ? "\x1b[Z" : "\t"
    case "backspace": return "\x7f"
    case "space": return " "
    case "up": return `${alt}\x1b[A`
    case "down": return `${alt}\x1b[B`
    case "right": return `${alt}\x1b[C`
    case "left": return `${alt}\x1b[D`
    case "home": return `${alt}\x1b[H`
    case "end": return `${alt}\x1b[F`
    case "pageup": return `${alt}\x1b[5~`
    case "pagedown": return `${alt}\x1b[6~`
    default: return alt + press.name
  }
}

function parseChord(chord: string): KeyPress {
  const parts = chord.split("+")
  const name = parts.at(-1) ?? chord
  return { name, ctrl: parts.includes("ctrl"), alt: parts.includes("alt"), shift: parts.includes("shift") }
}

export type PtySession = {
  readonly pid: number
  /** Type text key by key; "\n" presses Enter. */
  type(text: string, perKeyMs?: number): Promise<void>
  /** Press a named key ("ctrl+q", "enter") or a structured KeyPress. */
  press(press: KeyPress | string): void
  /** Wait until the rendered frame matches the pattern. */
  waitFor(pattern: RegExp | string, timeoutMs?: number): Promise<void>
  /** Wait until a frame predicate holds (for layout assertions). */
  waitUntil(predicate: (frame: string) => boolean, timeoutMs?: number): Promise<void>
  /** Current visible frame, right-trimmed per line. */
  frame(): string
  /** Wait for process exit; rejects when it outlives the timeout. */
  waitExit(timeoutMs?: number): Promise<number>
  /** Kill the process group and release terminal resources. Idempotent. */
  close(): Promise<void>
}

export async function spawnDaisy(options: { workspace: string; args?: string[]; env?: Record<string, string>; cols?: number; rows?: number }): Promise<PtySession> {
  const cols = options.cols ?? 100
  const rows = options.rows ?? 25
  const emulator = new xterm.Terminal({ cols, rows, scrollback: 0, allowProposedApi: true })
  let closed = false
  const terminal = new Bun.Terminal({ cols, rows, data: (_terminal, bytes) => {
    if (!closed) emulator.write(Buffer.from(bytes), () => {})
  } })
  const child = Bun.spawn([process.execPath, cli, options.workspace, ...(options.args ?? [])], {
    cwd: repoRoot,
    terminal,
    detached: true,
    env: { ...process.env, TERM: "xterm-256color", ...options.env },
  })
  let exitCode: number | undefined
  const exited = child.exited.then(code => { exitCode = code; return code }, () => { exitCode = -1; return -1 })

  const frame = (): string => {
    const buffer = emulator.buffer.active
    const lines: string[] = []
    for (let row = 0; row < rows; row++) lines.push(buffer.getLine(buffer.baseY + row)?.translateToString(true) ?? "")
    return lines.join("\n")
  }
  const waitFor = async (pattern: RegExp | string, timeoutMs = 10000): Promise<void> => {
    const regex = typeof pattern === "string" ? new RegExp(pattern.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")) : pattern
    const deadline = performance.now() + timeoutMs
    for (;;) {
      if (regex.test(frame())) return
      if (exitCode !== undefined) throw new Error(`CLI exited (${exitCode}) while waiting for ${regex}\n--- frame ---\n${frame()}`)
      if (performance.now() > deadline) throw new Error(`Timed out waiting for ${regex}\n--- frame ---\n${frame()}`)
      await Bun.sleep(25)
    }
  }
  const waitUntil = async (predicate: (frame: string) => boolean, timeoutMs = 10000): Promise<void> => {
    const deadline = performance.now() + timeoutMs
    for (;;) {
      if (predicate(frame())) return
      if (exitCode !== undefined) throw new Error(`CLI exited (${exitCode}) while waiting for frame condition\n--- frame ---\n${frame()}`)
      if (performance.now() > deadline) throw new Error(`Timed out waiting for frame condition\n--- frame ---\n${frame()}`)
      await Bun.sleep(25)
    }
  }
  const waitExit = async (timeoutMs = 15000): Promise<number> => {
    let timer: ReturnType<typeof setTimeout> | undefined
    const timeout = new Promise<never>((_, reject) => {
      timer = setTimeout(() => reject(new Error(`CLI did not exit within ${timeoutMs}ms\n--- frame ---\n${frame()}`)), timeoutMs)
    })
    try { return await Promise.race([exited, timeout]) } finally { clearTimeout(timer) }
  }
  const close = async (): Promise<void> => {
    if (closed) return
    closed = true
    if (exitCode === undefined) {
      try { process.kill(-child.pid, "SIGTERM") } catch {}
      const killer = setTimeout(() => { try { process.kill(-child.pid, "SIGKILL") } catch {} }, 500)
      await exited.catch(() => {})
      clearTimeout(killer)
    }
    terminal.close()
    emulator.dispose()
  }

  return {
    pid: child.pid,
    type: async (text, perKeyMs = 8) => {
      for (const char of text) { terminal.write(char === "\n" ? "\r" : char); await Bun.sleep(perKeyMs) }
    },
    press: press => terminal.write(sequence(typeof press === "string" ? parseChord(press) : press)),
    waitFor,
    waitUntil,
    frame,
    waitExit,
    close,
  }
}
