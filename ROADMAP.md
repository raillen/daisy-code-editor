# [ SYS.MAP ] Daisy Code Editor — Official Roadmap

<p align="center">
  <b>[ English Roadmap ]</b> · <a href="README.md"><b>[ Readme ]</b></a> · <a href="README.pt-BR.md"><b>[ 🇧🇷 Documentação em Português ]</b></a>
</p>

```text
╔══════════════════════════════════════════════════════════════════════════════╗
║  DAISY CODE EDITOR // STRATEGIC EVOLUTION MAP // v0.2.0 -> v1.0.0 LTS        ║
║  "Continuous delivery over The Wired protocol."                              ║
╚══════════════════════════════════════════════════════════════════════════════╝
```

Welcome to the official roadmap for **Daisy Code Editor**, tracking the technical milestones from the stable **v0.2.0 baseline** up to the long-term stable release **v1.0 (LTS)**.

---

## [ 🏛️ ] Architectural Invariants

1. **Absolute Velocity & Zero Bloat**: Sub-40ms cold boots and rock-solid 60 FPS rendering without Chromium, Electron, or heavy web runtimes.
2. **Fine-Grained Reactivity**: Built upon `@opentui/core` and `@opentui/solid` on **TypeScript & Bun**, recalculating only dirty screen buffer cells.
3. **Decoupled & Fail-Closed**: LSP, Git, and PTY run as external asynchronous processes via `$PATH`. External process errors never crash the editor.
4. **Ergonomic Hybrid Editing**: Intuitive CUA navigation with menubars, command palette, and Alt mnemonics paired with an optional Vim modal mode.
5. **Distinct Brutalist Aesthetic**: High-contrast cyberpunk HUD themes inspired by *Serial Experiments Lain* and 90s hacker culture with terminal legibility.

---

## [ 📊 ] Milestone Summary

| Version | Primary Engineering Focus | Lifecycle Status |
|---|---|---|
| **`v0.2.0`** | **Foundation, Cyberpunk HUD & Universal Packaging** | **Delivered & Stable** [OK] |
| **`v0.3.0`** | **Tree-Sitter WASM, Code Folding & AST Ergonomics** | [NEXT MILESTONE] |
| **`v0.4.0`** | **Inline Virtual Diagnostics, Quick Fixes & Multi-LSP** | [PLANNED] |
| **`v0.5.0`** | **Side-by-Side Split Diff, Hunk Staging & Git Tooling** | [PLANNED] |
| **`v0.6.0`** | **Terminal Graphics (Kitty/Sixel) & Interactive Markdown** | [PLANNED] |
| **`v0.7.0`** | **Headless Daemon & Persistent Remote Sessions** | [PLANNED] |
| **`v0.8.0`** | **Integrated Cyberpunk AI Pair Programmer** | [PLANNED] |
| **`v1.0.0`** | **Production Stability & Long-Term API Freeze (LTS)** | [TARGET] |

---

## [ 💾 ] v0.2.0 — Current Consolidated Baseline (Delivered)

All features below are implemented, tested (97 automated tests passing), and available in release `0.2.0`:

- [x] **Reactive Terminal Engine**: Fine-grained `@opentui/solid` cell buffer layout, full mouse drag, wheel scrolling, and dynamic splits.
- [x] **CUA Menubar with Interactive Alt Mnemonics**: Pressing `Alt` or `F10` underlines active mnemonics (`<u><b>F</b></u>ile`, `<u><b>E</b></u>dit`...) with bilingual hotkey routing (`pt-BR` and `en-US`).
- [x] **Global Command Palette (`Ctrl+Shift+P`)**: Unified fuzzy search indexing all menu actions, direct theme quick-pickers, locale options, and plugin commands.
- [x] **Daisy-Lain Cyberpunk HUD Theme Suite**:
  - `daisy-lain` (The Wired — Midnight Void `#070913`, Acid Yellow `#ffe500`, Cobalt `#0025c5`, Signal Cyan `#67f4ed`).
  - `daisy-lain-cobalt` (Hero Blue — Dense Cobalt `#020d58`, Acid Yellow wireframes, Signal White `#f8f7eb`).
  - `daisy-lain-glitch` (CRT Anomaly — Laser Glitch Crimson `#ff4068`, Signal Cyan, Midnight Void).
  - `daisy-lain-bone` (Brutalist Bone Paper — Raw bone `#f2efe6`, pitch black ink, yellow accents).
  - 11 classic community themes with live preview while browsing.
- [x] **Decoupled Language Server Protocol ($PATH Client)**:
  - Contextual autocompletion with workspace word indexing and fuzzy ranking.
  - Hover documentation popup (`K` / `Ctrl+K`).
  - Go to Definition (`F12` / `gd`).
  - Symbol rename across open buffers (`F2`).
  - Format on save (`format_on_save`).
  - Compiler diagnostics aggregation (`toggle_diagnostics`).
- [x] **Integrated Git / SCM**:
  - Real-time Git Gutter markers in line number column (`+` added, `~` modified, `-` deleted).
  - Dedicated SCM workspace panel (`Ctrl+Shift+G`) with branch status and commit count.
  - Git Blame overlay on active cursor line (`Alt+B`).
  - Formatted Git Diff reader (`show_diff`).
  - Safe `git pull` and `git push` synchronization dialogs.
- [x] **Live Markdown Preview**: Split viewport rendering formatted headings, tables, and Shiki-highlighted code blocks (`Alt+P`).
- [x] **Embedded PTY Terminal**: Full interactive terminal emulator (`Ctrl+\``) running user shell (`bash`, `zsh`, `fish`).
- [x] **Vim Modal Mode**: Normal, Insert, Visual, Visual-Line modes (`h, j, k, l, w, b, e, 0, $, gg, G, dd, yy, p, u, Ctrl+R, :w, :q, :wq, /`).
- [x] **Multicursor & Advanced Editing**: Add cursor above/below (`Ctrl+Alt+Up/Down`), select all occurrences, surround selection with delimiters, linear undo tree viewer (`undo_tree`).
- [x] **Project-wide Search & Replace**: Recursive grep with glob filtering and disk replacement (`Ctrl+Shift+F`).
- [x] **Plugin Architecture & Task Runner**: TOML plugins with `on_open` and `on_save` hooks, path variables (`$FILE`, `$DIR`, `$WORKSPACE`), and task runner (`tasks.toml`).
- [x] **Universal Packaging**:
  - Interactive one-line installer (`curl ... | bash`).
  - Binary packages for Arch Linux (`PKGBUILD`), Fedora (`.spec`), Debian/Ubuntu (`.deb`), AppImage, Flatpak, and standalone binaries (`linux-x64`, `linux-arm64`).

---

## [ 🗲 ] Detailed Milestone Roadmap (Upcoming Versions)

### [ 🗲 ] v0.3.0 — Syntactic Intelligence & AST (Tree-Sitter)
*Focus: Transitioning code parsing from regular expressions to real-time abstract syntax trees.*

- [ ] **Tree-sitter WASM Engine**:
  - Incorporate WebAssembly grammars for TypeScript, JavaScript, Rust, Python, Go, C/C++, JSON, HTML, and Markdown.
  - Eliminate syntax highlighting inaccuracies in multiline strings, nested template literals, and complex expressions.
- [ ] **Semantic Code Folding**:
  - Fold and expand functions, classes, interfaces, and blocks.
  - Interactive visual indicators in the gutter column (`▾` open / `▸` folded) with standard keyboard shortcuts (`zc` fold, `zo` unfold, `zM` fold all, `zR` unfold all).
- [ ] **Indent Rainbow & Bracket Pair Colorizer**:
  - Subtle vertical indentation guides aligned with tab size.
  - High-contrast neon highlighting on matching bracket pairs (`()`, `[]`, `{}`) enclosing the cursor.
- [ ] **Vim AST Text Objects**:
  - Structural operators for modal editing: `daf` (delete around function), `cib` (change inside block), `yac` (yank around class).

---

### [ ⚙ ] v0.4.0 — Visual Diagnostics & Refactoring Ergonomics
*Focus: Bringing compiler feedback directly into developer line-of-sight.*

- [ ] **Inline Virtual Diagnostics (Ghost Text)**:
  - Render compiler and linter errors/warnings directly at the end of code lines in dim ghost text (similar to Neovim / Helix / Zed), accompanied by glyphs (`󰅚` error, `󰀪` warning).
- [ ] **LSP Code Actions & Quick Fixes (`Ctrl+.` / `Alt+Enter`)**:
  - Floating contextual menu to apply automated resolutions suggested by the language server (add missing import, fix types, implement interface members).
- [ ] **Inlay Hints**:
  - Display inferred variable types and function parameter labels directly within code text as unselectable inline ghost text.
- [ ] **Concurrent Multi-LSP per Buffer**:
  - Attach multiple language servers to a single document (e.g., `typescript-language-server` + `tailwindcss-language-server` + `biome`).

---

### [ 🖧 ] v0.5.0 — Side-by-Side Diff, Hunk Staging & Git Tooling
*Focus: Complete version control review workflows without leaving the terminal editor.*

- [ ] **Side-by-Side Split Diff Viewer**:
  - Synchronized two-column visual diff viewport comparing modified working tree against `HEAD` or historical commits.
- [ ] **Interactive Hunk Staging**:
  - Navigate between modified hunks and apply stage/unstage partially line-by-line or hunk-by-hunk directly from the editor.
- [ ] **Interactive 3-Way Merge Conflict Resolver**:
  - Automatic detection of conflict markers (`<<<<<<<`, `=======`, `>>>>>>>`) with single-key action buttons: *Accept Current*, *Accept Incoming*, *Accept Both*.
- [ ] **Inline Git Blame Ghost Text**:
  - On-demand subtle display of author, relative date, and commit message at the end of the active cursor line.

---

### [ 🖵 ] v0.6.0 — Terminal Graphics & Interactive Markdown
*Focus: Breaking the text-only barrier with modern terminal graphics protocols.*

- [ ] **Kitty Graphics & Sixel Protocols**:
  - Render high-resolution images inside Markdown preview splits and file explorer previews on compatible terminals (Kitty, WezTerm, Ghostty, Alacritty with Sixel).
- [ ] **Interactive Markdown Preview**:
  - Clickable task checkboxes (`[ ]` / `[x]`) that toggle the source file state instantly.
  - Markdown tables with automatic border alignment.
- [ ] **Mermaid Diagrams & KaTeX Formulas**:
  - Render Mermaid diagrams and simplified math formulas directly in the preview pane.

---

### [ 🔌 ] v0.7.0 — Daisy Daemon & Persistent Remote Sessions
*Focus: Distributed productivity, session resilience, and remote terminal workflows.*

- [ ] **Headless Daemon Mode (`daisy daemon`)**:
  - Run the editor core as a detached background service on local or remote servers over Unix sockets or encrypted TCP tunnels.
- [ ] **Instant Session Reconnection (`daisy attach`)**:
  - Full persistence of open buffers, cursors, undo history, and running PTY shells when SSH drops or terminal windows close.
- [ ] **Multi-client Terminal Pairing**:
  - Simultaneous shared sessions allowing two developers to edit the same workspace with synchronized viewports and distinct cursor colors.

---

### [ 🤖 ] v0.8.0 — Cyberpunk AI Pair Programmer (Daisy Copilot)
*Focus: Native, contextual integration with local and cloud generative models.*

- [ ] **Cyberpunk AI HUD Sidebar**:
  - Conversational terminal chat pane supporting local models via Ollama as well as remote APIs (Anthropic Claude, OpenAI, Google Gemini).
- [ ] **Inline AI Diff Generation**:
  - Code refactoring and generation presented as interactive diffs directly inside the editor buffer, accepted with `Ctrl+Y` or rejected with `Ctrl+N`.
- [ ] **Context Injection**:
  - Automatic inclusion of type signatures, active compiler diagnostics, and selected code ranges into the prompt context.

---

### [ 🎯 ] v1.0.0 — Production Stability & LTS Freeze
*Focus: Rigorous quality assurance, enterprise stability, and long-term API commitments.*

- [ ] **Plugin API & Configuration Schema Freeze**:
  - Zero breaking changes guaranteed across minor releases for configuration files and plugins.
- [ ] **Unicode & CJK Accessibility Hardening**:
  - Perfect cell alignment for double-width CJK characters, complex emoji sequences, and terminal screen readers.
- [ ] **Automated CI Latency Benchmarks**:
  - Continuous enforcement of sub-30ms startup budgets and sub-40MB idle memory footprints.
- [ ] **Comprehensive Bilingual Documentation**:
  - Exhaustive user manuals, keyboard cheat sheets, and API documentation in English and Portuguese.

---

## [ 🤝 ] Contributing

Interested in contributing to any of the roadmap milestones?
1. Check active tasks on [GitHub Issues](https://github.com/raillen/daisy-code-editor/issues).
2. Consult the engineering guide in [README.md](README.md) and [AGENTS.md](AGENTS.md).
3. Submit a Pull Request focused on a single milestone feature with full automated test coverage.
