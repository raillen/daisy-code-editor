# 🌼 Daisy Code Editor — Official Roadmap

Bem-vindo ao roadmap oficial do **Daisy Code Editor**, detalhando o estado atual do projeto, as funcionalidades consolidadas na **v0.2.0** e as metas estratégicas até a versão estável de longo prazo **v1.0 (LTS)**.

---

## 🏛️ Princípios Arquiteturais

1. **Velocidade & Leveza Absoluta**: Inicialização sub-50ms e taxa de quadros estável em 60 FPS, sem runtime Chromium ou Electron.
2. **Reatividade Fina**: Construído sobre `@opentui/core` e `@opentui/solid` em **Bun & TypeScript**, recalculando apenas as células alteradas do buffer.
3. **Desacoplamento e Fail-Closed**: LSP, Git e shell PTY executam como processos externos assíncronos via `$PATH`. Falhas externas nunca travam a interface de edição.
4. **Ergonomia Híbrida**: Navegação CUA amigável com menus, barra de comandos e atalhos tradicionais, combinada a um modo modal Vim opcional e atalhos mnemônicos via `Alt`.
5. **Estética Distinta**: Design brutalista e cyberpunk inspirado em *Serial Experiments Lain* e na cultura hacker dos anos 90, com temas neon de alto contraste e legibilidade terminal impecável.

---

## 📊 Status dos Marcos

| Versão | Foco Principal | Status |
|---|---|---|
| **`v0.2.0`** | **Foundation, Cyberpunk HUD & Packaging** | **Entregue (Atual)** ✅ |
| **`v0.3.0`** | **Tree-Sitter, Code Folding & AST Ergonomics** | 🔨 Próxima Milestone |
| **`v0.4.0`** | **Visual Diagnostics, Quick Fixes & Multi-LSP** | 📅 Planejado |
| **`v0.5.0`** | **Side-by-Side Diff, Hunk Staging & Git Tooling** | 📅 Planejado |
| **`v0.6.0`** | **Terminal Graphics (Kitty/Sixel) & Interactive Markdown** | 📅 Planejado |
| **`v0.7.0`** | **Daisy Daemon & Persistent Remote Sessions** | 📅 Planejado |
| **`v0.8.0`** | **Integrated Cyberpunk AI Pair Programmer** | 📅 Planejado |
| **`v1.0.0`** | **Production Stability & LTS Freeze** | 🎯 Meta de Longo Prazo |

---

## ✅ v0.2.0 — Milestone Atual (Entregue e Consolidado)

Todas as funcionalidades abaixo estão implementadas, testadas e disponíveis na versão estável `0.2.0`:

- [x] **Motor Reativo OpenTUI + SolidJS**: Renderização por células em buffer otimizado, suporte completo a mouse, scroll suave e splits verticais e horizontais redimensionáveis.
- [x] **Barra de Menus CUA com Mnemonics Interativos**: Tecla `Alt` ou `F10` ativa sublinhados mnemônicos (`<u><b>A</b></u>rquivo`, `<u><b>E</b></u>ditar`, etc.) com navegação rápida de dois toques e suporte bilíngue (`pt-BR` e `en-US`).
- [x] **Barra de Comandos Global (`Ctrl+Shift+P`)**: Indexação unificada de todos os menus, ações, seletores diretos de tema, seletor de idioma, comandos de plugins e tags fuzzy inteligentes.
- [x] **Família de Temas Daisy-Lain Cyberpunk HUD**:
  - `daisy-lain` (The Wired — Midnight Void, Acid Yellow, Deep Cobalt, Cyan).
  - `daisy-lain-cobalt` (Hero Blue — Dense Cobalt, Acid Yellow wireframes, Signal White).
  - `daisy-lain-glitch` (CRT Anomaly — Laser Glitch Crimson, Cyan, Midnight).
  - `daisy-lain-bone` (Brutalist Bone Paper — High contrast off-white & black ink).
  - 11 temas clássicos da comunidade (Catppuccin, Nord, Tokyo Night, Dracula, etc.) com live preview interativo.
- [x] **LSP Desacoplado via PATH**:
  - Autocompletação baseada em contexto e palavras do workspace.
  - Hover de documentação e tipo (`K` / `lsp_hover`).
  - Ir para Definição (`F12` / `lsp_goto_definition`).
  - Renomeação de símbolos no arquivo e projeto (`F2` / `lsp_rename`).
  - Formatação de código ao salvar (`format_on_save`).
  - Agregação de diagnósticos de compiladores ativos.
- [x] **Git / SCM Integrado**:
  - Marcadores de Git Gutter em tempo real no editor (`+` adicionado, `~` modificado, `-` removido).
  - Painel lateral de SCM (`Ctrl+Shift+G`) com status de branch, commits à frente/atrás e staged files.
  - Git Blame da linha sob o cursor em overlay dedicado (`Alt+B`).
  - Visualizador de Diff rápido do arquivo atual.
  - Operações de `git pull` e `git push` com diálogo de confirmação.
- [x] **Markdown Live Preview**: Split integrado com renderização estilizada de títulos, listas e blocos de código com destaque de sintaxe (`Alt+P`).
- [x] **Terminal PTY Embutido**: Emulador de terminal completo no rodapé (`Ctrl+\``) com suporte a shells do sistema (bash, zsh, fish).
- [x] **Modo Modal Vim**: Suporte aos modos Normal, Insert, Visual e Visual-Line (`h, j, k, l, w, b, e, 0, $, gg, G, dd, yy, p, u, Ctrl+R, :w, :q, :wq, /`).
- [x] **Edição Multicursor & Ferramentas Avançadas**: Adicionar cursores acima/abaixo, selecionar todas as ocorrências de um identificador, envolver com delimitadores (`surround`) e árvore linear de undo (`undo_tree`).
- [x] **Busca e Substituição no Projeto**: Localizar e substituir em múltiplos arquivos com expressões regulares e filtros de glob (`Ctrl+Shift+F`).
- [x] **Sistema de Plugins e Tarefas**: Descoberta de plugins TOML com hooks de ciclo de vida (`on_open`, `on_save`) e executor de scripts (`tasks.toml`).
- [x] **Distribuição e Empacotamento Universal**:
  - Instalador interativo em uma linha (`curl ... | bash`).
  - Pacotes binários gerados para **Arch Linux** (`PKGBUILD`), **Fedora** (`.spec`), **Debian/Ubuntu** (`.deb`), **AppImage**, **Flatpak** e binários standalone compilados para `linux-x64` e `linux-arm64`.

---

## 🚀 Roadmap Detalhado (Próximas Versões)

### 🌟 v0.3.0 — Inteligência Sintática & AST (Tree-Sitter)
*Foco: Elevar o parsing de código de expressões regulares para árvores sintáticas completas.*

- [ ] **Tree-sitter WASM Engine**:
  - Incorporar gramáticas Tree-sitter em WASM para TypeScript, JavaScript, Rust, Python, Go, C/C++, JSON, HTML e Markdown.
  - Eliminar limitações de realce em código multilinha, template literals e expressões regulares complexas.
- [ ] **Code Folding Semântico (Dobra de Código)**:
  - Dobrar e expandir blocos de classes, métodos, funções e declarações de tipos.
  - Indicadores interativos na margem lateral (`▾` aberto / `▸` fechado), com atalhos de teclado (`zc` fold, `zo` unfold, `zM` fold all, `zR` unfold all).
- [ ] **Indent Rainbow & Bracket Pair Colorizer**:
  - Guias verticais sutis indicando os níveis de indentação do código.
  - Realce em cores neon do par de delimitadores correspondente (`()`, `[]`, `{}`) onde o cursor estiver posicionado.
- [ ] **Vim AST Text Objects**:
  - Suporte a operadores estruturais no modo modal: `daf` (delete around function), `cib` (change inside block), `yac` (yank around class).

---

### ⚡ v0.4.0 — Diagnósticos Visuais & Ergonomia de Refatoração
*Foco: Trazer o feedback do compilador diretamente para o campo de visão do desenvolvedor.*

- [ ] **Inline Ghost Diagnostics (Virtual Text)**:
  - Renderização de erros e advertências do compilador/LSP diretamente no final da linha em texto fantasma esmaecido (estilo Neovim / Helix / Zed), com glifos estilizados (`󰅚` erro, `󰀪` aviso).
- [ ] **LSP Code Actions & Quick Fixes (`Ctrl+.` / `Alt+Enter`)**:
  - Menu flutuante contextual para aplicar resoluções automáticas sugeridas pelo servidor de linguagem (adicionar import ausente, corrigir tipagem, implementar membros abstratos).
- [ ] **Inlay Hints**:
  - Exibição de tipos inferidos e nomes de parâmetros de funções diretamente entre os caracteres do código como texto inline não-editável.
- [ ] **Suporte a Múltiplos Servidores LSP por Arquivo**:
  - Capacidade de acoplar múltiplos servidores em um único buffer (exemplo: `typescript-language-server` + `tailwindcss-language-server` + `biome`).

---

### 🌿 v0.5.0 — Side-by-Side Diff, Hunk Staging & Git Tooling
*Foco: Experiência completa de revisão e controle de versão sem sair do editor.*

- [ ] **Side-by-Side Split Diff Viewer**:
  - Modo de visualização de diff em duas colunas sincronizadas para comparar a versão de trabalho com o `HEAD` do Git ou commits anteriores.
- [ ] **Staging Interativo de Hunks**:
  - Capacidade de navegar entre blocos modificados (`hunks`) e aplicar stage/unstage parcial linha a linha ou bloco a bloco diretamente pelo editor.
- [ ] **Interactive Merge Conflict 3-Way Resolver**:
  - Detecção automática de marcadores de conflito (`<<<<<<<`, `=======`, `>>>>>>>`) e botões/atalhos de ação imediata: *Aceitar Atual*, *Aceitar Entrada*, *Aceitar Ambos*.
- [ ] **Inline Git Blame Ghost Text**:
  - Exibição sob demanda de autor, data relativa e mensagem do commit no final da linha ativa sem necessidade de abrir modal.

---

### 🎨 v0.6.0 — Gráficos no Terminal & Markdown Rico
*Foco: Romper os limites do texto simples com suporte aos protocolos gráficos modernos.*

- [ ] **Suporte aos Protocolos Kitty Graphics & Sixel**:
  - Renderização de imagens de alta resolução dentro do painel de visualização de Markdown e no explorador de arquivos em terminais compatíveis (Kitty, WezTerm, Ghostty, Alacritty c/ Sixel).
- [ ] **Markdown Preview Interativo**:
  - Checkboxes de tarefas clicáveis (`[ ]` / `[x]`) que atualizam o arquivo original instantaneamente.
  - Tabelas Markdown com alinhamento visual automático e bordas desenhadas.
- [ ] **Diagramas Mermaid & Fórmulas Matemáticas**:
  - Renderização de diagramas Mermaid em ASCII/SVG e expressões KaTeX simplificadas.

---

### 🔌 v0.7.0 — Daisy Daemon & Sessões Remotas
*Foco: Produtividade distribuída, persistência e desenvolvimento remoto ininterrupto.*

- [ ] **Modo Headless Daemon (`daisy daemon`)**:
  - Capacidade de executar o núcleo do editor como um serviço desacoplado em segundo plano em servidores locais ou remotos via socket Unix / TCP criptografado.
- [ ] **Reconexão Instantânea de Sessões (`daisy attach`)**:
  - Persistência total de buffers abertos, cursores, histórico de desfazer e terminais PTY ao desconectar uma sessão SSH ou fechar o terminal.
- [ ] **Pair Programming em Terminal**:
  - Dois ou mais usuários conectando-se à mesma sessão remota do Daisy com cursores sincronizados e cores distintas.

---

### 🤖 v0.8.0 — Assistente Cyberpunk IA (Daisy Copilot)
*Foco: Integração nativa e contextual com inteligência artificial generativa.*

- [ ] **Painel Lateral de Assistente Cyberpunk**:
  - Interface conversacional TUI com suporte a modelos locais via Ollama e provedores de nuvem (Anthropic Claude, OpenAI, Google Gemini).
- [ ] **Diffs de Código Gerados por IA**:
  - Sugestões de refatoração e geração de código apresentadas como diffs no próprio buffer, aceitas com um único toque (`Ctrl+Y`) ou descartadas (`Ctrl+N`).
- [ ] **Context Injection Inteligente**:
  - Injeção automática da assinatura de tipos, erros do compilador sob o cursor e trechos de arquivos selecionados no prompt.

---

### 🎯 v1.0.0 — Estabilidade de Produção & LTS Freeze
*Foco: Garantia de qualidade rigorosa, estabilidade de longo prazo e maturidade de produto.*

- [ ] **Congelamento da API de Extensões & Configuração**:
  - Estabilidade garantida sem breaking changes no formato de plugins e opções de configuração.
- [ ] **Hardening de Acessibilidade e Unicode**:
  - Suporte perfeito a larguras de caracteres CJK, emojis compostos e modos de alto contraste para leitores de tela de terminal.
- [ ] **Benchmarks Automatizados no CI**:
  - Verificação de orçamento de inicialização (< 30ms em máquina padrão) e uso de memória (< 45MB em repouso).
- [ ] **Documentação Completa de 1º Nível**:
  - Manuais técnicos e guias de usuário completos em Português e Inglês com tutoriais interativos.

---

## 🤝 Como Contribuir

Interessado em ajudar a construir alguma das funcionalidades do roadmap?
1. Veja as tarefas abertas em nossas [Issues no GitHub](https://github.com/raillen/daisy-code-editor/issues).
2. Consulte o guia de desenvolvimento no [README.md](README.md) e [AGENTS.md](AGENTS.md).
3. Abra um Pull Request focado em uma única funcionalidade por vez com cobertura completa de testes automatizados.
