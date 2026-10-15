// Fixture mínima do Debug Adapter Protocol sobre stdio com framing
// Content-Length. Responde initialize/launch/threads/stackTrace/scopes/
// variables/disconnect e grava o último payload de setBreakpoints no
// arquivo informado em argv[2] para asserções do teste.
let buffer = ""
let seq = 0
const captureFile = process.argv[2] as string | undefined

function send(message: unknown): void {
  const payload = JSON.stringify(message)
  process.stdout.write(`Content-Length: ${Buffer.byteLength(payload)}\r\n\r\n${payload}`)
}

async function handle(message: { type?: string; seq?: number; command?: string; arguments?: unknown }): Promise<void> {
  if (message.type !== "request" || typeof message.seq !== "number" || !message.command) return
  const args = (message.arguments ?? {}) as Record<string, unknown>
  let body: unknown = {}
  switch (message.command) {
    case "initialize":
      body = { supportsConfigurationDoneRequest: true }
      break
    case "setBreakpoints": {
      const breakpoints = (args.breakpoints ?? []) as Array<{ line: number; condition?: string }>
      body = { breakpoints: breakpoints.map(breakpoint => ({ verified: true, line: breakpoint.line })) }
      if (captureFile) await Bun.write(captureFile, JSON.stringify({ source: args.source, breakpoints }))
      break
    }
    case "threads":
      body = { threads: [{ id: 1, name: "main" }] }
      break
    case "stackTrace":
      body = { stackFrames: [{ id: 11, name: "main", line: 10, column: 1, source: { path: "/test.ts", name: "test.ts" } }] }
      break
    case "scopes":
      body = { scopes: [{ name: "Locals", variablesReference: 7 }] }
      break
    case "variables":
      body = { variables: [{ name: "x", value: "42", type: "int", variablesReference: 0 }] }
      break
    default:
      body = {}
      break
  }
  send({ seq: ++seq, type: "response", request_seq: message.seq, success: true, command: message.command, body })
  if (message.command === "disconnect") process.exit(0)
}

process.stdin.on("data", (chunk: Buffer) => {
  buffer += chunk.toString()
  let headerEnd = buffer.indexOf("\r\n\r\n")
  while (headerEnd !== -1) {
    const length = /^Content-Length:\s*(\d+)/i.exec(buffer.slice(0, headerEnd))?.[1]
    if (!length) { buffer = buffer.slice(headerEnd + 4); headerEnd = buffer.indexOf("\r\n\r\n"); continue }
    const size = Number(length), bodyStart = headerEnd + 4
    if (buffer.length < bodyStart + size) return
    const message = JSON.parse(buffer.slice(bodyStart, bodyStart + size)) as { type?: string; seq?: number; command?: string; arguments?: unknown }
    buffer = buffer.slice(bodyStart + size)
    headerEnd = buffer.indexOf("\r\n\r\n")
    void handle(message)
  }
})
