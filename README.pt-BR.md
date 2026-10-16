<p align="center">
  <a href="README.md"><b>[ English ]</b></a> · <b>[ 🇧🇷 Português ]</b> · <a href="ROADMAP.md"><b>[ Roadmap ]</b></a>
</p>

<p align="center">
  <img src="assets/daisy-header.svg" alt="Daisy Code Editor // The Wired Terminal IDE" width="100%" />
</p>

<p align="center">
  <img src="assets/badges/made-with-bun.svg" alt="Bun 1.2+" />
  <img src="assets/badges/opentui.svg" alt="OpenTUI Engine" />
  <img src="assets/badges/solidjs.svg" alt="SolidJS Reactive" />
  <img src="assets/badges/no-electron.svg" alt="No Electron" />
  <img src="assets/badges/win98-aesthetic.svg" alt="Win98 Aesthetic" />
  <img src="assets/badges/the-wired.svg" alt="The Wired" />
</p>

<p align="center">
  <a href="https://bun.sh"><img src="https://img.shields.io/badge/Runtime-Bun%201.2+-070913?style=flat-square&logo=bun&logoColor=ffe500" alt="Runtime" /></a>
  <a href="https://www.typescriptlang.org"><img src="https://img.shields.io/badge/Linguagem-TypeScript%205.9+-070913?style=flat-square&logo=typescript&logoColor=67f4ed" alt="TypeScript" /></a>
  <a href="https://www.solidjs.com"><img src="https://img.shields.io/badge/UI-SolidJS%201.9+-070913?style=flat-square&logo=solid&logoColor=67f4ed" alt="SolidJS" /></a>
  <a href="https://github.com/opentui"><img src="https://img.shields.io/badge/Engine-OpenTUI-070913?style=flat-square&logoColor=ffe500" alt="OpenTUI" /></a>
  <a href="https://github.com/raillen/daisy-code-editor/releases"><img src="https://img.shields.io/badge/Versão-v0.2.0-0025c5?style=flat-square&color=0025c5" alt="Versão" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/Licença-MIT-070913?style=flat-square&color=3fb950" alt="Licença" /></a>
</p>

```text
╔══════════════════════════════════════════════════════════════════════════════╗
║  DAISY CODE EDITOR // BUILD DO SISTEMA 0.2.0 // PROTOCOLO: THE WIRED         ║
║  "Não importa para onde você vá, todos estão sempre conectados."             ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

> **Daisy Code Editor** é um editor de código de terminal e mini-IDE ultrarrápido, modular e brutalista construído com **OpenTUI + SolidJS** em cima de **TypeScript & Bun**.
>
> Projetado para inicialização a frio em menos de 40ms, zero sobrecarga de navegadores, edição híbrida CUA/Vim fluida e um HUD cyberpunk brutalista dos anos 90 inspirado em *Serial Experiments Lain* e sistemas clássicos de computação.

---

## [ 🖵 ] Recursos do Sistema

### [ 🗲 ] [SYS.ENGINE] Motor de Terminal ASCII Reativo
- Reatividade fina potencializada por `@opentui/core` e `@opentui/solid`.
- Recalcula exclusivamente as células alteradas em um buffer de memória otimizado; zero repinturas de tela inteira.
- Performance constante a 60 FPS com arrasto de mouse, scroll de roda e redimensionamento dinâmico de janela.

### [ 🗀 ] [DRIVE.TREE] Explorador de Arquivos & Abas
- Árvore de diretórios hierárquica navegável com suporte a glifos Nerd Font (`icons = true`).
- Gerenciamento de múltiplos buffers em abas, com fechamento via botão do meio do mouse e indicador de alterações pendentes (`●`).
- Divisão de visualização (splits verticais e horizontais) com divisores arrastáveis interativos.

### [ 🖳 ] [PTY.SHELL] Emulador de Terminal POSIX Embutido
- Terminal multiplexador interativo completo (`Ctrl+\``) executando o shell do usuário (`bash`, `zsh`, `fish`).
- Processamento nativo de sequências de escape VT100/ANSI com repasse direto de teclas.
- Divisor arrastável para redimensionar a altura do terminal ou o editor a qualquer momento.

### [ ⚙ ] [LSP.INTEL] Language Server Protocol (Cliente via $PATH)
- Arquitetura LSP desacoplada executando servidores externos pelo `$PATH` (`rust-analyzer`, `typescript-language-server`, `gopls`, `pyright`, `clangd`).
- **Autocompletação**: Sugestões contextuais de LSP combinadas com indexação local de palavras do workspace e ranking difuso.
- **Documentação em Hover (`K` / `Ctrl+K`)**: Assinaturas, tipos e docstrings Markdown em popup leitor flutuante.
- **Ir para Definição (`F12` / `gd`)**: Salto instantâneo para a definição do símbolo entre arquivos.
- **Renomeação de Símbolos (`F2`)**: Refatoração de AST em todo o projeto através de múltiplos buffers com edições atômicas.
- **Formatação ao Salvar**: Formatação automática via servidor de linguagem ou utilitários externos CLI.
- **Resiliência Fail-Closed**: Se um processo LSP falhar, o editor continua 100% estável e editável.

### [ ░▒▓ ] [HUD.THEMES] Estética Cyberpunk Daisy-Lain & 15 Temas
- Família visual brutalista cyberpunk inspirada no site [daisyweblain](https://daisyweblain.vercel.app/):
  - **`daisy-lain`** [The Wired]: Midnight Void `#070913`, Amarelo Ácido `#ffe500`, Cobalto `#0025c5`, Ciano `#67f4ed`.
  - **`daisy-lain-cobalt`** [Hero Blue]: Cobalto Denso `#020d58`, Wireframes em Amarelo Ácido, Branco Sinal `#f8f7eb`.
  - **`daisy-lain-glitch`** [CRT Anomaly]: Vermelho Glitch Laser `#ff4068`, Ciano Sinal, Midnight Void.
  - **`daisy-lain-bone`** [Brutalist Bone Paper]: Alto contraste osso cru `#f2efe6`, tinta preta profunda e destaques amarelos.
- 11 temas clássicos da comunidade inclusos: Catppuccin Mocha, Nord, Tokyo Night, Dracula, Gruvbox, One Dark, etc.
- Seletor interativo na Barra de Comandos com live preview instantâneo ao navegar com as setas.

### [ ⌕ ] [FAST.FIND] Busca e Substituição de Alta Velocidade
- Localização e substituição no buffer com expressões regulares, diferenciação de maiúsculas, palavras inteiras e remoção de acentos.
- Busca recursiva no projeto com padrões glob (`Ctrl+Shift+F`), listando ocorrências em múltiplos arquivos.
- Web Worker dedicado em segundo plano para busca fuzzy em coleções com mais de 20.000 arquivos sem travar a interface.

### [ 🖧 ] [GIT.SCM] Controle de Versão & Git Gutter em Tempo Real
- Marcadores de Git Gutter em tempo real na margem de numeração (`+` adicionado, `~` modificado, `-` removido).
- Modal SCM dedicado (`Ctrl+Shift+G`) exibindo nome da branch, commits à frente/atrás e arquivos em stage (`s` adiciona, `u` tira, `d` diff, `c` commit).
- **Git Blame na Linha (`Alt+B`)**: Exibe autor, hash do commit, data relativa e mensagem da linha atual.
- **Visualizador de Diff (`show_diff`)**: Overlay formatado com o diff do Git para alterações no arquivo ativo.
- Sincronização segura de `git pull` e `git push` com diálogos de confirmação.

### [ 🤖 ] [AI.AGENTS] Painel de Agentes (Slot Direito)
- Painel no lado direito para conversas com agentes de IA (`Ctrl+Shift+A`): input inline (`Enter` envia, `Esc` devolve o foco), abas por conversa (`Ctrl+N` nova, `Ctrl+W` fecha, `Ctrl+Tab` alterna) e divisor redimensionável persistido por workspace.
- Agnóstico de provedor por contrato: adapters implementam `AgentProvider` (`src/services/agents.ts`) — o provedor nativo do OpenCode e os agentes ACP entram neste slot em seguida.
- Traz um provedor `echo` de demonstração para o painel funcionar offline (testes e demos) e falha fechado com mensagem de status quando nenhum provedor está disponível.

### [ 🗎 ] [DOC.VIEW] Pré-visualização de Markdown ao Vivo
- Visualização em split dedicada para documentos Markdown e MDX (`Alt+P`).
- Renderiza títulos estruturados, tabelas formatadas com bordas, listas de marcadores e blocos de código com destaque Shiki.
- Rolagem sincronizada via teclado e roda do mouse sem modificar o texto de origem.

### [ 🖮 ] [VIM.MODAL] Edição Híbrida em Modo Duplo (CUA + Vim)
- Atalhos modernos universais (`Ctrl+C`, `Ctrl+V`, `Ctrl+Z`, `Ctrl+S`, `Ctrl+F`, `Ctrl+P`).
- Modo modal Vim opcional (`toggle_modal`):
  - Modos: Normal, Inserção, Visual, Visual-Line.
  - Movimentos: `h, j, k, l, w, b, e, 0, $, gg, G`.
  - Operadores: `dd, yy, x, p, P, u, Ctrl+R`.
  - Comandos: `:w, :wq, :q, :q!, :tasks, :health, /busca, n, N`.
- Fluxos multicursor: adicionar cursores acima/abaixo (`Ctrl+Alt+Up/Down`), selecionar todas as ocorrências e envolver seleção (`surround`).

### [ 🗖 ] [MENU.CUA] Menus com Mnemonics Windows 98 / CUA & Barra de Comandos
- **Sistema Mnemônico via Alt**: Pressionar `Alt` ou `F10` sublinha as teclas de atalho ativas (`<u><b>A</b></u>rquivo`, `<u><b>E</b></u>ditar`, etc.).
- Roteamento duplo de atalhos suportando tanto Português (`Alt+A, Alt+E, Alt+X...`) quanto Inglês (`Alt+F, Alt+E, Alt+V...`).
- Barra de Comandos Global (`Ctrl+Shift+P`): Busca fuzzy em todos os itens de menu, seletores de tema, idiomas e comandos de plugins.

### [ 🔌 ] [PLUG.HOOKS] Plugins Modulares & Executor de Tarefas
- Plugins declarativos carregados de `~/.config/daisy/plugins/` ou `.daisy/plugins/` via `plugin.toml`.
- Hooks de ciclo de vida: `on_open`, `on_save`.
- Interpolação dinâmica de caminhos: `$FILE`, `$DIR`, `$WORKSPACE`.
- Executor de tarefas de projeto via `tasks.toml` (`run_tasks`).

---

## [ C:\> ] Instalação & Início Rápido

```text
┌────────────────────────────────────────────────────────┐
│ C:\> INSTALAR.BAT --UNIVERSAL                          │
└────────────────────────────────────────────────────────┘
```

### [ 🗲 ] Instalador Interativo One-Line (Linux & macOS)

Instale o Daisy diretamente através do nosso instalador automático:

```bash
curl -fsSL https://raw.githubusercontent.com/raillen/daisy-code-editor/main/scripts/install.sh | bash
```

Instalação automatizada sem interação:

```bash
# Formato específico: appimage | arch | deb | rpm | flatpak | binary
FORMAT=appimage curl -fsSL https://raw.githubusercontent.com/raillen/daisy-code-editor/main/scripts/install.sh | bash

# Versão específica fixada
VERSION=v0.2.0 curl -fsSL https://raw.githubusercontent.com/raillen/daisy-code-editor/main/scripts/install.sh | bash
```

---

### [ 🖴 ] Pacotes Pré-compilados por Distribuição

Downloads binários diretos estão disponíveis em [GitHub Releases](https://github.com/raillen/daisy-code-editor/releases):

| Plataforma / Formato | Comando de Instalação |
|---|---|
| **AppImage Universal** | `chmod +x Daisy-x86_64.AppImage && ./Daisy-x86_64.AppImage` |
| **Arch Linux (PKGBUILD)** | `sudo pacman -U daisy-code-editor-0.2.0-1-x86_64.pkg.tar.zst` |
| **Debian / Ubuntu (.deb)** | `sudo dpkg -i daisy_0.2.0_amd64.deb` |
| **Fedora / RHEL (.rpm)** | `sudo dnf install ./daisy-code-editor-0.2.0-1.x86_64.rpm` |
| **Flatpak Bundle** | `flatpak install --bundle daisy-0.2.0.flatpak` |
| **Binário Standalone** | `chmod +x daisy-linux-x64 && sudo mv daisy-linux-x64 /usr/local/bin/daisy-code` |

---

### [ ⚙ ] Compilação a partir do Código-Fonte (Bun 1.2+)

```bash
# 1. Clonar repositório
git clone https://github.com/raillen/daisy-code-editor.git
cd daisy-code-editor

# 2. Instalar dependências
bun install

# 3. Executar testes automatizados (97 testes)
bun test

# 4. Iniciar editor em modo desenvolvimento
bun run dev

# 5. Gerar bundle de distribuição
bun run build
```

---

## [ 🖮 ] Guia de Navegação & Atalhos de Teclado

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ ATALHO              AÇÃO CORRESPONDENTE                                    │
├────────────────────────────────────────────────────────────────────────────┤
│ F10 / Alt           Alterna mnemônicos da barra de menu (letras sublinhadas)│
│ Alt+A / Alt+F       Menu Arquivo [A]rquivo / [F]ile                        │
│ Alt+E               Menu Editar [E]ditar / [E]dit                          │
│ Alt+X / Alt+V       Menu Exibir E[x]ibir / [V]iew                          │
│ Alt+C / Alt+S       Menu Configurações [C]onfigurações / [S]ettings        │
│ Alt+I / Alt+G       Menu Navegação [I]r / [G]o                             │
│ Alt+T               Menu Git Gi[t]                                         │
│ Alt+J / Alt+H       Menu Ajuda A[j]uda / [H]elp                            │
│ Ctrl+Shift+P        Barra de Comandos Global (Busca fuzzy de comandos)     │
│ Ctrl+P              Localizador Rápido de Arquivos                         │
│ Ctrl+`              Alterna Terminal POSIX Embutido (PTY)                  │
│ Ctrl+Shift+G        Alterna Modal Git / SCM do Workspace                   │
│ Ctrl+Shift+A        Alterna Painel de Agentes IA (Ctrl+N nova · Enter)     │
│ Alt+P               Alterna Pré-visualização de Markdown em Split          │
│ Ctrl+F              Localizar no Buffer Atual                              │
│ Ctrl+H              Localizar e Substituir no Buffer                       │
│ Ctrl+Shift+F        Busca no Projeto / Grep Global                         │
│ F2                  Renomear Símbolo (LSP / AST)                           │
│ F12                 Ir para Definição (LSP)                                │
│ K / Ctrl+K          Exibir Documentação em Hover (LSP)                     │
│ Alt+B               Git Blame para a Linha Atual                           │
│ Ctrl+\              Dividir Painel do Editor (Vertical)                    │
│ Ctrl+Shift+\        Dividir Painel do Editor (Horizontal)                  │
│ Alt+/               Guia Visual de Atalhos Rápidos (Which-Key)             │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## [ 💾 ] Configurações (`config.toml`)

O Daisy salva suas configurações em `~/.config/daisy/config.toml` (compatível com `.daisy/` ou `.oride/`). Abra e edite diretamente pelo Daisy em **Configurações -> Configurações do usuário (TOML)** (`open_user_config`):

```toml
# Configuração do Daisy Code Editor
theme = "daisy-lain"          # daisy-lain, daisy-lain-cobalt, daisy-lain-glitch, daisy-lain-bone, catppuccin-mocha...
locale = "pt-BR"              # "pt-BR" ou "en-US"
soft_wrap = false
auto_pairs = true
show_line_numbers = true
icons = true                  # Ativa ícones Nerd Font para arquivos
mouse = true                  # Ativa captura de mouse e rolagem

[editor]
tab_size = 2
insert_spaces = true
format_on_save = true

[tree]
width = 28
show_hidden = false

[keys]
"ctrl+s" = "save"
"ctrl+shift+p" = "command_palette"
"alt+p" = "toggle_md_preview"
```

---

## [ 🏛️ ] Arquitetura & Estrutura de Código

O Daisy segue rigorosamente os princípios de Clean Architecture:

```text
src/
  ├── core/        [DOMÍNIO PURO] document, keymap, search (0 dependências externas)
  ├── services/    [ADAPTADORES] files, git, lsp, terminal, catalogs, plugins
  ├── app/         [ORQUESTRAÇÃO] conteiner de estado, despacho de comandos, motor Vim
  ├── ui/          [APRESENTAÇÃO] componentes OpenTUI Solid (workbench, editor, temas)
  └── workers/     [WORKER BACKGROUND] indexador fuzzy de arquivos em alta escala
```

### Invariantes Centrais:
1. **Core Puro, Shell Imperativo**: `src/core` é 100% testável em memória sem necessidade de TTY ou emulador de DOM.
2. **Ações como Dados**: Cada atalho de teclado, comando da paleta ou menu despacha uma `ActionId` tipada.
3. **Design Fail-Closed**: Falhas em processos externos (LSP, Git CLI, PTY) reportam status de forma elegante sem quebrar o editor.
4. **Desacoplamento de Compiladores**: A inteligência de código roda estritamente através de binários externos no `$PATH`.

---

## [ ⚙.LOG ] Validação Automatizada

```bash
# Executar suíte completa de testes automatizados (97 testes)
bun test

# Executar checagem estrita de tipos TypeScript
bun run typecheck

# Gerar pacote de distribuição
bun run build
```

---

## [ SYS.MAP ] Roadmap & Próximas Versões

Consulte o planejamento detalhado, progresso ativo e próximas etapas desde a **v0.3.0** (Tree-Sitter, Code Folding) até a **v1.0 (LTS)** em nosso **[ROADMAP.md](ROADMAP.md)**.

---

## [ 📄 ] Licença & Créditos

Daisy Code Editor é um software de código aberto licenciado sob a **[Licença MIT](LICENSE)**.

Projetado e desenvolvido por **raillen** e colaboradores. Inspirado por *Serial Experiments Lain*, *The Wired* e pela estética brutalista de terminais hacker.
