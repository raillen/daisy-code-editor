import { spawn, type ChildProcessWithoutNullStreams } from "node:child_process"

// Adaptador mínimo do Debug Adapter Protocol. Mantém apenas o subconjunto
// necessário para um scaffolding de debug (initialize, setBreakpoints,
// launch/configurationDone/continue, threads/stackTrace) e falha de forma
// explícita quando o adapter não existe.
export type DebugBreakpoint = { line: number; verified: boolean }
export type DebugSourceBreakpoint = { line: number; condition?: string }
export type DebugSessionStatus = "ready" | "running" | "stopped" | "exited" | "failed"

export class DebugAdapter {
  private process: ChildProcessWithoutNullStreams
  private buffer = ""
  private nextSeq = 1
  private pending = new Map<number, (result: unknown) => void>()
  private onChange: (status: DebugSessionStatus, message?: string) => void
  readonly breakpoints = new Map<string, DebugSourceBreakpoint[]>()
  status: DebugSessionStatus = "ready"
  adapterError: string | undefined

  constructor(readonly command: readonly string[], readonly workspace: string, onChange: (status: DebugSessionStatus, message?: string) => void) {
    this.onChange = onChange
    if (!command.length) throw new Error("Empty debug adapter command")
    this.process = spawn(command[0]!, command.slice(1), { cwd: workspace, stdio: ["pipe", "pipe", "pipe"] })
    this.process.stdout.on("data", (chunk: Buffer) => this.consume(chunk))
    this.process.stderr.on("data", (chunk: Buffer) => { this.adapterError = chunk.toString().trim().slice(-1000) })
    this.process.on("exit", code => { this.status = "exited"; this.onChange(this.status, `Adapter exited with code ${code}`) })
    this.process.on("error", error => { this.status = "failed"; this.onChange(this.status, String(error)) })
  }

  private consume(chunk: Buffer): void {
    this.buffer += chunk.toString()
    let headerEnd = this.buffer.indexOf("\r\n\r\n")
    while (headerEnd !== -1) {
      const header = this.buffer.slice(0, headerEnd)
      const length = /^Content-Length:\s*(\d+)/i.exec(header)?.[1]
      if (!length) { this.buffer = this.buffer.slice(headerEnd + 4); headerEnd = this.buffer.indexOf("\r\n\r\n"); continue }
      const size = Number(length)
      const bodyStart = headerEnd + 4
      if (this.buffer.length < bodyStart + size) return
      const body = JSON.parse(this.buffer.slice(bodyStart, bodyStart + size))
      if (typeof body.request_seq === "number") this.pending.get(body.request_seq)?.(body)
      else if (body.event === "stopped") { this.status = "stopped"; this.onChange(this.status, body.body?.reason) }
      else if (body.event === "continued") { this.status = "running"; this.onChange(this.status) }
      else if (body.event === "terminated") { this.status = "exited"; this.onChange(this.status) }
      this.buffer = this.buffer.slice(bodyStart + size)
      headerEnd = this.buffer.indexOf("\r\n\r\n")
    }
  }

  async request<T = unknown>(command: string, args?: unknown, timeout = 5000): Promise<T> {
    if (this.status === "failed" || this.status === "exited") throw new Error(this.adapterError ?? "Debug adapter unavailable")
    const seq = this.nextSeq++
    const payload = JSON.stringify({ seq, command, arguments: args, type: "request" })
    return new Promise<T>((resolve, reject) => {
      const timer = setTimeout(() => { this.pending.delete(seq); reject(new Error(`DAP ${command} timed out`)) }, timeout)
      this.pending.set(seq, result => { clearTimeout(timer); const response = result as { success?: boolean; message?: string; body?: T }; if (response.success === false) reject(new Error(response.message ?? `DAP ${command} failed`)); else resolve((response.body ?? result) as T) })
      this.process.stdin.write(`Content-Length: ${new TextEncoder().encode(payload).length}\r\n\r\n${payload}`)
    })
  }

  async initialize(): Promise<void> {
    await this.request("initialize", { clientID: "daisy-code-editor", supportsConfigurationDoneRequest: true, linesStartAt1: true, columnsStartAt1: true })
    await this.request("setExceptionBreakpoints", { filters: [] })
  }

  async setBreakpoints(path: string, breakpoints: readonly DebugSourceBreakpoint[]): Promise<DebugBreakpoint[]> {
    this.breakpoints.set(path, breakpoints.map(entry => ({ ...entry })))
    const response = await this.request<{ breakpoints?: DebugBreakpoint[] }>("setBreakpoints", {
      source: { path, name: path.split("/").pop() ?? path },
      breakpoints: breakpoints.map(entry => (entry.condition ? { line: entry.line, condition: entry.condition } : { line: entry.line })),
      sourceModified: false,
    })
    return response?.breakpoints ?? []
  }

  async launch(config: unknown): Promise<void> {
    await this.request("launch", config, 10000)
    await this.request("configurationDone")
    this.status = "running"
    this.onChange(this.status)
  }

  async continue(threadId: number): Promise<void> {
    await this.request("continue", { threadId })
    this.status = "running"
    this.onChange(this.status)
  }
  async threads(): Promise<Array<{ id: number; name: string }>> {
    const response = await this.request<{ threads?: Array<{ id: number; name: string }> }>("threads")
    return response?.threads ?? []
  }
  async stackTrace(threadId: number): Promise<Array<{ id?: number; name: string; line?: number; source?: { path?: string; name?: string } }>> {
    const response = await this.request<{ stackFrames?: Array<{ name: string; line?: number; source?: { path?: string; name?: string } }> }>("stackTrace", { threadId })
    return response?.stackFrames ?? []
  }

  async pause(): Promise<void> { await this.request("pause", {}) ; this.status = "stopped"; this.onChange(this.status) }
  async disconnect(): Promise<void> {
    try { await this.request("disconnect", { terminateDebuggee: true }) } catch { /* ignore adapter shutdown errors */ }
    if (this.process.exitCode === null) this.process.kill("SIGTERM")
    this.status = this.status === "failed" ? "failed" : "exited"
    this.onChange(this.status)
  }
}
