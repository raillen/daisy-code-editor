# 🌼 Daisy Code Editor

[![Runtime: Bun](https://img.shields.io/badge/Runtime-Bun%201.2+-fbf0df?logo=bun&logoColor=black)](https://bun.sh)
[![Language: TypeScript](https://img.shields.io/badge/Language-TypeScript%205.9+-3178c6?logo=typescript&logoColor=white)](https://www.typescriptlang.org)
[![UI: SolidJS](https://img.shields.io/badge/UI-SolidJS%201.9+-2c4f7c?logo=solid&logoColor=white)](https://www.solidjs.com)
[![Engine: OpenTUI](https://img.shields.io/badge/Engine-OpenTUI-blueviolet)](https://github.com/opentui)
[![Version](https://img.shields.io/badge/Version-0.2.0-brightgreen)](https://github.com/raillen/daisy-code-editor)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> **Daisy Code Editor** is a modern, modular, and blazing-fast terminal code editor and mini-IDE built with **OpenTUI + SolidJS** on top of **TypeScript & Bun**.

Designed for high performance, keyboard ergonomics, and clean architecture, Daisy delivers a full-featured development environment right inside your terminal with zero browser bloat.

---

## ✨ Features

- ⚡ **Reactive Terminal Engine**: Built on `@opentui/core` and `@opentui/solid` with fine-grained reactivity, fast ASCII line layout, and smooth rendering.
- 📁 **File Explorer & Tabbed Buffers**: Navigable project file tree with Nerd Font glyphs, multi-tab buffer management, middle-click tab closing, and split editor views.
- 🖥️ **Embedded PTY Terminal**: Full interactive terminal emulator with ANSI/VT escape sequences, split-pane support, and draggable dividers.
- 🧠 **Code Intelligence & Autocomplete**:
  - **On-Demand LSP**: Language Server Protocol integration via `$PATH` (`typescript-language-server`, `rust-analyzer`, `gopls`, `pyright`, `clangd`).
  - **Offline Intelligence**: Built-in language catalog (keywords, functions, docstrings) and multi-buffer word indexing with fuzzy ranking.
  - **Smart Snippets**: Tabstop expansions (`$0`) for TypeScript, Rust, Go, and Python.
  - **Code Actions & Refactoring**: Local whole-word symbol rename (`F2`) and hover documentation (`Ctrl+K`).
- 🎨 **Syntax Highlighting & Theming**: Shiki-powered lexical highlighting supporting TypeScript, JavaScript, Rust, Go, Python, C, Bash, Markdown, HTML, CSS, Lua, and more. Built-in themes include Catppuccin, Nord, Tokyo Night, Gruvbox, and One Dark.
- 🔍 **High-Performance Search & Replace**:
  - Buffer-level find & replace with regex, case-sensitivity, accent-folding, and whole-word matching.
  - Project-wide recursive search & disk replacement.
  - Dedicated background Web Worker for fuzzy finding across 20,000+ files without UI lag.
- 🐙 **Git / SCM Integration**: Status gutter indicators (added/modified/deleted), staging/unstaging, commits, interactive diff view, and git blame overlay (`Ctrl+Alt+B`).
- 📖 **Markdown Preview**: Real-time side-by-side or pane preview with formatted tables, fenced code blocks, and syntax highlighting.
- ⚙️ **Config & Settings Menu**:
  - Interactive settings dropdown to toggle auto-wrap, auto-closing pairs, line numbers, modal mode, nerd icons, and mouse capture.
  - Direct opening of user and workspace `config.toml` files in editor tabs with live hot reload upon saving.
- ⌨️ **Vim Modal Mode & Rebindable Chords**: Optional Vim normal/insert/visual mode (`h, j, k, l, w, b, gg, G, x, u, :w, :q`) alongside standard intuitive shortcuts.
- 🌐 **Multilingual Support**: First-class internationalization (`en-US` and `pt-BR`) for menus, status badges, and notifications.

---

## 🚀 Quickstart & Installation

### ⚡ One-Line Universal Installer (Linux & macOS)

Install Daisy directly with our interactive installer that lets you choose the release version and installation format:

```bash
curl -fsSL https://raw.githubusercontent.com/raillen/daisy-code-editor/main/scripts/install.sh | bash
```

For automated or headless installs:
```bash
# Choose format: appimage, arch, deb, rpm, flatpak, or binary
FORMAT=appimage curl -fsSL https://raw.githubusercontent.com/raillen/daisy-code-editor/main/scripts/install.sh | bash

# Choose specific release version
VERSION=v0.2.0 curl -fsSL https://raw.githubusercontent.com/raillen/daisy-code-editor/main/scripts/install.sh | bash
```

### 📦 Precompiled Distribution Packages

Direct binary packages are available on [GitHub Releases](https://github.com/raillen/daisy-code-editor/releases):
- **Universal AppImage**: `chmod +x Daisy-*.AppImage && ./Daisy-*.AppImage`
- **Arch Linux**: `sudo pacman -U daisy-code-editor-*.pkg.tar.zst`
- **Debian / Ubuntu**: `sudo dpkg -i daisy_*_amd64.deb`
- **Fedora / RHEL**: `sudo dnf install ./daisy-code-editor-*.rpm`
- **Flatpak**: `flatpak install --bundle daisy-*.flatpak`

### 🛠️ Running from Source

**Prerequisites:** [Bun](https://bun.sh) (v1.2+)

```bash
git clone https://github.com/raillen/daisy-code-editor.git
cd daisy-code-editor
bun install
```

### Running Daisy

```bash
# Open Daisy in current directory
bun run start

# Or launch directly with daisy CLI
bun ./src/cli.tsx .

# Open a specific file
bun ./src/cli.tsx src/core/document.ts
```

---

## ⌨️ Essential Keybindings

| Keybinding | Action |
|---|---|
| **Editing** | |
| `Ctrl+S` | Save active file |
| `Ctrl+Shift+S` | Save as... (interactive path browser) |
| `Ctrl+Z` / `Ctrl+Y` | Undo / Redo (atomic multi-cursor history) |
| `Ctrl+C` / `Ctrl+X` / `Ctrl+V` | Copy / Cut / Paste |
| `Ctrl+/` | Toggle line comment |
| `Alt+Z` | Toggle soft word wrap |
| `Ctrl+G` | Go to line (`line:col`) |
| **Navigation & Layout** | |
| `Ctrl+P` | Quick file open (fuzzy finder) |
| `Ctrl+Shift+P` | Command palette |
| `Ctrl+N` / `Ctrl+W` | New tab / Close tab |
| `Ctrl+PgUp` / `Ctrl+PgDn` | Previous tab / Next tab |
| `Ctrl+Shift+O` | Buffer picker (switch open tabs) |
| `Ctrl+B` | Toggle project file tree |
| `Ctrl+` ` | Toggle embedded PTY terminal |
| `Ctrl+Alt+V` / `Ctrl+Alt+H` | Split editor vertically / horizontally |
| `Ctrl+Alt+W` | Close split pane |
| **Search & Replace** | |
| `Ctrl+F` / `F3` | Find in buffer / Next match |
| `Ctrl+H` | Replace in buffer |
| `Ctrl+Shift+F` | Find & Replace in Project |
| **Code Intelligence** | |
| `Ctrl+Space` | Trigger autocomplete dropdown |
| `Ctrl+K` | Show hover documentation |
| `F4` | Go to definition |
| `F2` | Rename symbol (whole word buffer fallback) |
| `Ctrl+Shift+I` | Format document |
| **Git & Version Control** | |
| `Ctrl+Shift+G` | Toggle Git / SCM sidebar |
| `Ctrl+Alt+B` | Git blame overlay for current line |
| **Menus & Help** | |
| `Alt+F`, `Alt+E`, `Alt+V`, `Alt+S`, `Alt+G`, `Alt+H` | Access Menubar (File, Edit, View, Settings, Go, Help) |
| `Alt+/` | Which-Key (quick shortcut cheat sheet) |

---

## 🛠️ Configuration

Daisy loads layered configuration from:
1. User global: `~/.config/daisy/config.toml` (compatible with `.oride/config.toml`)
2. Workspace local: `.daisy/config.toml`

You can open and edit your configuration files at any time via the **Settings** menu or command palette. Changes reload automatically when saved.

```toml
[editor]
theme = "catppuccin-mocha"
locale = "en-US"
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

## 📐 Architecture & Clean Code

Daisy is structured according to Clean Architecture and strict modularity principles:

```text
src/
  ├── core/        # Pure domain core (document, keymap, search) — 0 external dependencies
  ├── services/    # Infrastructure adapters (files, git, lsp, terminal, catalogs)
  ├── app/         # Application state container, command dispatchers, Vim engine
  ├── ui/          # OpenTUI Solid components (workbench, editor, statusbar, overlays)
  └── workers/     # Dedicated background Web Worker for fuzzy finding
```

### Architectural Invariants

1. **Pure Core, Imperative Shell**: `src/core` is 100% pure and testable in memory without any TTY or DOM emulation.
2. **Actions as Data**: Every keybinding, palette command, and menu item dispatches a declarative `ActionId`.
3. **Fail-Closed Design**: External process failures (LSP, Git CLI, PTY) report status gracefully and never crash the editor.
4. **Strict Unidirectional Dependencies**: High-level modules never depend on presentation layers.

---

## 🧪 Testing & Validation

Daisy includes an exhaustive test suite covering unit tests, integration workflows, OpenTUI rendering frames, and microbenchmarks:

```bash
# Run full automated test suite (94 tests)
bun test

# Run TypeScript typechecker
bun run typecheck

# Build distribution bundle
bun run build
```

---

## 📄 License

Daisy Code Editor is open-source software licensed under the [MIT License](LICENSE).
