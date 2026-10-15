// Geometria dos divisores arrastáveis: pura e testável.
//
// Os divisores (árvore, SCM, split, painel inferior) fazem a mesma conta —
// prender uma coordenada de mouse numa faixa — e um lugar só impede que cada
// um prenda à sua maneira. A UI liga o evento ao número; os pisos e tetos
// moram aqui, com teste. Geometria pura vive em core para que app e UI
// compartilhem sem violar o layering (services/app nunca importam de ui).
export function clampTreeWidth(x: number, columns: number): number {
  return Math.max(8, Math.min(columns - 20, x))
}
export function clampScmWidth(x: number, columns: number): number {
  return Math.max(15, Math.min(columns - 40, columns - x))
}
export function clampSplitRatio(relative: number, extent: number): number {
  return Math.max(15, Math.min(85, Math.round(relative / Math.max(1, extent) * 100)))
}
export function clampTerminalHeight(y: number, height: number): number {
  return Math.max(3, Math.min(height - 5, height - y - 1))
}
