import { spawn, type ChildProcessWithoutNullStreams } from "node:child_process"
import { pathToFileURL } from "node:url"
import { createMessageConnection, StreamMessageReader, StreamMessageWriter, CancellationTokenSource, type MessageConnection } from "vscode-languageserver-protocol/node"
import { InitializeRequest, InitializedNotification, DidOpenTextDocumentNotification, DidChangeTextDocumentNotification, DidSaveTextDocumentNotification, DidCloseTextDocumentNotification, PublishDiagnosticsNotification, CompletionRequest, HoverRequest, DefinitionRequest, DocumentFormattingRequest, RenameRequest, ShutdownRequest, ExitNotification, type Diagnostic, type Position, type TextEdit, type WorkspaceEdit } from "vscode-languageserver-protocol/node"

export class LspClient {
  private process: ChildProcessWithoutNullStreams
  private connection: MessageConnection
  private opened = new Map<string, number>()
  private closed = false
  private stderr = ""
  readonly diagnostics = new Map<string, Diagnostic[]>()

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

  async initialize(): Promise<void> {
    await this.deadline(source => this.connection.sendRequest(InitializeRequest.type, {
      processId: process.pid, rootUri: pathToFileURL(this.workspace).href,
      capabilities: { general: { positionEncodings: ["utf-16"] }, textDocument: { synchronization: { didSave: true }, completion: { completionItem: { snippetSupport: false } }, hover: { contentFormat: ["markdown", "plaintext"] }, publishDiagnostics: {} } },
    }, source.token))
    await this.connection.sendNotification(InitializedNotification.type, {})
  }

  async sync(path: string, language: string, text: string, version: number): Promise<void> {
    const uri = pathToFileURL(path).href
    if (!this.opened.has(uri)) await this.connection.sendNotification(DidOpenTextDocumentNotification.type, { textDocument: { uri, languageId: language, version, text } })
    else if (this.opened.get(uri) !== version) await this.connection.sendNotification(DidChangeTextDocumentNotification.type, { textDocument: { uri, version }, contentChanges: [{ text }] })
    this.opened.set(uri, version)
  }
  saved(path: string): Promise<void> { return this.connection.sendNotification(DidSaveTextDocumentNotification.type, { textDocument: { uri: pathToFileURL(path).href } }) }
  async closeDocument(path: string): Promise<void> {
    const uri = pathToFileURL(path).href
    if (!this.opened.delete(uri)) return
    await this.connection.sendNotification(DidCloseTextDocumentNotification.type, { textDocument: { uri } })
  }
  completion(path: string, position: Position) { return this.deadline(source => this.connection.sendRequest(CompletionRequest.type, { textDocument: { uri: pathToFileURL(path).href }, position }, source.token)) }
  hover(path: string, position: Position) { return this.deadline(source => this.connection.sendRequest(HoverRequest.type, { textDocument: { uri: pathToFileURL(path).href }, position }, source.token)) }
  definition(path: string, position: Position) { return this.deadline(source => this.connection.sendRequest(DefinitionRequest.type, { textDocument: { uri: pathToFileURL(path).href }, position }, source.token)) }
  formatting(path: string, tabSize: number, insertSpaces: boolean): Promise<TextEdit[] | null> { return this.deadline(source => this.connection.sendRequest(DocumentFormattingRequest.type, { textDocument: { uri: pathToFileURL(path).href }, options: { tabSize, insertSpaces } }, source.token)) }
  rename(path: string, position: Position, newName: string): Promise<WorkspaceEdit | null> { return this.deadline(source => this.connection.sendRequest(RenameRequest.type, { textDocument: { uri: pathToFileURL(path).href }, position, newName }, source.token)) }
  get running(): boolean { return !this.closed }

  async close(): Promise<void> {
    if (!this.closed) {
      await this.deadline(source => this.connection.sendRequest(ShutdownRequest.type, source.token)).catch(() => undefined)
      await this.connection.sendNotification(ExitNotification.type).catch(() => undefined)
    }
    this.closed = true
    this.connection.dispose()
    if (this.process.exitCode === null) this.process.kill("SIGKILL")
  }
}
