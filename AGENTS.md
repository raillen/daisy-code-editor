# AGENTS.md — Daisy Code Editor

Guia para agentes que implementam o editor TUI **Daisy Code Editor**.

**Precedência:** este arquivo > skills Grok > defaults do modelo.

## Produto

| Conceito | Nome |
|----------|------|
| Editor | **Daisy Code Editor** |
| CLI | `daisy` |
| Config | TOML (`~/.config/daisy/`, `.daisy/`, compatível com `.oride/`) |
| Versão | SemVer (versão atual `0.2.0`) |

## Skills (obrigatórias neste repo)

| Skill | Quando |
|-------|--------|
| **`rust`** | Todo código (workspace, `Result`, clippy/fmt) |
| **`clean-code`** | Módulos, nomes, KISS, anti-primitivo |
| **`living-docs`** | README / `docs/` / comportamento user-facing |

**Nota:** Inteligência de linguagem vem via **LSP no PATH** (`rust-analyzer`, `clangd`, `ori-lsp`, etc.), mantendo o editor desacoplado de compiladores específicos.

Não misturar lógica de UI (`ratatui`) em `oride-core` / `oride-config` / `oride-keymap`.

## Invariantes

1. **Core sem UI** — `oride-core` testável sem TTY.
2. **Actions como dados** — keymaps e palette disparam ações nomeadas.
3. **Fail closed** — LSP/Git/PTY falham com status, não crash.
4. **LSP via PATH** — spawn desacoplado via binários no PATH, sem dependência estática de compiladores.
5. **Um conceito por PR** — seguir fases em `docs/design.md`.

## Validação

```bash
bun run typecheck
bun test
bun run build
```

Para scripts do repo:
```bash
bash -n scripts/install.sh
node? não se aplica
```

## Lições absorvidas das crafting apps (daisy da storytold)

Diretrizes que passam a valer para este repositório:

1. **Nunca crashar (outranks feature work).**
   - Entrada malformada, parâmetros LSP, arquivo não existente, resposta inesperada ou cwd inexistente deve virar status/erro estruturado, não exceção não capturada.
   - Não usar `throw` em caminhos de comando sem `guard()`. Todo `run()` de comando deve capturar erros e chamar `app.status = ...`.
   - Validar índices (bounds, divisão, aritmética) antes de uso, strings char boundaries, alocações capadas (já temos 32 MiB por arquivo, 100k arquivos).
   - Recursão (detecção cyclic in FileTree, session paths) com limite/seen-set.
   - Guardas em pontos de entrada: `app.changed()`, `app.guard()`, LSP close, terminal close.

2. **Tudo é uma ação nomeada (everything is a command).**
   - Comportamento user-facing deve ser registrado em `commands.ts` (command id, label, menu).
   - Palette, keymap e menus devem despachar o mesmo ID: nada de lógica exclusiva na UI.
   - Programmatic/API-like calls (control/MCP futuros) devem apenas invocar o mesmo ID, nunca abrir dialogs.

3. **UI fina e dados-driveis.**
   - Workbench/dev containers thin: leem estado e chamam `app.run()`/`app.changed()`.
   - Tokens de temas/cores devem vir de `theme.ts`, não hardcode.
   - Editor render deve continuar separado de negócio (`editor.tsx`).

4. **Camadas (layering).**
   - `core/*` não pode depender de `services`, `ui`, `app`.
   - `services/*` não pode depender de UI.
   - App UI/runtime depende de core/service, não vice-versa.
   - Se adicionar novo módulo em app, manter `commands.ts` como composition.

5. **Repro & verify visually.**
   - TUI não possui screenshot automático (no Tauri), mas mudanças de rendering devem ser demonstradas através de frame print (`--print-frame`) ou testes.
   - Mudanças de keymap exigem doc/tests (doc-drift pattern).
   - Mudanças de layout devem rodar workbench render e validar terminal.

6. **Performance é feature.**
   - Preferir debounce (200ms LSP), adaptative polling, avoid full O(n²).
   - Fuzzy offload em worker.
   - Pintar terminal/search/file list sem scanner completo a cada frame.

7. **Progress docs.**
   - Atualizar `ROADMAP.md` / `docs/` se uma métrica melhora.
   - Registrar commit com ID de tarefa.
   - `log/devlog.md` se criado, para continuidade.

8. **Clean-room & assets.**
   - Não copiar código third-party.
   - Assets com proveniência documentada.
   - Fonts externos ficam fora do repo (exception already via system + fallback).
   - Não adicionar pacote sem justificativa/provenance.

9. **Agentes paralelos.**
   - Cada agente seu diretório target.
   - Arquivos manifest (`package.json`, tsconfig) alterados atomicamente.
   - Remover artefatos temporários a cada finalização.

## Fases (resumo)

| Fase | Foco |
|------|------|
| P0 | core + TUI + config/keymap/theme |
| P1 | tabs, árvore, terminal, palette (alpha.2) |
| P2 | tree-sitter + Markdown rico (alpha.3) |
| P3 | LSP engine & tooling |
| P4 | **polimento 0.2** (help, find, clipboard, session, i18n, …) |
| P5 | robustez + performance profiling (history O(n), polling, incremental LSP, snippets) |
| P6 | multi-cursor UX + DAP + release automation (roadmap future) |

Markdown futuro (preview, fence injections, MDX JSX): `docs/guides/pt/markdown.md` — **não** misturar no P3/P4.

## Fases (resumo)

| Fase | Foco |
|------|------|
| P0 | core + TUI + config/keymap/theme |
| P1 | tabs, árvore, terminal, palette (alpha.2) |
| P2 | tree-sitter + Markdown rico (alpha.3) |
| P3 | LSP engine & tooling |
| P4 | **polimento 0.2** (help, find, clipboard, session, i18n, …) |

Markdown futuro (preview, fence injections, MDX JSX): `docs/guides/pt/markdown.md` — **não** misturar no P3/P4.
