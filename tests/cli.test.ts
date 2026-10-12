import { expect, test } from "bun:test"
import { mkdtemp, rm } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join, resolve } from "node:path"

test("CLI version, help and frame are usable through the public entrypoint", async () => {
  for (const [args, expected] of [[["--version"], "daisy 0.2.0"], [["--help"], "OpenTUI + Solid"], [["--print-frame", "60x12"], "Daisy"]] as const) {
    const child = Bun.spawn([process.execPath, "src/cli.tsx", ...args], { stdout: "pipe", stderr: "pipe" })
    const output = await new Response(child.stdout).text()
    const errors = await new Response(child.stderr).text()
    expect(await child.exited).toBe(0)
    expect(output).toContain(expected)
    expect(errors).not.toContain("error:")
  }
// Timeout folgado de propósito: cada spawn paga o cold start do Bun + shiki, e
// numa máquina carregada isso estoura 10s sem nada estar quebrado (a falha é
// tempo, não produto — o exit code e o stderr continuam afirmando o contrato).
}, 60000)

test("interactive CLI runs in a real PTY, saves text and exits cleanly", async () => {
  const root = await mkdtemp(join(tmpdir(), "daisy-cli-")), path = join(root, "main.txt")
  await Bun.write(path, "before")
  let output = ""
  const terminal = new Bun.Terminal({ cols: 100, rows: 25, data: (_terminal, bytes) => { output = (output + new TextDecoder().decode(bytes)).slice(-100000) } })
  const child = Bun.spawn([process.execPath, resolve("src/cli.tsx"), path], { terminal, cwd: process.cwd(), env: { ...process.env, TERM: "xterm-256color", XDG_CONFIG_HOME: root, XDG_CACHE_HOME: root } })
  let completed = false
  try {
    const deadline = performance.now() + 15000
    while (!output.includes("Daisy") && performance.now() < deadline) await Bun.sleep(20)
    expect(output).toContain("Daisy")
    terminal.write("HELLO ")
    await Bun.sleep(100)
    terminal.write("\x13") // Ctrl+S
    for (let attempt = 0; attempt < 100 && await Bun.file(path).text() !== "HELLO before"; attempt++) await Bun.sleep(20)
    expect(await Bun.file(path).text()).toBe("HELLO before")
    terminal.write("\x11") // Ctrl+Q
    const code = await Promise.race([child.exited, Bun.sleep(3000).then(() => { throw new Error("CLI did not exit after Ctrl+Q") })])
    completed = true
    expect(code).toBe(0)
    expect(output).toContain("\x1b[?1049l") // Main screen restored.
  } finally {
    terminal.close()
    if (!completed) { child.kill("SIGKILL"); await child.exited }
    await rm(root, { recursive: true, force: true })
  }
// Mesmo motivo do teste acima: PTY real + máquina carregada, não produto lento.
}, 60000)
