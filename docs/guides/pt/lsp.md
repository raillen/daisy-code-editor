# LSP no Daisy Code Editor

## 0. Escopo (anti-bloat)

Este documento define os requisitos mínimos para adicionar LSPs sem tornar o TUI um editor pesado. Regra central: Tudo é plugin:nenhum servidor é startup do binário, e nenhum centro de funcionalidade avançada é carregado por padrão.

## 1. Objetivos primários

- Erros estruturados, nunca crashes.
- Tudo configurável por workspace (`lsp.servers` por linguagem) e por usuário.
- Cada linguagem carrega o LSP somente quando o documento ativo é daquela linguagem e o editor recebe um comando que necessite dele.
- Cada servidor possui lifecycle isolado (connect / sync / close), sem bloquear UI.
- Performance do TUI não degrada para arquivos de 1MB ou mais.
- O editor continua funcionando sem LSPs quando não configurados.

## 2. Componentes (camadas)

- `core/*` — funções puras de transformação (`processAt`, `offsetAt`, `TextEdit` -> `Document`), nenhuma dependência de rede.
- `services/lsp.ts` — `LspClient`, sem estado global, apenas spawn IPC + JSON-RPC. Sem bibliotecas pesadas (wrapper VSCode esquece 1MB já é ok).
- `app/application.ts` — Orquestra servers, timers e settings. LSP não pode viver no core.
- `markers` / `messages` (UI) — Renderiza diagnostics/hover, mas não chama comandos.

## 3. Lifecycles & timing

- `debounce` (default 200ms) para sync + diagnostics.
- `initialize` on first `didOpen` e `didChange` full document (bounded).
- `closeDocument` on tab close or workspace switch (no zombie).
- `sync()` com `version` incrementado tem que re-check changed state (missed sync is not allowed).
- `pun` (cleanup) with SIGTERM fallback, timeout.

## 4. Capability boundaries (anti-bloat)

- `textDocument/synchronization` — full (basic) or optional incremental prior sync.
- `textDocument/diagnostics` — zero LSP cost; only drained updates.
- `textDocument/hover` — on demand.
- `textDocument/completion` — on demand; limitado (default 100 items).
- `textDocument/formatting` — on demand + explicit.
- `textDocument/definition` — on demand.
- `textDocument/rename`, `codeAction`, `inlayHints` — opt-in.
- Recursos recursivos (references, workspace symbols) apenas se config de workspace setup.

## 5. Isolamento / segurança

- Nenhum comando shell interpolado: sempre array de argv.
- Arquivo aberto deve corresponder ao URI LSP.
- Language boundaries: paths são vinculados ao workspace.
- Adiciona guard em cada chamada JSON-RPC para `closed`.
- Limitar output de stderr (4MiB).

## 6. Performance & resiliência

- `document.version` + `text` only in change.
- No major allocations for huge files (parser will have size limit of 32MB).
- Timeout com CancellationToken (default 10s).
- Backoff/retry policy: failed `ensureLsp()` entry stored in `serverFailures`.
- Recomeço on retry invalida failure cache.

## 7. UX

- Diagnostics appear in status (or Problems panel).
- Hover content truncate (max 8KB).
- Completion dropdown no flicker.
- Formatting applies only with elected style.
- `lsp.rename` safe for multi-buffer (WorkspaceEdit supported, changed locally).
- If no server, command gracefully reports.

## 8. Minimal e2e tests (against fixture LSP)

- `server started` spawns executable mock.
- `didOpen` sent once.
- `didChange` called on timeout debounce.
- `closeDocument` (verified).
- `completion` respects cancellation.
- `formatting` with version conflict.
- `definition` returning invalid position handled.
- `panic`/crashed server doesn't crash editor.

## 9. Alternativas (não fazer)

- Carragar servidor de forma eager (started at boot).
- Sempre habilitar features pesadas (`workspaceSymbol`, `codeLens`).
- Usar LSP pool por processo global e não por linguagem.
- Terceirizar render do output de JSON parsing fora de bounds.
- Ter mudanças completas (incremental patch) sem validar invariantes.

## 10. Decisões pendentes (a discutir)

- Incremental change ranges vs full-text (manter full-text default, only diffs for > 1MB?).
- Limitar language discovery (auto detection by grammar vs only configured).
- Habilitar code-action / hover updates via LSP settings in `.daisy/config.toml`.
- Configuração de timeout via `context` (já existe).

---

**Compatibilidade**: este design reflete o estado atual sem bloat. Qualquer alteração deve passar por `bun run typecheck`, `bun test`, e benchmarks de large file (50k lines) antes de merge.