// Optional tree-sitter backend with zero bundled dependencies. The runtime
// (`tree-sitter`) and grammars (`tree-sitter-<language>`) load lazily from
// user-installed packages only when `highlighting.engine` is "tree-sitter".
// Every failure degrades to undefined so callers fall back to Shiki.
import { createRequire } from "node:module"

export type SyntaxNode = {
  type: string
  startIndex: number
  endIndex: number
  childCount: number
  child(index: number): SyntaxNode | null
  isNamed(): boolean
}
export type TreeSitterLoader = (moduleName: string) => unknown
export type TreeToken = { start: number; end: number; role: "comment" | "string" | "number" | "keyword" }

const grammarModules: Record<string, { module: string; member?: string }> = {
  rust: { module: "tree-sitter-rust" },
  c: { module: "tree-sitter-c" },
  cpp: { module: "tree-sitter-cpp" },
  typescript: { module: "tree-sitter-typescript", member: "typescript" },
  tsx: { module: "tree-sitter-typescript", member: "tsx" },
  javascript: { module: "tree-sitter-javascript" },
  jsx: { module: "tree-sitter-javascript" },
  python: { module: "tree-sitter-python" },
  bash: { module: "tree-sitter-bash" },
  ruby: { module: "tree-sitter-ruby" },
}

const keywords: Record<string, Set<string>> = {
  rust: new Set("fn let mut const static struct enum impl trait type pub use mod return if else match for while loop in break continue move ref self Self true false as async await".split(" ")),
  c: new Set("int char short long float double void signed unsigned struct union enum typedef static const extern return if else for while do switch case break continue sizeof".split(" ")),
  cpp: new Set("int char short long float double void bool struct union enum class typedef static const extern return if else for while do switch case break continue sizeof new delete namespace using public private protected virtual template typename".split(" ")),
  typescript: new Set("function const let var return if else for while do switch case break continue class extends import export from default new delete typeof instanceof in of try catch finally throw async await this super null true false undefined interface type".split(" ")),
  tsx: new Set("function const let var return if else for while class extends import export from default new typeof in of try catch finally throw async await this null true false undefined".split(" ")),
  javascript: new Set("function const let var return if else for while do switch case break continue class extends import export from default new delete typeof instanceof in of try catch finally throw async await this super null true false undefined".split(" ")),
  jsx: new Set("function const let var return if else for class extends import export from default new typeof in of try catch finally throw this null true false undefined".split(" ")),
  python: new Set("def return if else elif for while import from as class pass lambda None True False and or not in is with try except finally raise global nonlocal assert del yield".split(" ")),
  bash: new Set("if then else elif fi for while until do done case esac function return select in".split(" ")),
  ruby: new Set("def end class module if else elsif unless while until for in do return yield self nil true false and or not".split(" ")),
}

function roleOf(node: SyntaxNode, language: string): TreeToken["role"] | undefined {
  const type = node.type
  if (type.includes("comment")) return "comment"
  if (!node.isNamed()) return keywords[language]?.has(type) ? "keyword" : undefined
  if (/string|heredoc|template/.test(type) && !/content|fragment|escape|interpolation|substitution/.test(type)) return "string"
  if (/number|integer|float/.test(type)) return "number"
  return undefined
}

function defaultLoader(moduleName: string): unknown {
  try {
    return createRequire(import.meta.url)(moduleName)
  } catch {
    return undefined
  }
}

type ParserAdapter = { parse(text: string): { rootNode: SyntaxNode } }

export class TreeSitterEngine {
  private parsers = new Map<string, ParserAdapter>()
  private missing = new Set<string>()
  constructor(private loader: TreeSitterLoader = defaultLoader) {}

  available(language: string): boolean {
    return this.parserFor(language) !== undefined
  }

  tokenize(language: string, text: string, maxTokens = 100000): TreeToken[] | undefined {
    const parser = this.parserFor(language)
    if (!parser) return undefined
    let root: SyntaxNode
    try {
      root = parser.parse(text).rootNode
    } catch {
      return undefined
    }
    const tokens: TreeToken[] = []
    const stack: SyntaxNode[] = [root]
    while (stack.length > 0) {
      const node = stack.pop()!
      const role = roleOf(node, language)
      if (role) {
        if (node.endIndex > node.startIndex) tokens.push({ start: node.startIndex, end: node.endIndex, role })
        if (tokens.length >= maxTokens) return tokens
        continue
      }
      for (let index = node.childCount - 1; index >= 0; index--) {
        const child = node.child(index)
        if (child) stack.push(child)
      }
    }
    return tokens.sort((left, right) => left.start - right.start)
  }

  private parserFor(language: string): ParserAdapter | undefined {
    const grammar = grammarModules[language]
    if (!grammar) return undefined
    const cached = this.parsers.get(language)
    if (cached) return cached
    if (this.missing.has(language)) return undefined
    try {
      const runtime = this.loader("tree-sitter") as (new () => { setLanguage(grammar: unknown): void; parse(text: string): { rootNode: SyntaxNode } }) | undefined
      const module = this.loader(grammar.module) as Record<string, unknown> | undefined
      const selected = grammar.member && module ? module[grammar.member] : module
      if (!runtime || !selected) {
        this.missing.add(language)
        return undefined
      }
      const parser = new runtime()
      parser.setLanguage(selected)
      const adapter: ParserAdapter = { parse: (text: string) => parser.parse(text) }
      this.parsers.set(language, adapter)
      return adapter
    } catch {
      this.missing.add(language)
      return undefined
    }
  }
}
