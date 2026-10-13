import { RGBA } from "@opentui/core"
import type { Config } from "../services/config"
import { builtInThemes } from "../services/catalogs"
export type Theme = { name: string; isLain: boolean; surface: RGBA; panel: RGBA; text: RGBA; muted: RGBA; accent: RGBA; selection: RGBA; error: RGBA; warning: RGBA; cursor: RGBA; border: RGBA; noColor: boolean }
const palettes: Record<string, string[]> = {
  default: ["#161b22", "#0d1117", "#e6edf3", "#8b949e", "#79c0ff", "#264f78", "#ff7b72", "#e3b341", "#30363d"],
  "default-dark": ["#161b22", "#0d1117", "#e6edf3", "#8b949e", "#79c0ff", "#264f78", "#ff7b72", "#e3b341", "#30363d"],
  "daisy-lain": ["#070913", "#00106a", "#f1f1ea", "#525f70", "#ffe500", "#0025c5", "#e73748", "#ffe500", "#0025c5"],
  "zed-dark": ["#18181b", "#121215", "#e4e4e7", "#71717a", "#59c2ff", "#27272a", "#f43f5e", "#eab308", "#27272a"],
  dracula: ["#282a36", "#21222c", "#f8f8f2", "#6272a4", "#bd93f9", "#44475a", "#ff5555", "#f1fa8c", "#44475a"],
  nord: ["#2e3440", "#3b4252", "#eceff4", "#81a1c1", "#88c0d0", "#434c5e", "#bf616a", "#ebcb8b", "#434c5e"],
  "tokyo-night": ["#1a1b26", "#16161e", "#a9b1d6", "#565f89", "#7aa2f7", "#283457", "#f7768e", "#e0af68", "#292e42"],
  "catppuccin-mocha": ["#1e1e2e", "#181825", "#cdd6f4", "#6c7086", "#89b4fa", "#313244", "#f38ba8", "#f9e2af", "#313244"],
  "one-dark": ["#282c34", "#21252b", "#abb2bf", "#5c6370", "#61afef", "#3e4451", "#e06c75", "#e5c07b", "#3e4451"],
  monokai: ["#272822", "#1e1f1c", "#f8f8f2", "#75715e", "#66d9ef", "#3e3d32", "#f92672", "#fd971f", "#3e3d32"],
  "solarized-dark": ["#002b36", "#073642", "#839496", "#586e75", "#268bd2", "#073642", "#dc322f", "#b58900", "#073642"],
  "solarized-light": ["#fdf6e3", "#eee8d5", "#657b83", "#93a1a1", "#268bd2", "#eee8d5", "#dc322f", "#b58900", "#eee8d5"],
  "github-dark": ["#0d1117", "#161b22", "#c9d1d9", "#8b949e", "#58a6ff", "#264f78", "#ff7b72", "#d29922", "#30363d"],
  "github-light": ["#ffffff", "#f6f8fa", "#24292f", "#57606a", "#0969da", "#ddf4ff", "#cf222e", "#9a6700", "#d0d7de"],
}
const named: Record<string, string> = { darkgray: "#808080", black: "#000000", white: "#ffffff", red: "#ff5555", green: "#50fa7b", yellow: "#f1fa8c", blue: "#79c0ff", cyan: "#8be9fd", magenta: "#ff79c6", gray: "#aaaaaa" }
export function resolveColor(value: string | undefined, fallback: string, background = false): RGBA {
  if (value === "reset" || value === "" || value === "default") return background ? RGBA.defaultBackground() : RGBA.defaultForeground()
  return RGBA.fromHex(named[value ?? ""] ?? value ?? fallback)
}
export function themeOf(config: Config): Theme {
  const normTheme = config.theme.toLowerCase().replaceAll(" ", "-")
  const builtIn = Object.entries(builtInThemes).find(([key]) => key === normTheme || key === config.theme)?.[1]
  const effectiveUI: Record<string, string> = { ...builtIn?.ui, ...config.ui }
  const effectiveSyntax: Record<string, string> = { ...builtIn?.syntax, ...config.syntax }
  const values = palettes[normTheme] ?? palettes[config.theme] ?? palettes.default ?? []
  const noColor = config.theme === "no-color" || !!process.env.NO_COLOR
  const color = (token: string, index: number, ui?: string, background = false) => {
    if (noColor) return background ? RGBA.defaultBackground() : RGBA.defaultForeground()
    const hex = config.tokens[token] ?? (ui ? effectiveUI[ui] : undefined) ?? values[index] ?? "#ffffff"
    return resolveColor(hex, values[index] ?? "#ffffff", background)
  }
  const accentHex = config.tokens["accent"] ?? effectiveUI["accent"] ?? effectiveSyntax["function"] ?? effectiveSyntax["keyword"] ?? values[4] ?? "#79c0ff"
  const selectionHex = config.tokens["focus"] ?? effectiveUI["selection"] ?? values[5] ?? "#264f78"
  const borderHex = config.tokens["border"] ?? effectiveUI["border"] ?? values[8] ?? values[1] ?? "#333338"
  const isLain = !noColor && (normTheme === "daisy-lain" || config.theme.toLowerCase() === "daisy-lain")
  return {
    name: normTheme,
    isLain,
    surface: color("surface", 0, "background", true),
    panel: color("panel", 1, "status_bg", true),
    text: color("text", 2, "foreground"),
    muted: color("text_muted", 3, "line_number"),
    accent: noColor ? RGBA.defaultForeground() : resolveColor(accentHex, values[4] ?? "#79c0ff"),
    selection: noColor ? RGBA.defaultBackground() : resolveColor(selectionHex, values[5] ?? "#264f78", true),
    error: color("error", 6, undefined),
    warning: color("warning", 7, undefined),
    cursor: color("cursor", 2, "cursor_bg"),
    border: noColor ? RGBA.defaultForeground() : resolveColor(borderHex, values[8] ?? "#333338"),
    noColor,
  }
}
