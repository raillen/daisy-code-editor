import { join, basename, extname } from "node:path"
import { readdir, readFile } from "node:fs/promises"
import { readToml, userConfigDir } from "./config"

// Suporte a templates no estilo do catálogo interno (cada template pode usar
// "$0" como cursor final). Novo loader externo permite que um mesmo workspace
// defina snippets por linguagem sem recompilar o editor.
export type UserSnippet = { trigger: string; body: string; description?: string }

async function readJsonSnippets(path: string): Promise<UserSnippet[]> {
  try {
    const source = JSON.parse(await readFile(path, "utf8")) as unknown
    return normalizeSnippetSource(source, basename(path))
  } catch {
    return []
  }
}

async function readTomlSnippets(path: string): Promise<UserSnippet[]> {
  try {
    const source = await readToml(path)
    return normalizeSnippetSource(source, basename(path))
  } catch {
    return []
  }
}

function normalizeSnippetSource(source: unknown, origin: string): UserSnippet[] {
  const collected: UserSnippet[] = []
  const candidate = (raw: unknown, fallbackTrigger: string): void => {
    const entry = typeof raw === "object" && raw !== null ? raw as Record<string, unknown> : {}
    const trigger = typeof entry.trigger === "string" ? entry.trigger.trim() : (typeof entry.prefix === "string" ? entry.prefix.trim() : fallbackTrigger)
    const body = typeof entry.body === "string" ? entry.body : Array.isArray(entry.body) ? entry.body.join("\n") : typeof entry.snippet === "string" ? entry.snippet : undefined
    if (trigger && body) collected.push({ trigger, body, description: typeof entry.description === "string" ? entry.description : origin })
  }

  if (Array.isArray(source)) {
    source.forEach((entry, index) => candidate(entry, `snippet_${index + 1}`))
    return collected
  }
  if (typeof source === "object" && source !== null) {
    const record = source as Record<string, unknown>
    if (Array.isArray(record.snippets)) record.snippets.forEach((entry, index) => candidate(entry, `snippet_${index + 1}`))
    for (const [key, value] of Object.entries(record)) {
      if (key === "snippets") continue
      if (typeof value === "string") candidate({ trigger: key, body: value }, key)
      else candidate(value, key)
    }
  }
  return collected
}

export async function loadUserSnippets(workspace: string, language: string): Promise<UserSnippet[]> {
  const directories = [
    join(workspace, ".daisy", "snippets"),
    join(workspace, ".daisy", "snippets", language),
    join(workspace, ".oride", "snippets"),
    join(userConfigDir, "snippets"),
    join(userConfigDir, "snippets", language),
  ]
  const found: UserSnippet[] = []
  for (const directory of directories) {
    const entries = await readdir(directory, { withFileTypes: true }).catch(() => [])
    for (const entry of entries) {
      if (!entry.isFile()) continue
      const path = join(directory, entry.name)
      const extension = extname(entry.name).toLowerCase()
      if (extension === ".json") found.push(...await readJsonSnippets(path))
      else if (extension === ".toml") found.push(...await readTomlSnippets(path))
      else if (extension === ".snippet") found.push(...await readJsonSnippets(path))
    }
  }
  return found
}