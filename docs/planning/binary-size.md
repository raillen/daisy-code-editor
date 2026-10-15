# Binary size analysis — Daisy Code Editor

## 1. Medição atual (out/2026)

| Artefato | Tamanho | Notas |
|---|---:|---|
| `bun` 1.4.2 runtime | 76,0 MB | Base completa TS/JS runtime |
| `dist/cli.js` bundle app | 0,9 MB | OpenTUI + Solid + LSP bridge |
| `dist/cli.js.map` | 1,7 MB | sourcemap (não incluído em compile?) |
| `/usr/bin/daisy` (sistema) | 110,2 MB | binary not stripped |
| `~/.local/bin/daisy` (standalone, local) | 94,2 MB | compiled-ish, not stripped |
| `nvim` (sistema) | 5,4 MB | native, stripped, minimal deps |
| LazyVim | N/A | configuração sobre Neovim, não binário; plugins Lua/system runtime |
| Frontend deps as bundled (node_modules) | 120 MB dev | não embutido no binário final (external) |

## 2. Causa real

1. **Bun runtime embedded** (runtime JS completo) ~60–75MB em todos os `bun --compile` binários.
2. **Source/app code** ~2.7MB, pesado principalmente de Shiki + OpenTUI + tree-sitter-ish highlight logic, 6.6k LOC TUI.
3. **Native shared libs** (libc, pthread, etc) em `ldd`, mas são externas do OS e não contadas.
4. **Different bits**: o `.deb` embutido `/usr/bin/daisy` era 110MB com símbolos; local `daisy` stripped um pouco só (-49KB).
5. AppImage interno extrai um binário e armazena payload (`squashfs-root/usr/bin/daisy` + metadata) → maior ainda.

Neovim é nativo (C/Lua), sem runtime JS, por isso 5MB. LazyVim não tem “binário”: binário permanece 5MB + plugins (20-80MB de runtime Lua/C).

## 3. Alternativas (trade-offs)

### (A) Retirar Bun do payload, distribuir `.js` + runtime do sistema
- Tamanho final instalado: ~8MB (app) + Bun (≈70MB) instalado via OS/package manager.
- Vantagem: tiny app footprint, updates independentes do runtime.
- Desvantagem: depende de bun instalado; menos “one-line standalone”.

### (B) Build with `--bytecode` / `--compile` slim? Bun já é o runtime base.
- Não resolve ordem de grandeza; font code still embedded.
- Pode reduzir (~10% a 20%) com sourcemaps e minify.

### (C) Usar `upx`/compressor externo (compress best + lzma)
- Ex. compress binário de 94MB → ~30–40MB (estimativa).
- Desvantagens: startup 0.5–2s adicional; antivírus/startup warnings; test rigoroso.
- Not a code change; installer can compress after build.

### (D) Portar a UI a uma stack nativa (Rust/Go/Nim/Zig)
- Recorte: perda de ecossistema rapid, alto TTM, mas binary ~3–8MB (estilo nvim) e startup ~5ms.
- Menos dependências externas.
- Custo: meses e perda de TypeScript team.

### (E) Minify config + bundle only needed node_modules externals
- Já usamos external `@opentui/core`, `shiki`; source minificado (`Bun.build minify`) pode reduzir 0,9MB → 0,3MB (3x) mas impacto pequeno.

### (F) Split: server runtime + thin CLI stub
- `daisy` stub (~2MB) invoca `/usr/share/daisy-code-editor/runtime/bun` + `dist/cli.js`.
- Permite runtime partilhado entre installs (Bun installed deps).
- Des vantagem: versioning de runtime e install.

### (G) Use lighter JS runtime than Bun
- QuickJS/Duktape ARM/~5MB (mobile) não suporta TS/JSX native APIs (OpenTUI NAPI).
- Node SEA ~22MB base + deps (better than bun 76? node binary 20MB, dependencies includes tree).
- Deno compiled binary ~30MB.
- Objeto mínimo em Node pode melhorar 4–6x.

## 4. Recomendações

1. **Curto prazo**: strip minify (`compress` true in Bun.build) + drop sourcemaps = -1.7MB (ganho próximo).
2. **Installer/market**: continuar AppImage composto? Considerar distribuir stub C + system runtime, reduz download. No parcial.
3. **Médio prazo**: habilitar `upx --ultra-brute` em build pipeline e documentar.
4. **Longo prazo**: libervar da TypeScript migration. Go/Rust data model (há legado) pode virar backend-only, e UI thin? Não hoje.

## 5. Experiência em análise

- `nvim` é reference de tiny binary: 5MB, mas sua extensibilidade requer ecosystem Lua + runtime. LazyVim total download/install é ~ alguns MB de luarc + config.
- Daisy code data (6.6k LOC + 2.6k deps) está sobrepeso por runtime e deps, não por código próprio.
- 32MiB files, history cap e terminal scrollback (2000 linhas) não contribuem significativamente para binary size (runtime arrays).

## 6. Proposta de medida empírica (não implementada)

- `Bun.build minify:true` benchmark.
- `upx -9 --brute` benchmark (installed size, cold start).
- Strip + UPX compare.
- Montar 4 packaging modes:
  1. standalone bun (atual, 94MB)
  2. extern dep (1MB + dependency install)
  3. upx compress (~30MB)
  4. Node SEA (~20MB + native deps)

Como decisão de roadmap: curto prazo + compressão, médio prazo por heavy analysis.