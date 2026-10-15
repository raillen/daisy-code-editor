import { spawn, type ChildProcessWithoutNullStreams } from "node:child_process"
import { pathToFileURL } from "node:url"
import { positionAt } from "../core/document"
import { createMessageConnection, StreamMessageReader, StreamMessageWriter, CancellationTokenSource, type MessageConnection } from "vscode-languageserver-protocol/node"
import { InitializeRequest, InitializedNotification, DidOpenTextDocumentNotification, DidChangeTextDocumentNotification, DidSaveTextDocumentNotification, DidCloseTextDocumentNotification, PublishDiagnosticsNotification, CompletionRequest, CompletionResolveRequest, HoverRequest, DefinitionRequest, ReferencesRequest, SignatureHelpRequest, CodeActionRequest, DocumentFormattingRequest, RenameRequest, DocumentSymbolRequest, ShutdownRequest, ExitNotification, type Diagnostic, type Position, type Range, type TextEdit, type WorkspaceEdit, type DocumentSymbol, type SymbolInformation, type SignatureHelp, type CodeAction, type Command, type ServerCapabilities, type CompletionItem } from "vscode-languageserver-protocol/node"

export type { CodeAction, Command as LspCommand, SignatureHelp, ServerCapabilities, CompletionItem as LspCompletionItem } from "vscode-languageserver-protocol/node"

// Minimal single-span diff (offsets in UTF-16 code units, matching the
// negotiated `utf-16` position encoding). Returns null when texts match.
export function incrementalEdit(previous: string, next: string): { start: number; end: number; text: string } | null {
  if (previous === next) return null
  let start = 0
  const minLength = Math.min(previous.length, next.length)
  while (start < minLength && previous.charCodeAt(start) === next.charCodeAt(start)) start++
  let oldEnd = previous.length, newEnd = next.length
  while (oldEnd > start && newEnd > start && previous.charCodeAt(oldEnd - 1) === next.charCodeAt(newEnd - 1)) { oldEnd--; newEnd-- }
  return { start, end: oldEnd, text: next.slice(start, newEnd) }
}

export function lspBackoffDelay(consecutiveFailures: number): number {
  if (!Number.isFinite(consecutiveFailures) || consecutiveFailures < 0) return 1000
  return Math.min(30000, 1000 * 2 ** Math.min(10, Math.floor(consecutiveFailures)))
}

export function lspRetryAllowed(lastFailureAt: number, consecutiveFailures: number, now = Date.now()): boolean {
  return now - lastFailureAt >= lspBackoffDelay(consecutiveFailures)
}

export class LspClient {
  private process: ChildProcessWithoutNullStreams
  private connection: MessageConnection
  private opened = new Map<string, number>()
  private lastText = new Map<string, string>()
  private closed = false
  private stderr = ""
  readonly diagnostics = new Map<string, Diagnostic[]>()
  readonly serverCapabilities: ServerCapabilities = {}

  constructor(readonly command: readonly string[], readonly workspace: string, readonly timeout: number, readonly onChange: () => void) {
    const executable = command[0]
    if (!executable) throw new Error("Empty LSP command")
    if (!Bun.which(executable)) throw new Error(`LSP executable not found on PATH: ${executable}`)
    this.process = spawn(executable, command.slice(1), { cwd: workspace, stdio: "pipe" })
    this.connection = createMessageConnection(new StreamMessageReader(this.process.stdout), new StreamMessageWriter(this.process.stdin))
    this.process.stderr.on("data", (chunk: Buffer) => { this.stderr = (this.stderr + chunk.toString()).slice(-4096) })
    this.process.stdin.on("error", () => { this.closed = true; this.connection.dispose(); onChange() })
    this.process.on("error", () => { this.closed = true; this.connection.dispose(); onChange() })
    this.process.on("exit", () => { this.closed = true; this.connection.dispose(); onChange() })
    this.connection.onNotification(PublishDiagnosticsNotification.type, params => { this.diagnostics.set(params.uri, params.diagnostics); onChange() })
    this.connection.onRequest("workspace/configuration", (params: { items?: unknown[] }) => (params.items ?? []).map(() => null))
    this.connection.onRequest("client/registerCapability", () => null)
    this.connection.onRequest("workspace/workspaceFolders", () => [{ uri: pathToFileURL(workspace).href, name: workspace }])
    this.connection.listen()
  }

  private async deadline<T>(operation: (token: CancellationTokenSource) => Promise<T>): Promise<T> {
    if (this.closed) throw new Error(this.stderr || "LSP server exited")
    const cancellation = new CancellationTokenSource()
    let timer: ReturnType<typeof setTimeout> | undefined
    try {
      return await Promise.race([operation(cancellation), new Promise<never>((_, reject) => {
        timer = setTimeout(() => { cancellation.cancel(); reject(new Error(`LSP request timed out (${this.command[0]})`)) }, this.timeout)
      })])
    } finally { clearTimeout(timer); cancellation.dispose() }
  }

  supports(provider: keyof ServerCapabilities): boolean {
    return this.serverCapabilities[provider] !== undefined && this.serverCapabilities[provider] !== false && this.serverCapabilities[provider] !== null
  }

  async initialize(): Promise<void> {
    const result = await this.deadline(source => this.connection.sendRequest(InitializeRequest.type, {
      processId: process.pid, rootUri: pathToFileURL(this.workspace).href,
      capabilities: { general: { positionEncodings: ["utf-16"] }, textDocument: { synchronization: { didSave: true }, completion: { completionItem: { snippetSupport: false } }, hover: { contentFormat: ["markdown", "plaintext"] }, publishDiagnostics: {} } },
    }, source.token)) as { capabilities?: ServerCapabilities } | undefined
    Object.assign(this.serverCapabilities, result?.capabilities ?? {})
    await this.connection.sendNotification(InitializedNotification.type, {})
  }

  async sync(path: string, language: string, text: string, version: number): Promise<void> {
    const uri = pathToFileURL(path).href
    if (!this.opened.has(uri)) {
      await this.connection.sendNotification(DidOpenTextDocumentNotification.type, { textDocument: { uri, languageId: language, version, text } })
      this.lastText.set(uri, text)
    } else if (this.opened.get(uri) !== version) {
      const edit = this.lastText.has(uri) ? incrementalEdit(this.lastText.get(uri)!, text) : null
      if (edit) {
        const previous = this.lastText.get(uri)!
        await this.connection.sendNotification(DidChangeTextDocumentNotification.type, { textDocument: { uri, version }, contentChanges: [{ range: { start: positionAt(previous, edit.start), end: positionAt(previous, edit.end) }, text: edit.text }] })
      } else {
        await this.connection.sendNotification(DidChangeTextDocumentNotification.type, { textDocument: { uri, version }, contentChanges: [{ text }] })
      }
      this.lastText.set(uri, text)
    }
    this.opened.set(uri, version)
  }
  saved(path: string): Promise<void> { return this.connection.sendNotification(DidSaveTextDocumentNotification.type, { textDocument: { uri: pathToFileURL(path).href } }) }
  async closeDocument(path: string): Promise<void> {
    const uri = pathToFileURL(path).href
    if (!this.opened.delete(uri)) return
    this.lastText.delete(uri)
    await this.connection.sendNotification(DidCloseTextDocumentNotification.type, { textDocument: { uri } })
  }
  completion(path: string, position: Position) { return this.deadline(source => this.connection.sendRequest(CompletionRequest.type, { textDocument: { uri: pathToFileURL(path).href }, position }, source.token)) }
  resolveCompletion(item: CompletionItem): Promise<CompletionItem> { return this.deadline(source => this.connection.sendRequest(CompletionResolveRequest.type, item, source.token)) }
  hover(path: string, position: Position) { return this.deadline(source => this.connection.sendRequest(HoverRequest.type, { textDocument: { uri: pathToFileURL(path).href }, position }, source.token)) }
  definition(path: string, position: Position) { return this.deadline(source => this.connection.sendRequest(DefinitionRequest.type, { textDocument: { uri: pathToFileURL(path).href }, position }, source.token)) }
  references(path: string, position: Position, includeDeclaration = false) { return this.deadline(source => this.connection.sendRequest(ReferencesRequest.type, { textDocument: { uri: pathToFileURL(path).href }, position, context: { includeDeclaration } }, source.token)) }
  signatureHelp(path: string, position: Position): Promise<SignatureHelp | null> { return this.deadline(source => this.connection.sendRequest(SignatureHelpRequest.type, { textDocument: { uri: pathToFileURL(path).href }, position }, source.token)) }
  codeAction(path: string, range: Range, diagnostics: Diagnostic[]): Promise<Array<CodeAction | Command> | null> { return this.deadline(source => this.connection.sendRequest(CodeActionRequest.type, { textDocument: { uri: pathToFileURL(path).href }, range, context: { diagnostics } }, source.token)) }
  formatting(path: string, tabSize: number, insertSpaces: boolean): Promise<TextEdit[] | null> { return this.deadline(source => this.connection.sendRequest(DocumentFormattingRequest.type, { textDocument: { uri: pathToFileURL(path).href }, options: { tabSize, insertSpaces } }, source.token)) }
  rename(path: string, position: Position, newName: string): Promise<WorkspaceEdit | null> { return this.deadline(source => this.connection.sendRequest(RenameRequest.type, { textDocument: { uri: pathToFileURL(path).href }, position, newName }, source.token)) }
  documentSymbols(path: string): Promise<Array<DocumentSymbol | SymbolInformation> | null> { return this.deadline(source => this.connection.sendRequest(DocumentSymbolRequest.type, { textDocument: { uri: pathToFileURL(path).href } }, source.token)) }
  get running(): boolean { return !this.closed }

  async close(): Promise<void> {
    if (!this.closed) {
      await this.deadline(source => this.connection.sendRequest(ShutdownRequest.type, source.token)).catch(() => undefined)
      await this.connection.sendNotification(ExitNotification.type).catch(() => undefined)
    }
    this.closed = true
    this.connection.dispose()
    if (this.process.exitCode === null) this.process.kill("SIGKILL")
    this.lastText.clear()
  }
}

export type { DocumentSymbol, SymbolInformation } from "vscode-languageserver-protocol"
