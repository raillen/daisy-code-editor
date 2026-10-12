# 🌼 Daisy Code Editor

[![Runtime: Bun](https://img.shields.io/badge/Runtime-Bun%201.2+-fbf0df?logo=bun&logoColor=black)](https://bun.sh)
[![Language: TypeScript](https://img.shields.io/badge/Language-TypeScript%205.9+-3178c6?logo=typescript&logoColor=white)](https://www.typescriptlang.org)
[![UI: SolidJS](https://img.shields.io/badge/UI-SolidJS%201.9+-2c4f7c?logo=solid&logoColor=white)](https://www.solidjs.com)
[![Engine: OpenTUI](https://img.shields.io/badge/Engine-OpenTUI-blueviolet)](https://github.com/opentui)
[![Version](https://img.shields.io/badge/Versão-0.2.0-brightgreen)](https://github.com/raillen/daisy-code-editor)
[![License: MIT](https://img.shields.io/badge/Licença-MIT-blue.svg)](LICENSE)

> **Daisy Code Editor** é um editor de código de terminal e mini-IDE moderno, modular e extremamente rápido, construído com **OpenTUI + SolidJS** em **TypeScript & Bun**.

Projetado para alta performance, ergonomia de teclado e arquitetura limpa, o Daisy oferece uma experiência completa de desenvolvimento diretamente no terminal, sem qualquer inchaço de navegador.

[English](README.md) · **Português**

---

## ✨ Funcionalidades

- ⚡ **Motor de Terminal Reativo**: Construído sobre `@opentui/core` e `@opentui/solid`, com reatividade fina, layout rápido de linhas ASCII e renderização fluida.
- 📁 **Navegador de Arquivos & Abas**: Árvore de arquivos de projeto com ícones Nerd Font, gerenciamento de múltiplas abas com fechamento via botão do meio do mouse e suporte a divisão de janelas (splits).
- 🖥️ **Terminal PTY Integrado**: Emulador de terminal interativo com sequências ANSI/VT, suporte a painéis divididos e divisores arrastáveis.
- 🧠 **Inteligência de Código & Autocompletion**:
  - **LSP sob Demanda**: Integração com Language Server Protocol via `$PATH` (`typescript-language-server`, `rust-analyzer`, `gopls`, `pyright`, `clangd`).
  - **Inteligência Offline**: Catálogo integrado de linguagens (palavras-chave, funções built-in e documentações) e indexação de palavras do buffer com ranking difuso.
  - **Snippets Inteligentes**: Expansão de tabstops (`$0`) para TypeScript, Rust, Go e Python.
  - **Ações de Código & Refatoração**: Renomeação de símbolos no buffer por palavra inteira (`F2`) e documentação ao passar o cursor (`Ctrl+K`).
- 🎨 **Realce de Sintaxe & Temas**: Tokenizador léxico alimentado pelo Shiki com suporte a TypeScript, JavaScript, Rust, Go, Python, C, Bash, Markdown, HTML, CSS, Lua e outros. Temas embutidos incluem Catppuccin, Nord, Tokyo Night, Gruvbox e One Dark.
- 🔍 **Busca & Substituição de Alta Performance**:
  - Busca e substituição no buffer com regex, sensibilidade a maiúsculas, desconsideração de acentos e correspondência de palavra inteira.
  - Busca recursiva e substituição em disco em todo o projeto.
  - Web Worker dedicado em segundo plano para busca fuzzy em projetos com mais de 20.000 arquivos sem travamento de interface.
- 🐙 **Integração com Git / SCM**: Indicadores de alteração no gutter (adicionado/modificado/removido), stage/unstage, commit, visualização interativa de diff e overlay de git blame (`Ctrl+Alt+B`).
- 📖 **Pré-visualização de Markdown**: Preview ao vivo lado a lado ou em painel, com tabelas formatadas, blocos de código com realce sintático.
- ⚙️ **Menu de Configurações Interativo**:
  - Menu suspenso para alternar auto-wrap, fechamento automático de pares, números de linha, modo modal, ícones e captura de mouse.
  - Abertura direta dos arquivos `config.toml` de usuário e de workspace em abas com recarregamento a quente ao salvar.
- ⌨️ **Modo Modal Vim & Teclas Reconfiguráveis**: Modo Vim normal/inserção/visual opcional (`h, j, k, l, w, b, gg, G, x, u, :w, :q`) além de atalhos convencionais intuitivos.
- 🌐 **Suporte Multilíngue**: Internacionalização nativa (`en-US` e `pt-BR`) para menus, badges de status e notificações.

---

## 🚀 Início Rápido & Instalação

### ⚡ Instalador Interativo One-Line (Linux & macOS)

Instale o Daisy diretamente com nosso instalador interativo que permite escolher a versão e o formato de pacote desejado:

```bash
curl -fsSL https://raw.githubusercontent.com/raillen/daisy-code-editor/main/scripts/install.sh | bash
```

Para instalações automatizadas sem interação:
```bash
# Escolha o formato: appimage, arch, deb, rpm, flatpak, ou binary
FORMAT=appimage curl -fsSL https://raw.githubusercontent.com/raillen/daisy-code-editor/main/scripts/install.sh | bash

# Escolha a versão específica do release
VERSION=v0.2.0 curl -fsSL https://raw.githubusercontent.com/raillen/daisy-code-editor/main/scripts/install.sh | bash
```

### 📦 Pacotes Pré-compilados por Distribuição

Todos os pacotes estão disponíveis para download direto no [GitHub Releases](https://github.com/raillen/daisy-code-editor/releases):
- **AppImage Universal**: `chmod +x Daisy-*.AppImage && ./Daisy-*.AppImage`
- **Arch Linux**: `sudo pacman -U daisy-code-editor-*.pkg.tar.zst`
- **Debian / Ubuntu**: `sudo dpkg -i daisy_*_amd64.deb`
- **Fedora / RHEL**: `sudo dnf install ./daisy-code-editor-*.rpm`
- **Flatpak**: `flatpak install --bundle daisy-*.flatpak`

### 🛠️ Executando a partir do Código-Fonte

**Pré-requisitos:** [Bun](https://bun.sh) (v1.2+)

```bash
git clone https://github.com/raillen/daisy-code-editor.git
cd daisy-code-editor
bun install
```

### Executando o Daisy

```bash
# Iniciar o Daisy no diretório atual
bun run start

# Ou rodar diretamente com a CLI
bun ./src/cli.tsx .

# Abrir um arquivo específico
bun ./src/cli.tsx src/core/document.ts
```

---

## ⌨️ Atalhos Essenciais

| Atalho | Ação |
|---|---|
| **Edição** | |
| `Ctrl+S` | Salvar arquivo ativo |
| `Ctrl+Shift+S` | Salvar como... (navegador interativo de caminhos) |
| `Ctrl+Z` / `Ctrl+Y` | Desfazer / Refazer (histórico atômico com multicursor) |
| `Ctrl+C` / `Ctrl+X` / `Ctrl+V` | Copiar / Recortar / Colar |
| `Ctrl+/` | Alternar comentário de linha |
| `Alt+Z` | Alternar quebra suave de linha (soft wrap) |
| `Ctrl+G` | Ir para linha (`linha:coluna`) |
| **Navegação & Layout** | |
| `Ctrl+P` | Abertura rápida de arquivo (busca fuzzy) |
| `Ctrl+Shift+P` | Paleta de comandos |
| `Ctrl+N` / `Ctrl+W` | Nova aba / Fechar aba |
| `Ctrl+PgUp` / `Ctrl+PgDn` | Aba anterior / Próxima aba |
| `Ctrl+Shift+O` | Seletor de abas abertas (buffer picker) |
| `Ctrl+B` | Alternar árvore de arquivos do projeto |
| `Ctrl+` ` | Alternar terminal PTY embutido |
| `Ctrl+Alt+V` / `Ctrl+Alt+H` | Dividir editor verticalmente / horizontalmente |
| `Ctrl+Alt+W` | Fechar painel dividido |
| **Busca & Substituição** | |
| `Ctrl+F` / `F3` | Buscar no buffer / Próxima ocorrência |
| `Ctrl+H` | Substituir no buffer |
| `Ctrl+Shift+F` | Buscar e Substituir no Projeto |
| **Inteligência de Código** | |
| `Ctrl+Space` | Disparar dropdown de autocomplete |
| `Ctrl+K` | Exibir documentação de hover |
| `F4` | Ir para definição |
| `F2` | Renomear símbolo (fallback por palavra no buffer) |
| `Ctrl+Shift+I` | Formatar documento |
| **Git & Controle de Versão** | |
| `Ctrl+Shift+G` | Alternar barra lateral do Git / SCM |
| `Ctrl+Alt+B` | Overlay de Git blame para a linha atual |
| **Menus & Ajuda** | |
| `Alt+F`, `Alt+E`, `Alt+V`, `Alt+S`, `Alt+G`, `Alt+H` | Acessar menus (Arquivo, Editar, Exibir, Configurações, Ir, Ajuda) |
| `Alt+/` | Which-Key (lembrete de atalhos rápidos) |

---

## 🛠️ Configuração

O Daisy carrega configurações em camadas a partir de:
1. Global do usuário: `~/.config/daisy/config.toml` (compatível com `.oride/config.toml`)
2. Local do workspace: `.daisy/config.toml`

Você pode abrir e editar suas configurações a qualquer momento pelo menu **Configurações** ou pela paleta de comandos. As alterações são recarregadas automaticamente ao salvar.

```toml
[editor]
theme = "catppuccin-mocha"
locale = "pt-BR"
mouse = false
icons = true
line_numbers = true
auto_wrap = true
auto_pairs = true
modal_mode = false

[keymap]
save = "Ctrl+S"
command_palette = "Ctrl+Shift+P"
goto_line = "Ctrl+G"

[lsp]
typescript = ["typescript-language-server", "--stdio"]
rust = ["rust-analyzer"]
go = ["gopls"]
python = ["pyright-langserver", "--stdio"]
```

---

## 📐 Arquitetura & Clean Code

O Daisy é estruturado de acordo com os princípios de Clean Architecture e modularidade estrita:

```text
src/
  ├── core/        # Núcleo puro de domínio (document, keymap, search) — 0 dependências externas
  ├── services/    # Adaptadores de infraestrutura (files, git, lsp, terminal, catalogs)
  ├── app/         # Container de estado, despacho de comandos e motor Vim
  ├── ui/          # Componentes OpenTUI Solid (workbench, editor, statusbar, overlays)
  └── workers/     # Web Worker dedicado em background para busca fuzzy
```

---

## 🧪 Testes & Validação

```bash
# Rodar suíte completa de testes automatizados (94 testes)
bun test

# Verificação estática de tipos com TypeScript
bun run typecheck

# Gerar build de distribuição
bun run build
```

---

## 📄 Licença

O Daisy Code Editor é um software de código aberto sob a licença [MIT](LICENSE).
