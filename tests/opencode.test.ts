// Provedor OpenCode nativo contra um fake do binário (tests/helpers/).
// O fake fala o mesmo protocolo de `opencode run --format json` sem rede,
// sem modelo e sem custo; aqui valem args, streaming por sufixo, sessão
// multi-turno, erro, cancelamento e preferência sobre o echo.
//
// Nota de ambiente: `Bun.which` congela o PATH do nascimento do processo, por
// isso o provedor recebe um `resolve` injetável — os testes passam o fake
// direto, sem mutar PATH (que o `which` ignoraria).
import { expect, test } from "bun:test"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { chmodSync } from "node:fs"
import { fileURLToPath } from "node:url"
import { mkdtemp, rm } from "node:fs/promises"
import { Application } from "../src/app/application"
import { defaultConfig } from "../src/services/config"
import { echoProvider } from "../src/services/agents"
import { opencodeProvider, parseOpenCodeLine } from "../src/services/opencode"

type Harness = {
  root: string
  binary: string
  argsFile: string
  /** Resolver injetado no provedor: o wrapper, sem tocar no PATH. */
  resolve: () => string | null
  /** Troca o modo do fake com o vôo em andamento (o spawn não herda env). */
  setMode: (mode: string) => Promise<void>
  argv: () => Promise<string[]>
  cleanup: () => Promise<void>
}

const sh = (value: string): string => `'${value.replace(/'/g, `'\\''`)}'`

async function setupFake(mode = "ok"): Promise<Harness> {
  const root = await mkdtemp(join(tmpdir(), "oride-opencode-"))
  const binary = join(root, "opencode")
  const argsFile = join(root, "args.log")
  const modeFile = join(root, "mode")
  await Bun.write(argsFile, "")
  await Bun.write(modeFile, mode)
  const source = fileURLToPath(new URL("./helpers/fake-opencode.sh", import.meta.url))
  // Wrapper gerado com valores literais: o filho lê modo/args de arquivos,
  // porque o spawn não herda env mutado em runtime (Bun 1.4.2 congela).
  await Bun.write(
    binary,
    `#!/bin/sh\nexport FAKE_OPENCODE_ARGS=${sh(argsFile)}\nexport FAKE_OPENCODE_MODE_FILE=${sh(modeFile)}\nexec ${sh(source)} "$@"\n`,
  )
  chmodSync(binary, 0o755)
  return {
    root,
    binary,
    argsFile,
    resolve: () => binary,
    setMode: next => Bun.write(modeFile, next).then(() => undefined),
    argv: async () => (await Bun.file(argsFile).text()).trim().split("\n").filter(line => line.length > 0),
    cleanup: async () => {
      await rm(root, { recursive: true, force: true })
    },
  }
}

test("parseOpenCodeLine aceita eventos e rejeita lixo", () => {
  const event = parseOpenCodeLine('{"type":"text","timestamp":2,"sessionID":"s","part":{"id":"p","text":"hi"}}')
  expect(event?.type).toBe("text")
  expect(parseOpenCodeLine("! permission requested: edit; auto-rejecting")).toBeUndefined()
  expect(parseOpenCodeLine("not json")).toBeUndefined()
  expect(parseOpenCodeLine('{"nope":1}')).toBeUndefined()
  expect(parseOpenCodeLine('  {"type":"step_finish","timestamp":1}')?.type).toBe("step_finish")
})

test("unavailable when the binary is missing", async () => {
  const provider = opencodeProvider("/tmp", () => null)
  expect(await provider.available()).toBe(false)
  expect(provider.open("default")).rejects.toThrow("not found")
})

test("streams the full reply and records protocol args", async () => {
  const harness = await setupFake()
  try {
    const provider = opencodeProvider(harness.root, harness.resolve)
    expect(await provider.available()).toBe(true)
    expect(provider.models()).toEqual([])
    const conversation = await provider.open("default")
    const chunks: string[] = []
    const turn = await conversation.send("hello", chunk => chunks.push(chunk))
    expect(turn.cancelled).toBe(false)
    expect(chunks.join("")).toBe("Fake: hello")
    const lines = await harness.argv()
    expect(lines[0]).toContain("<--format> <json>")
    expect(lines[0]).toContain("<hello>")
    expect(lines[0]).not.toContain("<--model>")
    expect(lines[0]).not.toContain("<--session>")
    expect(lines[1]).toBe(`run-cwd: ${harness.root}`)
    await conversation.close()
  } finally { await harness.cleanup() }
})

test("split text events forward only the unseen suffix", async () => {
  const harness = await setupFake("split")
  try {
    const conversation = await opencodeProvider(harness.root, harness.resolve).open("default")
    const chunks: string[] = []
    await conversation.send("hello", chunk => chunks.push(chunk))
    expect(chunks).toEqual(["Fake: ", "hello"])
    await conversation.close()
  } finally { await harness.cleanup() }
})

test("second send continues the captured session", async () => {
  const harness = await setupFake()
  try {
    const conversation = await opencodeProvider(harness.root, harness.resolve).open("default")
    await conversation.send("first", () => {})
    await conversation.send("second", () => {})
    const lines = await harness.argv()
    const runs = lines.filter(line => line.startsWith("run-argv:"))
    expect(runs).toHaveLength(2)
    expect(runs[0]).not.toContain("<--session>")
    expect(runs[1]).toContain("<--session> <ses_test123>")
    await conversation.close()
  } finally { await harness.cleanup() }
})

test("noisy non-JSON lines do not break the run", async () => {
  const harness = await setupFake("noisy")
  try {
    const conversation = await opencodeProvider(harness.root, harness.resolve).open("default")
    const chunks: string[] = []
    const turn = await conversation.send("hello", chunk => chunks.push(chunk))
    expect(turn.cancelled).toBe(false)
    expect(chunks.join("")).toBe("Fake: hello")
    await conversation.close()
  } finally { await harness.cleanup() }
})

test("error event rejects the turn", async () => {
  const harness = await setupFake("error")
  try {
    const conversation = await opencodeProvider(harness.root, harness.resolve).open("default")
    expect(conversation.send("hello", () => {})).rejects.toThrow("boom from fake")
    await conversation.close()
  } finally { await harness.cleanup() }
})

test("non-zero exit without events rejects with the stderr tail", async () => {
  const harness = await setupFake("fail")
  try {
    const conversation = await opencodeProvider(harness.root, harness.resolve).open("default")
    expect(conversation.send("hello", () => {})).rejects.toThrow("exited with code 3")
    await conversation.close()
  } finally { await harness.cleanup() }
})

test("quiet run resolves with no text", async () => {
  const harness = await setupFake("quiet")
  try {
    const conversation = await opencodeProvider(harness.root, harness.resolve).open("default")
    const chunks: string[] = []
    const turn = await conversation.send("hello", chunk => chunks.push(chunk))
    expect(turn.cancelled).toBe(false)
    expect(chunks).toEqual([])
    await conversation.close()
  } finally { await harness.cleanup() }
})

test("cancel resolves cancelled and the conversation survives", async () => {
  const harness = await setupFake("slow")
  const conversation = await opencodeProvider(harness.root, harness.resolve).open("default")
  try {
    const started = performance.now()
    const pending = conversation.send("hello", () => {})
    await Bun.sleep(300)
    await conversation.cancel()
    const turn = await pending
    expect(turn.cancelled).toBe(true)
    expect(performance.now() - started).toBeLessThan(9000)
    await harness.setMode("ok")
    const chunks: string[] = []
    const again = await conversation.send("hello", chunk => chunks.push(chunk))
    expect(again.cancelled).toBe(false)
    expect(chunks.join("")).toBe("Fake: hello")
    await conversation.close()
  } finally { await harness.cleanup() }
})

test("closed conversation refuses new turns", async () => {
  const harness = await setupFake()
  try {
    const conversation = await opencodeProvider(harness.root, harness.resolve).open("default")
    await conversation.close()
    expect(conversation.send("hi", () => {})).rejects.toThrow("closed")
  } finally { await harness.cleanup() }
})

test("non-default model passes --model through", async () => {
  const harness = await setupFake()
  try {
    const conversation = await opencodeProvider(harness.root, harness.resolve).open("anthropic/claude-x")
    await conversation.send("hi", () => {})
    const [line] = await harness.argv()
    expect(line).toContain("<--model> <anthropic/claude-x>")
    await conversation.close()
  } finally { await harness.cleanup() }
})

test("app prefers opencode over echo when available, echo otherwise", async () => {
  const harness = await setupFake()
  const root = await mkdtemp(join(tmpdir(), "oride-app-pick-"))
  try {
    const app = new Application(root, defaultConfig())
    app.agentProviders = [opencodeProvider(root, harness.resolve), echoProvider()]
    expect((await app.availableAgentProviders())[0]?.id).toBe("opencode")
    app.agentProviders = [opencodeProvider(root, () => null), echoProvider()]
    expect((await app.availableAgentProviders())[0]?.id).toBe("echo")
    await app.close()
  } finally {
    await harness.cleanup()
    await rm(root, { recursive: true, force: true })
  }
})
