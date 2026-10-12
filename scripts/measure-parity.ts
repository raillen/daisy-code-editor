import { mkdtemp, mkdir, rm } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join, resolve, relative, dirname } from "node:path"
import { parse as parseToml } from "smol-toml"
import { lineRanges } from "../src/core/document"
import { Application } from "../src/app/application"
import { defaultConfig, record } from "../src/services/config"
import { actionIds, type ActionId } from "../src/core/keymap"
import { execute } from "../src/services/git"
import { searchDefaults } from "../src/core/search"

function string(value: unknown): string { if (typeof value !== "string") throw new Error("Expected string in case"); return value }
const oracle = process.env.ORIDE_ORACLE ?? resolve("target/debug/oride")
if (!await Bun.file(oracle).exists()) throw new Error("Build the Rust oracle with cargo build -p oride before measuring parity")
const paths = [...new Bun.Glob("conformance/cases/**/*.toml").scanSync(".")].sort()
if (!paths.length) throw new Error("No reference cases")
const results: { case: string; frames: number; equalFrames: number; differences: { step: number; fields: string[] }[] }[] = []
for (const casePath of paths) {
  const source = record(parseToml(await Bun.file(casePath).text()))
  const root = await mkdtemp(join(tmpdir(), "oride-parity-")), rustRoot = join(root, "rust"), tsRoot = join(root, "typescript")
  await mkdir(rustRoot); await mkdir(tsRoot)
  const config = defaultConfig(); config.lsp.enabled = false; config.editor.completion_auto = false
  const overrides = source.config ? record(source.config) : {}
  const topFields = ["theme", "locale", "show_line_numbers", "soft_wrap", "mouse"] as const
  for (const key of topFields) if (overrides[key] !== undefined) {
    if (key === "theme" || key === "locale") config[key] = string(overrides[key])
    else { if (typeof overrides[key] !== "boolean") throw new Error(`Invalid ${key}`); config[key] = overrides[key] }
  }
  if (typeof overrides.tab_size === "number") config.editor.tab_size = overrides.tab_size
  if (typeof overrides.insert_spaces === "boolean") config.editor.insert_spaces = overrides.insert_spaces
  if (overrides.keys) for (const [chord, raw] of Object.entries(record(overrides.keys))) {
    const action = actionIds.find(action => action === raw)
    if (!action) throw new Error(`Unknown action ${raw}`)
    config.keys[chord] = action
  }
  const app = new Application(tsRoot, config)
  try {
    if (!Array.isArray(source.files) || !Array.isArray(source.steps)) throw new Error("Case requires files and steps")
    for (const raw of source.files) {
      const file = record(raw), path = resolve(tsRoot, string(file.path)), rel = relative(tsRoot, path)
      if (rel === ".." || rel.startsWith("../")) throw new Error("Case path escapes workspace")
      await mkdir(dirname(path), { recursive: true }); await Bun.write(path, string(file.content))
    }
    if (typeof source.open === "string" && source.open) await app.openFile(join(tsRoot, source.open)); else app.newDocument()
    const rust = record(JSON.parse(await execute(oracle, ["conformance", "run", "--case", resolve(casePath), "--workspace", rustRoot, "--json"], process.cwd(), 60000)))
    if (!Array.isArray(rust.frames) || rust.frames.length !== source.steps.length) throw new Error("Oracle frame count mismatch")
    const differences: { step: number; fields: string[] }[] = []
    for (const [index, raw] of source.steps.entries()) {
      const step = record(raw)
      if (step.kind === "text") for (const character of string(step.text)) {
        if (character === "\t") await app.dispatch("insert_tab")
        else app.insert(character)
      }
      else if (step.kind === "search") {
        app.findQuery = typeof step.query === "string" ? step.query : ""
        app.findReplacement = typeof step.replace === "string" ? step.replace : ""
        app.searchOptions = { ...searchDefaults(), caseSensitive: step.case_sensitive === true, ignoreAccents: step.ignore_accents === true, wholeWord: step.whole_word === true, regex: step.use_regex === true }
        app.recomputeFind()
      } else if (step.kind === "action" || step.kind === "chord") {
        const rawAction = step.kind === "chord" ? app.config.keys[string(step.chord)] : step.action
        const action: ActionId | undefined = actionIds.find(action => action === rawAction)
        if (!action) throw new Error(`Unknown step action ${rawAction}`)
        await app.dispatch(action)
      } else throw new Error(`Unknown step ${step.kind}`)
      const state = record(record(rust.frames[index]).state), document = record(state.document), find = record(state.find)
      const bytes = (offset: number) => Buffer.byteLength(app.document.text.slice(0, offset))
      const measured: Record<string, unknown> = {
        text: app.document.text, bytes: Buffer.byteLength(app.document.text), chars: [...app.document.text].length, lines: lineRanges(app.document.text).length,
        matches: app.matches.map(match => ({ start: bytes(match.start), end: bytes(match.end) })),
      }
      const reference: Record<string, unknown> = { text: document.text, bytes: document.bytes, chars: document.chars, lines: document.lines, matches: find.matches }
      const fields = Object.keys(measured).filter(key => JSON.stringify(measured[key]) !== JSON.stringify(reference[key]))
      if (fields.length) differences.push({ step: index + 1, fields })
    }
    results.push({ case: casePath, frames: source.steps.length, equalFrames: source.steps.length - differences.length, differences })
  } finally { await app.close(); await rm(root, { recursive: true, force: true }) }
}
const frames = results.reduce((sum, result) => sum + result.frames, 0), equalFrames = results.reduce((sum, result) => sum + result.equalFrames, 0)
const report = { schema: 1, measuredFields: ["text", "bytes", "chars", "lines", "search.matches"], unmeasured: ["caret", "selection", "undo_labels", "redo_labels", "tabs", "config", "tree", "scm", "vim", "lsp", "terminal", "overlay", "visual_frames"], cases: results.length, frames, equalFrames, results }
await Bun.write("docs/migration/opentui-parity-report.json", JSON.stringify(report, null, 2) + "\n")
console.log(`Measured ${equalFrames}/${frames} frames across ${results.length} cases for ${report.measuredFields.join(", ")}`)
console.log("This is a partial differential measurement, not full feature parity.")
