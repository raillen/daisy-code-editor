// Provedor nativo do OpenCode: dirige `opencode run --format json` atrás do
// contrato AgentProvider.
//
// Protocolo verificado no fonte do CLI v2.0.29
// (packages/cli/src/run/noninteractive.ts): uma linha JSON por evento —
// `step_start`, `text` (texto ACUMULADO da parte, não delta), `tool_use`,
// `step_finish`, `error` — e exit 0 no sucesso. Linhas que não são JSON
// (avisos de permissão auto-rejeitada, por exemplo) são ignoradas.
//
// Decisões de segurança:
// - Sem `--auto`: o CLI rejeita pedidos de permissão com feedback e o modelo
//   continua sem a ação. O painel responde, lê e sugere — mas não edita nem
//   executa sem o usuário. Edição autônoma fica para configuração explícita.
// - stdin fechado (`ignore`): nada no filho pode bloquear esperando entrada.
// - Sem shell: argv direto do spawn, cwd preso ao workspace, stderr podado.
// - Cancel fecha o processo; o `send` em voo resolve `{ cancelled: true }`.
//
// Multi-turno: o primeiro `send` cria a sessão (sem `--session`) e captura o
// `sessionID` dos eventos; os próximos continuam com `--session <id>`.
import type { AgentConversation, AgentProvider, AgentTurn } from "./agents"

export type OpenCodeEvent =
  | { type: "step_start"; timestamp: number; sessionID: string; part: unknown }
  | { type: "text"; timestamp: number; sessionID: string; part: { id: string; text: string } }
  | { type: "tool_use"; timestamp: number; sessionID: string; part: unknown }
  | { type: "step_finish"; timestamp: number; sessionID: string; part: unknown }
  | { type: "error"; timestamp: number; sessionID?: string; error: { message?: string } & Record<string, unknown> }
  | { type: string; timestamp?: number; sessionID?: string; [key: string]: unknown }

/** Uma linha de stdout virada em evento; `undefined` para lixo não-JSON. */
export function parseOpenCodeLine(line: string): OpenCodeEvent | undefined {
  const trimmed = line.trim()
  if (!trimmed.startsWith("{")) return undefined
  try {
    const value = JSON.parse(trimmed) as { type?: unknown }
    return typeof value.type === "string" ? (value as OpenCodeEvent) : undefined
  } catch {
    return undefined
  }
}

const asRecord = (value: unknown): Record<string, unknown> | undefined =>
  typeof value === "object" && value !== null ? (value as Record<string, unknown>) : undefined

const STDERR_TAIL = 4096

class OpenCodeConversation implements AgentConversation {
  private sessionID: string | undefined
  private proc: Bun.Subprocess<"ignore", "pipe", "pipe"> | undefined
  private cancelled = false
  private closed = false
  /** Quanto de cada parte de texto já foi entregue (o evento traz acumulado). */
  private readonly forwarded = new Map<string, number>()

  constructor(
    private readonly directory: string,
    private readonly model: string | undefined,
    private readonly binary: string,
  ) {}

  async send(prompt: string, onChunk: (chunk: string) => void): Promise<AgentTurn> {
    if (this.closed) throw new Error("agent conversation is closed")
    this.cancelled = false
    // Sem `--dir`: o CLI 2.0.29 não tem a flag (chegou depois); o cwd do
    // spawn prende o workspace do mesmo jeito, em todas as versões.
    const args = ["run", "--format", "json"]
    if (this.sessionID) args.push("--session", this.sessionID)
    if (this.model && this.model !== "default") args.push("--model", this.model)
    args.push(prompt)
    const proc = Bun.spawn([this.binary, ...args], {
      cwd: this.directory,
      stdin: "ignore",
      stdout: "pipe",
      stderr: "pipe",
    })
    this.proc = proc
    const decoder = new TextDecoder()
    let pending = ""
    const consume = async (stream: ReadableStream<Uint8Array>, limit: number): Promise<string> => {
      let collected = ""
      for await (const bytes of stream) {
        collected += decoder.decode(bytes, { stream: true })
        if (collected.length > limit) collected = collected.slice(-limit)
      }
      return collected
    }
    const errors: string[] = []
    const pump = (async () => {
      for await (const bytes of proc.stdout) {
        pending += decoder.decode(bytes, { stream: true })
        const lines = pending.split("\n")
        pending = lines.pop() ?? ""
        for (const line of lines) {
          const event = parseOpenCodeLine(line)
          if (!event) continue
          if (typeof event.sessionID === "string" && !this.sessionID) this.sessionID = event.sessionID
          if (event.type === "text") {
            const part = asRecord(event.part)
            const id = part?.id
            const text = part?.text
            if (typeof id !== "string" || typeof text !== "string") continue
            const seen = this.forwarded.get(id) ?? 0
            if (text.length > seen) {
              this.forwarded.set(id, text.length)
              onChunk(text.slice(seen))
            }
          } else if (event.type === "error") {
            const message = asRecord(event.error)?.message
            errors.push(typeof message === "string" ? message : "opencode run failed")
          }
        }
      }
      const tail = parseOpenCodeLine(pending)
      if (tail?.type === "error") {
        const message = asRecord(tail.error)?.message
        if (typeof message === "string") errors.push(message)
      }
    })()
    const stderr = consume(proc.stderr, STDERR_TAIL)
    const code = await proc.exited
    if (this.cancelled || this.closed) {
      // Solta os pipes antes de esperar a bomba: netos do filho (tools do
      // agente, um `sleep` do fake) podem segurar a ponta de escrita por
      // minutos; sem o cancel, o `for await` pendura o turno cancelado.
      // O kill já foi pedido em cancel()/close(); aqui é só parar de ler.
      try { await proc.stdout.cancel() } catch { /* bomba já drenada */ }
      try { await proc.stderr.cancel() } catch { /* idem */ }
    }
    await pump.catch(() => undefined)
    const diagnostics = (await stderr.catch(() => "")).trim()
    if (this.proc === proc) this.proc = undefined
    if (this.cancelled || this.closed) return { cancelled: true }
    if (errors.length) throw new Error(errors[errors.length - 1])
    if (code !== 0) throw new Error(diagnostics ? `opencode exited with code ${code}: ${diagnostics}` : `opencode exited with code ${code}`)
    return { cancelled: false }
  }

  async cancel(): Promise<void> {
    this.cancelled = true
    try { this.proc?.kill() } catch { /* processo já saiu; o send resolve cancelado */ }
  }

  async close(): Promise<void> {
    this.closed = true
    this.cancelled = true
    try { this.proc?.kill() } catch { /* idem */ }
  }
}

export function opencodeProvider(
  directory: string,
  // Injetável porque `Bun.which` congela o PATH do nascimento do processo:
  // teste injeta o fake sem jogos de PATH; produção usa o padrão.
  resolve: () => string | null = () => Bun.which("opencode"),
): AgentProvider {
  return {
    id: "opencode",
    label: "OpenCode",
    models: () => [],
    available: async () => resolve() !== null,
    open: async model => {
      const binary = resolve()
      if (!binary) throw new Error("opencode binary not found on PATH")
      return new OpenCodeConversation(directory, model, binary)
    },
  }
}
