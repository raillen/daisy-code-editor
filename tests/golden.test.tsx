// Golden opt-in do Workbench: frames de caracteres exatos, versionados.
//
// Por que opt-in: frames são sensíveis a timing (highlight assíncrono,
// streaming do agente) e a ambiente (PATH do git, fontes). Rodar sempre
// deixaria o `bun test` lento e flaky; então este arquivo pula tudo, a menos
// que DAISY_GOLDEN=1. O CI pode rodar com a flag; o dia a dia não paga nada.
//
// - `DAISY_GOLDEN=1 bun test tests/golden.test.ts` — compara com
//   tests/golden/*.txt e falha em qualquer célula diferente.
// - `DAISY_GOLDEN_UPDATE=1 ...` — regrava os goldens (revise o diff!).
//
// Determinismo: workspace temporário com nome normalizado (<root>/<ws>),
// locale en, LSP desligado, provedores/listagens fixos, e espera explícita
// pelo estado estável (git entries, turno idle) antes do frame.
import { expect, test } from "bun:test"
import { tmpdir } from "node:os"
import { join, basename } from "node:path"
import { mkdtemp, rm } from "node:fs/promises"
import { testRender } from "@opentui/solid"
import { Application } from "../src/app/application"
import { defaultConfig } from "../src/services/config"
import { Workbench } from "../src/ui/workbench"
import { execute } from "../src/services/git"
import { echoProvider } from "../src/services/agents"

const enabled = !!process.env.DAISY_GOLDEN
const update = !!process.env.DAISY_GOLDEN_UPDATE
const it = enabled || update ? test : test.skip

const WIDTH = 100
const HEIGHT = 30

async function waitFor(predicate: () => boolean, timeout = 5000): Promise<void> {
  const deadline = performance.now() + timeout
  while (!predicate() && performance.now() < deadline) await Bun.sleep(10)
  expect(predicate()).toBe(true)
}

function normalize(frame: string, root: string): string {
  return frame.split(root).join("<root>").split(basename(root)).join("<ws>")
}

async function setup() {
  const root = await mkdtemp(join(tmpdir(), "oride-golden-"))
  const config = defaultConfig(); config.lsp.enabled = false; config.locale = "en"
  const app = new Application(root, config)
  // Goldens determinísticos: provedor echo pinado (ver workbench.test.tsx).
  app.agentProviders = [echoProvider()]
  await Bun.write(join(root, "main.ts"), 'const answer = 42\nconsole.log(answer)\n')
  await app.tree.refresh(); app.newDocument()
  const renderer = await testRender(() => <Workbench app={app} />, { width: WIDTH, height: HEIGHT, kittyKeyboard: true })
  await renderer.renderOnce()
  return { app, root, ...renderer, cleanup: async () => { renderer.renderer.destroy(); await app.close(); await rm(root, { recursive: true, force: true }) } }
}

async function frameOf(name: string, view: Awaited<ReturnType<typeof setup>>): Promise<void> {
  await view.renderOnce()
  const frame = normalize(view.captureCharFrame(), view.root)
  const path = new URL(`./golden/${name}.txt`, import.meta.url)
  if (update) {
    await Bun.write(path, frame + "\n")
    return
  }
  const expected = (await Bun.file(path).text()).replace(/\n$/, "")
  expect(frame).toBe(expected)
}

it("golden: boot com árvore e buffer vazio", async () => {
  const view = await setup()
  try { await frameOf("boot", view) }
  finally { await view.cleanup() }
})

it("golden: painel de agentes com resposta do echo", async () => {
  const view = await setup()
  try {
    await view.app.dispatch("toggle_agents")
    view.app.agentInput = "hello"
    await view.app.sendAgentPrompt()
    await waitFor(() => view.app.agentTabs[0]?.status === "idle" && (view.app.agentTabs[0]?.messages.length ?? 0) >= 2)
    await frameOf("agents", view)
  } finally { await view.cleanup() }
})

it("golden: modal SCM com mudança modificada", async () => {
  const view = await setup()
  try {
    await execute("git", ["init", "-b", "main"], view.root)
    await execute("git", ["config", "user.email", "test@example.invalid"], view.root)
    await execute("git", ["config", "user.name", "Golden Test"], view.root)
    await Bun.write(join(view.root, "tracked.txt"), "first\n")
    await execute("git", ["add", "-A"], view.root)
    await execute("git", ["commit", "-m", "base"], view.root)
    await Bun.write(join(view.root, "tracked.txt"), "second\n")
    await view.app.dispatch("toggle_scm")
    await waitFor(() => view.app.gitStatus.entries.length === 1)
    await frameOf("scm-modal", view)
  } finally { await view.cleanup() }
})

it("golden: paleta filtrando agentes", async () => {
  const view = await setup()
  try {
    await view.app.dispatch("command_palette")
    await waitFor(() => view.app.overlay?.kind === "picker")
    if (view.app.overlay?.kind === "picker") view.app.overlay.query = "agents"
    view.app.updatePicker()
    await frameOf("palette", view)
  } finally { await view.cleanup() }
})
