// Ícones Nerd Fonts para arquivos e pastas — glifos decorativos, nunca semântica.
//
// Curadoria de nvim-web-devicons (MIT, nvim-tree/nvim-web-devicons,
// lua/nvim-web-devicons/default/*.lua): só o glifo, sem a cor — as linhas da
// árvore e das abas têm uma cor só, e um ícone que some no tema no-color seria
// estado comunicado só por cor. Pastas usam Font Awesome via Nerd Fonts.
// "Primo" marca o que o upstream não tem e herdou do parente mais próximo.
//
// Sem Nerd Font instalada o terminal mostra tofu, e nada se perde: o nome do
// arquivo continua lá. Por isso o default é desligado ([ui] icons = false).
export const folderClosed = "\u{f07b}"
export const folderOpen = "\u{f07c}"
export const defaultFile = "\u{f15b}"
export const gitBranch = "\u{e725}"
export const gitBranchAlt = "\u{f418}"
export const lspGlyph = "\u{f0352}"
export const diagnosticError = "\u{f0048}"
export const diagnosticWarning = "\u{f071}"
export const searchGlyph = "\u{f002}"
export const commandGlyph = "\u{f013}"
export const tabClose = "×"
export const tabDirty = "●"
export const treeGuide = "│ "
const byExtension: Record<string, string> = {
  "rs": "\u{e68b}",
  "c": "\u{e61e}",
  "h": "\u{f0fd}",
  "cpp": "\u{e61d}",
  "hpp": "\u{f0fd}",
  "cc": "\u{e61d}",
  "sh": "\u{e795}",
  "bash": "\u{e760}",
  "fish": "\u{e795}",
  "zsh": "\u{e795}",
  "md": "\u{f48a}",
  "mdx": "\u{f48a}",
  "html": "\u{e736}",
  "css": "\u{e6b8}",
  "scss": "\u{e603}",
  "js": "\u{e60c}",
  "jsx": "\u{e625}",
  "mjs": "\u{e60c}",
  "cjs": "\u{e60c}",
  "ts": "\u{e628}",
  "tsx": "\u{e7ba}",
  "mts": "\u{e628}",
  "cts": "\u{e628}",
  "d.ts": "\u{e628}",
  "py": "\u{e606}",
  "pyi": "\u{e606}",
  "d": "\u{e7af}",
  "lua": "\u{e620}",
  "nim": "\u{e677}",
  "rb": "\u{e791}",
  "erb": "\u{e60e}",
  "go": "\u{e627}",
  "json": "\u{e60b}",
  "jsonc": "\u{e60b}",
  "toml": "\u{e6b2}",
  "yaml": "\u{e8eb}",
  "yml": "\u{e8eb}",
  "xml": "\u{f05c0}",
  "svg": "\u{f0721}",
  "png": "\u{e60d}",
  "jpg": "\u{e60d}",
  "jpeg": "\u{e60d}",
  "gif": "\u{e60d}",
  "webp": "\u{e60d}",
  "ico": "\u{e60d}",
  "woff": "\u{f031}",
  "woff2": "\u{f031}",
  "ttf": "\u{f031}",
  "otf": "\u{f031}",
  "eot": "\u{f031}",
  "mp4": "\u{e69f}",
  "mov": "\u{e69f}",
  "mkv": "\u{e69f}",
  "webm": "\u{e69f}",
  "mp3": "\u{f001}",
  "wav": "\u{f001}",
  "flac": "\u{f001}",
  "ogg": "\u{f001}",
  "m4a": "\u{f001}",
  "aac": "\u{f001}",
  "pdf": "\u{eaeb}",
  "zip": "\u{f410}",
  "7z": "\u{f410}",
  "rar": "\u{f410}",
  "gz": "\u{f410}",
  "bz2": "\u{f410}",
  "xz": "\u{f410}",
  "tgz": "\u{f410}",
  "zst": "\u{f410}",
  "exe": "\u{eae8}",
  "dll": "\u{eb9c}",
  "so": "\u{eb9c}",
  "o": "\u{eae8}",
  "a": "\u{eb9c}",
  "out": "\u{eae8}",
  "bin": "\u{eae8}",
  "wasm": "\u{e6a1}",
  "nix": "\u{f313}",
  "vue": "\u{e6a0}",
  "svelte": "\u{e697}",
  "astro": "\u{e6b3}",
  "prisma": "\u{e684}",
  "graphql": "\u{f20e}",
  "gql": "\u{f20e}",
  "sql": "\u{e706}",
  "sqlite3": "\u{e706}",
  "tex": "\u{e69b}",
  "vim": "\u{e62b}",
  "swift": "\u{e755}",
  "kt": "\u{e634}",
  "kts": "\u{e634}",
  "java": "\u{e738}",
  "cs": "\u{f031b}",
  "php": "\u{e608}",
  "pl": "\u{e769}",
  "r": "\u{f07d4}",
  "jl": "\u{e624}",
  "hs": "\u{e61f}",
  "elm": "\u{e62c}",
  "ex": "\u{e62d}",
  "exs": "\u{e62d}",
  "scala": "\u{e737}",
  "ps1": "\u{f0a0a}",
  "less": "\u{e614}",
  "sass": "\u{e603}",
  "styl": "\u{e600}",
  "hbs": "\u{e60f}",
  "mustache": "\u{e60f}",
  "twig": "\u{e61c}",
  "blade.php": "\u{f2f7}",
  "txt": "\u{f0219}",
  "log": "\u{f0331}",
  "lock": "\u{e672}",
  "ini": "\u{e615}",
  "cfg": "\u{e615}",
  "conf": "\u{e615}",
  "env": "\u{f462}",
  "mk": "\u{e779}",
  "cmake": "\u{e794}",
  "obj": "\u{f01a7}",
  "lib": "\u{eb9c}",
  "jar": "\u{e738}",
  "apk": "\u{e70e}",
  "iso": "\u{e271}",
  "torrent": "\u{f019}",
  "db": "\u{e706}",
  "sqlite": "\u{e706}",
  "pyc": "\u{e606}",
  "bak": "\u{f006f}",
  "patch": "\u{e728}",
  "diff": "\u{e728}",
  "tar": "\u{f410}", // primo de zip (ausente no upstream)
  "class": "\u{e738}", // primo de java (ausente no upstream)
  "war": "\u{e738}", // primo de java (ausente no upstream)
  "deb": "\u{f410}", // primo de zip (ausente no upstream)
  "rpm": "\u{f410}", // primo de zip (ausente no upstream)
  "dmg": "\u{e271}", // primo de iso (ausente no upstream)
  "fon": "\u{f031}", // primo de ttf (ausente no upstream)
  "mpg": "\u{e69f}", // primo de mp4 (ausente no upstream)
  "mpeg": "\u{e69f}", // primo de mp4 (ausente no upstream)
  "flv": "\u{e69f}", // primo de mp4 (ausente no upstream)
  "mka": "\u{e69f}", // primo de mp4 (ausente no upstream)
  "mid": "\u{f001}", // primo de mp3 (ausente no upstream)
  "midi": "\u{f001}", // primo de mp3 (ausente no upstream)
  "gem": "\u{e791}", // primo de rb (ausente no upstream)
  "phar": "\u{e608}", // primo de php (ausente no upstream)
  "swp": "\u{f0219}", // primo de txt (ausente no upstream)
  "swo": "\u{f0219}", // primo de txt (ausente no upstream)
  "tmp": "\u{f0219}", // primo de txt (ausente no upstream)
  "temp": "\u{f0219}", // primo de txt (ausente no upstream)
  "orig": "\u{e728}", // primo de diff (ausente no upstream)
  "rej": "\u{e728}", // primo de diff (ausente no upstream)
}
const byFilename: Record<string, string> = {
  "dockerfile": "\u{f0868}",
  ".dockerignore": "\u{f0868}",
  ".gitignore": "\u{e702}",
  "gemfile": "\u{e791}",
  "package.json": "\u{e71e}",
  "go.mod": "\u{e627}",
  "tsconfig.json": "\u{e69d}",
  "makefile": "\u{e779}",
  "gnumakefile": "\u{e779}",
  "docker-compose.yml": "\u{f0868}",
  "docker-compose.yaml": "\u{f0868}",
  "compose.yml": "\u{f0868}",
  "compose.yaml": "\u{f0868}",
  ".vimrc": "\u{e62b}",
  "_vimrc": "\u{e62b}",
  ".gvimrc": "\u{e62b}",
  ".bashrc": "\u{e615}",
  ".zshrc": "\u{e615}",
  ".bash_profile": "\u{e615}",
  ".bash_logout": "\u{e615}",
}
// Nome exato, depois minúsculas; sufixo de duas partes ("d.ts") antes da última.
export function fileGlyph(name?: string): string {
  const base = (name || "").split("/").at(-1) || "untitled"
  const direct = byFilename[base] ?? byFilename[base.toLowerCase()]
  if (direct) return direct
  const parts = base.toLowerCase().split(".")
  if (parts.length > 2) {
    const two = parts.slice(-2).join(".")
    if (byExtension[two]) return byExtension[two]
  }
  if (parts.length > 1) {
    const one = parts.at(-1) ?? ""
    if (byExtension[one]) return byExtension[one]
  }
  return defaultFile
}
export function folderGlyph(expanded: boolean): string {
  return expanded ? folderOpen : folderClosed
}
export function chevronGlyph(expanded: boolean): string {
  return expanded ? "▾ " : "▸ "
}
