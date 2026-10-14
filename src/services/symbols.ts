import { fuzzyScore } from "../core/search"
import { lineRanges } from "../core/document"

export type DocumentSymbol = {
  name: string
  kind: "function" | "type" | "class" | "interface" | "constant"
  signature: string
  doc?: string
  line: number
}

// Extrai todas as palavras/identificadores presentes nos buffers abertos
export function extractBufferWords(documents: { text: string }[]): Set<string> {
  const words = new Set<string>()
  for (const doc of documents) {
    const matches = doc.text.match(/[\p{L}\p{N}_]+/gu)
    if (matches) {
      for (const word of matches) {
        if (word.length >= 2 && /[_\p{L}]/u.test(word)) words.add(word)
      }
    }
  }
  return words
}

// Filtra e classifica palavras do buffer com correspondência por prefixo ou fuzzy
export function suggestBufferWords(
  prefix: string,
  words: Set<string>,
  limit = 50
): string[] {
  if (!prefix) return []
  const exact: string[] = []
  const fuzzy: { word: string; score: number }[] = []
  const lowerPrefix = prefix.toLowerCase()

  for (const word of words) {
    if (word === prefix) continue
    if (word.startsWith(prefix)) {
      exact.push(word)
    } else if (word.toLowerCase().startsWith(lowerPrefix)) {
      exact.push(word)
    } else if (lowerPrefix.length >= 3) {
      const score = fuzzyScore(word, prefix)
      if (score > -Infinity) {
        fuzzy.push({ word, score })
      }
    }
  }

  exact.sort((a, b) => a.length - b.length || a.localeCompare(b))
  fuzzy.sort((a, b) => b.score - a.score)

  return [...exact, ...fuzzy.map(item => item.word)].slice(0, limit)
}

// Identifica a palavra ou símbolo sob o offset (suporta caracteres Unicode e macros como println!)
export function wordAtOffset(text: string, offset: number): string {
  if (!text || offset < 0) return ""
  const clamped = Math.min(offset, text.length)
  let start = clamped
  // Se o cursor estiver exatamente no fim da palavra ou sobre pontuação, tenta recuar um caractere
  if (start > 0 && !/[\p{L}\p{N}_!]/u.test(text[start] ?? "") && /[\p{L}\p{N}_!]/u.test(text[start - 1] ?? "")) {
    start--
  }
  while (start > 0 && /[\p{L}\p{N}_!]/u.test(text[start - 1]!)) {
    start--
  }
  let end = start
  while (end < text.length && /[\p{L}\p{N}_!]/u.test(text[end]!)) {
    end++
  }
  return text.slice(start, end).trim()
}

// Extrai símbolos do documento e seus comentários precedentes imediatos (///, //, #, /** */)
export function extractDocumentSymbols(text: string): DocumentSymbol[] {
  const symbols: DocumentSymbol[] = []
  const lines = lineRanges(text).map(range => text.slice(range.start, range.end).replace(/\r?\n$/, ""))
  let docLines: string[] = []

  for (let index = 0; index < lines.length; index++) {
    const rawLine = lines[index] ?? ""
    const trimmed = rawLine.trim()

    // Comentários: acumula linhas precedentes
    if (trimmed.startsWith("///")) {
      docLines.push(trimmed.slice(3).trim())
      continue
    }
    if (trimmed.startsWith("//") && !trimmed.startsWith("////")) {
      docLines.push(trimmed.slice(2).trim())
      continue
    }
    if (trimmed.startsWith("#") && !trimmed.startsWith("#!")) {
      docLines.push(trimmed.slice(1).trim())
      continue
    }
    if (trimmed.startsWith("/**")) {
      const inline = trimmed.replace(/^\/\*\*/, "").replace(/\*\/$/, "").trim()
      if (inline) docLines.push(inline)
      continue
    }
    if (trimmed.startsWith("*") && !trimmed.startsWith("*/")) {
      docLines.push(trimmed.slice(1).trim())
      continue
    }
    if (trimmed === "" || trimmed === "*/") {
      // Linha vazia ou fim de bloco sem declaração: se tiver acumulação antiga sem declaração imediata, limpa
      if (trimmed === "") docLines = []
      continue
    }

    // 1. Funções / Métodos convencionais:
    // Rust: (pub )?(async )?fn name(args) -> Ret
    // Python: (async )?def name(args):
    // JS/TS: (export )?(async )?function name(args)
    // Go: func (recv)? name(args)
    const funcMatch = trimmed.match(
      /^(?:(?:export|pub)\s+)?(?:async\s+)?(?:function|fn|def)\s+([a-zA-Z0-9_]+)\s*(?:<[^>]*>)?\s*\(([^)]*)\)(?:\s*->\s*([^{;:]+))?/
    )
    if (funcMatch) {
      const name = funcMatch[1]!
      const args = funcMatch[2] ?? ""
      const ret = funcMatch[3] ? ` -> ${funcMatch[3].trim()}` : ""
      const doc = docLines.length ? docLines.join("\n") : undefined
      symbols.push({
        name,
        kind: "function",
        signature: `${name}(${args})${ret}`,
        doc,
        line: index,
      })
      docLines = []
      continue
    }

    // Go func (com ou sem receiver)
    const goFuncMatch = trimmed.match(
      /^func\s+(?:\([^)]*\)\s+)?([a-zA-Z0-9_]+)\s*\(([^)]*)\)(?:\s*([^{]+))?/
    )
    if (goFuncMatch) {
      const name = goFuncMatch[1]!
      const args = goFuncMatch[2] ?? ""
      const ret = goFuncMatch[3] ? ` ${goFuncMatch[3].trim()}` : ""
      const doc = docLines.length ? docLines.join("\n") : undefined
      symbols.push({
        name,
        kind: "function",
        signature: `func ${name}(${args})${ret}`,
        doc,
        line: index,
      })
      docLines = []
      continue
    }

    // JS/TS Arrow Function atribuída a const/let:
    // const name = (args) => ... ou const name = async (args) => ...
    const arrowMatch = trimmed.match(
      /^(?:export\s+)?(?:const|let)\s+([a-zA-Z0-9_]+)\s*=\s*(?:async\s+)?\(([^)]*)\)\s*(?::\s*([^=]+))?\s*=>/
    )
    if (arrowMatch) {
      const name = arrowMatch[1]!
      const args = arrowMatch[2] ?? ""
      const ret = arrowMatch[3] ? `: ${arrowMatch[3].trim()}` : ""
      const doc = docLines.length ? docLines.join("\n") : undefined
      symbols.push({
        name,
        kind: "function",
        signature: `${name}(${args})${ret}`,
        doc,
        line: index,
      })
      docLines = []
      continue
    }

    // 2. Tipos / Classes / Interfaces / Structs / Enums:
    const typeMatch = trimmed.match(
      /^(?:(?:export|pub)\s+)?(class|struct|interface|type|enum)\s+([a-zA-Z0-9_]+)/
    )
    if (typeMatch) {
      const kindStr = typeMatch[1]!
      const name = typeMatch[2]!
      const kind = kindStr === "class" ? "class" : kindStr === "interface" ? "interface" : "type"
      const doc = docLines.length ? docLines.join("\n") : undefined
      symbols.push({
        name,
        kind,
        signature: `${kindStr} ${name}`,
        doc,
        line: index,
      })
      docLines = []
      continue
    }

    // Não foi declaração: limpa doc acumulado
    docLines = []
  }

  return symbols
}

// Procura símbolo pelo nome nos documentos abertos
export function findSymbol(
  name: string,
  documents: { text: string; path?: string }[]
): DocumentSymbol | undefined {
  if (!name) return undefined
  for (const doc of documents) {
    const symbols = extractDocumentSymbols(doc.text)
    const match = symbols.find(s => s.name === name)
    if (match) return match
  }
  return undefined
}
