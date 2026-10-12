import { expect, test, afterEach } from "bun:test"
import { mkdtemp, rm } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { Document } from "../src/core/document"
import { Application, type Overlay } from "../src/app/application"
import { receiveKey } from "../src/app/input"
import { defaultConfig } from "../src/services/config"
import {
  extractBufferWords,
  suggestBufferWords,
  wordAtOffset,
  extractDocumentSymbols,
  findSymbol
} from "../src/services/symbols"
import {
  getLanguageCatalog,
  lookupCatalogItem
} from "../src/services/languageCatalog"

function clearOverlay(app: Application): void {
  app.overlay = undefined
}
function currentOverlay(app: Application): Overlay | undefined {
  return app.overlay
}

const roots: string[] = []
async function fixture(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), "oride-intelligence-"))
  roots.push(root)
  return root
}

afterEach(async () => {
  await Promise.all(roots.splice(0).map(root => rm(root, { recursive: true, force: true })))
})

test("extractBufferWords indexes unique identifiers across multiple documents", () => {
  const doc1 = new Document(1, "fn calculate_total(price: f64, tax_rate: f64) -> f64 { price * (1.0 + tax_rate) }")
  const doc2 = new Document(2, "let total = calculate_total(100.0, 0.2);\nconst A = 1;\nlet variável_olá = true;")

  const words = extractBufferWords([doc1, doc2])
  expect(words.has("calculate_total")).toBe(true)
  expect(words.has("price")).toBe(true)
  expect(words.has("tax_rate")).toBe(true)
  expect(words.has("total")).toBe(true)
  expect(words.has("variável_olá")).toBe(true)
  // Single char 'A' is filtered out (min length 2)
  expect(words.has("A")).toBe(false)
  // Numbers alone are filtered out
  expect(words.has("100")).toBe(false)
})

test("suggestBufferWords ranks prefix matches and fuzzy suggestions", () => {
  const words = new Set(["calculate_total", "calculate_subtotal", "calc", "calibrate", "calendar", "other_var"])
  const suggestions = suggestBufferWords("cal", words, 10)

  expect(suggestions.length).toBeGreaterThan(0)
  expect(suggestions[0]).toBe("calc")
  expect(suggestions).toContain("calculate_total")
  expect(suggestions).toContain("calculate_subtotal")
  expect(suggestions).not.toContain("other_var")
})

test("wordAtOffset extracts identifier and macro syntax at cursor", () => {
  const text = "println!(\"Hello\"); let count_value = 42;"
  // Inside "println!"
  expect(wordAtOffset(text, 3)).toBe("println!")
  expect(wordAtOffset(text, 7)).toBe("println!") // on '!'
  // Inside "count_value"
  expect(wordAtOffset(text, 25)).toBe("count_value")
  // At whitespace / punctuation where neither current nor previous is word
  expect(wordAtOffset(text, 18)).toBe("")
})

test("extractDocumentSymbols parses declarations and preceding doc comments", () => {
  const rustCode = `
/// Calculates the Euclidean distance between two points.
/// Returns a floating-point scalar.
pub fn distance(x1: f64, y1: f64, x2: f64, y2: f64) -> f64 {
    ((x2 - x1).powi(2) + (y2 - y1).powi(2)).sqrt()
}

// User representation struct
pub struct UserProfile {
    pub id: u64,
}
`
  const symbols = extractDocumentSymbols(rustCode)
  const fnSymbol = symbols.find(s => s.name === "distance")
  expect(fnSymbol).toBeDefined()
  expect(fnSymbol?.kind).toBe("function")
  expect(fnSymbol?.doc).toContain("Calculates the Euclidean distance")
  expect(fnSymbol?.signature).toContain("distance(x1: f64, y1: f64, x2: f64, y2: f64) -> f64")

  const structSymbol = symbols.find(s => s.name === "UserProfile")
  expect(structSymbol).toBeDefined()
  expect(structSymbol?.kind).toBe("type")
  expect(structSymbol?.doc).toContain("User representation struct")

  const tsCode = `
/**
 * Processes incoming network payload.
 */
export async function handlePayload(data: Uint8Array): Promise<void> {
}

export interface AppConfig {
    port: number;
}
`
  const tsSymbols = extractDocumentSymbols(tsCode)
  const handlePayloadSym = tsSymbols.find(s => s.name === "handlePayload")
  expect(handlePayloadSym).toBeDefined()
  expect(handlePayloadSym?.kind).toBe("function")
  expect(handlePayloadSym?.doc).toContain("Processes incoming network payload.")

  const interfaceSym = tsSymbols.find(s => s.name === "AppConfig")
  expect(interfaceSym).toBeDefined()
  expect(interfaceSym?.kind).toBe("interface")

  const pyCode = `
# Computes fibonacci sequence iteratively
def fibonacci(n: int) -> int:
    pass

class TaskRunner:
    pass
`
  const pySymbols = extractDocumentSymbols(pyCode)
  const fibSym = pySymbols.find(s => s.name === "fibonacci")
  expect(fibSym).toBeDefined()
  expect(fibSym?.doc).toContain("Computes fibonacci sequence iteratively")

  const classSym = pySymbols.find(s => s.name === "TaskRunner")
  expect(classSym).toBeDefined()
  expect(classSym?.kind).toBe("class")
})

test("findSymbol locates symbol declaration across document collection", () => {
  const doc1 = new Document(1, "pub fn primary_operation() {}\n", "/workspace/src/lib.rs")
  const doc2 = new Document(2, "pub fn secondary_operation() {}\n", "/workspace/src/util.rs")

  const found = findSymbol("secondary_operation", [doc1, doc2])
  expect(found).toBeDefined()
  expect(found?.name).toBe("secondary_operation")
  expect(found?.kind).toBe("function")
})

test("language catalog provides keywords, built-ins, and documentation", () => {
  const rustCat = getLanguageCatalog("rust")
  expect(rustCat).toBeDefined()
  expect(rustCat?.["println!"]).toBeDefined()
  expect(rustCat?.["println!"]?.kind).toBe("macro")
  expect(rustCat?.["println!"]?.doc).toContain("stdout")

  const tsCat = getLanguageCatalog("typescript")
  expect(tsCat).toBeDefined()
  expect(tsCat?.["console.log"]).toBeDefined()
  expect(tsCat?.["Promise"]).toBeDefined()

  const pyCat = getLanguageCatalog("python")
  expect(pyCat).toBeDefined()
  expect(pyCat?.["print"]).toBeDefined()
  expect(pyCat?.["len"]).toBeDefined()

  // Lookup helper
  const item = lookupCatalogItem("rust", "Vec")
  expect(item).toBeDefined()
  expect(item?.kind).toBe("type")
  expect(item?.detail).toContain("Vec<T>")

  const missing = lookupCatalogItem("rust", "non_existent_thing_12345")
  expect(missing).toBeUndefined()

  const missingLang = lookupCatalogItem("unsupported_lang", "test")
  expect(missingLang).toBeUndefined()
})

test("Application offline completion provides catalog items, symbols and buffer words", async () => {
  const root = await fixture()
  const filePath = join(root, "main.rs")
  const source = `
/// Compute hash
fn calculate_hash(val: &str) -> u64 { 0 }

fn main() {
    let custom_metric = 42;
    calc
}
`
  await Bun.write(filePath, source)
  const config = defaultConfig()
  config.lsp.enabled = false // completely offline, no LSP binary needed

  const app = new Application(root, config)
  await app.openFile(filePath)

  // Move cursor to end of "calc"
  const calcOffset = app.document.text.indexOf("calc") + 4
  app.document.select(calcOffset)

  await app.complete(false)

  // Inline completion dropdown should be active with completions
  expect(app.completion).toBeDefined()
  const items = app.completion?.items || []
  expect(items.length).toBeGreaterThan(0)

  // Should have "calculate_hash" (extracted symbol)
  const calcSymbolEntry = items.find(e => e.label === "calculate_hash")
  expect(calcSymbolEntry).toBeDefined()
  expect(calcSymbolEntry?.detail).toContain("function")

  // Cancel completion
  app.cancelCompletion()
  expect(app.completion).toBeUndefined()
  await app.close()
})

test("Application offline hover shows catalog docs or symbol docstrings without LSP", async () => {
  const root = await fixture()
  const filePath = join(root, "service.rs")
  const source = `
/// Dispatches event to registered handlers.
fn dispatch_event(event_id: u32) {
    println!("dispatched");
}
`
  await Bun.write(filePath, source)
  const config = defaultConfig()
  config.lsp.enabled = false

  const app = new Application(root, config)
  await app.openFile(filePath)

  // 1. Hover over declared symbol "dispatch_event"
  const dispatchOffset = app.document.text.indexOf("dispatch_event") + 2
  app.document.select(dispatchOffset)
  await app.hover()

  let overlay = currentOverlay(app)
  expect(overlay?.kind).toBe("read")
  if (overlay?.kind === "read") {
    expect(overlay.title).toContain("dispatch_event")
    expect(overlay.text).toContain("Dispatches event to registered handlers.")
  }

  // Dismiss hover
  clearOverlay(app)

  // 2. Hover over static catalog built-in "println!"
  const printlnOffset = app.document.text.indexOf("println!") + 2
  app.document.select(printlnOffset)
  await app.hover()

  overlay = currentOverlay(app)
  expect(overlay?.kind).toBe("read")
  if (overlay?.kind === "read") {
    expect(overlay.title).toContain("println!")
    expect(overlay.text).toContain("stdout")
  }

  // Dismiss hover
  clearOverlay(app)

  // 3. Hover over catalog type "u32"
  const u32Offset = app.document.text.indexOf("u32") + 1
  app.document.select(u32Offset)
  await app.hover()

  overlay = currentOverlay(app)
  expect(overlay?.kind).toBe("read")
  if (overlay?.kind === "read") {
    expect(overlay.title).toContain("u32")
  }

  clearOverlay(app)
  // Non-existent symbol in whitespace
  app.document.select(0)
  await app.hover()
  expect(currentOverlay(app)).toBeUndefined()

  await app.close()
})

test("TypeScript completion suggests 'if' keyword without noisy buffer words and accepts it", async () => {
  const root = await fixture()
  const filePath = join(root, "app.ts")
  const source = `// Some code with various words like config and certificate
const configuration = 1;
const certificate = 2;
function run() {
  if
}
`
  await Bun.write(filePath, source)
  const config = defaultConfig()
  config.lsp.enabled = false

  const app = new Application(root, config)
  await app.openFile(filePath)

  const ifOffset = app.document.text.indexOf("  if\n") + 4
  app.document.select(ifOffset)

  await app.complete(false)

  expect(app.completion).toBeDefined()
  const items = app.completion?.items ?? []
  expect(items.length).toBeGreaterThan(0)

  // Top item should be "if" keyword
  expect(items[0]?.label).toBe("if")
  expect(items[0]?.kind).toBe("keyword")

  // Should NOT contain fuzzy matches like configuration or certificate for prefix "if"
  const noisyMatch = items.find(i => i.label === "certificate" || i.label === "configuration")
  expect(noisyMatch).toBeUndefined()

  // Accepting completion
  app.acceptCompletion(0)
  expect(app.completion).toBeUndefined()
  expect(app.document.text).toContain("if")

  await app.close()
})

test("completion dropdown keyboard navigation and dismissal", async () => {
  const root = await fixture()
  const filePath = join(root, "main.ts")
  await Bun.write(filePath, "function test() {\n  co\n}")
  const app = new Application(root, defaultConfig())
  await app.openFile(filePath)

  const offset = app.document.text.indexOf("  co\n") + 4
  app.document.select(offset)

  await app.complete(false)
  expect(app.completion).toBeDefined()
  expect(app.completion?.items.length).toBeGreaterThan(1)
  expect(app.completion?.selected).toBe(0)

  // nextCompletion steps forward and backward
  app.nextCompletion(1)
  expect(app.completion?.selected).toBe(1)
  app.nextCompletion(-1)
  expect(app.completion?.selected).toBe(0)

  // cancelCompletion dismisses
  app.cancelCompletion()
  expect(app.completion).toBeUndefined()

  await app.close()
})

test("interactive typing and backspacing with active completion popup in editor", async () => {
  const root = await fixture()
  const filePath = join(root, "calc.ts")
  await Bun.write(filePath, "const base = 10;\nfunction calculate() {\n  \n}\n")
  const app = new Application(root, defaultConfig())
  await app.openFile(filePath)

  // Move cursor to empty line inside function (line 2)
  const line2Pos = app.document.text.indexOf("function calculate() {\n  ") + "function calculate() {\n  ".length
  app.document.select(line2Pos)
  expect(app.document.position.line).toBe(2)

  // Type 'i'
  receiveKey(app, { name: "i", sequence: "i", ctrl: false, meta: false, shift: false, preventDefault: () => {}, stopPropagation: () => {} } as any)
  expect(app.document.position.line).toBe(2)

  // Type 'f' -> word is "if"
  receiveKey(app, { name: "f", sequence: "f", ctrl: false, meta: false, shift: false, preventDefault: () => {}, stopPropagation: () => {} } as any)
  expect(app.document.position.line).toBe(2)

  // Complete manually or auto
  await app.complete(true)
  expect(app.completion).toBeDefined()
  expect(app.completion?.items[0]?.label).toBe("if")

  // Now press '(' with completion active -> autopair inserts '()'
  receiveKey(app, { name: "(", sequence: "(", ctrl: false, meta: false, shift: false, preventDefault: () => {}, stopPropagation: () => {} } as any)
  expect(app.completion).toBeUndefined() // dismissed because prefix is empty
  expect(app.document.position.line).toBe(2)
  expect(app.document.text).toContain("if()")
  expect(app.document.text[app.document.primary.head - 1]).toBe("(")
  expect(app.document.text[app.document.primary.head]).toBe(")")

  // Now press backspace -> deletes pair '()'
  receiveKey(app, { name: "backspace", ctrl: false, meta: false, shift: false, preventDefault: () => {}, stopPropagation: () => {} } as any)
  expect(app.document.position.line).toBe(2)
  expect(app.document.text).toContain("if")
  expect(app.document.text).not.toContain("if()")

  await app.close()
})


