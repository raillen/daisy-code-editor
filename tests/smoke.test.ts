// Smoke do binário real em PTY: o que o testRender não cobre (stdin real,
// alt-screen, fluxo completo do renderer, arquivos de verdade no disco)
// afirmado no frame emulado.
import { afterEach, expect, test } from "bun:test"
import { mkdir, mkdtemp, rm } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { spawnDaisy, type PtySession } from "./helpers/pty"

const supported = process.platform !== "win32"
const roots: string[] = []
afterEach(async () => { await Promise.all(roots.splice(0).map(root => rm(root, { recursive: true, force: true }))) })

async function fixture(): Promise<{ session: PtySession; workspace: string }> {
  const root = await mkdtemp(join(tmpdir(), "daisy-smoke-"))
  roots.push(root)
  const home = join(root, "home")
  const configHome = join(home, ".config")
  await mkdir(join(configHome, "daisy"), { recursive: true })
  await Bun.write(join(configHome, "daisy", "config.toml"), [
    'locale = "en"',
    "",
    "[editor]",
    "completion_auto = false",
    "",
    "[lsp]",
    "enabled = false",
    "",
  ].join("\n"))
  const workspace = join(root, "workspace")
  await mkdir(workspace, { recursive: true })
  await Bun.write(join(workspace, "notes.txt"), "smoke-file-content\n")
  const session = await spawnDaisy({
    workspace,
    env: { HOME: home, XDG_CONFIG_HOME: configHome, XDG_CACHE_HOME: join(home, ".cache") },
  })
  return { session, workspace }
}

test.skipIf(!supported)("real binary boots, saves a new buffer to disk and opens a file from the picker", async () => {
  const { session, workspace } = await fixture()
  try {
    await session.waitFor(/File\s+Edit\s+View/, 20000)
    await session.type("hello pty")
    await session.waitFor("hello pty")
    session.press("ctrl+s")
    await session.waitFor(/Save as/)
    await session.type("saved.txt")
    session.press("enter")
    const saved = join(workspace, "saved.txt")
    const deadline = performance.now() + 10000
    for (;;) {
      const text = await Bun.file(saved).text().catch(() => "")
      if (text.includes("hello pty")) break
      if (performance.now() > deadline) throw new Error(`saved.txt never reached disk\n--- frame ---\n${session.frame()}`)
      await Bun.sleep(50)
    }
    await session.waitFor("saved.txt")
    session.press("ctrl+p")
    await session.waitFor("Open file")
    await session.type("notes")
    await session.waitFor("notes.txt")
    session.press("enter")
    await session.waitFor("smoke-file-content")
    session.press("ctrl+q")
    expect(await session.waitExit()).toBe(0)
  } finally { await session.close() }
}, 60000)

test.skipIf(!supported)("quit confirmation protects unsaved work and can be cancelled", async () => {
  const { session } = await fixture()
  try {
    await session.waitFor(/File\s+Edit\s+View/, 20000)
    await session.type("draft")
    await session.waitFor("draft")
    session.press("ctrl+q")
    await session.waitFor(/Quit and discard unsaved changes/)
    session.press("escape")
    await Bun.sleep(120)
    await session.type(" continues")
    await session.waitFor("draft continues")
    session.press("ctrl+q")
    await session.waitFor(/Quit and discard unsaved changes/)
    session.press("enter")
    expect(await session.waitExit()).toBe(0)
  } finally { await session.close() }
}, 60000)
