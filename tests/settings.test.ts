import { expect, test, afterEach } from "bun:test"
import { mkdtemp, rm } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { Application } from "../src/app/application"
import { defaultConfig } from "../src/services/config"

const roots: string[] = []
async function fixture(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), "oride-settings-"))
  roots.push(root)
  return root
}

afterEach(async () => {
  await Promise.all(roots.splice(0).map(root => rm(root, { recursive: true, force: true })))
})

test("autopairs auto-closes brackets and quotes and positions cursor in middle", async () => {
  const root = await fixture()
  const app = new Application(root, defaultConfig())
  app.newDocument()

  // 1. Typing '(' auto-inserts '()' with cursor inside
  app.typeChar("(")
  expect(app.document.text).toBe("()")
  expect(app.document.primary.head).toBe(1)

  // 2. Typing ')' auto-skips existing closing bracket
  app.typeChar(")")
  expect(app.document.text).toBe("()")
  expect(app.document.primary.head).toBe(2)

  // 3. Typing '[' inside a new line
  app.insert(" ")
  app.typeChar("[")
  expect(app.document.text).toBe("() []")
  expect(app.document.primary.head).toBe(4)

  // 4. Typing '{'
  app.typeChar("{")
  expect(app.document.text).toBe("() [{}]")
  expect(app.document.primary.head).toBe(5)

  // 5. Typing '"'
  app.document.select(app.document.text.length)
  app.insert(" ")
  app.typeChar('"')
  expect(app.document.text).toBe('() [{}] ""')
  expect(app.document.primary.head).toBe(9)

  await app.close()
})

test("autopairs surrounds active selection with matching delimiters", async () => {
  const root = await fixture()
  const app = new Application(root, defaultConfig())
  app.newDocument()
  app.insert("selected_word")

  // Select the word
  app.document.select(0, 13)
  expect(app.document.selectedText).toBe("selected_word")

  // Type '(' to surround
  app.typeChar("(")
  expect(app.document.text).toBe("(selected_word)")

  // Select 'selected_word' again and surround with '"'
  app.document.select(1, 14)
  app.typeChar('"')
  expect(app.document.text).toBe('("selected_word")')

  await app.close()
})

test("autopairs backspace deletes matching pair when cursor is between them", async () => {
  const root = await fixture()
  const app = new Application(root, defaultConfig())
  app.newDocument()

  // Open pair '()'
  app.typeChar("(")
  expect(app.document.text).toBe("()")
  expect(app.document.primary.head).toBe(1)

  // Backspace deletes both
  app.backspace()
  expect(app.document.text).toBe("")
  expect(app.document.primary.head).toBe(0)

  // Type text inside pair
  app.typeChar("(")
  app.typeChar("a")
  expect(app.document.text).toBe("(a)")
  // Backspace here should only delete 'a', keeping parentheses
  app.backspace()
  expect(app.document.text).toBe("()")
  expect(app.document.primary.head).toBe(1)

  // Now between pair, backspace deletes both
  app.backspace()
  expect(app.document.text).toBe("")

  await app.close()
})

test("autopairs backspace preserves cursor on current line when deleting pairs in multi-line document", async () => {
  const root = await fixture()
  const app = new Application(root, defaultConfig())
  app.newDocument()
  app.insert("line 1\nline 2\nconst arr = \nline 4\n")
  // Move cursor to end of "const arr = " (line 2)
  const line2Pos = app.document.text.indexOf("const arr = ") + "const arr = ".length
  app.document.select(line2Pos)
  expect(app.document.position.line).toBe(2)

  // Type '[' -> inserts '[]'
  app.typeChar("[")
  expect(app.document.text).toContain("const arr = []")
  expect(app.document.position.line).toBe(2)
  expect(app.document.primary.head).toBe(line2Pos + 1)

  // Backspace deletes both brackets
  app.backspace()
  expect(app.document.text).toContain("const arr = \n")
  // Verify cursor DID NOT jump to line 0!
  expect(app.document.position.line).toBe(2)
  expect(app.document.primary.head).toBe(line2Pos)

  // Test with '{}'
  app.typeChar("{")
  expect(app.document.text).toContain("const arr = {}")
  expect(app.document.position.line).toBe(2)
  app.backspace()
  expect(app.document.position.line).toBe(2)
  expect(app.document.primary.head).toBe(line2Pos)

  // Test with '""'
  app.typeChar('"')
  expect(app.document.text).toContain('const arr = ""')
  expect(app.document.position.line).toBe(2)
  app.backspace()
  expect(app.document.position.line).toBe(2)
  expect(app.document.primary.head).toBe(line2Pos)

  await app.close()
})

test("autopairs can be disabled via config toggle", async () => {
  const root = await fixture()
  const config = defaultConfig()
  config.editor.auto_pairs = false
  const app = new Application(root, config)
  app.newDocument()

  app.typeChar("(")
  expect(app.document.text).toBe("(")
  expect(app.document.primary.head).toBe(1)

  await app.close()
})

test("menuLabel shows dynamic toggle state checkboxes", async () => {
  const root = await fixture()
  const app = new Application(root, defaultConfig())

  expect(app.menuLabel("toggle_soft_wrap")).toContain("[✓]")
  expect(app.menuLabel("toggle_auto_pairs")).toContain("[✓]")
  expect(app.menuLabel("toggle_line_numbers")).toContain("[✓]")
  expect(app.menuLabel("toggle_icons")).toContain("[ ]")
  expect(app.menuLabel("toggle_modal")).toContain("[ ]")

  // Toggle soft_wrap
  await app.dispatch("toggle_soft_wrap")
  expect(app.config.soft_wrap).toBe(false)
  expect(app.menuLabel("toggle_soft_wrap")).toContain("[ ]")

  // Toggle auto_pairs
  await app.dispatch("toggle_auto_pairs")
  expect(app.config.editor.auto_pairs).toBe(false)
  expect(app.menuLabel("toggle_auto_pairs")).toContain("[ ]")

  // Toggle line numbers
  await app.dispatch("toggle_line_numbers")
  expect(app.config.show_line_numbers).toBe(false)
  expect(app.menuLabel("toggle_line_numbers")).toContain("[ ]")

  // Toggle icons
  await app.dispatch("toggle_icons")
  expect(app.config.icons).toBe(true)
  expect(app.menuLabel("toggle_icons")).toContain("[✓]")

  await app.close()
})

test("openUserConfig and openWorkspaceConfig open configuration TOML as tabs", async () => {
  const root = await fixture()
  const previousXdg = process.env.XDG_CONFIG_HOME
  process.env.XDG_CONFIG_HOME = root
  const app = new Application(root, defaultConfig())

  // Open workspace config
  await app.dispatch("open_workspace_config")
  expect(app.document.path).toBe(join(root, ".daisy", "config.toml"))
  expect(app.document.text).toContain(".daisy/config.toml")
  expect(app.documents.some(d => d.path?.endsWith(".daisy/config.toml"))).toBe(true)

  // Open user config
  await app.dispatch("open_user_config")
  expect(app.document.path?.endsWith("config.toml")).toBe(true)
  expect(app.document.text).toContain("theme")

  await app.close()
  if (previousXdg === undefined) delete process.env.XDG_CONFIG_HOME
  else process.env.XDG_CONFIG_HOME = previousXdg
})

test("saving config.toml automatically triggers hot reload", async () => {
  const root = await fixture()
  const app = new Application(root, defaultConfig())

  await app.openWorkspaceConfig()
  const initialWrap = app.config.soft_wrap

  // Modify config buffer to set soft_wrap = false
  app.document.select(0, app.document.text.length)
  app.insert("soft_wrap = false\n")

  // Save the document
  await app.save()

  // Hot reload should have flipped config.soft_wrap to false!
  expect(app.config.soft_wrap).toBe(false)
  expect(app.config.soft_wrap).not.toBe(initialWrap)

  await app.close()
})
