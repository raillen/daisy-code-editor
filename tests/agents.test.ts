// Contrato dos provedores de agente, exercitado pelo provedor de demonstração:
// streaming em pedaços, cancelamento que preserva a conversa e falha fechada
// numa conversa fechada. Os provedores reais (OpenCode nativo, ACP) entram
// pelo mesmo contrato, então estes testes valem para eles também.
import { expect, test } from "bun:test"
import { echoProvider } from "../src/services/agents"

test("echo provider advertises itself and streams a reply in chunks", async () => {
  const provider = echoProvider(0)
  expect(provider.id).toBe("echo")
  expect(provider.models().length).toBeGreaterThan(0)
  expect(await provider.available()).toBe(true)
  const conversation = await provider.open("default")
  const chunks: string[] = []
  const turn = await conversation.send("status?", chunk => chunks.push(chunk))
  expect(turn.cancelled).toBe(false)
  expect(chunks.length).toBeGreaterThan(1)
  expect(chunks.join("")).toContain("status?")
  await conversation.close()
})

test("cancel stops the turn but keeps the conversation usable", async () => {
  const provider = echoProvider(1)
  const conversation = await provider.open()
  let seen = ""
  let count = 0
  const turn = await conversation.send("a much longer prompt to stream", chunk => {
    seen += chunk
    if (++count === 1) void conversation.cancel()
  })
  expect(turn.cancelled).toBe(true)
  expect(seen.length).toBeLessThan(40)
  const next = await conversation.send("again", () => {})
  expect(next.cancelled).toBe(false)
  await conversation.close()
})

test("a closed conversation refuses a new turn", async () => {
  const conversation = await echoProvider(0).open()
  await conversation.close()
  expect(conversation.send("hi", () => {})).rejects.toThrow("closed")
})
