# Migração para OpenTUI + Solid

**Estado: implementação funcional em TypeScript/Bun; paridade completa ainda não certificada.**

Branch: `feat/opentui-solid`. Worktree: `../oride-opentui-solid`.
Rust (`crates/`) e Go (`cmd/`, `internal/`) permanecem intactos como referências.
O runtime novo não chama nem compila código Go/Rust do produto.

## Executar

```bash
cd ../oride-opentui-solid
bun install --frozen-lockfile
bun run start                       # workspace atual e sessão anterior
bun run start caminho/pasta         # abre um workspace
bun run start caminho/arquivo       # abre um arquivo
bun run start --print-frame 100x30   # captura renderer nativo sem TTY
bun run build
bun dist/cli.js caminho/arquivo
```

Requisitos verificados: **Linux x64, Bun 1.4.2**, OpenTUI/Core e Solid `0.6.1`,
Solid JS exatamente `1.9.12`. O bundle em `dist/cli.js` ainda precisa do Bun e
`node_modules`; **não é um executável standalone**. Não há release/deploy novo.
A PTY nativa do Bun é POSIX; Windows não foi validado.

## Arquitetura

| Pasta | Responsabilidade |
|---|---|
| `src/core/` | Documento, seleção, multicursor, histórico, busca, ids e keymap; sem OpenTUI |
| `src/services/` | TOML, filesystem, Git, PTY/VT, highlight, LSP, plugins e preferências |
| `src/app/` | Estado headless, registro de comandos e roteamento de entrada |
| `src/ui/` | Composição Solid e pintura OpenTUI |
| `src/cli.tsx` | Argumentos, sessão e ciclo de vida do terminal |

Menus, palette e atalhos usam o **mesmo registro de 96 ações**.
Os **98 bindings** são gerados da referência por `scripts/migrate-keymap.ts`.
Os catálogos TOML e temas embutidos são gerados por `scripts/migrate-catalogs.ts`.

OpenTUI faz layout e renderização. O documento TypeScript é dono do texto,
seleções e histórico: não há um segundo buffer `textarea` competindo com ele.
Offsets internos são **UTF-16**, convertidos em bytes no relatório diferencial.
Movimentos/apagamentos respeitam grafemas; o LSP anuncia UTF-16.

## Funcionalidades exercitadas

| Área | Implementação e evidência |
|---|---|
| Edição | Texto, paste bracketed, seleção, undo/redo, salvar/salvar como, todas as abas, recarga; testes de núcleo, UI e CLI real |
| Multicursor | Acima/abaixo, seleção de ocorrências, Alt+clique, edição transacional; texto e histórico testados |
| Mouse | Clique, seleção por arrasto, double-click de palavra, árvore, abas, menus, roda e divisores; tests OpenTUI |
| Highlight | Shiki/TextMate para linguagens de referência; Ori usa fallback léxico; gramáticas carregadas sob demanda, cache por versão/caminho/tema |
| Workbench | Menubar, tabs, splits verticais/horizontais, linha atual, números, soft wrap e scroll horizontal |
| Árvore | Expandir, criar exclusivamente, renomear, remover com confirmação, copiar caminho; sem atravessar workspace na criação |
| Finder | Busca fuzzy, arquivos do workspace, buffers e palette; Ctrl+P → filtro → Enter provado pela UI |
| Busca | Literal/regex, case, acentos, palavra inteira, globs; substituir no buffer/projeto; não sobrescreve buffers sujos |
| Git | Porcelain `-z`, branch/ahead/behind, stage/unstage, commit, diff, pull/push com confirmação; repositório temporário real |
| Terminal | PTY nativa, emulação VT xterm, cores, resize, scrollback de 2.000 linhas, Ctrl+C, processo em sessão própria |
| LSP | Por PATH, stdio JSON-RPC, start sob demanda, sync, completion, hover, definition, format e diagnostics; servidor fixture real |
| Markdown | Preview nativo só para a família Markdown/MDX (fora dela, status em vez de abrir), tabelas, fences, imagem local opt-in via Kitty/Sixel/blocks, URL selecionada → Alt+Enter com confirmação |
| Config | Default → usuário → projeto, TOML; EditorConfig; temas e idiomas externos, escolha persistida |
| Sessão | Mesmo nome FNV-1a e schema TOML para arquivos/aba ativa/árvore/SCM, splits (orientação, arquivo secundário, ratio %) e scroll_y; salvar atômico com 100% de compatibilidade |
| Vim | Normal/insert/visual/visual-line, h/j/k/l, w/b/e, gg/G, dd/yy, p/P, i/a/I/A/o/O, :w/:q/:wq/:tasks/:health |
| Tasks/plugins | Tasks TOML no shell com confirmação; plugins como processos/argv, confiança por sessão; sem Lua no processo |
| Disco | Poll de arquivos abertos a cada 2s; clean recarrega, dirty oferece reload com confirm (uma oferta por mudança; com overlay aberto, só status) |
| Clipboard | Host nativo OpenTUI quando disponível, OSC52 e fallback de sessão; paste bracketed funciona sobre SSH; roteamento para prompt/picker/busca/terminal testado |
| Splits | No máximo 2 panes, como o oráculo (redividir reorienta); divisores arrastáveis com geometria centralizada em `src/ui/dividers.ts` |
| Jumps | Pilha espelhando `jump_list.rs` (dedupe, trunca futuro, cap 64); `jump_back` registra antes de voltar; buffer-picker, SCM, busca e definição registram; status como o oráculo |
| Git Gutter | Indicadores de diff por linha (`+` verde, `~` amarelo/azul, `-` vermelho) calculados via `git.diffGutter` e `parseDiffHunks` no editor pane |
| Navegador de Paths | Modal interativo `openPathBrowser` para `save_as` e `open_folder` com drilldown de subdiretórios, `..` e confirmação segura |
| Comentários Web | Suporte a comentários em bloco (`<!-- ... -->` para HTML/XML e `/* ... */` para CSS) com toggle bidirecional |
| I18n Centralizada | Dicionário de diálogos, prompts, status e mensagens dinâmicas em `pt-BR` e `en-US` via `app.t(section, key, params)` |
| Inteligência Zero-Config | Buffer-word completion (< 1ms, multi-buffer), catálogo estático de 9 linguagens (Rust, TS, JS, Python, Go, C, C++, Bash, Ori) com docs e assinaturas, extrator regex de símbolos e docstrings, hover e completion fail-closed sem LSP |
| Configurações & Autopairs | Menu `Configurações` na menubar (`Alt+S`) e Command Palette com checkboxes `[✓]` / `[ ]`, autopairs inteligente (fechamento, auto-skip, surround e backspace inteligente), abertura de arquivos `config.toml` do usuário e workspace como abas e hot-reload imediato ao salvar |

**Registro completo não significa certificação completa:** alguns caminhos ainda
precisam de testes visuais/integração dedicados (task runner, plugins/hooks pela
UI e LSP em servidores reais). Certificados nesta rodada: imagens do preview
(recusas), links (extração + fluxo sem seleção), resize por arrasto (geometria +
fiação do painel), jump list, disk confirm, multi-picker, paste, inteligência zero-config e menu de configurações com autopairs.

## Validar

```bash
bun run typecheck
bun test
bun run build
bun dist/cli.js --version
cargo fmt --all -- --check
cargo clippy --all-targets -- -D warnings
cargo test --workspace
go test ./...
cargo build -p oride
bun run parity
```

A validação desta rodada: **76 testes Bun passam** (0 falhas; menu de configurações interativo com autopairs, inteligência zero-config com buffer words, catálogo e símbolos regex, live preview de
temas com rollback no Escape, suporte completo a paletas incluindo `zed-dark`,
suíte de ícones e interface refinada no padrão Zed, rolagem e navegação rica e interativa
no preview Markdown via teclado e mouse sem mutação do buffer), `bun run typecheck` limpo e
`bun run build` + `dist/cli.js --version` verdes. `bun run parity` mede 73/76 frames
estáveis contra os casos de conformidade.

O teste de CLI abre uma **PTY real**, inicia o entrypoint público, digita,
salva com Ctrl+S, sai com Ctrl+Q e verifica a restauração da tela principal.
Os testes de UI usam `testRender` com preload Solid também em `[test]` do
`bunfig.toml`; sem ele Bun usa JSX/reatividade errados.

Relatório: [`opentui-parity-report.json`](opentui-parity-report.json).
As fixtures de caso vivem em `conformance/cases/**/*.toml` (TOML, com `description`,
`[config]`, `[[files]]`, `[[steps]]` e `[divergence]` opcional); os dois lados
leem o mesmo arquivo — o oráculo via `toml::from_str`, o harness TS via
`smol-toml` — e o relatório de saída continua JSON por ser artefato de máquina.
A medição atual compara **73/76 frames iguais, 11 casos**, apenas para texto,
bytes, scalars, linhas e matches da busca. Nenhum campo divergente foi excluído.
**Isso não representa 96% das funcionalidades nem prova paridade total.**

Três frames divergem de bugs já conhecidos do Rust: `extended-selection`, passos
9/10 (seleção restaurada no undo, B2/B3), e `multi-cursor`, passo 6 (B2/B4).
As diferenças ficam no relatório; não foram apagadas para declarar 100%.

## O que impede declarar migração completa

1. **Expandir o comparador diferencial:** caret, seleção, histórico de undo, config,
   tabs, árvore de arquivos, Git, LSP, terminal, Vim, overlays e frames ainda não
   são comparados em matriz automatizada com o oráculo Rust.
2. **Restauração completa de sessão:** `session.toml` restaura arquivos abertos e aba
   ativa, mas não restaura razão de splits, orientação de divisão ou posições de scroll
   dos panes.
3. **Internacionalização e prompts:** catálogo de comandos está traduzido (pt-BR / en-US),
   mas mensagens de status dinâmicas e modais de confirmação ainda têm literais fixos;
   which-key exibe lista linear sem paginação/categorização.
4. **Navegador de filesystem nativo e polimento:** salvar como e abrir pasta usam prompt textual
   de caminho em vez de árvore interativa navegável no modal; custom languages sem gramática Shiki
   ficam sem highlight; HTML/CSS não têm comentário de bloco; plugins só executam hooks após autorização explícita na sessão.
5. **Motor de histórico avançado:** undo/redo utiliza snapshots lineares com limite
   de memória (~32 MiB), não árvore ramificável de undo (undo tree com branches).
6. **Empacotamento e distribuição:** `dist/cli.js` depende do binário do Bun instalado
   no host; não há binário standalone compilado (`bun build --compile`) nem pacote Nix/AUR;
   falta benchmark de boot/input, suporte a arquivos gigantes e matriz multiplataforma macOS/Windows.

## Limites e decisões

- Texto editável: **32 MiB** por arquivo; rejeita binários e UTF-8 inválido.
- Histórico: snapshots, orçamento aproximado **32 MiB**; não usa rope ainda.
- Finder: **100.000 arquivos**, ignora `.git`, `node_modules`, `target`, `dist`;
  ainda não interpreta todas as regras de `.gitignore`.
- Busca: **100.000 matches** por buffer e **10.000 hits** no projeto.
- Regex: semântica JavaScript; não promete aceitar todo padrão do Rust `regex`.
- Highlight assíncrono ainda executa tokenização no processo JS; mover para worker
  e medir arquivos grandes é necessário antes de afirmar latência de IDE.
- Nada faz pull/push, commit ou task automaticamente na validação contra o
  workspace real. Testes Git/PTY usam diretórios temporários.
- No-color conserva pistas textuais; mídia remota não é baixada implicitamente.

## Auditoria de Event Loop, Paralelismo e Avaliação de Persistência

### 1. Benchmark MessagePack vs JSON (Persistência em Disco)
Avaliação com script isolado simulando cache de 5.000 arquivos (`path`, `size`, `mtimeMs`, `isDirectory`) no Bun 1.4.2:
- **Encode:** JSON `1.23ms` vs MessagePack `5.50ms` (JSON 4.4x mais veloz).
- **Escrita:** JSON `0.34ms` vs MessagePack `0.26ms` (MessagePack 24.2% mais rápido na escrita bruta).
- **Leitura/Decode:** JSON `6.49ms` vs MessagePack `16.25ms` (JSON 2.5x mais veloz).
- **Tamanho final:** JSON `452.5 KB` vs MessagePack `334.3 KB` (MessagePack 26.1% menor).

**Decisão técnica fundamentada:** O gate de adoção estipulava >30% de melhoria em tempo ou tamanho. O MessagePack foi significativamente mais lento em encode (-347%) e decode (-150%), e a redução de tamanho (26.1%) não atingiu o limiar. Além disso, a arquitetura do Oride não mantém caches massivos em disco — a persistência real consiste em `session.toml` (~10 linhas), `config.toml` e buffers de texto cru. Portanto, **`@msgpack/msgpack` não foi adicionado como dependência**, mantendo a superfície de dependências limpa e o motor ágil.

### 2. Proteção do Event Loop (I/O Síncrono)
- **Diagnóstico:** Apenas dois pontos `*Sync` existiam no código: chamadas a `realpathSync` na resolução de imagens de markdown em `resolvePreviewImage` (`src/services/languages.ts`), executadas durante a pintura da TUI.
- **Implementação:** Caches em memória `workspaceRootCache` e `imagePathCache` foram introduzidos, garantindo que o caminho raiz do workspace e caminhos já resolvidos não invoquem chamadas de sistema síncronas repetidas no event loop durante renders do preview. Adicionada também a variante assíncrona `resolvePreviewImageAsync` usando `node:fs/promises`.

### 3. Paralelismo de CPU com Worker Threads (Fuzzy Finder)
- **Diagnóstico:** No finder/palette, com até 100.000 caminhos de arquivo, executar `fuzzyScore` + `sort` síncrono na thread principal a cada tecla digitada extrapolava a janela de frame de 16ms (60 FPS).
- **Implementação:** `src/workers/fuzzy.ts` operando via `Worker` nativo do Bun.
  - Entradas `<= 2000`: filtragem síncrona direta para evitar custo de IPC / `structuredClone`.
  - Entradas `> 2000`: offload automático para o worker em background com debounce de 50ms, rastreamento de geração (`generation`) para descarte de respostas desatualizadas e janelamento do top-100.
  - Fail closed: fallback transparente para cálculo síncrono caso o worker não consiga ser inicializado.
  - Suíte de testes com 20.000 entradas valida que o resultado do worker é rigorosamente idêntico ao cálculo síncrono.

### 4. Otimização de Renderização (Janelamento de Matches de Busca)
- **Diagnóstico:** `EditorPane.paint` iterava `app.matches` (que pode conter até 100.000 ocorrências) em cada célula individual da tela do terminal (ex: 80x25 células = centenas de milhões de verificações por frame).
- **Implementação:** Janelamento visual em `src/ui/editor.tsx` filtrando `app.matches` previamente para o intervalo visível da tela (`visibleStart` até `visibleEnd`). Apenas as ocorrências do viewport ativo são testadas durante o desenho das células.

### 5. Auditoria de Alternativas em Rust (Mapeamento de Débito Técnico)

| Componente JS atual | Alternativa Rust / NAPI | Custo de Integração & Decisão |
|---|---|---|
| `fuzzyScore` próprio | `nucleo` / `skim` via NAPI | **Alto** (toolchain NAPI, binários multiplataforma). O Worker nativo do Bun resolveu o orçamento de 60 FPS com zero dependências externas. |
| Shiki / oniguruma WASM | `tree-sitter` nativo (`-tags treesitter`) | **Alto** (CGO/bindings nativos). Shiki atende com cache de tokens por versão. Avaliar em benchmark dedicado de arquivos >10MB. |
| Digest FNV (`sessionName`) | `Bun.hash` | **Decisão: Mantido FNV-1a**. A troca quebraria o determinismo cross-runtime compartilhado com as implementações Rust e Go. |
| `Bun.Glob`, `node:fs` | Crates Rust equivalentes | **Nenhum ganho**. As APIs nativas do Bun já utilizam chamadas de sistema otimizadas em Zig/C. |

