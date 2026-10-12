import { basename, extname, dirname, join, relative, resolve } from "node:path"
import { realpathSync } from "node:fs"
import { realpath } from "node:fs/promises"
import { createHighlighter, bundledLanguages, type Highlighter, type ThemedToken } from "shiki"
import type { Config } from "./config"
const extensions: Record<string, string> = { rs: "rust", c: "c", h: "c", cpp: "cpp", cc: "cpp", hpp: "cpp", sh: "bash", bash: "bash", md: "markdown", mdx: "mdx", orl: "ori", ori: "ori", oris: "ori", html: "html", htm: "html", css: "css", scss: "scss", js: "javascript", jsx: "jsx", ts: "typescript", tsx: "tsx", py: "python", d: "d", lua: "lua", nim: "nim", rb: "ruby", go: "go", json: "json", toml: "toml", yaml: "yaml", yml: "yaml", zig: "zig" }
export function languageOf(path: string | undefined, config: Config): string {
  if (!path) return "text"
  const extension = extname(path).slice(1)
  const custom = config.languages.find(language => language.extensions.some(value => value.replace(/^\./, "") === extension) || language.filenames.includes(basename(path)))
  if (custom) return custom.id
  if (basename(path) === "Dockerfile") return "dockerfile"
  return extensions[extension] ?? "text"
}
export type CommentDelimiters = { open: string; close?: string }

export function commentDelimiters(language: string, config: Config): CommentDelimiters | undefined {
  const custom = config.languages.find(entry => entry.id === language)
  if (custom?.line_comment) return { open: custom.line_comment }
  if (language === "html" || language === "xml") return { open: "<!--", close: "-->" }
  if (language === "css") return { open: "/*", close: "*/" }
  if (new Set(["bash", "python", "ruby", "nim", "toml", "yaml"]).has(language)) return { open: "#" }
  if (language === "lua" || language === "ori") return { open: "--" }
  if (language === "text") return undefined
  return { open: "//" }
}

export function lineComment(language: string, config: Config): string {
  return commentDelimiters(language, config)?.open ?? "//"
}
// A família Markdown, como o oráculo (`is_markdown_family`): só ela tem preview.
export function isMarkdownLanguage(language: string): boolean {
  return language === "markdown" || language === "mdx"
}

const workspaceRootCache = new Map<string, string>()
const imagePathCache = new Map<string, string | undefined>()

export function clearPreviewImageCache(): void {
  workspaceRootCache.clear()
  imagePathCache.clear()
}

// Resolve uma imagem do preview para um caminho local, ou recusa com undefined.
//
// Remoto nunca baixa sozinho; fora do workspace nunca sai; inexistente nunca
// estoura — cada recusa é silenciosa porque o markdown continua legível sem a
// figura, e um preview que falha por uma imagem quebrada mente sobre o texto.
export function resolvePreviewImage(href: string, docPath: string | undefined, workspace: string, imagesEnabled: boolean): string | undefined {
  if (!imagesEnabled) return undefined
  if (/^[a-z][a-z0-9+.-]*:/i.test(href)) return undefined
  const cacheKey = `${workspace}\0${docPath ?? ""}\0${href}`
  if (imagePathCache.has(cacheKey)) return imagePathCache.get(cacheKey)
  try {
    let root = workspaceRootCache.get(workspace)
    if (!root) {
      root = realpathSync(workspace)
      workspaceRootCache.set(workspace, root)
    }
    const path = realpathSync(resolve(dirname(docPath ?? join(workspace, "untitled.md")), href))
    const rel = relative(root, path)
    const result = (rel === ".." || rel.startsWith("../")) ? undefined : path
    imagePathCache.set(cacheKey, result)
    return result
  } catch {
    imagePathCache.set(cacheKey, undefined)
    return undefined
  }
}

export async function resolvePreviewImageAsync(href: string, docPath: string | undefined, workspace: string, imagesEnabled: boolean): Promise<string | undefined> {
  if (!imagesEnabled) return undefined
  if (/^[a-z][a-z0-9+.-]*:/i.test(href)) return undefined
  const cacheKey = `${workspace}\0${docPath ?? ""}\0${href}`
  if (imagePathCache.has(cacheKey)) return imagePathCache.get(cacheKey)
  try {
    let root = workspaceRootCache.get(workspace)
    if (!root) {
      root = await realpath(workspace)
      workspaceRootCache.set(workspace, root)
    }
    const path = await realpath(resolve(dirname(docPath ?? join(workspace, "untitled.md")), href))
    const rel = relative(root, path)
    const result = (rel === ".." || rel.startsWith("../")) ? undefined : path
    imagePathCache.set(cacheKey, result)
    return result
  } catch {
    imagePathCache.set(cacheKey, undefined)
    return undefined
  }
}
// Só http(s) selecionado abre no navegador, e sempre com confirmação no caller.
export function extractOpenableLink(selection: string): string | undefined {
  const url = selection.trim()
  return /^https?:\/\//i.test(url) ? url : undefined
}
let sharedHighlighter: Promise<Highlighter> | undefined
let loading: Promise<void> = Promise.resolve()
export type HighlightToken = { start: number; end: number; color?: string; fontStyle: number; role?: string }
export async function highlight(text: string, language: string, theme: string): Promise<HighlightToken[]> {
  if (language === "text") return []
  if (language === "ori") return highlightOri(text)
  const loadedLanguage = Object.keys(bundledLanguages).find((loaded): loaded is keyof typeof bundledLanguages => loaded === language)
  if (!loadedLanguage) return []
  const normalizedTheme = theme === "one-dark" || theme === "zed-dark" ? "one-dark-pro" : theme === "default-dark" || theme === "default" ? "github-dark" : theme
  const themes = ["github-dark", "github-light", "nord", "dracula", "monokai", "tokyo-night", "one-dark-pro", "catppuccin-mocha", "solarized-dark", "solarized-light"] as const
  const resolvedTheme = themes.find(theme => theme === normalizedTheme) ?? "github-dark"
  sharedHighlighter ??= createHighlighter({ themes: [resolvedTheme], langs: [loadedLanguage] })
  const engine = await sharedHighlighter
  const ready = loading.then(async () => {
    if (!engine.getLoadedThemes().includes(resolvedTheme)) await engine.loadTheme(resolvedTheme)
    if (!engine.getLoadedLanguages().includes(loadedLanguage)) await engine.loadLanguage(loadedLanguage)
  })
  loading = ready
  await ready
  // Shiki's native scanner needs one theme-only tokenization before requesting scopes.
  // Without this warm-up, a freshly loaded grammar can return a single invalid span.
  engine.codeToTokens(text, { lang: loadedLanguage, theme: resolvedTheme })
  const tokens = engine.codeToTokens(text, { lang: loadedLanguage, theme: resolvedTheme, includeExplanation: "scopeName" }).tokens
  return tokens.flatMap(line => line.map((token: ThemedToken) => ({ start: token.offset, end: token.offset + token.content.length, color: token.color, fontStyle: token.fontStyle ?? 0, role: syntaxRole(token.explanation?.flatMap(explanation => explanation.scopes.map(scope => scope.scopeName)) ?? []) })))
}
function syntaxRole(scopes: string[]): string | undefined {
  const scope = scopes.join(" ")
  const patterns: [RegExp, string][] = [
    [/comment/, "comment"], [/markup.heading/, "heading"], [/markup.bold/, "strong"], [/markup.italic/, "emphasis"], [/markup.quote/, "quote"], [/markup.list/, "list_marker"], [/markup.raw/, "code"], [/markup.underline.link/, "link"],
    [/string/, "string"], [/constant.numeric/, "number"], [/keyword.operator/, "operator"], [/keyword|storage.type|storage.modifier/, "keyword"], [/entity.name.function|support.function/, "function"], [/entity.name.type|support.type|entity.name.class/, "type_name"], [/entity.name.tag/, "tag"], [/entity.other.attribute-name/, "attribute"], [/variable.other.property|support.variable.property/, "property"], [/constant/, "constant"], [/variable/, "variable"], [/punctuation/, "punctuation"],
  ]
  return patterns.find(([pattern]) => pattern.test(scope))?.[1]
}
function highlightOri(text: string): HighlightToken[] {
  const expression = /\/\/[^\n]*|\/\*[\s\S]*?\*\/|"(?:\\.|[^"\\])*"|\b(?:fn|let|mut|if|else|for|while|return|kind|use|pub|match|struct|enum|impl|true|false|self|type|import)\b|\b\d+(?:\.\d+)?\b/g
  return [...text.matchAll(expression)].map(match => ({ start: match.index, end: match.index + match[0].length, color: match[0].startsWith("/") ? "#8b949e" : match[0].startsWith('"') ? "#a5d6ff" : /^\d/.test(match[0]) ? "#79c0ff" : "#ff7b72", fontStyle: 0 }))
}
