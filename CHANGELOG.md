# Changelog

## Unreleased

### Changed
- **Barra de busca agora usa o componente `textinput` (S3):** inserção, apagar por
  rune, placeholder e a cauda da consulta longa saíram do código próprio e
  passaram para a biblioteca. O quadro ficou byte a byte igual — os quatro goldens
  da barra passaram sem regerar. A barra de busca segue **sem cursor visível**:
  o cursor do componente pisca, e um golden não pode depender do relógio.

### Added
- **A árvore mostra o estado do git de cada arquivo**, na coluna da direita: `M`
  modificado, `A` novo, `D` apagado, `?` não rastreado. A letra é a pista principal
  e a cor a segunda, para que um terminal sem cor continue mostrando o estado. O
  dado já era lido do disco e não aparecia em lugar nenhum.

### Added
- **Painel inferior com abas — Terminal │ Problemas │ Busca:** um lugar só, no
  modelo do Lapce e do Zed. Antes o terminal ficava embaixo e a barra de busca
  acima, em regiões separadas: abrir a busca fazia o terminal sumir. A faixa come
  uma linha em todas as abas, a aba ativa é marcada em texto, e a aba de
  problemas **diz** que o LSP está desligado em vez de oferecer um "0 problemas"
  que ninguém mediu.

### Fixed
- **O terminal não aceita mais teclado quando outra aba está à frente:** o foco
  podia descansar num painel invisível, o que a regra do projeto proíbe.

### Added
- **A barra de status virou superfície de informação:** modo da linguagem, branch
  do git com `↑ahead ↓behind`, e o estado do LSP. O número de erros **só aparece
  quando há cliente LSP** — sem servidor, "0 erros" é ausência de medição, e a
  barra se recusa a dizer que está tudo bem. A branch é lida fora do caminho do
  frame, num tique de dois segundos.

### Changed
- **`[lsp].enabled` tem efeito:** a barra escreve `LSP off` quando o usuário
  desliga. Ele era lido por ninguém desde o port.

### Added
- **Soft wrap ligado por padrão, com a linha atual marcada:** uma linha de 121
  caracteres num terminal de 60 aparece **inteira**, partida em linhas de tela, em
  vez de mostrar 59 caracteres e um `…` sem caminho para o resto. A quebra é por
  palavra quando dá e por cluster quando a palavra não cabe (URL, caminho,
  identificador); o número da linha aparece uma vez; a seleção e o realce
  atravessam a quebra; e a linha atual tem duas pistas — o marcador de texto na
  gutter e, para quem quiser, o fundo do token `line_highlight`.

### Changed
- **`[editor].soft_wrap` passa a valer alguma coisa** (vem ligado, como no Zed).
  É uma decisão de produto, não uma paridade: o oráculo entrega desligado, e a
  diferença está registrada como `B19` no ledger de paridade.

### Added
- **Camada de tokens semânticos no tema** (`[tokens]`): um tema passa a nomear
  **intenções** — `surface`, `panel`, `text`, `text_muted`, `accent`, `success`,
  `warning`, `error`, `focus` — em vez de repetir cores. É o modelo do Zed, e é o
  que torna possível um tema **`no-color`**: uma paleta que afirma não ter cor
  nenhuma e é verificada como tal, em vez de um perfil que desliga a cor porque o
  terminal não sabe fazer.
- **A paleta de comandos executa, e mostra nomes em vez de ids:** `Enter` na linha
  destacada roda o comando (antes a paleta listava e não executava nada), os 26
  comandos que os menus nomeiam aparecem traduzidos ("Salvar", "Alternar árvore"), e
  digitar já abre o filtro sem precisar de `/`.
- **Seletor de tema e de idioma:** `Ctrl+Shift+S`-style pickers listam os 10 temas
  e os locales disponíveis, marcam o ativo em texto, e `Enter` aplica — trocando os
  estilos e recarregando o catálogo. Fechar com `Esc` cancela sem aplicar.
- **Painéis, foco e recarga funcionam:** `Ctrl+E` alterna o foco entre árvore e
  editor (e abre a árvore se ela estiver escondida), `F5` relê a árvore do disco,
  `Ctrl+R` recarrega o arquivo do disco **mantendo o caret** — e diz o motivo
  quando não consegue, em vez de devolver um erro que engole a tecla. `Ctrl+M`
  liga e desliga o mouse, e a mensagem explica por que o padrão é desligado.
- **Abas, paginação e área de transferência funcionam:** `Ctrl+N` (nova aba),
  `Ctrl+PgUp`/`Ctrl+PgDn` (trocar de aba), `Ctrl+W` (fechar aba), `PageUp`/
  `PageDown` (paginação pela altura da janela) e `Ctrl+C`/`Ctrl+V`/`Ctrl+X`
  (copiar, colar, cortar) eram anunciados no keymap e não faziam nada. A cópia
  segue a regra do editor de linha: a seleção, ou a linha inteira quando não há
  seleção; e usa OSC 52, então também vai para o clipboard do sistema e funciona
  sobre SSH.

### Fixed
- **Painel de terminal embutido:** `Ctrl+`` abre o painel no rodapé, o foco vai
  para ele, e as teclas vão para o shell (Enter, backspace, setas e demais arrive
  como o shell espera). `Esc` volta para o editor, `Ctrl+C` não é roubada do shell,
  e o painel cresce e encolhe entre 3 e 40 linhas. A PTY nasce fora do caminho do
  `Update`, então o modelo continua testável sem terminal — e uma falha ao subir o
  shell deixa o editor utilizável em vez de trancá-lo.
- **Regra de aptidão da configuração (S4):** uma regra de arquitetura nova
  (`R8`) falha o build quando um campo de configuração não tem leitor fora do
  pacote `config` e não está na dívida nomeada. Ela encontrou sozinha uma
  dezesseisésima chave morta — `[ui].gutter_width`.
- **13 teclas que não faziam nada agora funcionam** (abas, paginação, cópia,
  painéis, foco e recarga): 81 ações têm tecla no keymap, 49 são tratadas e **28
  continuam mortas**, todas nomeadas com o motivo e o que falta para cada uma.
- **40 teclas mortas agora são dívida nomeada (S4):** 81 ações têm tecla no
  keymap, 37 eram tratadas e **40 não faziam nada** — e a tabela de
  `docs/ui-ux/keymap.md` é gerada dessas mesmas bindings, ou seja, a documentação
  anunciava atalhos que o programa ignorava. A nova regra falha o build quando uma
  ação anunciada perde o tratamento, e a lista de dívida é pública.
- **Toda binding é agora provada alcançável pelo terminal (S4):** o teste sintetiza
  a tecla, tira a grafia que o runtime entregaria e pergunta ao keymap a que ação
  resolve. Uma binding com um nome que o terminal não produz falha no momento em
  que é escrita.
- **Teste de fumaça da TUI (S5):** a suíte agora dirige uma sessão inteira como
  um usuário — abrir, navegar a árvore, abrir arquivo, digitar, buscar, fechar —
  e afirma estado, não frame. É o item que faltava no gate: nove checks mediam
  frames, testes e paridade sem perguntar se uma tecla faz alguma coisa.
- **Medida de largura unificada (S2):** `layout` passou a medir com a mesma engine
  dos componentes `bubbles`, e caractere ambíguo passou a contar como estreito. A
  tabela ambígua do `go-runewidth` inclui "é" e não inclui "ã", então toda linha
  acentuada saía **uma coluna mais larga** que a linha vizinha — visível em editor,
  árvore, barra de status e em qualquer campo truncado. `Slice`, `Truncate` e
  `Cells` passaram a medir por cluster de grafema em vez de por rune, o que
  impede cortar acento combinado, emoji com ZWJ e bandeira ao meio.

### Added
- **Realce de sintaxe (S1):** Motor `chroma/v2` sem CGO, com os vinte papéis da
  tabela `[syntax]` ligados de fato — dezenove deles eram parseados e lidos por
  ninguém desde o primeiro marco da migração. O mapeamento é por família de
  linguagem, `[[languages]]` ganhou leitor, e o realce é assíncrono e cacheado por
  documento, versão e caminho, para que uma tecla não pague por lexar o arquivo
  inteiro.
- **Project Atlas Framework v0.4.2:** Governança canônica do projeto via `atlas.json`, manifesto de orquestração e agentes em `.ai/`, rastreabilidade de estado em `PROJECT_STATE.md` e roteador em `ENTRYPOINT.md` e `docs/ATLAS.md`.
- **Harness Google Antigravity:** Compilação nativa de regras e habilidades de workforce em `.agents/` via `atlas compile --target antigravity`.
- **Diretrizes de Contribuição Bilíngues:** Adicionados `CONTRIBUTING.md` (Inglês) e `CONTRIBUTING.pt-BR.md` (Português) detalhando invariantes arquiteturais, TDD, quality gates e fluxo de Pull Requests.
- **Estruturação de Guias de Usuário:** Centralização de todos os manuais sob `docs/guides/pt/` e `docs/guides/en/`, acompanhados de portais de índice.

## 0.2.0

Release focada em linguagens first-class, visualização rica e mídia no Markdown, Git/SCM interativo com sincronização, busca com globs, persistência completa de sessão e consolidação multi-LSP.

### Added

- **Multi-LSP sob demanda (`L2`):** Suporte nativo para servidores externos configurados em `[lsp.servers]` e nos providers embutidos (`rust-analyzer`, `pylsp`, `serve-d`, `lua-language-server`, `typescript-language-server`, `ori-lsp`, `oriscript lsp`), com ciclo de vida preguiçoso e tratamento fail-closed amigável caso o binário não esteja no `$PATH`
- **Git status bar & sync (`G1.3`):** Contador de commits à frente/atrás na barra de status em relação ao upstream (`↑ahead ↓behind`, ex.: `git:main ↑1 ↓2`), atalhos no SCM para `git pull` (`p`) e `git push` (`P` / `Shift+P`), além de ações dedicadas na Command Palette e no menu Git
- **Mídia no Markdown (`M2`):** Inspeção pura em Rust de dimensões e tipos de imagens locais (`PNG`, `JPEG`, `GIF`, `WebP`, `SVG`), detecção de protocolos de terminal gráfico (Kitty Graphics, Sixel, iTerm2), visualização enriquecida no preview e configuração `[markdown].terminal_images`
- **Persistência de Sessão (`E1.1`):** Restauração completa de layout em `.oride/session.toml` preservando posição exata de scroll (`scroll_y`), proporções de split e documento secundário, largura da árvore de arquivos e visibilidade dos painéis
- **Filtro por Glob na Busca de Projeto (`E1.2`):** Suporte a padrões glob (`*.rs`, `!target/**`) no `oride-search` (ripgrep e crawler interno), atalho `Alt+G` para alternar foco no campo de glob e navegação circular `Tab` no modal de busca e substituição
- **Linguagens First-Class (`L1`):** Suporte a Dlang (`.d`, `.di`) e Lua (`.lua`), além de Rust, Python, TypeScript/TSX, Ruby, Nim, Ori (`.orl`) e OriScript (`.oris`), com highlight, comentários, palavras para autocompletar e injeção de code fences em Markdown
- **Markdown Rico (`M1`):** Tabelas formatadas com desenho Unicode de caixas e alinhamento de colunas, destaque sintático dentro de blocos de código e abertura de links no navegador do sistema por clique do mouse ou `Alt+Enter`
- **Operações de Arquivos e Git SCM (`G1`):** Stage (`s`), unstage (`u`) e commit interativo (`c`) no painel SCM; renomear (`r`), deletar (`d`) e copiar caminho (`y`/`c`) na árvore de arquivos
- **Busca e Substituição no Projeto:** `replace_in_project` integrado com recarregamento em tempo real dos buffers abertos
- **Redimensionamento Dinâmico:** Arrastar divisores de árvore e divisões do editor via mouse e atalhos na palette
- Grammars tree-sitter para Rust, C, Bash, Markdown e OriScript; as demais linguagens (Python, TypeScript/TSX, Ruby, Nim, Ori, D, Lua, HTML, CSS, JavaScript) usam o realce lexical embutido — ou uma grammar dinâmica carregada em runtime, quando presente

### Removed

- **Remoção de Macros (`R1`):** Ações e menções a macros de teclado (`F9`/`F10`) removidas do core e da interface, consolidando a postura anti-bloat e mantendo o editor leve e rápido

### Fixed

- Autocomplete substitui o prefixo digitado em vez de anexar o texto completo, sincroniza cada edição com o LSP e converte snippets/text edits para inserção compatível com o editor
- Abrir arquivo com Enter na árvore sincroniza imediatamente o documento do painel focado, sem exigir troca de aba para renderizar o conteúdo
- Proteção de saída agora considera mudanças não salvas em **todas** as abas, não apenas na ativa
- `Save all` reporta falhas parciais em vez de anunciá-las silenciosamente como sucesso
- Reload externo atualiza a aba correta mesmo quando ela está em background; troca de workspace reinicia watcher, LSP e PTY
- Terminal embutido honra `[terminal].shell`, encerra a thread leitora com o PTY e limita scrollback Unicode sem `panic`
- LSP usa colunas UTF-16, aplica múltiplos `TextEdit` por range, preserva `insertText` de completion e mantém `didOpen`/`didClose` consistentes
- Navegação de project search converte colunas byte do ripgrep para caracteres; cut line remove finais CRLF completos
- Status bar mantém `git blame` em cache por arquivo/linha, evitando spawn de subprocesso a cada frame

## 0.1.0-alpha.6

Baseline mini-IDE TUI contida. Plano normativo: [`docs/planning/alpha6-roadmap.md`](docs/planning/alpha6-roadmap.md).

### Added (mouse)

- **Mouse completo** (**default off** — `mouse = true` no TOML ou **View → Enable mouse** / palette)
- Clique → caret · drag → seleção · duplo → palavra · triplo → linha
- Clique árvore/abas/SCM/terminal/menu · scroll por painel · botão direito → which-key
- Capture só quando ligado (não rouba scroll do emulador no default)

### Added (navigation / git UX)

- **Surround** (`F8`) · **Multi-picker** (`Ctrl+Shift+T`) · **Undo history** (`Ctrl+Shift+U`)
- **SCM panel** (`Ctrl+Shift+G`) · buffer picker · jump list · blame · diff read-only (`F2`)
- Menu bar, context banner, which-key, welcome, find mini-modal

### Added (find)

- Palavra completa (`Alt+W`) · modal Find/Replace legível · project find (`Ctrl+Shift+F`, rg+fallback)

### Added (markdown)

- Preview TUI + placeholders de imagem · task lists/tabelas/frontmatter/strike · fence inject oris/js/html/css
- Preview segue scroll do editor

### Added (P5–P9 stack)

- Plugins built-in · splits · multi-cursor · terminal PTY interativo · LSP OriScript (alpha.5+)

### Fixed

- Mouse drag lag (drain/coalesce events) · double-click word · editor scroll follows caret

### Docs

- Plano **alpha.6+** contido (sem macros, sem preview HTML/browser)

### Note

- Macros (`F9`/`F10`) presentes nesta tag mas **marcadas para remoção** (R1 no roadmap); não expandir

## 0.1.0-alpha.5

### Added (P4 final)

- `.editorconfig` (indent_style / indent_size)
- Reload on disk change (`notify`) + `Ctrl+R` + prompt se dirty
- Terminal resize (`Alt+=` / `Alt+-`)
- Find **regex** (`Alt+R`)
- Config sections `[tree]` `[terminal]` `[lsp]` `[syntax]` + `format_on_save`
- Clipboard **OSC52** (SSH)
- CI workflow + `scripts/install.sh`
- `docs/plugin-api.md`

### Added (P3 LSP)

- Crate `oride-lsp` — cliente stdio JSON-RPC
- Diagnostics panel (`Ctrl+Shift+M`)
- Completion (`Ctrl+Space`) · Hover (`Ctrl+K`) · Goto (`F4`) · Format (`Ctrl+Shift+I`)
- Sync didOpen/didChange/didSave para buffers `.oris`

## 0.1.0-alpha.4

### Added (P4 polish)

- Lista completa de keybinds (`F1` / `Ctrl+G` / `Ctrl+Shift+/`) — filtro + scroll
- Find compacto no rodapé + replace/replace-all, case e acentos (`Ctrl+F`, `Ctrl+H`, `Alt+C`/`Alt+A`)
- Clipboard copy/paste/cut (`Ctrl+C/V/X`) via arboard + fallback interno
- Seleção multi-linha (Shift+setas/Home/End, Ctrl+Shift+Home/End, Ctrl+A) com highlight azul
- Save as (`Ctrl+Shift+S`) · Save all (`Ctrl+Alt+S`)
- Terminal toggle (`Ctrl+"` / `Ctrl+'`)
- **Browser de paths** para abrir pasta/arquivo (F2 / Ctrl+Enter / Ctrl+O confirma pasta)
- **Save as** via browser: digite o nome · **Enter** ou Ctrl+S salva · → entra pasta
- Highlight de linha selecionada nos modais (fundo ciano full-width)
- Aba ativa com fundo **branco** (chip no buffer); atalhos `Ctrl+PgUp/PgDn`, `Alt+←/→`
- Session leve: restaura workspace e abas; salva ao sair
- Docs: `docs/polish.md`; Markdown **futuro** em `docs/markdown.md`

### Fixed

- Seleção / copy-paste / select-all / save-as / confirmar pasta em terminais sem Ctrl+Enter
- Highlight visual da aba ativa e da seleção no editor

## 0.1.0-alpha.3

### Added

- **P2** — tree-sitter syntax highlight for `.oris`, Markdown, HTML, CSS, JavaScript
- Crate `oride-syntax` + vendored `tree-sitter-oriscript`
- Language id in status line; syntax colors on editor viewport
- **Markdown rico** — block+inline queries, headings/links/code/lists/quotes
- Derivados: `.mdx`, `.qmd`, `.rmd`, `.markdown`, README bare, etc.
- Soft wrap (`Alt+Z`, default on em MD)
- Toggle comment (`Ctrl+/`, HTML comments em MD)
- Continuação de listas Markdown no Enter
- Docs: `docs/markdown.md`

### Fixed (P2 polish)

- Cursor visível (cell invertida + `set_cursor_position` no terminal)
- Atalhos explícitos: `Ctrl+B` foco árvore, `Ctrl+E` foco editor (`Ctrl+Shift+B` oculta painel)
- Navegação da árvore: ↑↓/jk, ←→/hl, Enter, Space, Home/End
- `Ctrl+O` / palette “Open folder…” — abrir pasta de projeto no sistema
- Highlight de seleção da árvore (linha inteira ciano)

## 0.1.0-alpha.2

### Added

- **P1.1** — multi-tab bar, next/prev/close/new, dirty close confirm
- **P1.2** — project tree (`oride-fs`): expand, open file, create file/folder
- **P1.3** — Nerd Font icons + git status badges (`oride-git`)
- **P1.4** — embedded terminal panel (`oride-terminal`, PTY), toggle/focus
- **P1.5** — command palette + fuzzy open file
- Open directory as workspace (`oride .`)

## 0.1.0-alpha.1

### Added

- **P0.1** — workspace, `oride-core` (rope, undo, documents), headless CLI
- **P0.2** — minimal TUI (`oride-ui`, `oride-app`): open/edit/save/quit, status line
- **P0.3** — TOML config layers (defaults → `~/.config/oride` → `.oride/`), keymaps, UI theme colors
- Example config: `assets/config.example.toml`
- Docs: `docs/design.md`, `docs/config.md`
