// Contrato dos provedores de agente.
//
// O painel de agentes fala só com esta interface: abrir uma conversa, mandar
// um prompt e receber a resposta em pedaços. O provedor nativo do OpenCode e
// os adapters ACP entram como implementações atrás dela — nada em app/UI
// precisa saber qual está por baixo.
//
// Regras do contrato:
// - `available()` nunca lança: devolve `false` quando o binário/socket não
//   está lá, para a escolha de provedor ser um dado, não uma exceção.
// - `open()` é barato: o provedor pode adiar o spawn do processo para o
//   primeiro `send`.
// - `send()` resolve no fim do turno; pedaços chegam por `onChunk`, que pode
//   ser chamado zero ou muitas vezes. Erro do provedor vira `throw`, e o
//   painel mostra `status` em vez de cair.
// - `cancel()` interrompe o turno em andamento sem derrubar a conversa: o
//   `send` corrente resolve com `cancelled: true`.

export type AgentRole = "user" | "assistant" | "notice"

export type AgentMessage = { role: AgentRole; text: string; at: number }

export type AgentStatus = "idle" | "busy" | "error"

export type AgentTab = {
  id: number
  provider: string
  providerLabel: string
  model: string
  messages: AgentMessage[]
  status: AgentStatus
  error?: string
}

export type AgentTurn = { cancelled: boolean }

export interface AgentConversation {
  send(prompt: string, onChunk: (chunk: string) => void): Promise<AgentTurn>
  cancel(): Promise<void>
  close(): Promise<void>
}

export interface AgentProvider {
  readonly id: string
  readonly label: string
  models(): string[]
  available(): Promise<boolean>
  open(model?: string): Promise<AgentConversation>
}

class EchoConversation implements AgentConversation {
  private cancelled = false
  private closed = false

  constructor(private readonly delay: number) {}

  async send(prompt: string, onChunk: (chunk: string) => void): Promise<AgentTurn> {
    if (this.closed) throw new Error("agent conversation is closed")
    this.cancelled = false
    const text = `Echo: ${prompt}\nDemo provider — no model is called. Native OpenCode and ACP providers land in the next steps.`
    for (const chunk of text.match(/[\s\S]{1,12}/g) ?? []) {
      if (this.cancelled) return { cancelled: true }
      onChunk(chunk)
      if (this.delay > 0) await Bun.sleep(this.delay)
    }
    return { cancelled: false }
  }

  async cancel(): Promise<void> { this.cancelled = true }

  async close(): Promise<void> { this.closed = true; this.cancelled = true }
}

// Provedor de demonstração: responde em pedaços, sem rede nem processo.
// Existe para o painel ser testável de ponta a ponta (e para as demos em VHS)
// antes dos provedores reais — e continua útil como fallback quando nenhum
// outro está no PATH. `delay` é injectável para os testes não esperarem.
export function echoProvider(delay = 12): AgentProvider {
  return {
    id: "echo",
    label: "Echo (demo)",
    models: () => ["default"],
    available: async () => true,
    open: async () => new EchoConversation(delay),
  }
}

export function defaultProviders(): AgentProvider[] {
  return [echoProvider()]
}
