import { createMessageConnection, StreamMessageReader, StreamMessageWriter } from "vscode-languageserver-protocol/node"
const connection = createMessageConnection(new StreamMessageReader(process.stdin), new StreamMessageWriter(process.stdout))
const changeCapture = process.argv[2] as string | undefined
let uri = "file:///test.ts"
connection.onRequest("initialize", () => ({ capabilities: { textDocumentSync: 1, completionProvider: {}, hoverProvider: true, definitionProvider: true, documentFormattingProvider: true, referencesProvider: true, signatureHelpProvider: {}, codeActionProvider: true } }))
connection.onNotification("textDocument/didOpen", (params: { textDocument: { uri: string } }) => {
  uri = params.textDocument.uri
  void connection.sendNotification("textDocument/publishDiagnostics", { uri, diagnostics: [{ range: { start: { line: 0, character: 0 }, end: { line: 0, character: 1 } }, severity: 2, message: "fixture warning" }] })
})
connection.onNotification("textDocument/didChange", (params: unknown) => {
  if (changeCapture) void Bun.write(changeCapture, JSON.stringify(params))
})
connection.onRequest("textDocument/completion", () => [{ label: "foobar", insertText: "foobar" }])
connection.onRequest("textDocument/hover", () => ({ contents: { kind: "plaintext", value: "hover from fixture" } }))
connection.onRequest("textDocument/definition", () => ({ uri, range: { start: { line: 0, character: 0 }, end: { line: 0, character: 1 } } }))
connection.onRequest("textDocument/formatting", () => [{ range: { start: { line: 0, character: 0 }, end: { line: 0, character: 13 } }, newText: "formatted" }])
connection.onRequest("textDocument/references", () => [{ uri, range: { start: { line: 0, character: 6 }, end: { line: 0, character: 9 } } }])
connection.onRequest("textDocument/signatureHelp", () => ({ signatures: [{ label: "foobar(arg: string): void", documentation: "fixture signature", parameters: [{ label: "arg" }] }], activeSignature: 0, activeParameter: 0 }))
connection.onRequest("textDocument/codeAction", () => [{ title: "Fixture fix", kind: "quickfix", edit: { changes: { [uri]: [{ range: { start: { line: 0, character: 0 }, end: { line: 0, character: 5 } }, newText: "fixed" }] } } }])
connection.onRequest("shutdown", () => null)
connection.onNotification("exit", () => { connection.dispose(); process.exit(0) })
connection.listen()
